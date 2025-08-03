package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// ClaudeAgent implements the Agent interface for Claude Code
type ClaudeAgent struct {
	config     entrypoint.AgentConfig
	binaryPath string
	mcpServers map[string]config.MCPServer
}

// NewClaudeAgent creates a new Claude agent instance
func NewClaudeAgent(cfg entrypoint.AgentConfig) *ClaudeAgent {
	return &ClaudeAgent{
		config:     cfg,
		binaryPath: "claude", // Will be found in PATH after installation
		mcpServers: make(map[string]config.MCPServer),
	}
}

// NewClaudeAgentWithMCP creates a new Claude agent instance with MCP servers
func NewClaudeAgentWithMCP(cfg entrypoint.AgentConfig, mcpServers map[string]config.MCPServer) *ClaudeAgent {
	return &ClaudeAgent{
		config:     cfg,
		binaryPath: "claude", // Will be found in PATH after installation
		mcpServers: mcpServers,
	}
}

// Name returns the agent name
func (c *ClaudeAgent) Name() string {
	return c.config.Name
}

// Type returns the agent type
func (c *ClaudeAgent) Type() string {
	return "claude"
}

// Run executes Claude with the given prompt and options
func (c *ClaudeAgent) Run(ctx context.Context, prompt string, options RunOptions) error {
	lgr := logger.FromContext(ctx)
	if options.DryRun {
		lgr.Info("DRY RUN: Would execute Claude with prompt", zap.Int("prompt_length", len(prompt)))
		return nil
	}

	// Update Claude before running
	if err := c.update(ctx); err != nil {
		lgr.Warn("Failed to update Claude", zap.Error(err))
	}

	// Build the command arguments
	args := c.buildArgs(options)

	var lastErr error
	maxRetries := options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lgr.Info("Claude execution attempt", zap.Int("attempt", attempt), zap.Int("max_retries", maxRetries))

		// Add continue flag when requested or for retry attempts
		currentArgs := args
		if options.ContinueMode || attempt > 1 {
			currentArgs = append(args, "--continue")
		}

		// Execute Claude
		cmd := exec.CommandContext(ctx, c.binaryPath, currentArgs...)
		cmd.Dir = options.WorkingDirectory
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = strings.NewReader(prompt)

		// Set environment variables
		cmd.Env = os.Environ()

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("claude execution failed (attempt %d): %w", attempt, err)
			lgr.Warn("Attempt failed", zap.Int("attempt", attempt), zap.Error(err))

			// Don't retry on context cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Wait before retry (exponential backoff)
			if attempt < maxRetries {
				backoff := time.Duration(attempt) * time.Second
				lgr.Info("Retrying", zap.Duration("backoff", backoff))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
		} else {
			lgr.Info("Claude execution succeeded", zap.Int("attempt", attempt))
			return nil
		}
	}

	return fmt.Errorf("all %d attempts failed, last error: %w", maxRetries, lastErr)
}

// IsAvailable checks if Claude is available
func (c *ClaudeAgent) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, c.binaryPath, "--version")
	return cmd.Run() == nil
}

// Install installs Claude Code via npm
func (c *ClaudeAgent) Install(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Installing Claude Code")

	// Check if npm is available
	if err := exec.CommandContext(ctx, "npm", "--version").Run(); err != nil {
		return fmt.Errorf("npm is not available, cannot install Claude Code: %w", err)
	}

	// Install Claude Code globally
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", "@anthropic-ai/claude-code", "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Claude Code: %w", err)
	}

	lgr.Info("Claude Code installed successfully")
	return nil
}

// Version returns the Claude version
func (c *ClaudeAgent) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, c.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Claude version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// update updates Claude to the latest version
func (c *ClaudeAgent) update(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.binaryPath, "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildArgs builds the command line arguments for Claude
func (c *ClaudeAgent) buildArgs(options RunOptions) []string {
	args := []string{
		"--dangerously-skip-permissions",
	}

	if options.Verbose {
		args = append(args, "--verbose")
	}

	if options.OutputFormat != "" {
		args = append(args, "--output-format", options.OutputFormat)
	} else {
		// Default to stream-json for better parsing
		args = append(args, "--output-format", "stream-json")
	}

	// Add MCP config file if MCP servers are configured
	if len(c.mcpServers) > 0 {
		mcpConfigPath := c.getMCPConfigPath()
		args = append(args, "--mcp-config", mcpConfigPath)
	}

	// Always add --print to display output
	args = append(args, "--print")

	return args
}

// Configure configures MCP servers in Claude configuration
func (c *ClaudeAgent) Configure(ctx context.Context) error {
	return c.ConfigureForProject(ctx, "")
}

// ConfigureForProject configures MCP servers for a specific agent
func (c *ClaudeAgent) ConfigureForProject(ctx context.Context, projectPath string) error {
	lgr := logger.FromContext(ctx)

	if len(c.mcpServers) == 0 {
		lgr.Debug("No MCP servers to configure")
		return nil
	}

	lgr.Info("Configuring MCP servers", zap.Int("count", len(c.mcpServers)), zap.String("agent", c.config.Name))

	// Create dedicated MCP configuration file for this agent
	if err := c.createMCPConfigFile(ctx); err != nil {
		return fmt.Errorf("failed to create MCP configuration file: %w", err)
	}

	lgr.Info("MCP servers configured successfully")
	return nil
}

// updateClaudeConfig updates the ~/.claude.json file with MCP server configurations (legacy method)
func (c *ClaudeAgent) updateClaudeConfig(ctx context.Context) error {
	return c.updateClaudeConfigForProject(ctx, "")
}

// updateClaudeConfigForProject updates the ~/.claude.json file with MCP server configurations for a specific project
func (c *ClaudeAgent) updateClaudeConfigForProject(ctx context.Context, projectPath string) error {
	lgr := logger.FromContext(ctx)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeConfigPath := filepath.Join(homeDir, ".claude.json")
	lgr.Info("Updating Claude configuration", zap.String("path", claudeConfigPath))

	// Read existing configuration
	var claudeConfig map[string]interface{}
	if data, err := os.ReadFile(claudeConfigPath); err == nil {
		if err := json.Unmarshal(data, &claudeConfig); err != nil {
			lgr.Warn("Failed to parse existing Claude config, creating new one", zap.Error(err))
			claudeConfig = make(map[string]interface{})
		}
	} else {
		lgr.Info("Creating new Claude configuration file")
		claudeConfig = make(map[string]interface{})
	}

	// Determine project path - use provided path or fallback to current directory
	var targetProjectPath string
	if projectPath != "" {
		targetProjectPath = projectPath
	} else {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		targetProjectPath = currentDir
	}

	lgr.Info("Configuring MCP servers for project", zap.String("project_path", targetProjectPath))

	// Convert MCP servers to Claude format
	mcpServersConfig := make(map[string]interface{})
	for name, server := range c.mcpServers {
		serverConfig := map[string]interface{}{
			"command": server.Command,
		}

		if len(server.Args) > 0 {
			serverConfig["args"] = server.Args
		}

		if len(server.Env) > 0 {
			serverConfig["env"] = server.Env
		}

		mcpServersConfig[name] = serverConfig
	}

	// Ensure projects object exists
	if claudeConfig["projects"] == nil {
		claudeConfig["projects"] = make(map[string]interface{})
	}

	projects, ok := claudeConfig["projects"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid format: projects is not a map")
	}

	// Ensure target project exists
	if projects[targetProjectPath] == nil {
		projects[targetProjectPath] = make(map[string]interface{})
	}

	project, ok := projects[targetProjectPath].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid format: project entry is not a map")
	}

	// Set MCP servers for the target project
	project["mcpServers"] = mcpServersConfig

	// Write updated configuration
	data, err := json.MarshalIndent(claudeConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Claude config: %w", err)
	}

	if err := os.WriteFile(claudeConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write Claude config: %w", err)
	}

	lgr.Info("Claude configuration updated successfully", zap.Int("mcp_servers", len(c.mcpServers)))
	return nil
}

// getMCPConfigPath returns the path to the MCP configuration file for this agent
func (c *ClaudeAgent) getMCPConfigPath() string {
	// Use config.Agent's GetNormalizedName() method for consistent normalization
	configAgent := &config.Agent{Name: c.config.Name}
	normalizedAgentName := configAgent.GetNormalizedName()
	return fmt.Sprintf("/opt/autoteam/agents/%s/mcp.json", normalizedAgentName)
}

// createMCPConfigFile creates the MCP configuration file for this agent
func (c *ClaudeAgent) createMCPConfigFile(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	mcpConfigPath := c.getMCPConfigPath()
	lgr.Info("Creating MCP configuration file", zap.String("path", mcpConfigPath))

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(mcpConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create MCP config directory: %w", err)
	}

	// Convert MCP servers to Claude format with mcpServers wrapper
	mcpServersMap := make(map[string]interface{})
	for name, server := range c.mcpServers {
		serverConfig := map[string]interface{}{
			"command": server.Command,
		}

		if len(server.Args) > 0 {
			serverConfig["args"] = server.Args
		}

		if len(server.Env) > 0 {
			serverConfig["env"] = server.Env
		}

		mcpServersMap[name] = serverConfig
	}

	// Wrap in mcpServers object as required by Claude format
	mcpConfig := map[string]interface{}{
		"mcpServers": mcpServersMap,
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	// Write the MCP configuration file
	if err := os.WriteFile(mcpConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write MCP config file: %w", err)
	}

	lgr.Info("MCP configuration file created successfully",
		zap.String("path", mcpConfigPath),
		zap.Int("mcp_servers", len(c.mcpServers)))

	return nil
}

// SetMCPServers sets the MCP servers for this agent
func (c *ClaudeAgent) SetMCPServers(mcpServers map[string]config.MCPServer) {
	c.mcpServers = mcpServers
}
