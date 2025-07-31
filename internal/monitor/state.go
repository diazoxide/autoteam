package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"autoteam/internal/config"
	"autoteam/internal/github"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// ProcessingState represents the current state of item processing
type ProcessingState struct {
	// Current item being processed
	CurrentItem *ProcessingItem `json:"current_item,omitempty"`

	// Processing history for recently failed items
	RecentFailures map[string]*FailureInfo `json:"recent_failures,omitempty"`

	// Last update timestamp
	LastUpdated time.Time `json:"last_updated"`
}

// ProcessingItem represents an item currently being processed
type ProcessingItem struct {
	// Item identification
	Type       string `json:"type"`       // "review_request", "assigned_pr", "assigned_issue", "pr_with_changes"
	Number     int    `json:"number"`     // GitHub issue/PR number
	Repository string `json:"repository"` // Repository in "owner/repo" format
	URL        string `json:"url"`        // GitHub URL
	Title      string `json:"title"`      // Issue/PR title

	// Processing metadata
	StartTime    time.Time `json:"start_time"`
	AttemptCount int       `json:"attempt_count"`
	LastAttempt  time.Time `json:"last_attempt"`
}

// FailureInfo tracks recent failures for an item
type FailureInfo struct {
	FailureCount  int       `json:"failure_count"`
	LastFailure   time.Time `json:"last_failure"`
	CooldownUntil time.Time `json:"cooldown_until"`
}

// StateManager manages the processing state persistence
type StateManager struct {
	statePath string
	state     *ProcessingState
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	statePath := filepath.Join(config.AutoTeamDir, "processing_state.json")

	sm := &StateManager{
		statePath: statePath,
		state: &ProcessingState{
			RecentFailures: make(map[string]*FailureInfo),
			LastUpdated:    time.Now(),
		},
	}

	// Load existing state if available
	if err := sm.load(); err != nil {
		// Create a basic logger for startup warnings
		if startupLogger, logErr := logger.NewLogger(logger.WarnLevel); logErr == nil {
			startupLogger.Warn("Failed to load existing state", zap.Error(err))
		}
	}

	return sm
}

// GetCurrentItem returns the currently processing item
func (sm *StateManager) GetCurrentItem() *ProcessingItem {
	return sm.state.CurrentItem
}

// SetCurrentItem sets the current processing item
func (sm *StateManager) SetCurrentItem(item *ProcessingItem) error {
	sm.state.CurrentItem = item
	sm.state.LastUpdated = time.Now()
	return sm.save()
}

// ClearCurrentItem clears the current processing item
func (sm *StateManager) ClearCurrentItem() error {
	sm.state.CurrentItem = nil
	sm.state.LastUpdated = time.Now()
	return sm.save()
}

// IsItemInProgress checks if any item is currently being processed
func (sm *StateManager) IsItemInProgress() bool {
	return sm.state.CurrentItem != nil
}

// IncrementAttempt increments the attempt count for current item
func (sm *StateManager) IncrementAttempt() error {
	if sm.state.CurrentItem == nil {
		return fmt.Errorf("no current item to increment attempt for")
	}

	sm.state.CurrentItem.AttemptCount++
	sm.state.CurrentItem.LastAttempt = time.Now()
	sm.state.LastUpdated = time.Now()

	return sm.save()
}

// RecordFailure records a failure for an item and applies cooldown
func (sm *StateManager) RecordFailure(itemKey string) error {
	failure, exists := sm.state.RecentFailures[itemKey]
	if !exists {
		failure = &FailureInfo{}
		sm.state.RecentFailures[itemKey] = failure
	}

	failure.FailureCount++
	failure.LastFailure = time.Now()

	// Apply cooldown based on failure count (exponential backoff)
	cooldownMinutes := failure.FailureCount * 30 // 30 min, 1hr, 1.5hr, etc.
	failure.CooldownUntil = time.Now().Add(time.Duration(cooldownMinutes) * time.Minute)

	sm.state.LastUpdated = time.Now()
	return sm.save()
}

// IsItemInCooldown checks if an item is in cooldown period
func (sm *StateManager) IsItemInCooldown(itemKey string) bool {
	failure, exists := sm.state.RecentFailures[itemKey]
	if !exists {
		return false
	}

	return time.Now().Before(failure.CooldownUntil)
}

// GetFailureCount returns the failure count for an item
func (sm *StateManager) GetFailureCount(itemKey string) int {
	failure, exists := sm.state.RecentFailures[itemKey]
	if !exists {
		return 0
	}

	return failure.FailureCount
}

// CleanupOldFailures removes old failure records (older than 24 hours)
func (sm *StateManager) CleanupOldFailures() error {
	cutoff := time.Now().Add(-24 * time.Hour)

	for key, failure := range sm.state.RecentFailures {
		if failure.LastFailure.Before(cutoff) {
			delete(sm.state.RecentFailures, key)
		}
	}

	sm.state.LastUpdated = time.Now()
	return sm.save()
}

// CreateProcessingItemFromPR creates a ProcessingItem from a PullRequestInfo
func CreateProcessingItemFromPR(pr github.PullRequestInfo, itemType string) *ProcessingItem {
	return &ProcessingItem{
		Type:         itemType,
		Number:       pr.Number,
		Repository:   pr.Repository,
		URL:          pr.URL,
		Title:        pr.Title,
		StartTime:    time.Now(),
		AttemptCount: 1,
		LastAttempt:  time.Now(),
	}
}

// CreateProcessingItemFromIssue creates a ProcessingItem from an IssueInfo
func CreateProcessingItemFromIssue(issue github.IssueInfo, itemType string) *ProcessingItem {
	return &ProcessingItem{
		Type:         itemType,
		Number:       issue.Number,
		Repository:   issue.Repository,
		URL:          issue.URL,
		Title:        issue.Title,
		StartTime:    time.Now(),
		AttemptCount: 1,
		LastAttempt:  time.Now(),
	}
}

// GetItemKey generates a unique key for an item including repository
func GetItemKey(itemType string, repository string, number int) string {
	// Normalize repository name for key (replace / with -)
	normalizedRepo := strings.ReplaceAll(repository, "/", "-")
	return fmt.Sprintf("%s_%s_%d", itemType, normalizedRepo, number)
}

// GetItemKeyFromProcessingItem generates a unique key from a ProcessingItem
func GetItemKeyFromProcessingItem(item *ProcessingItem) string {
	return GetItemKey(item.Type, item.Repository, item.Number)
}

// load loads the state from disk
func (sm *StateManager) load() error {
	if _, err := os.Stat(sm.statePath); os.IsNotExist(err) {
		// State file doesn't exist, start with empty state
		return nil
	}

	data, err := os.ReadFile(sm.statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, sm.state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Initialize RecentFailures map if nil
	if sm.state.RecentFailures == nil {
		sm.state.RecentFailures = make(map[string]*FailureInfo)
	}

	return nil
}

// save saves the state to disk
func (sm *StateManager) save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(sm.statePath), config.DirPerm); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(sm.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}
