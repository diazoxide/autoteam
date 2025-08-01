# AutoTeam

Universal AI Agent Management System for automated GitHub workflows across multiple repositories.

## Overview

AutoTeam is a configurable system that deploys AI agents to automatically handle GitHub issues, pull requests, and reviews across multiple repositories. Instead of manually checking GitHub and working on tasks, this system continuously monitors for new work and automatically provisions containerized development environments with specialized AI agents.

## Features

- **Multi-Repository Support**: Monitor and work across multiple repositories with pattern matching and regex support
- **Universal Configuration**: Single YAML file to define repositories, agents, and settings
- **Dynamic Agent Scaling**: Support for any number of specialized agents
- **Smart Review Detection**: Intelligent workflow handling that distinguishes between developer and reviewer actions
- **Repository Pattern Matching**: Flexible include/exclude patterns with regex support (`/pattern/` syntax)
- **Smart Name Normalization**: Automatically handles agent names with spaces and special characters
- **Template-Based Generation**: Docker Compose and entrypoint scripts generated from templates
- **Role-Based Agents**: Each agent can have specialized prompts and responsibilities
- **Agent-Specific Settings**: Per-agent Docker images, volumes, and environment overrides
- **Consolidated Prompt System**: Unified prompt handling with collaborator awareness
- **Organized File Structure**: All generated files in `.autoteam/` directory
- **Continuous Monitoring**: Configurable intervals for checking GitHub activity
- **Docker Integration**: Containerized environments with volume mounting and networking
- **Security Validation**: GitHub token/user validation for enhanced security
- **Cross-Platform Support**: macOS and Linux with universal installation script

## Quick Start

### 1. Installation

**Quick Install (Recommended):**
```bash
# Install latest version (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/diazoxide/autoteam/main/scripts/install.sh | bash
```

**Manual Install:**
```bash
# Download for your platform from releases page
# Or build from source
make build && make install
```

**Verify Installation:**
```bash
autoteam --version
```

See [INSTALL.md](INSTALL.md) for comprehensive installation instructions.

### 2. Initialize Configuration

```bash
autoteam init
```

This creates a sample `autoteam.yaml` with basic configuration.

### 3. Configure Your Setup

Edit `autoteam.yaml` to match your repositories and requirements:

```yaml
# Multi-repository configuration
repositories:
  include:
    - "myorg/project-alpha"           # Exact repository match
    - "/myorg\\/backend-.*/'"         # Regex pattern for multiple repos
    - "/diazoxide\\/.*/'"             # All repositories from diazoxide user
  exclude:
    - "myorg/legacy-project"          # Exclude specific repository
    - "/.*-archived$/'"               # Exclude archived repositories

agents:
  - name: "developer"
    prompt: "You are a developer agent responsible for implementing features and fixing bugs."
    github_token: "ghp_your_developer_token_here"
    github_user: "your-github-username"
  - name: "reviewer"
    prompt: "You are a code reviewer focused on quality and best practices."
    github_token: "ghp_your_reviewer_token_here"
    github_user: "your-github-username"
    settings:
      docker_image: "golang:1.21"  # Custom image for reviewer
      volumes:
        - "./tools:/opt/tools:ro"  # Additional volume mount

settings:
  docker_image: "node:18.17.1"
  docker_user: "developer"
  check_interval: 60
  team_name: "autoteam"
  install_deps: true
```

### 4. Add Your GitHub Tokens

You have two options for providing GitHub tokens:

**Option A: Direct in autoteam.yaml**
```yaml
agents:
  - name: "developer"
    github_token: "ghp_your_actual_developer_token"
  - name: "reviewer"  
    github_token: "ghp_your_actual_reviewer_token"
```

**Option B: Using .env file (recommended for security)**
Create a `.env` file in your project root:
```bash
# .env file
DEVELOPER_TOKEN=ghp_your_actual_developer_token
REVIEWER_TOKEN=ghp_your_actual_reviewer_token
```

Then reference them in `autoteam.yaml`:
```yaml
agents:
  - name: "developer"
    github_token: "${DEVELOPER_TOKEN}"
  - name: "reviewer"  
    github_token: "${REVIEWER_TOKEN}"
```

### 5. Deploy Your Team

```bash
# Generate Docker Compose configuration
autoteam generate

# Start the automated team
autoteam up

# Stop when needed
autoteam down
```

## Configuration

### Repository Settings

**Multi-Repository Support:**
- `repositories.include`: List of repository patterns to monitor
  - Exact matches: `"owner/repo"`
  - Regex patterns: `"/owner\\/prefix-.*/"`
  - User patterns: `"/username\\/.*/"`
- `repositories.exclude`: List of repository patterns to exclude (optional)

**Repository Pattern Examples:**
```yaml
repositories:
  include:
    - "myorg/main-project"        # Single repository
    - "/myorg\\/api-.*/'"         # All repositories starting with "api-"
    - "/username\\/.*/'"          # All repositories from a specific user
  exclude:
    - "myorg/legacy-system"       # Exclude specific repository
    - "/.*-archive$/'"            # Exclude archived repositories
```

### Agent Configuration

- `name`: Unique identifier for the agent (supports names with spaces and special characters)
- `prompt`: Primary role and responsibilities
- `github_token`: GitHub personal access token for this agent
- `github_user`: GitHub username associated with the token (required for security validation)
- `enabled`: Enable/disable agent without removing configuration (optional, defaults to true)
- `settings`: Agent-specific overrides for global settings (optional)
  - `docker_image`: Custom Docker image for this agent
  - `docker_user`: Custom user for this agent
  - `volumes`: Additional volume mounts
  - `environment`: Additional environment variables

### Settings

- `docker_image`: Docker image for agent containers
- `docker_user`: User account inside containers  
- `check_interval`: Monitoring frequency in seconds
- `team_name`: Project name used in paths
- `install_deps`: Install dependencies on startup
- `common_prompt`: Common instructions shared by all agents (optional)
- `max_attempts`: Maximum retry attempts for failed items (default: 3)
- `volumes`: Global volume mounts applied to all agents (optional)
- `environment`: Global environment variables for all agents (optional)

### Smart Review Detection

AutoTeam includes intelligent review workflow detection that:

- **Tracks Review States**: Monitors PR review states (approved, changes requested, commented)
- **Detects Re-Review Requests**: Automatically excludes PRs where developers have re-requested review
- **Workflow-Aware**: Distinguishes between "developer action needed" vs "waiting for reviewer response"
- **Prevents False Positives**: Avoids repeated notifications when waiting for reviewer feedback

**How It Works:**
1. When a PR has "changes requested" reviews → marked as pending developer action
2. Developer addresses feedback and re-requests review → automatically excluded from pending
3. Reviewer responds → PR returns to appropriate workflow state
4. Agents get clear prompts to complete review cycles properly

## Examples

See the [`examples/`](examples/) directory for various configuration patterns:

- **basic-setup.yaml**: Simple two-agent setup
- **multi-role-team.yaml**: Comprehensive team with specialized roles
- **minimal-config.yaml**: Absolute minimum configuration
- **custom-docker.yaml**: Custom Docker image example

## CLI Commands

```bash
autoteam init      # Create sample autoteam.yaml
autoteam generate  # Generate compose.yaml from config
autoteam up        # Generate and start containers
autoteam down      # Stop containers
autoteam agents     # List all agents and their states
```

All generated files are organized in the `.autoteam/` directory for better project organization.

### Agent Name Normalization

AutoTeam automatically normalizes agent names for Docker Compose services and directory paths while preserving the original names for identification:

```yaml
agents:
  - name: "Senior Developer"      # Original name (used in environment variables)
    prompt: "You are a senior developer"
    github_token: "ghp_token1"
    github_user: "senior-dev-user"
  - name: "API Agent #1"          # Original name with special characters
    prompt: "You are an API specialist"
    github_token: "ghp_token2"
    github_user: "api-dev-user"
```

This generates Docker Compose services with normalized names:
- `Senior Developer` → `senior_developer` (service name)
- `API Agent #1` → `api_agent_1` (service name)

Directory structure uses normalized names:
```
.autoteam/agents/
├── senior_developer/
│   └── codebase/
└── api_agent_1/
    └── codebase/
```

### Disabling Agents

You can temporarily disable agents without removing their configuration:

```yaml
agents:
  - name: "developer"
    prompt: "You are a developer agent"
    github_token: "ghp_token1"
    github_user: "dev-user"
    enabled: true  # Agent is active (default)
  
  - name: "reviewer"
    prompt: "You are a code reviewer"
    github_token: "ghp_token2"
    github_user: "reviewer-user"
    enabled: false  # Agent is disabled - won't be deployed
```

Benefits:
- Keep agent configurations for future use
- Temporarily reduce resource usage
- Test with specific agent combinations
- Preserve tokens and settings when not actively needed

Use `autoteam agents` to list all agents and their states.

## Architecture

```
autoteam.yaml → Generator → .autoteam/compose.yaml + entrypoints/
      ↓                           ↓
Multi-Repo Config → Docker Compose → Agent Containers
      ↓                           ↓
Pattern Matching → GitHub Monitoring → Claude Code → Cross-Repo Tasks
```

### Multi-Repository Structure

Each agent maintains separate working directories for each repository:

```
./
├── autoteam.yaml          # Configuration
└── .autoteam/             # Generated files directory
    ├── compose.yaml       # Docker Compose configuration
    ├── agents/            # Agent-specific directories
    │   ├── agent1/
    │   │   ├── codebase/
    │   │   │   ├── owner1-repo1/    # Repository-specific clone
    │   │   │   ├── owner1-repo2/    # Multiple repos per agent
    │   │   │   └── owner2-repo3/
    │   │   └── claude/              # Claude configuration
    │   └── agent2/
    │       ├── codebase/
    │       │   ├── owner1-repo1/    # Same repos, separate working dirs
    │       │   └── owner1-repo2/
    │       └── claude/
    ├── entrypoints/       # Agent entrypoint binaries
    │   ├── autoteam-entrypoint-*
    │   └── entrypoint.sh
    └── shared/            # Shared configurations
        ├── .claude
        └── .claude.json
```

### Repository Pattern Matching

- **Include Patterns**: Define which repositories to monitor
- **Exclude Patterns**: Filter out unwanted repositories  
- **Regex Support**: Use `/pattern/` syntax for complex matching
- **Dynamic Discovery**: Automatically discovers matching repositories

## Testing

The project includes comprehensive test coverage:

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/config
go test ./internal/generator
go test ./cmd/autoteam
go test ./cmd/entrypoint
```

### Test Coverage

- **Unit Tests**: Config parsing, validation, template generation
- **Integration Tests**: CLI command workflows
- **Template Tests**: Docker Compose and entrypoint generation
- **Normalization Tests**: Agent name normalization with various edge cases
- **Error Handling**: Invalid configurations and edge cases

## Development

### Project Structure

```
./
├── cmd/
│   ├── autoteam/          # Main CLI application
│   └── entrypoint/        # Go binary entrypoint for agents
├── internal/
│   ├── config/            # Configuration parsing
│   ├── generator/         # Template generation & embedded templates
│   └── testutil/          # Test utilities
├── examples/              # Configuration examples
├── scripts/               # Installation and utility scripts
└── .autoteam/             # Generated agent directories (created at runtime)
```

### Building & Development

```bash
# Development build
make dev

# Build for current platform
make build

# Build for all platforms  
make build-all

# Run tests
make test

# Format and lint
make check

# Create release packages
make package

# Install to system
make install
```

See `make help` for all available targets.

## Security Considerations

- Use separate GitHub tokens for each agent
- Configure minimal required permissions for GitHub tokens
- Ensure `github_user` matches the token owner (validated automatically)
- Regularly rotate access tokens
- Monitor API rate limits
- Review generated Docker configurations
- Use `.env` files for sensitive tokens instead of committing them to version control

## Troubleshooting

### Common Issues

1. **GitHub Authentication**: Ensure tokens are properly set and have required permissions
2. **Docker Issues**: Verify Docker and Docker Compose are installed and running
3. **Rate Limits**: Monitor GitHub API usage with multiple agents
4. **Port Conflicts**: Check for container port conflicts
5. **Permission Issues**: Ensure proper file permissions for generated scripts
6. **Repository Pattern Matching**: Verify regex patterns are properly escaped
7. **Review Detection**: Check that agents are submitting reviews properly and re-requesting when needed
8. **Multi-Repository Access**: Ensure GitHub tokens have access to all configured repositories

### Debug Mode

```bash
# Check generated files
autoteam generate
cat .autoteam/compose.yaml
ls .autoteam/entrypoints/

# Verify repository pattern matching
docker-compose -f .autoteam/compose.yaml logs | grep "Found.*repositories"

# Check individual agent working directories
ls .autoteam/agents/agent-name/codebase/

# Test individual containers
docker-compose -f .autoteam/compose.yaml up agent-name

# View container logs with repository context
docker-compose -f .autoteam/compose.yaml logs agent-name | grep -E "(Repository|Pending|Review)"

# Monitor real-time multi-repository activity
docker-compose -f .autoteam/compose.yaml logs -f --tail=50
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
