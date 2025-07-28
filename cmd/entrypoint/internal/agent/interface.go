package agent

import (
	"context"
	"fmt"
)

// Agent represents an AI agent that can process prompts and generate responses
type Agent interface {
	// Name returns the name of the agent
	Name() string

	// Type returns the type identifier of the agent
	Type() string

	// Run executes the agent with the given prompt and options
	Run(ctx context.Context, prompt string, options RunOptions) error

	// IsAvailable checks if the agent is available and ready to use
	IsAvailable(ctx context.Context) bool

	// Install installs or updates the agent if needed
	Install(ctx context.Context) error

	// Version returns the current version of the agent
	Version(ctx context.Context) (string, error)
}

// RunOptions contains options for running an agent
type RunOptions struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// ContinueMode indicates whether to continue from a previous session
	ContinueMode bool

	// OutputFormat specifies the output format (e.g., "stream-json")
	OutputFormat string

	// Verbose enables verbose output
	Verbose bool

	// DryRun prevents actual execution
	DryRun bool

	// WorkingDirectory is the directory to run the agent in
	WorkingDirectory string
}

// Registry manages available agents
type Registry struct {
	agents map[string]Agent
}

// NewRegistry creates a new agent registry
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

// Register registers an agent with the registry
func (r *Registry) Register(agentType string, agent Agent) {
	r.agents[agentType] = agent
}

// Get retrieves an agent by type
func (r *Registry) Get(agentType string) (Agent, error) {
	agent, exists := r.agents[agentType]
	if !exists {
		return nil, fmt.Errorf("agent type %s not registered", agentType)
	}
	return agent, nil
}

// List returns all registered agent types
func (r *Registry) List() []string {
	var types []string
	for agentType := range r.agents {
		types = append(types, agentType)
	}
	return types
}

// GetEnabled returns all registered agents
func (r *Registry) GetEnabled() []Agent {
	var agents []Agent
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}