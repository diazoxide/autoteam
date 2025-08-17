package entrypoint

import (
	"context"
	"testing"

	"autoteam/internal/config"
)

func TestGetHooksForEvent(t *testing.T) {
	hookConfig := &config.HookConfig{
		OnInit: []config.HookCommand{
			{Command: "echo", Args: []string{"init"}},
		},
		OnStart: []config.HookCommand{
			{Command: "echo", Args: []string{"start"}},
		},
		OnStop: []config.HookCommand{
			{Command: "echo", Args: []string{"stop"}},
		},
		OnError: []config.HookCommand{
			{Command: "echo", Args: []string{"error"}},
		},
	}

	tests := []struct {
		name      string
		eventType string
		expected  int
	}{
		{"on_init hooks", "on_init", 1},
		{"on_start hooks", "on_start", 1},
		{"on_stop hooks", "on_stop", 1},
		{"on_error hooks", "on_error", 1},
		{"unknown event", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := GetHooksForEvent(hookConfig, tt.eventType)
			if len(hooks) != tt.expected {
				t.Errorf("GetHooksForEvent() returned %d hooks, expected %d", len(hooks), tt.expected)
			}
		})
	}
}

func TestGetHooksForEvent_NilConfig(t *testing.T) {
	hooks := GetHooksForEvent(nil, "on_init")
	if hooks != nil {
		t.Errorf("GetHooksForEvent() with nil config should return nil, got %v", hooks)
	}
}

func TestExecuteHooks_EmptyHooks(t *testing.T) {
	ctx := context.Background()
	err := ExecuteHooks(ctx, nil, "on_init")
	if err != nil {
		t.Errorf("ExecuteHooks() with nil config should not return error, got %v", err)
	}

	hookConfig := &config.HookConfig{}
	err = ExecuteHooks(ctx, hookConfig, "on_init")
	if err != nil {
		t.Errorf("ExecuteHooks() with empty hooks should not return error, got %v", err)
	}
}

func TestExecuteHooks_SuccessfulExecution(t *testing.T) {
	ctx := context.Background()
	hookConfig := &config.HookConfig{
		OnInit: []config.HookCommand{
			{
				Command:     "echo",
				Args:        []string{"test hook execution"},
				Description: config.StringPtr("Test hook"),
			},
		},
	}

	err := ExecuteHooks(ctx, hookConfig, "on_init")
	if err != nil {
		t.Errorf("ExecuteHooks() should succeed for simple echo command, got error: %v", err)
	}
}

func TestExecuteHooks_WithTimeout(t *testing.T) {
	ctx := context.Background()
	hookConfig := &config.HookConfig{
		OnInit: []config.HookCommand{
			{
				Command: "echo",
				Args:    []string{"test with timeout"},
				Timeout: config.IntPtr(5), // 5 seconds timeout
			},
		},
	}

	err := ExecuteHooks(ctx, hookConfig, "on_init")
	if err != nil {
		t.Errorf("ExecuteHooks() with timeout should succeed for quick command, got error: %v", err)
	}
}

func TestExecuteHooks_ContinueOnError(t *testing.T) {
	ctx := context.Background()
	hookConfig := &config.HookConfig{
		OnInit: []config.HookCommand{
			{
				Command:    "false", // This command always fails
				ContinueOn: config.StringPtr("error"),
			},
			{
				Command: "echo",
				Args:    []string{"this should still run"},
			},
		},
	}

	err := ExecuteHooks(ctx, hookConfig, "on_init")
	if err != nil {
		t.Errorf("ExecuteHooks() with continue_on=error should not fail, got error: %v", err)
	}
}

func TestExecuteHooks_FailOnError(t *testing.T) {
	ctx := context.Background()
	hookConfig := &config.HookConfig{
		OnInit: []config.HookCommand{
			{
				Command:    "false", // This command always fails
				ContinueOn: config.StringPtr("success"),
			},
		},
	}

	err := ExecuteHooks(ctx, hookConfig, "on_init")
	if err == nil {
		t.Errorf("ExecuteHooks() with continue_on=success should fail when command fails")
	}
}

func TestExecuteHooks_WithEnvironment(t *testing.T) {
	ctx := context.Background()
	hookConfig := &config.HookConfig{
		OnInit: []config.HookCommand{
			{
				Command: "sh",
				Args:    []string{"-c", "echo $TEST_VAR"},
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
		},
	}

	err := ExecuteHooks(ctx, hookConfig, "on_init")
	if err != nil {
		t.Errorf("ExecuteHooks() with environment variables should succeed, got error: %v", err)
	}
}

func TestLoadHooks_EmptyEnv(t *testing.T) {
	// Test with no HOOKS_CONFIG environment variable
	hooks, err := LoadHooks()
	if err != nil {
		t.Errorf("LoadHooks() should not error with empty env, got: %v", err)
	}
	if hooks != nil {
		t.Errorf("LoadHooks() should return nil with empty env, got: %v", hooks)
	}
}
