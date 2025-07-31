package github

import (
	"context"
	"fmt"
	"strings"

	"autoteam/internal/logger"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
)

// GetPendingItems retrieves all pending items that need attention across filtered repositories
func (c *Client) GetPendingItems(ctx context.Context, username string) (*PendingItems, error) {
	log := logger.FromContext(ctx)
	log.Info("Getting pending items for user", zap.String("username", username))
	items := &PendingItems{}

	// Get review requests across all filtered repositories
	reviewRequests, err := c.getReviewRequests(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get review requests: %w", err)
	}
	items.ReviewRequests = reviewRequests
	log.Info("Found review requests", zap.Int("count", len(reviewRequests)))

	// Get assigned PRs across all filtered repositories
	assignedPRs, err := c.getAssignedPRs(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned PRs: %w", err)
	}
	items.AssignedPRs = assignedPRs
	log.Info("Found assigned PRs", zap.Int("count", len(assignedPRs)))

	// Get assigned issues (excluding those with linked PRs) across all filtered repositories
	assignedIssues, err := c.getAssignedIssues(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned issues: %w", err)
	}
	items.AssignedIssues = assignedIssues
	log.Info("Found assigned issues", zap.Int("count", len(assignedIssues)))

	// Get PRs with changes requested across all filtered repositories
	prsWithChanges, err := c.getPRsWithChangesRequested(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs with changes requested: %w", err)
	}
	items.PRsWithChanges = prsWithChanges
	log.Info("Found PRs with changes requested", zap.Int("count", len(prsWithChanges)))

	totalItems := len(reviewRequests) + len(assignedPRs) + len(assignedIssues) + len(prsWithChanges)
	log.Info("Total pending items found", zap.Int("total", totalItems))

	return items, nil
}

// getReviewRequests gets PRs where the user is requested as a reviewer across all filtered repositories
func (c *Client) getReviewRequests(ctx context.Context, username string) ([]PullRequestInfo, error) {
	// Search globally for PRs where the user is requested for review
	query := fmt.Sprintf("is:pr is:open review-requested:%s", username)
	log := logger.FromContext(ctx)
	log.Info("Searching for review requests", zap.String("query", query), zap.String("username", username))

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for review requests: %w", err)
	}
	log.Info("GitHub search returned results for review requests", zap.Int("count", len(result.Issues)))

	var prs []PullRequestInfo
	for i, issue := range result.Issues {
		log.Debug("Processing review request result", zap.Int("index", i+1), zap.Int("total", len(result.Issues)), zap.Int("pr_number", issue.GetNumber()))

		if issue.PullRequestLinks == nil {
			log.Debug("Skipping issue: not a pull request", zap.Int("issue_number", issue.GetNumber()))
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Debug("Issue has no repository object from GitHub API", zap.Int("issue_number", issue.GetNumber()))
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Debug("Attempting to extract repository from URL", zap.String("url", htmlURL))
				// GitHub URLs are typically: https://github.com/owner/repo/pull/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Debug("Extracted repository from URL", zap.String("repository", repoName))
					} else {
						log.Debug("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Debug("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Debug("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Debug("Checking repository for PR", zap.String("repository", repoName), zap.Int("pr_number", issue.GetNumber()))

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Debug("Repository filtered out by repository filter", zap.String("repository", repoName))
			continue
		}
		log.Debug("Repository passed filter", zap.String("repository", repoName))

		// Parse repository owner/name
		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Warn("Failed to parse repository", zap.String("repository", repoName), zap.Error(err))
			continue
		}

		// Get full PR details
		log.Debug("Getting PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName))
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, issue.GetNumber())
		if err != nil {
			log.Warn("Failed to get PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName), zap.Error(err))
			continue
		}

		log.Debug("Successfully retrieved PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName))
		prInfo := FromGitHubPullRequest(pr)
		prInfo.Repository = repoName
		prs = append(prs, prInfo)
	}

	return prs, nil
}

// getAssignedPRs gets PRs assigned to the user across all filtered repositories
func (c *Client) getAssignedPRs(ctx context.Context, username string) ([]PullRequestInfo, error) {
	log := logger.FromContext(ctx)
	// Search globally for PRs assigned to the user
	query := fmt.Sprintf("is:pr is:open assignee:%s", username)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for assigned PRs: %w", err)
	}

	var prs []PullRequestInfo
	for i, issue := range result.Issues {
		log.Debug("Processing assigned PR result", zap.Int("index", i+1), zap.Int("total", len(result.Issues)), zap.Int("pr_number", issue.GetNumber()))

		if issue.PullRequestLinks == nil {
			log.Debug("Skipping issue: not a pull request", zap.Int("issue_number", issue.GetNumber()))
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Debug("Issue has no repository object from GitHub API", zap.Int("issue_number", issue.GetNumber()))
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Debug("Attempting to extract repository from URL", zap.String("url", htmlURL))
				// GitHub URLs are typically: https://github.com/owner/repo/pull/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Debug("Extracted repository from URL", zap.String("repository", repoName))
					} else {
						log.Debug("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Debug("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Debug("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Debug("Checking repository for PR", zap.String("repository", repoName), zap.Int("pr_number", issue.GetNumber()))

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Debug("Repository filtered out by repository filter", zap.String("repository", repoName))
			continue
		}
		log.Debug("Repository passed filter", zap.String("repository", repoName))

		// Parse repository owner/name
		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Warn("Failed to parse repository", zap.String("repository", repoName), zap.Error(err))
			continue
		}

		// Get full PR details
		log.Debug("Getting PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName))
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, issue.GetNumber())
		if err != nil {
			log.Warn("Failed to get PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName), zap.Error(err))
			continue
		}

		log.Debug("Successfully retrieved assigned PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName))
		prInfo := FromGitHubPullRequest(pr)
		prInfo.Repository = repoName
		prs = append(prs, prInfo)
	}

	return prs, nil
}

// getAssignedIssues gets issues assigned to the user (excluding those with linked PRs) across all filtered repositories
func (c *Client) getAssignedIssues(ctx context.Context, username string) ([]IssueInfo, error) {
	log := logger.FromContext(ctx)
	// Search globally for issues assigned to the user
	query := fmt.Sprintf("is:issue is:open assignee:%s -linked:pr", username)
	log.Info("Searching for assigned issues", zap.String("query", query))

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for assigned issues: %w", err)
	}
	log.Info("GitHub search returned results for assigned issues", zap.Int("count", len(result.Issues)))

	var issues []IssueInfo
	for i, issue := range result.Issues {
		log.Debug("Processing assigned issue result", zap.Int("index", i+1), zap.Int("total", len(result.Issues)), zap.Int("issue_number", issue.GetNumber()))

		// Skip if it's actually a PR
		if issue.PullRequestLinks != nil {
			log.Debug("Skipping issue: actually a pull request", zap.Int("issue_number", issue.GetNumber()))
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Debug("Issue has no repository object from GitHub API", zap.Int("issue_number", issue.GetNumber()))
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Debug("Attempting to extract repository from URL", zap.String("url", htmlURL))
				// GitHub URLs are typically: https://github.com/owner/repo/issues/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Debug("Extracted repository from URL", zap.String("repository", repoName))
					} else {
						log.Debug("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Debug("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Debug("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Debug("Checking repository for issue", zap.String("repository", repoName), zap.Int("issue_number", issue.GetNumber()))

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Debug("Repository filtered out by repository filter", zap.String("repository", repoName))
			continue
		}
		log.Debug("Repository passed filter", zap.String("repository", repoName))

		log.Debug("Adding issue to assigned issues list", zap.Int("issue_number", issue.GetNumber()), zap.String("repository", repoName))
		issueInfo := FromGitHubIssue(issue)
		issueInfo.Repository = repoName
		issues = append(issues, issueInfo)
	}

	return issues, nil
}

// getPRsWithChangesRequested gets PRs authored by the user that have changes requested across all filtered repositories
func (c *Client) getPRsWithChangesRequested(ctx context.Context, username string) ([]PullRequestInfo, error) {
	log := logger.FromContext(ctx)
	// Search globally for PRs authored by the user
	query := fmt.Sprintf("is:pr is:open author:%s", username)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user's PRs: %w", err)
	}

	var prsWithChanges []PullRequestInfo
	for i, issue := range result.Issues {
		log.Debug("Processing PR with changes requested result", zap.Int("index", i+1), zap.Int("total", len(result.Issues)), zap.Int("pr_number", issue.GetNumber()))

		if issue.PullRequestLinks == nil {
			log.Debug("Skipping issue: not a pull request", zap.Int("issue_number", issue.GetNumber()))
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Debug("Issue has no repository object from GitHub API", zap.Int("issue_number", issue.GetNumber()))
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Debug("Attempting to extract repository from URL", zap.String("url", htmlURL))
				// GitHub URLs are typically: https://github.com/owner/repo/pull/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Debug("Extracted repository from URL", zap.String("repository", repoName))
					} else {
						log.Debug("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Debug("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Debug("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Debug("Checking repository for PR", zap.String("repository", repoName), zap.Int("pr_number", issue.GetNumber()))

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Debug("Repository filtered out by repository filter", zap.String("repository", repoName))
			continue
		}
		log.Debug("Repository passed filter", zap.String("repository", repoName))

		// Parse repository owner/name
		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Warn("Failed to parse repository", zap.String("repository", repoName), zap.Error(err))
			continue
		}

		// Get full PR details
		log.Debug("Getting PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName))
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, issue.GetNumber())
		if err != nil {
			log.Warn("Failed to get PR details", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName), zap.Error(err))
			continue
		}

		// Check if this PR has changes requested
		log.Debug("Checking if PR has changes requested", zap.Int("pr_number", issue.GetNumber()))
		hasChangesRequested, reviews, err := c.checkChangesRequested(ctx, owner, repo, issue.GetNumber(), pr)
		if err != nil {
			log.Warn("Failed to check reviews for PR", zap.Int("pr_number", issue.GetNumber()), zap.String("repository", repoName), zap.Error(err))
			continue
		}

		log.Debug("PR changes requested status", zap.Int("pr_number", issue.GetNumber()), zap.Bool("has_changes_requested", hasChangesRequested))
		if hasChangesRequested {
			log.Debug("Adding PR to changes requested list", zap.Int("pr_number", issue.GetNumber()))
			prInfo := FromGitHubPullRequest(pr)
			prInfo.Repository = repoName
			prInfo.HasChangesRequested = true
			prInfo.Reviews = reviews
			prsWithChanges = append(prsWithChanges, prInfo)
		}
	}

	return prsWithChanges, nil
}

// checkChangesRequested checks if a PR has changes requested in the latest reviews
// and excludes PRs where developer has requested re-review (waiting for reviewer response)
func (c *Client) checkChangesRequested(ctx context.Context, owner, repo string, prNumber int, pr *github.PullRequest) (bool, []ReviewInfo, error) {
	log := logger.FromContext(ctx)
	opts := &github.ListOptions{PerPage: 100}
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, prNumber, opts)
	if err != nil {
		return false, nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	log.Debug("Found total reviews for PR", zap.Int("review_count", len(reviews)), zap.Int("pr_number", prNumber))

	// Group reviews by reviewer and get the latest review from each
	latestReviews := make(map[string]*github.PullRequestReview)
	for _, review := range reviews {
		if review.User == nil {
			continue
		}
		reviewer := review.User.GetLogin()
		log.Debug("Review details", zap.String("reviewer", reviewer), zap.String("state", review.GetState()), zap.Time("submitted_at", review.GetSubmittedAt().Time))

		// Keep only the latest review from each reviewer
		if existing, exists := latestReviews[reviewer]; !exists || review.GetSubmittedAt().After(existing.GetSubmittedAt().Time) {
			latestReviews[reviewer] = review
		}
	}

	log.Debug("Latest reviews summary", zap.Int("reviewer_count", len(latestReviews)))

	// Check if any of the latest reviews request changes
	var hasChanges bool
	var reviewInfos []ReviewInfo
	for reviewer, review := range latestReviews {
		reviewInfo := FromGitHubReview(review)
		reviewInfos = append(reviewInfos, reviewInfo)

		log.Debug("Latest review from reviewer", zap.String("reviewer", reviewer), zap.String("state", review.GetState()))
		if strings.EqualFold(review.GetState(), "changes_requested") {
			log.Debug("Found changes requested from reviewer", zap.String("reviewer", reviewer))
			hasChanges = true
		}
	}

	log.Debug("PR final changes requested status", zap.Int("pr_number", prNumber), zap.Bool("has_changes", hasChanges))

	// If PR has changes requested, check if developer has re-requested review
	if hasChanges && pr.RequestedReviewers != nil && len(pr.RequestedReviewers) > 0 {
		var reviewerNames []string
		for _, reviewer := range pr.RequestedReviewers {
			if reviewer.Login != nil {
				reviewerNames = append(reviewerNames, reviewer.GetLogin())
			}
		}
		log.Debug("PR has pending re-review requests", zap.Int("pr_number", prNumber), zap.Int("request_count", len(reviewerNames)), zap.Strings("reviewers", reviewerNames))
		log.Debug("PR excluded from pending items (waiting for reviewer response)", zap.Int("pr_number", prNumber))
		return false, reviewInfos, nil // Exclude from pending - waiting for reviewers
	}

	return hasChanges, reviewInfos, nil
}
