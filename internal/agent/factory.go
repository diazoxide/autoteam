package agent

import (
	"fmt"

	"autoteam/internal/worker"
)

// Agent type constants
const (
	AgentTypeDebug      = "debug"
	AgentTypeClaudeCode = "claude"
	AgentTypeQwenCode   = "qwen"
	AgentTypeGeminiCli  = "gemini"
)

// CreateAgent creates an agent based on configuration
func CreateAgent(agentConfig AgentConfig, name string, mcpServers map[string]worker.MCPServer) (Agent, error) {
	switch agentConfig.Type {
	case AgentTypeClaudeCode:
		agent := NewClaudeCode(name, agentConfig.Args, agentConfig.Env, mcpServers)
		return agent, nil
	case AgentTypeDebug:
		agent := NewDebugAgent(name, agentConfig.Args, agentConfig.Env, mcpServers)
		return agent, nil
	case AgentTypeQwenCode:
		agent := NewQwenCode(name, agentConfig.Args, agentConfig.Env, mcpServers)
		return agent, nil
	case AgentTypeGeminiCli:
		agent := NewGeminiCli(name, agentConfig.Args, agentConfig.Env, mcpServers)
		return agent, nil
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agentConfig.Type)
	}
}
