# Auto-Team

Universal AI Agent Management System for automated GitHub workflows.

## Overview

Auto-Team is a configurable system that deploys AI agents to automatically handle GitHub issues, pull requests, and reviews. Instead of manually checking GitHub and working on tasks, this system continuously monitors for new work and automatically provisions containerized development environments with specialized AI agents.

## Features

- **Universal Configuration**: Single YAML file to define repository, agents, and settings
- **Dynamic Agent Scaling**: Support for any number of specialized agents
- **Template-Based Generation**: Docker Compose and entrypoint scripts generated from templates
- **Role-Based Agents**: Each agent can have specialized prompts and responsibilities
- **Continuous Monitoring**: Configurable intervals for checking GitHub activity
- **Docker Integration**: Containerized environments with volume mounting and networking

## Quick Start

### 1. Installation

**Quick Install (Recommended):**
```bash
# Install latest version (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/install.sh | bash
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
  url: "diazoxide/auto-team"
  main_branch: "main"

agents:
  - name: "developer"
    prompt: "You are a developer agent responsible for implementing features and fixing bugs."
    github_token_env: "DEVELOPER_GITHUB_TOKEN"
  - name: "reviewer"
    prompt: "You are a code reviewer focused on quality and best practices."
    github_token_env: "REVIEWER_GITHUB_TOKEN"

settings:
  docker_image: "node:18.17.1"
  docker_user: "developer"
  check_interval: 60
  team_name: "auto-team"
  install_deps: true
```

### 4. Set Environment Variables

```bash
export DEVELOPER_GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
export REVIEWER_GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
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
- `github_token_env`: Environment variable containing GitHub token
- `common_prompt`: Additional instructions for all agents

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

## Architecture

```
autoteam.yaml → Generator → compose.yaml + entrypoint.sh
                     ↓
              Docker Compose → Agent Containers
                     ↓
              GitHub Monitoring → Claude Code → Automated Tasks
```

### Generated Structure

```
./
├── autoteam.yaml          # Configuration
├── compose.yaml           # Generated Docker Compose
├── entrypoint.sh          # Generated startup script
├── agents/
│   ├── agent1/
│   │   ├── codebase/      # Repository clone
│   │   └── claude/        # Claude configuration
│   └── agent2/
│       ├── codebase/
│       └── claude/
└── shared/                # Shared configurations
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
go test ./templates
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
├── cmd/autoteam/           # CLI application
├── internal/
│   ├── config/            # Configuration parsing
│   ├── generator/         # Template generation
│   └── testutil/          # Test utilities
├── templates/             # Go templates
├── examples/              # Configuration examples
└── agents/                # Generated agent directories
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
./autoteam generate
cat compose.yaml
cat entrypoint.sh

# Test individual containers
docker-compose up agent-name

# View container logs
docker-compose logs agent-name
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.