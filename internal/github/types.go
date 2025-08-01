package github

import (
	"time"

	"github.com/google/go-github/v57/github"
)

// PendingItems represents all pending items that need attention
type PendingItems struct {
	ReviewRequests  []PullRequestInfo  `json:"review_requests"`
	AssignedPRs     []PullRequestInfo  `json:"assigned_prs"`
	AssignedIssues  []IssueInfo        `json:"assigned_issues"`
	PRsWithChanges  []PullRequestInfo  `json:"prs_with_changes"`
	Mentions        []MentionInfo      `json:"mentions"`
	UnreadComments  []CommentInfo      `json:"unread_comments"`
	Notifications   []NotificationInfo `json:"notifications"`
	FailedWorkflows []WorkflowInfo     `json:"failed_workflows"`
}

// PullRequestInfo contains information about a pull request
type PullRequestInfo struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	Author     string    `json:"author"`
	Repository string    `json:"repository"` // owner/repo format
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// For PRs with changes requested
	HasChangesRequested bool         `json:"has_changes_requested,omitempty"`
	Reviews             []ReviewInfo `json:"reviews,omitempty"`
	// For tracking re-review requests
	RequestedReviewers []string `json:"requested_reviewers,omitempty"`
}

// IssueInfo contains information about an issue
type IssueInfo struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	Author     string    `json:"author"`
	Repository string    `json:"repository"` // owner/repo format
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Labels     []string  `json:"labels"`
}

// RepositoryInfo contains information about a repository
type RepositoryInfo struct {
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"` // owner/repo format
	Owner         string    `json:"owner"`
	URL           string    `json:"url"`
	DefaultBranch string    `json:"default_branch"`
	Private       bool      `json:"private"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Description   string    `json:"description,omitempty"`
}

// ReviewInfo contains information about a pull request review
type ReviewInfo struct {
	Author      string    `json:"author"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// MentionInfo contains information about a mention in an issue or PR
type MentionInfo struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	Repository string    `json:"repository"`
	Type       string    `json:"type"` // "issue" or "pull_request"
	Author     string    `json:"author"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// CommentInfo contains information about a comment on an issue or PR
type CommentInfo struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	Repository string    `json:"repository"`
	Type       string    `json:"type"` // "issue" or "pull_request"
	Author     string    `json:"author"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// NotificationInfo contains information from GitHub notifications API
type NotificationInfo struct {
	ID         string    `json:"id"`
	Reason     string    `json:"reason"`
	Subject    string    `json:"subject"`
	URL        string    `json:"url"`
	Repository string    `json:"repository"`
	UpdatedAt  time.Time `json:"updated_at"`
	Unread     bool      `json:"unread"`
}

// WorkflowInfo contains information about a failed workflow run
type WorkflowInfo struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	HeadBranch   string    `json:"head_branch"`
	HeadSHA      string    `json:"head_sha"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	URL          string    `json:"url"`
	Repository   string    `json:"repository"`
	PullRequests []int     `json:"pull_requests,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsEmpty returns true if there are no pending items
func (p *PendingItems) IsEmpty() bool {
	return len(p.ReviewRequests) == 0 &&
		len(p.AssignedPRs) == 0 &&
		len(p.AssignedIssues) == 0 &&
		len(p.PRsWithChanges) == 0 &&
		len(p.Mentions) == 0 &&
		len(p.UnreadComments) == 0 &&
		len(p.Notifications) == 0 &&
		len(p.FailedWorkflows) == 0
}

// Count returns the total number of pending items
func (p *PendingItems) Count() int {
	return len(p.ReviewRequests) + len(p.AssignedPRs) + len(p.AssignedIssues) + len(p.PRsWithChanges) +
		len(p.Mentions) + len(p.UnreadComments) + len(p.Notifications) + len(p.FailedWorkflows)
}

// FromGitHubPullRequest converts a GitHub pull request to our PullRequestInfo
func FromGitHubPullRequest(pr *github.PullRequest) PullRequestInfo {
	info := PullRequestInfo{
		Number:    pr.GetNumber(),
		Title:     pr.GetTitle(),
		URL:       pr.GetHTMLURL(),
		CreatedAt: pr.GetCreatedAt().Time,
		UpdatedAt: pr.GetUpdatedAt().Time,
	}

	if pr.User != nil {
		info.Author = pr.User.GetLogin()
	}

	// Extract requested reviewers
	if pr.RequestedReviewers != nil {
		for _, reviewer := range pr.RequestedReviewers {
			if reviewer.Login != nil {
				info.RequestedReviewers = append(info.RequestedReviewers, reviewer.GetLogin())
			}
		}
	}

	return info
}

// FromGitHubIssue converts a GitHub issue to our IssueInfo
func FromGitHubIssue(issue *github.Issue) IssueInfo {
	info := IssueInfo{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		URL:       issue.GetHTMLURL(),
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
	}

	if issue.User != nil {
		info.Author = issue.User.GetLogin()
	}

	// Extract labels
	for _, label := range issue.Labels {
		info.Labels = append(info.Labels, label.GetName())
	}

	return info
}

// FromGitHubReview converts a GitHub review to our ReviewInfo
func FromGitHubReview(review *github.PullRequestReview) ReviewInfo {
	info := ReviewInfo{
		State:       review.GetState(),
		SubmittedAt: review.GetSubmittedAt().Time,
	}

	if review.User != nil {
		info.Author = review.User.GetLogin()
	}

	return info
}

// FromGitHubRepository converts a GitHub repository to our RepositoryInfo
func FromGitHubRepository(repo *github.Repository) RepositoryInfo {
	info := RepositoryInfo{
		Name:        repo.GetName(),
		FullName:    repo.GetFullName(),
		URL:         repo.GetHTMLURL(),
		Private:     repo.GetPrivate(),
		CreatedAt:   repo.GetCreatedAt().Time,
		UpdatedAt:   repo.GetUpdatedAt().Time,
		Description: repo.GetDescription(),
	}

	if repo.Owner != nil {
		info.Owner = repo.Owner.GetLogin()
	}

	if repo.DefaultBranch != nil {
		info.DefaultBranch = *repo.DefaultBranch
	} else {
		info.DefaultBranch = "main" // fallback
	}

	return info
}

// FromGitHubIssueComment converts a GitHub issue comment to our MentionInfo or CommentInfo
func FromGitHubIssueComment(comment *github.IssueComment, issueNumber int, issueTitle, repoName, itemType string) MentionInfo {
	info := MentionInfo{
		Number:     issueNumber,
		Title:      issueTitle,
		URL:        comment.GetHTMLURL(),
		Repository: repoName,
		Type:       itemType,
		Body:       comment.GetBody(),
		CreatedAt:  comment.GetCreatedAt().Time,
	}

	if comment.User != nil {
		info.Author = comment.User.GetLogin()
	}

	return info
}

// FromGitHubNotification converts a GitHub notification to our NotificationInfo
func FromGitHubNotification(notification *github.Notification) NotificationInfo {
	info := NotificationInfo{
		ID:        notification.GetID(),
		Reason:    notification.GetReason(),
		Subject:   notification.Subject.GetTitle(),
		URL:       notification.Subject.GetURL(),
		UpdatedAt: notification.GetUpdatedAt().Time,
		Unread:    notification.GetUnread(),
	}

	if notification.Repository != nil {
		info.Repository = notification.Repository.GetFullName()
	}

	return info
}

// FromGitHubWorkflowRun converts a GitHub workflow run to our WorkflowInfo
func FromGitHubWorkflowRun(run *github.WorkflowRun) WorkflowInfo {
	info := WorkflowInfo{
		ID:         run.GetID(),
		Name:       run.GetName(),
		HeadBranch: run.GetHeadBranch(),
		HeadSHA:    run.GetHeadSHA(),
		Status:     run.GetStatus(),
		Conclusion: run.GetConclusion(),
		URL:        run.GetHTMLURL(),
		CreatedAt:  run.GetCreatedAt().Time,
		UpdatedAt:  run.GetUpdatedAt().Time,
	}

	if run.Repository != nil {
		info.Repository = run.Repository.GetFullName()
	}

	// Extract associated PR numbers
	for _, pr := range run.PullRequests {
		if pr != nil {
			info.PullRequests = append(info.PullRequests, pr.GetNumber())
		}
	}

	return info
}
