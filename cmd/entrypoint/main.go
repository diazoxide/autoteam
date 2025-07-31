package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/deps"
	"autoteam/internal/entrypoint"
	"autoteam/internal/git"
	"autoteam/internal/github"
	"autoteam/internal/monitor"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

// Build-time variables (set by ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load .env file if it exists (ignore errors for optional file)
	_ = godotenv.Load()

	app := &cli.Command{
		Name:    "autoteam-entrypoint",
		Usage:   "AutoTeam Agent Entrypoint - GitHub monitoring and AI agent execution",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
		Action:  runEntrypoint,
		Flags: []cli.Flag{
			// GitHub Configuration
			&cli.StringFlag{
				Name:     "gh-token",
				Usage:    "GitHub token for authentication",
				Required: true,
				Sources:  cli.EnvVars("GH_TOKEN"),
			},

			// Repositories Configuration (multi-repo mode)
			&cli.StringFlag{
				Name:     "repositories-include",
				Usage:    "Comma-separated list of repository patterns to include (supports regex with /pattern/)",
				Required: true,
				Sources:  cli.EnvVars("REPOSITORIES_INCLUDE"),
			},
			&cli.StringFlag{
				Name:    "repositories-exclude",
				Usage:   "Comma-separated list of repository patterns to exclude (supports regex with /pattern/)",
				Sources: cli.EnvVars("REPOSITORIES_EXCLUDE"),
			},

			// Agent Configuration
			&cli.StringFlag{
				Name:     "agent-name",
				Usage:    "Name of the agent",
				Required: true,
				Sources:  cli.EnvVars("AGENT_NAME"),
			},
			&cli.StringFlag{
				Name:    "agent-type",
				Value:   "claude",
				Usage:   "Type of agent to use",
				Sources: cli.EnvVars("AGENT_TYPE"),
			},
			&cli.StringFlag{
				Name:    "agent-prompt",
				Usage:   "Primary prompt for the agent",
				Sources: cli.EnvVars("AGENT_PROMPT"),
			},

			// Git Configuration (optional overrides)
			&cli.StringFlag{
				Name:    "git-user",
				Usage:   "Git user name (defaults to repository owner)",
				Sources: cli.EnvVars("GH_USER"),
			},
			&cli.StringFlag{
				Name:    "git-email",
				Usage:   "Git user email (defaults to {user}@users.noreply.github.com)",
				Sources: cli.EnvVars("GH_EMAIL"),
			},
			&cli.StringFlag{
				Name:    "team-name",
				Value:   "autoteam",
				Usage:   "Team name for directory structure",
				Sources: cli.EnvVars("TEAM_NAME"),
			},

			// Monitoring Configuration
			&cli.IntFlag{
				Name:    "check-interval",
				Value:   60,
				Usage:   "Check interval in seconds",
				Sources: cli.EnvVars("CHECK_INTERVAL"),
			},
			&cli.IntFlag{
				Name:    "max-retries",
				Value:   100,
				Usage:   "Maximum number of retries for operations",
				Sources: cli.EnvVars("MAX_RETRIES"),
			},
			&cli.IntFlag{
				Name:    "max-attempts",
				Value:   3,
				Usage:   "Maximum number of attempts per item before moving to cooldown",
				Sources: cli.EnvVars("MAX_ATTEMPTS"),
			},

			// Dependencies Configuration
			&cli.BoolFlag{
				Name:    "install-deps",
				Usage:   "Install dependencies on startup",
				Sources: cli.EnvVars("INSTALL_DEPS"),
			},

			// Runtime Configuration
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Run in dry-run mode (don't execute AI agent)",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging",
				Sources: cli.EnvVars("DEBUG"),
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose logging",
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runEntrypoint(ctx context.Context, cmd *cli.Command) error {
	// Build configuration from CLI flags
	cfg, err := buildConfigFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to build configuration: %w", err)
	}

	// Validate configuration
	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Setup logging
	if cmd.Bool("verbose") || cmd.Bool("debug") {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	}

	log.Printf("Starting AutoTeam Entrypoint %s", Version)
	log.Printf("Agent: %s", cfg.Agent.Name)
	if len(cfg.Repositories.Include) > 0 {
		log.Printf("Repositories Include: %v", cfg.Repositories.Include)
	}
	if len(cfg.Repositories.Exclude) > 0 {
		log.Printf("Repositories Exclude: %v", cfg.Repositories.Exclude)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Initialize GitHub client with repositories filter
	githubClient, err := github.NewClientFromConfig(cfg.GitHub.Token, cfg.Repositories)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Validate GitHub token and user for security
	err = validateGitHubTokenAndUser(ctx, githubClient, cfg.Git.User)
	if err != nil {
		return fmt.Errorf("GitHub token/user validation failed: %w", err)
	}

	// Initialize agent registry and register available agents
	agentRegistry := agent.NewRegistry()
	claudeAgent := agent.NewClaudeAgent(cfg.Agent)
	agentRegistry.Register("claude", claudeAgent)

	// Get the configured agent
	selectedAgent, err := agentRegistry.Get(cfg.Agent.Type)
	if err != nil {
		return fmt.Errorf("failed to get agent %s: %w", cfg.Agent.Type, err)
	}

	// Install dependencies if needed
	installer := deps.NewInstaller(cfg.Dependencies)
	if err := installer.Install(ctx, selectedAgent); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// Setup Git configuration and credentials
	gitSetup := git.NewSetup(cfg.Git, cfg.GitHub, cfg.Repositories)
	if err := gitSetup.Configure(ctx); err != nil {
		return fmt.Errorf("failed to setup git: %w", err)
	}

	// Initialize and start monitor
	monitorConfig := monitor.Config{
		CheckInterval: cfg.Monitoring.CheckInterval,
		MaxRetries:    cfg.Monitoring.MaxRetries,
		DryRun:        cmd.Bool("dry-run"),
		TeamName:      cfg.Git.TeamName,
		MaxAttempts:   cmd.Int("max-attempts"),
	}

	mon := monitor.New(githubClient, selectedAgent, monitorConfig, cfg)

	log.Println("Starting monitoring loop...")
	return mon.Start(ctx)
}

// buildConfigFromFlags builds a Config struct from CLI flags
func buildConfigFromFlags(cmd *cli.Command) (*entrypoint.Config, error) {
	cfg := &entrypoint.Config{}

	// GitHub configuration
	cfg.GitHub.Token = cmd.String("gh-token")

	// Repositories configuration (multi-repo mode)
	includeStr := cmd.String("repositories-include")
	excludeStr := cmd.String("repositories-exclude")
	cfg.Repositories = entrypoint.BuildRepositoriesConfig(includeStr, excludeStr)

	// Validate that at least one repository is included
	if len(cfg.Repositories.Include) == 0 {
		return nil, fmt.Errorf("at least one repository must be configured via REPOSITORIES_INCLUDE")
	}

	// Agent configuration
	cfg.Agent.Name = cmd.String("agent-name")
	cfg.Agent.Type = cmd.String("agent-type")
	cfg.Agent.Prompt = cmd.String("agent-prompt")

	// Git configuration
	cfg.Git.User = cmd.String("git-user")
	cfg.Git.Email = cmd.String("git-email")
	if cfg.Git.Email == "" && cfg.Git.User != "" {
		cfg.Git.Email = cfg.Git.User + "@users.noreply.github.com"
	}
	cfg.Git.TeamName = cmd.String("team-name")

	// Monitoring configuration
	checkInterval := cmd.Int("check-interval")
	cfg.Monitoring.CheckInterval = time.Duration(checkInterval) * time.Second
	cfg.Monitoring.MaxRetries = cmd.Int("max-retries")

	// Dependencies configuration
	cfg.Dependencies.InstallDeps = cmd.Bool("install-deps")

	// Debug configuration
	cfg.Debug = cmd.Bool("debug")

	return cfg, nil
}

// validateGitHubTokenAndUser validates that the GitHub token belongs to the expected user
func validateGitHubTokenAndUser(ctx context.Context, client *github.Client, expectedUser string) error {
	if expectedUser == "" {
		log.Println("Warning: No GitHub user specified for validation, skipping token/user validation")
		return nil
	}

	log.Printf("Validating GitHub token for user: %s", expectedUser)

	// Get the authenticated user from the token
	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated user from token: %w", err)
	}

	// Check if the token belongs to the expected user
	var actualUser string
	if user.Login != nil {
		actualUser = *user.Login
	} else {
		return fmt.Errorf("unable to determine authenticated user from token")
	}

	if actualUser != expectedUser {
		return fmt.Errorf("security validation failed: token belongs to user '%s' but expected user '%s'", actualUser, expectedUser)
	}

	log.Printf("GitHub token/user validation successful: token belongs to user '%s'", actualUser)
	return nil
}
