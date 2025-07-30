package monitor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	gitSetup := git.NewSetup(globalConfig.Git, globalConfig.GitHub)

	return &Monitor{
		githubClient: githubClient,
		agent:        selectedAgent,
		config:       monitorConfig,
		globalConfig: globalConfig,
		gitSetup:     gitSetup,
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

	// Get repository info and default branch
	defaultBranch, err := m.githubClient.GetDefaultBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}
	log.Printf("Repository default branch: %s", defaultBranch)

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
			log.Printf("%s: Checking for pending items...", time.Now().Format(time.RFC3339))

			if err := m.checkAndProcess(ctx, username, defaultBranch); err != nil {
				log.Printf("Check failed: %v", err)
			}
		}
	}
}

// checkAndProcess checks for pending items and processes them if found
func (m *Monitor) checkAndProcess(ctx context.Context, username, defaultBranch string) error {
	// Get pending items from GitHub
	pendingItems, err := m.githubClient.GetPendingItems(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get pending items: %w", err)
	}

	if pendingItems.IsEmpty() {
		log.Println("No pending items found")
		return nil
	}

	log.Printf("Found %d pending items that need attention", pendingItems.Count())

	// Format pending items for the prompt
	pendingList := m.formatPendingItems(pendingItems)
	log.Printf("Pending items:\n%s", pendingList)

	// Switch to main branch and pull latest changes
	log.Printf("Switching to %s branch and pulling latest changes...", defaultBranch)
	if err := m.gitSetup.SwitchToMainBranch(ctx, defaultBranch); err != nil {
		return fmt.Errorf("failed to switch to main branch: %w", err)
	}

	// Build the complete prompt
	prompt := m.buildPrompt(pendingList)
	log.Printf("Built prompt for AI agent (length: %d characters)", len(prompt))

	// Execute the AI agent
	runOptions := agent.RunOptions{
		MaxRetries:       m.config.MaxRetries,
		ContinueMode:     false,
		OutputFormat:     "stream-json",
		Verbose:          true,
		DryRun:           m.config.DryRun,
		WorkingDirectory: m.gitSetup.GetWorkingDirectory(),
	}

	if err := m.agent.Run(ctx, prompt, runOptions); err != nil {
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

	// Add important instructions
	importantPrompt := "IMPORTANT: Submit only one Pull Request per iteration. Avoid the '1 PR = 1 commit' approach. Large Pull Requests should be broken down into multiple small, logical commits that each represent a cohesive change."
	promptParts = append(promptParts, "", importantPrompt)

	// Add agent-specific prompt
	if m.globalConfig.Agent.Prompt != "" {
		promptParts = append(promptParts, "", m.globalConfig.Agent.Prompt)
	}

	// Add common prompt
	if m.globalConfig.Agent.CommonPrompt != "" {
		promptParts = append(promptParts, "", m.globalConfig.Agent.CommonPrompt)
	}

	// Add repository-specific prompts if they exist
	workingDir := m.gitSetup.GetWorkingDirectory()

	// Common prompt file
	commonPromptPath := filepath.Join(workingDir, ".autoteam", "common.md")
	if commonPrompt := m.readPromptFile(commonPromptPath); commonPrompt != "" {
		promptParts = append(promptParts, "", commonPrompt)
	}

	// Agent-specific prompt file
	agentPromptPath := filepath.Join(workingDir, ".autoteam", fmt.Sprintf("agent-%s.md", m.globalConfig.Agent.Name))
	if agentPrompt := m.readPromptFile(agentPromptPath); agentPrompt != "" {
		promptParts = append(promptParts, "", agentPrompt)
	}

	return strings.Join(promptParts, "\n")
}

// readPromptFile reads a prompt file and returns its contents
func (m *Monitor) readPromptFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		// File doesn't exist or can't be read, which is fine
		return ""
	}

	trimmed := strings.TrimSpace(string(content))
	if trimmed != "" {
		log.Printf("Loaded prompt from: %s", filePath)
	}

	return trimmed
}
