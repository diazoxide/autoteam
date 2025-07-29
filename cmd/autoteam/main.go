package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"autoteam/internal/config"
	"autoteam/internal/generator"

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
		Name:    "autoteam",
		Usage:   "Universal AI Agent Management System",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
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
	cfg, err := config.LoadConfig("autoteam.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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
	if err := runDockerCompose("up", "-d"); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	fmt.Println("Containers started successfully")
	return nil
}

func downCommand(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Stopping containers...")
	if err := runDockerCompose("down"); err != nil {
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

func runDockerCompose(args ...string) error {
	// Use the compose.yaml file from .autoteam directory
	composeArgs := []string{"-f", config.ComposeFilePath}
	composeArgs = append(composeArgs, args...)

	cmd := exec.Command("docker-compose", composeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
