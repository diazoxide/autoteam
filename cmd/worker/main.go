package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"autoteam/internal/logger"
	"autoteam/internal/monitor"
	"autoteam/internal/server"
	"autoteam/internal/worker"

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
		Name:    "autoteam-worker",
		Usage:   "AutoTeam Worker - AI agent worker execution via MCP servers",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
		Action:  runWorker,
		Flags: []cli.Flag{
			// Primary configuration file
			&cli.StringFlag{
				Name:     "config-file",
				Aliases:  []string{"c"},
				Usage:    "Path to worker configuration file (YAML)",
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

			// HTTP Server Configuration
			&cli.IntFlag{
				Name:    "http-port",
				Usage:   "HTTP server port (0 for dynamic port discovery)",
				Value:   8080,
				Sources: cli.EnvVars("HTTP_PORT"),
			},
			&cli.StringFlag{
				Name:    "http-api-key",
				Usage:   "HTTP API key for authentication (optional)",
				Sources: cli.EnvVars("HTTP_API_KEY"),
			},
			&cli.BoolFlag{
				Name:    "disable-http",
				Usage:   "Disable HTTP server",
				Sources: cli.EnvVars("DISABLE_HTTP"),
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runWorker(ctx context.Context, cmd *cli.Command) error {
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
	log.Info("Starting AutoTeam Worker",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
		zap.String("log_level", string(logLevel)),
	)

	// Load worker configuration from file
	configPath := cmd.String("config-file")
	workerConfig, err := worker.LoadWorkerFromFile(configPath)
	if err != nil {
		log.Error("Failed to load worker configuration from file", zap.String("config_path", configPath), zap.Error(err))
		return fmt.Errorf("failed to load worker configuration from file %s: %w", configPath, err)
	}

	// Get worker effective settings (without global settings - worker is standalone)
	effectiveSettings := workerConfig.GetEffectiveSettings(worker.WorkerSettings{})

	// Check if debug is enabled in config and update log level if needed
	if effectiveSettings.GetDebug() && logLevel != logger.DebugLevel {
		logLevel = logger.DebugLevel
		ctx, err = logger.SetupContext(ctx, logLevel)
		if err != nil {
			return fmt.Errorf("failed to update logger to debug level: %w", err)
		}
		log = logger.FromContext(ctx)
		log.Debug("Updated log level to debug based on worker configuration")
	}

	log.Debug("Worker configuration loaded successfully",
		zap.String("worker_name", workerConfig.Name),
		zap.String("team_name", effectiveSettings.GetTeamName()),
		zap.Bool("debug_enabled", effectiveSettings.GetDebug()),
	)

	// Create Worker instance for HTTP server
	workerImpl := worker.NewWorker(workerConfig, effectiveSettings)

	// Start HTTP server if not disabled
	var httpServer *server.Server
	if !cmd.Bool("disable-http") {
		// Use port from worker config, fallback to CLI flag
		httpPort := effectiveSettings.GetHTTPPort()
		if httpPort == 0 {
			httpPort = cmd.Int("http-port")
		}

		serverConfig := server.Config{
			Port:       httpPort,
			APIKey:     cmd.String("http-api-key"),
			WorkingDir: workerImpl.GetWorkingDir(),
		}

		httpServer = server.NewServer(workerImpl, serverConfig)

		if err := httpServer.Start(ctx); err != nil {
			log.Error("Failed to start HTTP server", zap.Error(err))
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}

		log.Info("HTTP API server started",
			zap.String("url", httpServer.GetURL()),
			zap.Int("port", httpServer.Port()))

		// Graceful shutdown for HTTP server
		defer func() {
			if shutdownErr := httpServer.Stop(context.Background()); shutdownErr != nil {
				log.Error("Failed to stop HTTP server", zap.Error(shutdownErr))
			}
		}()
	}

	// Execute on_init hooks from worker settings
	if hookErr := worker.ExecuteHooks(ctx, effectiveSettings.Hooks, "on_init"); hookErr != nil {
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
		log.Info("Shutting down gracefully", zap.String("signal", sig.String()))

		// Execute on_stop hooks from worker settings
		if hookErr := worker.ExecuteHooks(ctx, effectiveSettings.Hooks, "on_stop"); hookErr != nil {
			log.Error("Failed to execute on_stop hooks", zap.Error(hookErr))
		}

		cancel()
	}()

	// Flow configuration is required
	if len(effectiveSettings.Flow) == 0 {
		log.Error("No flow configuration found")
		return fmt.Errorf("flow configuration is required")
	}

	// Note: Git operations now handled via MCP servers

	// Initialize flow-based monitor with worker and effective settings
	monitorConfig := monitor.Config{
		SleepDuration: time.Duration(effectiveSettings.GetSleepDuration()) * time.Second,
		TeamName:      effectiveSettings.GetTeamName(),
	}

	log.Info("Creating flow-based monitor", zap.Int("flow_steps", len(effectiveSettings.Flow)))
	mon := monitor.New(workerConfig, effectiveSettings, monitorConfig)

	// Pass the HTTP server to monitor for management
	if httpServer != nil {
		mon.SetHTTPServer(httpServer)
	}

	// Execute on_start hooks from worker settings
	if hookErr := worker.ExecuteHooks(ctx, effectiveSettings.Hooks, "on_start"); hookErr != nil {
		log.Error("Failed to execute on_start hooks", zap.Error(hookErr))
		return fmt.Errorf("failed to execute on_start hooks: %w", hookErr)
	}

	log.Info("Starting flow-based agent monitoring loop",
		zap.Duration("sleep_duration", time.Duration(effectiveSettings.GetSleepDuration())*time.Second),
		zap.Int("flow_steps", len(effectiveSettings.Flow)))

	// Start monitoring with error handling for on_error hooks
	err = mon.Start(ctx)
	if err != nil {
		log.Error("Monitoring loop failed", zap.Error(err))

		// Execute on_error hooks from worker settings
		if hookErr := worker.ExecuteHooks(ctx, effectiveSettings.Hooks, "on_error"); hookErr != nil {
			log.Error("Failed to execute on_error hooks", zap.Error(hookErr))
		}

		return err
	}

	return nil
}
