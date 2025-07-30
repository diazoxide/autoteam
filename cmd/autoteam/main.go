package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"autoteam/internal/config"
	"autoteam/internal/generator"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
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
		Before:  loadGlobalConfig,
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
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func generateCommand(ctx context.Context, cmd *cli.Command) error {
	cfg := getConfigFromContext(ctx)
	if cfg == nil {
		return fmt.Errorf("config not available in context")
	}

	gen := generator.New()
	if err := gen.GenerateCompose(cfg); err != nil {
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	fmt.Println("Generated compose.yaml successfully")
	return nil
}

func upCommand(ctx context.Context, cmd *cli.Command) error {
	if err := generateCommand(ctx, cmd); err != nil {
		return err
	}

	fmt.Println("Starting containers...")
	if err := runDockerCompose(ctx, "up", "-d"); err != nil {
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

func runDockerCompose(ctx context.Context, args ...string) error {
	cfg := getConfigFromContext(ctx)

	// Use the compose.yaml file from .autoteam directory
	composeArgs := []string{"-f", config.ComposeFilePath}

	// If config is available, use custom project name, otherwise use default
	if cfg != nil && cfg.Settings.TeamName != "" {
		composeArgs = append(composeArgs, "-p", cfg.Settings.TeamName)
	}

	composeArgs = append(composeArgs, args...)

	cmd := exec.Command("docker-compose", composeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// loadGlobalConfig loads the config and stores it in the context
func loadGlobalConfig(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	// Skip loading config for init command as it creates the config file
	// Check command line arguments since Before hook runs on root command
	if len(os.Args) > 1 && os.Args[1] == "init" {
		return ctx, nil
	}

	cfg, err := config.LoadConfig("autoteam.yaml")
	if err != nil {
		return ctx, fmt.Errorf("failed to load config: %w", err)
	}

	return context.WithValue(ctx, configContextKey, cfg), nil
}

// getConfigFromContext retrieves the config from context
func getConfigFromContext(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(configContextKey).(*config.Config)
	if !ok {
		return nil
	}
	return cfg
}
