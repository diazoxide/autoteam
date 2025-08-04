package deps

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"autoteam/internal/agent"
	"autoteam/internal/entrypoint"
	"autoteam/internal/logger"

	"go.uber.org/zap"
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
	lgr := logger.FromContext(ctx)

	if !i.config.InstallDeps {
		lgr.Info("Dependency installation disabled, skipping")
		return nil
	}

	lgr.Info("Installing dependencies")

	// Install system dependencies (includes GitHub CLI)
	if err := i.installSystemDependencies(ctx); err != nil {
		return fmt.Errorf("failed to install system dependencies: %w", err)
	}

	// Fallback GitHub CLI installation if not installed via package manager
	if err := i.installGitHubCLI(ctx); err != nil {
		return fmt.Errorf("failed to install GitHub CLI: %w", err)
	}

	// Install GitHub MCP Server
	if err := i.installGitHubMCPServer(ctx); err != nil {
		return fmt.Errorf("failed to install GitHub MCP Server: %w", err)
	}

	// Install the AI agent if not available
	if !selectedAgent.IsAvailable(ctx) {
		lgr.Info("Installing agent", zap.String("agent_type", selectedAgent.Type()))
		if err := selectedAgent.Install(ctx); err != nil {
			return fmt.Errorf("failed to install %s agent: %w", selectedAgent.Type(), err)
		}
	} else {
		lgr.Info("Agent is already available", zap.String("agent_type", selectedAgent.Type()))
	}

	lgr.Info("All dependencies installed successfully")
	return nil
}

// installSystemDependencies installs required system packages
func (i *Installer) installSystemDependencies(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Checking system dependencies")

	// Check which packages need to be installed
	requiredPackages := []string{"curl", "git", "nodejs", "npm", "gh"}
	missingPackages := []string{}

	for _, pkg := range requiredPackages {
		if !i.isPackageInstalled(ctx, pkg) {
			missingPackages = append(missingPackages, pkg)
		} else {
			lgr.Info("Package already installed", zap.String("package", pkg))
		}
	}

	if len(missingPackages) == 0 {
		lgr.Info("All system dependencies are already installed")
		return nil
	}

	lgr.Info("Installing missing system dependencies", zap.Strings("packages", missingPackages))

	// Detect package manager and install missing dependencies
	if i.hasCommand(ctx, "apt") {
		return i.installWithApt(ctx, missingPackages)
	} else if i.hasCommand(ctx, "apk") {
		return i.installWithApk(ctx, missingPackages)
	} else if i.hasCommand(ctx, "yum") {
		return i.installWithYum(ctx, missingPackages)
	}

	lgr.Warn("No supported package manager found, skipping system dependency installation")
	return nil
}

// hasCommand checks if a command is available
func (i *Installer) hasCommand(ctx context.Context, command string) bool {
	cmd := exec.CommandContext(ctx, "which", command)
	return cmd.Run() == nil
}

// isPackageInstalled checks if a package/command is installed
func (i *Installer) isPackageInstalled(ctx context.Context, pkg string) bool {
	// Special handling for nodejs (can be 'node' or 'nodejs')
	if pkg == "nodejs" {
		return i.hasCommand(ctx, "node") || i.hasCommand(ctx, "nodejs")
	}
	return i.hasCommand(ctx, pkg)
}

// installWithApt installs dependencies using apt (Debian/Ubuntu)
func (i *Installer) installWithApt(ctx context.Context, missingPackages []string) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Using apt package manager")

	// Update package list
	cmd := exec.CommandContext(ctx, "apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		lgr.Warn("Failed to update apt package list", zap.Error(err))
	}

	// Filter packages for apt installation (gh handled separately)
	aptPackages := []string{}
	needsGH := false

	for _, pkg := range missingPackages {
		if pkg == "gh" {
			needsGH = true
		} else {
			aptPackages = append(aptPackages, pkg)
		}
	}

	// Install regular packages
	if len(aptPackages) > 0 {
		args := append([]string{"install", "-y"}, aptPackages...)
		cmd = exec.CommandContext(ctx, "apt", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install packages with apt: %w", err)
		}
		lgr.Info("Successfully installed system dependencies with apt", zap.Strings("packages", aptPackages))
	}

	// Install GitHub CLI if needed
	if needsGH {
		if err := i.installGitHubCLIWithApt(ctx); err != nil {
			lgr.Warn("Failed to install gh with apt, will try alternative method later", zap.Error(err))
		} else {
			lgr.Info("Successfully installed GitHub CLI with apt")
		}
	}

	return nil
}

// installWithApk installs dependencies using apk (Alpine)
func (i *Installer) installWithApk(ctx context.Context, missingPackages []string) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Using apk package manager")

	// Update package index
	cmd := exec.CommandContext(ctx, "apk", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		lgr.Warn("Failed to update apk package index", zap.Error(err))
	}

	// Filter packages for apk installation (gh handled separately)
	apkPackages := []string{}
	needsGH := false

	for _, pkg := range missingPackages {
		if pkg == "gh" {
			needsGH = true
		} else {
			apkPackages = append(apkPackages, pkg)
		}
	}

	// Install regular packages
	if len(apkPackages) > 0 {
		args := append([]string{"add", "--no-cache"}, apkPackages...)
		cmd = exec.CommandContext(ctx, "apk", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install packages with apk: %w", err)
		}
		lgr.Info("Successfully installed system dependencies with apk", zap.Strings("packages", apkPackages))
	}

	// Install GitHub CLI if needed
	if needsGH {
		if err := i.installGitHubCLIWithApk(ctx); err != nil {
			lgr.Warn("Failed to install gh with apk, will try alternative method later", zap.Error(err))
		} else {
			lgr.Info("Successfully installed GitHub CLI with apk")
		}
	}

	return nil
}

// installWithYum installs dependencies using yum (RHEL/CentOS)
func (i *Installer) installWithYum(ctx context.Context, missingPackages []string) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Using yum package manager")

	// Filter packages for yum installation (gh handled separately)
	yumPackages := []string{}
	needsGH := false

	for _, pkg := range missingPackages {
		if pkg == "gh" {
			needsGH = true
		} else {
			yumPackages = append(yumPackages, pkg)
		}
	}

	// Install regular packages
	if len(yumPackages) > 0 {
		args := append([]string{"install", "-y"}, yumPackages...)
		cmd := exec.CommandContext(ctx, "yum", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install packages with yum: %w", err)
		}
		lgr.Info("Successfully installed system dependencies with yum", zap.Strings("packages", yumPackages))
	}

	// Install GitHub CLI if needed
	if needsGH {
		if err := i.installGitHubCLIWithYum(ctx); err != nil {
			lgr.Warn("Failed to install gh with yum, will try alternative method later", zap.Error(err))
		} else {
			lgr.Info("Successfully installed GitHub CLI with yum")
		}
	}

	return nil
}

// installGitHubCLI installs the GitHub CLI (gh) using the official installation script
func (i *Installer) installGitHubCLI(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// Check if gh is already available
	if i.hasCommand(ctx, "gh") {
		lgr.Info("GitHub CLI (gh) is already available")
		return nil
	}

	lgr.Info("Installing GitHub CLI (gh)")

	// Try to install via package manager first
	if i.hasCommand(ctx, "apt") {
		if err := i.installGitHubCLIWithApt(ctx); err == nil {
			return nil
		} else {
			lgr.Warn("Failed to install gh with apt, trying alternative method", zap.Error(err))
		}
	} else if i.hasCommand(ctx, "apk") {
		if err := i.installGitHubCLIWithApk(ctx); err == nil {
			return nil
		} else {
			lgr.Warn("Failed to install gh with apk, trying alternative method", zap.Error(err))
		}
	} else if i.hasCommand(ctx, "yum") || i.hasCommand(ctx, "dnf") {
		if err := i.installGitHubCLIWithYum(ctx); err == nil {
			return nil
		} else {
			lgr.Warn("Failed to install gh with yum/dnf, trying alternative method", zap.Error(err))
		}
	}

	// Fallback to official installation script
	return i.installGitHubCLIWithScript(ctx)
}

// installGitHubCLIWithApt installs GitHub CLI using apt
func (i *Installer) installGitHubCLIWithApt(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

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

	lgr.Info("Successfully installed GitHub CLI with apt")
	return nil
}

// installGitHubCLIWithApk installs GitHub CLI using apk
func (i *Installer) installGitHubCLIWithApk(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// GitHub CLI is available in Alpine edge/community repository
	cmd := exec.CommandContext(ctx, "apk", "add", "--no-cache", "github-cli")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install gh with apk: %w", err)
	}

	lgr.Info("Successfully installed GitHub CLI with apk")
	return nil
}

// installGitHubCLIWithYum installs GitHub CLI using yum/dnf
func (i *Installer) installGitHubCLIWithYum(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

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

	lgr.Info("Successfully installed GitHub CLI with yum/dnf")
	return nil
}

// installGitHubCLIWithScript installs GitHub CLI using the official installation script
func (i *Installer) installGitHubCLIWithScript(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Installing GitHub CLI using official installation script")

	// Download and run the official installation script
	cmd := exec.CommandContext(ctx, "bash", "-c", "curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg && echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && apt update && apt install -y gh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Try a simpler approach with direct binary download
		lgr.Warn("Official script failed, trying direct binary download", zap.Error(err))
		return i.installGitHubCLIBinary(ctx)
	}

	lgr.Info("Successfully installed GitHub CLI with official script")
	return nil
}

// installGitHubCLIBinary downloads and installs GitHub CLI binary directly
func (i *Installer) installGitHubCLIBinary(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

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

	lgr.Info("Successfully installed GitHub CLI binary")
	return nil
}

// installGitHubMCPServer downloads and installs GitHub MCP Server binary
func (i *Installer) installGitHubMCPServer(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// Check if GitHub MCP Server is already installed
	mcpServerPath := "/opt/autoteam/bin/github-mcp-server"
	if _, err := os.Stat(mcpServerPath); err == nil {
		lgr.Info("GitHub MCP Server is already installed", zap.String("path", mcpServerPath))
		return nil
	}

	lgr.Info("Installing GitHub MCP Server")

	// Create unified bin directory if it doesn't exist
	if err := os.MkdirAll("/opt/autoteam/bin", 0755); err != nil {
		return fmt.Errorf("failed to create unified bin directory: %w", err)
	}

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
		downloadArch = "x86_64"
	case "aarch64\n", "arm64\n":
		downloadArch = "arm64"
	case "i386\n", "i686\n":
		downloadArch = "i386"
	default:
		return fmt.Errorf("unsupported architecture for GitHub MCP Server: %s", arch)
	}

	// Download and install GitHub MCP Server binary
	version := "v0.10.0"
	downloadURL := fmt.Sprintf("https://github.com/github/github-mcp-server/releases/download/%s/github-mcp-server_Linux_%s.tar.gz", version, downloadArch)

	commands := [][]string{
		{"curl", "-fsSL", downloadURL, "-o", "/tmp/github-mcp-server.tar.gz"},
		{"tar", "-xzf", "/tmp/github-mcp-server.tar.gz", "-C", "/tmp"},
		{"bash", "-c", "find /tmp -name 'github-mcp-server' -type f | head -1 | xargs -I {} cp {} /opt/autoteam/bin/github-mcp-server"},
		{"chmod", "+x", "/opt/autoteam/bin/github-mcp-server"},
		{"rm", "-rf", "/tmp/github-mcp-server*"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run command %v: %w", cmdArgs, err)
		}
	}

	lgr.Info("Successfully installed GitHub MCP Server", zap.String("path", mcpServerPath))
	return nil
}

// CheckDependencies checks if all required dependencies are available
func (i *Installer) CheckDependencies(ctx context.Context, selectedAgent agent.Agent) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Checking dependencies")

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

	lgr.Info("All dependencies are available")
	return nil
}
