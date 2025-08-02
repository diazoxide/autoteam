package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"autoteam/internal/logger"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
)

// GetPendingItems retrieves all pending items that need attention across filtered repositories
// Uses notification-first strategy to minimize API calls
func (c *Client) GetPendingItems(ctx context.Context, username string) (*PendingItems, error) {
	log := logger.FromContext(ctx)
	log.Info("Getting pending items for user using notification-first strategy", zap.String("username", username))
	items := &PendingItems{}

	// Phase 1: Get enhanced notifications first (primary source)
	notifications, err := c.getEnhancedNotifications(ctx)
	if err != nil {
		log.Warn("Failed to get notifications, falling back to REST API", zap.Error(err))
		return c.getFallbackPendingItems(ctx, username)
	}
	log.Info("Found notifications", zap.Int("count", len(notifications)))

	// Phase 2: Correlate notifications to pending items
	notificationMap := make(map[string][]NotificationInfo)
	for _, notification := range notifications {
		if notification.CorrelatedType != "" {
			notificationMap[notification.CorrelatedType] = append(notificationMap[notification.CorrelatedType], notification)
		}
	}

	// Phase 3: Build items from notification correlations
	items.ReviewRequests = c.buildReviewRequestsFromNotifications(ctx, notificationMap["review_request"], username)
	items.AssignedPRs = c.buildAssignedPRsFromNotifications(ctx, notificationMap["assigned_pr"], username)
	items.AssignedIssues = c.buildAssignedIssuesFromNotifications(ctx, notificationMap["assigned_issue"], username)
	items.Mentions = c.buildMentionsFromNotifications(ctx, notificationMap["mention"], username)
	items.UnreadComments = c.buildUnreadCommentsFromNotifications(ctx, notificationMap["unread_comment"], username)
	items.FailedWorkflows = c.buildFailedWorkflowsFromNotifications(ctx, notificationMap["failed_workflow"], username)

	// Keep generic notifications that don't correlate to specific item types
	items.Notifications = append(items.Notifications, notificationMap["notification"]...)

	// Phase 4: Supplement with critical items not in notifications (minimal REST calls)
	c.supplementWithCriticalItems(ctx, items, username)

	// Phase 5: Get PRs with changes requested (still needs REST API)
	prsWithChanges, err := c.getPRsWithChangesRequested(ctx, username)
	if err != nil {
		log.Warn("Failed to get PRs with changes requested", zap.Error(err))
	} else {
		items.PRsWithChanges = prsWithChanges
	}

	totalItems := items.Count()
	log.Info("Total pending items found using notification-first strategy",
		zap.Int("total", totalItems),
		zap.Int("from_notifications", len(notifications)))

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

// getMentions gets recent mentions of the user in issues and PRs
func (c *Client) getMentions(ctx context.Context, username string) ([]MentionInfo, error) {
	log := logger.FromContext(ctx)
	// Search for recent mentions in issues and PRs
	query := fmt.Sprintf("mentions:%s updated:>%s", username, time.Now().AddDate(0, 0, -7).Format("2006-01-02"))
	log.Info("Searching for mentions", zap.String("query", query))

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Sort:        "updated",
		Order:       "desc",
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for mentions: %w", err)
	}

	var mentions []MentionInfo
	for _, issue := range result.Issues {
		var repoName string
		if issue.Repository != nil {
			repoName = issue.Repository.GetFullName()
		} else if htmlURL := issue.GetHTMLURL(); htmlURL != "" && strings.Contains(htmlURL, "github.com") {
			parts := strings.Split(htmlURL, "/")
			if len(parts) >= 5 && parts[2] == "github.com" {
				repoName = parts[3] + "/" + parts[4]
			}
		}

		if repoName == "" || !c.filter.ShouldIncludeRepository(repoName) {
			continue
		}

		// Parse repository owner/name
		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Warn("Failed to parse repository", zap.String("repository", repoName), zap.Error(err))
			continue
		}

		// Get recent comments to find the mention
		sinceTime := time.Now().AddDate(0, 0, -7)
		opts := &github.IssueListCommentsOptions{
			Since:       &sinceTime,
			ListOptions: github.ListOptions{PerPage: 100},
		}

		comments, _, err := c.client.Issues.ListComments(ctx, owner, repo, issue.GetNumber(), opts)
		if err != nil {
			log.Warn("Failed to get comments", zap.Int("issue_number", issue.GetNumber()), zap.Error(err))
			continue
		}

		// Find comments that mention the user
		for _, comment := range comments {
			if strings.Contains(comment.GetBody(), "@"+username) {
				itemType := "issue"
				if issue.PullRequestLinks != nil {
					itemType = "pull_request"
				}
				mention := FromGitHubIssueComment(comment, issue.GetNumber(), issue.GetTitle(), repoName, itemType)
				mentions = append(mentions, mention)
			}
		}
	}

	return mentions, nil
}

// GetSingleNotification gets the first unread notification from GitHub for simplified processing
func (c *Client) GetSingleNotification(ctx context.Context) (*NotificationInfo, error) {
	log := logger.FromContext(ctx)
	log.Debug("Getting single unread notification")

	opts := &github.NotificationListOptions{
		All:         false,                          // Only unread
		ListOptions: github.ListOptions{PerPage: 1}, // Only get 1 notification
	}

	notifications, _, err := c.client.Activity.ListNotifications(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	// Process first notification that matches repository filter
	for _, notification := range notifications {
		if notification.Repository == nil {
			continue
		}

		repoName := notification.Repository.GetFullName()
		if !c.filter.ShouldIncludeRepository(repoName) {
			continue
		}

		info := FromGitHubNotification(notification)
		log.Info("Found single notification to process",
			zap.String("reason", info.Reason),
			zap.String("subject", info.Subject),
			zap.String("repository", info.Repository),
			zap.String("thread_id", info.ThreadID))
		return &info, nil
	}

	// No unread notifications found
	log.Debug("No unread notifications found")
	return nil, nil
}

// getNotifications gets unread notifications from GitHub (deprecated - use GetSingleNotification for new workflow)
func (c *Client) getNotifications(ctx context.Context) ([]NotificationInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("Getting unread notifications")

	opts := &github.NotificationListOptions{
		All:         false, // Only unread
		ListOptions: github.ListOptions{PerPage: 100},
	}

	notifications, _, err := c.client.Activity.ListNotifications(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	var notificationInfos []NotificationInfo
	for _, notification := range notifications {
		if notification.Repository == nil {
			continue
		}

		repoName := notification.Repository.GetFullName()
		if !c.filter.ShouldIncludeRepository(repoName) {
			continue
		}

		info := FromGitHubNotification(notification)
		notificationInfos = append(notificationInfos, info)
	}

	return notificationInfos, nil
}

// getFailedWorkflows gets failed workflow runs for user's PRs
func (c *Client) getFailedWorkflows(ctx context.Context, username string) ([]WorkflowInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("Getting failed workflows for user's PRs")

	var failedWorkflows []WorkflowInfo

	// Get repositories the user has recently contributed to
	repos, err := c.getRecentContributionRepos(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent contribution repos: %w", err)
	}

	for _, repoName := range repos {
		if !c.filter.ShouldIncludeRepository(repoName) {
			continue
		}

		owner, repo, err := parseRepository(repoName)
		if err != nil {
			log.Warn("Failed to parse repository", zap.String("repository", repoName), zap.Error(err))
			continue
		}

		// Get recent workflow runs
		opts := &github.ListWorkflowRunsOptions{
			Actor:       username,
			Status:      "completed",
			ListOptions: github.ListOptions{PerPage: 20},
		}

		runs, _, err := c.client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
		if err != nil {
			log.Warn("Failed to get workflow runs", zap.String("repository", repoName), zap.Error(err))
			continue
		}

		for _, run := range runs.WorkflowRuns {
			// Only include failed runs from the last 24 hours
			if run.GetConclusion() == "failure" && run.GetCreatedAt().After(time.Now().AddDate(0, 0, -1)) {
				info := FromGitHubWorkflowRun(run)
				info.Repository = repoName
				failedWorkflows = append(failedWorkflows, info)
			}
		}
	}

	return failedWorkflows, nil
}

// getRecentContributionRepos gets repositories the user has recently contributed to
func (c *Client) getRecentContributionRepos(ctx context.Context, username string) ([]string, error) {
	// Search for recent PRs by the user to find active repositories
	query := fmt.Sprintf("is:pr author:%s updated:>%s", username, time.Now().AddDate(0, 0, -7).Format("2006-01-02"))

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user's recent PRs: %w", err)
	}

	repoSet := make(map[string]bool)
	for _, issue := range result.Issues {
		if issue.Repository != nil {
			repoSet[issue.Repository.GetFullName()] = true
		} else if htmlURL := issue.GetHTMLURL(); htmlURL != "" && strings.Contains(htmlURL, "github.com") {
			parts := strings.Split(htmlURL, "/")
			if len(parts) >= 5 && parts[2] == "github.com" {
				repoSet[parts[3]+"/"+parts[4]] = true
			}
		}
	}

	var repos []string
	for repo := range repoSet {
		repos = append(repos, repo)
	}

	return repos, nil
}

// getEnhancedNotifications gets notifications with enhanced correlation data
func (c *Client) getEnhancedNotifications(ctx context.Context) ([]NotificationInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("Getting enhanced notifications with correlation data")

	opts := &github.NotificationListOptions{
		All:         false, // Only unread
		ListOptions: github.ListOptions{PerPage: 100},
	}

	notifications, _, err := c.client.Activity.ListNotifications(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	var notificationInfos []NotificationInfo
	for _, notification := range notifications {
		if notification.Repository == nil {
			continue
		}

		repoName := notification.Repository.GetFullName()
		if !c.filter.ShouldIncludeRepository(repoName) {
			continue
		}

		// Use enhanced FromGitHubNotification with correlation
		info := FromGitHubNotification(notification)
		notificationInfos = append(notificationInfos, info)
	}

	log.Info("Enhanced notifications processed",
		zap.Int("total", len(notificationInfos)),
		zap.Int("raw_count", len(notifications)))

	return notificationInfos, nil
}

// getFallbackPendingItems uses the original REST-heavy approach as fallback
func (c *Client) getFallbackPendingItems(ctx context.Context, username string) (*PendingItems, error) {
	log := logger.FromContext(ctx)
	log.Info("Using fallback REST API strategy")
	items := &PendingItems{}

	// Original implementation as fallback
	reviewRequests, err := c.getReviewRequests(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get review requests: %w", err)
	}
	items.ReviewRequests = reviewRequests

	assignedPRs, err := c.getAssignedPRs(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned PRs: %w", err)
	}
	items.AssignedPRs = assignedPRs

	assignedIssues, err := c.getAssignedIssues(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned issues: %w", err)
	}
	items.AssignedIssues = assignedIssues

	prsWithChanges, err := c.getPRsWithChangesRequested(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs with changes requested: %w", err)
	}
	items.PRsWithChanges = prsWithChanges

	mentions, err := c.getMentions(ctx, username)
	if err != nil {
		log.Warn("Failed to get mentions", zap.Error(err))
	} else {
		items.Mentions = mentions
	}

	notifications, err := c.getNotifications(ctx)
	if err != nil {
		log.Warn("Failed to get notifications", zap.Error(err))
	} else {
		items.Notifications = notifications
	}

	failedWorkflows, err := c.getFailedWorkflows(ctx, username)
	if err != nil {
		log.Warn("Failed to get failed workflows", zap.Error(err))
	} else {
		items.FailedWorkflows = failedWorkflows
	}

	return items, nil
}

// buildReviewRequestsFromNotifications builds review requests from notifications
func (c *Client) buildReviewRequestsFromNotifications(ctx context.Context, notifications []NotificationInfo, username string) []PullRequestInfo {
	log := logger.FromContext(ctx)
	var reviewRequests []PullRequestInfo

	for _, notification := range notifications {
		if notification.Number == 0 || notification.Repository == "" {
			continue
		}

		owner, repo, err := parseRepository(notification.Repository)
		if err != nil {
			log.Warn("Failed to parse repository from notification", zap.String("repository", notification.Repository), zap.Error(err))
			continue
		}

		// Get PR details
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, notification.Number)
		if err != nil {
			log.Warn("Failed to get PR details from notification", zap.Int("pr_number", notification.Number), zap.String("repository", notification.Repository), zap.Error(err))
			continue
		}

		prInfo := FromGitHubPullRequest(pr)
		prInfo.Repository = notification.Repository
		// Store notification ID in details for later correlation
		prInfo.Details = map[string]interface{}{
			"notification_thread_id": notification.ThreadID,
		}
		reviewRequests = append(reviewRequests, prInfo)
	}

	log.Info("Built review requests from notifications", zap.Int("count", len(reviewRequests)))
	return reviewRequests
}

// buildAssignedPRsFromNotifications builds assigned PRs from notifications
func (c *Client) buildAssignedPRsFromNotifications(ctx context.Context, notifications []NotificationInfo, username string) []PullRequestInfo {
	log := logger.FromContext(ctx)
	var assignedPRs []PullRequestInfo

	for _, notification := range notifications {
		if notification.Number == 0 || notification.Repository == "" {
			continue
		}

		owner, repo, err := parseRepository(notification.Repository)
		if err != nil {
			log.Warn("Failed to parse repository from notification", zap.String("repository", notification.Repository), zap.Error(err))
			continue
		}

		// Get PR details
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, notification.Number)
		if err != nil {
			log.Warn("Failed to get PR details from notification", zap.Int("pr_number", notification.Number), zap.String("repository", notification.Repository), zap.Error(err))
			continue
		}

		prInfo := FromGitHubPullRequest(pr)
		prInfo.Repository = notification.Repository
		// Store notification ID in details for later correlation
		prInfo.Details = map[string]interface{}{
			"notification_thread_id": notification.ThreadID,
		}
		assignedPRs = append(assignedPRs, prInfo)
	}

	log.Info("Built assigned PRs from notifications", zap.Int("count", len(assignedPRs)))
	return assignedPRs
}

// buildAssignedIssuesFromNotifications builds assigned issues from notifications
func (c *Client) buildAssignedIssuesFromNotifications(ctx context.Context, notifications []NotificationInfo, username string) []IssueInfo {
	log := logger.FromContext(ctx)
	var assignedIssues []IssueInfo

	for _, notification := range notifications {
		if notification.Number == 0 || notification.Repository == "" {
			continue
		}

		owner, repo, err := parseRepository(notification.Repository)
		if err != nil {
			log.Warn("Failed to parse repository from notification", zap.String("repository", notification.Repository), zap.Error(err))
			continue
		}

		// Get issue details
		issue, _, err := c.client.Issues.Get(ctx, owner, repo, notification.Number)
		if err != nil {
			log.Warn("Failed to get issue details from notification", zap.Int("issue_number", notification.Number), zap.String("repository", notification.Repository), zap.Error(err))
			continue
		}

		// Skip if it's actually a PR
		if issue.PullRequestLinks != nil {
			continue
		}

		issueInfo := FromGitHubIssue(issue)
		issueInfo.Repository = notification.Repository
		// Store notification ID in details for later correlation
		issueInfo.Details = map[string]interface{}{
			"notification_thread_id": notification.ThreadID,
		}
		assignedIssues = append(assignedIssues, issueInfo)
	}

	log.Info("Built assigned issues from notifications", zap.Int("count", len(assignedIssues)))
	return assignedIssues
}

// buildMentionsFromNotifications builds mentions from notifications
func (c *Client) buildMentionsFromNotifications(ctx context.Context, notifications []NotificationInfo, username string) []MentionInfo {
	log := logger.FromContext(ctx)
	var mentions []MentionInfo

	for _, notification := range notifications {
		if notification.Number == 0 || notification.Repository == "" {
			continue
		}

		// Create mention info from notification
		mention := MentionInfo{
			Number:     notification.Number,
			Title:      notification.Subject,
			URL:        notification.URL,
			Repository: notification.Repository,
			Type:       strings.ToLower(notification.SubjectType),
			CreatedAt:  notification.UpdatedAt,
			Body:       fmt.Sprintf("Mentioned in notification: %s", notification.Subject),
			Details: map[string]interface{}{
				"notification_thread_id": notification.ThreadID,
			},
		}

		mentions = append(mentions, mention)
	}

	log.Info("Built mentions from notifications", zap.Int("count", len(mentions)))
	return mentions
}

// buildUnreadCommentsFromNotifications builds unread comments from notifications
func (c *Client) buildUnreadCommentsFromNotifications(ctx context.Context, notifications []NotificationInfo, username string) []CommentInfo {
	log := logger.FromContext(ctx)
	var comments []CommentInfo

	for _, notification := range notifications {
		if notification.Number == 0 || notification.Repository == "" {
			continue
		}

		// Create comment info from notification
		comment := CommentInfo{
			Number:     notification.Number,
			Title:      notification.Subject,
			URL:        notification.URL,
			Repository: notification.Repository,
			Type:       strings.ToLower(notification.SubjectType),
			CreatedAt:  notification.UpdatedAt,
			Body:       fmt.Sprintf("Unread comment from notification: %s", notification.Subject),
			Details: map[string]interface{}{
				"notification_thread_id": notification.ThreadID,
			},
		}

		comments = append(comments, comment)
	}

	log.Info("Built unread comments from notifications", zap.Int("count", len(comments)))
	return comments
}

// buildFailedWorkflowsFromNotifications builds failed workflows from notifications
func (c *Client) buildFailedWorkflowsFromNotifications(ctx context.Context, notifications []NotificationInfo, username string) []WorkflowInfo {
	log := logger.FromContext(ctx)
	var workflows []WorkflowInfo

	for _, notification := range notifications {
		if notification.Repository == "" {
			continue
		}

		// Create workflow info from notification
		workflow := WorkflowInfo{
			Name:       notification.Subject,
			URL:        notification.URL,
			Repository: notification.Repository,
			Status:     "completed",
			Conclusion: "failure",
			CreatedAt:  notification.UpdatedAt,
			UpdatedAt:  notification.UpdatedAt,
		}

		// Try to extract PR numbers if available
		if notification.Number > 0 {
			workflow.PullRequests = []int{notification.Number}
		}

		workflows = append(workflows, workflow)
	}

	log.Info("Built failed workflows from notifications", zap.Int("count", len(workflows)))
	return workflows
}

// supplementWithCriticalItems adds critical items that might not be in notifications
func (c *Client) supplementWithCriticalItems(ctx context.Context, items *PendingItems, username string) {
	log := logger.FromContext(ctx)
	log.Info("Supplementing with critical items not found in notifications")

	// This is intentionally minimal - only add truly critical items that notifications might miss
	// For now, we'll skip this to minimize API calls, but it could be extended if needed

	log.Info("Skipping supplemental items to minimize API calls")
}
