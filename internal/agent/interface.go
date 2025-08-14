package agent

import (
	"context"
)

// AgentOutput contains the output from an agent execution
type AgentOutput struct {
	Stdout string
	Stderr string
}

// Agent represents an AI agent that can process prompts and generate responses
type Agent interface {
	// Name returns the name of the agent
	Name() string

	// Type returns the type identifier of the agent
	Type() string

	// Run executes the agent with the given prompt and options, returns output
	Run(ctx context.Context, prompt string, options RunOptions) (*AgentOutput, error)

	// IsAvailable checks if the agent is available and ready to use
	IsAvailable(ctx context.Context) bool

	// CheckAvailability checks if the agent is available and returns an error with installation instructions if not
	CheckAvailability(ctx context.Context) error

	// Version returns the current version of the agent
	Version(ctx context.Context) (string, error)
}

// Configurable represents an agent that supports configuration
type Configurable interface {
	// Configure performs any necessary configuration for the agent
	Configure(ctx context.Context) error

	// ConfigureForProject performs configuration for a specific project path
	ConfigureForProject(ctx context.Context, projectPath string) error
}

// RunOptions contains options for running an agent
type RunOptions struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// ContinueMode indicates whether to continue from a previous session
	ContinueMode bool

	// WorkingDirectory is the directory to run the agent in
	WorkingDirectory string
}
