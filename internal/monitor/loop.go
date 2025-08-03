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

			// Configure MCP servers once per agent (not per repository)
			if configurable, ok := m.agent.(agent.Configurable); ok {
				lgr.Debug("Ensuring MCP servers are configured for agent")
				if err := configurable.Configure(ctx); err != nil {
					lgr.Warn("Failed to configure MCP servers for agent", zap.Error(err))
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

// buildNotificationPrompt builds a type-specific prompt for AI to handle a notification
func (m *Monitor) buildNotificationPrompt(notification *github.NotificationInfo) string {
	var promptParts []string

	// Route to type-specific prompt builder
	var typeSpecificPrompt string
	switch notification.CorrelatedType {
	case "review_request":
		typeSpecificPrompt = m.buildReviewRequestPrompt(notification)
	case "assigned_issue":
		typeSpecificPrompt = m.buildAssignedIssuePrompt(notification)
	case "assigned_pr":
		typeSpecificPrompt = m.buildAssignedPRPrompt(notification)
	case "mention":
		typeSpecificPrompt = m.buildMentionPrompt(notification)
	case "failed_workflow":
		typeSpecificPrompt = m.buildFailedWorkflowPrompt(notification)
	case "unread_comment":
		typeSpecificPrompt = m.buildUnreadCommentPrompt(notification)
	default:
		// Generic notification or unknown type
		typeSpecificPrompt = m.buildGenericNotificationPrompt(notification)
	}

	promptParts = append(promptParts, typeSpecificPrompt)

	// Add agent-specific prompt from configuration
	if m.globalConfig.Agent.Prompt != "" {
		promptParts = append(promptParts, "", "**Your Role-Specific Instructions:**", m.globalConfig.Agent.Prompt)
	}

	// Add mandatory read-marking instruction (applies to all types)
	finalInstructions := fmt.Sprintf("**CRITICAL REQUIREMENT**: You MUST mark this notification as read after processing, regardless of whether it was actionable or stale. Use this exact command:\n\n```bash\ngh api -X PATCH \"notifications/threads/%s\"\n```\n\nThis is mandatory for preventing duplicate processing in future iterations.", notification.ThreadID)
	promptParts = append(promptParts, "", finalInstructions)

	return strings.Join(promptParts, "\n")
}
