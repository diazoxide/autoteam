package entrypoint

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"autoteam/internal/config"
	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for the entrypoint
type Config struct {
	Agent        AgentConfig                 `yaml:"agent"`
	TeamName     string                      `yaml:"team_name"`
	Monitoring   MonitoringConfig            `yaml:"monitoring"`
	Dependencies DependenciesConfig          `yaml:"dependencies"`
	MCPServers   map[string]config.MCPServer `yaml:"mcp_servers,omitempty"`
	Hooks        *config.HookConfig          `yaml:"hooks,omitempty"`
	Flow         []config.FlowStep           `yaml:"flow"`
	Debug        bool                        `yaml:"debug"`
}

// AgentConfig contains AI agent configuration
type AgentConfig struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Prompt string `yaml:"prompt"`
}

// MonitoringConfig contains monitoring loop configuration
type MonitoringConfig struct {
	CheckInterval time.Duration `yaml:"check_interval"`
	MaxRetries    int           `yaml:"max_retries"`
}

// DependenciesConfig contains dependency installation configuration
type DependenciesConfig struct {
	InstallDeps bool `yaml:"install_deps"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Agent configuration
	cfg.Agent.Name = os.Getenv("AGENT_NAME")
	if cfg.Agent.Name == "" {
		return nil, fmt.Errorf("AGENT_NAME environment variable is required")
	}

	cfg.Agent.Type = getEnvOrDefault("AGENT_TYPE", "claude")
	cfg.Agent.Prompt = os.Getenv("AGENT_PROMPT")

	// Team configuration
	cfg.TeamName = getEnvOrDefault("TEAM_NAME", "autoteam")

	// Monitoring configuration
	checkInterval := getEnvOrDefault("CHECK_INTERVAL", "60")
	interval, err := strconv.Atoi(checkInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid CHECK_INTERVAL: %w", err)
	}
	cfg.Monitoring.CheckInterval = time.Duration(interval) * time.Second

	maxRetries := getEnvOrDefault("MAX_RETRIES", "100")
	cfg.Monitoring.MaxRetries, err = strconv.Atoi(maxRetries)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_RETRIES: %w", err)
	}

	// Dependencies configuration
	cfg.Dependencies.InstallDeps = getEnvOrDefault("INSTALL_DEPS", "false") == "true"

	// MCP servers configuration
	cfg.MCPServers, err = LoadMCPServers()
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP servers: %w", err)
	}

	// Hooks configuration
	cfg.Hooks, err = LoadHooks()
	if err != nil {
		return nil, fmt.Errorf("failed to load hooks: %w", err)
	}

	// Flow configuration
	cfg.Flow, err = LoadFlow()
	if err != nil {
		return nil, fmt.Errorf("failed to load flow: %w", err)
	}

	// Debug configuration
	cfg.Debug = getEnvOrDefault("DEBUG", "false") == "true"

	return cfg, nil
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if c.Monitoring.CheckInterval < time.Second {
		return fmt.Errorf("check interval must be at least 1 second")
	}
	if c.Monitoring.MaxRetries < 1 {
		return fmt.Errorf("max retries must be at least 1")
	}
	return nil
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return &cfg, nil
}

// LoadMCPServers loads MCP server configuration from environment variables
func LoadMCPServers() (map[string]config.MCPServer, error) {
	mcpServersJSON := os.Getenv("MCP_SERVERS")
	if mcpServersJSON == "" {
		return nil, nil // No MCP servers configured
	}

	var mcpServers map[string]config.MCPServer
	if err := json.Unmarshal([]byte(mcpServersJSON), &mcpServers); err != nil {
		return nil, fmt.Errorf("failed to parse MCP_SERVERS JSON: %w", err)
	}

	return mcpServers, nil
}

// LoadHooks loads hook configuration from environment variables
func LoadHooks() (*config.HookConfig, error) {
	hooksJSON := os.Getenv("HOOKS_CONFIG")
	if hooksJSON == "" {
		return nil, nil // No hooks configured
	}

	var hooks config.HookConfig
	if err := json.Unmarshal([]byte(hooksJSON), &hooks); err != nil {
		return nil, fmt.Errorf("failed to parse HOOKS_CONFIG JSON: %w", err)
	}

	return &hooks, nil
}

// LoadFlow loads flow configuration from environment variables
// Deprecated: Use CLI flags instead
func LoadFlow() ([]config.FlowStep, error) {
	flowJSON := os.Getenv("FLOW_CONFIG")
	if flowJSON == "" {
		return nil, fmt.Errorf("FLOW_CONFIG environment variable is not set or empty")
	}

	var flow []config.FlowStep
	if err := json.Unmarshal([]byte(flowJSON), &flow); err != nil {
		return nil, fmt.Errorf("failed to parse FLOW_CONFIG JSON: %w", err)
	}

	if len(flow) == 0 {
		return nil, fmt.Errorf("flow configuration is empty")
	}

	return flow, nil
}
