package config

import (
	"fmt"
	"testing"
)

func TestAgentGetEffectiveSettings(t *testing.T) {
	globalSettings := Settings{
		Service: map[string]interface{}{
			"image": "node:18",
			"user":  "developer",
		},
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
					Service: map[string]interface{}{
						"image": "python:3.11",
					},
					CheckInterval: intPtr(30),
				},
			},
			expectedResult: Settings{
				Service: map[string]interface{}{
					"image": "python:3.11", // overridden
					"user":  "developer",   // from global
				},
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
					Service: map[string]interface{}{
						"image": "golang:1.21",
						"user":  "admin",
					},
					CheckInterval: intPtr(15),
					TeamName:      stringPtr("custom-team"),
					InstallDeps:   boolPtr(false),
				},
			},
			expectedResult: Settings{
				Service: map[string]interface{}{
					"image": "golang:1.21",
					"user":  "admin",
				},
				CheckInterval: 15,
				TeamName:      "custom-team",
				InstallDeps:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.GetEffectiveSettings(globalSettings)

			// Check service configuration
			if len(result.Service) != len(tt.expectedResult.Service) {
				t.Errorf("Service length = %v, want %v", len(result.Service), len(tt.expectedResult.Service))
			} else {
				for key, expectedVal := range tt.expectedResult.Service {
					if actualVal, ok := result.Service[key]; !ok {
						t.Errorf("Service missing key %v", key)
					} else if actualVal != expectedVal {
						t.Errorf("Service[%v] = %v, want %v", key, actualVal, expectedVal)
					}
				}
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
					Service: map[string]interface{}{
						"image": "python:3.11",
					},
					CheckInterval: intPtr(30),
				},
			},
		},
		Settings: Settings{
			Service: map[string]interface{}{
				"image": "node:18",
				"user":  "developer",
			},
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
	if devAgent.EffectiveSettings.Service["image"] != "node:18" {
		t.Errorf("Dev agent Service[image] = %v, want node:18", devAgent.EffectiveSettings.Service["image"])
	}
	if devAgent.EffectiveSettings.CheckInterval != 60 {
		t.Errorf("Dev agent CheckInterval = %v, want 60", devAgent.EffectiveSettings.CheckInterval)
	}

	// Second agent should have overridden settings
	archAgent := result[1]
	if archAgent.Agent.Name != "arch" {
		t.Errorf("Second agent name = %v, want arch", archAgent.Agent.Name)
	}
	if archAgent.EffectiveSettings.Service["image"] != "python:3.11" {
		t.Errorf("Arch agent Service[image] = %v, want python:3.11", archAgent.EffectiveSettings.Service["image"])
	}
	if archAgent.EffectiveSettings.CheckInterval != 30 {
		t.Errorf("Arch agent CheckInterval = %v, want 30", archAgent.EffectiveSettings.CheckInterval)
	}
	// Non-overridden settings should use global values
	if archAgent.EffectiveSettings.Service["user"] != "developer" {
		t.Errorf("Arch agent Service[user] = %v, want developer", archAgent.EffectiveSettings.Service["user"])
	}
	if archAgent.EffectiveSettings.TeamName != "test-team" {
		t.Errorf("Arch agent TeamName = %v, want test-team", archAgent.EffectiveSettings.TeamName)
	}
}

func TestAgentGetEffectiveSettingsWithServiceMerging(t *testing.T) {
	globalSettings := Settings{
		Service: map[string]interface{}{
			"image":   "node:18",
			"user":    "developer",
			"volumes": []string{"./global-vol:/app/global"},
			"environment": map[string]string{
				"GLOBAL_VAR": "global_value",
				"SHARED_VAR": "global_shared",
			},
		},
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
			name: "custom volumes should replace global",
			agent: Agent{
				Name:        "test1",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN1",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"volumes": []string{
							"./custom-vol:/app/custom",
							"/host/path:/container/path:ro",
						},
					},
				},
			},
			expectedResult: Settings{
				Service: map[string]interface{}{
					"image":   "node:18",
					"user":    "developer",
					"volumes": []string{"./custom-vol:/app/custom", "/host/path:/container/path:ro"},
					"environment": map[string]string{
						"GLOBAL_VAR": "global_value",
						"SHARED_VAR": "global_shared",
					},
				},
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
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
					Service: map[string]interface{}{
						"entrypoint": []string{"/custom/entrypoint.sh"},
					},
				},
			},
			expectedResult: Settings{
				Service: map[string]interface{}{
					"image":      "node:18",
					"user":       "developer",
					"volumes":    []string{"./global-vol:/app/global"},
					"entrypoint": []string{"/custom/entrypoint.sh"},
					"environment": map[string]string{
						"GLOBAL_VAR": "global_value",
						"SHARED_VAR": "global_shared",
					},
				},
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
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
					Service: map[string]interface{}{
						"environment": map[string]string{
							"CUSTOM_VAR": "custom_value",
							"SHARED_VAR": "agent_override", // Should override global
						},
					},
				},
			},
			expectedResult: Settings{
				Service: map[string]interface{}{
					"image":   "node:18",
					"user":    "developer",
					"volumes": []string{"./global-vol:/app/global"},
					"environment": map[string]string{
						"GLOBAL_VAR": "global_value",
						"SHARED_VAR": "agent_override", // Agent wins
						"CUSTOM_VAR": "custom_value",
					},
				},
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
			},
		},
		{
			name: "all custom service fields combined",
			agent: Agent{
				Name:        "test4",
				Prompt:      "test prompt",
				GitHubToken: "TOKEN4",
				GitHubUser:  "test-user",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"image":       "python:3.11",
						"volumes":     []string{"./python-vol:/app/python"},
						"entrypoint":  []string{"python", "/app/main.py"},
						"environment": map[string]string{"PYTHON_ENV": "production"},
					},
				},
			},
			expectedResult: Settings{
				Service: map[string]interface{}{
					"image":      "python:3.11",
					"user":       "developer",
					"volumes":    []string{"./python-vol:/app/python"},
					"entrypoint": []string{"python", "/app/main.py"},
					"environment": map[string]string{
						"GLOBAL_VAR": "global_value",
						"SHARED_VAR": "global_shared",
						"PYTHON_ENV": "production",
					},
				},
				CheckInterval: 60,
				TeamName:      "global-team",
				InstallDeps:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.GetEffectiveSettings(globalSettings)

			// Check service configuration
			if len(result.Service) != len(tt.expectedResult.Service) {
				t.Errorf("Service length = %v, want %v", len(result.Service), len(tt.expectedResult.Service))
			} else {
				for key, expectedVal := range tt.expectedResult.Service {
					actualVal, ok := result.Service[key]
					if !ok {
						t.Errorf("Service missing key %v", key)
						continue
					}

					// Special handling for environment maps
					if key == "environment" {
						expectedEnv, expectedOk := expectedVal.(map[string]string)
						actualEnv, actualOk := actualVal.(map[string]string)
						if !expectedOk || !actualOk {
							t.Errorf("Environment values are not map[string]string")
							continue
						}
						if len(actualEnv) != len(expectedEnv) {
							t.Errorf("Environment length = %v, want %v", len(actualEnv), len(expectedEnv))
						}
						for envKey, envExpected := range expectedEnv {
							if envActual, envOk := actualEnv[envKey]; !envOk {
								t.Errorf("Environment missing key %v", envKey)
							} else if envActual != envExpected {
								t.Errorf("Environment[%v] = %v, want %v", envKey, envActual, envExpected)
							}
						}
					} else {
						// For other types, use simple comparison
						if fmt.Sprintf("%v", actualVal) != fmt.Sprintf("%v", expectedVal) {
							t.Errorf("Service[%v] = %v, want %v", key, actualVal, expectedVal)
						}
					}
				}
			}

			// Check other fields
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
