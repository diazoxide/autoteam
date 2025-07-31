package monitor

import (
	"log"
	"sort"
	"strings"
	"time"

	"autoteam/internal/github"
)

// PrioritizedItem represents an item with its calculated priority score
type PrioritizedItem struct {
	// Item details
	Type      string
	Number    int
	Title     string
	URL       string
	Author    string
	UpdatedAt time.Time

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
		log.Println("Item already in progress, skipping selection")
		return nil
	}

	// Collect all items and calculate their priorities
	allItems := p.collectAndPrioritizeItems(pendingItems)

	if len(allItems) == 0 {
		log.Println("No pending items to prioritize")
		return nil
	}

	// Sort by priority score (highest first)
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Score > allItems[j].Score
	})

	// Select the highest priority item that's not in cooldown
	for _, item := range allItems {
		itemKey := GetItemKey(item.Type, item.Number)

		if p.stateManager.IsItemInCooldown(itemKey) {
			log.Printf("Skipping item %s #%d - in cooldown period", item.Type, item.Number)
			continue
		}

		log.Printf("Selected item: %s #%d (%s) - Score: %d (%s)",
			item.Type, item.Number, item.Title, item.Score, item.Reason)
		return item
	}

	log.Println("All items are in cooldown, no item selected")
	return nil
}

// collectAndPrioritizeItems collects all pending items and calculates their priority scores
func (p *ItemPrioritizer) collectAndPrioritizeItems(pendingItems *github.PendingItems) []*PrioritizedItem {
	var allItems []*PrioritizedItem

	// Process review requests (highest base priority)
	for _, pr := range pendingItems.ReviewRequests {
		item := &PrioritizedItem{
			Type:      "review_request",
			Number:    pr.Number,
			Title:     pr.Title,
			URL:       pr.URL,
			Author:    pr.Author,
			UpdatedAt: pr.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 1000) // Base score: 1000
		allItems = append(allItems, item)
	}

	// Process assigned PRs (high priority)
	for _, pr := range pendingItems.AssignedPRs {
		item := &PrioritizedItem{
			Type:      "assigned_pr",
			Number:    pr.Number,
			Title:     pr.Title,
			URL:       pr.URL,
			Author:    pr.Author,
			UpdatedAt: pr.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 800) // Base score: 800
		allItems = append(allItems, item)
	}

	// Process PRs with changes requested (medium priority)
	for _, pr := range pendingItems.PRsWithChanges {
		item := &PrioritizedItem{
			Type:      "pr_with_changes",
			Number:    pr.Number,
			Title:     pr.Title,
			URL:       pr.URL,
			Author:    pr.Author,
			UpdatedAt: pr.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 600) // Base score: 600
		allItems = append(allItems, item)
	}

	// Process assigned issues (lower priority)
	for _, issue := range pendingItems.AssignedIssues {
		item := &PrioritizedItem{
			Type:      "assigned_issue",
			Number:    issue.Number,
			Title:     issue.Title,
			URL:       issue.URL,
			Author:    issue.Author,
			UpdatedAt: issue.UpdatedAt,
		}
		item.Score, item.Reason = p.calculatePriority(item, 400) // Base score: 400
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
	itemKey := GetItemKey(item.Type, item.Number)
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
