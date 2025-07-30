package entrypoint

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the complete configuration for the entrypoint
type Config struct {
	GitHub       GitHubConfig
	Agent        AgentConfig
	Git          GitConfig
	Monitoring   MonitoringConfig
	Dependencies DependenciesConfig
	Debug        bool
}

// GitHubConfig contains GitHub-related configuration
type GitHubConfig struct {
	Token      string
	Repository string
	Owner      string
	Repo       string
}

// AgentConfig contains AI agent configuration
type AgentConfig struct {
	Name   string
	Type   string
	Prompt string
}

// GitConfig contains Git-related configuration
type GitConfig struct {
	User     string
	Email    string
	TeamName string
}

// MonitoringConfig contains monitoring loop configuration
type MonitoringConfig struct {
	CheckInterval time.Duration
	MaxRetries    int
}

// DependenciesConfig contains dependency installation configuration
type DependenciesConfig struct {
	InstallDeps bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// GitHub configuration
	cfg.GitHub.Token = os.Getenv("GH_TOKEN")
	if cfg.GitHub.Token == "" {
		return nil, fmt.Errorf("GH_TOKEN environment variable is required")
	}

	cfg.GitHub.Repository = os.Getenv("GITHUB_REPO")
	if cfg.GitHub.Repository == "" {
		return nil, fmt.Errorf("GITHUB_REPO environment variable is required")
	}

	// Parse owner/repo from repository string
	if err := cfg.ParseRepository(); err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}

	// Agent configuration
	cfg.Agent.Name = os.Getenv("AGENT_NAME")
	if cfg.Agent.Name == "" {
		return nil, fmt.Errorf("AGENT_NAME environment variable is required")
	}

	cfg.Agent.Type = getEnvOrDefault("AGENT_TYPE", "claude")
	cfg.Agent.Prompt = os.Getenv("AGENT_PROMPT")

	// Git configuration
	cfg.Git.User = os.Getenv("GH_USER")
	cfg.Git.Email = getEnvOrDefault("GH_EMAIL", cfg.Git.User+"@users.noreply.github.com")
	cfg.Git.TeamName = getEnvOrDefault("TEAM_NAME", "autoteam")

	// Monitoring configuration
	checkInterval := getEnvOrDefault("CHECK_INTERVAL", "60")
	interval, err := strconv.Atoi(checkInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid CHECK_INTERVAL: %w", err)
	}
	cfg.Monitoring.CheckInterval = time.Duration(interval) * time.Second

	maxRetries := getEnvOrDefault("MAX_RETRIES", "100")
	cfg.Monitoring.MaxRetries, err = strconv.Atoi(maxRetries)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_RETRIES: %w", err)
	}

	// Dependencies configuration
	cfg.Dependencies.InstallDeps = getEnvOrDefault("INSTALL_DEPS", "false") == "true"

	// Debug configuration
	cfg.Debug = getEnvOrDefault("DEBUG", "false") == "true"

	return cfg, nil
}

// ParseRepository parses the GITHUB_REPO into owner and repo
func (c *Config) ParseRepository() error {
	repo := c.GitHub.Repository
	if repo == "" {
		return fmt.Errorf("repository is empty")
	}

	// Split owner/repo format
	parts := make([]string, 0, 2)
	current := ""
	for _, char := range repo {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) != 2 {
		return fmt.Errorf("repository must be in format 'owner/repo', got: %s", repo)
	}

	c.GitHub.Owner = parts[0]
	c.GitHub.Repo = parts[1]

	return nil
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.GitHub.Token == "" {
		return fmt.Errorf("GitHub token is required")
	}
	if c.GitHub.Owner == "" || c.GitHub.Repo == "" {
		return fmt.Errorf("GitHub repository owner and repo are required")
	}
	if c.Agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if c.Monitoring.CheckInterval < time.Second {
		return fmt.Errorf("check interval must be at least 1 second")
	}
	if c.Monitoring.MaxRetries < 1 {
		return fmt.Errorf("max retries must be at least 1")
	}
	return nil
}
