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

// Config contains configuration for the monitor
type Config struct {
	CheckInterval time.Duration
	MaxRetries    int
	DryRun        bool
	TeamName      string
	MaxAttempts   int
}

// Monitor handles the main monitoring loop
type Monitor struct {
	githubClient *github.Client
	agent        agent.Agent
	config       Config
	globalConfig *entrypoint.Config
	gitSetup     *git.Setup

	// New components for single item processing
	stateManager       *StateManager
	resolutionDetector *ResolutionDetector
	itemPrioritizer    *ItemPrioritizer

	// Configuration for max attempts
	maxAttempts int
}

// New creates a new monitor instance
func New(githubClient *github.Client, selectedAgent agent.Agent, monitorConfig Config, globalConfig *entrypoint.Config) *Monitor {
	gitSetup := git.NewSetup(globalConfig.Git, globalConfig.GitHub, globalConfig.Repositories)

	// Initialize new components
	stateManager := NewStateManager()
	resolutionDetector := NewResolutionDetector(githubClient)
	itemPrioritizer := NewItemPrioritizer(stateManager)

	// Use configured max attempts, default to 3 if not set
	maxAttempts := monitorConfig.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}

	return &Monitor{
		githubClient:       githubClient,
		agent:              selectedAgent,
		config:             monitorConfig,
		globalConfig:       globalConfig,
		gitSetup:           gitSetup,
		stateManager:       stateManager,
		resolutionDetector: resolutionDetector,
		itemPrioritizer:    itemPrioritizer,
		maxAttempts:        maxAttempts,
	}
}

// Start starts the monitoring loop
func (m *Monitor) Start(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Starting monitor", zap.Duration("check_interval", m.config.CheckInterval))

	// Get authenticated user info
	user, err := m.githubClient.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}

	username := user.GetLogin()
	lgr.Info("Authenticated as GitHub user", zap.String("username", username))

	// TODO: Multi-repository support - get default branch per repository
	// For now, use "main" as default branch for multi-repo compatibility
	defaultBranch := "main"
	lgr.Info("Using default branch", zap.String("branch", defaultBranch))

	// Log all matched repositories before starting monitoring
	lgr.Info("Initializing repository discovery")
	filteredRepos, err := m.githubClient.GetFilteredRepositories(ctx, username)
	if err != nil {
		lgr.Warn("Failed to get filtered repositories during initialization", zap.Error(err))
		// Show patterns as fallback
		if m.globalConfig.Repositories != nil {
			lgr.Debug("Repository patterns configured",
				zap.Strings("include", m.globalConfig.Repositories.Include),
				zap.Strings("exclude", m.globalConfig.Repositories.Exclude))
		}
	} else {
		if len(filteredRepos) == 0 {
			lgr.Warn("No repositories found matching the configured patterns")
			if m.globalConfig.Repositories != nil {
				lgr.Debug("Configured patterns",
					zap.Strings("include", m.globalConfig.Repositories.Include),
					zap.Strings("exclude", m.globalConfig.Repositories.Exclude))
			}
		} else {
			lgr.Info("Found repositories matching configured patterns", zap.Int("count", len(filteredRepos)))
			for i, repo := range filteredRepos {
				lgr.Debug("Repository discovered", zap.Int("index", i+1), zap.String("name", repo.FullName), zap.String("url", repo.URL))
			}
		}
	}

	// Run initial check
	if err := m.checkAndProcess(ctx, username, defaultBranch); err != nil {
		lgr.Warn("Initial check failed", zap.Error(err))
	}

	// Start periodic monitoring
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lgr.Info("Monitor shutting down due to context cancellation")
			return ctx.Err()

		case <-ticker.C:
			// Build repository list for logging by resolving patterns to actual repos
			var repoLog string
			filteredRepos, err := m.githubClient.GetFilteredRepositories(ctx, username)
			if err != nil {
				lgr.Warn("Failed to get filtered repositories for logging", zap.Error(err))
				// Fallback to showing patterns
				if m.globalConfig.Repositories != nil {
					includedPatterns := strings.Join(m.globalConfig.Repositories.Include, ", ")
					repoLog = fmt.Sprintf(" (patterns: %s)", includedPatterns)
				}
			} else {
				var repoNames []string
				for _, repo := range filteredRepos {
					repoNames = append(repoNames, repo.FullName)
				}
				if len(repoNames) > 0 {
					repoLog = fmt.Sprintf(" (tracking: %s)", strings.Join(repoNames, ", "))
				} else {
					repoLog = " (no matching repositories found)"
				}
			}
			lgr.Info("Checking for pending items", zap.String("timestamp", time.Now().Format(time.RFC3339)), zap.String("repositories", repoLog))

			if err := m.checkAndProcess(ctx, username, defaultBranch); err != nil {
				lgr.Warn("Check failed", zap.Error(err))
			}
		}
	}
}

// checkAndProcess implements the single item processing workflow
func (m *Monitor) checkAndProcess(ctx context.Context, username, defaultBranch string) error {
	lgr := logger.FromContext(ctx)
	// Clean up old failure records
	if err := m.stateManager.CleanupOldFailures(); err != nil {
		lgr.Warn("Failed to cleanup old failures", zap.Error(err))
	}

	// Check if we have an item currently being processed
	currentItem := m.stateManager.GetCurrentItem()
	if currentItem != nil {
		lgr.Info("Continuing with item",
			zap.String("type", currentItem.Type),
			zap.Int("number", currentItem.Number),
			zap.String("title", currentItem.Title),
			zap.Int("attempt", currentItem.AttemptCount))

		// Check if current item was resolved since last attempt
		result, err := m.resolutionDetector.CheckItemResolution(ctx, currentItem, username)
		if err != nil {
			lgr.Warn("Failed to check item resolution", zap.Error(err))
		} else {
			LogResolutionResult(result, currentItem)

			if result == ItemNotFound {
				// Item was resolved, clear it and continue to next item
				if err := m.stateManager.ClearCurrentItem(); err != nil {
					lgr.Warn("Failed to clear resolved item", zap.Error(err))
				}
				lgr.Info("Item resolved successfully! Selecting next item")
				return m.selectAndProcessNextItem(ctx, username, defaultBranch)
			}
		}

		// Item still needs work, continue processing with continue mode
		return m.processItem(ctx, currentItem, defaultBranch, true)
	}

	// No current item, select next one
	return m.selectAndProcessNextItem(ctx, username, defaultBranch)
}

// selectAndProcessNextItem selects the highest priority item and processes it
func (m *Monitor) selectAndProcessNextItem(ctx context.Context, username, defaultBranch string) error {
	lgr := logger.FromContext(ctx)
	// Get all pending items from GitHub
	pendingItems, err := m.githubClient.GetPendingItems(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get pending items: %w", err)
	}

	if pendingItems.IsEmpty() {
		lgr.Info("No pending items found")
		return nil
	}

	lgr.Info("Found total pending items", zap.Int("count", pendingItems.Count()))

	// Select the highest priority item
	selectedItem := m.itemPrioritizer.SelectNextItem(pendingItems)
	if selectedItem == nil {
		lgr.Info("No suitable item to process (all may be in cooldown)")
		return nil
	}

	// Create processing item and set as current
	processingItem := CreateProcessingItemFromPrioritized(selectedItem)

	// Set as current item
	if err := m.stateManager.SetCurrentItem(processingItem); err != nil {
		return fmt.Errorf("failed to set current item: %w", err)
	}

	// Process the selected item
	return m.processItem(ctx, processingItem, defaultBranch, false)
}

// processItem processes a single item
func (m *Monitor) processItem(ctx context.Context, item *ProcessingItem, globalDefaultBranch string, continueMode bool) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Processing item",
		zap.String("type", item.Type),
		zap.Int("number", item.Number),
		zap.String("title", item.Title),
		zap.Int("attempt", item.AttemptCount))

	// Ensure repository is cloned before processing
	if err := m.gitSetup.SetupRepository(ctx, item.Repository); err != nil {
		return fmt.Errorf("failed to setup repository %s: %w", item.Repository, err)
	}

	// Get the actual default branch for this repository
	parts := strings.Split(item.Repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s", item.Repository)
	}
	owner, repo := parts[0], parts[1]

	defaultBranch, err := m.githubClient.GetDefaultBranch(ctx, owner, repo)
	if err != nil {
		lgr.Warn("Failed to get default branch, using fallback",
			zap.String("repository", item.Repository),
			zap.String("fallback_branch", globalDefaultBranch),
			zap.Error(err))
		defaultBranch = globalDefaultBranch
	}

	// Git state management: Only reset for new items, preserve state for continuations
	if !continueMode {
		// New item: Fresh git state (fetch + reset to main)
		lgr.Info("New item: switching to branch and resetting to clean state",
			zap.String("branch", defaultBranch),
			zap.String("repository", item.Repository))
		if err := m.gitSetup.SwitchToMainBranch(ctx, item.Repository, defaultBranch); err != nil {
			return fmt.Errorf("failed to switch to main branch for repository %s: %w", item.Repository, err)
		}
	} else {
		// Continuation: Keep existing git state, don't reset
		lgr.Info("Continuing item: preserving current git state (no reset)")
	}

	// Build the prompt for this specific item
	prompt := m.buildItemPrompt(item, continueMode)
	lgr.Debug("Built prompt for item", zap.Int("length", len(prompt)))

	// Execute the AI agent
	runOptions := agent.RunOptions{
		MaxRetries:       1, // Single retry per iteration, we handle retries at the loop level
		ContinueMode:     continueMode,
		OutputFormat:     "stream-json",
		Verbose:          true,
		DryRun:           m.config.DryRun,
		WorkingDirectory: m.gitSetup.GetRepositoryWorkingDirectory(item.Repository),
	}

	// Increment attempt count
	if err := m.stateManager.IncrementAttempt(); err != nil {
		lgr.Warn("Failed to increment attempt count", zap.Error(err))
	}

	if err := m.agent.Run(ctx, prompt, runOptions); err != nil {
		lgr.Error("Agent execution failed", zap.Error(err))

		// Record failure if max attempts reached
		maxAttempts := m.getMaxAttempts()
		if item.AttemptCount >= maxAttempts {
			itemKey := GetItemKeyFromProcessingItem(item)
			if recordErr := m.stateManager.RecordFailure(itemKey); recordErr != nil {
				lgr.Warn("Failed to record failure", zap.Error(recordErr))
			}

			// Clear current item after max attempts
			if clearErr := m.stateManager.ClearCurrentItem(); clearErr != nil {
				lgr.Warn("Failed to clear failed item", zap.Error(clearErr))
			}

			lgr.Warn("Max attempts reached, moving to cooldown",
				zap.String("type", item.Type),
				zap.Int("number", item.Number))
		}

		return fmt.Errorf("agent execution failed: %w", err)
	}

	lgr.Info("Agent execution completed successfully", zap.String("timestamp", time.Now().Format(time.RFC3339)))
	return nil
}

// buildItemPrompt builds a prompt for a specific item
func (m *Monitor) buildItemPrompt(item *ProcessingItem, continueMode bool) string {
	var promptParts []string

	// Add item-specific context
	var itemContext string
	switch item.Type {
	case "review_request":
		itemContext = fmt.Sprintf("📥 Review Request: [#%d](%s) %s\n\nPlease review this pull request and provide feedback. \n\nIMPORTANT: Make sure you submit your review and mark it as reviewed (approve, request changes, or comment) - don't just leave comments without submitting the review.",
			item.Number, item.URL, item.Title)
	case "assigned_pr":
		itemContext = fmt.Sprintf("🧷 Assigned PR: [#%d](%s) %s\n\nThis pull request is assigned to you. Please work on it.",
			item.Number, item.URL, item.Title)
	case "assigned_issue":
		itemContext = fmt.Sprintf("🚧 Assigned Issue: [#%d](%s) %s\n\nThis issue is assigned to you. Please address it. IMPORTANT: If this issue doesn't require a PR and is completed, make sure to close the issue when done.",
			item.Number, item.URL, item.Title)
	case "pr_with_changes":
		itemContext = fmt.Sprintf("🛠 PR with Changes Requested: [#%d](%s) %s\n\nThis is your pull request that has changes requested. Please address the feedback and then re-request review from the reviewers who requested changes. \n\nIMPORTANT: After making your changes, use the GitHub interface to re-request review so the reviewers are notified.",
			item.Number, item.URL, item.Title)
	case "mention":
		itemContext = fmt.Sprintf("💬 Mention: [#%d](%s) %s\n\nYou were mentioned in this %s. Please respond to the mention or take appropriate action.",
			item.Number, item.URL, item.Title, item.Details["type"])
	case "unread_comment":
		itemContext = fmt.Sprintf("💭 Unread Comment: [#%d](%s) %s\n\nThere's a new comment on this %s that needs your attention.",
			item.Number, item.URL, item.Title, item.Details["type"])
	case "notification":
		itemContext = fmt.Sprintf("🔔 Notification: %s\n\nYou have an unread notification. Reason: %s",
			item.Title, item.Details["reason"])
	case "failed_workflow":
		prNumbers := ""
		if prs, ok := item.Details["pull_requests"].(string); ok && prs != "" {
			prNumbers = fmt.Sprintf(" (Associated PRs: %s)", prs)
		}
		itemContext = fmt.Sprintf("❌ Failed Workflow: %s%s\n\nThe workflow '%s' has failed. Please investigate and fix the issues.",
			item.Title, prNumbers, item.Details["workflow_name"])
	}

	promptParts = append(promptParts, itemContext)

	// Add continuation context if continuing from previous attempt
	if continueMode && item.AttemptCount > 1 {
		continueContext := fmt.Sprintf("\n⚠️ CONTINUATION: This is attempt %d for this item. The previous attempt may not have fully resolved the issue. Please continue where you left off and ensure the task is completed.",
			item.AttemptCount)
		promptParts = append(promptParts, continueContext)
	}

	// Add agent prompt from configuration
	if m.globalConfig.Agent.Prompt != "" {
		promptParts = append(promptParts, "", m.globalConfig.Agent.Prompt)
	}

	// Add important instructions
	importantPrompt := `IMPORTANT: Focus exclusively on the specific item mentioned above. Complete your work thoroughly and ensure all requirements are fully addressed. If you make changes to the codebase, test them appropriately for your role and document your approach.`
	promptParts = append(promptParts, "", importantPrompt)

	return strings.Join(promptParts, "\n")
}

// getMaxAttempts returns the maximum number of attempts for processing an item
func (m *Monitor) getMaxAttempts() int {
	return m.maxAttempts
}
