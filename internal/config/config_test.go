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
				Repository: Repository{
					URL:        "owner/test-repo",
					MainBranch: "main",
				},
				Agents: []Agent{
					{
						Name:           "dev1",
						Prompt:         "You are a developer agent",
						GitHubTokenEnv: "DEV1_TOKEN",
						CommonPrompt:   "Follow best practices",
					},
					{
						Name:           "arch1",
						Prompt:         "You are an architect agent",
						GitHubTokenEnv: "ARCH1_TOKEN",
					},
				},
				Settings: Settings{
					DockerImage:   "node:18.17.1",
					DockerUser:    "developer",
					CheckInterval: 60,
					TeamName:      "test-team",
					InstallDeps:   true,
				},
			},
		},
		{
			name:     "minimal config with defaults",
			filename: "testdata/minimal.yaml",
			want: Config{
				Repository: Repository{
					URL:        "owner/repo",
					MainBranch: "main", // default
				},
				Agents: []Agent{
					{
						Name:           "dev1",
						Prompt:         "Developer",
						GitHubTokenEnv: "TOKEN",
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

			if got.Repository.URL != tt.want.Repository.URL {
				t.Errorf("Repository.URL = %v, want %v", got.Repository.URL, tt.want.Repository.URL)
			}
			if got.Repository.MainBranch != tt.want.Repository.MainBranch {
				t.Errorf("Repository.MainBranch = %v, want %v", got.Repository.MainBranch, tt.want.Repository.MainBranch)
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
				if agent.GitHubTokenEnv != wantAgent.GitHubTokenEnv {
					t.Errorf("Agent[%d].GitHubTokenEnv = %v, want %v", i, agent.GitHubTokenEnv, wantAgent.GitHubTokenEnv)
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
			wantErr:  "repository.url is required",
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
	if cfg.Repository.URL != "owner/repo-name" {
		t.Errorf("Sample config Repository.URL = %v, want owner/repo-name", cfg.Repository.URL)
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
				Repository: Repository{URL: "owner/repo"},
				Agents: []Agent{
					{Name: "dev1", Prompt: "prompt", GitHubTokenEnv: "TOKEN"},
				},
			},
			wantErr: "",
		},
		{
			name: "missing repository URL",
			config: Config{
				Agents: []Agent{
					{Name: "dev1", Prompt: "prompt", GitHubTokenEnv: "TOKEN"},
				},
			},
			wantErr: "repository.url is required",
		},
		{
			name: "no agents",
			config: Config{
				Repository: Repository{URL: "owner/repo"},
				Agents:     []Agent{},
			},
			wantErr: "at least one agent must be configured",
		},
		{
			name: "agent missing name",
			config: Config{
				Repository: Repository{URL: "owner/repo"},
				Agents: []Agent{
					{Prompt: "prompt", GitHubTokenEnv: "TOKEN"},
				},
			},
			wantErr: "agent[0].name is required",
		},
		{
			name: "agent missing prompt",
			config: Config{
				Repository: Repository{URL: "owner/repo"},
				Agents: []Agent{
					{Name: "dev1", GitHubTokenEnv: "TOKEN"},
				},
			},
			wantErr: "agent[0].prompt is required",
		},
		{
			name: "agent missing github token env",
			config: Config{
				Repository: Repository{URL: "owner/repo"},
				Agents: []Agent{
					{Name: "dev1", Prompt: "prompt"},
				},
			},
			wantErr: "agent[0].github_token_env is required",
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
	if config.Repository.MainBranch != "main" {
		t.Errorf("MainBranch = %v, want main", config.Repository.MainBranch)
	}

	// Test that existing values are not overridden
	config2 := &Config{
		Settings: Settings{
			DockerImage:   "custom:latest",
			DockerUser:    "custom-user",
			CheckInterval: 120,
			TeamName:      "custom-team",
		},
		Repository: Repository{
			MainBranch: "develop",
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
	if config2.Repository.MainBranch != "develop" {
		t.Errorf("MainBranch should not be overridden, got %v", config2.Repository.MainBranch)
	}
}
