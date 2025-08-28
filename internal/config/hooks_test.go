package config

import (
	"testing"

	"autoteam/internal/worker"
)

func TestCopyHookConfig(t *testing.T) {
	// Since copyHookConfig was moved to worker package, we need to test via GetEffectiveSettings
	// This test is no longer directly testable from config package
	// Testing is now done through integration with GetEffectiveSettings
}

func TestCopyHookCommands(t *testing.T) {
	// This test is also moved to worker package functionality
	// Testing is now done through integration
}

func TestMergeHookConfigs(t *testing.T) {
	// This test is also moved to worker package functionality
	// Testing is now done through integration
}

func TestWorkerGetEffectiveSettings_WithHooks(t *testing.T) {
	globalSettings := worker.WorkerSettings{
		Hooks: &worker.HookConfig{
			OnInit: []worker.HookCommand{
				{Command: "global-init"},
			},
		},
	}

	w := worker.Worker{
		Name:   "test-worker",
		Prompt: "test prompt",
		Settings: &worker.WorkerSettings{
			Hooks: &worker.HookConfig{
				OnStart: []worker.HookCommand{
					{Command: "worker-start"},
				},
			},
		},
	}

	effective := w.GetEffectiveSettings(globalSettings)

	// Worker-level hooks should override global hooks
	if effective.Hooks == nil {
		t.Fatal("Effective settings should have hooks")
	}

	if len(effective.Hooks.OnStart) != 1 || effective.Hooks.OnStart[0].Command != "worker-start" {
		t.Error("Worker-level hooks should be in effective settings")
	}

	// Global hooks should not be included when worker has hooks
	if len(effective.Hooks.OnInit) != 0 {
		t.Error("Global hooks should not override worker-level hooks configuration")
	}
}

func TestWorkerGetEffectiveSettings_InheritGlobalHooks(t *testing.T) {
	globalSettings := worker.WorkerSettings{
		Hooks: &worker.HookConfig{
			OnInit: []worker.HookCommand{
				{Command: "global-init"},
			},
		},
	}

	w := worker.Worker{
		Name:     "test-worker",
		Prompt:   "test prompt",
		Settings: nil, // No worker-specific settings
	}

	effective := w.GetEffectiveSettings(globalSettings)

	// Should inherit global hooks
	if effective.Hooks == nil {
		t.Fatal("Effective settings should inherit global hooks")
	}

	if len(effective.Hooks.OnInit) != 1 || effective.Hooks.OnInit[0].Command != "global-init" {
		t.Error("Should inherit global hooks when no worker-level hooks")
	}
}
