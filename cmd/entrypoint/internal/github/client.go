package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client with convenience methods
type Client struct {
	client *github.Client
	owner  string
	repo   string
}

// NewClient creates a new GitHub API client
func NewClient(token, repository string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// Parse repository (owner/repo format)
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("repository must be in format 'owner/repo', got: %s", repository)
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	// Create GitHub client
	githubClient := github.NewClient(tc)

	return &Client{
		client: githubClient,
		owner:  parts[0],
		repo:   parts[1],
	}, nil
}

// GetAuthenticatedUser returns information about the authenticated user
func (c *Client) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}
	return user, nil
}

// GetRepository returns information about the repository
func (c *Client) GetRepository(ctx context.Context) (*github.Repository, error) {
	repo, _, err := c.client.Repositories.Get(ctx, c.owner, c.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", c.owner, c.repo, err)
	}
	return repo, nil
}

// GetDefaultBranch returns the default branch name for the repository
func (c *Client) GetDefaultBranch(ctx context.Context) (string, error) {
	repo, err := c.GetRepository(ctx)
	if err != nil {
		return "", err
	}

	if repo.DefaultBranch == nil {
		return "main", nil // fallback to main
	}

	return *repo.DefaultBranch, nil
}

// Owner returns the repository owner
func (c *Client) Owner() string {
	return c.owner
}

// Repo returns the repository name
func (c *Client) Repo() string {
	return c.repo
}

// Repository returns the full repository name (owner/repo)
func (c *Client) Repository() string {
	return fmt.Sprintf("%s/%s", c.owner, c.repo)
}
