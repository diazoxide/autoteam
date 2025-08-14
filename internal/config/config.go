package config

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Default configuration constants
const (
	DefaultTeamName = "autoteam"
)

type Config struct {
	Agents     []Agent                           `yaml:"agents"`
	Services   map[string]map[string]interface{} `yaml:"services,omitempty"`
	Settings   AgentSettings                     `yaml:"settings"`
	MCPServers map[string]MCPServer              `yaml:"mcp_servers,omitempty"`
}

type Agent struct {
	Name       string               `yaml:"name"`
	Prompt     string               `yaml:"prompt"`
	Enabled    *bool                `yaml:"enabled,omitempty"`
	Settings   *AgentSettings       `yaml:"settings,omitempty"`
	MCPServers map[string]MCPServer `yaml:"mcp_servers,omitempty"`
}

type AgentSettings struct {
	CheckInterval *int                   `yaml:"check_interval,omitempty"`
	TeamName      *string                `yaml:"team_name,omitempty"`
	InstallDeps   *bool                  `yaml:"install_deps,omitempty"`
	CommonPrompt  *string                `yaml:"common_prompt,omitempty"`
	MaxAttempts   *int                   `yaml:"max_attempts,omitempty"`
	Service       map[string]interface{} `yaml:"service,omitempty"`
	MCPServers    map[string]MCPServer   `yaml:"mcp_servers,omitempty"`
	// Per-agent First Layer override
	CollectorAgent *AgentConfig `yaml:"collector_agent,omitempty"`
	// Per-agent Second Layer override
	ExecutionAgent *AgentConfig `yaml:"execution_agent,omitempty"`
}

// MCPServer represents a Model Context Protocol server configuration
type MCPServer struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// AgentConfig represents unified agent configuration structure
type AgentConfig struct {
	Type   string            `yaml:"type"`
	Args   []string          `yaml:"args,omitempty"`
	Env    map[string]string `yaml:"env,omitempty"`
	Prompt *string           `yaml:"prompt,omitempty"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	setDefaults(&config)

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	// Count enabled agents
	enabledCount := 0
	for _, agent := range config.Agents {
		if agent.IsEnabled() {
			enabledCount++
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("at least one agent must be enabled")
	}

	for i, agent := range config.Agents {
		if agent.Name == "" {
			return fmt.Errorf("agent[%d].name is required", i)
		}
		// Only validate required fields for enabled agents
		if agent.IsEnabled() {
			if agent.Prompt == "" {
				return fmt.Errorf("agent[%d].prompt is required for enabled agents", i)
			}
		}
	}

	return nil
}

func setDefaults(config *Config) {
	if config.Settings.CheckInterval == nil {
		config.Settings.CheckInterval = IntPtr(60)
	}
	if config.Settings.TeamName == nil {
		config.Settings.TeamName = StringPtr(DefaultTeamName)
	}
	if config.Settings.MaxAttempts == nil {
		config.Settings.MaxAttempts = IntPtr(3)
	}
	// Set default service configuration if not provided
	if config.Settings.Service == nil {
		config.Settings.Service = map[string]interface{}{
			"image": "node:18.17.1",
			"user":  "developer",
		}
	}
}

func CreateSampleConfig(filename string) error {
	sampleConfig := Config{
		Agents: []Agent{
			{
				Name:   "dev1",
				Prompt: "You are a developer agent responsible for implementing features and fixing bugs.",
			},
			{
				Name:   "arch1",
				Prompt: "You are an architecture agent responsible for system design and code reviews.",
				Settings: &AgentSettings{
					CheckInterval: IntPtr(30),
					Service: map[string]interface{}{
						"image": "python:3.11",
						"volumes": []string{
							"./custom-configs:/app/configs:ro",
							"/var/run/docker.sock:/var/run/docker.sock",
						},
						"environment": map[string]string{
							"PYTHON_PATH": "/app/custom",
							"DEBUG_MODE":  "true",
						},
					},
				},
			},
			{
				Name:    "devops1",
				Prompt:  "You are a DevOps agent responsible for CI/CD and infrastructure.",
				Enabled: BoolPtr(false), // This agent is disabled
			},
		},
		Services: map[string]map[string]interface{}{
			"postgres": {
				"image": "postgres:15",
				"environment": map[string]string{
					"POSTGRES_DB":       "autoteam_dev",
					"POSTGRES_USER":     "autoteam",
					"POSTGRES_PASSWORD": "development_password",
				},
				"ports": []string{"5432:5432"},
				"volumes": []string{
					"postgres_data:/var/lib/postgresql/data",
					"./sql/init.sql:/docker-entrypoint-initdb.d/init.sql:ro",
				},
			},
			"redis": {
				"image":   "redis:7",
				"ports":   []string{"6379:6379"},
				"volumes": []string{"redis_data:/data"},
			},
		},
		Settings: AgentSettings{
			CheckInterval: IntPtr(60),
			TeamName:      StringPtr(DefaultTeamName),
			InstallDeps:   BoolPtr(true),
			CommonPrompt:  StringPtr("Always follow coding best practices and write comprehensive tests."),
			MaxAttempts:   IntPtr(3),
			Service: map[string]interface{}{
				"image": "node:18.17.1",
				"user":  "developer",
			},
		},
		MCPServers: map[string]MCPServer{
			"memory": {
				Command: "npx",
				Args:    []string{"-y", "mcp-memory-service"},
			},
		},
	}

	data, err := yaml.Marshal(&sampleConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal sample config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}

// mergeServiceConfigs merges global and agent service configurations
// Agent service properties override global ones, with special handling for maps and arrays
func mergeServiceConfigs(global, agent map[string]interface{}) map[string]interface{} {
	if global == nil && agent == nil {
		return nil
	}
	if global == nil {
		return copyServiceConfig(agent)
	}
	if agent == nil {
		return copyServiceConfig(global)
	}

	// Start with a copy of global config
	result := copyServiceConfig(global)

	// Override/merge with agent config
	for key, agentValue := range agent {
		globalValue, exists := result[key]

		// If key doesn't exist in global, just add it
		if !exists {
			result[key] = agentValue
			continue
		}

		// Universal map merging - merge any map-type values recursively
		if merged := tryMergeAsMapRecursive(globalValue, agentValue); merged != nil {
			result[key] = merged
			continue
		}

		// For all other properties (including arrays like volumes, ports), agent replaces global
		result[key] = agentValue
	}

	return result
}

// tryMergeAsMapRecursive attempts to merge two values as maps recursively using golang maps package
// Returns the merged map if successful, nil if values aren't compatible maps
func tryMergeAsMapRecursive(globalValue, agentValue interface{}) interface{} {
	// Try map[string]string first (most common for environment, labels, etc.)
	if globalMap, ok := globalValue.(map[string]string); ok {
		if agentMap, ok := agentValue.(map[string]string); ok {
			// Use maps.Clone for efficient copying, then merge
			merged := maps.Clone(globalMap)
			maps.Copy(merged, agentMap) // Agent values override global
			return merged
		}
	}

	// Try map[string]interface{} (common after YAML unmarshaling) with recursive merging
	if globalMap, ok := globalValue.(map[string]interface{}); ok {
		if agentMap, ok := agentValue.(map[string]interface{}); ok {
			// Use maps.Clone for efficient copying
			merged := maps.Clone(globalMap)

			// Recursively merge/override with agent values
			for k, agentVal := range agentMap {
				globalVal, exists := merged[k]

				if !exists {
					// Key doesn't exist in global, just add it (deep copy)
					merged[k] = deepCopyValue(agentVal)
				} else {
					// Try recursive merge for nested maps
					if recursiveMerged := tryMergeAsMapRecursive(globalVal, agentVal); recursiveMerged != nil {
						merged[k] = recursiveMerged
					} else {
						// Not mergeable maps, agent value replaces global (deep copy)
						merged[k] = deepCopyValue(agentVal)
					}
				}
			}
			return merged
		}
	}

	// Try mixed map types - convert map[string]string to map[string]interface{}
	if globalMap, ok := globalValue.(map[string]string); ok {
		if agentMap, ok := agentValue.(map[string]interface{}); ok {
			// Convert global to interface{} map and clone
			merged := make(map[string]interface{})
			for k, v := range globalMap {
				merged[k] = v
			}

			// Recursively merge/override with agent values
			for k, agentVal := range agentMap {
				globalVal, exists := merged[k]

				if !exists {
					merged[k] = deepCopyValue(agentVal)
				} else {
					if recursiveMerged := tryMergeAsMapRecursive(globalVal, agentVal); recursiveMerged != nil {
						merged[k] = recursiveMerged
					} else {
						merged[k] = deepCopyValue(agentVal)
					}
				}
			}
			return merged
		}
	}

	// Try reverse mixed types - convert map[string]interface{} to accommodate map[string]string
	if globalMap, ok := globalValue.(map[string]interface{}); ok {
		if agentMap, ok := agentValue.(map[string]string); ok {
			// Use maps.Clone for efficient copying
			merged := maps.Clone(globalMap)

			// Override with agent values (convert to interface{})
			for k, v := range agentMap {
				merged[k] = v
			}
			return merged
		}
	}

	// Values aren't compatible maps
	return nil
}

// deepCopyValue creates a deep copy of various value types
func deepCopyValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]string:
		return maps.Clone(v) // Use efficient maps.Clone
	case map[string]interface{}:
		// Deep copy for nested interface{} maps
		copied := make(map[string]interface{})
		for k, val := range v {
			copied[k] = deepCopyValue(val) // Recursive deep copy
		}
		return copied
	case []string:
		// Use slices.Clone if available, otherwise manual copy
		copied := make([]string, len(v))
		copy(copied, v)
		return copied
	case []interface{}:
		// Deep copy for slice of interfaces
		copied := make([]interface{}, len(v))
		for i, val := range v {
			copied[i] = deepCopyValue(val) // Recursive deep copy
		}
		return copied
	default:
		// For primitive types (string, int, bool, etc.), direct assignment is fine
		return value
	}
}

// copyServiceConfig creates a deep copy of a service configuration map
func copyServiceConfig(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range source {
		// Special handling for map types (like environment)
		if envMap, ok := value.(map[string]string); ok {
			newEnvMap := make(map[string]string)
			for k, v := range envMap {
				newEnvMap[k] = v
			}
			result[key] = newEnvMap
		} else if strSlice, ok := value.([]string); ok {
			// Copy string slices (like volumes)
			newSlice := make([]string, len(strSlice))
			copy(newSlice, strSlice)
			result[key] = newSlice
		} else {
			// For other types, direct assignment (should be safe for scalars)
			result[key] = value
		}
	}
	return result
}

// mergeMCPServers merges MCP server configurations from global settings, agent settings, and agent-level MCP servers
// Priority: agent-level MCPServers > agent.settings.MCPServers > global settings MCPServers
func mergeMCPServers(globalMCPServers, agentSettingsMCPServers, agentMCPServers map[string]MCPServer) map[string]MCPServer {
	if globalMCPServers == nil && agentSettingsMCPServers == nil && agentMCPServers == nil {
		return nil
	}

	result := make(map[string]MCPServer)

	// Start with global MCP servers
	for name, server := range globalMCPServers {
		result[name] = copyMCPServer(server)
	}

	// Override with agent settings MCP servers
	for name, server := range agentSettingsMCPServers {
		result[name] = copyMCPServer(server)
	}

	// Override with agent-level MCP servers (highest priority)
	for name, server := range agentMCPServers {
		result[name] = copyMCPServer(server)
	}

	return result
}

// copyMCPServer creates a deep copy of an MCPServer
func copyMCPServer(server MCPServer) MCPServer {
	copied := MCPServer{
		Command: server.Command,
	}

	// Copy args slice
	if server.Args != nil {
		copied.Args = make([]string, len(server.Args))
		copy(copied.Args, server.Args)
	}

	// Copy env map using maps.Clone
	if server.Env != nil {
		copied.Env = maps.Clone(server.Env)
	}

	return copied
}

// mergeAgentConfig merges agent configurations with agent-level overriding global
func mergeAgentConfig(global, agentLevel *AgentConfig) *AgentConfig {
	if agentLevel != nil {
		return copyAgentConfig(agentLevel)
	}
	if global != nil {
		return copyAgentConfig(global)
	}
	return nil
}

// copyAgentConfig creates a deep copy of an AgentConfig
func copyAgentConfig(source *AgentConfig) *AgentConfig {
	if source == nil {
		return nil
	}

	copied := &AgentConfig{
		Type: source.Type,
	}

	// Copy args slice
	if source.Args != nil {
		copied.Args = make([]string, len(source.Args))
		copy(copied.Args, source.Args)
	}

	// Copy env map using maps.Clone
	if source.Env != nil {
		copied.Env = maps.Clone(source.Env)
	}

	// Copy prompt
	if source.Prompt != nil {
		copied.Prompt = StringPtr(*source.Prompt)
	}

	return copied
}

// copyAgentSettings creates a deep copy of an AgentSettings
func copyAgentSettings(source AgentSettings) AgentSettings {
	copied := AgentSettings{}

	if source.CheckInterval != nil {
		copied.CheckInterval = IntPtr(*source.CheckInterval)
	}
	if source.TeamName != nil {
		copied.TeamName = StringPtr(*source.TeamName)
	}
	if source.InstallDeps != nil {
		copied.InstallDeps = BoolPtr(*source.InstallDeps)
	}
	if source.CommonPrompt != nil {
		copied.CommonPrompt = StringPtr(*source.CommonPrompt)
	}
	if source.MaxAttempts != nil {
		copied.MaxAttempts = IntPtr(*source.MaxAttempts)
	}

	// Copy service configuration
	if source.Service != nil {
		copied.Service = copyServiceConfig(source.Service)
	}

	// Copy MCP servers
	if source.MCPServers != nil {
		copied.MCPServers = make(map[string]MCPServer)
		for k, v := range source.MCPServers {
			copied.MCPServers[k] = copyMCPServer(v)
		}
	}

	// Copy agent configurations
	copied.CollectorAgent = copyAgentConfig(source.CollectorAgent)
	copied.ExecutionAgent = copyAgentConfig(source.ExecutionAgent)

	return copied
}

// GetEffectiveSettings returns the effective settings for an agent,
// merging global settings with agent-specific overrides
func (a *Agent) GetEffectiveSettings(globalSettings AgentSettings) AgentSettings {
	effective := copyAgentSettings(globalSettings) // Start with copy of global settings

	// Always merge MCP servers, even if agent settings is nil
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, nil, a.MCPServers)

	if a.Settings == nil {
		return effective
	}

	// Override with agent-specific settings where provided
	if a.Settings.CheckInterval != nil {
		effective.CheckInterval = a.Settings.CheckInterval
	}
	if a.Settings.TeamName != nil {
		effective.TeamName = a.Settings.TeamName
	}
	if a.Settings.InstallDeps != nil {
		effective.InstallDeps = a.Settings.InstallDeps
	}
	if a.Settings.CommonPrompt != nil {
		effective.CommonPrompt = a.Settings.CommonPrompt
	}
	if a.Settings.MaxAttempts != nil {
		effective.MaxAttempts = a.Settings.MaxAttempts
	}

	// Merge service configurations
	if len(a.Settings.Service) > 0 {
		effective.Service = mergeServiceConfigs(globalSettings.Service, a.Settings.Service)
	}

	// Merge MCP server configurations
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, a.Settings.MCPServers, a.MCPServers)

	// Merge layer agent configurations
	effective.CollectorAgent = mergeAgentConfig(globalSettings.CollectorAgent, a.Settings.CollectorAgent)
	effective.ExecutionAgent = mergeAgentConfig(globalSettings.ExecutionAgent, a.Settings.ExecutionAgent)

	return effective
}

// GetAllAgentsWithEffectiveSettings returns a slice of agents with their effective settings
func (c *Config) GetAllAgentsWithEffectiveSettings() []AgentWithSettings {
	var agents []AgentWithSettings
	for _, agent := range c.Agents {
		agents = append(agents, AgentWithSettings{
			Agent:             agent,
			EffectiveSettings: agent.GetEffectiveSettings(c.Settings),
		})
	}
	return agents
}

// GetEnabledAgentsWithEffectiveSettings returns only enabled agents with their effective settings
func (c *Config) GetEnabledAgentsWithEffectiveSettings() []AgentWithSettings {
	var agents []AgentWithSettings
	for _, agent := range c.Agents {
		if agent.IsEnabled() {
			agents = append(agents, AgentWithSettings{
				Agent:             agent,
				EffectiveSettings: agent.GetEffectiveSettings(c.Settings),
			})
		}
	}
	return agents
}

type AgentWithSettings struct {
	Agent             Agent
	EffectiveSettings AgentSettings
}

// GetConsolidatedPrompt returns the agent prompt combined with common prompt
func (aws *AgentWithSettings) GetConsolidatedPrompt(cfg *Config) string {
	var promptParts []string

	// Add agent-specific prompt
	if aws.Agent.Prompt != "" {
		promptParts = append(promptParts, aws.Agent.Prompt)
	}

	// Add common prompt
	if aws.EffectiveSettings.CommonPrompt != nil && *aws.EffectiveSettings.CommonPrompt != "" {
		promptParts = append(promptParts, *aws.EffectiveSettings.CommonPrompt)
	}

	if len(promptParts) == 0 {
		return ""
	}

	return strings.Join(promptParts, "\n\n")
}

// normalizeAgentName converts agent names to snake_case for use in service names and paths
func normalizeAgentName(name string) string {
	// Replace any non-alphanumeric characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	normalized := reg.ReplaceAllString(name, "_")

	// Convert to lowercase
	normalized = strings.ToLower(normalized)

	// Remove leading/trailing underscores
	normalized = strings.Trim(normalized, "_")

	// Replace multiple consecutive underscores with single underscore
	multiUnderscoreReg := regexp.MustCompile(`_+`)
	normalized = multiUnderscoreReg.ReplaceAllString(normalized, "_")

	return normalized
}

// GetNormalizedName returns the normalized agent name suitable for service names and paths
func (a *Agent) GetNormalizedName() string {
	return normalizeAgentName(a.Name)
}

// GetNormalizedNameWithVariation returns the normalized agent name with a variation (e.g., collector, executor)
// for two-layer architecture using subdirectory structure
func (a *Agent) GetNormalizedNameWithVariation(variation string) string {
	normalizedName := normalizeAgentName(a.Name)
	if variation == "" {
		return normalizedName
	}
	return fmt.Sprintf("%s/%s", normalizedName, variation)
}

// IsEnabled returns true if the agent is enabled (default is true)
func (a *Agent) IsEnabled() bool {
	if a.Enabled == nil {
		return true
	}
	return *a.Enabled
}

// StringPtr returns a pointer to the given string value. Suitable for optional string parameters or configurations.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int value. Suitable for optional int parameters or configurations.
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to the given boolean value.
func BoolPtr(b bool) *bool {
	return &b
}

// Helper methods to get values with defaults for AgentSettings
func (s *AgentSettings) GetCheckInterval() int {
	if s.CheckInterval != nil {
		return *s.CheckInterval
	}
	return 60 // default
}

func (s *AgentSettings) GetTeamName() string {
	if s.TeamName != nil {
		return *s.TeamName
	}
	return DefaultTeamName // default
}

func (s *AgentSettings) GetInstallDeps() bool {
	if s.InstallDeps != nil {
		return *s.InstallDeps
	}
	return false // default
}

func (s *AgentSettings) GetCommonPrompt() string {
	if s.CommonPrompt != nil {
		return *s.CommonPrompt
	}
	return "" // default
}

func (s *AgentSettings) GetMaxAttempts() int {
	if s.MaxAttempts != nil {
		return *s.MaxAttempts
	}
	return 3 // default
}

// GetEffectiveLayers returns the effective layer configurations for an agent
func (a *Agent) GetEffectiveLayers(globalSettings AgentSettings) (firstLayer, secondLayer *AgentConfig) {
	effectiveSettings := a.GetEffectiveSettings(globalSettings)

	// First Layer (Task Collection) - CollectorAgent field
	firstLayer = effectiveSettings.CollectorAgent

	// Second Layer (Task Execution) - ExecutionAgent field
	secondLayer = effectiveSettings.ExecutionAgent

	return firstLayer, secondLayer
}

// GetEffectiveFirstLayerPrompt returns the custom prompt for the first layer (aggregation agent) if configured
func (a *Agent) GetEffectiveFirstLayerPrompt(globalSettings AgentSettings) *string {
	firstLayer, _ := a.GetEffectiveLayers(globalSettings)
	if firstLayer != nil && firstLayer.Prompt != nil {
		return firstLayer.Prompt
	}
	return nil
}

// GetEffectiveSecondLayerPrompt returns the custom prompt for the second layer (execution agent) if configured
func (a *Agent) GetEffectiveSecondLayerPrompt(globalSettings AgentSettings) *string {
	_, secondLayer := a.GetEffectiveLayers(globalSettings)
	if secondLayer != nil && secondLayer.Prompt != nil {
		return secondLayer.Prompt
	}
	return nil
}
