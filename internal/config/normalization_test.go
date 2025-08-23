package config

import (
	"testing"
)

func TestNormalizeAgentName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "dev1",
			expected: "dev1",
		},
		{
			name:     "name with spaces",
			input:    "Senior Developer",
			expected: "senior_developer",
		},
		{
			name:     "name with special characters",
			input:    "Dev-Agent #1",
			expected: "dev_agent_1",
		},
		{
			name:     "name with multiple spaces",
			input:    "Lead  Frontend   Developer",
			expected: "lead_frontend_developer",
		},
		{
			name:     "name with mixed case and special chars",
			input:    "BackEnd-API_Developer@Team1",
			expected: "backend_api_developer_team1",
		},
		{
			name:     "name with leading/trailing spaces",
			input:    "  developer  ",
			expected: "developer",
		},
		{
			name:     "name with underscores",
			input:    "dev_agent_1",
			expected: "dev_agent_1",
		},
		{
			name:     "name with numbers",
			input:    "Agent123",
			expected: "agent123",
		},
		{
			name:     "complex name",
			input:    "Senior Full-Stack Developer (Team Lead) - V2",
			expected: "senior_full_stack_developer_team_lead_v2",
		},
		{
			name:     "name with consecutive special chars",
			input:    "dev---agent___test",
			expected: "dev_agent_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeWorkerName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeWorkerName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAgentGetNormalizedName(t *testing.T) {
	agent := Worker{
		Name:   "Senior Developer Agent",
		Prompt: "You are a senior developer",
	}

	expected := "senior_developer_agent"
	result := agent.GetNormalizedName()

	if result != expected {
		t.Errorf("Agent.GetNormalizedName() = %q, want %q", result, expected)
	}
}

func TestAgentGetNormalizedNameWithVariation(t *testing.T) {
	agent := Worker{
		Name:   "Senior Developer Agent",
		Prompt: "You are a senior developer",
	}

	tests := []struct {
		name      string
		variation string
		expected  string
	}{
		{
			name:      "no variation",
			variation: "",
			expected:  "senior_developer_agent",
		},
		{
			name:      "collector variation",
			variation: "collector",
			expected:  "senior_developer_agent/collector",
		},
		{
			name:      "executor variation",
			variation: "executor",
			expected:  "senior_developer_agent/executor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.GetNormalizedNameWithVariation(tt.variation)
			if result != tt.expected {
				t.Errorf("Agent.GetNormalizedNameWithVariation(%q) = %q, want %q", tt.variation, result, tt.expected)
			}
		})
	}
}
