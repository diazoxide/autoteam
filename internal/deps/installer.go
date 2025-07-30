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

	// Install GitHub CLI
	if err := i.installGitHubCLI(ctx); err != nil {
		return fmt.Errorf("failed to install GitHub CLI: %w", err)
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

// installGitHubCLI installs the GitHub CLI (gh) using the official installation script
func (i *Installer) installGitHubCLI(ctx context.Context) error {
	// Check if gh is already available
	if i.hasCommand(ctx, "gh") {
		log.Println("GitHub CLI (gh) is already available")
		return nil
	}

	log.Println("Installing GitHub CLI (gh)...")

	// Try to install via package manager first
	if i.hasCommand(ctx, "apt") {
		if err := i.installGitHubCLIWithApt(ctx); err == nil {
			return nil
		} else {
			log.Printf("Failed to install gh with apt, trying alternative method: %v", err)
		}
	} else if i.hasCommand(ctx, "apk") {
		if err := i.installGitHubCLIWithApk(ctx); err == nil {
			return nil
		} else {
			log.Printf("Failed to install gh with apk, trying alternative method: %v", err)
		}
	} else if i.hasCommand(ctx, "yum") || i.hasCommand(ctx, "dnf") {
		if err := i.installGitHubCLIWithYum(ctx); err == nil {
			return nil
		} else {
			log.Printf("Failed to install gh with yum/dnf, trying alternative method: %v", err)
		}
	}

	// Fallback to official installation script
	return i.installGitHubCLIWithScript(ctx)
}

// installGitHubCLIWithApt installs GitHub CLI using apt
func (i *Installer) installGitHubCLIWithApt(ctx context.Context) error {
	// Add GitHub CLI repository
	commands := [][]string{
		{"curl", "-fsSL", "https://cli.github.com/packages/githubcli-archive-keyring.gpg", "-o", "/usr/share/keyrings/githubcli-archive-keyring.gpg"},
		{"bash", "-c", "echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\" | tee /etc/apt/sources.list.d/github-cli.list"},
		{"apt", "update"},
		{"apt", "install", "-y", "gh"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run command %v: %w", cmdArgs, err)
		}
	}

	log.Println("Successfully installed GitHub CLI with apt")
	return nil
}

// installGitHubCLIWithApk installs GitHub CLI using apk
func (i *Installer) installGitHubCLIWithApk(ctx context.Context) error {
	// GitHub CLI is available in Alpine edge/community repository
	cmd := exec.CommandContext(ctx, "apk", "add", "--no-cache", "github-cli")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install gh with apk: %w", err)
	}

	log.Println("Successfully installed GitHub CLI with apk")
	return nil
}

// installGitHubCLIWithYum installs GitHub CLI using yum/dnf
func (i *Installer) installGitHubCLIWithYum(ctx context.Context) error {
	// Add GitHub CLI repository for RHEL/CentOS
	commands := [][]string{
		{"dnf", "config-manager", "--add-repo", "https://cli.github.com/packages/rpm/gh-cli.repo"},
		{"dnf", "install", "-y", "gh"},
	}

	// Fallback to yum if dnf is not available
	if !i.hasCommand(ctx, "dnf") {
		commands = [][]string{
			{"yum-config-manager", "--add-repo", "https://cli.github.com/packages/rpm/gh-cli.repo"},
			{"yum", "install", "-y", "gh"},
		}
	}

	for _, cmdArgs := range commands {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run command %v: %w", cmdArgs, err)
		}
	}

	log.Println("Successfully installed GitHub CLI with yum/dnf")
	return nil
}

// installGitHubCLIWithScript installs GitHub CLI using the official installation script
func (i *Installer) installGitHubCLIWithScript(ctx context.Context) error {
	log.Println("Installing GitHub CLI using official installation script...")

	// Download and run the official installation script
	cmd := exec.CommandContext(ctx, "bash", "-c", "curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg && echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && apt update && apt install -y gh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Try a simpler approach with direct binary download
		log.Println("Official script failed, trying direct binary download...")
		return i.installGitHubCLIBinary(ctx)
	}

	log.Println("Successfully installed GitHub CLI with official script")
	return nil
}

// installGitHubCLIBinary downloads and installs GitHub CLI binary directly
func (i *Installer) installGitHubCLIBinary(ctx context.Context) error {
	// Determine architecture
	archCmd := exec.CommandContext(ctx, "uname", "-m")
	archOutput, err := archCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to determine architecture: %w", err)
	}

	arch := string(archOutput)
	var downloadArch string
	switch arch {
	case "x86_64\n":
		downloadArch = "amd64"
	case "aarch64\n", "arm64\n":
		downloadArch = "arm64"
	default:
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Get the latest release version and download GitHub CLI binary
	commands := [][]string{
		{"bash", "-c", "curl -s https://api.github.com/repos/cli/cli/releases/latest | grep 'tag_name' | cut -d'\"' -f4 | sed 's/v//' > /tmp/gh_version"},
		{"bash", "-c", fmt.Sprintf("curl -fsSL https://github.com/cli/cli/releases/latest/download/gh_$(cat /tmp/gh_version)_linux_%s.tar.gz -o /tmp/gh.tar.gz", downloadArch)},
		{"tar", "-xzf", "/tmp/gh.tar.gz", "-C", "/tmp"},
		{"bash", "-c", "find /tmp -name 'gh_*_linux_*' -type d | head -1 | xargs -I {} cp {}/bin/gh /usr/local/bin/"},
		{"chmod", "+x", "/usr/local/bin/gh"},
		{"rm", "-rf", "/tmp/gh*"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run command %v: %w", cmdArgs, err)
		}
	}

	log.Println("Successfully installed GitHub CLI binary")
	return nil
}

// CheckDependencies checks if all required dependencies are available
func (i *Installer) CheckDependencies(ctx context.Context, selectedAgent agent.Agent) error {
	log.Println("Checking dependencies...")

	// Check git
	if !i.hasCommand(ctx, "git") {
		return fmt.Errorf("git is not available")
	}

	// Check GitHub CLI
	if !i.hasCommand(ctx, "gh") {
		return fmt.Errorf("GitHub CLI (gh) is not available")
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
