package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/logger"
	"autoteam/internal/runtime"

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
		Name:    "autoteam",
		Usage:   "Universal AI Agent Management System",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
		Before:  setupContextWithLogger,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set log level (debug, info, warn, error)",
				Value:   "warn",
			},
			&cli.StringFlag{
				Name:    "config-file",
				Aliases: []string{"c"},
				Usage:   "Path to configuration file",
				Value:   "autoteam.yaml",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "up",
				Usage:  "Deploy and start all services",
				Action: upCommand,
			},
			{
				Name:   "down",
				Usage:  "Stop all services",
				Action: downCommand,
			},
			{
				Name:   "status",
				Usage:  "Show status of all services",
				Action: statusCommand,
			},
			{
				Name:   "logs",
				Usage:  "Show logs for a service",
				Action: logsCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "service",
						Aliases:  []string{"s"},
						Usage:    "Service name to get logs for",
						Required: true,
					},
					&cli.IntFlag{
						Name:    "lines",
						Aliases: []string{"n"},
						Usage:   "Number of lines to retrieve",
						Value:   100,
					},
				},
			},
			{
				Name:   "init",
				Usage:  "Create sample autoteam.yaml",
				Action: initCommand,
			},
			{
				Name:   "generate",
				Usage:  "Generate configuration files (for compatibility)",
				Action: generateCommand,
			},
			{
				Name:   "workers",
				Usage:  "List all workers and their states",
				Action: workersCommand,
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		// Create emergency logger for fatal errors
		if emergencyLogger, logErr := logger.NewLogger(logger.ErrorLevel); logErr == nil {
			emergencyLogger.Fatal("Application failed to run", zap.Error(err))
		} else {
			os.Exit(1)
		}
	}
}

// Helper function to create runtime instance
func createRuntime(cfg *config.Config) (runtime.Runtime, error) {
	return runtime.NewRuntime(cfg.Deployments)
}

func upCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	log.Debug("Config loaded successfully",
		zap.String("config_file", configFile),
		zap.String("team_name", cfg.GetTeamName()),
		zap.String("runtime", cfg.Deployments.Runtime))

	// Create runtime instance
	rt, err := createRuntime(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	// Initialize runtime
	fmt.Println("Initializing runtime...")
	if err := rt.Initialize(ctx, cfg); err != nil {
		return fmt.Errorf("failed to initialize runtime: %w", err)
	}

	// Deploy all enabled workers
	fmt.Println("Deploying workers...")
	workersWithSettings := cfg.GetEnabledWorkersWithEffectiveSettings()
	for _, workerWithSettings := range workersWithSettings {
		worker := workerWithSettings.Worker
		settings := workerWithSettings.Settings

		log.Debug("Deploying worker", zap.String("worker", worker.Name))
		if err := rt.DeployWorker(ctx, worker, settings, cfg); err != nil {
			return fmt.Errorf("failed to deploy worker %s: %w", worker.Name, err)
		}
		fmt.Printf("Worker %s deployed successfully\n", worker.Name)
	}

	// Deploy control plane if enabled
	if cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		fmt.Println("Deploying control plane...")
		if err := rt.DeployControlPlane(ctx, cfg); err != nil {
			return fmt.Errorf("failed to deploy control plane: %w", err)
		}
		fmt.Println("Control plane deployed successfully")
	}

	// Deploy dashboard if enabled
	if cfg.Dashboard != nil && cfg.Dashboard.Enabled {
		fmt.Println("Deploying dashboard...")
		if err := rt.DeployDashboard(ctx, cfg); err != nil {
			return fmt.Errorf("failed to deploy dashboard: %w", err)
		}
		fmt.Println("Dashboard deployed successfully")
	}

	// Deploy custom services
	if cfg.Services != nil {
		fmt.Println("Deploying custom services...")
		for serviceName, serviceConfig := range cfg.Services {
			log.Debug("Deploying custom service", zap.String("service", serviceName))
			if err := rt.DeployService(ctx, serviceName, serviceConfig, cfg); err != nil {
				return fmt.Errorf("failed to deploy service %s: %w", serviceName, err)
			}
			fmt.Printf("Service %s deployed successfully\n", serviceName)
		}
	}

	fmt.Println("\nAll services deployed successfully!")
	return nil
}

func downCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	log.Debug("Config loaded successfully for down command",
		zap.String("config_file", configFile),
		zap.String("team_name", cfg.GetTeamName()),
		zap.String("runtime", cfg.Deployments.Runtime))

	// Create runtime instance
	rt, err := createRuntime(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	fmt.Println("Stopping all services...")
	if err := rt.StopAll(ctx, cfg); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	fmt.Println("All services stopped successfully")
	return nil
}

func statusCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	// Create runtime instance
	rt, err := createRuntime(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	// Get status of all services
	services, err := rt.GetStatus(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	if len(services) == 0 {
		fmt.Println("No services are currently running.")
		return nil
	}

	// Print status table
	fmt.Printf("%-20s %-10s %-10s %-20s %-15s\n", "SERVICE", "STATUS", "HEALTH", "IMAGE", "PORTS")
	fmt.Println(strings.Repeat("-", 80))

	for _, service := range services {
		ports := strings.Join(service.Ports, ", ")
		if ports == "" {
			ports = "-"
		}
		fmt.Printf("%-20s %-10s %-10s %-20s %-15s\n",
			service.Name,
			service.Status,
			service.Health,
			service.Image,
			ports)
	}

	fmt.Printf("\nTotal services: %d\n", len(services))
	return nil
}

func logsCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	// Create runtime instance
	rt, err := createRuntime(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	serviceName := cmd.String("service")
	lines := cmd.Int("lines")

	// Get logs for the specified service
	logLines, err := rt.GetLogs(ctx, serviceName, lines, cfg)
	if err != nil {
		return fmt.Errorf("failed to get logs for service %s: %w", serviceName, err)
	}

	if len(logLines) == 0 {
		fmt.Printf("No logs available for service: %s\n", serviceName)
		return nil
	}

	// Print logs
	for _, line := range logLines {
		fmt.Println(line)
	}

	return nil
}

func initCommand(ctx context.Context, cmd *cli.Command) error {
	if err := config.CreateSampleConfig("autoteam.yaml"); err != nil {
		return fmt.Errorf("failed to create sample config: %w", err)
	}

	fmt.Println("Created sample autoteam.yaml")
	return nil
}

func generateCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	log.Debug("Config loaded successfully for generate command",
		zap.String("config_file", configFile),
		zap.String("team_name", cfg.GetTeamName()),
		zap.String("runtime", cfg.Deployments.Runtime))

	// Create runtime instance
	rt, err := createRuntime(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	// Initialize runtime (this generates config files)
	fmt.Println("Generating configuration files...")
	if err := rt.Initialize(ctx, cfg); err != nil {
		return fmt.Errorf("failed to initialize runtime and generate files: %w", err)
	}

	fmt.Println("Configuration files generated successfully")
	return nil
}

func workersCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	fmt.Println("Workers configuration:")
	fmt.Println()

	for i, worker := range cfg.Workers {
		status := "enabled"
		if !worker.IsEnabled() {
			status = "disabled"
		}

		fmt.Printf("%d. %s (%s)\n", i+1, worker.Name, status)
		if worker.Prompt != "" {
			// Show first line of prompt
			lines := strings.Split(worker.Prompt, "\n")
			if len(lines) > 0 && lines[0] != "" {
				prompt := lines[0]
				if len(prompt) > 80 {
					prompt = prompt[:77] + "..."
				}
				fmt.Printf("   Prompt: %s\n", prompt)
			}
		}
		fmt.Println()
	}

	// Summary
	enabledCount := 0
	for _, worker := range cfg.Workers {
		if worker.IsEnabled() {
			enabledCount++
		}
	}
	fmt.Printf("Total workers: %d (enabled: %d, disabled: %d)\n",
		len(cfg.Workers), enabledCount, len(cfg.Workers)-enabledCount)

	return nil
}

// setupContextWithLogger sets up logger and loads config into context
func setupContextWithLogger(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	// Setup logger first
	logLevelStr := cmd.String("log-level")
	logLevel, err := logger.ParseLogLevel(logLevelStr)
	if err != nil {
		return ctx, fmt.Errorf("invalid log level: %w", err)
	}

	ctx, err = logger.SetupContext(ctx, logLevel)
	if err != nil {
		return ctx, fmt.Errorf("failed to setup logger: %w", err)
	}

	log := logger.FromContext(ctx)
	log.Info("Starting autoteam",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
		zap.String("log_level", string(logLevel)),
	)

	// Skip loading config for init command as it creates the config file
	// For other commands, let them handle their own config loading
	if len(os.Args) > 1 && os.Args[1] == "init" {
		return ctx, nil
	}

	// For commands that need config, they will load it themselves with proper flag handling
	return ctx, nil
}
