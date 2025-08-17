package config

import (
	"reflect"
	"testing"
)

func TestCopyHookConfig(t *testing.T) {
	original := &HookConfig{
		OnInit: []HookCommand{
			{
				Command:     "echo",
				Args:        []string{"init"},
				Env:         map[string]string{"VAR": "value"},
				WorkingDir:  StringPtr("/tmp"),
				Timeout:     IntPtr(30),
				ContinueOn:  StringPtr("always"),
				Description: StringPtr("Test hook"),
			},
		},
		OnStart: []HookCommand{
			{Command: "start", Args: []string{"arg1", "arg2"}},
		},
	}

	copied := copyHookConfig(original)

	// Test that it's a deep copy
	if copied == original {
		t.Error("copyHookConfig() should return a different pointer")
	}

	if !reflect.DeepEqual(copied, original) {
		t.Error("copyHookConfig() should return an equivalent copy")
	}

	// Test that modifying the copy doesn't affect the original
	copied.OnInit[0].Command = "modified"
	if original.OnInit[0].Command == "modified" {
		t.Error("Modifying copy should not affect original")
	}

	// Test with nil
	nilCopy := copyHookConfig(nil)
	if nilCopy != nil {
		t.Error("copyHookConfig(nil) should return nil")
	}
}

func TestCopyHookCommands(t *testing.T) {
	original := []HookCommand{
		{
			Command:     "echo",
			Args:        []string{"test"},
			Env:         map[string]string{"VAR": "value"},
			WorkingDir:  StringPtr("/tmp"),
			Timeout:     IntPtr(30),
			ContinueOn:  StringPtr("error"),
			Description: StringPtr("Test command"),
		},
		{
			Command: "simple",
		},
	}

	copied := copyHookCommands(original)

	if !reflect.DeepEqual(copied, original) {
		t.Error("copyHookCommands() should return an equivalent copy")
	}

	// Test that modifying the copy doesn't affect the original
	copied[0].Command = "modified"
	if original[0].Command == "modified" {
		t.Error("Modifying copy should not affect original")
	}

	// Test with nil
	nilCopy := copyHookCommands(nil)
	if nilCopy != nil {
		t.Error("copyHookCommands(nil) should return nil")
	}
}

func TestMergeHookConfigs(t *testing.T) {
	global := &HookConfig{
		OnInit: []HookCommand{
			{Command: "global-init"},
		},
		OnStart: []HookCommand{
			{Command: "global-start"},
		},
	}

	agentLevel := &HookConfig{
		OnInit: []HookCommand{
			{Command: "agent-init"},
		},
		OnStop: []HookCommand{
			{Command: "agent-stop"},
		},
	}

	// Test agent-level overrides global
	merged := mergeHookConfigs(global, agentLevel)
	if len(merged.OnInit) != 1 || merged.OnInit[0].Command != "agent-init" {
		t.Error("Agent-level hooks should override global hooks")
	}
	if len(merged.OnStop) != 1 || merged.OnStop[0].Command != "agent-stop" {
		t.Error("Agent-level hooks should be preserved")
	}
	if len(merged.OnStart) != 0 {
		t.Error("Global hooks not in agent-level should not be included")
	}

	// Test global only
	merged = mergeHookConfigs(global, nil)
	if !reflect.DeepEqual(merged, global) {
		t.Error("With nil agent-level, should return copy of global")
	}

	// Test agent-level only
	merged = mergeHookConfigs(nil, agentLevel)
	if !reflect.DeepEqual(merged, agentLevel) {
		t.Error("With nil global, should return copy of agent-level")
	}

	// Test both nil
	merged = mergeHookConfigs(nil, nil)
	if merged != nil {
		t.Error("With both nil, should return nil")
	}
}

func TestAgentGetEffectiveSettings_WithHooks(t *testing.T) {
	globalSettings := AgentSettings{
		Hooks: &HookConfig{
			OnInit: []HookCommand{
				{Command: "global-init"},
			},
		},
	}

	agent := Agent{
		Name:   "test-agent",
		Prompt: "test prompt",
		Settings: &AgentSettings{
			Hooks: &HookConfig{
				OnStart: []HookCommand{
					{Command: "agent-start"},
				},
			},
		},
	}

	effective := agent.GetEffectiveSettings(globalSettings)

	// Agent-level hooks should override global hooks
	if effective.Hooks == nil {
		t.Fatal("Effective settings should have hooks")
	}

	if len(effective.Hooks.OnStart) != 1 || effective.Hooks.OnStart[0].Command != "agent-start" {
		t.Error("Agent-level hooks should be in effective settings")
	}

	// Global hooks should not be included when agent has hooks
	if len(effective.Hooks.OnInit) != 0 {
		t.Error("Global hooks should not override agent-level hooks configuration")
	}
}

func TestAgentGetEffectiveSettings_InheritGlobalHooks(t *testing.T) {
	globalSettings := AgentSettings{
		Hooks: &HookConfig{
			OnInit: []HookCommand{
				{Command: "global-init"},
			},
		},
	}

	agent := Agent{
		Name:     "test-agent",
		Prompt:   "test prompt",
		Settings: nil, // No agent-specific settings
	}

	effective := agent.GetEffectiveSettings(globalSettings)

	// Should inherit global hooks
	if effective.Hooks == nil {
		t.Fatal("Effective settings should inherit global hooks")
	}

	if len(effective.Hooks.OnInit) != 1 || effective.Hooks.OnInit[0].Command != "global-init" {
		t.Error("Should inherit global hooks when no agent-level hooks")
	}
}
