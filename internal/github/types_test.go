package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/stretchr/testify/assert"
)

func TestPendingItems_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		items    *PendingItems
		expected bool
	}{
		{
			name:     "empty items",
			items:    &PendingItems{},
			expected: true,
		},
		{
			name: "has review requests",
			items: &PendingItems{
				ReviewRequests: []PullRequestInfo{{Number: 1}},
			},
			expected: false,
		},
		{
			name: "has mentions",
			items: &PendingItems{
				Mentions: []MentionInfo{{Number: 1}},
			},
			expected: false,
		},
		{
			name: "has notifications",
			items: &PendingItems{
				Notifications: []NotificationInfo{{ID: "1"}},
			},
			expected: false,
		},
		{
			name: "has failed workflows",
			items: &PendingItems{
				FailedWorkflows: []WorkflowInfo{{ID: 1}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.items.IsEmpty())
		})
	}
}

func TestPendingItems_Count(t *testing.T) {
	items := &PendingItems{
		ReviewRequests:  []PullRequestInfo{{Number: 1}, {Number: 2}},
		AssignedPRs:     []PullRequestInfo{{Number: 3}},
		AssignedIssues:  []IssueInfo{{Number: 4}},
		PRsWithChanges:  []PullRequestInfo{{Number: 5}},
		Mentions:        []MentionInfo{{Number: 6}},
		UnreadComments:  []CommentInfo{{Number: 7}},
		Notifications:   []NotificationInfo{{ID: "8"}},
		FailedWorkflows: []WorkflowInfo{{ID: 9}},
	}

	assert.Equal(t, 9, items.Count())
}

func TestFromGitHubIssueComment(t *testing.T) {
	now := time.Now()
	login := "testuser"
	body := "Test comment body"
	url := "https://github.com/owner/repo/issues/1#issuecomment-123"

	comment := &github.IssueComment{
		User: &github.User{
			Login: &login,
		},
		Body:      &body,
		HTMLURL:   &url,
		CreatedAt: &github.Timestamp{Time: now},
	}

	result := FromGitHubIssueComment(comment, 1, "Test Issue", "owner/repo", "issue")

	assert.Equal(t, 1, result.Number)
	assert.Equal(t, "Test Issue", result.Title)
	assert.Equal(t, url, result.URL)
	assert.Equal(t, "owner/repo", result.Repository)
	assert.Equal(t, "issue", result.Type)
	assert.Equal(t, body, result.Body)
	assert.Equal(t, login, result.Author)
	assert.Equal(t, now, result.CreatedAt)
}

func TestFromGitHubNotification(t *testing.T) {
	id := "123"
	reason := "mention"
	title := "Test notification"
	url := "https://api.github.com/notifications/123"
	unread := true
	now := time.Now()
	repoName := "owner/repo"

	notification := &github.Notification{
		ID:     &id,
		Reason: &reason,
		Subject: &github.NotificationSubject{
			Title: &title,
			URL:   &url,
		},
		Unread:    &unread,
		UpdatedAt: &github.Timestamp{Time: now},
		Repository: &github.Repository{
			FullName: &repoName,
		},
	}

	result := FromGitHubNotification(notification)

	assert.Equal(t, id, result.ID)
	assert.Equal(t, reason, result.Reason)
	assert.Equal(t, title, result.Subject)
	assert.Equal(t, url, result.URL)
	assert.Equal(t, unread, result.Unread)
	assert.Equal(t, now, result.UpdatedAt)
	assert.Equal(t, repoName, result.Repository)
}

func TestFromGitHubWorkflowRun(t *testing.T) {
	id := int64(12345)
	name := "CI Tests"
	headBranch := "feature/test"
	headSHA := "abc123"
	status := "completed"
	conclusion := "failure"
	htmlURL := "https://github.com/owner/repo/actions/runs/12345"
	now := time.Now()
	repoName := "owner/repo"
	prNumber := 42

	run := &github.WorkflowRun{
		ID:         &id,
		Name:       &name,
		HeadBranch: &headBranch,
		HeadSHA:    &headSHA,
		Status:     &status,
		Conclusion: &conclusion,
		HTMLURL:    &htmlURL,
		CreatedAt:  &github.Timestamp{Time: now},
		UpdatedAt:  &github.Timestamp{Time: now},
		Repository: &github.Repository{
			FullName: &repoName,
		},
		PullRequests: []*github.PullRequest{
			{Number: &prNumber},
		},
	}

	result := FromGitHubWorkflowRun(run)

	assert.Equal(t, id, result.ID)
	assert.Equal(t, name, result.Name)
	assert.Equal(t, headBranch, result.HeadBranch)
	assert.Equal(t, headSHA, result.HeadSHA)
	assert.Equal(t, status, result.Status)
	assert.Equal(t, conclusion, result.Conclusion)
	assert.Equal(t, htmlURL, result.URL)
	assert.Equal(t, now, result.CreatedAt)
	assert.Equal(t, now, result.UpdatedAt)
	assert.Equal(t, repoName, result.Repository)
	assert.Equal(t, []int{prNumber}, result.PullRequests)
}
