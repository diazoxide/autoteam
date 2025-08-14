package agent

import (
	"fmt"

	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
)

// Agent type constants
const (
	AgentTypeClaudeCode = "claude"
	AgentTypeQwenCode   = "qwen"
	AgentTypeGeminiCli  = "gemini"
)

// CreateAgent creates an agent based on configuration
func CreateAgent(agentConfig config.AgentConfig, name string, mcpServers map[string]config.MCPServer) (Agent, error) {
	switch agentConfig.Type {
	case AgentTypeClaudeCode:
		agent := NewClaudeCodeWithConfig(agentConfig, name, mcpServers)
		return agent, nil
	case AgentTypeQwenCode:
		agent := NewQwenCodeWithConfig(agentConfig, name, mcpServers)
		return agent, nil
	case AgentTypeGeminiCli:
		agent := NewGeminiCliWithConfig(agentConfig, name, mcpServers)
		return agent, nil
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agentConfig.Type)
	}
}

// CreateAgentFromEntrypointConfig creates an agent from entrypoint configuration (legacy support)
func CreateAgentFromEntrypointConfig(cfg entrypoint.AgentConfig, mcpServers map[string]config.MCPServer) Agent {
	// Default to Claude for backward compatibility
	return NewClaudeCodeWithMCP(cfg, mcpServers)
}

// NewClaudeCodeWithConfig creates a Claude agent from AgentConfig
func NewClaudeCodeWithConfig(agentConfig config.AgentConfig, name string, mcpServers map[string]config.MCPServer) Agent {
	cfg := entrypoint.AgentConfig{Name: name}
	agent := NewClaudeCodeWithMCP(cfg, mcpServers)

	// Claude agent doesn't currently use custom args/env from AgentConfig
	// This could be extended in the future if needed

	return agent
}

// GetSupportedAgentTypes returns a list of supported agent types
func GetSupportedAgentTypes() []string {
	return []string{AgentTypeClaudeCode, AgentTypeQwenCode}
}

// IsValidAgentType checks if the given agent type is supported
func IsValidAgentType(agentType string) bool {
	supportedTypes := GetSupportedAgentTypes()
	for _, supportedType := range supportedTypes {
		if agentType == supportedType {
			return true
		}
	}
	return false
}
