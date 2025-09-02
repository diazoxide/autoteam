package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"autoteam/internal/config"
	controlplane "autoteam/internal/control-plane"
	"autoteam/internal/logger"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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
		Name:    "autoteam-control-plane",
		Usage:   "AutoTeam Control Plane - Central orchestrator for managing multiple workers",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
		Action:  runControlPlane,
		Flags: []cli.Flag{
			// Runtime Configuration
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set log level (debug, info, warn, error)",
				Value:   "info",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},

			// HTTP Server Configuration
			&cli.IntFlag{
				Name:    "port",
				Usage:   "HTTP server port (overrides config file)",
				Value:   0, // 0 means use config file value
				Sources: cli.EnvVars("CONTROL_PLANE_PORT"),
			},
			&cli.StringFlag{
				Name:    "api-key",
				Usage:   "HTTP API key for authentication (overrides config file)",
				Sources: cli.EnvVars("CONTROL_PLANE_API_KEY"),
			},

			// Health check interval
			&cli.DurationFlag{
				Name:    "health-check-interval",
				Usage:   "Interval between worker health checks",
				Value:   30 * time.Second,
				Sources: cli.EnvVars("HEALTH_CHECK_INTERVAL"),
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runControlPlane(ctx context.Context, cmd *cli.Command) error {
	// Setup structured logger
	logLevel, err := logger.ParseLogLevel(cmd.String("log-level"))
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	ctx, err = logger.SetupContext(ctx, logLevel)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	log := logger.FromContext(ctx)
	log.Info("Starting AutoTeam Control Plane",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
		zap.String("log_level", string(logLevel)),
	)

	// Load control-plane specific config from environment variable
	controlPlaneConfigPath := os.Getenv("CONTROL_PLANE_CONFIG")
	if controlPlaneConfigPath == "" {
		controlPlaneConfigPath = "/opt/autoteam/control-plane/config.yaml"
	}
	controlPlaneData, err := os.ReadFile(controlPlaneConfigPath)
	if err != nil {
		log.Error("Failed to load control-plane config", zap.String("config_path", controlPlaneConfigPath), zap.Error(err))
		return fmt.Errorf("failed to load control-plane config from %s: %w", controlPlaneConfigPath, err)
	}

	var controlPlaneConfig config.ControlPlaneConfig
	if unmarshalErr := yaml.Unmarshal(controlPlaneData, &controlPlaneConfig); unmarshalErr != nil {
		log.Error("Failed to parse control-plane config", zap.String("config_path", controlPlaneConfigPath), zap.Error(unmarshalErr))
		return fmt.Errorf("failed to parse control-plane config: %w", unmarshalErr)
	}

	// Check if control plane is enabled
	if !controlPlaneConfig.Enabled {
		log.Error("Control plane is not enabled in configuration")
		return fmt.Errorf("control plane must be enabled in configuration")
	}

	log.Info("Control plane configuration loaded",
		zap.String("config_path", controlPlaneConfigPath),
		zap.Strings("workers_apis", controlPlaneConfig.WorkersAPIs),
		zap.Int("configured_port", controlPlaneConfig.Port))

	// Override configuration with CLI flags if provided
	serverConfig := controlplane.ServerConfig{
		Port:   controlPlaneConfig.Port,
		APIKey: controlPlaneConfig.APIKey,
	}

	if cmd.Int("port") != 0 {
		serverConfig.Port = cmd.Int("port")
		log.Info("Overriding port from CLI flag", zap.Int("port", serverConfig.Port))
	}

	if cmd.String("api-key") != "" {
		serverConfig.APIKey = cmd.String("api-key")
		log.Info("Overriding API key from CLI flag")
	}

	// Create worker registry
	registry, err := controlplane.NewWorkerRegistry(&controlPlaneConfig)
	if err != nil {
		log.Error("Failed to create worker registry", zap.Error(err))
		return fmt.Errorf("failed to create worker registry: %w", err)
	}

	log.Info("Worker registry created", zap.Int("registered_workers", registry.GetWorkerCount()))

	// Create and start HTTP server
	server := controlplane.NewServer(registry, serverConfig)

	if err := server.Start(ctx); err != nil {
		log.Error("Failed to start HTTP server", zap.Error(err))
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	log.Info("Control plane HTTP server started",
		zap.String("url", server.GetURL()),
		zap.Int("port", server.Port()))

	// Graceful shutdown for HTTP server
	defer func() {
		if shutdownErr := server.Stop(context.Background()); shutdownErr != nil {
			log.Error("Failed to stop HTTP server", zap.Error(shutdownErr))
		}
	}()

	// Start health check routine
	healthCheckInterval := cmd.Duration("health-check-interval")
	log.Info("Starting health check routine", zap.Duration("interval", healthCheckInterval))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start health check goroutine
	go func() {
		ticker := time.NewTicker(healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("Stopping health check routine")
				return
			case <-ticker.C:
				log.Debug("Performing periodic health checks")
				registry.PerformHealthChecks(ctx)
			}
		}
	}()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Info("Shutting down gracefully", zap.String("signal", sig.String()))
		cancel()
	case <-ctx.Done():
		log.Info("Context canceled, shutting down")
	}

	log.Info("Control plane shutdown completed")
	return nil
}
