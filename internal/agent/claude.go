package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"autoteam/internal/entrypoint"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// ClaudeAgent implements the Agent interface for Claude Code
type ClaudeAgent struct {
	config     entrypoint.AgentConfig
	binaryPath string
}

// NewClaudeAgent creates a new Claude agent instance
func NewClaudeAgent(cfg entrypoint.AgentConfig) *ClaudeAgent {
	return &ClaudeAgent{
		config:     cfg,
		binaryPath: "claude", // Will be found in PATH after installation
	}
}

// Name returns the agent name
func (c *ClaudeAgent) Name() string {
	return c.config.Name
}

// Type returns the agent type
func (c *ClaudeAgent) Type() string {
	return "claude"
}

// Run executes Claude with the given prompt and options
func (c *ClaudeAgent) Run(ctx context.Context, prompt string, options RunOptions) error {
	lgr := logger.FromContext(ctx)
	if options.DryRun {
		lgr.Info("DRY RUN: Would execute Claude with prompt", zap.Int("prompt_length", len(prompt)))
		return nil
	}

	// Update Claude before running
	if err := c.update(ctx); err != nil {
		lgr.Warn("Failed to update Claude", zap.Error(err))
	}

	// Build the command arguments
	args := c.buildArgs(options)

	var lastErr error
	maxRetries := options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lgr.Info("Claude execution attempt", zap.Int("attempt", attempt), zap.Int("max_retries", maxRetries))

		// Add continue flag when requested or for retry attempts
		currentArgs := args
		if options.ContinueMode || attempt > 1 {
			currentArgs = append(args, "--continue")
		}

		// Execute Claude
		cmd := exec.CommandContext(ctx, c.binaryPath, currentArgs...)
		cmd.Dir = options.WorkingDirectory
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = strings.NewReader(prompt)

		// Set environment variables
		cmd.Env = os.Environ()

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("claude execution failed (attempt %d): %w", attempt, err)
			lgr.Warn("Attempt failed", zap.Int("attempt", attempt), zap.Error(err))

			// Don't retry on context cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Wait before retry (exponential backoff)
			if attempt < maxRetries {
				backoff := time.Duration(attempt) * time.Second
				lgr.Info("Retrying", zap.Duration("backoff", backoff))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
		} else {
			lgr.Info("Claude execution succeeded", zap.Int("attempt", attempt))
			return nil
		}
	}

	return fmt.Errorf("all %d attempts failed, last error: %w", maxRetries, lastErr)
}

// IsAvailable checks if Claude is available
func (c *ClaudeAgent) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, c.binaryPath, "--version")
	return cmd.Run() == nil
}

// Install installs Claude Code via npm
func (c *ClaudeAgent) Install(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Installing Claude Code")

	// Check if npm is available
	if err := exec.CommandContext(ctx, "npm", "--version").Run(); err != nil {
		return fmt.Errorf("npm is not available, cannot install Claude Code: %w", err)
	}

	// Install Claude Code globally
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", "@anthropic-ai/claude-code", "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Claude Code: %w", err)
	}

	lgr.Info("Claude Code installed successfully")
	return nil
}

// Version returns the Claude version
func (c *ClaudeAgent) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, c.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Claude version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// update updates Claude to the latest version
func (c *ClaudeAgent) update(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.binaryPath, "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildArgs builds the command line arguments for Claude
func (c *ClaudeAgent) buildArgs(options RunOptions) []string {
	args := []string{
		"--dangerously-skip-permissions",
	}

	if options.Verbose {
		args = append(args, "--verbose")
	}

	if options.OutputFormat != "" {
		args = append(args, "--output-format", options.OutputFormat)
	} else {
		// Default to stream-json for better parsing
		args = append(args, "--output-format", "stream-json")
	}

	// Always add --print to display output
	args = append(args, "--print")

	return args
}
