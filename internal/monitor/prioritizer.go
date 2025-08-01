package monitor

import (
	"autoteam/internal/logger"

	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"autoteam/internal/github"
)

// PrioritizedItem represents an item with its calculated priority score
type PrioritizedItem struct {
	// Item details
	Type       string
	Number     int
	Repository string
	Title      string
	URL        string
	Author     string
	UpdatedAt  time.Time
	Details    map[string]interface{} // Additional details for specific event types

	// Prioritization
	Score  int
	Reason string
}

// ItemPrioritizer handles prioritization of pending items
type ItemPrioritizer struct {
	stateManager *StateManager
}

// NewItemPrioritizer creates a new item prioritizer
func NewItemPrioritizer(stateManager *StateManager) *ItemPrioritizer {
	return &ItemPrioritizer{
		stateManager: stateManager,
	}
}

// SelectNextItem selects the highest priority item that's not currently being processed
func (p *ItemPrioritizer) SelectNextItem(pendingItems *github.PendingItems) *PrioritizedItem {
	// If an item is currently being processed, don't select anything new
	if p.stateManager.IsItemInProgress() {
		// Create a basic logger for this function since we don't have context
		if lgr, err := logger.NewLogger(logger.InfoLevel); err == nil {
			lgr.Info("Item already in progress, skipping selection")
		}
		return nil
	}

	// Collect all items and calculate their priorities
	allItems := p.collectAndPrioritizeItems(pendingItems)

	if len(allItems) == 0 {
		// Create a basic logger for this function since we don't have context
		if lgr, err := logger.NewLogger(logger.InfoLevel); err == nil {
			lgr.Info("No pending items to prioritize")
		}
		return nil
	}

	// Sort by priority score (highest first)
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Score > allItems[j].Score
	})

	// Select the highest priority item that's not in cooldown
	for _, item := range allItems {
		itemKey := GetItemKey(item.Type, item.Repository, item.Number)

		if p.stateManager.IsItemInCooldown(itemKey) {
			// Create a basic logger for this function since we don't have context
			if lgr, err := logger.NewLogger(logger.InfoLevel); err == nil {
				lgr.Info("Skipping item - in cooldown period",
					zap.String("type", item.Type),
					zap.Int("number", item.Number))
			}
			continue
		}

		// Create a basic logger for this function since we don't have context
		if lgr, err := logger.NewLogger(logger.InfoLevel); err == nil {
			lgr.Info("Selected item",
				zap.String("type", item.Type),
				zap.Int("number", item.Number),
				zap.String("title", item.Title),
				zap.Int("score", item.Score),
				zap.String("reason", item.Reason))
		}
		return item
	}

	// Create a basic logger for this function since we don't have context
	if lgr, err := logger.NewLogger(logger.InfoLevel); err == nil {
		lgr.Info("All items are in cooldown, no item selected")
	}
	return nil
}

// collectAndPrioritizeItems collects all pending items and calculates their priority scores
func (p *ItemPrioritizer) collectAndPrioritizeItems(pendingItems *github.PendingItems) []*PrioritizedItem {
	var allItems []*PrioritizedItem

	// Process review requests (highest base priority)
	for _, pr := range pendingItems.ReviewRequests {
		item := &PrioritizedItem{
			Type:       "review_request",
			Number:     pr.Number,
			Repository: pr.Repository,
			Title:      pr.Title,
			URL:        pr.URL,
			Author:     pr.Author,
			UpdatedAt:  pr.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 1000) // Base score: 1000
		allItems = append(allItems, item)
	}

	// Process assigned PRs (high priority)
	for _, pr := range pendingItems.AssignedPRs {
		item := &PrioritizedItem{
			Type:       "assigned_pr",
			Number:     pr.Number,
			Repository: pr.Repository,
			Title:      pr.Title,
			URL:        pr.URL,
			Author:     pr.Author,
			UpdatedAt:  pr.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 800) // Base score: 800
		allItems = append(allItems, item)
	}

	// Process PRs with changes requested (medium priority)
	for _, pr := range pendingItems.PRsWithChanges {
		item := &PrioritizedItem{
			Type:       "pr_with_changes",
			Number:     pr.Number,
			Repository: pr.Repository,
			Title:      pr.Title,
			URL:        pr.URL,
			Author:     pr.Author,
			UpdatedAt:  pr.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 600) // Base score: 600
		allItems = append(allItems, item)
	}

	// Process assigned issues (lower priority)
	for _, issue := range pendingItems.AssignedIssues {
		item := &PrioritizedItem{
			Type:       "assigned_issue",
			Number:     issue.Number,
			Repository: issue.Repository,
			Title:      issue.Title,
			URL:        issue.URL,
			Author:     issue.Author,
			UpdatedAt:  issue.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 400) // Base score: 400
		allItems = append(allItems, item)
	}

	// Process mentions (high priority - requires immediate response)
	for _, mention := range pendingItems.Mentions {
		item := &PrioritizedItem{
			Type:       "mention",
			Number:     mention.Number,
			Repository: mention.Repository,
			Title:      mention.Title,
			URL:        mention.URL,
			Author:     mention.Author,
			UpdatedAt:  mention.CreatedAt,
			Details: map[string]interface{}{
				"type": mention.Type,
				"body": mention.Body,
			},
		}
		item.Score, item.Reason = p.calculatePriority(item, 900) // Base score: 900
		allItems = append(allItems, item)
	}

	// Process unread comments (medium priority)
	for _, comment := range pendingItems.UnreadComments {
		item := &PrioritizedItem{
			Type:       "unread_comment",
			Number:     comment.Number,
			Repository: comment.Repository,
			Title:      comment.Title,
			URL:        comment.URL,
			Author:     comment.Author,
			UpdatedAt:  comment.CreatedAt,
			Details: map[string]interface{}{
				"type": comment.Type,
				"body": comment.Body,
			},
		}
		item.Score, item.Reason = p.calculatePriority(item, 500) // Base score: 500
		allItems = append(allItems, item)
	}

	// Process notifications (medium priority)
	for _, notif := range pendingItems.Notifications {
		item := &PrioritizedItem{
			Type:       "notification",
			Number:     0, // Notifications don't have issue/PR numbers
			Repository: notif.Repository,
			Title:      notif.Subject,
			URL:        notif.URL,
			Author:     "", // Notifications don't have authors
			UpdatedAt:  notif.UpdatedAt,
			Details: map[string]interface{}{
				"id":     notif.ID,
				"reason": notif.Reason,
			},
		}
		item.Score, item.Reason = p.calculatePriority(item, 300) // Base score: 300
		allItems = append(allItems, item)
	}

	// Process failed workflows (high priority - blocks deployment)
	for _, workflow := range pendingItems.FailedWorkflows {
		prNumbers := ""
		if len(workflow.PullRequests) > 0 {
			prStrings := make([]string, len(workflow.PullRequests))
			for i, pr := range workflow.PullRequests {
				prStrings[i] = fmt.Sprintf("#%d", pr)
			}
			prNumbers = strings.Join(prStrings, ", ")
		}

		item := &PrioritizedItem{
			Type:       "failed_workflow",
			Number:     0, // Workflows don't have issue/PR numbers directly
			Repository: workflow.Repository,
			Title:      fmt.Sprintf("Workflow '%s' failed", workflow.Name),
			URL:        workflow.URL,
			Author:     "", // Workflows don't have authors
			UpdatedAt:  workflow.UpdatedAt,
			Details: map[string]interface{}{
				"workflow_id":   workflow.ID,
				"workflow_name": workflow.Name,
				"head_branch":   workflow.HeadBranch,
				"head_sha":      workflow.HeadSHA,
				"pull_requests": prNumbers,
			},
		}
		item.Score, item.Reason = p.calculatePriority(item, 700) // Base score: 700
		allItems = append(allItems, item)
	}

	return allItems
}

// calculatePriority calculates the priority score for an item
func (p *ItemPrioritizer) calculatePriority(item *PrioritizedItem, baseScore int) (int, string) {
	score := baseScore
	var reasons []string

	// Age bonus: older items get higher priority
	age := time.Since(item.UpdatedAt)
	ageDays := int(age.Hours() / 24)

	if ageDays >= 7 {
		score += 300
		reasons = append(reasons, "very old (7+ days)")
	} else if ageDays >= 3 {
		score += 200
		reasons = append(reasons, "old (3+ days)")
	} else if ageDays >= 1 {
		score += 100
		reasons = append(reasons, "aging (1+ day)")
	}

	// Urgency detection based on title keywords
	title := strings.ToLower(item.Title)
	urgentKeywords := []string{"urgent", "critical", "blocker", "hotfix", "emergency", "p0", "sev1"}

	for _, keyword := range urgentKeywords {
		if strings.Contains(title, keyword) {
			score += 500
			reasons = append(reasons, "urgent keyword: "+keyword)
			break
		}
	}

	// High priority keywords
	highPriorityKeywords := []string{"bug", "fix", "error", "broken", "failing", "p1", "sev2"}

	for _, keyword := range highPriorityKeywords {
		if strings.Contains(title, keyword) {
			score += 200
			reasons = append(reasons, "high priority: "+keyword)
			break
		}
	}

	// Apply failure penalty
	itemKey := GetItemKey(item.Type, item.Repository, item.Number)
	failureCount := p.stateManager.GetFailureCount(itemKey)

	if failureCount > 0 {
		penalty := failureCount * 100
		score -= penalty
		reasons = append(reasons, "failure penalty: -"+string(rune(penalty)))
	}

	// Build reason string
	reasonStr := strings.Join(reasons, ", ")
	if reasonStr == "" {
		reasonStr = "base priority"
	}

	return score, reasonStr
}
