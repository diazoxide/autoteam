package config

import (
	"testing"
)

func TestAgentGetEffectiveSettings(t *testing.T) {
	globalSettings := Settings{
		DockerImage:   "node:18",
		DockerUser:    "developer",
		CheckInterval: 60,
		TeamName:      "global-team",
		InstallDeps:   true,
	}

	tests := []struct {
		name           string
		agent          Agent
		expectedResult Settings
	}{
		{
			name: "no agent settings - should use global",
			agent: Agent{
				Name:        "test1",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN1",
				GitHubUser:  "test-user",
				Settings:    nil,
			},
			expectedResult: globalSettings,
		},
		{
			name: "partial agent settings - should override only specified",
			agent: Agent{
				Name:        "test2",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN2",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					DockerImage:   stringPtr("python:3.11"),
					CheckInterval: intPtr(30),
				},
			},
			expectedResult: Settings{
				DockerImage:   "python:3.11", // overridden
				DockerUser:    "developer",   // from global
				CheckInterval: 30,            // overridden
				TeamName:      "global-team", // from global
				InstallDeps:   true,          // from global
			},
		},
		{
			name: "full agent settings - should override all",
			agent: Agent{
				Name:        "test3",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN3",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					DockerImage:   stringPtr("golang:1.21"),
					DockerUser:    stringPtr("admin"),
					CheckInterval: intPtr(15),
					TeamName:      stringPtr("custom-team"),
					InstallDeps:   boolPtr(false),
				},
			},
			expectedResult: Settings{
				DockerImage:   "golang:1.21",
				DockerUser:    "admin",
				CheckInterval: 15,
				TeamName:      "custom-team",
				InstallDeps:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.GetEffectiveSettings(globalSettings)

			if result.DockerImage != tt.expectedResult.DockerImage {
				t.Errorf("DockerImage = %v, want %v", result.DockerImage, tt.expectedResult.DockerImage)
			}
			if result.DockerUser != tt.expectedResult.DockerUser {
				t.Errorf("DockerUser = %v, want %v", result.DockerUser, tt.expectedResult.DockerUser)
			}
			if result.CheckInterval != tt.expectedResult.CheckInterval {
				t.Errorf("CheckInterval = %v, want %v", result.CheckInterval, tt.expectedResult.CheckInterval)
			}
			if result.TeamName != tt.expectedResult.TeamName {
				t.Errorf("TeamName = %v, want %v", result.TeamName, tt.expectedResult.TeamName)
			}
			if result.InstallDeps != tt.expectedResult.InstallDeps {
				t.Errorf("InstallDeps = %v, want %v", result.InstallDeps, tt.expectedResult.InstallDeps)
			}
		})
	}
}

func TestConfigGetAllAgentsWithEffectiveSettings(t *testing.T) {
	cfg := &Config{
		Repositories: Repositories{
			Include: []string{"owner/repo"},
		},
		Agents: []Agent{
			{
				Name:        "dev",
				Prompt:      "developer prompt",
				GitHubToken: "DEV_TOKEN",
				GitHubUser:  "dev-user",
				Settings:    nil, // no overrides
			},
			{
				Name:        "arch",
				Prompt:      "architect prompt",
				GitHubToken: "ARCH_TOKEN",
				GitHubUser:  "arch-user",
				Settings: &AgentSettings{
					DockerImage:   stringPtr("python:3.11"),
					CheckInterval: intPtr(30),
				},
			},
		},
		Settings: Settings{
			DockerImage:   "node:18",
			DockerUser:    "developer",
			CheckInterval: 60,
			TeamName:      "test-team",
			InstallDeps:   true,
		},
	}

	result := cfg.GetAllAgentsWithEffectiveSettings()

	if len(result) != 2 {
		t.Fatalf("Expected 2 agents, got %d", len(result))
	}

	// First agent should use global settings
	devAgent := result[0]
	if devAgent.Agent.Name != "dev" {
		t.Errorf("First agent name = %v, want dev", devAgent.Agent.Name)
	}
	if devAgent.EffectiveSettings.DockerImage != "node:18" {
		t.Errorf("Dev agent DockerImage = %v, want node:18", devAgent.EffectiveSettings.DockerImage)
	}
	if devAgent.EffectiveSettings.CheckInterval != 60 {
		t.Errorf("Dev agent CheckInterval = %v, want 60", devAgent.EffectiveSettings.CheckInterval)
	}

	// Second agent should have overridden settings
	archAgent := result[1]
	if archAgent.Agent.Name != "arch" {
		t.Errorf("Second agent name = %v, want arch", archAgent.Agent.Name)
	}
	if archAgent.EffectiveSettings.DockerImage != "python:3.11" {
		t.Errorf("Arch agent DockerImage = %v, want python:3.11", archAgent.EffectiveSettings.DockerImage)
	}
	if archAgent.EffectiveSettings.CheckInterval != 30 {
		t.Errorf("Arch agent CheckInterval = %v, want 30", archAgent.EffectiveSettings.CheckInterval)
	}
	// Non-overridden settings should use global values
	if archAgent.EffectiveSettings.DockerUser != "developer" {
		t.Errorf("Arch agent DockerUser = %v, want developer", archAgent.EffectiveSettings.DockerUser)
	}
	if archAgent.EffectiveSettings.TeamName != "test-team" {
		t.Errorf("Arch agent TeamName = %v, want test-team", archAgent.EffectiveSettings.TeamName)
	}
}

func TestAgentGetEffectiveSettingsWithCustomFields(t *testing.T) {
	globalSettings := Settings{
		DockerImage:   "node:18",
		DockerUser:    "developer",
		CheckInterval: 60,
		TeamName:      "global-team",
		InstallDeps:   true,
		Volumes:       []string{"./global-vol:/app/global"},
		Entrypoint:    "",
		Environment:   map[string]string{"GLOBAL_VAR": "global_value", "SHARED_VAR": "global_shared"},
	}

	tests := []struct {
		name           string
		agent          Agent
		expectedResult Settings
	}{
		{
			name: "custom volumes should override global",
			agent: Agent{
				Name:        "test1",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN1",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					Volumes: []string{
						"./custom-vol:/app/custom",
						"/host/path:/container/path:ro",
					},
				},
			},
			expectedResult: Settings{
				DockerImage:   "node:18",
				DockerUser:    "developer",
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
				Volumes:       []string{"./custom-vol:/app/custom", "/host/path:/container/path:ro"},
				Entrypoint:    "",
				Environment:   map[string]string{"GLOBAL_VAR": "global_value", "SHARED_VAR": "global_shared"},
			},
		},
		{
			name: "custom entrypoint should override global",
			agent: Agent{
				Name:        "test2",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN2",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					Entrypoint: stringPtr("/custom/entrypoint.sh"),
				},
			},
			expectedResult: Settings{
				DockerImage:   "node:18",
				DockerUser:    "developer",
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
				Volumes:       []string{"./global-vol:/app/global"},
				Entrypoint:    "/custom/entrypoint.sh",
				Environment:   map[string]string{"GLOBAL_VAR": "global_value", "SHARED_VAR": "global_shared"},
			},
		},
		{
			name: "custom environment should merge with global",
			agent: Agent{
				Name:        "test3",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN3",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					Environment: map[string]string{
						"CUSTOM_VAR": "custom_value",
						"SHARED_VAR": "agent_override", // Should override global
					},
				},
			},
			expectedResult: Settings{
				DockerImage:   "node:18",
				DockerUser:    "developer",
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
				Volumes:       []string{"./global-vol:/app/global"},
				Entrypoint:    "",
				Environment: map[string]string{
					"GLOBAL_VAR": "global_value",
					"SHARED_VAR": "agent_override", // Agent wins
					"CUSTOM_VAR": "custom_value",
				},
			},
		},
		{
			name: "all custom fields combined",
			agent: Agent{
				Name:        "test4",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN4",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					DockerImage: stringPtr("python:3.11"),
					Volumes:     []string{"./python-vol:/app/python"},
					Entrypoint:  stringPtr("python /app/main.py"),
					Environment: map[string]string{"PYTHON_ENV": "production"},
				},
			},
			expectedResult: Settings{
				DockerImage:   "python:3.11",
				DockerUser:    "developer",
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
				Volumes:       []string{"./python-vol:/app/python"},
				Entrypoint:    "python /app/main.py",
				Environment: map[string]string{
					"GLOBAL_VAR": "global_value",
					"SHARED_VAR": "global_shared",
					"PYTHON_ENV": "production",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.GetEffectiveSettings(globalSettings)

			// Check basic fields
			if result.DockerImage != tt.expectedResult.DockerImage {
				t.Errorf("DockerImage = %v, want %v", result.DockerImage, tt.expectedResult.DockerImage)
			}
			if result.Entrypoint != tt.expectedResult.Entrypoint {
				t.Errorf("Entrypoint = %v, want %v", result.Entrypoint, tt.expectedResult.Entrypoint)
			}

			// Check volumes
			if len(result.Volumes) != len(tt.expectedResult.Volumes) {
				t.Errorf("Volumes length = %v, want %v", len(result.Volumes), len(tt.expectedResult.Volumes))
			} else {
				for i, vol := range result.Volumes {
					if vol != tt.expectedResult.Volumes[i] {
						t.Errorf("Volumes[%d] = %v, want %v", i, vol, tt.expectedResult.Volumes[i])
					}
				}
			}

			// Check environment
			if len(result.Environment) != len(tt.expectedResult.Environment) {
				t.Errorf("Environment length = %v, want %v", len(result.Environment), len(tt.expectedResult.Environment))
			} else {
				for key, expectedVal := range tt.expectedResult.Environment {
					if actualVal, ok := result.Environment[key]; !ok {
						t.Errorf("Environment missing key %v", key)
					} else if actualVal != expectedVal {
						t.Errorf("Environment[%v] = %v, want %v", key, actualVal, expectedVal)
					}
				}
			}
		})
	}
}
