package github

import (
	"context"
	"fmt"
	"log"
	"strings"

	"autoteam/internal/config"

	"github.com/google/go-github/v57/github"
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
	log.Printf("Getting filtered repositories for user: %s", username)

	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []RepositoryInfo

	// Get user's own repositories
	repos, _, err := c.client.Repositories.List(ctx, username, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories for user %s: %w", username, err)
	}
	log.Printf("Found %d repositories owned by user %s", len(repos), username)

	for _, repo := range repos {
		repoName := repo.GetFullName()
		log.Printf("Checking owned repository: %s", repoName)
		if c.filter.ShouldIncludeRepository(repoName) {
			allRepos = append(allRepos, FromGitHubRepository(repo))
			log.Printf("Added repository to filtered list: %s", repoName)
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
		log.Printf("Warning: failed to get accessible repositories: %v", err)
	} else {
		log.Printf("Found %d accessible repositories for authenticated user", len(accessibleRepos))

		for _, repo := range accessibleRepos {
			repoName := repo.GetFullName()
			log.Printf("Checking accessible repository: %s", repoName)

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
				log.Printf("Added accessible repository to filtered list: %s", repoName)
			}
		}
	}

	log.Printf("Total filtered repositories: %d", len(allRepos))
	return allRepos, nil
}
