package monitor

import (
	"context"
	"fmt"
	"autoteam/internal/logger"

	"go.uber.org/zap"

	"autoteam/internal/github"
)

// ResolutionResult represents the result of checking if an item was resolved
type ResolutionResult int

const (
	// ItemNotFound means the item no longer exists in pending items (resolved)
	ItemNotFound ResolutionResult = iota

	// ItemStillPending means the item still exists and needs more work
	ItemStillPending

	// ItemChanged means the item exists but has been modified (partial progress)
	ItemChanged
)

// ResolutionDetector checks if items have been resolved by comparing API snapshots
type ResolutionDetector struct {
	githubClient *github.Client
}

// NewResolutionDetector creates a new resolution detector
func NewResolutionDetector(githubClient *github.Client) *ResolutionDetector {
	return &ResolutionDetector{
		githubClient: githubClient,
	}
}

// CheckItemResolution checks if a processing item has been resolved
func (rd *ResolutionDetector) CheckItemResolution(ctx context.Context, item *ProcessingItem, username string) (ResolutionResult, error) {
	if item == nil {
		return ItemNotFound, fmt.Errorf("no item to check")
	}

	log.Printf("Checking resolution for %s #%d: %s", item.Type, item.Number, item.Title)

	// Get fresh pending items from GitHub
	currentPendingItems, err := rd.githubClient.GetPendingItems(ctx, username)
	if err != nil {
		return ItemStillPending, fmt.Errorf("failed to get current pending items: %w", err)
	}

	// Check if the item still exists in the appropriate category
	switch item.Type {
	case "review_request":
		return rd.checkInReviewRequests(item, currentPendingItems.ReviewRequests), nil

	case "assigned_pr":
		return rd.checkInAssignedPRs(item, currentPendingItems.AssignedPRs), nil

	case "assigned_issue":
		return rd.checkInAssignedIssues(item, currentPendingItems.AssignedIssues), nil

	case "pr_with_changes":
		return rd.checkInPRsWithChanges(item, currentPendingItems.PRsWithChanges), nil

	default:
		return ItemStillPending, fmt.Errorf("unknown item type: %s", item.Type)
	}
}

// checkInReviewRequests checks if item exists in review requests
func (rd *ResolutionDetector) checkInReviewRequests(item *ProcessingItem, reviewRequests []github.PullRequestInfo) ResolutionResult {
	for _, pr := range reviewRequests {
		if pr.Number == item.Number && pr.Repository == item.Repository {
			// Item still exists, check if it has changed
			if pr.Title != item.Title || pr.URL != item.URL {
				log.Printf("Review request #%d in %s has changed: title or URL updated", item.Number, item.Repository)
				return ItemChanged
			}
			log.Printf("Review request #%d in %s still pending", item.Number, item.Repository)
			return ItemStillPending
		}
	}

	log.Printf("Review request #%d in %s no longer in pending list - likely resolved", item.Number, item.Repository)
	return ItemNotFound
}

// checkInAssignedPRs checks if item exists in assigned PRs
func (rd *ResolutionDetector) checkInAssignedPRs(item *ProcessingItem, assignedPRs []github.PullRequestInfo) ResolutionResult {
	for _, pr := range assignedPRs {
		if pr.Number == item.Number && pr.Repository == item.Repository {
			// Item still exists, check if it has changed
			if pr.Title != item.Title || pr.URL != item.URL {
				log.Printf("Assigned PR #%d in %s has changed: title or URL updated", item.Number, item.Repository)
				return ItemChanged
			}
			log.Printf("Assigned PR #%d in %s still pending", item.Number, item.Repository)
			return ItemStillPending
		}
	}

	log.Printf("Assigned PR #%d in %s no longer in pending list - likely resolved", item.Number, item.Repository)
	return ItemNotFound
}

// checkInAssignedIssues checks if item exists in assigned issues
func (rd *ResolutionDetector) checkInAssignedIssues(item *ProcessingItem, assignedIssues []github.IssueInfo) ResolutionResult {
	for _, issue := range assignedIssues {
		if issue.Number == item.Number && issue.Repository == item.Repository {
			// Item still exists, check if it has changed
			if issue.Title != item.Title || issue.URL != item.URL {
				log.Printf("Assigned issue #%d in %s has changed: title or URL updated", item.Number, item.Repository)
				return ItemChanged
			}
			log.Printf("Assigned issue #%d in %s still pending", item.Number, item.Repository)
			return ItemStillPending
		}
	}

	log.Printf("Assigned issue #%d in %s no longer in pending list - likely resolved", item.Number, item.Repository)
	return ItemNotFound
}

// checkInPRsWithChanges checks if item exists in PRs with changes requested
func (rd *ResolutionDetector) checkInPRsWithChanges(item *ProcessingItem, prsWithChanges []github.PullRequestInfo) ResolutionResult {
	for _, pr := range prsWithChanges {
		if pr.Number == item.Number && pr.Repository == item.Repository {
			// Item still exists, check if it has changed
			if pr.Title != item.Title || pr.URL != item.URL {
				log.Printf("PR with changes #%d in %s has changed: title or URL updated", item.Number, item.Repository)
				return ItemChanged
			}
			log.Printf("PR with changes #%d in %s still pending", item.Number, item.Repository)
			return ItemStillPending
		}
	}

	log.Printf("PR with changes #%d in %s no longer in pending list - likely resolved", item.Number, item.Repository)
	return ItemNotFound
}

// LogResolutionResult logs the resolution result with appropriate message
func LogResolutionResult(result ResolutionResult, item *ProcessingItem) {
	switch result {
	case ItemNotFound:
		log.Printf("✅ SUCCESS: %s #%d in %s appears to be resolved", item.Type, item.Number, item.Repository)
	case ItemStillPending:
		log.Printf("⚠️  STILL PENDING: %s #%d in %s requires more work", item.Type, item.Number, item.Repository)
	case ItemChanged:
		log.Printf("🔄 CHANGED: %s #%d in %s has been modified (partial progress)", item.Type, item.Number, item.Repository)
	}
}
