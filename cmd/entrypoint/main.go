package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"auto-team/cmd/entrypoint/internal/agent"
	"auto-team/cmd/entrypoint/internal/config"
	"auto-team/cmd/entrypoint/internal/deps"
	"auto-team/cmd/entrypoint/internal/git"
	"auto-team/cmd/entrypoint/internal/github"
	"auto-team/cmd/entrypoint/internal/monitor"

	"github.com/urfave/cli/v3"
)

// Build-time variables (set by ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	app := &cli.Command{
		Name:    "autoteam-entrypoint",
		Usage:   "Auto-Team Agent Entrypoint - GitHub monitoring and AI agent execution",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
		Action:  runEntrypoint,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file (optional, uses env vars by default)",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Run in dry-run mode (don't execute AI agent)",
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
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup logging
	if cmd.Bool("verbose") || cfg.Debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	}

	log.Printf("Starting Auto-Team Entrypoint %s", Version)
	log.Printf("Agent: %s, Repository: %s", cfg.Agent.Name, cfg.GitHub.Repository)

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

	// Initialize GitHub client
	githubClient, err := github.NewClient(cfg.GitHub.Token, cfg.GitHub.Repository)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
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
	gitSetup := git.NewSetup(cfg.Git, cfg.GitHub)
	if err := gitSetup.Configure(ctx); err != nil {
		return fmt.Errorf("failed to setup git: %w", err)
	}

	// Initialize and start monitor
	monitorConfig := monitor.Config{
		CheckInterval: cfg.Monitoring.CheckInterval,
		MaxRetries:    cfg.Monitoring.MaxRetries,
		DryRun:        cmd.Bool("dry-run"),
		TeamName:      cfg.Git.TeamName,
	}

	mon := monitor.New(githubClient, selectedAgent, monitorConfig, cfg)

	log.Println("Starting monitoring loop...")
	return mon.Start(ctx)
}
