package config

import (
	"path/filepath"
	"strings"
	"testing"

	"autoteam/internal/testutil"
)

func TestLoadConfig_Valid(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     Config
	}{
		{
			name:     "valid config",
			filename: "testdata/valid.yaml",
			want: Config{
				Repositories: Repositories{
					Include: []string{"owner/test-repo"},
				},
				Agents: []Agent{
					{
						Name:        "dev1",
						Prompt:      "You are a developer agent",
						GitHubToken: "DEV1_TOKEN",
						GitHubUser:  "dev-user",
					},
					{
						Name:        "arch1",
						Prompt:      "You are an architect agent",
						GitHubToken: "ARCH1_TOKEN",
						GitHubUser:  "arch-user",
					},
				},
				Settings: Settings{
					DockerImage:   "node:18.17.1",
					DockerUser:    "developer",
					CheckInterval: 60,
					TeamName:      "test-team",
					InstallDeps:   true,
					CommonPrompt:  "Follow best practices",
				},
			},
		},
		{
			name:     "minimal config with defaults",
			filename: "testdata/minimal.yaml",
			want: Config{
				Repositories: Repositories{
					Include: []string{"owner/repo"},
				},
				Agents: []Agent{
					{
						Name:        "dev1",
						Prompt:      "Developer",
						GitHubToken: "TOKEN",
						GitHubUser:  "dev-user",
					},
				},
				Settings: Settings{
					DockerImage:   "node:18.17.1", // default
					DockerUser:    "developer",    // default
					CheckInterval: 60,             // default
					TeamName:      "autoteam",     // default
					InstallDeps:   false,          // default
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadConfig(tt.filename)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if len(got.Repositories.Include) != len(tt.want.Repositories.Include) {
				t.Errorf("Repositories.Include length = %v, want %v", len(got.Repositories.Include), len(tt.want.Repositories.Include))
			}
			if len(got.Repositories.Include) > 0 && got.Repositories.Include[0] != tt.want.Repositories.Include[0] {
				t.Errorf("Repositories.Include[0] = %v, want %v", got.Repositories.Include[0], tt.want.Repositories.Include[0])
			}

			if len(got.Agents) != len(tt.want.Agents) {
				t.Fatalf("len(Agents) = %v, want %v", len(got.Agents), len(tt.want.Agents))
			}

			for i, agent := range got.Agents {
				wantAgent := tt.want.Agents[i]
				if agent.Name != wantAgent.Name {
					t.Errorf("Agent[%d].Name = %v, want %v", i, agent.Name, wantAgent.Name)
				}
				if agent.Prompt != wantAgent.Prompt {
					t.Errorf("Agent[%d].Prompt = %v, want %v", i, agent.Prompt, wantAgent.Prompt)
				}
				if agent.GitHubToken != wantAgent.GitHubToken {
					t.Errorf("Agent[%d].GitHubToken = %v, want %v", i, agent.GitHubToken, wantAgent.GitHubToken)
				}
			}

			if got.Settings.DockerImage != tt.want.Settings.DockerImage {
				t.Errorf("Settings.DockerImage = %v, want %v", got.Settings.DockerImage, tt.want.Settings.DockerImage)
			}
			if got.Settings.DockerUser != tt.want.Settings.DockerUser {
				t.Errorf("Settings.DockerUser = %v, want %v", got.Settings.DockerUser, tt.want.Settings.DockerUser)
			}
			if got.Settings.CheckInterval != tt.want.Settings.CheckInterval {
				t.Errorf("Settings.CheckInterval = %v, want %v", got.Settings.CheckInterval, tt.want.Settings.CheckInterval)
			}
			if got.Settings.TeamName != tt.want.Settings.TeamName {
				t.Errorf("Settings.TeamName = %v, want %v", got.Settings.TeamName, tt.want.Settings.TeamName)
			}
		})
	}
}

func TestLoadConfig_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  string
	}{
		{
			name:     "missing repository",
			filename: "testdata/invalid_no_repo.yaml",
			wantErr:  "at least one repository must be specified in repositories.include",
		},
		{
			name:     "no agents",
			filename: "testdata/invalid_no_agents.yaml",
			wantErr:  "at least one agent must be configured",
		},
		{
			name:     "non-existent file",
			filename: "testdata/nonexistent.yaml",
			wantErr:  "failed to read config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(tt.filename)
			if err == nil {
				t.Fatalf("LoadConfig() expected error containing %q, got nil", tt.wantErr)
			}

			if err.Error() == "" || len(tt.wantErr) == 0 {
				t.Fatalf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check if error contains expected substring
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadConfig() error = %v, wantErr containing %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateSampleConfig(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	configPath := filepath.Join(tempDir, "test-config.yaml")

	err := CreateSampleConfig(configPath)
	if err != nil {
		t.Fatalf("CreateSampleConfig() error = %v", err)
	}

	if !testutil.FileExists(configPath) {
		t.Fatalf("Config file was not created")
	}

	// Test that the created config can be loaded
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() on created sample error = %v", err)
	}

	// Verify some basic properties
	if len(cfg.Repositories.Include) == 0 || cfg.Repositories.Include[0] != "myorg/project-alpha" {
		t.Errorf("Sample config Repositories.Include[0] = %v, want myorg/project-alpha", cfg.Repositories.Include)
	}

	if len(cfg.Agents) != 2 {
		t.Errorf("Sample config len(Agents) = %v, want 2", len(cfg.Agents))
	}

	if cfg.Agents[0].Name != "dev1" {
		t.Errorf("Sample config Agents[0].Name = %v, want dev1", cfg.Agents[0].Name)
	}

	if cfg.Agents[1].Name != "arch1" {
		t.Errorf("Sample config Agents[1].Name = %v, want arch1", cfg.Agents[1].Name)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "valid config",
			config: Config{
				Repositories: Repositories{Include: []string{"owner/repo"}},
				Agents: []Agent{
					{Name: "dev1", Prompt: "prompt", GitHubToken: "TOKEN", GitHubUser: "dev-user"},
				},
			},
			wantErr: "",
		},
		{
			name: "missing repository URL",
			config: Config{
				Agents: []Agent{
					{Name: "dev1", Prompt: "prompt", GitHubToken: "TOKEN", GitHubUser: "dev-user"},
				},
			},
			wantErr: "at least one repository must be specified in repositories.include",
		},
		{
			name: "no agents",
			config: Config{
				Repositories: Repositories{Include: []string{"owner/repo"}},
				Agents:       []Agent{},
			},
			wantErr: "at least one agent must be configured",
		},
		{
			name: "agent missing name",
			config: Config{
				Repositories: Repositories{Include: []string{"owner/repo"}},
				Agents: []Agent{
					{Prompt: "prompt", GitHubToken: "TOKEN", GitHubUser: "dev-user"},
				},
			},
			wantErr: "agent[0].name is required",
		},
		{
			name: "agent missing prompt",
			config: Config{
				Repositories: Repositories{Include: []string{"owner/repo"}},
				Agents: []Agent{
					{Name: "dev1", GitHubToken: "TOKEN", GitHubUser: "dev-user"},
				},
			},
			wantErr: "agent[0].prompt is required",
		},
		{
			name: "agent missing github token env",
			config: Config{
				Repositories: Repositories{Include: []string{"owner/repo"}},
				Agents: []Agent{
					{Name: "dev1", Prompt: "prompt", GitHubUser: "dev-user"},
				},
			},
			wantErr: "agent[0].github_token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validateConfig() error = %v, wantErr nil", err)
				}
				return
			}

			if err == nil {
				t.Errorf("validateConfig() error = nil, wantErr %v", tt.wantErr)
				return
			}

			if err.Error() != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	config := &Config{}
	setDefaults(config)

	if config.Settings.DockerImage != "node:18.17.1" {
		t.Errorf("DockerImage = %v, want node:18.17.1", config.Settings.DockerImage)
	}
	if config.Settings.DockerUser != "developer" {
		t.Errorf("DockerUser = %v, want developer", config.Settings.DockerUser)
	}
	if config.Settings.CheckInterval != 60 {
		t.Errorf("CheckInterval = %v, want 60", config.Settings.CheckInterval)
	}
	if config.Settings.TeamName != "autoteam" {
		t.Errorf("TeamName = %v, want autoteam", config.Settings.TeamName)
	}
	// MainBranch is no longer a global config setting - it's handled per repository

	// Test that existing values are not overridden
	config2 := &Config{
		Settings: Settings{
			DockerImage:   "custom:latest",
			DockerUser:    "custom-user",
			CheckInterval: 120,
			TeamName:      "custom-team",
		},
		Repositories: Repositories{
			Include: []string{"owner/repo"},
		},
	}

	setDefaults(config2)

	if config2.Settings.DockerImage != "custom:latest" {
		t.Errorf("DockerImage should not be overridden, got %v", config2.Settings.DockerImage)
	}
	if config2.Settings.DockerUser != "custom-user" {
		t.Errorf("DockerUser should not be overridden, got %v", config2.Settings.DockerUser)
	}
	if config2.Settings.CheckInterval != 120 {
		t.Errorf("CheckInterval should not be overridden, got %v", config2.Settings.CheckInterval)
	}
	if config2.Settings.TeamName != "custom-team" {
		t.Errorf("TeamName should not be overridden, got %v", config2.Settings.TeamName)
	}
	// MainBranch is no longer a global config setting
}
