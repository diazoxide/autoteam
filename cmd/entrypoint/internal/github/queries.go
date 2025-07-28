package github

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v57/github"
)

// GetPendingItems retrieves all pending items that need attention
func (c *Client) GetPendingItems(ctx context.Context, username string) (*PendingItems, error) {
	items := &PendingItems{}

	// Get review requests
	reviewRequests, err := c.getReviewRequests(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get review requests: %w", err)
	}
	items.ReviewRequests = reviewRequests

	// Get assigned PRs
	assignedPRs, err := c.getAssignedPRs(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned PRs: %w", err)
	}
	items.AssignedPRs = assignedPRs

	// Get assigned issues (excluding those with linked PRs)
	assignedIssues, err := c.getAssignedIssues(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned issues: %w", err)
	}
	items.AssignedIssues = assignedIssues

	// Get PRs with changes requested
	prsWithChanges, err := c.getPRsWithChangesRequested(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs with changes requested: %w", err)
	}
	items.PRsWithChanges = prsWithChanges

	return items, nil
}

// getReviewRequests gets PRs where the user is requested as a reviewer
func (c *Client) getReviewRequests(ctx context.Context, username string) ([]PullRequestInfo, error) {
	// Search for PRs where the user is requested for review
	query := fmt.Sprintf("repo:%s/%s is:pr is:open review-requested:%s", c.owner, c.repo, username)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for review requests: %w", err)
	}

	var prs []PullRequestInfo
	for _, issue := range result.Issues {
		if issue.PullRequestLinks != nil {
			// Get full PR details
			pr, _, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, issue.GetNumber())
			if err != nil {
				log.Printf("Warning: failed to get PR #%d: %v", issue.GetNumber(), err)
				continue
			}
			prs = append(prs, FromGitHubPullRequest(pr))
		}
	}

	return prs, nil
}

// getAssignedPRs gets PRs assigned to the user
func (c *Client) getAssignedPRs(ctx context.Context, username string) ([]PullRequestInfo, error) {
	query := fmt.Sprintf("repo:%s/%s is:pr is:open assignee:%s", c.owner, c.repo, username)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for assigned PRs: %w", err)
	}

	var prs []PullRequestInfo
	for _, issue := range result.Issues {
		if issue.PullRequestLinks != nil {
			pr, _, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, issue.GetNumber())
			if err != nil {
				log.Printf("Warning: failed to get PR #%d: %v", issue.GetNumber(), err)
				continue
			}
			prs = append(prs, FromGitHubPullRequest(pr))
		}
	}

	return prs, nil
}

// getAssignedIssues gets issues assigned to the user (excluding those with linked PRs)
func (c *Client) getAssignedIssues(ctx context.Context, username string) ([]IssueInfo, error) {
	query := fmt.Sprintf("repo:%s/%s is:issue is:open assignee:%s -linked:pr", c.owner, c.repo, username)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for assigned issues: %w", err)
	}

	var issues []IssueInfo
	for _, issue := range result.Issues {
		// Skip if it's actually a PR
		if issue.PullRequestLinks == nil {
			issues = append(issues, FromGitHubIssue(issue))
		}
	}

	return issues, nil
}

// getPRsWithChangesRequested gets PRs authored by the user that have changes requested
func (c *Client) getPRsWithChangesRequested(ctx context.Context, username string) ([]PullRequestInfo, error) {
	query := fmt.Sprintf("repo:%s/%s is:pr is:open author:%s", c.owner, c.repo, username)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user's PRs: %w", err)
	}

	var prsWithChanges []PullRequestInfo
	for _, issue := range result.Issues {
		if issue.PullRequestLinks != nil {
			pr, _, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, issue.GetNumber())
			if err != nil {
				log.Printf("Warning: failed to get PR #%d: %v", issue.GetNumber(), err)
				continue
			}

			// Check if this PR has changes requested
			hasChangesRequested, reviews, err := c.checkChangesRequested(ctx, issue.GetNumber())
			if err != nil {
				log.Printf("Warning: failed to check reviews for PR #%d: %v", issue.GetNumber(), err)
				continue
			}

			if hasChangesRequested {
				prInfo := FromGitHubPullRequest(pr)
				prInfo.HasChangesRequested = true
				prInfo.Reviews = reviews
				prsWithChanges = append(prsWithChanges, prInfo)
			}
		}
	}

	return prsWithChanges, nil
}

// checkChangesRequested checks if a PR has changes requested in the latest reviews
func (c *Client) checkChangesRequested(ctx context.Context, prNumber int) (bool, []ReviewInfo, error) {
	opts := &github.ListOptions{PerPage: 100}
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, c.owner, c.repo, prNumber, opts)
	if err != nil {
		return false, nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	// Group reviews by reviewer and get the latest review from each
	latestReviews := make(map[string]*github.PullRequestReview)
	for _, review := range reviews {
		if review.User == nil {
			continue
		}
		reviewer := review.User.GetLogin()

		// Keep only the latest review from each reviewer
		if existing, exists := latestReviews[reviewer]; !exists || review.GetSubmittedAt().After(existing.GetSubmittedAt().Time) {
			latestReviews[reviewer] = review
		}
	}

	// Check if any of the latest reviews request changes
	var hasChanges bool
	var reviewInfos []ReviewInfo
	for _, review := range latestReviews {
		reviewInfo := FromGitHubReview(review)
		reviewInfos = append(reviewInfos, reviewInfo)

		if strings.EqualFold(review.GetState(), "changes_requested") {
			hasChanges = true
		}
	}

	return hasChanges, reviewInfos, nil
}
