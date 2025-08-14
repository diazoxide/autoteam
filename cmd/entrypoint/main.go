package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/config"
	"autoteam/internal/deps"
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

			// Agent Configuration
			&cli.StringFlag{
				Name:     "agent-name",
				Usage:    "Name of the agent",
				Required: true,
				Sources:  cli.EnvVars("AGENT_NAME"),
			},
			&cli.StringFlag{
				Name:    "agent-type",
				Value:   "claude",
				Usage:   "Type of agent to use",
				Sources: cli.EnvVars("AGENT_TYPE"),
			},
			&cli.StringFlag{
				Name:    "agent-prompt",
				Usage:   "Primary prompt for the agent",
				Sources: cli.EnvVars("AGENT_PROMPT"),
			},

			// Two-Layer Agent Architecture Configuration
			&cli.StringFlag{
				Name:    "collector-agent-type",
				Value:   "qwen",
				Usage:   "Type of agent for task collection (first layer)",
				Sources: cli.EnvVars("COLLECTOR_AGENT_TYPE"),
			},
			&cli.StringFlag{
				Name:    "collector-agent-args",
				Usage:   "Comma-separated arguments for collector agent",
				Sources: cli.EnvVars("COLLECTOR_AGENT_ARGS"),
			},
			&cli.StringFlag{
				Name:    "collector-agent-env",
				Usage:   "Comma-separated key=value environment variables for collector agent",
				Sources: cli.EnvVars("COLLECTOR_AGENT_ENV"),
			},
			&cli.StringFlag{
				Name:    "execution-agent-type",
				Value:   "claude",
				Usage:   "Type of agent for task execution (second layer)",
				Sources: cli.EnvVars("EXECUTION_AGENT_TYPE"),
			},
			&cli.StringFlag{
				Name:    "execution-agent-args",
				Usage:   "Comma-separated arguments for execution agent",
				Sources: cli.EnvVars("EXECUTION_AGENT_ARGS"),
			},
			&cli.StringFlag{
				Name:    "execution-agent-env",
				Usage:   "Comma-separated key=value environment variables for execution agent",
				Sources: cli.EnvVars("EXECUTION_AGENT_ENV"),
			},
			&cli.StringFlag{
				Name:    "collector-agent-prompt",
				Usage:   "Custom prompt for the collector agent (first layer)",
				Sources: cli.EnvVars("COLLECTOR_AGENT_PROMPT"),
			},
			&cli.StringFlag{
				Name:    "execution-agent-prompt",
				Usage:   "Custom prompt for the execution agent (second layer)",
				Sources: cli.EnvVars("EXECUTION_AGENT_PROMPT"),
			},

			&cli.StringFlag{
				Name:    "team-name",
				Value:   "autoteam",
				Usage:   "Team name for directory structure",
				Sources: cli.EnvVars("TEAM_NAME"),
			},

			// Monitoring Configuration
			&cli.IntFlag{
				Name:    "check-interval",
				Value:   60,
				Usage:   "Check interval in seconds",
				Sources: cli.EnvVars("CHECK_INTERVAL"),
			},
			&cli.IntFlag{
				Name:    "max-retries",
				Value:   100,
				Usage:   "Maximum number of retries for operations",
				Sources: cli.EnvVars("MAX_RETRIES"),
			},
			&cli.IntFlag{
				Name:    "max-attempts",
				Value:   3,
				Usage:   "Maximum number of attempts per item before moving to cooldown",
				Sources: cli.EnvVars("MAX_ATTEMPTS"),
			},

			// Dependencies Configuration
			&cli.BoolFlag{
				Name:    "install-deps",
				Usage:   "Install dependencies on startup",
				Sources: cli.EnvVars("INSTALL_DEPS"),
			},

			// Runtime Configuration
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging",
				Sources: cli.EnvVars("DEBUG"),
			},
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

	// Build configuration from CLI flags and environment variables
	cfg, err := buildConfigFromFlags(cmd)
	if err != nil {
		log.Error("Failed to build configuration", zap.Error(err))
		return fmt.Errorf("failed to build configuration: %w", err)
	}

	// Load MCP servers from environment
	mcpServers, mcpErr := entrypoint.LoadMCPServers()
	if mcpErr != nil {
		log.Error("Failed to load MCP servers", zap.Error(mcpErr))
		return fmt.Errorf("failed to load MCP servers: %w", mcpErr)
	}
	cfg.MCPServers = mcpServers

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

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("Received signal, shutting down gracefully", zap.String("signal", sig.String()))
		cancel()
	}()

	// Create layer agent configurations from CLI flags
	// First Layer (Task Collection) - CollectorAgent
	firstLayerConfig := config.AgentConfig{
		Type: cmd.String("collector-agent-type"),
		Args: parseCommaSeparated(cmd.String("collector-agent-args")),
		Env:  parseKeyValuePairs(cmd.String("collector-agent-env")),
	}
	if collectorPrompt := cmd.String("collector-agent-prompt"); collectorPrompt != "" {
		firstLayerConfig.Prompt = &collectorPrompt
	}

	// Second Layer (Task Execution) - Agent
	secondLayerConfig := config.AgentConfig{
		Type: cmd.String("execution-agent-type"),
		Args: parseCommaSeparated(cmd.String("execution-agent-args")),
		Env:  parseKeyValuePairs(cmd.String("execution-agent-env")),
	}
	if executionPrompt := cmd.String("execution-agent-prompt"); executionPrompt != "" {
		secondLayerConfig.Prompt = &executionPrompt
	}

	// Create agent helper for consistent naming
	baseAgent := &config.Agent{Name: cfg.Agent.Name}

	// Create First Layer agent (Task Collection)
	log.Debug("Creating task collection agent", zap.String("agent_type", firstLayerConfig.Type))
	collectorAgentName := baseAgent.GetNormalizedNameWithVariation("collector")
	taskCollectionAgent, err := agent.CreateAgent(firstLayerConfig, collectorAgentName, cfg.MCPServers)
	if err != nil {
		log.Error("Failed to create task collection agent", zap.Error(err))
		return fmt.Errorf("failed to create task collection agent: %w", err)
	}
	log.Info("Task collection agent initialized successfully", zap.String("agent_type", firstLayerConfig.Type))

	// Create Second Layer agent (Task Execution)
	log.Debug("Creating task execution agent", zap.String("agent_type", secondLayerConfig.Type))
	executorAgentName := baseAgent.GetNormalizedNameWithVariation("executor")
	taskExecutionAgent, err := agent.CreateAgent(secondLayerConfig, executorAgentName, cfg.MCPServers)
	if err != nil {
		log.Error("Failed to create task execution agent", zap.Error(err))
		return fmt.Errorf("failed to create task execution agent: %w", err)
	}
	log.Info("Task execution agent initialized successfully", zap.String("agent_type", secondLayerConfig.Type))

	// Install dependencies for both agents if needed
	log.Debug("Installing dependencies for agents")
	installer := deps.NewInstaller(cfg.Dependencies)
	if err := installer.Install(ctx, taskCollectionAgent, taskExecutionAgent); err != nil {
		log.Warn("Failed to install dependencies for agents", zap.Error(err))
	}

	// Note: Git operations now handled via MCP servers

	// Initialize and start monitor with two-layer config
	monitorConfig := monitor.Config{
		CheckInterval: cfg.Monitoring.CheckInterval,
		TeamName:      cfg.TeamName,
	}

	mon := monitor.New(taskCollectionAgent, taskExecutionAgent, monitorConfig, cfg)

	// Set layer configurations for custom prompt support
	mon.SetLayerConfigs(&firstLayerConfig, &secondLayerConfig)

	log.Info("Starting two-layer agent monitoring loop",
		zap.Duration("check_interval", cfg.Monitoring.CheckInterval),
		zap.String("collection_agent", firstLayerConfig.Type),
		zap.String("execution_agent", secondLayerConfig.Type),
	)
	return mon.Start(ctx)
}

// buildConfigFromFlags builds a Config struct from CLI flags
func buildConfigFromFlags(cmd *cli.Command) (*entrypoint.Config, error) {
	cfg := &entrypoint.Config{}

	// Agent configuration
	cfg.Agent.Name = cmd.String("agent-name")
	cfg.Agent.Type = cmd.String("agent-type")
	cfg.Agent.Prompt = cmd.String("agent-prompt")

	// Team configuration
	cfg.TeamName = cmd.String("team-name")

	// Monitoring configuration
	checkInterval := cmd.Int("check-interval")
	cfg.Monitoring.CheckInterval = time.Duration(checkInterval) * time.Second
	cfg.Monitoring.MaxRetries = cmd.Int("max-retries")

	// Dependencies configuration
	cfg.Dependencies.InstallDeps = cmd.Bool("install-deps")

	// Debug configuration
	cfg.Debug = cmd.Bool("debug")

	return cfg, nil
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
