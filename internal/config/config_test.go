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
				Agents: []Agent{
					{
						Name:   "dev1",
						Prompt: "You are a developer agent",
					},
					{
						Name:   "arch1",
						Prompt: "You are an architect agent",
					},
				},
				Settings: AgentSettings{
					Service: map[string]interface{}{
						"image": "node:18.17.1",
						"user":  "developer",
					},
					CheckInterval: IntPtr(60),
					TeamName:      StringPtr("test-team"),
					InstallDeps:   BoolPtr(true),
					CommonPrompt:  StringPtr("Follow best practices"),
					Flow: []FlowStep{
						{Name: "collector", Type: "gemini", Prompt: "Collect tasks"},
						{Name: "executor", Type: "claude", DependsOn: []string{"collector"}, Prompt: "Execute tasks"},
					},
				},
			},
		},
		{
			name:     "minimal config with defaults",
			filename: "testdata/minimal.yaml",
			want: Config{
				Agents: []Agent{
					{
						Name:   "dev1",
						Prompt: "Developer",
					},
				},
				Settings: AgentSettings{
					Service: map[string]interface{}{
						"image": "node:18.17.1", // default
						"user":  "developer",    // default
					},
					CheckInterval: IntPtr(60),            // default
					TeamName:      StringPtr("autoteam"), // default
					InstallDeps:   BoolPtr(false),        // default
					Flow: []FlowStep{
						{Name: "collector", Type: "gemini", Prompt: "Collect tasks"},
						{Name: "executor", Type: "claude", DependsOn: []string{"collector"}, Prompt: "Execute tasks"},
					},
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
			}

			if got.Settings.Service["image"] != tt.want.Settings.Service["image"] {
				t.Errorf("Settings.Service[image] = %v, want %v", got.Settings.Service["image"], tt.want.Settings.Service["image"])
			}
			if got.Settings.Service["user"] != tt.want.Settings.Service["user"] {
				t.Errorf("Settings.Service[user] = %v, want %v", got.Settings.Service["user"], tt.want.Settings.Service["user"])
			}
			if got.Settings.GetCheckInterval() != tt.want.Settings.GetCheckInterval() {
				t.Errorf("Settings.CheckInterval = %v, want %v", got.Settings.GetCheckInterval(), tt.want.Settings.GetCheckInterval())
			}
			if got.Settings.GetTeamName() != tt.want.Settings.GetTeamName() {
				t.Errorf("Settings.TeamName = %v, want %v", got.Settings.GetTeamName(), tt.want.Settings.GetTeamName())
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

	if len(cfg.Agents) != 3 {
		t.Errorf("Sample config len(Agents) = %v, want 3", len(cfg.Agents))
	}

	if cfg.Agents[0].Name != "dev1" {
		t.Errorf("Sample config Agents[0].Name = %v, want dev1", cfg.Agents[0].Name)
	}

	if cfg.Agents[1].Name != "arch1" {
		t.Errorf("Sample config Agents[1].Name = %v, want arch1", cfg.Agents[1].Name)
	}

	if cfg.Agents[2].Name != "devops1" {
		t.Errorf("Sample config Agents[2].Name = %v, want devops1", cfg.Agents[2].Name)
	}

	// Check that the third agent is disabled
	if cfg.Agents[2].IsEnabled() {
		t.Errorf("Sample config Agents[2] should be disabled")
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
				Agents: []Agent{
					{Name: "dev1", Prompt: "prompt"},
				},
				Settings: AgentSettings{
					Flow: []FlowStep{
						{Name: "step1", Type: "claude", Prompt: "test"},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "no agents",
			config: Config{
				Agents: []Agent{},
			},
			wantErr: "at least one agent must be configured",
		},
		{
			name: "agent missing name",
			config: Config{
				Agents: []Agent{
					{Prompt: "prompt"},
				},
			},
			wantErr: "agent[0].name is required",
		},
		{
			name: "agent missing prompt",
			config: Config{
				Agents: []Agent{
					{Name: "dev1"},
				},
			},
			wantErr: "agent[0].prompt is required for enabled agents",
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

	if config.Settings.Service["image"] != "node:18.17.1" {
		t.Errorf("Service[image] = %v, want node:18.17.1", config.Settings.Service["image"])
	}
	if config.Settings.Service["user"] != "developer" {
		t.Errorf("Service[user] = %v, want developer", config.Settings.Service["user"])
	}
	if config.Settings.GetCheckInterval() != 60 {
		t.Errorf("CheckInterval = %v, want 60", config.Settings.GetCheckInterval())
	}
	if config.Settings.GetTeamName() != "autoteam" {
		t.Errorf("TeamName = %v, want autoteam", config.Settings.GetTeamName())
	}
	// MaxAttempts should also be set
	if config.Settings.GetMaxAttempts() != 3 {
		t.Errorf("MaxAttempts = %v, want 3", config.Settings.GetMaxAttempts())
	}

	// Test that existing values are not overridden
	config2 := &Config{
		Settings: AgentSettings{
			Service: map[string]interface{}{
				"image": "custom:latest",
				"user":  "custom-user",
			},
			CheckInterval: IntPtr(120),
			TeamName:      StringPtr("custom-team"),
		},
	}

	setDefaults(config2)

	if config2.Settings.Service["image"] != "custom:latest" {
		t.Errorf("Service[image] should not be overridden, got %v", config2.Settings.Service["image"])
	}
	if config2.Settings.Service["user"] != "custom-user" {
		t.Errorf("Service[user] should not be overridden, got %v", config2.Settings.Service["user"])
	}
	if config2.Settings.GetCheckInterval() != 120 {
		t.Errorf("CheckInterval should not be overridden, got %v", config2.Settings.GetCheckInterval())
	}
	if config2.Settings.GetTeamName() != "custom-team" {
		t.Errorf("TeamName should not be overridden, got %v", config2.Settings.GetTeamName())
	}
}

func TestAgentIsEnabled(t *testing.T) {
	tests := []struct {
		name  string
		agent Agent
		want  bool
	}{
		{
			name: "agent with enabled=true",
			agent: Agent{
				Name:    "test",
				Enabled: BoolPtr(true),
			},
			want: true,
		},
		{
			name: "agent with enabled=false",
			agent: Agent{
				Name:    "test",
				Enabled: BoolPtr(false),
			},
			want: false,
		},
		{
			name: "agent without enabled field (default)",
			agent: Agent{
				Name: "test",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.agent.IsEnabled(); got != tt.want {
				t.Errorf("Agent.IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnabledAgentsWithEffectiveSettings(t *testing.T) {
	config := &Config{
		Agents: []Agent{
			{
				Name:    "dev1",
				Prompt:  "Developer",
				Enabled: BoolPtr(true),
			},
			{
				Name:    "dev2",
				Prompt:  "Developer",
				Enabled: BoolPtr(false),
			},
			{
				Name:   "dev3",
				Prompt: "Developer",
				// Enabled not set, defaults to true
			},
		},
		Settings: AgentSettings{
			CheckInterval: IntPtr(60),
			TeamName:      StringPtr("test"),
			Flow: []FlowStep{
				{Name: "step1", Type: "claude", Prompt: "test"},
			},
		},
	}

	agents := config.GetEnabledAgentsWithEffectiveSettings()
	if len(agents) != 2 {
		t.Errorf("GetEnabledAgentsWithEffectiveSettings() returned %d agents, want 2", len(agents))
	}

	// Check that only enabled agents are returned
	for _, agent := range agents {
		if agent.Agent.Name == "dev2" {
			t.Errorf("GetEnabledAgentsWithEffectiveSettings() returned disabled agent dev2")
		}
	}
}

func TestValidateConfigWithDisabledAgents(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "all agents disabled",
			config: Config{
				Agents: []Agent{
					{
						Name:    "dev1",
						Prompt:  "prompt",
						Enabled: BoolPtr(false),
					},
				},
			},
			wantErr: "at least one agent must be enabled",
		},
		{
			name: "disabled agent without required fields",
			config: Config{
				Agents: []Agent{
					{
						Name:    "dev1",
						Enabled: BoolPtr(false),
						// Missing required fields, but should be OK since agent is disabled
					},
					{
						Name:    "dev2",
						Prompt:  "prompt",
						Enabled: BoolPtr(true),
					},
				},
				Settings: AgentSettings{
					Flow: []FlowStep{
						{Name: "step1", Type: "claude", Prompt: "test"},
					},
				},
			},
			wantErr: "", // Should be valid
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
