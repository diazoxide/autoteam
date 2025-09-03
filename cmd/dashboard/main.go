package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"autoteam/internal/config"
	"autoteam/internal/dashboard"
	"autoteam/internal/logger"

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
		Name:    "autoteam-dashboard",
		Usage:   "AutoTeam Dashboard - Web interface for monitoring workers and control plane",
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
		Action:  runDashboard,
		Flags: []cli.Flag{
			// Runtime Configuration
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set log level (debug, info, warn, error)",
				Value:   "info",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},

			// Dashboard Configuration
			&cli.IntFlag{
				Name:    "port",
				Usage:   "Dashboard server port",
				Value:   8081,
				Sources: cli.EnvVars("DASHBOARD_PORT"),
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "Control plane API URL",
				Value:   "http://localhost:9090",
				Sources: cli.EnvVars("API_URL"),
			},
			&cli.StringFlag{
				Name:    "title",
				Usage:   "Dashboard title",
				Value:   "AutoTeam Dashboard",
				Sources: cli.EnvVars("DASHBOARD_TITLE"),
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runDashboard(ctx context.Context, cmd *cli.Command) error {
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
	log.Info("Starting AutoTeam Dashboard",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
		zap.String("log_level", string(logLevel)),
	)

	// Create dashboard config from CLI flags
	dashboardConfig := &config.DashboardConfig{
		Enabled: true,
		Port:    cmd.Int("port"),
		APIUrl:  cmd.String("api-url"),
		Title:   cmd.String("title"),
	}

	log.Info("Dashboard configuration",
		zap.Int("port", dashboardConfig.Port),
		zap.String("api_url", dashboardConfig.APIUrl),
		zap.String("title", dashboardConfig.Title),
	)

	// Create and start dashboard server
	server := dashboard.NewServer(dashboardConfig)

	log.Info("Dashboard server starting",
		zap.String("url", fmt.Sprintf("http://localhost:%d", dashboardConfig.Port)),
	)

	if err := server.Start(); err != nil {
		log.Error("Failed to start dashboard server", zap.Error(err))
		return fmt.Errorf("failed to start dashboard server: %w", err)
	}

	return nil
}

// Helper functions for environment variable parsing
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}