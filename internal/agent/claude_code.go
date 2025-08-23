package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
	"autoteam/internal/logger"
	"autoteam/internal/server"

	"go.uber.org/zap"
)

// ClaudeCode implements the Agent interface for Claude Code
type ClaudeCode struct {
	config     entrypoint.AgentConfig
	binaryPath string
	mcpServers map[string]config.MCPServer
	args       []string
	env        map[string]string
}

// NewClaudeCode creates a new Claude Code agent instance
func NewClaudeCode(name string, args []string, env map[string]string, mcpServers map[string]config.MCPServer) *ClaudeCode {
	return &ClaudeCode{
		config: entrypoint.AgentConfig{
			Name: name,
		},
		binaryPath: "claude", // Will be found in PATH after installation
		mcpServers: mcpServers,
		args:       args,
		env:        env,
	}
}

// Name returns the agent name
func (c *ClaudeCode) Name() string {
	return c.config.Name
}

// Type returns the agent type
func (c *ClaudeCode) Type() string {
	return "claude"
}

// Run executes Claude with the given prompt and options
func (c *ClaudeCode) Run(ctx context.Context, prompt string, options RunOptions) (*AgentOutput, error) {
	lgr := logger.FromContext(ctx)

	lgr.Debug("Running Claude agent", zap.String("agent", c.config.Name), zap.Int("prompt_length", len(prompt)))

	// Update Claude before running
	if err := c.update(ctx); err != nil {
		lgr.Warn("Failed to update Claude", zap.Error(err))
	}

	// Build the command arguments
	args := c.buildArgs()

	// Add continue flag when requested
	if options.ContinueMode {
		args = append(args, "--continue")
	}

	// Prepare output capture buffers
	var stdout, stderr bytes.Buffer

	// Execute Claude
	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	cmd.Dir = options.WorkingDirectory
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(prompt)

	// Set environment variables
	cmd.Env = os.Environ()

	// Add Claude-specific environment
	cmd.Env = append(cmd.Env, "IS_SANDBOX=1")

	// Add custom environment variables
	for k, v := range c.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	lgr.Debug("Executing Claude command",
		zap.String("binary", c.binaryPath),
		zap.Strings("args", args),
		zap.String("working_dir", options.WorkingDirectory),
		zap.Int("prompt_length", len(prompt)))

	if err := cmd.Run(); err != nil {
		return &AgentOutput{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, fmt.Errorf("claude execution failed: %w", err)
	}

	return &AgentOutput{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}

// IsAvailable checks if Claude is available
func (c *ClaudeCode) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, c.binaryPath, "--version")
	return cmd.Run() == nil
}

// CheckAvailability checks if Claude Code is available, returns error if not found
func (c *ClaudeCode) CheckAvailability(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Checking Claude Code availability")

	if !c.IsAvailable(ctx) {
		return fmt.Errorf("claude command not found - please install Claude Code using: npm install -g @anthropic-ai/claude-code")
	}

	lgr.Debug("Claude Code is available")
	return nil
}

// Version returns the Claude version
func (c *ClaudeCode) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, c.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Claude version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// update updates Claude to the latest version
func (c *ClaudeCode) update(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.binaryPath, "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildArgs builds the command line arguments for Claude
func (c *ClaudeCode) buildArgs() []string {
	args := []string{
		"--dangerously-skip-permissions",
	}

	// Add custom args from agent configuration
	args = append(args, c.args...)

	// Add MCP config file if MCP servers are configured
	if len(c.mcpServers) > 0 {
		mcpConfigPath := c.getMCPConfigPath()
		args = append(args, "--mcp-config", mcpConfigPath)
	}

	return args
}

// Configure configures MCP servers in Claude configuration
func (c *ClaudeCode) Configure(ctx context.Context) error {
	return c.ConfigureForProject(ctx, "")
}

// ConfigureForProject configures MCP servers for a specific agent
func (c *ClaudeCode) ConfigureForProject(ctx context.Context, projectPath string) error {
	lgr := logger.FromContext(ctx)

	if len(c.mcpServers) == 0 {
		lgr.Debug("No MCP servers to configure")
		return nil
	}

	lgr.Debug("Configuring MCP servers", zap.Int("count", len(c.mcpServers)), zap.String("agent", c.config.Name))

	// Create dedicated MCP configuration file for this agent
	if err := c.createMCPConfigFile(ctx); err != nil {
		return fmt.Errorf("failed to create MCP configuration file: %w", err)
	}

	lgr.Debug("MCP servers configured successfully")
	return nil
}

// getMCPConfigPath returns the path to the MCP configuration file for this agent
func (c *ClaudeCode) getMCPConfigPath() string {
	// Use the agent name as passed from the factory (already normalized with variations)
	// Don't re-normalize as it would convert senior_developer/executor back to senior_developer_executor
	return fmt.Sprintf("/opt/autoteam/agents/%s/.mcp.json", c.config.Name)
}

// createMCPConfigFile creates the MCP configuration file for this agent
func (c *ClaudeCode) createMCPConfigFile(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	mcpConfigPath := c.getMCPConfigPath()
	lgr.Debug("Creating MCP configuration file", zap.String("path", mcpConfigPath))

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

	lgr.Debug("MCP configuration file created",
		zap.String("path", mcpConfigPath),
		zap.Int("mcp_servers", len(c.mcpServers)))

	return nil
}

// SetMCPServers sets the MCP servers for this agent
func (c *ClaudeCode) SetMCPServers(mcpServers map[string]config.MCPServer) {
	c.mcpServers = mcpServers
}

// CreateHTTPServer creates an HTTP API server for this agent
func (c *ClaudeCode) CreateHTTPServer(workingDir string, port int, apiKey string) HTTPServer {
	config := server.Config{
		Port:       port,
		APIKey:     apiKey,
		WorkingDir: workingDir,
	}
	return server.NewServer(c, config)
}
