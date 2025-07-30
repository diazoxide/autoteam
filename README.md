# AutoTeam

Universal AI Agent Management System for automated GitHub workflows.

## Overview

AutoTeam is a configurable system that deploys AI agents to automatically handle GitHub issues, pull requests, and reviews. Instead of manually checking GitHub and working on tasks, this system continuously monitors for new work and automatically provisions containerized development environments with specialized AI agents.

## Features

- **Universal Configuration**: Single YAML file to define repository, agents, and settings
- **Dynamic Agent Scaling**: Support for any number of specialized agents
- **Template-Based Generation**: Docker Compose and entrypoint scripts generated from templates
- **Role-Based Agents**: Each agent can have specialized prompts and responsibilities
- **Agent-Specific Settings**: Per-agent Docker images, volumes, and environment overrides
- **Organized File Structure**: All generated files in `.autoteam/` directory
- **Continuous Monitoring**: Configurable intervals for checking GitHub activity
- **Docker Integration**: Containerized environments with volume mounting and networking
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

Edit `autoteam.yaml` to match your repository and requirements:

```yaml
repository:
  url: "diazoxide/autoteam"
  main_branch: "main"

agents:
  - name: "developer"
    prompt: "You are a developer agent responsible for implementing features and fixing bugs."
    github_token: "ghp_your_developer_token_here"
  - name: "reviewer"
    prompt: "You are a code reviewer focused on quality and best practices."
    github_token: "ghp_your_reviewer_token_here"
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

- `url`: GitHub repository in "owner/repo" format
- `main_branch`: Main branch name (defaults to "main")

### Agent Configuration

- `name`: Unique identifier for the agent
- `prompt`: Primary role and responsibilities
- `github_token`: GitHub personal access token for this agent
- `common_prompt`: Additional instructions for all agents
- `settings`: Agent-specific overrides for global settings (optional)
  - `docker_image`: Custom Docker image for this agent
  - `docker_user`: Custom user for this agent
  - `volumes`: Additional volume mounts
  - `environment`: Additional environment variables
  - `entrypoint`: Custom entrypoint override

### Settings

- `docker_image`: Docker image for agent containers
- `docker_user`: User account inside containers  
- `check_interval`: Monitoring frequency in seconds
- `team_name`: Project name used in paths
- `install_deps`: Install dependencies on startup

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
```

All generated files are organized in the `.autoteam/` directory for better project organization.

## Architecture

```
autoteam.yaml → Generator → .autoteam/compose.yaml + entrypoints/
                     ↓
              Docker Compose → Agent Containers
                     ↓
              GitHub Monitoring → Claude Code → Automated Tasks
```

### Generated Structure

```
./
├── autoteam.yaml          # Configuration
└── .autoteam/             # Generated files directory
    ├── compose.yaml       # Docker Compose configuration
    ├── agents/            # Agent-specific directories
    │   ├── agent1/
    │   │   ├── codebase/  # Repository clone
    │   │   └── claude/    # Claude configuration
    │   └── agent2/
    │       ├── codebase/
    │       └── claude/
    ├── entrypoints/       # Agent entrypoint binaries
    │   ├── autoteam-entrypoint-*
    │   └── entrypoint.sh
    └── shared/            # Shared configurations
        ├── .claude
        └── .claude.json
```

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
- Configure minimal required permissions
- Regularly rotate access tokens
- Monitor API rate limits
- Review generated Docker configurations

## Troubleshooting

### Common Issues

1. **GitHub Authentication**: Ensure tokens are properly set and have required permissions
2. **Docker Issues**: Verify Docker and Docker Compose are installed and running
3. **Rate Limits**: Monitor GitHub API usage with multiple agents
4. **Port Conflicts**: Check for container port conflicts
5. **Permission Issues**: Ensure proper file permissions for generated scripts

### Debug Mode

```bash
# Check generated files
autoteam generate
cat .autoteam/compose.yaml
ls .autoteam/entrypoints/

# Test individual containers
docker-compose -f .autoteam/compose.yaml up agent-name

# View container logs
docker-compose -f .autoteam/compose.yaml logs agent-name
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
