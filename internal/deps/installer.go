package deps

import (
	"context"
	"fmt"
	"os/exec"

	"autoteam/internal/agent"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// DependenciesConfig holds configuration for dependency management
type DependenciesConfig struct {
	InstallDeps bool
}

// Installer handles dependency installation
type Installer struct {
	config DependenciesConfig
}

// NewInstaller creates a new dependency installer
func NewInstaller(cfg DependenciesConfig) *Installer {
	return &Installer{
		config: cfg,
	}
}

// Install checks if all required dependencies are available for multiple agents
func (i *Installer) Install(ctx context.Context, agents ...agent.Agent) error {
	lgr := logger.FromContext(ctx)

	if !i.config.InstallDeps {
		lgr.Info("Dependency checking disabled, skipping")
		return nil
	}

	lgr.Info("Checking dependencies for agents", zap.Int("agent_count", len(agents)))

	// Check shared dependencies once
	if err := i.checkSharedDependencies(ctx); err != nil {
		return fmt.Errorf("shared dependencies not available: %w\n\nPlease install missing dependencies manually", err)
	}

	// Check agent-specific dependencies for each agent
	for _, selectedAgent := range agents {
		if err := i.checkAgentSpecificDependencies(ctx, selectedAgent); err != nil {
			return fmt.Errorf("agent %s dependencies not available: %w", selectedAgent.Type(), err)
		}
	}

	lgr.Info("All dependencies are available")
	return nil
}

// hasCommand checks if a command is available
func (i *Installer) hasCommand(ctx context.Context, command string) bool {
	cmd := exec.CommandContext(ctx, "which", command)
	return cmd.Run() == nil
}

// CheckDependencies checks if all required dependencies are available for multiple agents
func (i *Installer) CheckDependencies(ctx context.Context, agents ...agent.Agent) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Checking dependencies for agents", zap.Int("agent_count", len(agents)))

	// Check shared dependencies once
	if err := i.checkSharedDependencies(ctx); err != nil {
		return err
	}

	// Check each agent's specific dependencies
	for _, selectedAgent := range agents {
		if err := i.checkAgentSpecificDependencies(ctx, selectedAgent); err != nil {
			return err
		}
	}

	lgr.Info("All dependencies are available")
	return nil
}

// checkSharedDependencies checks dependencies that are shared across all agents
func (i *Installer) checkSharedDependencies(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// Check git
	if !i.hasCommand(ctx, "git") {
		return fmt.Errorf("git command not found - please install git")
	}
	lgr.Debug("git is available")

	// Check GitHub CLI
	if !i.hasCommand(ctx, "gh") {
		return fmt.Errorf("GitHub CLI (gh) command not found - please install GitHub CLI from https://cli.github.com")
	}
	lgr.Debug("GitHub CLI (gh) is available")

	return nil
}

// checkAgentSpecificDependencies checks dependencies specific to a single agent
func (i *Installer) checkAgentSpecificDependencies(ctx context.Context, selectedAgent agent.Agent) error {
	lgr := logger.FromContext(ctx)

	// Check if agent is available - this will also call agent.CheckAvailability() which now just checks
	if err := selectedAgent.CheckAvailability(ctx); err != nil {
		return err // Agent.CheckAvailability() now returns detailed error messages with installation instructions
	}

	lgr.Debug("Agent is available", zap.String("agent_type", selectedAgent.Type()))
	return nil
}
