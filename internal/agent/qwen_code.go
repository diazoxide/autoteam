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
	"time"

	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
	"autoteam/internal/logger"
	"autoteam/internal/server"

	"go.uber.org/zap"
)

// QwenCode implements the Agent interface for Qwen Code
type QwenCode struct {
	config     entrypoint.AgentConfig
	binaryPath string
	mcpServers map[string]config.MCPServer
	agentArgs  []string
	agentEnv   map[string]string
}

// NewQwenCode creates a new Qwen agent instance
func NewQwenCode(cfg entrypoint.AgentConfig) *QwenCode {
	return &QwenCode{
		config:     cfg,
		binaryPath: "qwen", // Will be found in PATH after npm installation
		mcpServers: make(map[string]config.MCPServer),
		agentArgs:  []string{},
		agentEnv:   make(map[string]string),
	}
}

// NewQwenCodeWithMCP creates a new Qwen agent instance with MCP servers
func NewQwenCodeWithMCP(cfg entrypoint.AgentConfig, mcpServers map[string]config.MCPServer) *QwenCode {
	return &QwenCode{
		config:     cfg,
		binaryPath: "qwen", // Will be found in PATH after npm installation
		mcpServers: mcpServers,
		agentArgs:  []string{},
		agentEnv:   make(map[string]string),
	}
}

// NewQwenCodeWithConfig creates a new Qwen agent instance with full configuration
func NewQwenCodeWithConfig(agentConfig config.AgentConfig, name string, mcpServers map[string]config.MCPServer) *QwenCode {
	cfg := entrypoint.AgentConfig{Name: name}
	agent := &QwenCode{
		config:     cfg,
		binaryPath: "qwen", // Will be found in PATH after npm installation
		mcpServers: mcpServers,
		agentArgs:  make([]string, len(agentConfig.Args)),
		agentEnv:   make(map[string]string),
	}

	// Copy args
	copy(agent.agentArgs, agentConfig.Args)

	// Copy env
	for k, v := range agentConfig.Env {
		agent.agentEnv[k] = v
	}

	return agent
}

// Name returns the agent name
func (q *QwenCode) Name() string {
	return q.config.Name
}

// Type returns the agent type
func (q *QwenCode) Type() string {
	return "qwen"
}

// Run executes Qwen with the given prompt and options
func (q *QwenCode) Run(ctx context.Context, prompt string, options RunOptions) (*AgentOutput, error) {
	lgr := logger.FromContext(ctx)

	// Build the command arguments
	args := q.buildArgs(options)

	var lastErr error
	var lastOutput *AgentOutput
	maxRetries := options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lgr.Info("Qwen execution attempt", zap.Int("attempt", attempt), zap.Int("max_retries", maxRetries))

		// Add continue flag when requested or for retry attempts
		currentArgs := args
		if options.ContinueMode || attempt > 1 {
			currentArgs = append(args, "--continue")
		}

		// Prepare output capture buffers
		var stdout, stderr bytes.Buffer

		// Execute Qwen
		cmd := exec.CommandContext(ctx, q.binaryPath, currentArgs...)

		// Set working directory to agent's directory so Qwen can find .qwen/settings.json
		// If user specified a working directory, use that; otherwise use agent directory
		if options.WorkingDirectory != "" {
			cmd.Dir = options.WorkingDirectory
		} else {
			// Get agent directory where .qwen/settings.json is located
			configAgent := &config.Agent{Name: q.config.Name}
			normalizedAgentName := configAgent.GetNormalizedName()
			cmd.Dir = fmt.Sprintf("/opt/autoteam/agents/%s", normalizedAgentName)
		}

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Stdin = strings.NewReader(prompt)

		// Set environment variables
		cmd.Env = os.Environ()
		// Add custom environment variables from agent config
		for k, v := range q.agentEnv {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		if err := cmd.Run(); err != nil {
			// Create output after command execution
			lastOutput = &AgentOutput{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}
			lastErr = fmt.Errorf("qwen execution failed (attempt %d): %w", attempt, err)
			lgr.Warn("Attempt failed",
				zap.Int("attempt", attempt),
				zap.Error(err),
				zap.String("stderr", stderr.String()))

			// Don't retry on context cancellation
			if ctx.Err() != nil {
				return lastOutput, ctx.Err()
			}

			// Wait before retry (exponential backoff)
			if attempt < maxRetries {
				backoff := time.Duration(attempt) * time.Second
				lgr.Info("Retrying", zap.Duration("backoff", backoff))
				select {
				case <-ctx.Done():
					return lastOutput, ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
		} else {
			// Create output after successful execution
			lastOutput = &AgentOutput{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}
			lgr.Info("Qwen execution succeeded",
				zap.Int("attempt", attempt),
				zap.Int("stdout_length", len(stdout.String())))
			return lastOutput, nil
		}
	}

	return lastOutput, fmt.Errorf("all %d attempts failed, last error: %w", maxRetries, lastErr)
}

// IsAvailable checks if Qwen is available
func (q *QwenCode) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, q.binaryPath, "--version")
	return cmd.Run() == nil
}

// CheckAvailability checks if Qwen Code is available, returns error if not found
func (q *QwenCode) CheckAvailability(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Checking Qwen Code availability")

	if !q.IsAvailable(ctx) {
		return fmt.Errorf("qwen command not found - please install Qwen Code using: npm install -g @qwen-code/qwen-code@latest")
	}

	lgr.Info("Qwen Code is available")
	return nil
}

// Version returns the Qwen version
func (q *QwenCode) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, q.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Qwen version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// buildArgs builds the command line arguments for Qwen
func (q *QwenCode) buildArgs(options RunOptions) []string {
	args := make([]string, 0)

	// Add default yolo parameter for non-interactive execution
	args = append(args, "--yolo")

	// Add custom args from agent configuration
	args = append(args, q.agentArgs...)

	// Qwen Code automatically looks for .qwen/settings.json in the current working directory
	// Since we set cmd.Dir to the agent's working directory, Qwen will find the .qwen/settings.json file we created there.

	return args
}

// Configure configures MCP servers for Qwen
func (q *QwenCode) Configure(ctx context.Context) error {
	return q.ConfigureForProject(ctx, "")
}

// ConfigureForProject configures MCP servers for a specific agent
func (q *QwenCode) ConfigureForProject(ctx context.Context, projectPath string) error {
	lgr := logger.FromContext(ctx)

	if len(q.mcpServers) == 0 {
		lgr.Debug("No MCP servers to configure for Qwen agent")
		return nil
	}

	lgr.Info("Configuring MCP servers for Qwen", zap.Int("count", len(q.mcpServers)), zap.String("agent", q.config.Name))

	// Create dedicated MCP configuration file for this agent
	if err := q.createMCPConfigFile(ctx); err != nil {
		return fmt.Errorf("failed to create MCP configuration file: %w", err)
	}

	lgr.Info("MCP servers configured successfully for Qwen")
	return nil
}

// getMCPConfigPath returns the path to the MCP configuration file for this agent
// Qwen Code looks for configuration in .qwen/settings.json in the project directory
func (q *QwenCode) getMCPConfigPath() string {
	// Use the agent name as passed from the factory (already normalized with variations)
	// Don't re-normalize as it would convert senior_developer/collector back to senior_developer_collector
	return fmt.Sprintf("/opt/autoteam/agents/%s/.qwen/settings.json", q.config.Name)
}

// createMCPConfigFile creates the MCP configuration file for this agent
func (q *QwenCode) createMCPConfigFile(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	mcpConfigPath := q.getMCPConfigPath()
	lgr.Info("Creating MCP configuration file for Qwen", zap.String("path", mcpConfigPath))

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(mcpConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create MCP config directory: %w", err)
	}

	// Start with existing settings or create new config
	qwenConfig := make(map[string]interface{})

	// Try to read existing settings file
	if existingData, err := os.ReadFile(mcpConfigPath); err == nil {
		if err := json.Unmarshal(existingData, &qwenConfig); err != nil {
			lgr.Warn("Failed to parse existing Qwen settings file, creating new one", zap.Error(err))
			qwenConfig = make(map[string]interface{})
		}
	}

	// Convert MCP servers to Qwen format (similar to Gemini/Claude format)
	mcpServersMap := make(map[string]interface{})
	for name, server := range q.mcpServers {
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

	// Add MCP servers to config while preserving existing settings
	qwenConfig["mcpServers"] = mcpServersMap

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(qwenConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Qwen config: %w", err)
	}

	// Write the configuration file
	if err := os.WriteFile(mcpConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write Qwen config file: %w", err)
	}

	lgr.Info("MCP configuration file created successfully for Qwen",
		zap.String("path", mcpConfigPath),
		zap.Int("mcp_servers", len(q.mcpServers)))

	return nil
}

// SetMCPServers sets the MCP servers for this agent
func (q *QwenCode) SetMCPServers(mcpServers map[string]config.MCPServer) {
	q.mcpServers = mcpServers
}

// CreateHTTPServer creates an HTTP API server for this agent
func (q *QwenCode) CreateHTTPServer(workingDir string, port int, apiKey string) HTTPServer {
	config := server.Config{
		Port:       port,
		APIKey:     apiKey,
		WorkingDir: workingDir,
	}
	return server.NewServer(q, config)
}
