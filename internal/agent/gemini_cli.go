package agent

import (
	"autoteam/internal/config"
	"autoteam/internal/logger"
	"autoteam/internal/server"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// GeminiCli implements the Agent interface for Gemini CLI
type GeminiCli struct {
	name       string
	binaryPath string
	mcpServers map[string]config.MCPServer
	agentArgs  []string
	agentEnv   map[string]string
}

// NewGeminiCli creates a new Gemini agent instance
func NewGeminiCli(name string, args []string, env map[string]string, mcpServers map[string]config.MCPServer) *GeminiCli {
	return &GeminiCli{
		name:       name,
		binaryPath: "gemini", // Will be found in PATH after npm installation
		mcpServers: mcpServers,
		agentArgs:  args,
		agentEnv:   env,
	}
}

// Name returns the agent name
func (q *GeminiCli) Name() string {
	return q.name
}

// Type returns the agent type
func (q *GeminiCli) Type() string {
	return "gemini"
}

// Run executes Gemini with the given prompt and options
func (q *GeminiCli) Run(ctx context.Context, prompt string, options RunOptions) (*AgentOutput, error) {
	lgr := logger.FromContext(ctx)

	// Build the command arguments
	args := q.buildArgs()

	// Add continue flag when requested
	if options.ContinueMode {
		args = append(args, "--continue")
	}

	// Log the full prompt being sent for debugging
	lgr.Info("Sending full prompt to Gemini",
		zap.String("prompt_length", fmt.Sprintf("%d chars", len(prompt))),
		zap.String("full_prompt", prompt))

	// Prepare output capture buffers
	var stdout, stderr bytes.Buffer

	// Execute Gemini
	cmd := exec.CommandContext(ctx, q.binaryPath, args...)

	// Set working directory
	if options.WorkingDirectory != "" {
		cmd.Dir = options.WorkingDirectory
	} else {
		// Use the agent name as passed (already normalized with variations)
		cmd.Dir = fmt.Sprintf("/opt/autoteam/workers/%s", q.name)
	}

	// Log execution details for debugging
	lgr.Info("Executing Gemini CLI",
		zap.String("binary_path", q.binaryPath),
		zap.Strings("args", args),
		zap.String("working_dir", cmd.Dir))

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
		return &AgentOutput{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, fmt.Errorf("gemini execution failed: %w", err)
	}

	return &AgentOutput{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}

// IsAvailable checks if Gemini is available
func (q *GeminiCli) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, q.binaryPath, "--version")
	return cmd.Run() == nil
}

// CheckAvailability checks if Gemini CLI is available, returns error if not found
func (q *GeminiCli) CheckAvailability(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Checking Gemini CLI availability")

	if !q.IsAvailable(ctx) {
		return fmt.Errorf("gemini command not found - please install Gemini CLI using: npm install -g @google/gemini-cli")
	}

	lgr.Info("Gemini CLI is available")
	return nil
}

// Version returns the Gemini version
func (q *GeminiCli) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, q.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Gemini version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// buildArgs builds the command line arguments for Gemini
func (q *GeminiCli) buildArgs() []string {
	args := []string{"--yolo"} // Add default yolo parameter for non-interactive execution

	// Add custom args from agent configuration
	args = append(args, q.agentArgs...)

	// Gemini automatically looks for .gemini/settings.json in the current working directory
	// or ~/.gemini/settings.json globally. Since we set cmd.Dir to the agent's working directory,
	// Gemini will find the .gemini/settings.json file we created there.

	return args
}

// Configure configures MCP servers for Gemini
func (q *GeminiCli) Configure(ctx context.Context) error {
	return q.ConfigureForProject(ctx, "")
}

// ConfigureForProject configures MCP servers for a specific agent
func (q *GeminiCli) ConfigureForProject(ctx context.Context, projectPath string) error {
	lgr := logger.FromContext(ctx)

	if len(q.mcpServers) == 0 {
		lgr.Debug("No MCP servers to configure for Gemini agent")
		return nil
	}

	lgr.Info("Configuring MCP servers for Gemini", zap.Int("count", len(q.mcpServers)), zap.String("agent", q.name))

	// Create dedicated MCP configuration file for this agent
	if err := q.createMCPConfigFile(ctx); err != nil {
		return fmt.Errorf("failed to create MCP configuration file: %w", err)
	}

	lgr.Info("MCP servers configured successfully for Gemini")
	return nil
}

// getMCPConfigPath returns the path to the MCP configuration file for this agent
// Gemini looks for configuration in ~/.gemini/settings.json or project-specific .gemini/settings.json
func (q *GeminiCli) getMCPConfigPath() string {
	// Use the agent name as passed from the factory (already normalized with variations)
	// Don't re-normalize as it would convert senior_developer/collector back to senior_developer_collector
	return fmt.Sprintf("/opt/autoteam/workers/%s/.gemini/settings.json", q.name)
}

// createMCPConfigFile creates the MCP configuration file for this agent
func (q *GeminiCli) createMCPConfigFile(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	mcpConfigPath := q.getMCPConfigPath()
	lgr.Info("Creating MCP configuration file for Gemini", zap.String("path", mcpConfigPath))

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(mcpConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create MCP config directory: %w", err)
	}

	// Start with existing settings or create new config
	geminiConfig := make(map[string]interface{})

	// Try to read existing settings file
	if existingData, err := os.ReadFile(mcpConfigPath); err == nil {
		if err := json.Unmarshal(existingData, &geminiConfig); err != nil {
			lgr.Warn("Failed to parse existing Gemini settings file, creating new one", zap.Error(err))
			geminiConfig = make(map[string]interface{})
		}
	}

	// Convert MCP servers to Gemini format (same as Claude/Qwen format)
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
	geminiConfig["mcpServers"] = mcpServersMap

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(geminiConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Gemini config: %w", err)
	}

	// Write the configuration file
	if err := os.WriteFile(mcpConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write Gemini config file: %w", err)
	}

	lgr.Info("MCP configuration file created successfully for Gemini",
		zap.String("path", mcpConfigPath),
		zap.Int("mcp_servers", len(q.mcpServers)))

	return nil
}

// SetMCPServers sets the MCP servers for this agent
func (q *GeminiCli) SetMCPServers(mcpServers map[string]config.MCPServer) {
	q.mcpServers = mcpServers
}

// CreateHTTPServer creates an HTTP API server for this agent
func (q *GeminiCli) CreateHTTPServer(workingDir string, port int, apiKey string) HTTPServer {
	config := server.Config{
		Port:       port,
		APIKey:     apiKey,
		WorkingDir: workingDir,
	}
	return server.NewServer(q, config)
}
