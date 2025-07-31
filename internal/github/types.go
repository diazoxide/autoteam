package github

import (
	"time"

	"github.com/google/go-github/v57/github"
)

// PendingItems represents all pending items that need attention
type PendingItems struct {
	ReviewRequests []PullRequestInfo `json:"review_requests"`
	AssignedPRs    []PullRequestInfo `json:"assigned_prs"`
	AssignedIssues []IssueInfo       `json:"assigned_issues"`
	PRsWithChanges []PullRequestInfo `json:"prs_with_changes"`
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

// IsEmpty returns true if there are no pending items
func (p *PendingItems) IsEmpty() bool {
	return len(p.ReviewRequests) == 0 &&
		len(p.AssignedPRs) == 0 &&
		len(p.AssignedIssues) == 0 &&
		len(p.PRsWithChanges) == 0
}

// Count returns the total number of pending items
func (p *PendingItems) Count() int {
	return len(p.ReviewRequests) + len(p.AssignedPRs) + len(p.AssignedIssues) + len(p.PRsWithChanges)
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
