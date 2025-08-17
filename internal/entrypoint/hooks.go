package entrypoint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"autoteam/internal/config"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// ExecuteHooks executes hooks for a specific lifecycle event
func ExecuteHooks(ctx context.Context, hookConfig *config.HookConfig, hookType string) error {
	hooks := GetHooksForEvent(hookConfig, hookType)
	if len(hooks) == 0 {
		return nil
	}

	lgr := logger.FromContext(ctx)
	lgr.Info("Executing container hooks",
		zap.String("hook_type", hookType),
		zap.Int("hook_count", len(hooks)))

	for i, hook := range hooks {
		if err := executeHook(ctx, hookType, i+1, hook); err != nil {
			lgr.Error("Hook execution failed",
				zap.String("hook_type", hookType),
				zap.Int("hook_index", i+1),
				zap.Error(err))

			// Check continue_on setting
			continueOn := "error" // default
			if hook.ContinueOn != nil {
				continueOn = *hook.ContinueOn
			}

			switch continueOn {
			case "always":
				lgr.Info("Continuing despite hook failure (continue_on: always)")
				continue
			case "success":
				return fmt.Errorf("hook %d failed and continue_on is 'success': %w", i+1, err)
			case "error":
				lgr.Info("Continuing after hook failure (continue_on: error)")
				continue
			default:
				return fmt.Errorf("hook %d failed: %w", i+1, err)
			}
		}
	}

	lgr.Info("All hooks executed successfully", zap.String("hook_type", hookType))
	return nil
}

// executeHook executes a single hook command
func executeHook(ctx context.Context, hookType string, index int, hook config.HookCommand) error {
	lgr := logger.FromContext(ctx)

	description := fmt.Sprintf("Hook %d", index)
	if hook.Description != nil {
		description = *hook.Description
	}

	lgr.Info("Executing hook",
		zap.String("hook_type", hookType),
		zap.Int("index", index),
		zap.String("description", description),
		zap.String("command", hook.Command),
		zap.Strings("args", hook.Args))

	// Set working directory - default to container working directory
	workingDir := "/opt/autoteam"
	if hook.WorkingDir != nil {
		workingDir = *hook.WorkingDir
	}

	// Set timeout
	timeout := 30 * time.Second // default timeout
	if hook.Timeout != nil {
		timeout = time.Duration(*hook.Timeout) * time.Second
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(timeoutCtx, hook.Command, hook.Args...)
	cmd.Dir = workingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range hook.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture output
	output, err := cmd.CombinedOutput()

	if err != nil {
		lgr.Error("Hook command failed",
			zap.String("hook_type", hookType),
			zap.Int("index", index),
			zap.String("command", hook.Command),
			zap.String("working_dir", workingDir),
			zap.String("output", string(output)),
			zap.Error(err))
		return fmt.Errorf("hook command failed: %w\nOutput: %s", err, string(output))
	}

	if len(output) > 0 {
		lgr.Info("Hook command output",
			zap.String("hook_type", hookType),
			zap.Int("index", index),
			zap.String("output", strings.TrimSpace(string(output))))
	}

	lgr.Info("Hook executed successfully",
		zap.String("hook_type", hookType),
		zap.Int("index", index))

	return nil
}

// GetHooksForEvent retrieves hooks for a specific event from hook configuration
func GetHooksForEvent(hookConfig *config.HookConfig, eventType string) []config.HookCommand {
	if hookConfig == nil {
		return nil
	}

	switch eventType {
	case "on_init":
		return hookConfig.OnInit
	case "on_start":
		return hookConfig.OnStart
	case "on_stop":
		return hookConfig.OnStop
	case "on_error":
		return hookConfig.OnError
	default:
		return nil
	}
}
