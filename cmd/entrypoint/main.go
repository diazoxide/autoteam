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
	"autoteam/internal/logger"
	"autoteam/internal/monitor"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
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
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set log level (debug, info, warn, error)",
				Value:   "info",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runEntrypoint(ctx context.Context, cmd *cli.Command) error {
	// Setup structured logger first
	logLevelStr := cmd.String("log-level")

	// Handle legacy debug/verbose flags
	if cmd.Bool("debug") {
		logLevelStr = "debug"
	} else if cmd.Bool("verbose") {
		logLevelStr = "debug"
	}

	logLevel, err := logger.ParseLogLevel(logLevelStr)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	ctx, err = logger.SetupContext(ctx, logLevel)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	log := logger.FromContext(ctx)
	log.Info("Starting AutoTeam Entrypoint",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
		zap.String("log_level", string(logLevel)),
	)

	// Build configuration from CLI flags and environment variables
	cfg, err := buildConfigFromFlags(cmd)
	if err != nil {
		log.Error("Failed to build configuration", zap.Error(err))
		return fmt.Errorf("failed to build configuration: %w", err)
	}

	// Load MCP servers from environment
	if mcpServers, err := entrypoint.LoadMCPServers(); err != nil {
		log.Error("Failed to load MCP servers", zap.Error(err))
		return fmt.Errorf("failed to load MCP servers: %w", err)
	} else {
		cfg.MCPServers = mcpServers
	}

	// Validate configuration
	if err = cfg.Validate(); err != nil {
		log.Error("Invalid configuration", zap.Error(err))
		return fmt.Errorf("invalid configuration: %w", err)
	}

	log.Info("Configuration loaded successfully",
		zap.String("agent_name", cfg.Agent.Name),
		zap.String("agent_type", cfg.Agent.Type),
		zap.Strings("repositories_include", cfg.Repositories.Include),
		zap.Strings("repositories_exclude", cfg.Repositories.Exclude),
		zap.String("team_name", cfg.Git.TeamName),
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("Received signal, shutting down gracefully", zap.String("signal", sig.String()))
		cancel()
	}()

	// Initialize GitHub client with repositories filter
	log.Debug("Initializing GitHub client")
	githubClient, err := github.NewClientFromConfig(cfg.GitHub.Token, cfg.Repositories)
	if err != nil {
		log.Error("Failed to create GitHub client", zap.Error(err))
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Validate GitHub token and user for security
	err = validateGitHubTokenAndUser(ctx, githubClient, cfg.Git.User)
	if err != nil {
		log.Error("GitHub token/user validation failed", zap.Error(err))
		return fmt.Errorf("GitHub token/user validation failed: %w", err)
	}

	// Initialize agent registry and register available agents
	log.Debug("Initializing agent registry")
	agentRegistry := agent.NewRegistry()
	claudeAgent := agent.NewClaudeAgentWithMCP(cfg.Agent, cfg.MCPServers)
	agentRegistry.Register("claude", claudeAgent)

	// Get the configured agent
	selectedAgent, err := agentRegistry.Get(cfg.Agent.Type)
	if err != nil {
		log.Error("Failed to get agent", zap.String("agent_type", cfg.Agent.Type), zap.Error(err))
		return fmt.Errorf("failed to get agent %s: %w", cfg.Agent.Type, err)
	}
	log.Info("Agent initialized successfully", zap.String("agent_type", cfg.Agent.Type))

	// Install dependencies if needed
	log.Debug("Installing dependencies")
	installer := deps.NewInstaller(cfg.Dependencies)
	if err := installer.Install(ctx, selectedAgent); err != nil {
		log.Error("Failed to install dependencies", zap.Error(err))
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// Note: Agent MCP configuration happens after repo discovery in notification processing

	// Setup Git configuration and credentials
	log.Debug("Setting up Git configuration")
	gitSetup := git.NewSetup(cfg.Git, cfg.GitHub, cfg.Repositories)
	if err := gitSetup.Configure(ctx); err != nil {
		log.Error("Failed to setup git", zap.Error(err))
		return fmt.Errorf("failed to setup git: %w", err)
	}

	// Initialize and start monitor with simplified config
	monitorConfig := monitor.Config{
		CheckInterval: cfg.Monitoring.CheckInterval,
		DryRun:        cmd.Bool("dry-run"),
		TeamName:      cfg.Git.TeamName,
	}

	mon := monitor.New(githubClient, selectedAgent, monitorConfig, cfg)

	log.Info("Starting simplified notification monitoring loop",
		zap.Duration("check_interval", cfg.Monitoring.CheckInterval),
		zap.Bool("dry_run", cmd.Bool("dry-run")),
	)
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
	log := logger.FromContext(ctx)

	if expectedUser == "" {
		log.Warn("No GitHub user specified for validation, skipping token/user validation")
		return nil
	}

	log.Info("Validating GitHub token", zap.String("expected_user", expectedUser))

	// Get the authenticated user from the token
	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		log.Error("Failed to get authenticated user from token", zap.Error(err))
		return fmt.Errorf("failed to get authenticated user from token: %w", err)
	}

	// Check if the token belongs to the expected user
	var actualUser string
	if user.Login != nil {
		actualUser = *user.Login
	} else {
		log.Error("Unable to determine authenticated user from token")
		return fmt.Errorf("unable to determine authenticated user from token")
	}

	if actualUser != expectedUser {
		log.Error("Security validation failed: user mismatch",
			zap.String("actual_user", actualUser),
			zap.String("expected_user", expectedUser),
		)
		return fmt.Errorf("security validation failed: token belongs to user '%s' but expected user '%s'", actualUser, expectedUser)
	}

	log.Info("GitHub token/user validation successful", zap.String("validated_user", actualUser))
	return nil
}
