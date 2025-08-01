package monitor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"autoteam/internal/github"
)

func TestItemPrioritizer_CollectAndPrioritizeItems(t *testing.T) {
	sm := NewStateManager()
	p := &ItemPrioritizer{
		stateManager: sm,
	}

	now := time.Now()
	oldTime := now.AddDate(0, 0, -5) // 5 days ago

	pendingItems := &github.PendingItems{
		ReviewRequests: []github.PullRequestInfo{
			{Number: 1, Title: "Fix bug", Repository: "owner/repo", UpdatedAt: oldTime},
		},
		AssignedPRs: []github.PullRequestInfo{
			{Number: 2, Title: "Add feature", Repository: "owner/repo", UpdatedAt: now},
		},
		AssignedIssues: []github.IssueInfo{
			{Number: 3, Title: "Documentation update", Repository: "owner/repo", UpdatedAt: now},
		},
		PRsWithChanges: []github.PullRequestInfo{
			{Number: 4, Title: "Refactor code", Repository: "owner/repo", UpdatedAt: now},
		},
		Mentions: []github.MentionInfo{
			{Number: 5, Title: "Question about API", Repository: "owner/repo", Type: "issue", CreatedAt: now},
		},
		UnreadComments: []github.CommentInfo{
			{Number: 6, Title: "Discussion on PR", Repository: "owner/repo", Type: "pull_request", CreatedAt: now},
		},
		Notifications: []github.NotificationInfo{
			{ID: "7", Subject: "New release", Repository: "owner/repo", UpdatedAt: now},
		},
		FailedWorkflows: []github.WorkflowInfo{
			{ID: 8, Name: "CI Tests", Repository: "owner/repo", UpdatedAt: now},
		},
	}

	items := p.collectAndPrioritizeItems(pendingItems)

	// Verify all items are collected
	assert.Len(t, items, 8)

	// Verify item types and base scores
	typeScores := map[string]int{
		"review_request":  1000,
		"mention":         900,
		"assigned_pr":     800,
		"failed_workflow": 700,
		"pr_with_changes": 600,
		"unread_comment":  500,
		"assigned_issue":  400,
		"notification":    300,
	}

	for _, item := range items {
		expectedBaseScore, ok := typeScores[item.Type]
		assert.True(t, ok, "Unknown item type: %s", item.Type)

		// The actual score should be at least the base score
		assert.GreaterOrEqual(t, item.Score, expectedBaseScore,
			"Item type %s should have score >= %d", item.Type, expectedBaseScore)
	}

	// Verify that old items get age bonus
	var reviewRequestItem *PrioritizedItem
	for _, item := range items {
		if item.Type == "review_request" && item.Number == 1 {
			reviewRequestItem = item
			break
		}
	}
	assert.NotNil(t, reviewRequestItem)
	assert.Greater(t, reviewRequestItem.Score, 1000, "Old review request should have age bonus")
	assert.Contains(t, reviewRequestItem.Reason, "old")
}

func TestItemPrioritizer_CalculatePriority(t *testing.T) {
	sm := NewStateManager()
	p := &ItemPrioritizer{
		stateManager: sm,
	}

	tests := []struct {
		name          string
		item          *PrioritizedItem
		baseScore     int
		expectedMin   int
		expectedWords []string
	}{
		{
			name: "urgent keyword",
			item: &PrioritizedItem{
				Title:     "URGENT: Fix critical bug",
				UpdatedAt: time.Now(),
			},
			baseScore:     500,
			expectedMin:   1000, // 500 base + 500 urgent bonus
			expectedWords: []string{"urgent"},
		},
		{
			name: "old item",
			item: &PrioritizedItem{
				Title:     "Regular task",
				UpdatedAt: time.Now().AddDate(0, 0, -8), // 8 days old
			},
			baseScore:     500,
			expectedMin:   800, // 500 base + 300 very old bonus
			expectedWords: []string{"very old"},
		},
		{
			name: "bug keyword",
			item: &PrioritizedItem{
				Title:     "Fix login bug",
				UpdatedAt: time.Now(),
			},
			baseScore:     500,
			expectedMin:   600, // 500 base + 100 bug bonus
			expectedWords: []string{"bug"},
		},
		{
			name: "failed workflow with PR",
			item: &PrioritizedItem{
				Type:      "failed_workflow",
				Title:     "Workflow 'CI Tests' failed",
				UpdatedAt: time.Now(),
				Details: map[string]interface{}{
					"pull_requests": "#123, #456",
				},
			},
			baseScore:     700,
			expectedMin:   700,
			expectedWords: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, reason := p.calculatePriority(tt.item, tt.baseScore)

			assert.GreaterOrEqual(t, score, tt.expectedMin,
				"Score should be at least %d, got %d", tt.expectedMin, score)

			for _, word := range tt.expectedWords {
				assert.Contains(t, reason, word,
					"Reason should contain '%s'", word)
			}
		})
	}
}

func TestItemPrioritizer_SelectNextItem(t *testing.T) {
	sm := NewStateManager()
	p := NewItemPrioritizer(sm)

	now := time.Now()

	// Create pending items with different priorities
	pendingItems := &github.PendingItems{
		ReviewRequests: []github.PullRequestInfo{
			{Number: 1, Title: "Review this PR", Repository: "owner/repo", UpdatedAt: now},
		},
		Mentions: []github.MentionInfo{
			{Number: 2, Title: "Question for you", Repository: "owner/repo", Type: "issue", CreatedAt: now},
		},
		Notifications: []github.NotificationInfo{
			{ID: "3", Subject: "New comment", Repository: "owner/repo", UpdatedAt: now},
		},
	}

	selected := p.SelectNextItem(pendingItems)

	assert.NotNil(t, selected)
	// Review requests have highest base priority (1000)
	assert.Equal(t, "review_request", selected.Type)
	assert.Equal(t, 1, selected.Number)
}

func TestProcessingItemDetails(t *testing.T) {
	// Test that Details field is properly propagated
	prioritized := &PrioritizedItem{
		Type:       "mention",
		Number:     123,
		Repository: "owner/repo",
		Title:      "Test mention",
		URL:        "https://github.com/owner/repo/issues/123",
		Details: map[string]interface{}{
			"type": "issue",
			"body": "@user please review this",
		},
	}

	processing := CreateProcessingItemFromPrioritized(prioritized)

	assert.Equal(t, prioritized.Type, processing.Type)
	assert.Equal(t, prioritized.Number, processing.Number)
	assert.Equal(t, prioritized.Details, processing.Details)
	assert.Equal(t, "issue", processing.Details["type"])
	assert.Equal(t, "@user please review this", processing.Details["body"])
}

func TestGetItemKeyWithNewEventTypes(t *testing.T) {
	tests := []struct {
		itemType   string
		repository string
		number     int
		expected   string
	}{
		{"mention", "owner/repo", 123, "mention_owner-repo_123"},
		{"notification", "owner/repo", 0, "notification_owner-repo_0"},
		{"failed_workflow", "owner/repo", 0, "failed_workflow_owner-repo_0"},
		{"unread_comment", "org/project", 456, "unread_comment_org-project_456"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%d", tt.itemType, tt.number), func(t *testing.T) {
			key := GetItemKey(tt.itemType, tt.repository, tt.number)
			assert.Equal(t, tt.expected, key)
		})
	}
}
