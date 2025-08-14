package config

import (
	"reflect"
	"testing"
)

func TestMergeServiceConfigs(t *testing.T) {
	tests := []struct {
		name     string
		global   map[string]interface{}
		agent    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "merge environment maps",
			global: map[string]interface{}{
				"image": "node:18",
				"environment": map[string]interface{}{
					"GLOBAL_VAR": "global_value",
					"SHARED_VAR": "global_shared",
				},
			},
			agent: map[string]interface{}{
				"environment": map[string]interface{}{
					"AGENT_VAR":  "agent_value",
					"SHARED_VAR": "agent_shared", // should override global
				},
			},
			expected: map[string]interface{}{
				"image": "node:18", // from global
				"environment": map[string]interface{}{
					"GLOBAL_VAR": "global_value", // from global
					"AGENT_VAR":  "agent_value",  // from agent
					"SHARED_VAR": "agent_shared", // agent overrides global
				},
			},
		},
		{
			name: "merge nested objects",
			global: map[string]interface{}{
				"build": map[string]interface{}{
					"context":    ".",
					"dockerfile": "Dockerfile",
					"args": map[string]interface{}{
						"BASE_IMAGE": "node:18",
						"BUILD_ENV":  "production",
					},
				},
			},
			agent: map[string]interface{}{
				"build": map[string]interface{}{
					"args": map[string]interface{}{
						"BUILD_ENV":   "development", // override
						"CUSTOM_FLAG": "true",        // add new
					},
					"target": "dev", // add new field
				},
			},
			expected: map[string]interface{}{
				"build": map[string]interface{}{
					"context":    ".",          // from global
					"dockerfile": "Dockerfile", // from global
					"target":     "dev",        // from agent
					"args": map[string]interface{}{
						"BASE_IMAGE":  "node:18",     // from global
						"BUILD_ENV":   "development", // agent overrides global
						"CUSTOM_FLAG": "true",        // from agent
					},
				},
			},
		},
		{
			name: "merge arrays (should replace, not merge)",
			global: map[string]interface{}{
				"volumes": []interface{}{"./global:/app/global"},
				"ports":   []string{"8080:8080"},
			},
			agent: map[string]interface{}{
				"volumes": []interface{}{"./agent:/app/agent", "./shared:/app/shared"},
			},
			expected: map[string]interface{}{
				"volumes": []interface{}{"./agent:/app/agent", "./shared:/app/shared"}, // agent replaces global
				"ports":   []string{"8080:8080"},                                       // from global (not overridden)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeServiceConfigs(tt.global, tt.agent)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("mergeServiceConfigs() mismatch:\nGot:      %+v\nExpected: %+v", result, tt.expected)
			}
		})
	}
}

func TestTryMergeAsMapRecursive(t *testing.T) {
	tests := []struct {
		name      string
		globalVal interface{}
		agentVal  interface{}
		expected  interface{}
	}{
		{
			name: "recursive map merging",
			globalVal: map[string]interface{}{
				"level1": map[string]interface{}{
					"global_key": "global_value",
					"shared_key": "global_shared",
					"level2": map[string]interface{}{
						"deep_global": "deep_value",
					},
				},
			},
			agentVal: map[string]interface{}{
				"level1": map[string]interface{}{
					"agent_key":  "agent_value",
					"shared_key": "agent_shared", // should override
					"level2": map[string]interface{}{
						"deep_agent": "deep_agent_value",
					},
				},
			},
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"global_key": "global_value", // from global
					"agent_key":  "agent_value",  // from agent
					"shared_key": "agent_shared", // agent overrides
					"level2": map[string]interface{}{
						"deep_global": "deep_value",       // from global
						"deep_agent":  "deep_agent_value", // from agent
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tryMergeAsMapRecursive(tt.globalVal, tt.agentVal)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("tryMergeAsMapRecursive() mismatch:\nGot:      %+v\nExpected: %+v", result, tt.expected)
			}
		})
	}
}
