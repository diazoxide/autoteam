package entrypoint

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"autoteam/internal/config"
)

// Config represents the complete configuration for the entrypoint
type Config struct {
	GitHub       GitHubConfig
	Repositories *config.Repositories
	Agent        AgentConfig
	Git          GitConfig
	Monitoring   MonitoringConfig
	Dependencies DependenciesConfig
	Debug        bool
}

// GitHubConfig contains GitHub-related configuration
type GitHubConfig struct {
	Token string
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

	// Repositories configuration
	includeStr := os.Getenv("REPOSITORIES_INCLUDE")
	excludeStr := os.Getenv("REPOSITORIES_EXCLUDE")
	cfg.Repositories = BuildRepositoriesConfig(includeStr, excludeStr)

	// Validate that at least one repository is included
	if len(cfg.Repositories.Include) == 0 {
		return nil, fmt.Errorf("at least one repository must be configured via REPOSITORIES_INCLUDE")
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
	if len(c.Repositories.Include) == 0 {
		return fmt.Errorf("at least one repository must be configured via repositories include")
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

// parseRepositoriesFromString parses comma-separated repository patterns
func parseRepositoriesFromString(patterns string) []string {
	if patterns == "" {
		return nil
	}

	var result []string
	for _, pattern := range strings.Split(patterns, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			result = append(result, pattern)
		}
	}
	return result
}

// BuildRepositoriesConfig creates repositories configuration from environment variables
func BuildRepositoriesConfig(includeStr, excludeStr string) *config.Repositories {
	repositories := &config.Repositories{}

	// Parse include patterns
	repositories.Include = parseRepositoriesFromString(includeStr)

	// Parse exclude patterns
	repositories.Exclude = parseRepositoriesFromString(excludeStr)

	return repositories
}
