# Examples

This guide provides real-world AutoTeam configurations for common use cases. Each example includes complete configuration files and setup instructions.

## Development Team Automation

### Autonomous Code Review and Issue Management

**Use Case**: Automate routine development tasks including PR reviews, issue triage, and team coordination.

```yaml
# autoteam.yaml
workers:
  - name: "Senior Developer"
    enabled: true
    prompt: |
      You are a senior developer responsible for code quality and implementation.
      
      Your responsibilities:
      - Review pull requests for code quality, security, and best practices
      - Implement features from well-defined GitHub issues
      - Triage and label incoming issues appropriately
      - Maintain coding standards and documentation
      
      Always explain your decisions and provide constructive feedback.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${SENIOR_DEVELOPER_GITHUB_TOKEN}

  - name: "DevOps Engineer"
    enabled: true
    prompt: |
      You are a DevOps engineer focused on infrastructure and deployment.
      
      Your responsibilities:
      - Monitor CI/CD pipelines and deployment status
      - Handle infrastructure-related issues and PRs
      - Manage deployment workflows and rollbacks
      - Ensure system reliability and performance
      
      Escalate critical infrastructure issues to human team members.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${DEVOPS_ENGINEER_GITHUB_TOKEN}

settings:
  team_name: "dev-automation"
  sleep_duration: 60  # Check every minute
  
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN
        GITHUB_USER: $$GITHUB_USER
    
    slack:
      command: /opt/autoteam/bin/slack-mcp-server
      args: ["stdio"]
      env:
        SLACK_BOT_TOKEN: $$SLACK_BOT_TOKEN

  flow:
    # Parallel scanning of different notification types
    - name: scan_pr_reviews
      type: gemini
      args: ["--model", "gemini-2.5-flash"]
      prompt: |
        Scan GitHub for pull requests requiring code review.
        Focus on PRs that are ready for review and not in draft status.
        Output: List of PRs with repository, PR number, and urgency level.

    - name: scan_issues
      type: gemini
      args: ["--model", "gemini-2.5-flash"]
      prompt: |
        Scan GitHub for issues requiring attention:
        - New issues without labels
        - Issues assigned to the team
        - Bug reports needing triage
        Output: List of issues with repository, issue number, and type.

    - name: scan_deployments
      type: gemini
      args: ["--model", "gemini-2.5-flash"]
      prompt: |
        Check CI/CD pipeline status and recent deployments:
        - Failed builds or deployments
        - Pending deployment approvals
        - Infrastructure alerts
        Output: List of deployment-related items needing attention.

    # Specialized processing based on scan results
    - name: handle_code_reviews
      type: claude
      depends_on: [scan_pr_reviews]
      skip_when: "{{- index .inputs 0 | contains \"No PRs\" -}}"
      prompt: |
        Review the pull requests identified in the scan:
        {{index .inputs 0}}
        
        For each PR:
        1. Analyze code changes for quality and best practices
        2. Check for security vulnerabilities
        3. Verify tests are included and passing
        4. Provide constructive feedback or approve if ready
        5. Request changes if issues are found

    - name: triage_issues
      type: claude
      depends_on: [scan_issues]
      skip_when: "{{- index .inputs 0 | contains \"No issues\" -}}"
      prompt: |
        Triage the issues identified in the scan:
        {{index .inputs 0}}
        
        For each issue:
        1. Analyze the issue description and classify the type
        2. Add appropriate labels (bug, feature, documentation, etc.)
        3. Set priority based on severity and impact
        4. Assign to appropriate team member if clear ownership
        5. Add comments requesting clarification if needed

    - name: handle_deployments
      type: claude
      depends_on: [scan_deployments]
      skip_when: "{{- index .inputs 0 | contains \"No deployment issues\" -}}"
      prompt: |
        Handle deployment and infrastructure issues:
        {{index .inputs 0}}
        
        For each item:
        1. Investigate the root cause of failures
        2. Retry failed deployments if safe to do so
        3. Approve pending deployments if all checks pass
        4. Escalate critical infrastructure issues to on-call team
        5. Update relevant stakeholders on status

    # Team communication and summary
    - name: team_status_update
      type: gemini
      depends_on: [handle_code_reviews, triage_issues, handle_deployments]
      prompt: |
        Create a concise team status update based on automated actions:
        - Code Reviews: {{index .inputs 0}}
        - Issue Triage: {{index .inputs 1}}  
        - Deployments: {{index .inputs 2}}
        
        Post summary to #dev-team Slack channel if significant actions were taken.
```

**Environment Variables (.env):**
```bash
# GitHub tokens for different team roles
SENIOR_DEVELOPER_GITHUB_TOKEN=ghp_senior_dev_token_here
DEVOPS_ENGINEER_GITHUB_TOKEN=ghp_devops_token_here
GITHUB_USER=your-github-username

# Slack integration
SLACK_BOT_TOKEN=xoxb-your-slack-bot-token
```

**Results**: Handles ~70% of routine development tasks, allowing human developers to focus on architecture and complex problem-solving.

---

## Personal Developer Assistant

### Intelligent Notification Management

**Use Case**: Reduce notification overload while ensuring nothing important is missed.

```yaml
# autoteam.yaml
workers:
  - name: "Personal Assistant"
    enabled: true
    prompt: |
      You are my personal development assistant. Your goal is to help me stay
      focused on deep work while ensuring I don't miss important communications.
      
      Handle routine tasks autonomously but escalate anything that requires
      my personal attention or decision-making.
      
      Prioritize based on:
      - Urgency and impact
      - Relationship to current projects  
      - Whether it requires unique human insight
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${PERSONAL_GITHUB_TOKEN}
          SLACK_TOKEN: ${PERSONAL_SLACK_TOKEN}

settings:
  team_name: "personal-assistant"
  sleep_duration: 300  # Check every 5 minutes
  
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN
    
    slack:
      command: /opt/autoteam/bin/slack-mcp-server
      args: ["stdio"] 
      env:
        SLACK_BOT_TOKEN: $$SLACK_TOKEN

  flow:
    - name: morning_priorities_scan
      type: gemini
      args: ["--model", "gemini-2.5-flash"]
      prompt: |
        Perform a comprehensive morning scan of all platforms:
        
        Check GitHub for:
        - Mentions in issues, PRs, or discussions
        - PRs ready for my review
        - Issues assigned to me
        - New notifications since yesterday
        
        Check Slack for:
        - Direct messages
        - Mentions in channels
        - Thread updates I'm following
        
        Categorize each item as:
        - URGENT: Needs immediate attention
        - IMPORTANT: Should handle today
        - ROUTINE: Can be handled automatically
        - INFO: Just for awareness

    - name: handle_routine_tasks
      type: claude
      depends_on: [morning_priorities_scan]
      skip_when: "{{- index .inputs 0 | contains \"No routine tasks\" -}}"
      prompt: |
        Handle routine tasks from the morning scan:
        {{index .inputs 0}}
        
        For ROUTINE items:
        - Approve simple/obvious PRs (typo fixes, documentation updates)
        - Respond to straightforward questions I've answered before
        - Update issue statuses that are clear-cut
        - Acknowledge messages that just need a quick response
        
        Do NOT handle anything that:
        - Requires strategic decisions
        - Involves complex technical discussions  
        - Could set expectations about timelines or commitments
        - Relates to sensitive or controversial topics

    - name: prepare_focus_summary
      type: claude
      depends_on: [morning_priorities_scan, handle_routine_tasks]
      prompt: |
        Create a focused summary for my attention:
        
        Original scan results: {{index .inputs 0}}
        Items handled automatically: {{index .inputs 1}}
        
        Prepare:
        1. URGENT items that need my immediate attention (with context)
        2. IMPORTANT items for today (prioritized list)
        3. Summary of routine tasks completed on my behalf
        4. Any patterns or insights worth noting
        
        Keep it concise - I should be able to review this in under 2 minutes.

    - name: send_daily_digest
      type: gemini
      depends_on: [prepare_focus_summary]
      skip_when: "{{- index .inputs 0 | contains \"Nothing significant\" -}}"
      prompt: |
        Send my personalized daily digest:
        {{index .inputs 0}}
        
        Send as a direct message to myself in Slack with:
        - Clear action items that need my attention
        - Summary of what was handled automatically  
        - Estimated time needed for my review
        - Links to specific items for quick access
```

**Environment Variables (.env):**
```bash
# Personal tokens
PERSONAL_GITHUB_TOKEN=ghp_your_personal_token
PERSONAL_SLACK_TOKEN=xoxb-your-personal_bot_token
```

**Results**: 85% reduction in notification interruptions while maintaining awareness of all important communications.

---

## Multi-Platform Data Pipeline

### Customer Support Orchestration

**Use Case**: Coordinate customer support across multiple channels with intelligent routing and response.

```yaml
# autoteam.yaml
workers:
  - name: "Support Coordinator"
    enabled: true
    prompt: |
      You coordinate customer support across multiple channels and platforms.
      
      Your responsibilities:
      - Monitor support requests from all channels
      - Route requests to appropriate team members
      - Provide initial responses for common issues
      - Escalate complex or sensitive issues appropriately
      - Track resolution status and follow up as needed
      
      Always maintain a helpful, professional tone and ensure customers
      feel heard even when escalating to human agents.
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${SUPPORT_GITHUB_TOKEN}
          SLACK_TOKEN: ${SUPPORT_SLACK_TOKEN}
          DATABASE_URL: ${SUPPORT_DATABASE_URL}

settings:
  team_name: "support-coordination"
  sleep_duration: 120  # Check every 2 minutes
  
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN
    
    slack:
      command: /opt/autoteam/bin/slack-mcp-server
      args: ["stdio"]
      env:
        SLACK_BOT_TOKEN: $$SLACK_TOKEN
    
    support_db:
      command: /opt/autoteam/bin/postgresql-mcp-server
      args: ["stdio"]
      env:
        DATABASE_URL: $$DATABASE_URL
    
    knowledge_base:
      command: /opt/autoteam/bin/knowledge-base-mcp-server
      args: ["stdio"]
      env:
        KB_API_KEY: $$KNOWLEDGE_BASE_API_KEY

  flow:
    # Parallel collection from all support channels
    - name: collect_github_issues
      type: gemini
      prompt: |
        Scan GitHub issues for customer support requests:
        - Issues labeled with 'support', 'help', 'question'
        - Issues from external contributors asking for help
        - Bug reports that need response or investigation
        
        For each issue, extract:
        - Issue number and repository
        - Customer/user information
        - Problem description and urgency
        - Current status and last activity

    - name: collect_slack_requests
      type: gemini
      prompt: |
        Check Slack channels for support requests:
        - #support channel messages
        - Direct messages to support bot
        - Mentions of support team in other channels
        
        For each request, extract:
        - Channel and message timestamp
        - Customer information if available
        - Issue description and severity
        - Any previous responses or acknowledgments

    - name: collect_database_tickets
      type: qwen
      prompt: |
        Query the support database for pending tickets:
        - New tickets without initial response
        - Tickets requiring follow-up
        - Escalated tickets needing attention
        - SLA violations or approaching deadlines
        
        Return structured data with:
        - Ticket ID and customer information
        - Issue category and priority
        - Current status and assigned agent
        - Time since last update

    - name: query_knowledge_base
      type: qwen
      prompt: |
        Search the knowledge base for relevant solutions:
        - Common issues and their resolutions
        - Product documentation and FAQs
        - Troubleshooting guides
        - Escalation procedures
        
        Build a reference of available solutions for common problems.

    # Intelligent request processing and routing
    - name: triage_and_route
      type: claude
      depends_on: [collect_github_issues, collect_slack_requests, collect_database_tickets]
      prompt: |
        Triage all incoming support requests:
        
        GitHub Issues: {{index .inputs 0}}
        Slack Requests: {{index .inputs 1}}
        Database Tickets: {{index .inputs 2}}
        
        For each request:
        1. Categorize the issue type (bug, feature request, how-to, billing, etc.)
        2. Assess urgency (critical, high, medium, low)
        3. Determine if it can be handled with existing knowledge
        4. Identify the best person or team to handle it
        5. Check for duplicates or related issues
        
        Route requests appropriately and prepare initial responses.

    - name: provide_initial_responses
      type: claude
      depends_on: [triage_and_route, query_knowledge_base]
      skip_when: "{{- index .inputs 0 | contains \"No responses needed\" -}}"
      prompt: |
        Provide initial responses for requests that can be handled immediately:
        
        Triaged Requests: {{index .inputs 0}}
        Available Solutions: {{index .inputs 1}}
        
        For routine requests:
        - Acknowledge receipt quickly (within 1 hour)
        - Provide solutions from knowledge base when applicable
        - Ask clarifying questions if needed
        - Set appropriate expectations for response time
        
        For complex requests:
        - Acknowledge and confirm escalation to human agent
        - Provide initial guidance if available
        - Estimate response timeframe
        - Ensure customer feels heard

    - name: escalate_complex_issues
      type: claude
      depends_on: [triage_and_route]
      skip_when: "{{- index .inputs 0 | contains \"No escalation needed\" -}}"
      prompt: |
        Escalate complex issues to appropriate team members:
        {{index .inputs 0}}
        
        For issues requiring escalation:
        1. Notify the appropriate team member via Slack
        2. Provide full context and customer information
        3. Include analysis and any initial investigation done
        4. Set follow-up reminders for tracking
        5. Update customer with escalation status
        
        Escalate immediately for:
        - Security vulnerabilities or data breaches
        - Billing or payment issues
        - Service outages affecting multiple customers
        - Angry or dissatisfied customers

    # Follow-up and metrics tracking
    - name: update_metrics_and_followup
      type: qwen
      depends_on: [provide_initial_responses, escalate_complex_issues]
      prompt: |
        Update support metrics and schedule follow-ups:
        
        Initial Responses: {{index .inputs 0}}
        Escalated Issues: {{index .inputs 1}}
        
        Actions:
        1. Update response time metrics in database
        2. Log all actions taken for tracking
        3. Schedule follow-up reminders for pending issues
        4. Update customer satisfaction tracking
        5. Identify trends or recurring issues for team review
        
        Generate daily summary of support activities and metrics.
```

**Environment Variables (.env):**
```bash
# Support system integration
SUPPORT_GITHUB_TOKEN=ghp_support_team_token
SUPPORT_SLACK_TOKEN=xoxb-support-bot-token
SUPPORT_DATABASE_URL=postgresql://user:pass@localhost/support_db
KNOWLEDGE_BASE_API_KEY=kb_api_key_here
```

**Results**: 60% faster response time, 45% better escalation accuracy, improved customer satisfaction through consistent communication.

---

## Cross-Platform Project Management

### Agile Development Coordination

**Use Case**: Synchronize work across GitHub, Jira, and Slack for agile development teams.

```yaml
# autoteam.yaml
workers:
  - name: "Scrum Master"
    enabled: true
    prompt: |
      You are an AI Scrum Master responsible for facilitating agile development
      processes across multiple platforms.
      
      Your responsibilities:
      - Keep GitHub PRs, Jira tickets, and Slack communications in sync
      - Identify blockers and impediments across platforms
      - Update sprint progress and generate status reports
      - Facilitate standup preparation and retrospective data collection
      - Ensure team has visibility into work status and dependencies
    settings:
      service:
        environment:
          GITHUB_TOKEN: ${SCRUM_GITHUB_TOKEN}
          JIRA_TOKEN: ${JIRA_API_TOKEN}
          SLACK_TOKEN: ${SCRUM_SLACK_TOKEN}

settings:
  team_name: "agile-coordination"
  sleep_duration: 180  # Check every 3 minutes
  
  mcp_servers:
    github:
      command: /opt/autoteam/bin/github-mcp-server
      args: ["stdio"]
      env:
        GITHUB_TOKEN: $$GITHUB_TOKEN
    
    jira:
      command: /opt/autoteam/bin/jira-mcp-server
      args: ["stdio"]
      env:
        JIRA_URL: $$JIRA_BASE_URL
        JIRA_TOKEN: $$JIRA_TOKEN
    
    slack:
      command: /opt/autoteam/bin/slack-mcp-server
      args: ["stdio"]
      env:
        SLACK_BOT_TOKEN: $$SLACK_TOKEN

  flow:
    # Platform status collection
    - name: github_sprint_status
      type: gemini
      prompt: |
        Collect current sprint status from GitHub:
        - PRs linked to sprint tickets (via commit messages or PR descriptions)
        - PR review status and merge readiness
        - Recent commits and development activity
        - Branch status for sprint features
        
        Focus on work items related to current sprint goals.

    - name: jira_sprint_progress
      type: qwen
      prompt: |
        Query Jira for current sprint progress:
        - Active sprint tickets and their status
        - Story points completed vs remaining
        - Blocked or impeded tickets
        - Tickets without recent activity
        - Sprint burndown metrics
        
        Calculate sprint health and identify risks.

    - name: team_communication_scan
      type: claude
      prompt: |
        Scan Slack for team communication indicators:
        - Blockers mentioned in team channels
        - Questions about sprint work
        - Status updates from team members
        - Meeting discussions about current work
        
        Identify communication gaps or coordination issues.

    # Cross-platform synchronization
    - name: sync_github_jira
      type: claude
      depends_on: [github_sprint_status, jira_sprint_progress]
      prompt: |
        Synchronize status between GitHub and Jira:
        
        GitHub Status: {{index .inputs 0}}
        Jira Progress: {{index .inputs 1}}
        
        Actions:
        1. Update Jira ticket status based on PR merge status
        2. Link PRs to Jira tickets if not already connected
        3. Update story point estimates if work scope changed
        4. Move tickets to appropriate workflow status
        5. Flag mismatches between GitHub activity and Jira status

    - name: identify_blockers_and_risks
      type: claude
      depends_on: [jira_sprint_progress, team_communication_scan]
      prompt: |
        Identify blockers and sprint risks:
        
        Jira Progress: {{index .inputs 0}}
        Team Communications: {{index .inputs 1}}
        
        Analyze for:
        1. Tickets blocked for more than 24 hours
        2. Work items without recent progress
        3. Dependencies affecting multiple tickets  
        4. Team members with high workload concentration
        5. Technical debt items affecting sprint goals
        
        Prioritize blockers by impact on sprint success.

    # Team communication and reporting
    - name: prepare_standup_report
      type: claude
      depends_on: [sync_github_jira, identify_blockers_and_risks]
      prompt: |
        Prepare daily standup report:
        
        Sync Status: {{index .inputs 0}}
        Blockers/Risks: {{index .inputs 1}}
        
        Generate:
        1. Sprint progress summary (completed vs remaining work)
        2. Yesterday's completed items by team member
        3. Current blockers requiring team discussion
        4. Today's priorities and dependencies
        5. Sprint goal progress assessment
        
        Format for easy consumption in 10-minute standup.

    - name: update_stakeholders
      type: gemini
      depends_on: [prepare_standup_report]
      skip_when: "{{- now.Hour | lt 9 | or (now.Hour | gt 17) -}}"  # Only during work hours
      prompt: |
        Share standup report with stakeholders:
        {{index .inputs 0}}
        
        Post to:
        - #dev-team channel for daily standup
        - #project-updates for stakeholder visibility
        - Update sprint dashboard if significant changes
        
        Highlight any critical blockers or risks that need attention.

    # Weekly sprint analysis
    - name: weekly_retrospective_prep
      type: claude
      skip_when: "{{- now.Weekday | ne 5 -}}"  # Only on Fridays
      depends_on: [identify_blockers_and_risks]
      prompt: |
        Prepare weekly retrospective data:
        {{index .inputs 0}}
        
        Analyze the week's patterns:
        1. Most common types of blockers
        2. Velocity trends and estimation accuracy
        3. Cross-platform synchronization issues
        4. Team coordination challenges
        5. Process improvements that could help
        
        Prepare discussion topics for retrospective meeting.
```

**Environment Variables (.env):**
```bash
# Project management integration
SCRUM_GITHUB_TOKEN=ghp_scrum_master_token
JIRA_API_TOKEN=your_jira_api_token
JIRA_BASE_URL=https://yourcompany.atlassian.net
SCRUM_SLACK_TOKEN=xoxb-scrum-bot-token
```

**Results**: Improved sprint visibility, 40% reduction in status meeting time, better cross-platform work coordination.

---

## Deployment Commands

### Running the Examples

For any example configuration:

1. **Copy the configuration** to `autoteam.yaml`
2. **Create the environment file** with your tokens and credentials
3. **Initialize and deploy**:

```bash
# Generate Docker Compose configuration
autoteam generate

# Start the automation team
autoteam up

# Monitor logs
docker compose logs -f

# Stop when needed
autoteam down
```

### Customization Tips

- **Adjust sleep_duration** based on how frequently you want automation to run
- **Modify prompts** to match your specific workflow and terminology
- **Add/remove MCP servers** based on your platform integrations
- **Customize skip conditions** for flow steps to match your business logic
- **Scale workers** by adding more specialized agents for different responsibilities

## Next Steps

- [Configuration Guide](configuration.md) - Detailed configuration options
- [Flow System](flows.md) - Advanced workflow patterns
- [MCP Integration](mcp.md) - Adding new platform integrations
- [Development](development.md) - Contributing custom features