package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Default configuration constants
const (
	DefaultDockerImage = "node:18.17.1"
	DefaultDockerUser  = "developer"
	DefaultTeamName    = "auto-team"
	DefaultMainBranch  = "main"
)

type Config struct {
	Repository Repository `yaml:"repository"`
	Agents     []Agent    `yaml:"agents"`
	Settings   Settings   `yaml:"settings"`
}

type Repository struct {
	URL        string `yaml:"url"`
	MainBranch string `yaml:"main_branch"`
}

type Agent struct {
	Name           string            `yaml:"name"`
	Prompt         string            `yaml:"prompt"`
	GitHubTokenEnv string            `yaml:"github_token_env"`
	CommonPrompt   string            `yaml:"common_prompt,omitempty"`
	Settings       *AgentSettings    `yaml:"settings,omitempty"`
}

type AgentSettings struct {
	DockerImage   *string           `yaml:"docker_image,omitempty"`
	DockerUser    *string           `yaml:"docker_user,omitempty"`
	CheckInterval *int              `yaml:"check_interval,omitempty"`
	TeamName      *string           `yaml:"team_name,omitempty"`
	InstallDeps   *bool             `yaml:"install_deps,omitempty"`
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

func validateConfig(config *Config) error {
	if config.Repository.URL == "" {
		return fmt.Errorf("repository.url is required")
	}

	if len(config.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	for i, agent := range config.Agents {
		if agent.Name == "" {
			return fmt.Errorf("agent[%d].name is required", i)
		}
		if agent.GitHubTokenEnv == "" {
			return fmt.Errorf("agent[%d].github_token_env is required", i)
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
	if config.Repository.MainBranch == "" {
		config.Repository.MainBranch = DefaultMainBranch
	}
}

func CreateSampleConfig(filename string) error {
	sampleConfig := Config{
		Repository: Repository{
			URL:        "owner/repo-name",
			MainBranch: DefaultMainBranch,
		},
		Agents: []Agent{
			{
				Name:           "dev1",
				Prompt:         "You are a developer agent responsible for implementing features and fixing bugs.",
				GitHubTokenEnv: "DEV1_GITHUB_TOKEN",
				CommonPrompt:   "Always follow coding best practices and write comprehensive tests.",
			},
			{
				Name:           "arch1",
				Prompt:         "You are an architecture agent responsible for system design and code reviews.",
				GitHubTokenEnv: "ARCH1_GITHUB_TOKEN",
				CommonPrompt:   "Focus on maintainability, scalability, and architectural best practices.",
				Settings: &AgentSettings{
					DockerImage:   stringPtr("python:3.11"),
					CheckInterval: intPtr(30),
					Volumes: []string{
						"./custom-configs:/app/configs:ro",
						"/var/run/docker.sock:/var/run/docker.sock",
					},
					Environment: map[string]string{
						"PYTHON_PATH": "/app/custom",
						"DEBUG_MODE": "true",
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

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
