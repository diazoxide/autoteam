package deps

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"autoteam/internal/agent"
	"autoteam/internal/entrypoint"
)

// Installer handles dependency installation
type Installer struct {
	config entrypoint.DependenciesConfig
}

// NewInstaller creates a new dependency installer
func NewInstaller(cfg entrypoint.DependenciesConfig) *Installer {
	return &Installer{
		config: cfg,
	}
}

// Install installs required dependencies
func (i *Installer) Install(ctx context.Context, selectedAgent agent.Agent) error {
	if !i.config.InstallDeps {
		log.Println("Dependency installation disabled, skipping...")
		return nil
	}

	log.Println("Installing dependencies...")

	// Install system dependencies
	if err := i.installSystemDependencies(ctx); err != nil {
		return fmt.Errorf("failed to install system dependencies: %w", err)
	}

	// Install the AI agent if not available
	if !selectedAgent.IsAvailable(ctx) {
		log.Printf("Installing %s agent...", selectedAgent.Type())
		if err := selectedAgent.Install(ctx); err != nil {
			return fmt.Errorf("failed to install %s agent: %w", selectedAgent.Type(), err)
		}
	} else {
		log.Printf("%s agent is already available", selectedAgent.Type())
	}

	log.Println("All dependencies installed successfully")
	return nil
}

// installSystemDependencies installs required system packages
func (i *Installer) installSystemDependencies(ctx context.Context) error {
	log.Println("Installing system dependencies...")

	// Detect package manager and install dependencies
	if i.hasCommand(ctx, "apt") {
		return i.installWithApt(ctx)
	} else if i.hasCommand(ctx, "apk") {
		return i.installWithApk(ctx)
	} else if i.hasCommand(ctx, "yum") {
		return i.installWithYum(ctx)
	}

	log.Println("No supported package manager found, skipping system dependency installation")
	return nil
}

// hasCommand checks if a command is available
func (i *Installer) hasCommand(ctx context.Context, command string) bool {
	cmd := exec.CommandContext(ctx, "which", command)
	return cmd.Run() == nil
}

// installWithApt installs dependencies using apt (Debian/Ubuntu)
func (i *Installer) installWithApt(ctx context.Context) error {
	log.Println("Using apt package manager...")

	// Update package list
	cmd := exec.CommandContext(ctx, "apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to update apt package list: %v", err)
	}

	// Install required packages
	packages := []string{"curl", "git", "nodejs", "npm"}

	args := append([]string{"install", "-y"}, packages...)
	cmd = exec.CommandContext(ctx, "apt", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages with apt: %w", err)
	}

	log.Println("Successfully installed system dependencies with apt")
	return nil
}

// installWithApk installs dependencies using apk (Alpine)
func (i *Installer) installWithApk(ctx context.Context) error {
	log.Println("Using apk package manager...")

	// Update package index
	cmd := exec.CommandContext(ctx, "apk", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to update apk package index: %v", err)
	}

	// Install required packages
	packages := []string{"curl", "git", "nodejs", "npm"}

	args := append([]string{"add", "--no-cache"}, packages...)
	cmd = exec.CommandContext(ctx, "apk", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages with apk: %w", err)
	}

	log.Println("Successfully installed system dependencies with apk")
	return nil
}

// installWithYum installs dependencies using yum (RHEL/CentOS)
func (i *Installer) installWithYum(ctx context.Context) error {
	log.Println("Using yum package manager...")

	// Install required packages
	packages := []string{"curl", "git", "nodejs", "npm"}

	args := append([]string{"install", "-y"}, packages...)
	cmd := exec.CommandContext(ctx, "yum", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages with yum: %w", err)
	}

	log.Println("Successfully installed system dependencies with yum")
	return nil
}

// CheckDependencies checks if all required dependencies are available
func (i *Installer) CheckDependencies(ctx context.Context, selectedAgent agent.Agent) error {
	log.Println("Checking dependencies...")

	// Check git
	if !i.hasCommand(ctx, "git") {
		return fmt.Errorf("git is not available")
	}

	// Check nodejs/npm (required for Claude)
	if selectedAgent.Type() == "claude" {
		if !i.hasCommand(ctx, "node") && !i.hasCommand(ctx, "nodejs") {
			return fmt.Errorf("nodejs is not available")
		}
		if !i.hasCommand(ctx, "npm") {
			return fmt.Errorf("npm is not available")
		}
	}

	// Check if agent is available
	if !selectedAgent.IsAvailable(ctx) {
		return fmt.Errorf("%s agent is not available", selectedAgent.Type())
	}

	log.Println("All dependencies are available")
	return nil
}
