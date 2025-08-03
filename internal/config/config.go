package config

import (
	"fmt"
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
	Repositories Repositories         `yaml:"repositories"`
	Agents       []Agent              `yaml:"agents"`
	Settings     Settings             `yaml:"settings"`
	MCPServers   map[string]MCPServer `yaml:"mcp_servers,omitempty"`
}

type Repositories struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type Agent struct {
	Name        string               `yaml:"name"`
	Prompt      string               `yaml:"prompt"`
	GitHubToken string               `yaml:"github_token"`
	GitHubUser  string               `yaml:"github_user"`
	Enabled     *bool                `yaml:"enabled,omitempty"`
	Settings    *AgentSettings       `yaml:"settings,omitempty"`
	MCPServers  map[string]MCPServer `yaml:"mcp_servers,omitempty"`
}

type AgentSettings struct {
	CheckInterval *int                   `yaml:"check_interval,omitempty"`
	TeamName      *string                `yaml:"team_name,omitempty"`
	InstallDeps   *bool                  `yaml:"install_deps,omitempty"`
	CommonPrompt  *string                `yaml:"common_prompt,omitempty"`
	MaxAttempts   *int                   `yaml:"max_attempts,omitempty"`
	Service       map[string]interface{} `yaml:"service,omitempty"`
	MCPServers    map[string]MCPServer   `yaml:"mcp_servers,omitempty"`
}

type Settings struct {
	CheckInterval int                    `yaml:"check_interval"`
	TeamName      string                 `yaml:"team_name"`
	InstallDeps   bool                   `yaml:"install_deps"`
	CommonPrompt  string                 `yaml:"common_prompt,omitempty"`
	MaxAttempts   int                    `yaml:"max_attempts"`
	Service       map[string]interface{} `yaml:"service,omitempty"`
	MCPServers    map[string]MCPServer   `yaml:"mcp_servers,omitempty"`
}

// MCPServer represents a Model Context Protocol server configuration
type MCPServer struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
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

// isRegexPattern checks if a pattern is a regex pattern (wrapped with /)
func isRegexPattern(pattern string) bool {
	return strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2
}

// extractRegexPattern extracts the regex pattern from /pattern/ format
func extractRegexPattern(pattern string) string {
	if !isRegexPattern(pattern) {
		return pattern
	}
	return pattern[1 : len(pattern)-1]
}

// matchesPattern checks if a repository name matches a pattern (exact or regex)
func matchesPattern(repoName, pattern string) bool {
	if isRegexPattern(pattern) {
		regex, err := regexp.Compile(extractRegexPattern(pattern))
		if err != nil {
			return false // Invalid regex, no match
		}
		return regex.MatchString(repoName)
	}
	return repoName == pattern
}

// ShouldIncludeRepository determines if a repository should be included based on include/exclude patterns
func (r *Repositories) ShouldIncludeRepository(repoName string) bool {
	// If no include patterns specified, include all by default
	includeAll := len(r.Include) == 0

	// Check include patterns
	if !includeAll {
		included := false
		for _, pattern := range r.Include {
			if matchesPattern(repoName, pattern) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	// Check exclude patterns
	for _, pattern := range r.Exclude {
		if matchesPattern(repoName, pattern) {
			return false
		}
	}

	return true
}

func validateConfig(config *Config) error {
	if len(config.Repositories.Include) == 0 && len(config.Repositories.Exclude) == 0 {
		return fmt.Errorf("at least one repository must be specified in repositories.include")
	}

	// Validate regex patterns
	for i, pattern := range config.Repositories.Include {
		if isRegexPattern(pattern) {
			if _, err := regexp.Compile(extractRegexPattern(pattern)); err != nil {
				return fmt.Errorf("repositories.include[%d]: invalid regex pattern '%s': %w", i, pattern, err)
			}
		}
	}
	for i, pattern := range config.Repositories.Exclude {
		if isRegexPattern(pattern) {
			if _, err := regexp.Compile(extractRegexPattern(pattern)); err != nil {
				return fmt.Errorf("repositories.exclude[%d]: invalid regex pattern '%s': %w", i, pattern, err)
			}
		}
	}

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
			if agent.GitHubToken == "" {
				return fmt.Errorf("agent[%d].github_token is required for enabled agents", i)
			}
			if agent.GitHubUser == "" {
				return fmt.Errorf("agent[%d].github_user is required for enabled agents", i)
			}
			if agent.Prompt == "" {
				return fmt.Errorf("agent[%d].prompt is required for enabled agents", i)
			}
		}
	}

	return nil
}

func setDefaults(config *Config) {
	if config.Settings.CheckInterval == 0 {
		config.Settings.CheckInterval = 60
	}
	if config.Settings.TeamName == "" {
		config.Settings.TeamName = DefaultTeamName
	}
	if config.Settings.MaxAttempts == 0 {
		config.Settings.MaxAttempts = 3
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
		Repositories: Repositories{
			Include: []string{
				"myorg/project-alpha",
				"/myorg\\/backend-.*/",
			},
			Exclude: []string{
				"myorg/legacy-project",
				"/.*-archived$/",
			},
		},
		Agents: []Agent{
			{
				Name:        "dev1",
				Prompt:      "You are a developer agent responsible for implementing features and fixing bugs.",
				GitHubToken: "ghp_your_github_token_here",
				GitHubUser:  "your-github-username",
			},
			{
				Name:        "arch1",
				Prompt:      "You are an architecture agent responsible for system design and code reviews.",
				GitHubToken: "ghp_your_github_token_here",
				GitHubUser:  "your-github-username",
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
				Name:        "devops1",
				Prompt:      "You are a DevOps agent responsible for CI/CD and infrastructure.",
				GitHubToken: "ghp_your_github_token_here",
				GitHubUser:  "your-github-username",
				Enabled:     BoolPtr(false), // This agent is disabled
			},
		},
		Settings: Settings{
			CheckInterval: 60,
			TeamName:      DefaultTeamName,
			InstallDeps:   true,
			CommonPrompt:  "Always follow coding best practices and write comprehensive tests.",
			MaxAttempts:   3,
			Service: map[string]interface{}{
				"image": "node:18.17.1",
				"user":  "developer",
			},
		},
		MCPServers: map[string]MCPServer{
			"github": {
				Command: "npx",
				Args:    []string{"-y", "@github/github-mcp-server"},
				Env: map[string]string{
					"GITHUB_TOKEN": "${GITHUB_TOKEN}",
				},
			},
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

		// Special handling for environment variables (maps) - merge them
		if key == "environment" {
			if globalEnv, ok := globalValue.(map[string]string); ok {
				if agentEnv, ok := agentValue.(map[string]string); ok {
					merged := make(map[string]string)
					// Copy global environment first
					for k, v := range globalEnv {
						merged[k] = v
					}
					// Override with agent environment
					for k, v := range agentEnv {
						merged[k] = v
					}
					result[key] = merged
					continue
				}
			}
			// If we can't merge as maps, fall back to replacement
		}

		// For all other properties (including arrays like volumes, ports), agent replaces global
		result[key] = agentValue
	}

	return result
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
	if globalMCPServers != nil {
		for name, server := range globalMCPServers {
			result[name] = copyMCPServer(server)
		}
	}

	// Override with agent settings MCP servers
	if agentSettingsMCPServers != nil {
		for name, server := range agentSettingsMCPServers {
			result[name] = copyMCPServer(server)
		}
	}

	// Override with agent-level MCP servers (highest priority)
	if agentMCPServers != nil {
		for name, server := range agentMCPServers {
			result[name] = copyMCPServer(server)
		}
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

	// Copy env map
	if server.Env != nil {
		copied.Env = make(map[string]string)
		for k, v := range server.Env {
			copied.Env[k] = v
		}
	}

	return copied
}

// GetEffectiveSettings returns the effective settings for an agent,
// merging global settings with agent-specific overrides
func (a *Agent) GetEffectiveSettings(globalSettings Settings) Settings {
	effective := globalSettings // Start with global settings

	// Always merge MCP servers, even if agent settings is nil
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, nil, a.MCPServers)

	if a.Settings == nil {
		return effective
	}

	// Override with agent-specific settings where provided
	if a.Settings.CheckInterval != nil {
		effective.CheckInterval = *a.Settings.CheckInterval
	}
	if a.Settings.TeamName != nil {
		effective.TeamName = *a.Settings.TeamName
	}
	if a.Settings.InstallDeps != nil {
		effective.InstallDeps = *a.Settings.InstallDeps
	}
	if a.Settings.CommonPrompt != nil {
		effective.CommonPrompt = *a.Settings.CommonPrompt
	}
	if a.Settings.MaxAttempts != nil {
		effective.MaxAttempts = *a.Settings.MaxAttempts
	}

	// Merge service configurations
	if len(a.Settings.Service) > 0 {
		effective.Service = mergeServiceConfigs(globalSettings.Service, a.Settings.Service)
	}

	// Merge MCP server configurations
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, a.Settings.MCPServers, a.MCPServers)

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
	EffectiveSettings Settings
}

// GetConsolidatedPrompt returns the agent prompt combined with common prompt and collaborators list
func (aws *AgentWithSettings) GetConsolidatedPrompt(cfg *Config) string {
	var promptParts []string

	// Add agent-specific prompt
	if aws.Agent.Prompt != "" {
		promptParts = append(promptParts, aws.Agent.Prompt)
	}

	// Add common prompt
	if aws.EffectiveSettings.CommonPrompt != "" {
		promptParts = append(promptParts, aws.EffectiveSettings.CommonPrompt)
	}

	// Add list of all collaborators
	if collaboratorsList := buildCollaboratorsList(cfg); collaboratorsList != "" {
		promptParts = append(promptParts, collaboratorsList)
	}

	if len(promptParts) == 0 {
		return ""
	}

	return strings.Join(promptParts, "\n\n")
}

// buildCollaboratorsList builds a list of all enabled agents/collaborators from the config
func buildCollaboratorsList(cfg *Config) string {
	var enabledAgents []Agent
	for _, agent := range cfg.Agents {
		if agent.IsEnabled() {
			enabledAgents = append(enabledAgents, agent)
		}
	}

	if len(enabledAgents) <= 1 {
		// If there's only one enabled agent, no need to show collaborators list
		return ""
	}

	var collaborators []string
	collaborators = append(collaborators, "# List of all collaborators:")

	for i, agent := range enabledAgents {
		if agent.GitHubUser != "" && agent.Name != "" {
			collaborators = append(collaborators, fmt.Sprintf("%d. %s - %s", i+1, agent.GitHubUser, agent.Name))
		}
	}

	// Only return the list if we have more than just the header
	if len(collaborators) > 1 {
		return strings.Join(collaborators, "\n")
	}

	return ""
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
