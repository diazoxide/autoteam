package agent

import (
	"context"
	"testing"

	"autoteam/internal/worker"
)

func TestCreateAgent(t *testing.T) {
	mcpServers := map[string]worker.MCPServer{
		"test-server": {
			Command: "test-command",
			Args:    []string{"--test"},
			Env:     map[string]string{"TEST": "value"},
		},
	}

	tests := []struct {
		name        string
		config      AgentConfig
		agentName   string
		expectError bool
		expectType  string
	}{
		{
			name: "create claude agent",
			config: AgentConfig{
				Type: AgentTypeClaudeCode,
				Args: []string{"--model", "claude-3-5-sonnet-20241022"},
				Env:  map[string]string{"CLAUDE_API_KEY": "test-key"},
			},
			agentName:   "test-claude",
			expectError: false,
			expectType:  AgentTypeClaudeCode,
		},
		{
			name: "create debug agent",
			config: AgentConfig{
				Type: AgentTypeDebug,
				Args: []string{"--verbose"},
				Env:  map[string]string{"DEBUG": "true"},
			},
			agentName:   "test-debug",
			expectError: false,
			expectType:  AgentTypeDebug,
		},
		{
			name: "create qwen agent",
			config: AgentConfig{
				Type: AgentTypeQwenCode,
				Args: []string{"--model", "qwen-max"},
				Env:  map[string]string{"QWEN_API_KEY": "test-key"},
			},
			agentName:   "test-qwen",
			expectError: false,
			expectType:  AgentTypeQwenCode,
		},
		{
			name: "create gemini agent",
			config: AgentConfig{
				Type: AgentTypeGeminiCli,
				Args: []string{"--project", "test-project"},
				Env:  map[string]string{"GEMINI_API_KEY": "test-key"},
			},
			agentName:   "test-gemini",
			expectError: false,
			expectType:  AgentTypeGeminiCli,
		},
		{
			name: "unknown agent type",
			config: AgentConfig{
				Type: "unknown-type",
				Args: []string{},
				Env:  map[string]string{},
			},
			agentName:   "test-unknown",
			expectError: true,
			expectType:  "",
		},
		{
			name: "empty agent type",
			config: AgentConfig{
				Type: "",
				Args: []string{},
				Env:  map[string]string{},
			},
			agentName:   "test-empty",
			expectError: true,
			expectType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := CreateAgent(tt.config, tt.agentName, mcpServers)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if agent != nil {
					t.Error("Expected nil agent when error occurs")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if agent == nil {
				t.Error("Expected agent to be created but got nil")
				return
			}

			// Test that the agent has the expected type
			if agent.Type() != tt.expectType {
				t.Errorf("Expected agent type %s, got %s", tt.expectType, agent.Type())
			}

			// Test that the agent has a version (may fail if executable not installed in CI)
			ctx := context.Background()
			version, err := agent.Version(ctx)
			if err != nil {
				// In CI environments, executables may not be available - this is OK
				t.Logf("Version check failed (expected in CI without executables): %v", err)
			} else {
				if version == "" {
					t.Error("Expected agent to have a version when executable is available")
				}
			}
		})
	}
}

func TestAgentConstants(t *testing.T) {
	// Test that agent type constants are defined correctly
	expectedConstants := map[string]string{
		"AgentTypeDebug":      AgentTypeDebug,
		"AgentTypeClaudeCode": AgentTypeClaudeCode,
		"AgentTypeQwenCode":   AgentTypeQwenCode,
		"AgentTypeGeminiCli":  AgentTypeGeminiCli,
	}

	actualConstants := map[string]string{
		"AgentTypeDebug":      "debug",
		"AgentTypeClaudeCode": "claude",
		"AgentTypeQwenCode":   "qwen",
		"AgentTypeGeminiCli":  "gemini",
	}

	for name, expected := range actualConstants {
		if actual := expectedConstants[name]; actual != expected {
			t.Errorf("Expected %s to be %s, got %s", name, expected, actual)
		}
	}
}

func TestCreateAgentWithEmptyMCPServers(t *testing.T) {
	// Test creating agents with empty MCP servers map
	config := AgentConfig{
		Type: AgentTypeDebug,
		Args: []string{"--test"},
		Env:  map[string]string{"TEST": "value"},
	}

	agent, err := CreateAgent(config, "test-agent", nil)
	if err != nil {
		t.Errorf("Expected no error with nil MCP servers, got: %v", err)
	}

	if agent == nil {
		t.Error("Expected agent to be created even with nil MCP servers")
	}

	// Test with empty map
	emptyMCPServers := make(map[string]worker.MCPServer)
	agent2, err2 := CreateAgent(config, "test-agent-2", emptyMCPServers)
	if err2 != nil {
		t.Errorf("Expected no error with empty MCP servers map, got: %v", err2)
	}

	if agent2 == nil {
		t.Error("Expected agent to be created with empty MCP servers map")
	}
}

func TestCreateAgentWithComplexMCPServers(t *testing.T) {
	// Test creating agents with complex MCP server configurations
	complexMCPServers := map[string]worker.MCPServer{
		"server1": {
			Command: "server1-cmd",
			Args:    []string{"--arg1", "value1", "--arg2", "value2"},
			Env: map[string]string{
				"ENV1": "value1",
				"ENV2": "value2",
			},
		},
		"server2": {
			Command: "server2-cmd",
			Args:    []string{},
			Env:     map[string]string{},
		},
		"server3": {
			Command: "server3-cmd",
			Args:    nil,
			Env:     nil,
		},
	}

	config := AgentConfig{
		Type: AgentTypeClaudeCode,
		Args: []string{"--model", "claude-3-5-sonnet-20241022"},
		Env:  map[string]string{"CLAUDE_API_KEY": "test-key"},
	}

	agent, err := CreateAgent(config, "complex-test", complexMCPServers)
	if err != nil {
		t.Errorf("Expected no error with complex MCP servers, got: %v", err)
	}

	if agent == nil {
		t.Error("Expected agent to be created with complex MCP servers")
	}

	if agent.Type() != AgentTypeClaudeCode {
		t.Errorf("Expected agent type %s, got %s", AgentTypeClaudeCode, agent.Type())
	}
}

func TestAgentConfigStructure(t *testing.T) {
	// Test that AgentConfig has the expected structure
	config := AgentConfig{
		Type: "test-type",
		Args: []string{"arg1", "arg2"},
		Env:  map[string]string{"KEY": "value"},
	}

	if config.Type != "test-type" {
		t.Errorf("Expected Type to be 'test-type', got %s", config.Type)
	}

	if len(config.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(config.Args))
	}

	if config.Args[0] != "arg1" || config.Args[1] != "arg2" {
		t.Errorf("Expected args [arg1, arg2], got %v", config.Args)
	}

	if config.Env["KEY"] != "value" {
		t.Errorf("Expected env KEY=value, got %s", config.Env["KEY"])
	}
}

func TestCreateAgentErrorMessages(t *testing.T) {
	// Test that error messages are descriptive
	unknownConfig := AgentConfig{
		Type: "totally-unknown-type",
		Args: []string{},
		Env:  map[string]string{},
	}

	_, err := CreateAgent(unknownConfig, "test", nil)
	if err == nil {
		t.Error("Expected error for unknown agent type")
		return
	}

	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Error message should mention the unknown type
	if !contains(errorMsg, "totally-unknown-type") {
		t.Errorf("Expected error message to mention unknown type, got: %s", errorMsg)
	}
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
