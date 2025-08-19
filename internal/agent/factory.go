package agent

import (
	"fmt"

	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
)

// Agent type constants
const (
	AgentTypeDebug      = "debug"
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
	case AgentTypeDebug:
		agent := NewDebugAgent(agentConfig, name, mcpServers)
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

// NewClaudeCodeWithConfig creates a Claude agent from AgentConfig
func NewClaudeCodeWithConfig(agentConfig config.AgentConfig, name string, mcpServers map[string]config.MCPServer) Agent {
	cfg := entrypoint.AgentConfig{Name: name}
	agent := NewClaudeCodeWithMCP(cfg, mcpServers)

	// Claude agent doesn't currently use custom args/env from AgentConfig
	// This could be extended in the future if needed

	return agent
}
