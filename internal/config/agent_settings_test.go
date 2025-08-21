package config

import (
	"fmt"
	"testing"
)

func TestAgentGetSettings(t *testing.T) {
	globalSettings := AgentSettings{
		Service: map[string]interface{}{
			"image": "node:18",
			"user":  "developer",
		},
		SleepDuration: IntPtr(60),
		TeamName:      StringPtr("global-team"),
		InstallDeps:   BoolPtr(true),
	}

	tests := []struct {
		name           string
		worker         Worker
		expectedResult AgentSettings
	}{
		{
			name: "no agent settings - should use global",
			worker: Worker{
				Name:     "test1",
				Prompt:   "test prompt",
				Settings: nil,
			},
			expectedResult: globalSettings,
		},
		{
			name: "partial agent settings - should override only specified",
			worker: Worker{
				Name:   "test2",
				Prompt: "test prompt",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"image": "python:3.11",
					},
					SleepDuration: IntPtr(30),
				},
			},
			expectedResult: AgentSettings{
				Service: map[string]interface{}{
					"image": "python:3.11", // overridden
					"user":  "developer",   // from global
				},
				SleepDuration: IntPtr(30),               // overridden
				TeamName:      StringPtr("global-team"), // from global
				InstallDeps:   BoolPtr(true),            // from global
			},
		},
		{
			name: "full agent settings - should override all",
			worker: Worker{
				Name:   "test3",
				Prompt: "test prompt",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"image": "golang:1.21",
						"user":  "admin",
					},
					SleepDuration: IntPtr(15),
					TeamName:      StringPtr("custom-team"),
					InstallDeps:   BoolPtr(false),
				},
			},
			expectedResult: AgentSettings{
				Service: map[string]interface{}{
					"image": "golang:1.21",
					"user":  "admin",
				},
				SleepDuration: IntPtr(15),
				TeamName:      StringPtr("custom-team"),
				InstallDeps:   BoolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.worker.GetEffectiveSettings(globalSettings)

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
			if result.GetSleepDuration() != tt.expectedResult.GetSleepDuration() {
				t.Errorf("SleepDuration = %v, want %v", result.GetSleepDuration(), tt.expectedResult.GetSleepDuration())
			}
			if result.GetTeamName() != tt.expectedResult.GetTeamName() {
				t.Errorf("TeamName = %v, want %v", result.GetTeamName(), tt.expectedResult.GetTeamName())
			}
			if result.GetInstallDeps() != tt.expectedResult.GetInstallDeps() {
				t.Errorf("InstallDeps = %v, want %v", result.GetInstallDeps(), tt.expectedResult.GetInstallDeps())
			}
		})
	}
}

func TestConfigGetAllWorkersWithSettings(t *testing.T) {
	cfg := &Config{
		Workers: []Worker{
			{
				Name:     "dev",
				Prompt:   "developer prompt",
				Settings: nil, // no overrides
			},
			{
				Name:   "arch",
				Prompt: "architect prompt",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"image": "python:3.11",
					},
					SleepDuration: IntPtr(30),
				},
			},
		},
		Settings: AgentSettings{
			Service: map[string]interface{}{
				"image": "node:18",
				"user":  "developer",
			},
			SleepDuration: IntPtr(60),
			TeamName:      StringPtr("test-team"),
			InstallDeps:   BoolPtr(true),
		},
	}

	result := cfg.GetAllWorkersWithEffectiveSettings()

	if len(result) != 2 {
		t.Fatalf("Expected 2 workers, got %d", len(result))
	}

	// First agent should use global settings
	devWorker := result[0]
	if devWorker.Worker.Name != "dev" {
		t.Errorf("First agent name = %v, want dev", devWorker.Worker.Name)
	}
	if devWorker.Settings.Service["image"] != "node:18" {
		t.Errorf("Dev agent Service[image] = %v, want node:18", devWorker.Settings.Service["image"])
	}
	if devWorker.Settings.GetSleepDuration() != 60 {
		t.Errorf("Dev agent SleepDuration = %v, want 60", devWorker.Settings.GetSleepDuration())
	}

	// Second agent should have overridden settings
	archWorker := result[1]
	if archWorker.Worker.Name != "arch" {
		t.Errorf("Second agent name = %v, want arch", archWorker.Worker.Name)
	}
	if archWorker.Settings.Service["image"] != "python:3.11" {
		t.Errorf("Arch agent Service[image] = %v, want python:3.11", archWorker.Settings.Service["image"])
	}
	if archWorker.Settings.GetSleepDuration() != 30 {
		t.Errorf("Arch agent SleepDuration = %v, want 30", archWorker.Settings.GetSleepDuration())
	}
	// Non-overridden settings should use global values
	if archWorker.Settings.Service["user"] != "developer" {
		t.Errorf("Arch agent Service[user] = %v, want developer", archWorker.Settings.Service["user"])
	}
	if archWorker.Settings.GetTeamName() != "test-team" {
		t.Errorf("Arch agent TeamName = %v, want test-team", archWorker.Settings.GetTeamName())
	}
}

func TestAgentGetSettingsWithServiceMerging(t *testing.T) {
	globalSettings := AgentSettings{
		Service: map[string]interface{}{
			"image":   "node:18",
			"user":    "developer",
			"volumes": []string{"./global-vol:/app/global"},
			"environment": map[string]string{
				"GLOBAL_VAR": "global_value",
				"SHARED_VAR": "global_shared",
			},
		},
		SleepDuration: IntPtr(60),
		TeamName:      StringPtr("global-team"),
		InstallDeps:   BoolPtr(true),
	}

	tests := []struct {
		name           string
		worker         Worker
		expectedResult AgentSettings
	}{
		{
			name: "custom volumes should replace global",
			worker: Worker{
				Name:   "test1",
				Prompt: "test prompt",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"volumes": []string{
							"./custom-vol:/app/custom",
							"/host/path:/container/path:ro",
						},
					},
				},
			},
			expectedResult: AgentSettings{
				Service: map[string]interface{}{
					"image":   "node:18",
					"user":    "developer",
					"volumes": []string{"./custom-vol:/app/custom", "/host/path:/container/path:ro"},
					"environment": map[string]string{
						"GLOBAL_VAR": "global_value",
						"SHARED_VAR": "global_shared",
					},
				},
				SleepDuration: IntPtr(60),
				TeamName:      StringPtr("global-team"),
				InstallDeps:   BoolPtr(true),
			},
		},
		{
			name: "custom entrypoint should override global",
			worker: Worker{
				Name:   "test2",
				Prompt: "test prompt",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"entrypoint": []string{"/custom/entrypoint.sh"},
					},
				},
			},
			expectedResult: AgentSettings{
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
				SleepDuration: IntPtr(60),
				TeamName:      StringPtr("global-team"),
				InstallDeps:   BoolPtr(true),
			},
		},
		{
			name: "custom environment should merge with global",
			worker: Worker{
				Name:   "test3",
				Prompt: "test prompt",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"environment": map[string]string{
							"CUSTOM_VAR": "custom_value",
							"SHARED_VAR": "agent_override", // Should override global
						},
					},
				},
			},
			expectedResult: AgentSettings{
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
				SleepDuration: IntPtr(60),
				TeamName:      StringPtr("global-team"),
				InstallDeps:   BoolPtr(true),
			},
		},
		{
			name: "all custom service fields combined",
			worker: Worker{
				Name:   "test4",
				Prompt: "test prompt",
				Settings: &AgentSettings{
					Service: map[string]interface{}{
						"image":       "python:3.11",
						"volumes":     []string{"./python-vol:/app/python"},
						"entrypoint":  []string{"python", "/app/main.py"},
						"environment": map[string]string{"PYTHON_ENV": "production"},
					},
				},
			},
			expectedResult: AgentSettings{
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
				SleepDuration: IntPtr(60),
				TeamName:      StringPtr("global-team"),
				InstallDeps:   BoolPtr(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.worker.GetEffectiveSettings(globalSettings)

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
			if result.GetSleepDuration() != tt.expectedResult.GetSleepDuration() {
				t.Errorf("SleepDuration = %v, want %v", result.GetSleepDuration(), tt.expectedResult.GetSleepDuration())
			}
			if result.GetTeamName() != tt.expectedResult.GetTeamName() {
				t.Errorf("TeamName = %v, want %v", result.GetTeamName(), tt.expectedResult.GetTeamName())
			}
			if result.GetInstallDeps() != tt.expectedResult.GetInstallDeps() {
				t.Errorf("InstallDeps = %v, want %v", result.GetInstallDeps(), tt.expectedResult.GetInstallDeps())
			}
		})
	}
}
