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
	DefaultDockerImage = "node:18.17.1"
	DefaultDockerUser  = "developer"
	DefaultTeamName    = "autoteam"
)

type Config struct {
	Repositories Repositories `yaml:"repositories"`
	Agents       []Agent      `yaml:"agents"`
	Settings     Settings     `yaml:"settings"`
}

type Repositories struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type Agent struct {
	Name        string         `yaml:"name"`
	Prompt      string         `yaml:"prompt"`
	GitHubToken string         `yaml:"github_token"`
	GitHubUser  string         `yaml:"github_user"`
	Settings    *AgentSettings `yaml:"settings,omitempty"`
}

type AgentSettings struct {
	DockerImage   *string           `yaml:"docker_image,omitempty"`
	DockerUser    *string           `yaml:"docker_user,omitempty"`
	CheckInterval *int              `yaml:"check_interval,omitempty"`
	TeamName      *string           `yaml:"team_name,omitempty"`
	InstallDeps   *bool             `yaml:"install_deps,omitempty"`
	CommonPrompt  *string           `yaml:"common_prompt,omitempty"`
	MaxAttempts   *int              `yaml:"max_attempts,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Entrypoint    *string           `yaml:"entrypoint,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
}

type Settings struct {
	DockerImage   string            `yaml:"docker_image"`
	DockerUser    string            `yaml:"docker_user"`
	CheckInterval int               `yaml:"check_interval"`
	TeamName      string            `yaml:"team_name"`
	InstallDeps   bool              `yaml:"install_deps"`
	CommonPrompt  string            `yaml:"common_prompt,omitempty"`
	MaxAttempts   int               `yaml:"max_attempts"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Entrypoint    string            `yaml:"entrypoint,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
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

	for i, agent := range config.Agents {
		if agent.Name == "" {
			return fmt.Errorf("agent[%d].name is required", i)
		}
		if agent.GitHubToken == "" {
			return fmt.Errorf("agent[%d].github_token is required", i)
		}
		if agent.GitHubUser == "" {
			return fmt.Errorf("agent[%d].github_user is required", i)
		}
		if agent.Prompt == "" {
			return fmt.Errorf("agent[%d].prompt is required", i)
		}
	}

	return nil
}

func setDefaults(config *Config) {
	if config.Settings.DockerImage == "" {
		config.Settings.DockerImage = DefaultDockerImage
	}
	if config.Settings.DockerUser == "" {
		config.Settings.DockerUser = DefaultDockerUser
	}
	if config.Settings.CheckInterval == 0 {
		config.Settings.CheckInterval = 60
	}
	if config.Settings.TeamName == "" {
		config.Settings.TeamName = DefaultTeamName
	}
	if config.Settings.MaxAttempts == 0 {
		config.Settings.MaxAttempts = 3
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
					DockerImage:   stringPtr("python:3.11"),
					CheckInterval: intPtr(30),
					Volumes: []string{
						"./custom-configs:/app/configs:ro",
						"/var/run/docker.sock:/var/run/docker.sock",
					},
					Environment: map[string]string{
						"PYTHON_PATH": "/app/custom",
						"DEBUG_MODE":  "true",
					},
				},
			},
		},
		Settings: Settings{
			DockerImage:   DefaultDockerImage,
			DockerUser:    DefaultDockerUser,
			CheckInterval: 60,
			TeamName:      DefaultTeamName,
			InstallDeps:   true,
			CommonPrompt:  "Always follow coding best practices and write comprehensive tests.",
			MaxAttempts:   3,
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

// GetEffectiveSettings returns the effective settings for an agent,
// merging global settings with agent-specific overrides
func (a *Agent) GetEffectiveSettings(globalSettings Settings) Settings {
	effective := globalSettings // Start with global settings

	if a.Settings == nil {
		return effective
	}

	// Override with agent-specific settings where provided
	if a.Settings.DockerImage != nil {
		effective.DockerImage = *a.Settings.DockerImage
	}
	if a.Settings.DockerUser != nil {
		effective.DockerUser = *a.Settings.DockerUser
	}
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

	// Handle new fields
	if len(a.Settings.Volumes) > 0 {
		effective.Volumes = a.Settings.Volumes
	}
	if a.Settings.Entrypoint != nil {
		effective.Entrypoint = *a.Settings.Entrypoint
	}
	if len(a.Settings.Environment) > 0 {
		// Merge environment variables (agent-specific overrides global)
		effective.Environment = make(map[string]string)
		// Copy global environment first
		for k, v := range globalSettings.Environment {
			effective.Environment[k] = v
		}
		// Override with agent-specific environment
		for k, v := range a.Settings.Environment {
			effective.Environment[k] = v
		}
	}

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

// buildCollaboratorsList builds a list of all agents/collaborators from the config
func buildCollaboratorsList(cfg *Config) string {
	if len(cfg.Agents) <= 1 {
		// If there's only one agent, no need to show collaborators list
		return ""
	}

	var collaborators []string
	collaborators = append(collaborators, "# List of all collaborators:")

	for i, agent := range cfg.Agents {
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

// Helper functions for creating pointers
func StringPtr(s string) *string {
	return &s
}

func IntPtr(i int) *int {
	return &i
}

func BoolPtr(b bool) *bool {
	return &b
}

// Deprecated: use StringPtr instead
func stringPtr(s string) *string {
	return StringPtr(s)
}

// Deprecated: use IntPtr instead
func intPtr(i int) *int {
	return IntPtr(i)
}

// Deprecated: use BoolPtr instead
func boolPtr(b bool) *bool {
	return BoolPtr(b)
}
