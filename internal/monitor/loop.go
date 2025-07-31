package monitor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/entrypoint"
	"autoteam/internal/git"
	"autoteam/internal/github"
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
	log.Printf("Starting monitor with check interval: %v", m.config.CheckInterval)

	// Get authenticated user info
	user, err := m.githubClient.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}

	username := user.GetLogin()
	log.Printf("Authenticated as GitHub user: %s", username)

	// TODO: Multi-repository support - get default branch per repository
	// For now, use "main" as default branch for multi-repo compatibility
	defaultBranch := "main"
	log.Printf("Using default branch: %s (multi-repo support pending)", defaultBranch)

	// Log all matched repositories before starting monitoring
	log.Println("Initializing repository discovery...")
	filteredRepos, err := m.githubClient.GetFilteredRepositories(ctx, username)
	if err != nil {
		log.Printf("Warning: failed to get filtered repositories during initialization: %v", err)
		// Show patterns as fallback
		if m.globalConfig.Repositories != nil {
			log.Printf("Repository patterns configured: include=%v, exclude=%v",
				m.globalConfig.Repositories.Include, m.globalConfig.Repositories.Exclude)
		}
	} else {
		if len(filteredRepos) == 0 {
			log.Println("No repositories found matching the configured patterns")
			if m.globalConfig.Repositories != nil {
				log.Printf("Configured patterns: include=%v, exclude=%v",
					m.globalConfig.Repositories.Include, m.globalConfig.Repositories.Exclude)
			}
		} else {
			log.Printf("Found %d repositories matching configured patterns:", len(filteredRepos))
			for i, repo := range filteredRepos {
				log.Printf("  %d. %s (%s)", i+1, repo.FullName, repo.URL)
			}
		}
	}

	// Run initial check
	if err := m.checkAndProcess(ctx, username, defaultBranch); err != nil {
		log.Printf("Initial check failed: %v", err)
	}

	// Start periodic monitoring
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Monitor shutting down due to context cancellation")
			return ctx.Err()

		case <-ticker.C:
			// Build repository list for logging by resolving patterns to actual repos
			var repoLog string
			filteredRepos, err := m.githubClient.GetFilteredRepositories(ctx, username)
			if err != nil {
				log.Printf("Warning: failed to get filtered repositories for logging: %v", err)
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
			log.Printf("%s: Checking for pending items...%s", time.Now().Format(time.RFC3339), repoLog)

			if err := m.checkAndProcess(ctx, username, defaultBranch); err != nil {
				log.Printf("Check failed: %v", err)
			}
		}
	}
}

// checkAndProcess implements the single item processing workflow
func (m *Monitor) checkAndProcess(ctx context.Context, username, defaultBranch string) error {
	// Clean up old failure records
	if err := m.stateManager.CleanupOldFailures(); err != nil {
		log.Printf("Warning: failed to cleanup old failures: %v", err)
	}

	// Check if we have an item currently being processed
	currentItem := m.stateManager.GetCurrentItem()
	if currentItem != nil {
		log.Printf("Continuing with item: %s #%d (%s) - Attempt %d",
			currentItem.Type, currentItem.Number, currentItem.Title, currentItem.AttemptCount)

		// Check if current item was resolved since last attempt
		result, err := m.resolutionDetector.CheckItemResolution(ctx, currentItem, username)
		if err != nil {
			log.Printf("Warning: failed to check item resolution: %v", err)
		} else {
			LogResolutionResult(result, currentItem)

			if result == ItemNotFound {
				// Item was resolved, clear it and continue to next item
				if err := m.stateManager.ClearCurrentItem(); err != nil {
					log.Printf("Warning: failed to clear resolved item: %v", err)
				}
				log.Println("✅ Item resolved successfully! Selecting next item...")
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
	// Get all pending items from GitHub
	pendingItems, err := m.githubClient.GetPendingItems(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get pending items: %w", err)
	}

	if pendingItems.IsEmpty() {
		log.Println("No pending items found")
		return nil
	}

	log.Printf("Found %d total pending items", pendingItems.Count())

	// Select the highest priority item
	selectedItem := m.itemPrioritizer.SelectNextItem(pendingItems)
	if selectedItem == nil {
		log.Println("No suitable item to process (all may be in cooldown)")
		return nil
	}

	// Create processing item and set as current
	var processingItem *ProcessingItem
	switch selectedItem.Type {
	case "review_request", "assigned_pr", "pr_with_changes":
		processingItem = CreateProcessingItemFromPR(github.PullRequestInfo{
			Number:     selectedItem.Number,
			Repository: selectedItem.Repository,
			Title:      selectedItem.Title,
			URL:        selectedItem.URL,
			Author:     selectedItem.Author,
		}, selectedItem.Type)
	case "assigned_issue":
		processingItem = CreateProcessingItemFromIssue(github.IssueInfo{
			Number:     selectedItem.Number,
			Repository: selectedItem.Repository,
			Title:      selectedItem.Title,
			URL:        selectedItem.URL,
			Author:     selectedItem.Author,
		}, selectedItem.Type)
	default:
		return fmt.Errorf("unknown item type: %s", selectedItem.Type)
	}

	// Set as current item
	if err := m.stateManager.SetCurrentItem(processingItem); err != nil {
		return fmt.Errorf("failed to set current item: %w", err)
	}

	// Process the selected item
	return m.processItem(ctx, processingItem, defaultBranch, false)
}

// processItem processes a single item
func (m *Monitor) processItem(ctx context.Context, item *ProcessingItem, globalDefaultBranch string, continueMode bool) error {
	log.Printf("Processing %s #%d: %s (attempt %d)", item.Type, item.Number, item.Title, item.AttemptCount)

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
		log.Printf("Warning: failed to get default branch for %s, using %s: %v", item.Repository, globalDefaultBranch, err)
		defaultBranch = globalDefaultBranch
	}

	// Git state management: Only reset for new items, preserve state for continuations
	if !continueMode {
		// New item: Fresh git state (fetch + reset to main)
		log.Printf("New item: Switching to %s branch and resetting to clean state for repository %s...", defaultBranch, item.Repository)
		if err := m.gitSetup.SwitchToMainBranch(ctx, item.Repository, defaultBranch); err != nil {
			return fmt.Errorf("failed to switch to main branch for repository %s: %w", item.Repository, err)
		}
	} else {
		// Continuation: Keep existing git state, don't reset
		log.Printf("Continuing item: Preserving current git state (no reset)")
	}

	// Build the prompt for this specific item
	prompt := m.buildItemPrompt(item, continueMode)
	log.Printf("Built prompt for item (length: %d characters)", len(prompt))

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
		log.Printf("Warning: failed to increment attempt count: %v", err)
	}

	if err := m.agent.Run(ctx, prompt, runOptions); err != nil {
		log.Printf("Agent execution failed: %v", err)

		// Record failure if max attempts reached
		maxAttempts := m.getMaxAttempts()
		if item.AttemptCount >= maxAttempts {
			itemKey := GetItemKeyFromProcessingItem(item)
			if err := m.stateManager.RecordFailure(itemKey); err != nil {
				log.Printf("Warning: failed to record failure: %v", err)
			}

			// Clear current item after max attempts
			if err := m.stateManager.ClearCurrentItem(); err != nil {
				log.Printf("Warning: failed to clear failed item: %v", err)
			}

			log.Printf("Max attempts reached for %s #%d, moving to cooldown", item.Type, item.Number)
		}

		return fmt.Errorf("agent execution failed: %w", err)
	}

	log.Printf("Agent execution completed successfully at %s", time.Now().Format(time.RFC3339))
	return nil
}

// formatPendingItems formats pending items into a readable string
func (m *Monitor) formatPendingItems(items *github.PendingItems) string {
	var sections []string

	// Review Requests
	if len(items.ReviewRequests) > 0 {
		sections = append(sections, "📥 Review Requests:")
		for _, pr := range items.ReviewRequests {
			sections = append(sections, fmt.Sprintf("- [#%d](%s) %s", pr.Number, pr.URL, pr.Title))
		}
		sections = append(sections, "")
	}

	// Assigned PRs
	if len(items.AssignedPRs) > 0 {
		sections = append(sections, "🧷 Assigned PRs:")
		for _, pr := range items.AssignedPRs {
			sections = append(sections, fmt.Sprintf("- [#%d](%s) %s", pr.Number, pr.URL, pr.Title))
		}
		sections = append(sections, "")
	}

	// Assigned Issues
	if len(items.AssignedIssues) > 0 {
		sections = append(sections, "🚧 Assigned Issues (no PR):")
		for _, issue := range items.AssignedIssues {
			sections = append(sections, fmt.Sprintf("- [#%d](%s) %s", issue.Number, issue.URL, issue.Title))
		}
		sections = append(sections, "")
	}

	// PRs with Changes Requested
	if len(items.PRsWithChanges) > 0 {
		sections = append(sections, "🛠 My PRs with Changes Requested:")
		for _, pr := range items.PRsWithChanges {
			sections = append(sections, fmt.Sprintf("- [#%d](%s) %s", pr.Number, pr.URL, pr.Title))
		}
		sections = append(sections, "")
	}

	return strings.Join(sections, "\n")
}

// buildPrompt builds the complete prompt for the AI agent
func (m *Monitor) buildPrompt(pendingList string) string {
	var promptParts []string

	// Add pending items
	promptParts = append(promptParts, pendingList)

	// Add consolidated agent prompt (already includes agent-specific prompt, common prompt, and collaborators list)
	if m.globalConfig.Agent.Prompt != "" {
		promptParts = append(promptParts, "", m.globalConfig.Agent.Prompt)
	}

	// Add important instructions at the end
	importantPrompt := `IMPORTANT GUIDELINES:
- Work systematically and thoroughly on each assigned item
- Provide clear documentation of your work and decisions
- Collaborate effectively with team members when needed
- Follow project standards and best practices for your role
- Communicate progress and blockers transparently`
	promptParts = append(promptParts, "", importantPrompt)

	return strings.Join(promptParts, "\n")
}

// buildItemPrompt builds a prompt for a specific item
func (m *Monitor) buildItemPrompt(item *ProcessingItem, continueMode bool) string {
	var promptParts []string

	// Add item-specific context
	var itemContext string
	switch item.Type {
	case "review_request":
		itemContext = fmt.Sprintf("📥 Review Request: [#%d](%s) %s\n\nPlease review this pull request and provide feedback. IMPORTANT: Make sure you submit your review and mark it as reviewed (approve, request changes, or comment) - don't just leave comments without submitting the review.",
			item.Number, item.URL, item.Title)
	case "assigned_pr":
		itemContext = fmt.Sprintf("🧷 Assigned PR: [#%d](%s) %s\n\nThis pull request is assigned to you. Please work on it.",
			item.Number, item.URL, item.Title)
	case "assigned_issue":
		itemContext = fmt.Sprintf("🚧 Assigned Issue: [#%d](%s) %s\n\nThis issue is assigned to you. Please address it.",
			item.Number, item.URL, item.Title)
	case "pr_with_changes":
		itemContext = fmt.Sprintf("🛠 PR with Changes Requested: [#%d](%s) %s\n\nThis is your pull request that has changes requested. Please address the feedback and then re-request review from the reviewers who requested changes. IMPORTANT: After making your changes, use the GitHub interface to re-request review so the reviewers are notified.",
			item.Number, item.URL, item.Title)
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
