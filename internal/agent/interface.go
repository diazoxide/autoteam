package agent

import (
	"context"
)

// HTTPServer represents the HTTP API server interface for agents
type HTTPServer interface {
	// Start starts the HTTP server
	Start(ctx context.Context) error

	// Stop gracefully stops the HTTP server
	Stop(ctx context.Context) error

	// Port returns the server port
	Port() int

	// IsRunning returns true if the server is running
	IsRunning() bool

	// GetURL returns the base URL for the server
	GetURL() string

	// GetDocsURL returns the documentation URL
	GetDocsURL() string
}

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

// HTTPServerCapable represents an agent that can provide HTTP API server functionality
type HTTPServerCapable interface {
	// CreateHTTPServer creates an HTTP API server for this agent
	CreateHTTPServer(workingDir string, port int, apiKey string) HTTPServer
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
