# Development Guide

## Overview

AutoTeam is built in Go with a focus on modularity, testability, and maintainability. This guide covers contributing to AutoTeam, building from source, and extending functionality.

## Development Environment Setup

### Prerequisites

- **Go 1.22+** - [Install Go](https://golang.org/doc/install)
- **Docker & Docker Compose** - [Install Docker](https://docs.docker.com/get-docker/)
- **Git** - Version control
- **Make** - Build automation
- **GitHub CLI (optional)** - For GitHub integration testing

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/diazoxide/autoteam.git
cd autoteam

# Install dependencies
go mod tidy

# Run tests
make test

# Build binaries
make build        # Main binary only
make build-all    # All binaries for all platforms

# Run locally
./build/autoteam --version
```

### Development Workflow

```bash
# Format code
make fmt

# Run linters
make vet

# Run all checks (format, vet, test)
make check

# Build and install locally
make install
```

## Project Structure

```
autoteam/
├── cmd/
│   ├── autoteam/          # Main CLI application
│   └── worker/            # Worker binary for agents
├── internal/
│   ├── config/            # Configuration parsing and validation
│   ├── generator/         # Template generation engine
│   │   └── templates/     # Embedded Docker Compose templates
│   ├── flow/              # Flow execution engine
│   ├── agent/             # AI agent implementations
│   ├── mcp/               # MCP server management
│   └── logger/            # Structured logging
├── pkg/                   # Public APIs (if any)
├── examples/              # Example configurations
├── scripts/               # Installation and utility scripts
├── docs/                  # Documentation
├── .github/               # GitHub workflows and templates
├── Makefile              # Build automation
├── go.mod                # Go module definition
└── README.md             # Project overview
```

## Architecture Deep Dive

### Core Components

#### Configuration System

The configuration system handles YAML parsing, validation, and environment variable substitution:

```go
// internal/config/config.go
type Config struct {
    Workers  []Worker  `yaml:"workers"`
    Settings Settings  `yaml:"settings"`
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    return config.Validate()
}
```

Key features:
- Environment variable substitution
- Configuration validation
- Default value handling
- Error reporting with line numbers

#### Flow Execution Engine

The flow engine orchestrates AI agents through workflow execution:

```go
// internal/flow/executor.go
type Executor struct {
    logger *zap.Logger
    agents map[string]agent.Agent
}

func (e *Executor) Execute(ctx context.Context, flow []FlowStep) error {
    // Resolve dependencies into execution levels
    levels := e.resolveDependencies(flow)
    
    // Execute each level in parallel
    for _, level := range levels {
        if err := e.executeLevel(ctx, level); err != nil {
            return err
        }
    }
    
    return nil
}
```

Features:
- Dependency resolution algorithm
- Parallel execution within levels
- Error handling and recovery
- Step output chaining
- Conditional execution

#### Agent Interface

All AI agents implement a common interface:

```go
// internal/agent/interface.go
type Agent interface {
    Name() string
    Type() string
    Run(ctx context.Context, prompt string, args []string) (*AgentOutput, error)
    Configure(config AgentConfig) error
    CheckAvailability() error
}

type AgentOutput struct {
    Stdout string
    Stderr string
    Success bool
}
```

Current implementations:
- **Claude Agent** - Uses Claude CLI
- **Gemini Agent** - Uses Gemini CLI tools
- **Qwen Agent** - Uses Qwen CLI interface

### Template Generation

AutoTeam uses Go templates to generate Docker Compose files:

```go
// internal/generator/generator.go
func (g *Generator) Generate(config *config.Config) error {
    tmpl, err := template.ParseFS(templates.FS, "compose.yaml.tmpl")
    if err != nil {
        return err
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, config); err != nil {
        return err
    }
    
    return os.WriteFile("compose.yaml", buf.Bytes(), 0644)
}
```

Templates are embedded at build time using `go:embed`:

```go
//go:embed templates/*
var FS embed.FS
```

## Adding New Features

### Adding a New Agent Type

1. **Implement the Agent interface**:

```go
// internal/agent/custom_agent.go
type CustomAgent struct {
    name   string
    config AgentConfig
    logger *zap.Logger
}

func NewCustomAgent(name string, logger *zap.Logger) *CustomAgent {
    return &CustomAgent{
        name:   name,
        logger: logger,
    }
}

func (ca *CustomAgent) Name() string {
    return ca.name
}

func (ca *CustomAgent) Type() string {
    return "custom"
}

func (ca *CustomAgent) Run(ctx context.Context, prompt string, args []string) (*AgentOutput, error) {
    // Implement custom agent execution logic
    return &AgentOutput{
        Stdout:  "Custom agent response",
        Success: true,
    }, nil
}

func (ca *CustomAgent) Configure(config AgentConfig) error {
    ca.config = config
    return nil
}

func (ca *CustomAgent) CheckAvailability() error {
    // Check if custom agent dependencies are available
    return nil
}
```

2. **Register the agent type**:

```go
// internal/agent/factory.go
func CreateAgent(agentType, name string, logger *zap.Logger) (Agent, error) {
    switch agentType {
    case "claude":
        return NewClaudeAgent(name, logger), nil
    case "gemini":
        return NewGeminiAgent(name, logger), nil
    case "qwen":
        return NewQwenAgent(name, logger), nil
    case "custom":
        return NewCustomAgent(name, logger), nil
    default:
        return nil, fmt.Errorf("unknown agent type: %s", agentType)
    }
}
```

3. **Update configuration validation**:

```go
// internal/config/validation.go
var validAgentTypes = []string{"claude", "gemini", "qwen", "custom"}
```

### Adding MCP Server Support

1. **Create MCP server configuration**:

```go
// internal/mcp/server.go
type CustomMCPServer struct {
    Name    string
    Command string
    Args    []string
    Env     map[string]string
}

func (s *CustomMCPServer) Start(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, s.Command, s.Args...)
    cmd.Env = s.buildEnvironment()
    
    return cmd.Start()
}
```

2. **Add to MCP server registry**:

```go
// internal/mcp/registry.go
var knownMCPServers = map[string]MCPServerInfo{
    "github":    {Command: "/opt/autoteam/bin/github-mcp-server", DefaultArgs: []string{"stdio"}},
    "slack":     {Command: "/opt/autoteam/bin/slack-mcp-server", DefaultArgs: []string{"stdio"}},
    "custom":    {Command: "/opt/autoteam/bin/custom-mcp-server", DefaultArgs: []string{"stdio"}},
}
```

### Extending Flow System

Add new flow step types or execution patterns:

```go
// internal/flow/step.go
type FlowStep struct {
    Name      string            `yaml:"name"`
    Type      string            `yaml:"type"`
    Prompt    string            `yaml:"prompt"`
    DependsOn []string          `yaml:"depends_on"`
    SkipWhen  string            `yaml:"skip_when"`
    Args      []string          `yaml:"args"`
    Output    string            `yaml:"output"`
    Custom    map[string]string `yaml:"custom"`  // New: custom step properties
}
```

Add step execution logic:

```go
// internal/flow/executor.go
func (e *Executor) executeStep(ctx context.Context, step FlowStep) error {
    // Handle custom step properties
    if step.Custom != nil {
        return e.executeCustomStep(ctx, step)
    }
    
    // Standard step execution
    return e.executeStandardStep(ctx, step)
}
```

## Testing

### Unit Tests

AutoTeam uses standard Go testing:

```go
// internal/config/config_test.go
func TestConfigLoad(t *testing.T) {
    config, err := Load("testdata/valid_config.yaml")
    assert.NoError(t, err)
    assert.Equal(t, "Test Team", config.Settings.TeamName)
}

func TestConfigValidation(t *testing.T) {
    config := &Config{
        Workers: []Worker{
            {Name: "", Enabled: true}, // Invalid: empty name
        },
    }
    
    err := config.Validate()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "worker name cannot be empty")
}
```

### Integration Tests

Test full workflows with real configurations:

```go
// internal/flow/integration_test.go
func TestFlowExecution(t *testing.T) {
    ctx := context.Background()
    
    flow := []FlowStep{
        {Name: "step1", Type: "claude", Prompt: "Test prompt"},
        {Name: "step2", Type: "gemini", Prompt: "Test step 2", DependsOn: []string{"step1"}},
    }
    
    executor := NewExecutor(logger, mockAgents)
    err := executor.Execute(ctx, flow)
    assert.NoError(t, err)
}
```

### Test Data

Use `testdata/` directories for test configurations:

```
internal/config/testdata/
├── valid_config.yaml
├── invalid_config.yaml
├── minimal_config.yaml
└── complex_config.yaml
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test package
go test ./internal/config -v

# Run tests with race detection
go test -race ./...
```

## Building and Releasing

### Build System

The Makefile provides comprehensive build targets:

```makefile
# Build for current platform
build: $(BUILD_DIR)/$(BINARY_NAME)

# Build for all platforms
build-all: clean-build
	@$(MAKE) -j$(shell nproc) $(PLATFORMS) $(PLATFORMS:=/worker)

# Cross-compilation targets
$(PLATFORMS):
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) $(MAIN_PATH)
```

### Release Process

1. **Version Tagging**:
```bash
git tag v1.0.0
git push origin v1.0.0
```

2. **Automated Release** (GitHub Actions):
```yaml
# .github/workflows/release.yml
- name: Build Release Binaries
  run: make build-all

- name: Create Release
  uses: actions/create-release@v1
  with:
    tag_name: ${{ github.ref }}
    release_name: Release ${{ github.ref }}
```

3. **Manual Release**:
```bash
# Build all platforms
make build-all

# Create packages
make package

# Generate checksums
make checksums
```

## Code Style and Standards

### Go Code Style

Follow standard Go conventions:

- Use `gofmt` for formatting
- Use `go vet` for static analysis
- Follow effective Go patterns
- Use meaningful variable and function names
- Write clear, concise comments

### Project Conventions

**Logging**:
```go
// Use structured logging with context
logger := logger.FromContext(ctx)
logger.Info("processing flow step", 
    zap.String("step", step.Name),
    zap.String("type", step.Type))
```

**Error Handling**:
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to execute step %s: %w", step.Name, err)
}
```

**Configuration**:
```go
// Use yaml tags and validation
type Config struct {
    TeamName     string `yaml:"team_name" validate:"required"`
    SleepDuration int   `yaml:"sleep_duration" validate:"min=1"`
}
```

## Contributing

### Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork**:
```bash
git clone https://github.com/yourusername/autoteam.git
```

3. **Create a feature branch**:
```bash
git checkout -b feature/your-feature-name
```

4. **Make your changes** following the style guide
5. **Run tests**:
```bash
make check
```

6. **Commit your changes**:
```bash
git commit -m "Add your feature description"
```

7. **Push and create a pull request**

### Pull Request Guidelines

- **Clear description** of changes and motivation
- **Test coverage** for new functionality
- **Documentation updates** if needed
- **Small, focused changes** are preferred
- **Follow existing code style** and patterns

### Areas for Contribution

**High Priority:**
- MCP server implementations for popular platforms
- Performance optimizations for large-scale deployments
- Additional AI agent integrations
- Enhanced error handling and recovery

**Medium Priority:**
- Visual flow designer (web UI)
- Advanced monitoring and metrics
- Configuration validation improvements
- Documentation and examples

**Low Priority:**
- Additional output formats
- Custom webhook integrations
- Advanced scheduling features
- Plugin system architecture

## Debugging

### Development Debugging

```bash
# Run with debug logging
LOG_LEVEL=debug ./build/autoteam up

# Debug specific component
DEBUG=flow,agent ./build/autoteam up
```

### Container Debugging

```bash
# Inspect running containers
docker compose ps

# View container logs
docker compose logs -f autoteam-worker

# Execute commands in container
docker compose exec autoteam-worker /bin/bash

# Debug MCP server communication
docker compose exec autoteam-worker cat /var/log/mcp.log
```

### Performance Profiling

```go
import _ "net/http/pprof"

// Add to main function
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Then use `go tool pprof http://localhost:6060/debug/pprof/profile`

## Documentation

### Writing Documentation

- Use clear, concise language
- Include practical examples
- Keep documentation up-to-date with code changes
- Use Mermaid diagrams for architecture visualization

### Documentation Structure

- **README.md** - Project overview and quick start
- **docs/** - Detailed documentation by topic
- **examples/** - Real-world configuration examples
- **Code comments** - Explain complex logic and APIs

## Next Steps

- [Architecture](architecture.md) - Deep dive into system design
- [Examples](examples.md) - Real-world usage patterns
- [Configuration](configuration.md) - Advanced configuration options