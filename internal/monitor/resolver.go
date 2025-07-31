package monitor

import (
	"context"
	"fmt"

	"autoteam/internal/github"
	"autoteam/internal/logger"
	"go.uber.org/zap"
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
	lgr := logger.FromContext(ctx)
	if item == nil {
		return ItemNotFound, fmt.Errorf("no item to check")
	}

	lgr.Info("Checking resolution for item",
		zap.String("type", item.Type),
		zap.Int("number", item.Number),
		zap.String("title", item.Title))

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
				logger.FromContext(context.Background()).Info("Review request has changed: title or URL updated",
					zap.Int("number", item.Number),
					zap.String("repository", item.Repository))
				return ItemChanged
			}
			logger.FromContext(context.Background()).Info("Review request still pending",
				zap.Int("number", item.Number),
				zap.String("repository", item.Repository))
			return ItemStillPending
		}
	}

	logger.FromContext(context.Background()).Info("Review request no longer in pending list - likely resolved",
		zap.Int("number", item.Number),
		zap.String("repository", item.Repository))
	return ItemNotFound
}

// checkInAssignedPRs checks if item exists in assigned PRs
func (rd *ResolutionDetector) checkInAssignedPRs(item *ProcessingItem, assignedPRs []github.PullRequestInfo) ResolutionResult {
	for _, pr := range assignedPRs {
		if pr.Number == item.Number && pr.Repository == item.Repository {
			// Item still exists, check if it has changed
			if pr.Title != item.Title || pr.URL != item.URL {
				logger.FromContext(context.Background()).Info("Assigned PR has changed: title or URL updated",
					zap.Int("number", item.Number),
					zap.String("repository", item.Repository))
				return ItemChanged
			}
			logger.FromContext(context.Background()).Info("Assigned PR still pending",
				zap.Int("number", item.Number),
				zap.String("repository", item.Repository))
			return ItemStillPending
		}
	}

	logger.FromContext(context.Background()).Info("Assigned PR no longer in pending list - likely resolved",
		zap.Int("number", item.Number),
		zap.String("repository", item.Repository))
	return ItemNotFound
}

// checkInAssignedIssues checks if item exists in assigned issues
func (rd *ResolutionDetector) checkInAssignedIssues(item *ProcessingItem, assignedIssues []github.IssueInfo) ResolutionResult {
	for _, issue := range assignedIssues {
		if issue.Number == item.Number && issue.Repository == item.Repository {
			// Item still exists, check if it has changed
			if issue.Title != item.Title || issue.URL != item.URL {
				logger.FromContext(context.Background()).Info("Assigned issue has changed: title or URL updated",
					zap.Int("number", item.Number),
					zap.String("repository", item.Repository))
				return ItemChanged
			}
			logger.FromContext(context.Background()).Info("Assigned issue still pending",
				zap.Int("number", item.Number),
				zap.String("repository", item.Repository))
			return ItemStillPending
		}
	}

	logger.FromContext(context.Background()).Info("Assigned issue no longer in pending list - likely resolved",
		zap.Int("number", item.Number),
		zap.String("repository", item.Repository))
	return ItemNotFound
}

// checkInPRsWithChanges checks if item exists in PRs with changes requested
func (rd *ResolutionDetector) checkInPRsWithChanges(item *ProcessingItem, prsWithChanges []github.PullRequestInfo) ResolutionResult {
	for _, pr := range prsWithChanges {
		if pr.Number == item.Number && pr.Repository == item.Repository {
			// Item still exists, check if it has changed
			if pr.Title != item.Title || pr.URL != item.URL {
				logger.FromContext(context.Background()).Info("PR with changes has changed: title or URL updated",
					zap.Int("number", item.Number),
					zap.String("repository", item.Repository))
				return ItemChanged
			}
			logger.FromContext(context.Background()).Info("PR with changes still pending",
				zap.Int("number", item.Number),
				zap.String("repository", item.Repository))
			return ItemStillPending
		}
	}

	logger.FromContext(context.Background()).Info("PR with changes no longer in pending list - likely resolved",
		zap.Int("number", item.Number),
		zap.String("repository", item.Repository))
	return ItemNotFound
}

// LogResolutionResult logs the resolution result with appropriate message
func LogResolutionResult(result ResolutionResult, item *ProcessingItem) {
	// Use a basic logger for this logging function since we don't have context
	logger, err := logger.NewLogger(logger.InfoLevel)
	if err != nil {
		return // Skip logging if we can't create logger
	}

	switch result {
	case ItemNotFound:
		logger.Info("✅ SUCCESS: item appears to be resolved",
			zap.String("type", item.Type),
			zap.Int("number", item.Number),
			zap.String("repository", item.Repository))
	case ItemStillPending:
		logger.Warn("⚠️  STILL PENDING: item requires more work",
			zap.String("type", item.Type),
			zap.Int("number", item.Number),
			zap.String("repository", item.Repository))
	case ItemChanged:
		logger.Info("🔄 CHANGED: item has been modified (partial progress)",
			zap.String("type", item.Type),
			zap.Int("number", item.Number),
			zap.String("repository", item.Repository))
	}
}
