package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/entrypoint"
	"autoteam/internal/git"
	"autoteam/internal/github"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// Config contains configuration for the simplified monitor
type Config struct {
	CheckInterval time.Duration
	DryRun        bool
	TeamName      string
}

// Monitor handles the main monitoring loop
type Monitor struct {
	githubClient *github.Client
	agent        agent.Agent
	config       Config
	globalConfig *entrypoint.Config
	gitSetup     *git.Setup
}

// New creates a new monitor instance
func New(githubClient *github.Client, selectedAgent agent.Agent, monitorConfig Config, globalConfig *entrypoint.Config) *Monitor {
	gitSetup := git.NewSetup(globalConfig.Git, globalConfig.GitHub, globalConfig.Repositories)

	return &Monitor{
		githubClient: githubClient,
		agent:        selectedAgent,
		config:       monitorConfig,
		globalConfig: globalConfig,
		gitSetup:     gitSetup,
	}
}

// Start starts the simplified notification processing loop
func (m *Monitor) Start(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Starting simplified notification monitor", zap.Duration("check_interval", m.config.CheckInterval))

	// Get authenticated user info
	user, err := m.githubClient.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}

	username := user.GetLogin()
	lgr.Info("Authenticated as GitHub user", zap.String("username", username))

	// Log repository patterns for transparency
	if m.globalConfig.Repositories != nil {
		lgr.Info("Repository patterns configured",
			zap.Strings("include", m.globalConfig.Repositories.Include),
			zap.Strings("exclude", m.globalConfig.Repositories.Exclude))
	}

	lgr.Info("Starting single notification processing loop (no complex prioritization or state management)")

	// Start continuous notification processing loop
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lgr.Info("Monitor shutting down due to context cancellation")
			return ctx.Err()

		case <-ticker.C:
			// Process single notification
			if err := m.processSingleNotification(ctx); err != nil {
				lgr.Warn("Failed to process notification", zap.Error(err))
			}
		}
	}
}

// processSingleNotification gets one notification and processes it with AI validation
func (m *Monitor) processSingleNotification(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// Get single unread notification
	notification, err := m.githubClient.GetSingleNotification(ctx)
	if err != nil {
		return fmt.Errorf("failed to get single notification: %w", err)
	}

	// No notifications found - this is normal, just wait for next cycle
	if notification == nil {
		lgr.Debug("No unread notifications found")
		return nil
	}

	lgr.Info("Processing notification",
		zap.String("reason", notification.Reason),
		zap.String("subject", notification.Subject),
		zap.String("repository", notification.Repository),
		zap.String("thread_id", notification.ThreadID))

	// Ensure repository is cloned before processing (if it has a repository)
	if notification.Repository != "" {
		if err := m.gitSetup.SetupRepository(ctx, notification.Repository); err != nil {
			lgr.Warn("Failed to setup repository, continuing with notification processing",
				zap.String("repository", notification.Repository),
				zap.Error(err))
		} else {
			// Get default branch and setup git state
			parts := strings.Split(notification.Repository, "/")
			if len(parts) == 2 {
				owner, repo := parts[0], parts[1]
				defaultBranch, err := m.githubClient.GetDefaultBranch(ctx, owner, repo)
				if err != nil {
					lgr.Debug("Failed to get default branch, using main", zap.Error(err))
					defaultBranch = "main"
				}

				// Setup fresh git state for notification processing
				lgr.Debug("Setting up fresh git state", zap.String("repository", notification.Repository), zap.String("branch", defaultBranch))
				if err := m.gitSetup.SwitchToMainBranch(ctx, notification.Repository, defaultBranch); err != nil {
					lgr.Warn("Failed to switch to main branch", zap.Error(err))
				}
			}
		}
	}

	// Build notification prompt for AI
	prompt := m.buildNotificationPrompt(notification)
	lgr.Debug("Built notification prompt", zap.Int("length", len(prompt)))

	// Get working directory (use repository-specific if available, otherwise current)
	workingDir := ""
	if notification.Repository != "" {
		workingDir = m.gitSetup.GetRepositoryWorkingDirectory(notification.Repository)
	}

	// Execute AI agent with notification
	runOptions := agent.RunOptions{
		MaxRetries:       1,
		ContinueMode:     false,
		OutputFormat:     "stream-json",
		Verbose:          true,
		DryRun:           m.config.DryRun,
		WorkingDirectory: workingDir,
	}

	if err := m.agent.Run(ctx, prompt, runOptions); err != nil {
		lgr.Error("AI agent failed to process notification",
			zap.String("thread_id", notification.ThreadID),
			zap.String("reason", notification.Reason),
			zap.Error(err))
		return fmt.Errorf("agent execution failed: %w", err)
	}

	lgr.Info("Notification processed successfully by AI",
		zap.String("thread_id", notification.ThreadID),
		zap.String("reason", notification.Reason))

	return nil
}

// buildNotificationPrompt builds a prompt for AI to handle a raw notification
func (m *Monitor) buildNotificationPrompt(notification *github.NotificationInfo) string {
	var promptParts []string

	// Main notification handling prompt
	notificationPrompt := fmt.Sprintf(`🔔 **GitHub Notification to Process**

**Notification Details:**
- **Reason**: %s
- **Subject**: %s
- **Repository**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Process this GitHub notification intelligently

**CRITICAL INSTRUCTIONS**:

1. **Validate if Actual**: First, check if this notification represents actual pending work:
   - Use GitHub CLI to check current state (e.g., 'gh pr view <number>' or 'gh issue view <number>')
   - Some notifications may be stale (PR already merged, issue closed, etc.)
   - If notification is stale/not actionable → Mark as read and explain why

2. **Handle if Actionable**: If the notification represents real pending work:
   - **Review Requests**: Use 'gh pr view <number>' to examine PR, then 'gh pr review <number>' to submit review
   - **Mentions**: Use 'gh issue view <number>' or 'gh pr view <number>' to read context, then comment with 'gh issue comment' or 'gh pr comment'
   - **Comments**: Use 'gh issue view <number>' or 'gh pr view <number>' to read thread, then respond appropriately
   - **Assignments**: Use 'gh issue view <number>' or 'gh pr view <number>' to understand requirements, then take action
   - **CI Failures**: Use 'gh run view <run-id>' to examine logs, then fix issues and push changes
   - **State Changes**: Use GitHub CLI to review changes and take appropriate action

3. **MANDATORY**: After processing (whether actionable or stale):
   - **ALWAYS** mark this notification as read using GitHub CLI command:
     `+"`"+`bash
     gh api -X PATCH "notifications/threads/%s"
     `+"`"+`
   - This prevents duplicate processing in future iterations
   - You MUST run this command regardless of whether the notification was actionable or stale

4. **Communication**: If the required action is unclear, write a comment asking for clarification rather than ignoring

**Expected Workflow**:
1. Check notification validity using GitHub CLI commands
2. If stale → Mark as read with GitHub CLI and log why
3. If actionable → Perform the required action thoroughly
4. Mark notification as read with GitHub CLI when complete
5. Document what you did

**GitHub CLI Command to Mark as Read**:
`+"`"+`bash
gh api -X PATCH "notifications/threads/%s"
`+"`"+`

**Focus**: Handle ONE notification completely before moving to the next.`,
		notification.Reason,
		notification.Subject,
		notification.Repository,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.ThreadID,
		notification.ThreadID)

	promptParts = append(promptParts, notificationPrompt)

	// Add agent-specific prompt from configuration
	if m.globalConfig.Agent.Prompt != "" {
		promptParts = append(promptParts, "", "**Your Role-Specific Instructions:**", m.globalConfig.Agent.Prompt)
	}

	// Add final emphasis
	finalInstructions := fmt.Sprintf("**REMEMBER**: You MUST mark the notification as read after processing, regardless of whether it was actionable or stale. Use this exact command:\n\n```bash\ngh api -X PATCH \"notifications/threads/%s\"\n```\n\nThis is critical for preventing duplicate work in future iterations.", notification.ThreadID)
	promptParts = append(promptParts, "", finalInstructions)

	return strings.Join(promptParts, "\n")
}
