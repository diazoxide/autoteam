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
	"autoteam/internal/database"
	"autoteam/internal/logger"
	"autoteam/internal/runtime"

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
			// Configuration
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to main autoteam.yaml configuration file",
				Value:   "autoteam.yaml",
				Sources: cli.EnvVars("AUTOTEAM_CONFIG"),
			},

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

			// Database Configuration
			&cli.StringFlag{
				Name:     "database-dsn",
				Usage:    "Database connection string (required)",
				Sources:  cli.EnvVars("DATABASE_DSN"),
				Required: true,
			},
			&cli.StringFlag{
				Name:    "database-type",
				Usage:   "Database type (sqlite, postgres, mysql)",
				Value:   "sqlite",
				Sources: cli.EnvVars("DATABASE_TYPE"),
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

	// Load control-plane specific config from generated file
	controlPlaneConfigPath := os.Getenv("CONTROL_PLANE_CONFIG")
	if controlPlaneConfigPath == "" {
		controlPlaneConfigPath = "/opt/autoteam/control-plane/config.yaml"
	}

	log.Info("Loading control-plane configuration", zap.String("config_path", controlPlaneConfigPath))

	controlPlaneData, err := os.ReadFile(controlPlaneConfigPath)
	if err != nil {
		log.Error("Failed to load control-plane config", zap.String("config_path", controlPlaneConfigPath), zap.Error(err))
		return fmt.Errorf("failed to load control-plane config from %s: %w", controlPlaneConfigPath, err)
	}

	var controlPlaneConfig config.ControlPlaneConfig
	if err := yaml.Unmarshal(controlPlaneData, &controlPlaneConfig); err != nil {
		log.Error("Failed to parse control-plane config", zap.String("config_path", controlPlaneConfigPath), zap.Error(err))
		return fmt.Errorf("failed to parse control-plane config: %w", err)
	}

	// Check if control plane is enabled
	if !controlPlaneConfig.Enabled {
		log.Error("Control plane is not enabled in configuration")
		return fmt.Errorf("control plane must be enabled in configuration")
	}

	// Load full autoteam configuration to get settings including flow configuration
	configPath := cmd.String("config")
	log.Info("Loading full autoteam configuration", zap.String("config_path", configPath))

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Error("Failed to load autoteam configuration", zap.String("config_path", configPath), zap.Error(err))
		return fmt.Errorf("failed to load autoteam configuration from %s: %w", configPath, err)
	}

	// Get database configuration from CLI flags
	dbDSN := cmd.String("database-dsn")
	dbType := cmd.String("database-type")

	log.Info("Database configuration from CLI flags",
		zap.String("database_type", dbType),
		zap.String("config_path", controlPlaneConfigPath))

	// Add database config to the config
	cfg.Settings.Database = &database.Config{
		Type: database.DatabaseType(dbType),
		DSN:  dbDSN,
	}

	// Initialize database connection
	log.Info("Initializing database connection")
	conn, err := database.NewConnection(*cfg.Settings.Database)
	if err != nil {
		log.Error("Failed to initialize database connection", zap.Error(err))
		return fmt.Errorf("failed to initialize database connection: %w", err)
	}
	defer conn.Close()

	db := conn.GetDB()
	log.Info("Database connection established")

	// Create database-aware worker registry
	registry, err := controlplane.NewWorkerRegistry(db)
	if err != nil {
		log.Error("Failed to create worker registry", zap.Error(err))
		return fmt.Errorf("failed to create worker registry: %w", err)
	}

	log.Info("Worker registry created", zap.Int("registered_workers", registry.GetWorkerCount()))

	// Initialize runtime for worker lifecycle operations
	var rt runtime.Runtime
	// Make a copy of the config to avoid modifying the original
	runtimeConfig := make(map[string]interface{})
	if cfg != nil && cfg.Deployments != nil {
		for k, v := range cfg.Deployments.Config {
			runtimeConfig[k] = v
		}
	}

	// Inject HOST_WORKING_DIR from environment if available
	if hostDir := os.Getenv("HOST_WORKING_DIR"); hostDir != "" {
		runtimeConfig["host_working_dir"] = hostDir
		log.Info("Using HOST_WORKING_DIR from environment", zap.String("host_working_dir", hostDir))
	}

	deploymentConfig := &config.DeploymentConfig{
		Runtime: "docker",
		Config:  runtimeConfig,
	}
	rt, err = runtime.NewRuntime(deploymentConfig)
	if err != nil {
		log.Error("Failed to initialize default runtime", zap.Error(err))
		return fmt.Errorf("failed to initialize default runtime: %w", err)
	}
	log.Info("Default Docker runtime initialized")

	// Deploy database workers if registry has database workers
	if registry.GetWorkerCount() > 0 {
		log.Info("Deploying database workers", zap.Int("worker_count", registry.GetWorkerCount()))
		if err := registry.DeployDatabaseWorkers(ctx, rt, cfg); err != nil {
			log.Error("Failed to deploy database workers", zap.Error(err))
			// Don't fail the entire startup, just log the error
		}
	}

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

	// Create and start HTTP server
	server := controlplane.NewServer(registry, serverConfig, rt, cfg)

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
