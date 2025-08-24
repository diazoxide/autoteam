# AutoTeam

Universal AI Agent Management System powered by the Model Context Protocol (MCP) for platform-agnostic automation.

## Overview

AutoTeam is a cutting-edge system that orchestrates intelligent AI agents to work autonomously across any platform that supports the Model Context Protocol (MCP). Using a dynamic flow-based execution system with parallel processing capabilities, AutoTeam agents can collect tasks from multiple platforms simultaneously and execute them with context-aware intelligence, whether it's GitHub repositories, Slack channels, databases, file systems, or any other MCP-enabled service.

## Features

### 🌐 Universal Platform Integration
- **MCP Protocol Support**: Works with ANY platform that provides MCP servers (GitHub, Slack, databases, filesystems, web services, etc.)
- **Platform-Agnostic Design**: Add new platforms by simply configuring MCP servers - no code changes required
- **Multi-AI Agent Support**: Deploy Claude, Gemini, and Qwen agents with specialized capabilities
- **Universal Task Processing**: Agents understand context across different platforms and services

### 🔄 Dynamic Flow System
- **Flow-Based Execution**: Define custom workflows with any number of steps and dependencies
- **Parallel Processing**: Independent steps execute concurrently for maximum efficiency
- **Conditional Logic**: Skip steps dynamically based on runtime conditions using templates
- **Dependency Resolution**: Smart execution ordering based on step dependencies
- **Data Sharing**: Seamless input/output data flow between workflow steps

### 🏗️ Advanced Architecture
- **Intelligent Agent Orchestration**: Multiple AI models working together with different strengths
- **Containerized Isolation**: Each agent runs in isolated Docker containers with dedicated configurations
- **Dynamic Scaling**: Deploy any number of specialized agents with custom roles and capabilities
- **Sleep-Based Monitoring**: Efficient resource management with configurable intervals between execution cycles

### ⚙️ Configuration & Management
- **Universal Configuration**: Single YAML file to define agents, platforms, and MCP server connections
- **Agent-Specific Settings**: Per-agent Docker images, volumes, MCP servers, and environment overrides
- **Smart Name Normalization**: Automatically handles agent names with spaces and special characters
- **Flexible MCP Configuration**: Global, agent-settings, and agent-level MCP server configurations with priority merging

### 📊 Monitoring & Logging
- **Streaming Task Logs**: Individual log files per task with timestamped, normalized filenames
- **Real-time Processing**: Live task execution monitoring and output capture
- **Structured Logging**: Comprehensive logging with contextual information using zap
- **Organized File Structure**: Clean directory structure in `.autoteam/` with agent-specific subdirectories

### 🔧 Developer Experience  
- **Cross-Platform Support**: macOS and Linux with universal installation script
- **Docker-First Design**: Container-native deployment with Docker Compose orchestration
- **Template-Based Generation**: Automated configuration generation from templates
- **Comprehensive Testing**: Full test coverage with CI/CD integration

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

Edit `autoteam.yaml` to define your AI agents and dynamic workflows:

```yaml
workers:
  - name: "Platform Specialist"
    enabled: true
    prompt: |
      You are a platform specialist that can work across multiple services.
      Use available MCP tools to handle tasks from various platforms.
      Always provide helpful, accurate responses and take appropriate actions.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${PLATFORM_SPECIALIST_GITHUB_TOKEN}
          SLACK_TOKEN: ${PLATFORM_SPECIALIST_SLACK_TOKEN}
          DATABASE_URL: ${DATABASE_URL}

settings:
  service:
    image: "node:18.17.1"
    user: "developer"
    volumes:
      - "./shared/claude:/root/.claude"
      - "./shared/claude.json:/root/.claude.json"
  sleep_duration: 30  # Sleep 30 seconds between workflow cycles
  team_name: "autoteam"
  install_deps: true
  
  # Dynamic Flow-Based Architecture
  flow:
    # Parallel data collection from multiple platforms
    - name: collect_github
      type: gemini
      args: ["--model", "gemini-2.5-flash"]
      prompt: |
        Collect GitHub notifications and issues using GitHub MCP.
        Output: List of GitHub tasks that need attention.
        
    - name: collect_slack
      type: gemini
      args: ["--model", "gemini-2.5-flash"]  
      prompt: |
        Collect Slack messages and mentions using Slack MCP.
        Output: List of Slack tasks requiring responses.
        
    - name: collect_database
      type: qwen
      prompt: |
        Check database for pending operations using Database MCP.
        Output: List of database tasks to process.
        
    # Task processing after all collection is complete
    - name: process_all_tasks
      type: claude
      depends_on: [collect_github, collect_slack, collect_database]
      prompt: |
        Process all collected tasks from previous steps.
        Use appropriate MCP tools for each platform.
        Prioritize based on urgency and context.
        
    # Optional notification step (runs only if tasks were processed)
    - name: send_summary
      type: claude
      depends_on: [process_all_tasks]
      skip_when: "{{- index .inputs 0 | trim | eq \"No tasks processed\" -}}"
      prompt: |
        Send a summary of completed tasks to the team.
        Use Slack MCP to post in the updates channel.
      
  # Universal MCP Server Configuration
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_PERSONAL_ACCESS_TOKEN: $$GITHUB_TOKEN
    slack:
      command: /opt/autoteam/bin/slack-mcp-server
      args: ["stdio"]
      env:
        SLACK_BOT_TOKEN: $$SLACK_TOKEN
    database:
      command: /opt/autoteam/bin/sqlite-mcp-server
      args: ["stdio"]
      env:
        DATABASE_URL: $$DATABASE_URL
```

### 4. Add Platform Credentials

Create a `.env` file in your project root to securely provide platform credentials:

```bash
# .env file - Platform credentials for MCP servers
PLATFORM_SPECIALIST_GITHUB_TOKEN=ghp_your_github_token
PLATFORM_SPECIALIST_SLACK_TOKEN=xoxb_your_slack_bot_token
DATABASE_URL=sqlite:///path/to/your/database.db
```

These credentials enable your agents to work with different platforms through MCP servers. Each MCP server handles secure authentication for its respective platform.

### 5. Deploy Your Team

```bash
# Generate Docker Compose configuration
autoteam generate

# Start the automated team
autoteam up

# Stop when needed
autoteam down
```

## Environment Variables and Secrets Management

AutoTeam provides a secure and flexible system for managing environment variables, secrets, and credentials. The recommended approach uses Docker Compose service-level environment variables, which provides proper isolation and security without copying sensitive data to generated files.

### 🔒 Security Best Practices

**Secure Configuration Layer:**
- Set environment variables in worker `service.environment` section
- Nothing is copied to `.autoteam/` directory - only references are used
- Runtime uses your local environment variables or `.env` file
- Credentials stay in your secure environment, not in generated files

### Configuration Approach

The most secure way to handle secrets is through the worker service layer in `autoteam.yaml`:

```yaml
workers:
  - name: Senior Developer
    enabled: true
    settings:
      service:
        environment:
          # GitHub credentials
          GITHUB_USER: ${SENIOR_DEVELOPER_GITHUB_USER}
          GITHUB_PERSONAL_ACCESS_TOKEN: ${SENIOR_DEVELOPER_GITHUB_TOKEN}
          
          # Database credentials
          DATABASE_URL: ${SENIOR_DEV_DATABASE_URL}
          
          # API keys
          OPENAI_API_KEY: ${SENIOR_DEV_OPENAI_KEY}
          SLACK_BOT_TOKEN: ${SENIOR_DEV_SLACK_TOKEN}

  - name: DevOps Engineer
    enabled: true
    settings:
      service:
        environment:
          # Different credentials for different roles
          GITHUB_TOKEN: ${DEVOPS_ENGINEER_GITHUB_TOKEN}
          AWS_ACCESS_KEY_ID: ${DEVOPS_AWS_ACCESS_KEY}
          AWS_SECRET_ACCESS_KEY: ${DEVOPS_AWS_SECRET_KEY}
          DOCKER_REGISTRY_TOKEN: ${DEVOPS_DOCKER_TOKEN}

# Global settings (applied to all workers unless overridden)
settings:
  service:
    environment:
      # Common environment variables
      TODO_DB_PATH: ${AUTOTEAM_WORKER_DIR}/todo.db
      LOG_LEVEL: info
      ENVIRONMENT: production
```

### Environment Variable Sources

AutoTeam supports multiple ways to provide environment variables:

#### 1. `.env` File (Recommended for Development)

Create a `.env` file in your project root:

```bash
# .env file - Keep this file secure and never commit to version control
# GitHub credentials for different workers
SENIOR_DEVELOPER_GITHUB_USER=your-github-username
SENIOR_DEVELOPER_GITHUB_TOKEN=ghp_your_personal_access_token_here

DEVOPS_ENGINEER_GITHUB_TOKEN=ghp_different_token_for_devops_role
SYSTEM_ARCHITECT_GITHUB_TOKEN=ghp_architect_review_token

# API Keys
SENIOR_DEV_OPENAI_KEY=sk-your-openai-api-key
SENIOR_DEV_SLACK_TOKEN=xoxb-your-slack-bot-token

# Database URLs
SENIOR_DEV_DATABASE_URL=postgresql://user:pass@localhost/senior_dev_db
SHARED_ANALYTICS_DB=sqlite:///opt/autoteam/shared/analytics.db

# AWS credentials for DevOps
DEVOPS_AWS_ACCESS_KEY=AKIA...
DEVOPS_AWS_SECRET_KEY=your-aws-secret-key

# Custom service credentials
JIRA_API_TOKEN=your-jira-token
CONFLUENCE_API_KEY=your-confluence-key
```

#### 2. System Environment Variables

Set variables in your shell profile or system:

```bash
# In ~/.bashrc, ~/.zshrc, or system environment
export SENIOR_DEVELOPER_GITHUB_TOKEN="ghp_your_token"
export DEVOPS_ENGINEER_GITHUB_TOKEN="ghp_different_token"
export DATABASE_URL="postgresql://user:pass@host/db"
```

#### 3. CI/CD Environment Variables

For production deployments, use your CI/CD platform's secret management:

```yaml
# GitHub Actions example
env:
  SENIOR_DEVELOPER_GITHUB_TOKEN: ${{ secrets.SENIOR_DEV_TOKEN }}
  DEVOPS_ENGINEER_GITHUB_TOKEN: ${{ secrets.DEVOPS_TOKEN }}
  DATABASE_URL: ${{ secrets.PRODUCTION_DB_URL }}
```

### Environment Variable Patterns

#### Role-Based Credentials
```yaml
workers:
  - name: Senior Developer
    settings:
      service:
        environment:
          GITHUB_USER: ${SENIOR_DEVELOPER_GITHUB_USER}
          GITHUB_TOKEN: ${SENIOR_DEVELOPER_GITHUB_TOKEN}
          # Role-specific permissions and access

  - name: Security Auditor  
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${SECURITY_AUDITOR_GITHUB_TOKEN}
          # Read-only or audit-specific permissions
```

#### Service-Specific Credentials
```yaml
settings:
  service:
    environment:
      # MCP server credentials
      GITHUB_MCP_TOKEN: ${GITHUB_API_TOKEN}
      SLACK_MCP_TOKEN: ${SLACK_BOT_TOKEN}
      DATABASE_MCP_URL: ${DATABASE_CONNECTION_STRING}
      
      # Application-specific settings
      LOG_LEVEL: ${APP_LOG_LEVEL:-info}
      RETRY_ATTEMPTS: ${MAX_RETRIES:-3}
```

#### Dynamic Placeholders
```yaml
settings:
  service:
    environment:
      # AutoTeam provides these placeholders automatically
      WORKER_NAME: ${AUTOTEAM_WORKER_NAME}
      WORKER_DIR: ${AUTOTEAM_WORKER_DIR}
      CONFIG_FILE: ${AUTOTEAM_WORKER_DIR}/config.json
      LOG_FILE: ${AUTOTEAM_WORKER_DIR}/worker.log
```

### Runtime Behavior

#### Security Features:
- **No Secret Copying**: Environment variables are referenced, not copied to `.autoteam/` files
- **Runtime Resolution**: Variables resolved when containers start, not during generation
- **Isolation**: Each worker gets its own isolated environment
- **No Persistence**: Secrets stay in memory, not written to disk

#### Generated Docker Compose:
```yaml
# Generated .autoteam/compose.yaml (example)
services:
  senior_developer:
    environment:
      # References resolved at runtime
      GITHUB_USER: ${SENIOR_DEVELOPER_GITHUB_USER}
      GITHUB_PERSONAL_ACCESS_TOKEN: ${SENIOR_DEVELOPER_GITHUB_TOKEN}
      # AutoTeam variables automatically added
      AUTOTEAM_WORKER_NAME: "Senior Developer"
      AUTOTEAM_WORKER_DIR: "/opt/autoteam/workers/senior_developer"
```

### Credential Management Examples

#### GitHub Integration
```yaml
workers:
  - name: Code Reviewer
    settings:
      service:
        environment:
          # GitHub API access
          GITHUB_TOKEN: ${CODE_REVIEWER_GITHUB_TOKEN}
          GITHUB_USER: ${CODE_REVIEWER_GITHUB_USER}
      
      # Corresponding .env entry:
      # CODE_REVIEWER_GITHUB_TOKEN=ghp_token_with_repo_and_read_permissions
      # CODE_REVIEWER_GITHUB_USER=reviewer-bot
```

#### Multi-Platform Setup
```yaml
workers:
  - name: Integration Specialist
    settings:
      service:
        environment:
          # Multiple platform credentials
          GITHUB_TOKEN: ${INTEGRATION_GITHUB_TOKEN}
          SLACK_BOT_TOKEN: ${INTEGRATION_SLACK_TOKEN}
          JIRA_API_TOKEN: ${INTEGRATION_JIRA_TOKEN}
          DATABASE_URL: ${INTEGRATION_DATABASE_URL}
          REDIS_URL: ${INTEGRATION_REDIS_URL}
```

#### Development vs Production
```yaml
# Development configuration
settings:
  service:
    environment:
      DATABASE_URL: ${DEV_DATABASE_URL:-sqlite:///tmp/dev.db}
      LOG_LEVEL: ${LOG_LEVEL:-debug}
      GITHUB_API_URL: ${GITHUB_API_URL:-https://api.github.com}

# Production overrides using different environment variables
# PROD_DATABASE_URL=postgresql://prod-server/app_db
# LOG_LEVEL=warn
# GITHUB_API_URL=https://github.company.com/api/v3
```

### Security Recommendations

#### Credential Separation
- **Use different tokens** for different workers/roles
- **Minimal permissions** - only grant required access levels
- **Regular rotation** - rotate tokens and keys periodically
- **Audit access** - monitor which workers use which credentials

#### Environment File Security
```bash
# Secure your .env file
chmod 600 .env

# Add to .gitignore
echo ".env" >> .gitignore
echo ".env.*" >> .gitignore

# Example .env.template for team sharing (without actual secrets)
cp .env .env.template
# Remove actual values, keep variable names as examples
```

#### Production Deployment
- **Never commit** `.env` files to version control
- **Use secret management** systems (AWS Secrets Manager, Azure Key Vault, etc.)
- **Environment-specific** configurations
- **Container security** scanning and runtime protection

### Common Patterns

#### Hook Initialization
```yaml
settings:
  hooks:
    on_init:
      - command: "/bin/sh"
        args: ["-c", "echo https://$$GITHUB_USER:$$GITHUB_PERSONAL_ACCESS_TOKEN@github.com > ~/.git-credentials"]
        description: "Initialize Git credentials securely"
```

#### MCP Server Configuration
```yaml
mcp_servers:
  github:
    command: /opt/autoteam/custom/mcp/github/bin/github-mcp-server
    args: ["stdio"]
    env:
      # MCP server uses same credentials as worker
      GITHUB_TOKEN: $$GITHUB_PERSONAL_ACCESS_TOKEN

  database:
    command: /opt/autoteam/bin/database-mcp-server
    args: ["stdio"] 
    env:
      DATABASE_URL: $$DATABASE_URL
```

### Troubleshooting Environment Variables

#### Common Issues:
1. **Missing Variables**: Ensure all referenced variables are defined
2. **Permission Errors**: Check token permissions for required operations  
3. **Syntax Errors**: Use `${VAR_NAME}` syntax in autoteam.yaml, `$$VAR_NAME` in shell commands
4. **Variable Priority**: Worker-level overrides global-level environment variables

#### Debug Commands:
```bash
# Check environment variable resolution
docker-compose -f .autoteam/compose.yaml config

# Verify specific worker environment
docker-compose -f .autoteam/compose.yaml exec worker_name env | grep GITHUB

# Test credential access
docker-compose -f .autoteam/compose.yaml exec worker_name curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

This approach ensures your sensitive credentials remain secure while providing flexible configuration options for different deployment scenarios.

## Dynamic Flow System

AutoTeam's powerful flow-based execution system allows you to create sophisticated workflows that can process tasks from multiple platforms with intelligent dependency resolution and parallel execution.

### How Flows Work

Flows consist of **steps** that are organized into **dependency levels** based on their relationships:

- **Level 0**: Steps with no dependencies (execute in parallel immediately)
- **Level 1**: Steps that depend on Level 0 steps (execute in parallel after Level 0 completes)
- **Level N**: Steps that depend on Level N-1 steps

This creates an efficient execution model where independent operations run simultaneously while maintaining proper ordering for dependent tasks.

### Flow Step Properties

Each flow step supports the following configuration:

```yaml
flow:
  - name: step_name           # Unique identifier for the step
    type: claude|gemini|qwen  # AI agent type to execute this step
    args: ["--model", "..."]  # Optional command-line arguments
    prompt: |                 # Step-specific instructions
      Your task description here...
    depends_on: [step1, step2] # Optional: steps that must complete first
    skip_when: "{{ condition }}" # Optional: template condition to skip step
    env:                      # Optional: step-specific environment variables
      CUSTOM_VAR: value
```

### Execution Examples

#### Simple Sequential Flow
```yaml
flow:
  - name: collect
    type: gemini
    prompt: "Collect all notifications..."
  - name: process
    type: claude
    depends_on: [collect]
    prompt: "Process the collected notifications..."
```

#### Parallel Collection with Unified Processing
```yaml
flow:
  # These three steps run simultaneously (Level 0)
  - name: github_tasks
    type: gemini
    prompt: "Collect GitHub notifications..."
  - name: slack_tasks
    type: gemini
    prompt: "Collect Slack messages..."
  - name: jira_tasks
    type: qwen
    prompt: "Collect Jira tickets..."
    
  # This step waits for all collectors to complete (Level 1)
  - name: process_all
    type: claude
    depends_on: [github_tasks, slack_tasks, jira_tasks]
    prompt: "Process all collected tasks..."
```

#### Conditional Execution
```yaml
flow:
  - name: check_notifications
    type: gemini
    prompt: "Count unread notifications. Output: just the number."
  - name: process_notifications
    type: claude
    depends_on: [check_notifications]
    skip_when: "{{- index .inputs 0 | trim | eq \"0\" -}}"
    prompt: "Process the notifications..."
  - name: send_summary
    type: claude
    depends_on: [process_notifications]
    prompt: "Send completion summary..."
```

### Data Flow Between Steps

Steps can share data through inputs and outputs:

- **Outputs**: Each step's output (stdout) is automatically captured
- **Inputs**: Dependent steps receive outputs from their dependencies via `.inputs` array
- **Templates**: Use Go template syntax in `skip_when` conditions to access input data

### Conditional Step Execution

The `skip_when` field allows dynamic step skipping using Go template syntax with Sprig functions:

```yaml
# Skip if no notifications found
skip_when: "{{- index .inputs 0 | trim | eq \"0\" -}}"

# Skip if previous step output contains "SKIP"
skip_when: "{{- index .inputs 0 | contains \"SKIP\" -}}"

# Skip based on multiple conditions
skip_when: "{{- and (index .inputs 0 | trim | ne \"\") (index .inputs 0 | contains \"error\") -}}"
```

### Performance Benefits

- **Parallel Execution**: Independent steps run simultaneously
- **Efficient Resource Usage**: Only run necessary steps
- **Smart Dependency Resolution**: Automatic execution ordering
- **Conditional Logic**: Skip unnecessary work dynamically

### Migration from Two-Layer System

If you're upgrading from the old two-layer architecture:

**Old Format:**
```yaml
collector_agent:
  type: gemini
  prompt: "Collect tasks..."
execution_agent:
  type: claude
  prompt: "Execute tasks..."
```

**New Format:**
```yaml
flow:
  - name: collector
    type: gemini
    prompt: "Collect tasks..."
  - name: executor
    type: claude
    depends_on: [collector]
    prompt: "Execute tasks..."
```

## Configuration

### Agent Configuration

**Agent Properties:**
- `name`: Unique identifier for the agent (supports names with spaces and special characters)
- `prompt`: Primary role and responsibilities (supports multi-line YAML)
- `enabled`: Enable/disable agent without removing configuration (optional, defaults to true)
- `settings`: Agent-specific overrides for global settings (optional)
  - `service`: Docker service configuration overrides
    - `image`: Custom Docker image for this agent
    - `user`: Custom user for this agent  
    - `volumes`: Additional volume mounts
    - `environment`: Environment variables (including GITHUB_TOKEN)

**Example Agent Configuration:**
```yaml
workers:
  - name: "Senior Developer"
    enabled: true
    prompt: |
      You are a senior developer responsible for implementing features and fixes.
      Focus on code quality, testing, and comprehensive documentation.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${DEVELOPER_GITHUB_TOKEN}
        image: "golang:1.21"
        volumes:
          - "./custom-tools:/opt/tools:ro"
```

### Global Settings

**Service Configuration:**
- `service`: Docker service defaults applied to all agents
  - `image`: Default Docker image for agent containers
  - `user`: Default user account inside containers
  - `volumes`: Default volume mounts applied to all agents
  - `environment`: Default environment variables for all agents

**Dynamic Flow Configuration:**
- `flow`: Array of flow steps defining workflow execution
  - `name`: Unique step identifier
  - `type`: Agent type (claude, gemini, qwen)
  - `args`: Command line arguments (optional)
  - `prompt`: Step-specific instructions
  - `depends_on`: Dependencies on other steps (optional)
  - `skip_when`: Conditional execution template (optional)
  - `env`: Step-specific environment variables (optional)

**System Settings:**
- `sleep_duration`: Sleep duration in seconds between flow execution cycles
- `team_name`: Project name used in paths and Docker Compose project name
- `install_deps`: Install dependencies on container startup
- `common_prompt`: Common instructions shared by all agents (optional)
- `mcp_servers`: Global MCP (Model Context Protocol) servers for all agents

### Configuration Variable Replacement

AutoTeam supports placeholder variables in environment variable values that are automatically replaced during Docker Compose generation. This provides clean, consistent configuration management with runtime value resolution.

**Supported Placeholder Variables:**
- `${AUTOTEAM_AGENT_NAME}` → actual agent name (e.g., "Senior Developer")
- `${AUTOTEAM_AGENT_DIR}` → agent directory path (e.g., "/opt/autoteam/agents/senior_developer")
- `${AUTOTEAM_AGENT_NORMALIZED_NAME}` → normalized agent name (e.g., "senior_developer")

**Usage Example:**
```yaml
settings:
  service:
    environment:
      TODO_DB_PATH: ${AUTOTEAM_AGENT_DIR}/todo.db
      LOG_FILE: ${AUTOTEAM_AGENT_DIR}/logs/${AUTOTEAM_AGENT_NORMALIZED_NAME}.log
      WORKSPACE_NAME: ${AUTOTEAM_AGENT_NAME}
```

**Generated Result in compose.yaml:**
```yaml
environment:
  TODO_DB_PATH: /opt/autoteam/agents/senior_developer/todo.db
  LOG_FILE: /opt/autoteam/agents/senior_developer/logs/senior_developer.log
  WORKSPACE_NAME: Senior Developer
  AUTOTEAM_AGENT_NAME: Senior Developer
  AUTOTEAM_AGENT_DIR: /opt/autoteam/agents/senior_developer
  AUTOTEAM_AGENT_NORMALIZED_NAME: senior_developer
```

**Benefits:**
- Dynamic agent-specific configuration without hardcoding paths
- Consistent environment variable naming with `AUTOTEAM_` prefix
- Clean configuration management with compile-time variable resolution
- Easy to reference agent-specific directories and names in MCP servers and tools

### MCP Server Configuration

AutoTeam supports Model Context Protocol (MCP) servers to enhance agent capabilities with additional context and tools. MCP servers can be configured at three levels with priority-based merging:

1. **Global Level** (`settings.mcp_servers`): Applied to all agents
2. **Agent Settings Level** (`agent.settings.mcp_servers`): Applied to specific agent, overrides global
3. **Agent Level** (`agent.mcp_servers`): Highest priority, overrides both global and agent settings

**Configuration Example:**
```yaml
# Global MCP servers (applied to all agents)
settings:
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_PERSONAL_ACCESS_TOKEN: $$GITHUB_TOKEN
    memory:
      command: "npx"
      args: ["-y", "mcp-memory-service"]

workers:
  - name: "developer"
    prompt: |
      You are a developer agent responsible for implementing features.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${DEVELOPER_GITHUB_TOKEN}
      # Agent settings-level MCP servers (medium priority)
      mcp_servers:
        filesystem:
          command: "npx"
          args: ["-y", "mcp-filesystem-server"]
    # Agent-level MCP servers (highest priority)
    mcp_servers:
      sqlite:
        command: "npx"
        args: ["-y", "mcp-sqlite-server"]
        env:
          DATABASE_URL: "sqlite:///opt/autoteam/agents/developer/collector/data.db"
```

**MCP Server Properties:**
- `command`: Executable command to run the MCP server
- `args`: Command line arguments (optional)
- `env`: Environment variables for the MCP server (optional)

**Popular MCP Servers:**
- **GitHub**: `@github/github-mcp-server` - Enhanced GitHub operations and context
- **Memory**: `mcp-memory-service` - Persistent conversation history
- **SQLite**: `mcp-sqlite-server` - Database persistence and queries
- **Filesystem**: `mcp-filesystem-server` - File system operations with context
- **Web**: `mcp-web-server` - Web browsing and content fetching

**Merging Behavior:**
- Same-named servers at higher priority levels completely replace lower priority ones
- Different-named servers from all levels are combined
- Agent-level > Agent Settings-level > Global-level priority

### Intelligent Notification Processing

AutoTeam uses a sophisticated notification processing system that handles GitHub events intelligently:

**Notification Types Supported:**
- **Review Requests**: Professional code review workflow with quality guidelines
- **Assigned Issues**: Smart intent detection for consultation vs implementation requests
- **Assigned PRs**: Complete pull request handling workflow
- **Mentions**: Context analysis to determine if consultation or action is needed
- **Failed Workflows**: CI/CD failure investigation and automatic fixing
- **Unread Comments**: Thread-appropriate responses based on comment type
- **Generic Notifications**: Fallback handling for unknown notification types

**Key Features:**
- **Type-Specific Prompts**: Each notification type gets specialized guidance
- **Intent Recognition**: Distinguishes between "What do you think?" (consultation) vs "Please implement" (action)
- **AI-Driven Validation**: Uses GitHub CLI commands to check if notifications are still current
- **Automatic Read Marking**: Prevents duplicate processing by marking notifications as read
- **Context-Aware Responses**: Agents respond appropriately based on notification type and user intent

This prevents the common problem where agents over-implement when users are simply asking for advice or opinions.

### Intent Recognition Examples

The system intelligently distinguishes between different types of requests:

**Consultation Requests (Comment Only):**
- "What do you think about this approach?"
- "How should we structure the database?"
- "Any thoughts on the best way to implement OAuth?"
- "What's your opinion on using GraphQL here?"

**Implementation Requests (Create PRs/Code):**
- "Please implement user authentication"
- "Fix the memory leak in the parser"
- "Add support for file uploads"
- "Create a REST API endpoint for users"

**Smart Response Behavior:**
- **For Consultation**: Agents provide thoughtful analysis and recommendations via comments
- **For Implementation**: Agents write code, create pull requests, and implement solutions
- **For Reviews**: Agents examine code thoroughly and provide professional feedback
- **For CI Failures**: Agents investigate logs, identify issues, and create fixes

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
autoteam workers    # List all workers and their states
```

All generated files are organized in the `.autoteam/` directory for better project organization.

### Agent Name Normalization

AutoTeam automatically normalizes agent names for Docker Compose services and directory paths while preserving the original names for identification:

```yaml
workers:
  - name: "Senior Developer"      # Original name (used in environment variables)
    prompt: "You are a senior developer"
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${SENIOR_DEVELOPER_GITHUB_TOKEN}
  - name: "API Agent #1"          # Original name with special characters
    prompt: "You are an API specialist"
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${API_AGENT_GITHUB_TOKEN}
```

This generates Docker Compose services with normalized names:
- `Senior Developer` → `senior_developer` (service name)
- `API Agent #1` → `api_agent_1` (service name)

Directory structure uses normalized names with flow-based architecture:
```
.autoteam/agents/
├── senior_developer/
│   ├── collect_tasks/      # Flow step: Task collection
│   │   ├── .gemini/        # Gemini CLI configuration
│   │   └── data/           # Step-specific data
│   ├── process_tasks/      # Flow step: Task processing
│   │   ├── mcp.json        # Claude MCP configuration
│   │   └── data/           # Step-specific data
│   ├── send_updates/       # Flow step: Status updates
│   │   ├── mcp.json        # Claude MCP configuration
│   │   └── data/           # Step-specific data
│   └── shared/             # Shared agent resources
└── api_agent_1/
    ├── analyze_apis/       # Flow step: API analysis
    │   ├── .qwen/          # Qwen Code configuration
    │   └── data/           # Step-specific data
    ├── implement_features/ # Flow step: Feature implementation
    │   ├── mcp.json        # Claude MCP configuration
    │   └── data/           # Step-specific data
    └── shared/             # Shared agent resources
```

### Disabling Agents

You can temporarily disable agents without removing their configuration:

```yaml
workers:
  - name: "developer"
    prompt: "You are a developer agent"
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${DEVELOPER_GITHUB_TOKEN}
    enabled: true  # Agent is active (default)
  
  - name: "reviewer"
    prompt: "You are a code reviewer"
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${REVIEWER_GITHUB_TOKEN}
    enabled: false  # Agent is disabled - won't be deployed
```

Benefits:
- Keep agent configurations for future use
- Temporarily reduce resource usage
- Test with specific agent combinations
- Preserve tokens and settings when not actively needed

Use `autoteam workers` to list all workers and their states.

## Architecture

```mermaid
graph TB
    Config[autoteam.yaml] --> Generator[Configuration Generator]
    Generator --> Compose[.autoteam/compose.yaml]
    Generator --> Binaries[.autoteam/bin/]
    
    Compose --> Docker[Docker Compose]
    Docker --> AgentContainers[Agent Containers]
    
    subgraph "Dynamic Flow Architecture"
        AgentContainers --> FlowSteps[Flow Steps<br/>Parallel Execution]
        FlowSteps --> DependencyLevels[Dependency Levels<br/>Sequential Ordering]
    end
    
    subgraph "Universal MCP Integration"
        Collector --> MCP1[GitHub MCP]
        Collector --> MCP2[Slack MCP]
        Collector --> MCP3[Database MCP]
        Collector --> MCP4[Filesystem MCP]
        Collector --> MCP5[Web MCP]
        Collector --> MCPn[Any MCP Server...]
    end
    
    subgraph "Platform Ecosystem"
        MCP1 --> GitHub[GitHub APIs]
        MCP2 --> Slack[Slack APIs]
        MCP3 --> DB[(Databases)]
        MCP4 --> FS[File Systems]
        MCP5 --> Web[Web Services]
        MCPn --> Platform[Any Platform]
    end
    
    DependencyLevels --> FlowExecution[Flow Execution]
    FlowExecution --> Actions[Platform Actions]
    Actions --> GitHub
    Actions --> Slack  
    Actions --> DB
    Actions --> FS
    Actions --> Web
    Actions --> Platform
```

### Universal MCP-Based Design

AutoTeam's architecture is built around the Model Context Protocol (MCP), enabling seamless integration with any platform:

- **Flow-Based Processing**: Dynamic workflow execution with parallel and sequential steps
- **Dependency Resolution**: Smart execution ordering based on step dependencies
- **Platform Abstraction**: MCP servers provide standardized interfaces to any service
- **Agent Orchestration**: Container-based agents with per-step MCP configurations

### Multi-Platform Agent Structure

Each agent maintains separate working directories with flow-based architecture and platform-specific configurations:

```
./
├── autoteam.yaml          # Universal platform configuration
└── .autoteam/             # Generated files directory
    ├── compose.yaml       # Docker Compose configuration
    ├── agents/            # Agent-specific directories
    │   ├── platform_specialist/
    │   │   ├── collect_github/     # Flow step: GitHub collection
    │   │   │   ├── .gemini/        # Gemini CLI config
    │   │   │   ├── data/           # Step-specific data
    │   │   │   └── logs/           # Step execution logs
    │   │   ├── collect_slack/      # Flow step: Slack collection
    │   │   │   ├── .gemini/        # Gemini CLI config
    │   │   │   ├── data/           # Step-specific data
    │   │   │   └── logs/           # Step execution logs
    │   │   ├── process_all_tasks/  # Flow step: Unified processing
    │   │   │   ├── .mcp.json       # Claude MCP configuration
    │   │   │   ├── data/           # Processing workspace
    │   │   │   └── logs/           # Step execution logs
    │   │   ├── send_summary/       # Flow step: Summary notifications
    │   │   │   ├── .mcp.json       # Claude MCP configuration
    │   │   │   ├── data/           # Summary workspace
    │   │   │   └── logs/           # Step execution logs
    │   │   └── shared/             # Shared agent resources
    │   └── content_manager/
    │       ├── analyze_content/    # Flow step: Content analysis
    │       │   ├── .qwen/          # Qwen configuration
    │       │   ├── data/           # Analysis workspace
    │       │   └── logs/           # Step execution logs
    │       ├── generate_content/   # Flow step: Content generation
    │       │   ├── .mcp.json       # Claude MCP configuration
    │       │   ├── data/           # Generation workspace
    │       │   └── logs/           # Step execution logs
    │       └── shared/             # Shared agent resources
    ├── bin/               # Unified binary directory
    │   ├── autoteam-entrypoint-*
    │   ├── entrypoint.sh
    │   ├── github-mcp-server      # GitHub MCP server
    │   ├── slack-mcp-server       # Slack MCP server
    │   ├── sqlite-mcp-server      # Database MCP server
    │   └── custom-mcp-servers/    # Custom MCP implementations
    └── shared/            # Shared configurations
        ├── claude/
        ├── claude.json
        └── gemini/
```

### Platform Integration Patterns

- **MCP Server Discovery**: Automatically detect and configure available MCP servers per step
- **Parallel Flow Execution**: Execute independent steps simultaneously across platforms
- **Context-Aware Processing**: Steps share data and understand cross-platform context
- **Dynamic Platform Addition**: Add new platforms by installing MCP servers - no code changes needed

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
│   └── worker/            # Go binary worker for agents
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

- **Platform Credentials**: Use separate tokens/credentials for each agent and platform
- **Minimal Permissions**: Configure minimal required permissions for all platform tokens
- **Credential Validation**: Ensure credentials match the intended service accounts
- **Regular Rotation**: Regularly rotate access tokens and API keys across all platforms
- **Rate Limit Monitoring**: Monitor API usage across all integrated platforms
- **Container Security**: Review generated Docker configurations and agent isolation
- **Environment Variables**: Use `.env` files for sensitive credentials - never commit to version control
- **MCP Server Security**: Ensure MCP servers run with appropriate permissions and network access controls

## Troubleshooting

### Common Issues

1. **Platform Authentication**: Ensure tokens are properly set for all configured platforms (GitHub, Slack, databases, etc.)
2. **MCP Server Connectivity**: Verify MCP servers are running and accessible to agents
3. **Docker Issues**: Verify Docker and Docker Compose are installed and running
4. **API Rate Limits**: Monitor rate limits across all integrated platforms
5. **Port Conflicts**: Check for container port conflicts
6. **Permission Issues**: Ensure proper file permissions for generated scripts and MCP configurations
7. **Agent Communication**: Verify agents can communicate with their assigned MCP servers
8. **Multi-Platform Access**: Ensure credentials have access to all configured services and platforms
9. **MCP Protocol Issues**: Check MCP server logs for protocol-specific errors

### Debug Mode

```bash
# Check generated files
autoteam generate
cat .autoteam/compose.yaml
ls .autoteam/bin/

# Verify MCP server connectivity
docker-compose -f .autoteam/compose.yaml logs | grep -E "(MCP|mcp)"

# Check individual agent working directories and MCP configurations
ls .autoteam/agents/agent-name/collector/data/
ls .autoteam/agents/agent-name/executor/data/
cat .autoteam/agents/agent-name/executor/.mcp.json

# Test individual containers
docker-compose -f .autoteam/compose.yaml up agent-name

# View platform task processing logs
docker-compose -f .autoteam/compose.yaml logs agent-name | grep -E "(Task|Processing|Platform|MCP)"

# Monitor real-time multi-platform activity
docker-compose -f .autoteam/compose.yaml logs -f --tail=50

# View streaming task logs for individual tasks
ls .autoteam/agents/agent-name/executor/logs/
cat .autoteam/agents/agent-name/executor/logs/20250814-121735-process_slack_message.log
cat .autoteam/agents/agent-name/executor/logs/20250814-122304-update_database_record.log
```

## Streaming Task Logs

AutoTeam provides detailed task-specific logging for the executor layer with individual log files for each task:

### Log File Structure

Each task execution creates a separate log file with the format:
```
.autoteam/agents/{agent-name}/executor/logs/{YYYYMMDD-HHMMSS}-{normalized_task_name}.log
```

**Examples:**
```
20250814-121735-handle_bug_fix_in_parser.log
20250814-122015-review_pull_request_for_authentication.log
20250814-122304-implement_user_dashboard_feature.log
```

### Log Features

- **Timestamped Files**: Each log file starts with execution timestamp (YYYYMMDD-HHMMSS)
- **Normalized Names**: Task descriptions converted to safe, lowercase filenames
- **Real-time Streaming**: Logs written immediately during execution (not batched at completion)
- **Complete Output**: Captures both stdout and stderr from agent execution
- **Task Context**: Each log includes the original task description and execution metadata
- **Backward Compatible**: Maintains existing `output.txt` functionality

### Viewing Task Logs

```bash
# List all task logs for an agent
ls .autoteam/agents/senior_developer/executor/logs/

# View specific task execution
cat .autoteam/agents/senior_developer/executor/logs/20250814-121735-fix_memory_leak_issue.log

# Monitor latest task logs in real-time
tail -f .autoteam/agents/senior_developer/executor/logs/*.log

# Find logs for specific task types
ls .autoteam/agents/senior_developer/executor/logs/*review*.log
ls .autoteam/agents/senior_developer/executor/logs/*implement*.log
```

### Log Content Format

Each log file contains:
- **Header**: Task description and execution timestamp
- **Agent Stdout**: Complete output from the AI agent
- **Agent Stderr**: Any error messages or warnings
- **Completion**: Task completion timestamp

This detailed logging makes it easy to debug individual task executions, track agent behavior over time, and understand how specific notifications were processed.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
