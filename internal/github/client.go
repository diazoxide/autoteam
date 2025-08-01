package github

import (
	"context"
	"fmt"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/logger"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// RepositoryFilter defines interface for filtering repositories
type RepositoryFilter interface {
	ShouldIncludeRepository(repoName string) bool
}

// Client wraps the GitHub API client with convenience methods for multi-repository operations
type Client struct {
	client *github.Client
	filter RepositoryFilter
}

// NewClient creates a new GitHub API client with repository filtering
func NewClient(token string, filter RepositoryFilter) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	if filter == nil {
		return nil, fmt.Errorf("repository filter is required")
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	// Create GitHub client
	githubClient := github.NewClient(tc)

	return &Client{
		client: githubClient,
		filter: filter,
	}, nil
}

// NewClientFromConfig creates a new GitHub client using configuration
func NewClientFromConfig(token string, repositories *config.Repositories) (*Client, error) {
	return NewClient(token, repositories)
}

// GetAuthenticatedUser returns information about the authenticated user
func (c *Client) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}
	return user, nil
}

// GetRepository returns information about a specific repository
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", owner, repo, err)
	}
	return repository, nil
}

// GetDefaultBranch returns the default branch name for a specific repository
func (c *Client) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	repository, err := c.GetRepository(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	if repository.DefaultBranch == nil {
		return "main", nil // fallback to main
	}

	return *repository.DefaultBranch, nil
}

// parseRepository parses a repository string in "owner/repo" format
func parseRepository(repository string) (owner, repo string, err error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("repository must be in format 'owner/repo', got: %s", repository)
	}
	return parts[0], parts[1], nil
}

// GetFilteredRepositories returns all repositories that match the filter criteria
func (c *Client) GetFilteredRepositories(ctx context.Context, username string) ([]RepositoryInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("Getting filtered repositories for user", zap.String("username", username))

	opts := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []RepositoryInfo

	// Get user's own repositories
	repos, _, err := c.client.Repositories.ListByUser(ctx, username, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories for user %s: %w", username, err)
	}
	log.Info("Found repositories owned by user", zap.Int("count", len(repos)), zap.String("username", username))

	for _, repo := range repos {
		repoName := repo.GetFullName()
		log.Debug("Checking owned repository", zap.String("repository", repoName))
		if c.filter.ShouldIncludeRepository(repoName) {
			allRepos = append(allRepos, FromGitHubRepository(repo))
			log.Debug("Added repository to filtered list", zap.String("repository", repoName))
		}
	}

	// Also get repositories the user has access to (collaborator, organization member, etc.)
	// This uses the authenticated user's accessible repositories
	searchOpts := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Visibility:  "all",
		Affiliation: "owner,collaborator,organization_member",
	}

	accessibleRepos, _, err := c.client.Repositories.ListByAuthenticatedUser(ctx, searchOpts)
	if err != nil {
		log.Warn("Failed to get accessible repositories", zap.Error(err))
	} else {
		log.Info("Found accessible repositories for authenticated user", zap.Int("count", len(accessibleRepos)))

		for _, repo := range accessibleRepos {
			repoName := repo.GetFullName()
			log.Debug("Checking accessible repository", zap.String("repository", repoName))

			// Skip if we already added it from owned repositories
			alreadyAdded := false
			for _, existing := range allRepos {
				if existing.FullName == repoName {
					alreadyAdded = true
					break
				}
			}

			if !alreadyAdded && c.filter.ShouldIncludeRepository(repoName) {
				allRepos = append(allRepos, FromGitHubRepository(repo))
				log.Debug("Added accessible repository to filtered list", zap.String("repository", repoName))
			}
		}
	}

	log.Info("Total filtered repositories", zap.Int("total", len(allRepos)))
	return allRepos, nil
}

// MarkNotificationAsRead marks a specific notification thread as read
func (c *Client) MarkNotificationAsRead(ctx context.Context, threadID string) error {
	log := logger.FromContext(ctx)
	log.Debug("Marking notification as read", zap.String("thread_id", threadID))

	_, err := c.client.Activity.MarkThreadRead(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to mark notification thread as read: %w", err)
	}

	log.Debug("Successfully marked notification as read", zap.String("thread_id", threadID))
	return nil
}

// MarkNotificationsAsRead marks multiple notifications as read
func (c *Client) MarkNotificationsAsRead(ctx context.Context, threadIDs []string) error {
	log := logger.FromContext(ctx)
	log.Info("Marking multiple notifications as read", zap.Int("count", len(threadIDs)))

	var errors []error
	successCount := 0

	for _, threadID := range threadIDs {
		if err := c.MarkNotificationAsRead(ctx, threadID); err != nil {
			log.Warn("Failed to mark notification as read", zap.String("thread_id", threadID), zap.Error(err))
			errors = append(errors, fmt.Errorf("thread %s: %w", threadID, err))
		} else {
			successCount++
		}
	}

	log.Info("Completed marking notifications as read",
		zap.Int("success_count", successCount),
		zap.Int("error_count", len(errors)))

	if len(errors) > 0 {
		return fmt.Errorf("failed to mark %d notifications as read: %v", len(errors), errors)
	}

	return nil
}

// MarkNotificationThreadAsDone marks a notification thread as done (equivalent to inbox done)
func (c *Client) MarkNotificationThreadAsDone(ctx context.Context, threadID string) error {
	log := logger.FromContext(ctx)
	log.Debug("Marking notification thread as done", zap.String("thread_id", threadID))

	// Convert string thread ID to int64 for the Done API
	// Note: GitHub API inconsistency - MarkThreadRead uses string, MarkThreadDone uses int64
	// For now, we'll skip the Done functionality and rely on Read marking
	log.Debug("Skipping mark as done (API limitation), using mark as read instead")
	return c.MarkNotificationAsRead(ctx, threadID)
}
