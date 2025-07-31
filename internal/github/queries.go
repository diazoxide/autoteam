package github

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v57/github"
)

// GetPendingItems retrieves all pending items that need attention across filtered repositories
func (c *Client) GetPendingItems(ctx context.Context, username string) (*PendingItems, error) {
	log.Printf("Getting pending items for user: %s", username)
	items := &PendingItems{}

	// Get review requests across all filtered repositories
	reviewRequests, err := c.getReviewRequests(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get review requests: %w", err)
	}
	items.ReviewRequests = reviewRequests
	log.Printf("Found %d review requests", len(reviewRequests))

	// Get assigned PRs across all filtered repositories
	assignedPRs, err := c.getAssignedPRs(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned PRs: %w", err)
	}
	items.AssignedPRs = assignedPRs
	log.Printf("Found %d assigned PRs", len(assignedPRs))

	// Get assigned issues (excluding those with linked PRs) across all filtered repositories
	assignedIssues, err := c.getAssignedIssues(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned issues: %w", err)
	}
	items.AssignedIssues = assignedIssues
	log.Printf("Found %d assigned issues", len(assignedIssues))

	// Get PRs with changes requested across all filtered repositories
	prsWithChanges, err := c.getPRsWithChangesRequested(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs with changes requested: %w", err)
	}
	items.PRsWithChanges = prsWithChanges
	log.Printf("Found %d PRs with changes requested", len(prsWithChanges))

	totalItems := len(reviewRequests) + len(assignedPRs) + len(assignedIssues) + len(prsWithChanges)
	log.Printf("Total pending items found: %d", totalItems)

	return items, nil
}

// getReviewRequests gets PRs where the user is requested as a reviewer across all filtered repositories
func (c *Client) getReviewRequests(ctx context.Context, username string) ([]PullRequestInfo, error) {
	// Search globally for PRs where the user is requested for review
	query := fmt.Sprintf("is:pr is:open review-requested:%s", username)
	log.Printf("Searching for review requests with query: %s", query)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for review requests: %w", err)
	}
	log.Printf("GitHub search returned %d results for review requests", len(result.Issues))

	var prs []PullRequestInfo
	for i, issue := range result.Issues {
		log.Printf("Processing review request result %d/%d: PR #%d", i+1, len(result.Issues), issue.GetNumber())

		if issue.PullRequestLinks == nil {
			log.Printf("Skipping issue #%d: not a pull request (no PR links)", issue.GetNumber())
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Printf("Issue #%d: no repository object from GitHub API", issue.GetNumber())
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Printf("Attempting to extract repository from URL: %s", htmlURL)
				// GitHub URLs are typically: https://github.com/owner/repo/pull/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Printf("Extracted repository from URL: %s", repoName)
					} else {
						log.Printf("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Printf("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Printf("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Printf("Checking repository: %s for PR #%d", repoName, issue.GetNumber())

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Printf("Repository %s filtered out by repository filter", repoName)
			continue
		}
		log.Printf("Repository %s passed filter", repoName)

		// Parse repository owner/name
		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Printf("Warning: failed to parse repository %s: %v", repoName, err)
			continue
		}

		// Get full PR details
		log.Printf("Getting PR details for #%d in %s", issue.GetNumber(), repoName)
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, issue.GetNumber())
		if err != nil {
			log.Printf("Warning: failed to get PR #%d in %s: %v", issue.GetNumber(), repoName, err)
			continue
		}

		log.Printf("Successfully retrieved PR #%d details from %s", issue.GetNumber(), repoName)
		prInfo := FromGitHubPullRequest(pr)
		prInfo.Repository = repoName
		prs = append(prs, prInfo)
	}

	return prs, nil
}

// getAssignedPRs gets PRs assigned to the user across all filtered repositories
func (c *Client) getAssignedPRs(ctx context.Context, username string) ([]PullRequestInfo, error) {
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
		log.Printf("Processing assigned PR result %d/%d: PR #%d", i+1, len(result.Issues), issue.GetNumber())

		if issue.PullRequestLinks == nil {
			log.Printf("Skipping issue #%d: not a pull request (no PR links)", issue.GetNumber())
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Printf("Issue #%d: no repository object from GitHub API", issue.GetNumber())
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Printf("Attempting to extract repository from URL: %s", htmlURL)
				// GitHub URLs are typically: https://github.com/owner/repo/pull/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Printf("Extracted repository from URL: %s", repoName)
					} else {
						log.Printf("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Printf("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Printf("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Printf("Checking repository: %s for PR #%d", repoName, issue.GetNumber())

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Printf("Repository %s filtered out by repository filter", repoName)
			continue
		}
		log.Printf("Repository %s passed filter", repoName)

		// Parse repository owner/name
		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Printf("Warning: failed to parse repository %s: %v", repoName, err)
			continue
		}

		// Get full PR details
		log.Printf("Getting PR details for #%d in %s", issue.GetNumber(), repoName)
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, issue.GetNumber())
		if err != nil {
			log.Printf("Warning: failed to get PR #%d in %s: %v", issue.GetNumber(), repoName, err)
			continue
		}

		log.Printf("Successfully retrieved assigned PR #%d details from %s", issue.GetNumber(), repoName)
		prInfo := FromGitHubPullRequest(pr)
		prInfo.Repository = repoName
		prs = append(prs, prInfo)
	}

	return prs, nil
}

// getAssignedIssues gets issues assigned to the user (excluding those with linked PRs) across all filtered repositories
func (c *Client) getAssignedIssues(ctx context.Context, username string) ([]IssueInfo, error) {
	// Search globally for issues assigned to the user
	query := fmt.Sprintf("is:issue is:open assignee:%s -linked:pr", username)
	log.Printf("Searching for assigned issues with query: %s", query)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for assigned issues: %w", err)
	}
	log.Printf("GitHub search returned %d results for assigned issues", len(result.Issues))

	var issues []IssueInfo
	for i, issue := range result.Issues {
		log.Printf("Processing assigned issue result %d/%d: Issue #%d", i+1, len(result.Issues), issue.GetNumber())

		// Skip if it's actually a PR
		if issue.PullRequestLinks != nil {
			log.Printf("Skipping issue #%d: it's actually a pull request", issue.GetNumber())
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Printf("Issue #%d: no repository object from GitHub API", issue.GetNumber())
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Printf("Attempting to extract repository from URL: %s", htmlURL)
				// GitHub URLs are typically: https://github.com/owner/repo/issues/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Printf("Extracted repository from URL: %s", repoName)
					} else {
						log.Printf("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Printf("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Printf("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Printf("Checking repository: %s for issue #%d", repoName, issue.GetNumber())

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Printf("Repository %s filtered out by repository filter", repoName)
			continue
		}
		log.Printf("Repository %s passed filter", repoName)

		log.Printf("Adding issue #%d from %s to assigned issues list", issue.GetNumber(), repoName)
		issueInfo := FromGitHubIssue(issue)
		issueInfo.Repository = repoName
		issues = append(issues, issueInfo)
	}

	return issues, nil
}

// getPRsWithChangesRequested gets PRs authored by the user that have changes requested across all filtered repositories
func (c *Client) getPRsWithChangesRequested(ctx context.Context, username string) ([]PullRequestInfo, error) {
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
		log.Printf("Processing PR with changes requested result %d/%d: PR #%d", i+1, len(result.Issues), issue.GetNumber())

		if issue.PullRequestLinks == nil {
			log.Printf("Skipping issue #%d: not a pull request (no PR links)", issue.GetNumber())
			continue
		}

		var repoName string
		if issue.Repository == nil {
			log.Printf("Issue #%d: no repository object from GitHub API", issue.GetNumber())
			// Try to extract repository info from the HTML URL as workaround
			if htmlURL := issue.GetHTMLURL(); htmlURL != "" {
				log.Printf("Attempting to extract repository from URL: %s", htmlURL)
				// GitHub URLs are typically: https://github.com/owner/repo/pull/123
				if strings.Contains(htmlURL, "github.com") {
					parts := strings.Split(htmlURL, "/")
					if len(parts) >= 5 && parts[2] == "github.com" {
						repoName = parts[3] + "/" + parts[4]
						log.Printf("Extracted repository from URL: %s", repoName)
					} else {
						log.Printf("Could not parse repository from URL structure")
						continue
					}
				} else {
					log.Printf("URL is not a GitHub URL")
					continue
				}
			} else {
				log.Printf("No URL available to extract repository from")
				continue
			}
		} else {
			repoName = issue.Repository.GetFullName()
		}
		log.Printf("Checking repository: %s for PR #%d", repoName, issue.GetNumber())

		// Apply repository filter
		if !c.filter.ShouldIncludeRepository(repoName) {
			log.Printf("Repository %s filtered out by repository filter", repoName)
			continue
		}
		log.Printf("Repository %s passed filter", repoName)

		// Parse repository owner/name
		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Printf("Warning: failed to parse repository %s: %v", repoName, err)
			continue
		}

		// Get full PR details
		log.Printf("Getting PR details for #%d in %s", issue.GetNumber(), repoName)
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, issue.GetNumber())
		if err != nil {
			log.Printf("Warning: failed to get PR #%d in %s: %v", issue.GetNumber(), repoName, err)
			continue
		}

		// Check if this PR has changes requested
		log.Printf("Checking if PR #%d has changes requested", issue.GetNumber())
		hasChangesRequested, reviews, err := c.checkChangesRequested(ctx, owner, repo, issue.GetNumber(), pr)
		if err != nil {
			log.Printf("Warning: failed to check reviews for PR #%d in %s: %v", issue.GetNumber(), repoName, err)
			continue
		}

		log.Printf("PR #%d has changes requested: %v", issue.GetNumber(), hasChangesRequested)
		if hasChangesRequested {
			log.Printf("Adding PR #%d to changes requested list", issue.GetNumber())
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
	opts := &github.ListOptions{PerPage: 100}
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, prNumber, opts)
	if err != nil {
		return false, nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	log.Printf("Found %d total reviews for PR #%d", len(reviews), prNumber)

	// Group reviews by reviewer and get the latest review from each
	latestReviews := make(map[string]*github.PullRequestReview)
	for _, review := range reviews {
		if review.User == nil {
			continue
		}
		reviewer := review.User.GetLogin()
		log.Printf("Review from %s: state=%s, submitted=%v", reviewer, review.GetState(), review.GetSubmittedAt())

		// Keep only the latest review from each reviewer
		if existing, exists := latestReviews[reviewer]; !exists || review.GetSubmittedAt().After(existing.GetSubmittedAt().Time) {
			latestReviews[reviewer] = review
		}
	}

	log.Printf("Latest reviews from %d reviewers", len(latestReviews))

	// Check if any of the latest reviews request changes
	var hasChanges bool
	var reviewInfos []ReviewInfo
	for reviewer, review := range latestReviews {
		reviewInfo := FromGitHubReview(review)
		reviewInfos = append(reviewInfos, reviewInfo)

		log.Printf("Latest review from %s: state=%s", reviewer, review.GetState())
		if strings.EqualFold(review.GetState(), "changes_requested") {
			log.Printf("Found changes requested from %s", reviewer)
			hasChanges = true
		}
	}

	log.Printf("PR #%d final changes requested status: %v", prNumber, hasChanges)

	// If PR has changes requested, check if developer has re-requested review
	if hasChanges && pr.RequestedReviewers != nil && len(pr.RequestedReviewers) > 0 {
		var reviewerNames []string
		for _, reviewer := range pr.RequestedReviewers {
			if reviewer.Login != nil {
				reviewerNames = append(reviewerNames, reviewer.GetLogin())
			}
		}
		log.Printf("PR #%d has %d pending re-review requests from: %v", prNumber, len(reviewerNames), reviewerNames)
		log.Printf("PR #%d excluded from pending items (waiting for reviewer response)", prNumber)
		return false, reviewInfos, nil // Exclude from pending - waiting for reviewers
	}

	return hasChanges, reviewInfos, nil
}
