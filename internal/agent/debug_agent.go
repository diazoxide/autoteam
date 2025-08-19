package agent

import (
	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
	"context"
	"fmt"
)

// DebugAgent implements the Agent interface for debugging purposes
type DebugAgent struct {
	config     entrypoint.AgentConfig
	mcpServers map[string]config.MCPServer
}

// NewDebugAgent creates a new Debug agent instance
func NewDebugAgent(agentConfig config.AgentConfig, name string, mcpServers map[string]config.MCPServer) Agent {
	cfg := entrypoint.AgentConfig{Name: name}
	return &DebugAgent{
		config:     cfg,
		mcpServers: mcpServers,
	}
}

// Name returns the agent name
func (d *DebugAgent) Name() string {
	return d.config.Name
}

// Type returns the agent type
func (d *DebugAgent) Type() string {
	return "debug"
}

// Run executes the debug agent with the given prompt
func (d *DebugAgent) Run(ctx context.Context, prompt string, options RunOptions) (*AgentOutput, error) {
	return &AgentOutput{
		Stdout: fmt.Sprintf("Debug agent '%s' executed with prompt: %s", d.config.Name, prompt),
		Stderr: "",
	}, nil
}

// IsAvailable checks if the debug agent is available (always true for debug)
func (d *DebugAgent) IsAvailable(ctx context.Context) bool {
	return true
}

// CheckAvailability checks if the debug agent is available (always true for debug)
func (d *DebugAgent) CheckAvailability(ctx context.Context) error {
	return nil
}

// Version returns the debug agent version
func (d *DebugAgent) Version(ctx context.Context) (string, error) {
	return "debug-1.0.0", nil
}
