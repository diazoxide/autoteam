package agent

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"autoteam/internal/worker"
)

// DebugAgent implements the Agent interface for debugging purposes
type DebugAgent struct {
	name       string
	mcpServers map[string]worker.MCPServer
	args       []string
	env        map[string]string
}

// NewDebugAgent creates a new Debug agent instance
func NewDebugAgent(name string, args []string, env map[string]string, mcpServers map[string]worker.MCPServer) Agent {
	return &DebugAgent{
		name:       name,
		mcpServers: mcpServers,
		args:       args,
		env:        env,
	}
}

// Name returns the agent name
func (d *DebugAgent) Name() string {
	return d.name
}

// Type returns the agent type
func (d *DebugAgent) Type() string {
	return "debug"
}

// Run executes the debug agent with the given prompt
func (d *DebugAgent) Run(ctx context.Context, prompt string, options RunOptions) (*AgentOutput, error) {
	randState := rand.Float32() < 0.5

	mn := 5
	mx := 20
	secs := rand.Intn(mx-mn) + mn

	// Respect context cancellation during sleep
	select {
	case <-time.After(time.Duration(secs) * time.Second):
		// Normal completion - continue execution
	case <-ctx.Done():
		// Context was cancelled
		return &AgentOutput{
			Stdout: "",
			Stderr: fmt.Sprintf("debug agent cancelled: %v", ctx.Err()),
		}, ctx.Err()
	}

	if randState {
		return &AgentOutput{
			Stdout: fmt.Sprintf("Debug agent '%s' executed with prompt: %s", d.name, prompt),
			Stderr: "",
		}, nil
	}

	return &AgentOutput{
		Stdout: "",
		Stderr: "Error happened",
	}, fmt.Errorf("debug agent simulated failure")
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
