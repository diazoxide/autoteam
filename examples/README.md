# AutoTeam Configuration Examples

This directory contains example configurations for different use cases and team setups.

## Available Examples

### 1. `basic-setup.yaml`
A minimal setup with two agents (developer and reviewer) showing the most common configuration.

**Use case**: Small projects with basic development workflow
**Agents**: 2 agents (developer, reviewer)
**Monitoring**: Every 60 seconds

### 2. `multi-role-team.yaml`
A comprehensive team setup with specialized roles for enterprise projects.

**Use case**: Large projects requiring specialized expertise
**Agents**: 5 agents (frontend-dev, backend-dev, devops-engineer, architect, qa-engineer)
**Monitoring**: Every 30 seconds (faster response)

### 3. `minimal-config.yaml`
The absolute minimum configuration required to run autoteam.

**Use case**: Quick testing or single-agent automation
**Agents**: 1 agent (general bot)
**Settings**: All defaults applied

### 4. `custom-docker.yaml`
Example showing how to customize Docker settings for specific technology stacks.

**Use case**: Python projects with specific runtime requirements
**Agents**: 2 agents (python-dev, data-scientist)
**Docker**: Custom Python image with specialized configuration

## How to Use

1. Choose the example that best matches your use case
2. Copy the configuration to your project as `autoteam.yaml`
3. Update the repository URL and replace placeholder GitHub tokens with real ones
4. Customize agent prompts and roles for your specific needs
5. Run `autoteam generate` to create the Docker Compose configuration

## Configuration Guidelines

### Repository Settings
- `url`: Your GitHub repository in "owner/repo" format
- `main_branch`: Main branch name (defaults to "main")

### Agent Configuration
- `name`: Unique identifier for the agent (used in container names)
- `prompt`: Primary role and responsibilities of the agent
- `github_token`: GitHub personal access token for this agent
- `common_prompt`: Additional instructions applied to all agents

### Settings
- `docker_image`: Docker image to use for agent containers
- `docker_user`: User account inside containers
- `check_interval`: How often (in seconds) to check for new work
- `team_name`: Project name used in container paths
- `install_deps`: Whether to install dependencies on container startup

## GitHub Tokens

Each agent needs a GitHub personal access token. You have two options:

**Option A: Direct in YAML**
```yaml
agents:
  - name: "developer"
    github_token: "ghp_your_developer_token_here"
  - name: "reviewer"
    github_token: "ghp_your_reviewer_token_here"
```

**Option B: Using .env file (recommended)**
Create a `.env` file:
```bash
DEVELOPER_TOKEN=ghp_your_actual_token
REVIEWER_TOKEN=ghp_your_actual_token
```

Then reference in YAML:
```yaml
agents:
  - name: "developer"
    github_token: "${DEVELOPER_TOKEN}"
  - name: "reviewer"
    github_token: "${REVIEWER_TOKEN}"
```

Both autoteam and entrypoint commands automatically load `.env` files if present.

## Scaling Considerations

- **Check Interval**: Lower values (30s) provide faster response but use more API calls
- **Agent Count**: More agents provide better coverage but require more resources
- **Docker Resources**: Consider memory and CPU limits for multiple containers

## Best Practices

1. **Agent Specialization**: Give each agent a specific role and expertise area
2. **Token Management**: Use separate tokens for better audit trails and permissions
3. **Prompt Engineering**: Be specific about agent responsibilities and constraints
4. **Resource Planning**: Monitor GitHub API rate limits with multiple agents
5. **Security**: Use minimal required permissions for each agent's GitHub token
