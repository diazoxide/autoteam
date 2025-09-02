package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/generator"
	"autoteam/internal/logger"
	"autoteam/internal/ports"

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

// Context key for storing config
type contextKey string

const configContextKey contextKey = "config"

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
				Name:   "generate",
				Usage:  "Generate compose.yaml from autoteam.yaml",
				Action: generateCommand,
			},
			{
				Name:   "up",
				Usage:  "Generate and start containers",
				Action: upCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "docker-compose-args",
						Aliases: []string{"args"},
						Usage:   "Additional arguments to pass to docker compose command",
						Value:   "",
					},
				},
			},
			{
				Name:   "down",
				Usage:  "Stop containers",
				Action: downCommand,
			},
			{
				Name:   "init",
				Usage:  "Create sample autoteam.yaml",
				Action: initCommand,
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

func generateCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	log.Debug("Generating compose.yaml", zap.String("team_name", cfg.Settings.GetTeamName()))
	gen := generator.New()
	if err := gen.GenerateCompose(cfg); err != nil {
		log.Error("Failed to generate compose.yaml", zap.Error(err))
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	log.Debug("Generated compose.yaml successfully")
	fmt.Println("Generated compose.yaml successfully")
	return nil
}

func upCommand(ctx context.Context, cmd *cli.Command) error {
	log := logger.FromContext(ctx)

	// Load config using the specified config file
	configFile := cmd.String("config-file")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err), zap.String("config_file", configFile))
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	log.Debug("Config loaded successfully",
		zap.String("config_file", configFile),
		zap.String("team_name", cfg.Settings.GetTeamName()),
		zap.Bool("debug_enabled", cfg.Settings.GetDebug()))

	// Find free ports for enabled workers
	enabledWorkersWithSettings := cfg.GetEnabledWorkersWithEffectiveSettings()
	if len(enabledWorkersWithSettings) > 0 {
		fmt.Printf("Finding free ports for %d workers...\n", len(enabledWorkersWithSettings))

		portManager := ports.NewPortManager()
		var serviceNames []string

		// Get service names for enabled workers
		for _, workerWithSettings := range enabledWorkersWithSettings {
			serviceNames = append(serviceNames, workerWithSettings.Worker.GetNormalizedName())
		}

		// Allocate ports for all enabled worker services
		portAllocation, err := portManager.AllocatePortsForServices(serviceNames)
		if err != nil {
			log.Error("Failed to allocate ports", zap.Error(err))
			return fmt.Errorf("failed to allocate ports: %w", err)
		}

		// Display port allocations
		fmt.Println("Port allocations:")
		for serviceName, port := range portAllocation {
			fmt.Printf("  %s: http://localhost:%d\n", serviceName, port)
		}
		fmt.Println()

		// Generate compose with port allocations
		log.Debug("Generating compose.yaml with dynamic ports", zap.String("team_name", cfg.Settings.GetTeamName()))
		gen := generator.New()
		if err := gen.GenerateComposeWithPorts(cfg, portAllocation); err != nil {
			log.Error("Failed to generate compose.yaml", zap.Error(err))
			return fmt.Errorf("failed to generate compose.yaml: %w", err)
		}

		log.Debug("Generated compose.yaml successfully with port mappings")
		fmt.Println("Generated compose.yaml successfully with port mappings")
	} else {
		// No enabled agents, use regular generation
		if err := generateCommand(ctx, cmd); err != nil {
			return err
		}
	}

	fmt.Println("Starting containers...")

	// Start with default args
	args := []string{"up", "-d", "--remove-orphans"}

	// Add additional docker-compose-args if provided
	if dockerComposeArgs := cmd.String("docker-compose-args"); dockerComposeArgs != "" {
		// Split the args string by spaces and append to args
		additionalArgs := strings.Fields(dockerComposeArgs)
		args = append(args, additionalArgs...)
	}

	if err := runDockerComposeWithConfig(ctx, cfg, args...); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	fmt.Println("Containers started successfully")
	return nil
}

func downCommand(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Stopping containers...")
	if err := runDockerCompose(ctx, "down"); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	fmt.Println("Containers stopped successfully")
	return nil
}

func initCommand(ctx context.Context, cmd *cli.Command) error {
	if err := config.CreateSampleConfig("autoteam.yaml"); err != nil {
		return fmt.Errorf("failed to create sample config: %w", err)
	}

	fmt.Println("Created sample autoteam.yaml")
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

func runDockerCompose(ctx context.Context, args ...string) error {
	cfg := getConfigFromContext(ctx)
	return runDockerComposeWithConfig(ctx, cfg, args...)
}

func runDockerComposeWithConfig(ctx context.Context, cfg *config.Config, args ...string) error {
	// Use the compose.yaml file from .autoteam directory
	composeArgs := []string{"-f", config.ComposeFilePath}

	// If config is available, use custom project name, otherwise use default
	if cfg != nil && cfg.Settings.GetTeamName() != "" {
		composeArgs = append(composeArgs, "-p", cfg.Settings.GetTeamName())
	}

	composeArgs = append(composeArgs, args...)

	cmd := exec.Command("docker-compose", composeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

// getConfigFromContext retrieves the config from context
func getConfigFromContext(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(configContextKey).(*config.Config)
	if !ok {
		return nil
	}
	return cfg
}
