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
	Name           string `yaml:"name"`
	Prompt         string `yaml:"prompt"`
	GitHubTokenEnv string `yaml:"github_token_env"`
	CommonPrompt   string `yaml:"common_prompt,omitempty"`
}

type Settings struct {
	DockerImage   string `yaml:"docker_image"`
	DockerUser    string `yaml:"docker_user"`
	CheckInterval int    `yaml:"check_interval"`
	TeamName      string `yaml:"team_name"`
	InstallDeps   bool   `yaml:"install_deps"`
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
