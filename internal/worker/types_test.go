package worker

import (
	"testing"

	"autoteam/internal/util"
)

func TestWorker_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		worker   *Worker
		expected bool
	}{
		{
			name:     "enabled by default",
			worker:   &Worker{Name: "test"},
			expected: true,
		},
		{
			name:     "explicitly enabled",
			worker:   &Worker{Name: "test", Enabled: util.BoolPtr(true)},
			expected: true,
		},
		{
			name:     "explicitly disabled",
			worker:   &Worker{Name: "test", Enabled: util.BoolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.worker.IsEnabled()
			if got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWorker_GetNormalizedName(t *testing.T) {
	tests := []struct {
		name     string
		worker   *Worker
		expected string
	}{
		{
			name:     "simple name",
			worker:   &Worker{Name: "test"},
			expected: "test",
		},
		{
			name:     "name with spaces",
			worker:   &Worker{Name: "Test Worker"},
			expected: "test_worker",
		},
		{
			name:     "name with special characters",
			worker:   &Worker{Name: "Senior-Developer@v2.0"},
			expected: "senior_developer_v2_0",
		},
		{
			name:     "name with multiple underscores",
			worker:   &Worker{Name: "test___worker"},
			expected: "test_worker",
		},
		{
			name:     "name with leading/trailing underscores",
			worker:   &Worker{Name: "_test_worker_"},
			expected: "test_worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.worker.GetNormalizedName()
			if got != tt.expected {
				t.Errorf("GetNormalizedName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWorker_GetWorkerDir(t *testing.T) {
	worker := &Worker{Name: "Test Worker"}
	expected := "/opt/autoteam/workers/test_worker"

	got := worker.GetWorkerDir()
	if got != expected {
		t.Errorf("GetWorkerDir() = %v, want %v", got, expected)
	}
}

func TestWorkerSettings_Defaults(t *testing.T) {
	settings := &WorkerSettings{}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{
			name:     "default sleep duration",
			got:      settings.GetSleepDuration(),
			expected: 60,
		},
		{
			name:     "default team name",
			got:      settings.GetTeamName(),
			expected: "autoteam",
		},
		{
			name:     "default install deps",
			got:      settings.GetInstallDeps(),
			expected: false,
		},
		{
			name:     "default max attempts",
			got:      settings.GetMaxAttempts(),
			expected: 3,
		},
		{
			name:     "default debug",
			got:      settings.GetDebug(),
			expected: false,
		},
		{
			name:     "default common prompt",
			got:      settings.GetCommonPrompt(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}
