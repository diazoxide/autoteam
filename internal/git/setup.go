package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// Setup handles Git configuration and credential management for multiple repositories
type Setup struct {
	gitConfig    entrypoint.GitConfig
	githubConfig entrypoint.GitHubConfig
	repositories *config.Repositories
	clonedRepos  map[string]bool // tracks which repositories have been cloned
}

// NewSetup creates a new Git setup instance for multi-repository operations
func NewSetup(gitConfig entrypoint.GitConfig, githubConfig entrypoint.GitHubConfig, repositories *config.Repositories) *Setup {
	return &Setup{
		gitConfig:    gitConfig,
		githubConfig: githubConfig,
		repositories: repositories,
		clonedRepos:  make(map[string]bool),
	}
}

// getRepositoryOwner extracts the owner from repository URL (e.g., "owner/repo" -> "owner")
func (s *Setup) getRepositoryOwner(repository string) string {
	parts := strings.Split(repository, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// normalizeRepositoryName converts "owner/repo" to "owner-repo" for directory names
func (s *Setup) normalizeRepositoryName(repository string) string {
	return strings.ReplaceAll(repository, "/", "-")
}

// Configure sets up Git configuration and credentials for multi-repository operations
func (s *Setup) Configure(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Setting up Git configuration and credentials")

	// Ensure git is available
	if err := s.checkGitAvailable(ctx); err != nil {
		return fmt.Errorf("git is not available: %w", err)
	}

	// Set up global Git configuration
	if err := s.configureGitUser(ctx); err != nil {
		return fmt.Errorf("failed to configure git user: %w", err)
	}

	// Set up credential helper
	if err := s.configureCredentialHelper(ctx); err != nil {
		return fmt.Errorf("failed to configure credential helper: %w", err)
	}

	// Set up credentials file
	if err := s.setupCredentialsFile(); err != nil {
		return fmt.Errorf("failed to setup credentials file: %w", err)
	}

	lgr.Info("Git configuration completed successfully")
	lgr.Info("Repositories will be cloned on-demand when needed")
	return nil
}

// checkGitAvailable verifies that git is available
func (s *Setup) checkGitAvailable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command not found: %w", err)
	}
	return nil
}

// configureGitUser sets up the global Git user configuration
func (s *Setup) configureGitUser(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	// Determine user name - use provided user or fall back to first repository owner
	userName := s.gitConfig.User
	if userName == "" && len(s.repositories.Include) > 0 {
		userName = s.getRepositoryOwner(s.repositories.Include[0])
	}

	// Set user name
	if userName != "" {
		cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.name", userName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.name: %w", err)
		}
		lgr.Info("Set git user.name", zap.String("user_name", userName))
	}

	// Determine email - use provided email or generate from user name
	userEmail := s.gitConfig.Email
	if userEmail == "" && userName != "" {
		userEmail = userName + "@users.noreply.github.com"
	}

	// Set user email
	if userEmail != "" {
		cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.email", userEmail)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.email: %w", err)
		}
		lgr.Info("Set git user.email", zap.String("user_email", userEmail))
	}

	return nil
}

// configureCredentialHelper sets up the Git credential helper
func (s *Setup) configureCredentialHelper(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "credential.helper", "store")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set credential helper: %w", err)
	}
	lgr.Info("Configured git credential helper to use store")
	return nil
}

// setupCredentialsFile creates the Git credentials file with the GitHub token
func (s *Setup) setupCredentialsFile() error {
	lgr, err := logger.NewLogger(logger.InfoLevel)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	credentialsPath := filepath.Join(homeDir, ".git-credentials")

	// Create credentials content with HTTPS token
	// For GitHub Personal Access Tokens, use the token as username and leave password empty
	// or use token:x-oauth-basic format
	credentialsContent := fmt.Sprintf("https://%s:x-oauth-basic@github.com", s.githubConfig.Token)

	// Write credentials file
	if err := os.WriteFile(credentialsPath, []byte(credentialsContent), 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	lgr.Info("Created git credentials file", zap.String("path", credentialsPath))
	lgr.Info("Will support all repositories configured for this agent")
	return nil
}

// SetupRepository clones or updates a specific repository on-demand
func (s *Setup) SetupRepository(ctx context.Context, repository string) error {
	lgr := logger.FromContext(ctx)
	if !s.repositories.ShouldIncludeRepository(repository) {
		return fmt.Errorf("repository %s is not included in the configured patterns", repository)
	}

	workingDir := s.getRepositoryWorkingDirectory(repository)

	// Create the working directory if it doesn't exist
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory for %s: %w", repository, err)
	}

	// Check if repository is already cloned
	if s.isRepositoryCloned(repository) {
		lgr.Info("Repository already cloned, updating", zap.String("repository", repository))
		return s.updateRepository(ctx, repository)
	}

	lgr.Info("Cloning repository", zap.String("repository", repository))
	return s.cloneRepository(ctx, repository)
}

// isRepositoryCloned checks if a specific repository is already cloned
func (s *Setup) isRepositoryCloned(repository string) bool {
	workingDir := s.getRepositoryWorkingDirectory(repository)
	gitDir := filepath.Join(workingDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}
	s.clonedRepos[repository] = true
	return true
}

// cloneRepository clones a specific repository using HTTPS
func (s *Setup) cloneRepository(ctx context.Context, repository string) error {
	lgr := logger.FromContext(ctx)
	workingDir := s.getRepositoryWorkingDirectory(repository)
	repoURL := fmt.Sprintf("https://github.com/%s.git", repository)

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, workingDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository %s: %w", repository, err)
	}

	s.clonedRepos[repository] = true
	lgr.Info("Successfully cloned repository", zap.String("repository", repository), zap.String("working_dir", workingDir))
	return nil
}

// updateRepository updates an existing repository
func (s *Setup) updateRepository(ctx context.Context, repository string) error {
	lgr := logger.FromContext(ctx)
	workingDir := s.getRepositoryWorkingDirectory(repository)

	// Change to repository directory
	if err := os.Chdir(workingDir); err != nil {
		return fmt.Errorf("failed to change to repository directory %s: %w", workingDir, err)
	}

	// Fetch latest changes
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	if err := cmd.Run(); err != nil {
		lgr.Warn("Failed to fetch from origin", zap.String("repository", repository), zap.Error(err))
	}

	lgr.Info("Repository updated", zap.String("repository", repository))
	return nil
}

// getRepositoryWorkingDirectory returns the working directory path for a specific repository
func (s *Setup) getRepositoryWorkingDirectory(repository string) string {
	// Use agent-specific path for container deployments
	// Require normalized name for directory structure
	agentName := os.Getenv("AGENT_NORMALIZED_NAME")
	if agentName == "" {
		panic("AGENT_NORMALIZED_NAME environment variable is required but not set")
	}

	// Normalize repository name for directory structure
	normalizedRepo := s.normalizeRepositoryName(repository)
	return fmt.Sprintf("/opt/autoteam/agents/%s/codebase/%s", agentName, normalizedRepo)
}

// GetRepositoryWorkingDirectory returns the working directory path for a repository (public method)
func (s *Setup) GetRepositoryWorkingDirectory(repository string) string {
	return s.getRepositoryWorkingDirectory(repository)
}

// GetWorkingDirectory returns the base working directory path
func (s *Setup) GetWorkingDirectory() string {
	agentName := os.Getenv("AGENT_NORMALIZED_NAME")
	if agentName == "" {
		panic("AGENT_NORMALIZED_NAME environment variable is required but not set")
	}
	return fmt.Sprintf("/opt/autoteam/agents/%s/codebase", agentName)
}

// SwitchToMainBranch switches to the main branch and resets to origin for a specific repository
func (s *Setup) SwitchToMainBranch(ctx context.Context, repository, mainBranch string) error {
	lgr := logger.FromContext(ctx)
	workingDir := s.getRepositoryWorkingDirectory(repository)

	// Change to working directory
	if err := os.Chdir(workingDir); err != nil {
		return fmt.Errorf("failed to change to working directory %s: %w", workingDir, err)
	}

	// Fetch latest changes
	cmd := exec.CommandContext(ctx, "git", "fetch")
	if err := cmd.Run(); err != nil {
		lgr.Warn("Failed to fetch", zap.String("repository", repository), zap.Error(err))
	}

	// Checkout main branch
	cmd = exec.CommandContext(ctx, "git", "checkout", mainBranch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout %s branch in %s: %w", mainBranch, repository, err)
	}

	// Hard reset to origin
	originBranch := fmt.Sprintf("origin/%s", mainBranch)
	cmd = exec.CommandContext(ctx, "git", "reset", "--hard", originBranch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset to %s in %s: %w", originBranch, repository, err)
	}

	lgr.Info("Switched to main branch and reset to origin", zap.String("branch", mainBranch), zap.String("repository", repository))
	return nil
}
