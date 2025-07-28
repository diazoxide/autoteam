package git

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"auto-team/cmd/entrypoint/internal/config"
)

// Setup handles Git configuration and credential management
type Setup struct {
	gitConfig    config.GitConfig
	githubConfig config.GitHubConfig
}

// NewSetup creates a new Git setup instance
func NewSetup(gitConfig config.GitConfig, githubConfig config.GitHubConfig) *Setup {
	return &Setup{
		gitConfig:    gitConfig,
		githubConfig: githubConfig,
	}
}

// Configure sets up Git configuration and credentials
func (s *Setup) Configure(ctx context.Context) error {
	log.Println("Setting up Git configuration and credentials...")

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

	// Clone or update repository
	if err := s.setupRepository(ctx); err != nil {
		return fmt.Errorf("failed to setup repository: %w", err)
	}

	log.Println("Git configuration completed successfully")
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
	// Set user name
	if s.gitConfig.User != "" {
		cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.name", s.gitConfig.User)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.name: %w", err)
		}
		log.Printf("Set git user.name to: %s", s.gitConfig.User)
	}

	// Set user email
	if s.gitConfig.Email != "" {
		cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.email", s.gitConfig.Email)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.email: %w", err)
		}
		log.Printf("Set git user.email to: %s", s.gitConfig.Email)
	}

	return nil
}

// configureCredentialHelper sets up the Git credential helper
func (s *Setup) configureCredentialHelper(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "credential.helper", "store")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set credential helper: %w", err)
	}
	log.Println("Configured git credential helper to use store")
	return nil
}

// setupCredentialsFile creates the Git credentials file with the GitHub token
func (s *Setup) setupCredentialsFile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	credentialsPath := filepath.Join(homeDir, ".git-credentials")

	// Create credentials content with HTTPS token
	credentialsContent := fmt.Sprintf("https://%s:%s@github.com", s.gitConfig.User, s.githubConfig.Token)

	// Write credentials file
	if err := os.WriteFile(credentialsPath, []byte(credentialsContent), 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	log.Printf("Created git credentials file at: %s", credentialsPath)
	return nil
}

// setupRepository clones or updates the repository
func (s *Setup) setupRepository(ctx context.Context) error {
	workingDir := s.getWorkingDirectory()

	// Create the working directory if it doesn't exist
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	// Change to the working directory
	if err := os.Chdir(workingDir); err != nil {
		return fmt.Errorf("failed to change to working directory: %w", err)
	}

	// Check if repository is already cloned
	if s.isRepositoryCloned() {
		log.Println("Repository already cloned, updating...")
		return s.updateRepository(ctx)
	}

	log.Println("Cloning repository...")
	return s.cloneRepository(ctx)
}

// isRepositoryCloned checks if the repository is already cloned
func (s *Setup) isRepositoryCloned() bool {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return false
	}
	return true
}

// cloneRepository clones the repository using HTTPS
func (s *Setup) cloneRepository(ctx context.Context) error {
	repoURL := fmt.Sprintf("https://github.com/%s.git", s.githubConfig.Repository)

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Printf("Successfully cloned repository: %s", s.githubConfig.Repository)
	return nil
}

// updateRepository updates the existing repository
func (s *Setup) updateRepository(ctx context.Context) error {
	// Fetch latest changes
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to fetch from origin: %v", err)
	}

	log.Println("Repository updated")
	return nil
}

// getWorkingDirectory returns the working directory path
func (s *Setup) getWorkingDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return filepath.Join(".", s.gitConfig.TeamName, "codebase")
	}
	return filepath.Join(homeDir, s.gitConfig.TeamName, "codebase")
}

// GetWorkingDirectory returns the working directory path (public method)
func (s *Setup) GetWorkingDirectory() string {
	return s.getWorkingDirectory()
}

// SwitchToMainBranch switches to the main branch and resets to origin
func (s *Setup) SwitchToMainBranch(ctx context.Context, mainBranch string) error {
	workingDir := s.getWorkingDirectory()

	// Change to working directory
	if err := os.Chdir(workingDir); err != nil {
		return fmt.Errorf("failed to change to working directory: %w", err)
	}

	// Fetch latest changes
	cmd := exec.CommandContext(ctx, "git", "fetch")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to fetch: %v", err)
	}

	// Checkout main branch
	cmd = exec.CommandContext(ctx, "git", "checkout", mainBranch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout %s branch: %w", mainBranch, err)
	}

	// Hard reset to origin
	originBranch := fmt.Sprintf("origin/%s", mainBranch)
	cmd = exec.CommandContext(ctx, "git", "reset", "--hard", originBranch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset to %s: %w", originBranch, err)
	}

	log.Printf("Switched to %s branch and reset to origin", mainBranch)
	return nil
}
