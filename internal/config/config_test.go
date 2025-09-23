package config

import (
	"path/filepath"
	"testing"

	"autoteam/internal/testutil"
	"autoteam/internal/worker"
)

func TestLoadConfig_Valid(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     Config
	}{
		{
			name:     "database-only config",
			filename: "testdata/database-only.yaml",
			want: Config{
				Settings: worker.WorkerSettings{
					Service: map[string]interface{}{
						"image": "node:18.17.1",
						"user":  "developer",
					},
					SleepDuration: 60,
					TeamName:      "test-team",
					InstallDeps:   true,
					CommonPrompt:  "Follow best practices",
					Flow: []worker.FlowStep{
						{Name: "collector", Type: "gemini", Input: "Collect tasks"},
						{Name: "executor", Type: "claude", DependsOn: []string{"collector"}, Input: "Execute tasks"},
					},
				},
				ControlPlane: &ControlPlaneConfig{
					Enabled: true,
					Port:    9090,
				},
				Dashboard: &DashboardConfig{
					Enabled: true,
					Port:    8081,
					APIUrl:  "http://localhost:9090",
				},
			},
		},
		{
			name:     "minimal config with defaults",
			filename: "testdata/minimal.yaml",
			want: Config{
				Settings: worker.WorkerSettings{
					Service: map[string]interface{}{
						"image": "node:18.17.1",
						"user":  "developer",
					},
					SleepDuration: 60,
					TeamName:      DefaultTeamName,
					MaxAttempts:   3,
					Flow: []worker.FlowStep{
						{Name: "debug", Type: "debug", Input: "Debug task"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file with expected content
			if !testutil.FileExists(tt.filename) {
				t.Skipf("Test file %s does not exist", tt.filename)
			}

			got, err := LoadConfig(tt.filename)
			if err != nil {
				t.Errorf("LoadConfig() error = %v", err)
				return
			}

			// Compare settings
			if got.Settings.TeamName != tt.want.Settings.TeamName {
				t.Errorf("TeamName = %v, want %v", got.Settings.TeamName, tt.want.Settings.TeamName)
			}
			if got.Settings.SleepDuration != tt.want.Settings.SleepDuration {
				t.Errorf("SleepDuration = %v, want %v", got.Settings.SleepDuration, tt.want.Settings.SleepDuration)
			}

			// Compare control plane config
			if (got.ControlPlane == nil) != (tt.want.ControlPlane == nil) {
				t.Errorf("ControlPlane presence mismatch")
			}
			if got.ControlPlane != nil && tt.want.ControlPlane != nil {
				if got.ControlPlane.Enabled != tt.want.ControlPlane.Enabled {
					t.Errorf("ControlPlane.Enabled = %v, want %v", got.ControlPlane.Enabled, tt.want.ControlPlane.Enabled)
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "database-only config is valid",
			config: Config{
				Settings: worker.WorkerSettings{
					TeamName: "test",
					Flow: []worker.FlowStep{
						{Name: "debug", Type: "debug", Input: "Debug task"},
					},
				},
				ControlPlane: &ControlPlaneConfig{
					Enabled: true,
				},
			},
			wantErr: false,
		},
		{
			name: "empty config is valid",
			config: Config{
				Settings: worker.WorkerSettings{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateSampleConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "sample.yaml")

	if err := CreateSampleConfig(tmpFile); err != nil {
		t.Fatalf("CreateSampleConfig() error = %v", err)
	}

	// Verify file was created
	if !testutil.FileExists(tmpFile) {
		t.Fatal("Sample config file was not created")
	}

	// Load and verify the sample config
	cfg, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load sample config: %v", err)
	}

	// Check that it has database-driven configuration
	if cfg.ControlPlane == nil || !cfg.ControlPlane.Enabled {
		t.Error("Sample config should have control plane enabled")
	}

	if cfg.Dashboard == nil || !cfg.Dashboard.Enabled {
		t.Error("Sample config should have dashboard enabled")
	}

	// Check global flow configuration exists
	if len(cfg.Settings.Flow) == 0 {
		t.Error("Sample config should have global flow configuration")
	}
}

func TestGetTeamName(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "custom team name",
			config: Config{
				Settings: worker.WorkerSettings{
					TeamName: "custom-team",
				},
			},
			expected: "custom-team",
		},
		{
			name: "default team name",
			config: Config{
				Settings: worker.WorkerSettings{},
			},
			expected: DefaultTeamName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetTeamName()
			if got != tt.expected {
				t.Errorf("GetTeamName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	config := &Config{
		Settings: worker.WorkerSettings{},
	}

	setDefaults(config)

	if config.Settings.SleepDuration == 0 {
		t.Error("Default sleep duration should be set")
	}
	if config.Settings.TeamName == "" {
		t.Error("Default team name should be set")
	}
	if config.Settings.MaxAttempts == 0 {
		t.Error("Default max attempts should be set")
	}
	if config.Settings.Service == nil {
		t.Error("Default service config should be set")
	}
	if config.Deployments == nil {
		t.Error("Default deployment config should be set")
	}
}
