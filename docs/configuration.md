# Configuration Guide

## Overview

AutoTeam uses a single `autoteam.yaml` file to configure agents, platforms, and workflows. This file defines:

- **Workers** - AI agents with specific roles and capabilities
- **MCP Servers** - Platform connections (GitHub, Slack, databases, etc.)
- **Flows** - Workflow definitions with parallel execution
- **Settings** - Global configuration and environment

## Basic Configuration

### Initialize Configuration

```bash
autoteam init
```

This creates a sample `autoteam.yaml` with basic configuration.

### Minimal Configuration

```yaml
# autoteam.yaml
workers:
  - name: "AI Assistant"
    enabled: true
    prompt: |
      You are an AI assistant that handles tasks across multiple platforms.
      Use available MCP tools to interact with services and complete tasks.

settings:
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN

  flow:
    - name: process_tasks
      type: claude
      prompt: "Process available tasks using MCP tools"
```

## Workers Configuration

Workers are AI agents that execute tasks. Each worker can have different capabilities and access to platforms.

### Worker Structure

```yaml
workers:
  - name: "Worker Name"           # Display name
    enabled: true                 # Enable/disable worker
    prompt: |                     # System prompt for the agent
      Your role and instructions...
    settings:                     # Worker-specific settings
      service:                    # Docker service configuration
        image: "custom:image"     # Custom Docker image (optional)
        environment:              # Environment variables
          CUSTOM_VAR: ${VALUE}
        volumes:                  # Volume mounts
          - "./data:/app/data"
      mcp_servers:               # Worker-specific MCP servers
        custom:
          command: /path/to/server
          args: ["stdio"]
```

### Multiple Workers Example

```yaml
workers:
  - name: "GitHub Specialist"
    enabled: true
    prompt: |
      You specialize in GitHub operations. Handle PRs, issues, and code reviews.
      Use GitHub MCP tools for all GitHub interactions.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${GITHUB_SPECIALIST_TOKEN}

  - name: "Communication Manager"  
    enabled: true
    prompt: |
      You manage team communication across Slack and other platforms.
      Send updates, respond to mentions, and coordinate discussions.
    settings:
      service:
        environment:
          SLACK_TOKEN: ${SLACK_BOT_TOKEN}

  - name: "Data Processor"
    enabled: true
    prompt: |
      You handle database operations and data processing tasks.
      Query databases, process data, and generate reports.
    settings:
      service:
        environment:
          DATABASE_URL: ${POSTGRES_URL}
```

## MCP Server Configuration

MCP servers provide platform connectivity. Configure them at global or worker level.

### Global MCP Servers

```yaml
settings:
  mcp_servers:
    # GitHub integration
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN
        GITHUB_USER: $$GITHUB_USER

    # Slack integration
    slack:
      command: /opt/autoteam/bin/slack-mcp-server
      args: ["stdio"]
      env:
        SLACK_BOT_TOKEN: $$SLACK_BOT_TOKEN
        SLACK_SIGNING_SECRET: $$SLACK_SIGNING_SECRET

    # Database integration
    database:
      command: /opt/autoteam/bin/sqlite-mcp-server
      args: ["stdio"]
      env:
        DATABASE_URL: $$DATABASE_URL

    # Filesystem integration
    filesystem:
      command: /opt/autoteam/bin/filesystem-mcp-server
      args: ["stdio"]
      env:
        ALLOWED_PATHS: "/data,/tmp"
```

### Worker-Specific MCP Servers

```yaml
workers:
  - name: "Specialized Agent"
    settings:
      mcp_servers:
        # This worker gets additional MCP servers
        custom_api:
          command: /opt/autoteam/bin/custom-mcp-server
          args: ["stdio"]
          env:
            API_KEY: $$CUSTOM_API_KEY
```

## Environment Variables

AutoTeam supports secure environment variable management.

### .env File (Recommended)

Create a `.env` file in your project root:

```bash
# .env - Keep secure, never commit to version control

# GitHub tokens for different workers
GITHUB_SPECIALIST_TOKEN=ghp_your_github_token
GITHUB_TOKEN=ghp_main_github_token
GITHUB_USER=your-username

# Slack integration
SLACK_BOT_TOKEN=xoxb-your-slack-bot-token
SLACK_SIGNING_SECRET=your-slack-signing-secret

# Database connections
DATABASE_URL=postgresql://user:pass@localhost/db
POSTGRES_URL=postgresql://user:pass@localhost/postgres

# API keys
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=your-anthropic-key
CUSTOM_API_KEY=your-custom-api-key
```

### Variable Substitution

AutoTeam supports these variable formats:

- `${VAR_NAME}` - Standard environment variable
- `$$VAR_NAME` - Escaped for Docker Compose (recommended for MCP env)
- `${VAR_NAME:-default}` - With default value

### Worker-Level Environment

```yaml
workers:
  - name: "Secure Worker"
    settings:
      service:
        environment:
          # These variables are isolated to this worker
          GITHUB_TOKEN: ${WORKER_GITHUB_TOKEN}
          DATABASE_URL: ${WORKER_DATABASE_URL}
          LOG_LEVEL: debug
```

## Flow Configuration

Flows define the workflow steps and their dependencies. See [Flow System](flows.md) for detailed information.

### Simple Flow

```yaml
settings:
  flow:
    - name: collect_data
      type: gemini
      prompt: "Collect data from available sources"
    
    - name: process_data
      type: claude
      depends_on: [collect_data]
      prompt: "Process the collected data"
```

### Parallel Flow

```yaml
settings:
  flow:
    # These run in parallel (Level 0)
    - name: scan_github
      type: gemini
      prompt: "Scan GitHub for notifications"
    
    - name: scan_slack
      type: claude
      prompt: "Check Slack for mentions"
    
    - name: scan_database
      type: qwen
      prompt: "Query database for pending tasks"
    
    # This waits for all parallel tasks (Level 1)
    - name: process_all
      type: claude
      depends_on: [scan_github, scan_slack, scan_database]
      prompt: "Process all collected information"
```

## Global Settings

```yaml
settings:
  # Team configuration
  team_name: "my-ai-team"           # Docker Compose project name
  sleep_duration: 30                # Seconds between workflow cycles
  install_deps: true                # Auto-install dependencies
  
  # Default service configuration (applies to all workers)
  service:
    image: "node:18.17.1"          # Default Docker image
    user: "developer"               # Container user
    volumes:                        # Default volume mounts
      - "./shared:/app/shared"
    environment:                    # Default environment variables
      LOG_LEVEL: info
      ENVIRONMENT: production
```

## Configuration Validation

AutoTeam validates configuration on startup:

```bash
# Validate configuration
autoteam generate --dry-run

# Check for syntax errors
autoteam validate
```

Common validation errors:
- Missing required environment variables
- Invalid MCP server commands
- Circular flow dependencies
- Invalid worker configurations

## Configuration Examples

### Development Team Automation

```yaml
workers:
  - name: "Senior Developer"
    enabled: true
    prompt: |
      You are a senior developer focused on code quality and implementation.
      Review PRs, implement features, and maintain coding standards.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${SENIOR_DEV_GITHUB_TOKEN}

  - name: "DevOps Engineer"
    enabled: true
    prompt: |
      You handle infrastructure and deployment concerns.
      Manage CI/CD, monitor systems, and handle DevOps tasks.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${DEVOPS_GITHUB_TOKEN}
          DOCKER_TOKEN: ${DOCKER_REGISTRY_TOKEN}

settings:
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN
    
  flow:
    - name: scan_notifications
      type: gemini
      prompt: "Scan GitHub notifications and categorize by type"
    
    - name: handle_reviews
      type: claude
      depends_on: [scan_notifications]
      prompt: "Handle code reviews requiring senior developer attention"
    
    - name: handle_devops
      type: claude
      depends_on: [scan_notifications]
      prompt: "Handle infrastructure and deployment tasks"
```

## Next Steps

- [Flow System](flows.md) - Learn about workflow orchestration
- [MCP Integration](mcp.md) - Connect platforms and services
- [Examples](examples.md) - Real-world configuration examples