package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"autoteam/internal/entrypoint"
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
		Usage:   "AutoTeam Agent Entrypoint - AI agent execution via MCP servers",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
		Action:  runEntrypoint,
		Flags: []cli.Flag{
			// Primary configuration file
			&cli.StringFlag{
				Name:     "config-file",
				Aliases:  []string{"c"},
				Usage:    "Path to agent configuration file (YAML)",
				Required: true,
				Sources:  cli.EnvVars("CONFIG_FILE"),
			},

			// Runtime Configuration
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

	// Load configuration from file
	configPath := cmd.String("config-file")
	cfg, err := entrypoint.LoadFromFile(configPath)
	if err != nil {
		log.Error("Failed to load configuration from file", zap.String("config_path", configPath), zap.Error(err))
		return fmt.Errorf("failed to load configuration from file %s: %w", configPath, err)
	}

	// Validate configuration
	if err = cfg.Validate(); err != nil {
		log.Error("Invalid configuration", zap.Error(err))
		return fmt.Errorf("invalid configuration: %w", err)
	}

	log.Info("Configuration loaded successfully",
		zap.String("agent_name", cfg.Agent.Name),
		zap.String("agent_type", cfg.Agent.Type),
		zap.String("team_name", cfg.TeamName),
	)

	// Execute on_init hooks
	if hookErr := entrypoint.ExecuteHooks(ctx, cfg.Hooks, "on_init"); hookErr != nil {
		log.Error("Failed to execute on_init hooks", zap.Error(hookErr))
		return fmt.Errorf("failed to execute on_init hooks: %w", hookErr)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("Received signal, shutting down gracefully", zap.String("signal", sig.String()))

		// Execute on_stop hooks
		if hookErr := entrypoint.ExecuteHooks(ctx, cfg.Hooks, "on_stop"); hookErr != nil {
			log.Error("Failed to execute on_stop hooks", zap.Error(hookErr))
		}

		cancel()
	}()

	// Flow configuration is required
	if len(cfg.Flow) == 0 {
		log.Error("No flow configuration found")
		return fmt.Errorf("flow configuration is required")
	}

	// Note: Git operations now handled via MCP servers

	// Initialize flow-based monitor
	monitorConfig := monitor.Config{
		CheckInterval: cfg.Monitoring.CheckInterval,
		TeamName:      cfg.TeamName,
	}

	log.Info("Creating flow-based monitor", zap.Int("flow_steps", len(cfg.Flow)))
	mon := monitor.New(cfg.Flow, monitorConfig, cfg)

	// Execute on_start hooks
	if hookErr := entrypoint.ExecuteHooks(ctx, cfg.Hooks, "on_start"); hookErr != nil {
		log.Error("Failed to execute on_start hooks", zap.Error(hookErr))
		return fmt.Errorf("failed to execute on_start hooks: %w", hookErr)
	}

	log.Info("Starting flow-based agent monitoring loop",
		zap.Duration("check_interval", cfg.Monitoring.CheckInterval),
		zap.Int("flow_steps", len(cfg.Flow)))

	// Start monitoring with error handling for on_error hooks
	err = mon.Start(ctx)
	if err != nil {
		log.Error("Monitoring loop failed", zap.Error(err))

		// Execute on_error hooks
		if hookErr := entrypoint.ExecuteHooks(ctx, cfg.Hooks, "on_error"); hookErr != nil {
			log.Error("Failed to execute on_error hooks", zap.Error(hookErr))
		}

		return err
	}

	return nil
}

// parseCommaSeparated parses comma-separated arguments from a string
func parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}
	args := strings.Split(value, ",")
	// Trim whitespace from each arg
	for i, arg := range args {
		args[i] = strings.TrimSpace(arg)
	}
	return args
}

// parseKeyValuePairs parses key=value pairs from a string (comma-separated)
func parseKeyValuePairs(value string) map[string]string {
	if value == "" {
		return map[string]string{}
	}

	envMap := make(map[string]string)
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		if kv := strings.SplitN(strings.TrimSpace(pair), "=", 2); len(kv) == 2 {
			envMap[kv[0]] = kv[1]
		}
	}
	return envMap
}
