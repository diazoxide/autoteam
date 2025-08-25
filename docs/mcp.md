# MCP Integration

## Overview

AutoTeam uses the Model Context Protocol (MCP) to provide universal platform integration. Instead of building custom connectors for each service, AutoTeam acts as an MCP hub that connects AI agents with any MCP-enabled platform.

## What is MCP?

The Model Context Protocol is a standardized way for AI agents to interact with external services and tools. MCP servers provide a uniform interface for platforms like GitHub, Slack, databases, file systems, and custom APIs.

```mermaid
graph TB
    subgraph "AutoTeam MCP Hub"
        ATC[AutoTeam Core]
        MCH[MCP Client Handler]
    end
    
    subgraph "AI Agents"
        Claude[Claude Agent]
        Gemini[Gemini Agent]
        Qwen[Qwen Agent]
    end
    
    subgraph "MCP Server Layer"
        GMCP[GitHub MCP Server]
        SMCP[Slack MCP Server]
        DMCP[Database MCP Server]
        FMCP[Filesystem MCP Server]
        CMCP[Custom MCP Server]
    end
    
    subgraph "External Services"
        GitHub_API[GitHub API]
        Slack_API[Slack API]
        Database[(Database)]
        FileSystem[File System]
        Custom_API[Custom APIs]
    end
    
    ATC --> MCH
    MCH --> Claude
    MCH --> Gemini
    MCH --> Qwen
    
    Claude --> GMCP
    Claude --> SMCP
    Gemini --> DMCP
    Qwen --> FMCP
    Claude --> CMCP
    
    GMCP --> GitHub_API
    SMCP --> Slack_API
    DMCP --> Database
    FMCP --> FileSystem
    CMCP --> Custom_API
```

## MCP Server Configuration

### Global MCP Servers

Configure MCP servers globally in `autoteam.yaml`:

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
        ALLOWED_PATHS: "/data,/tmp,/app/workspace"
```

### Worker-Specific MCP Servers

Workers can have their own MCP servers in addition to global ones:

```yaml
workers:
  - name: "Database Specialist"
    enabled: true
    prompt: "You specialize in database operations and analytics"
    settings:
      mcp_servers:
        # Additional MCP server for this worker
        analytics_db:
          command: /opt/autoteam/bin/postgresql-mcp-server
          args: ["stdio"]
          env:
            DATABASE_URL: $$ANALYTICS_DATABASE_URL
        
        # Custom reporting MCP server
        reporting:
          command: /opt/autoteam/bin/reporting-mcp-server
          args: ["stdio"]
          env:
            REPORT_API_KEY: $$REPORT_API_KEY
```

### MCP Server Priority

MCP servers are merged with the following priority (highest to lowest):

1. **Worker-level MCP servers** - Specific to individual workers
2. **Worker settings MCP servers** - In worker.settings.mcp_servers
3. **Global MCP servers** - In settings.mcp_servers

## Available MCP Servers

### GitHub MCP Server

Provides comprehensive GitHub API access:

```yaml
github:
  command: /opt/autoteam/bin/github-mcp-server
  args: ["stdio"]
  env:
    GITHUB_TOKEN: $$GITHUB_TOKEN              # Personal access token
    GITHUB_USER: $$GITHUB_USER                # GitHub username
    GITHUB_API_BASE_URL: $$GITHUB_API_URL     # For GitHub Enterprise (optional)
```

**Capabilities:**
- Repository management
- Issue and PR operations
- Code review and comments
- Notification handling
- Repository content access
- Search and filtering

**Usage in Prompts:**
```yaml
prompt: |
  Use GitHub MCP tools to:
  1. List pending pull requests requiring review
  2. Check for issues assigned to the team
  3. Review and comment on code changes
  4. Update issue labels and status
```

### Slack MCP Server

Enables Slack workspace integration:

```yaml
slack:
  command: /opt/autoteam/bin/slack-mcp-server
  args: ["stdio"]
  env:
    SLACK_BOT_TOKEN: $$SLACK_BOT_TOKEN         # Bot user OAuth token
    SLACK_SIGNING_SECRET: $$SLACK_SIGNING_SECRET # App signing secret
    SLACK_APP_TOKEN: $$SLACK_APP_TOKEN         # App-level token (optional)
```

**Capabilities:**
- Send and receive messages
- Channel and DM management
- User and workspace information
- File uploads and sharing
- Thread management
- Reaction handling

**Usage in Prompts:**
```yaml
prompt: |
  Use Slack MCP tools to:
  1. Check for mentions and direct messages
  2. Post status updates to team channels
  3. Share files and documents
  4. Respond to team communications
```

### Database MCP Servers

#### SQLite MCP Server

```yaml
sqlite:
  command: /opt/autoteam/bin/sqlite-mcp-server
  args: ["stdio"]
  env:
    DATABASE_URL: $$SQLITE_DATABASE_URL        # sqlite:///path/to/db.sqlite
```

#### PostgreSQL MCP Server

```yaml
postgresql:
  command: /opt/autoteam/bin/postgresql-mcp-server
  args: ["stdio"]
  env:
    DATABASE_URL: $$POSTGRES_DATABASE_URL      # postgresql://user:pass@host/db
    CONNECTION_POOL_SIZE: "10"                 # Optional
    CONNECTION_TIMEOUT: "30"                   # Optional (seconds)
```

**Database Capabilities:**
- SQL query execution
- Schema introspection
- Transaction management
- Data import/export
- Table and index operations

### Filesystem MCP Server

Provides secure file system access:

```yaml
filesystem:
  command: /opt/autoteam/bin/filesystem-mcp-server
  args: ["stdio"]
  env:
    ALLOWED_PATHS: "/data,/tmp,/app/workspace" # Comma-separated allowed paths
    MAX_FILE_SIZE: "10485760"                  # 10MB limit (optional)
    READONLY_MODE: "false"                     # Enable/disable writes
```

**Capabilities:**
- File and directory operations
- Content reading and writing
- File metadata and permissions
- Path validation and security
- Directory traversal

### Custom MCP Servers

Create custom MCP servers for proprietary systems:

```yaml
custom_api:
  command: /opt/autoteam/bin/custom-mcp-server
  args: ["stdio"]
  env:
    API_ENDPOINT: $$CUSTOM_API_ENDPOINT
    API_KEY: $$CUSTOM_API_KEY
    TIMEOUT: "30"
```

## MCP Server Development

### Creating Custom MCP Servers

Build MCP servers for your specific platforms:

```go
// Example MCP server structure
package main

import (
    "github.com/your-org/mcp-sdk-go"
)

func main() {
    server := mcp.NewServer()
    
    // Register tools
    server.RegisterTool("custom_action", customActionHandler)
    server.RegisterTool("fetch_data", fetchDataHandler)
    
    // Register resources
    server.RegisterResource("custom_resource", customResourceHandler)
    
    // Start server
    server.Start()
}

func customActionHandler(params map[string]interface{}) (interface{}, error) {
    // Implement custom action logic
    return result, nil
}
```

### MCP Server Requirements

MCP servers must:
- Support stdio communication protocol
- Implement standard MCP message format
- Handle tool calls and resource requests
- Provide proper error handling
- Support graceful shutdown

## Configuration Examples

### Multi-Platform Development Team

```yaml
settings:
  mcp_servers:
    # Core development platforms
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN
    
    slack:
      command: /opt/autoteam/bin/slack-mcp-server
      args: ["stdio"]
      env:
        SLACK_BOT_TOKEN: $$SLACK_BOT_TOKEN
    
    # Project management
    jira:
      command: /opt/autoteam/bin/jira-mcp-server
      args: ["stdio"]
      env:
        JIRA_URL: $$JIRA_BASE_URL
        JIRA_TOKEN: $$JIRA_API_TOKEN
    
    # CI/CD integration
    jenkins:
      command: /opt/autoteam/bin/jenkins-mcp-server
      args: ["stdio"]
      env:
        JENKINS_URL: $$JENKINS_URL
        JENKINS_TOKEN: $$JENKINS_API_TOKEN
    
    # Documentation
    confluence:
      command: /opt/autoteam/bin/confluence-mcp-server
      args: ["stdio"]
      env:
        CONFLUENCE_URL: $$CONFLUENCE_URL
        CONFLUENCE_TOKEN: $$CONFLUENCE_TOKEN

workers:
  - name: "Full Stack Developer"
    prompt: |
      You have access to GitHub, Slack, Jira, Jenkins, and Confluence.
      Coordinate development work across all these platforms.
```

### Data Processing Pipeline

```yaml
settings:
  mcp_servers:
    # Data sources
    postgresql:
      command: /opt/autoteam/bin/postgresql-mcp-server
      args: ["stdio"]
      env:
        DATABASE_URL: $$POSTGRES_URL
    
    redis:
      command: /opt/autoteam/bin/redis-mcp-server
      args: ["stdio"]
      env:
        REDIS_URL: $$REDIS_URL
    
    s3:
      command: /opt/autoteam/bin/s3-mcp-server
      args: ["stdio"]
      env:
        AWS_ACCESS_KEY_ID: $$AWS_ACCESS_KEY
        AWS_SECRET_ACCESS_KEY: $$AWS_SECRET_KEY
        AWS_REGION: $$AWS_REGION
    
    # Analytics platform
    analytics:
      command: /opt/autoteam/bin/analytics-mcp-server
      args: ["stdio"]
      env:
        ANALYTICS_API_KEY: $$ANALYTICS_API_KEY

workers:
  - name: "Data Engineer"
    prompt: |
      Process data across PostgreSQL, Redis, S3, and analytics platforms.
      Orchestrate ETL pipelines and data quality checks.
```

## Environment Variable Management

### Secure Variable Handling

Use `$$` prefix for MCP environment variables to ensure proper Docker Compose escaping:

```yaml
mcp_servers:
  secure_service:
    env:
      API_KEY: $$SECURE_API_KEY        # Correct - escaped for Docker
      DATABASE_URL: $$DATABASE_URL     # Correct - escaped for Docker
```

### Variable Sources

MCP servers can access environment variables from:

1. **Global .env file** - Project-level variables
2. **System environment** - OS-level variables
3. **CI/CD secrets** - Pipeline-injected variables
4. **Container environment** - Runtime-injected variables

## Troubleshooting MCP Integration

### Common Issues

**MCP Server Not Starting:**
```bash
# Check MCP server binary
ls -la /opt/autoteam/bin/github-mcp-server

# Test MCP server manually
/opt/autoteam/bin/github-mcp-server stdio
```

**Authentication Failures:**
```bash
# Verify environment variables
echo $GITHUB_TOKEN
env | grep SLACK

# Check .env file
cat .env | grep -E "(GITHUB|SLACK|DATABASE)"
```

**Connection Issues:**
```yaml
# Add debugging to MCP server
mcp_servers:
  github:
    command: /opt/autoteam/bin/github-mcp-server
    args: ["stdio", "--debug"]  # Enable debug logging
    env:
      DEBUG: "true"
      LOG_LEVEL: "debug"
```

### Debugging MCP Communication

Enable MCP debugging in agent configurations:

```yaml
workers:
  - name: "Debug Agent"
    settings:
      service:
        environment:
          MCP_DEBUG: "true"
          LOG_LEVEL: "debug"
```

Check MCP logs in agent containers:
```bash
# View agent logs
docker logs autoteam-senior_developer-1

# Check MCP server logs
docker exec autoteam-senior_developer-1 cat /var/log/mcp-server.log
```

## Next Steps

- [Examples](examples.md) - Real-world MCP integration examples
- [Architecture](architecture.md) - Deep dive into MCP hub architecture
- [Development](development.md) - Building custom MCP servers