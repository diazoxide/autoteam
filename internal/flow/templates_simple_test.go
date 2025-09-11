package flow

import (
	"context"
	"testing"

	"autoteam/internal/worker"

	"github.com/stretchr/testify/assert"
)

// TestBasicTemplateRendering tests basic template functionality
func TestBasicTemplateRendering(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		data        map[string]interface{}
		expectedOut string
		expectError bool
	}{
		{
			name:     "simple_variable_substitution",
			template: "Hello {{.name}}",
			data: map[string]interface{}{
				"name": "World",
			},
			expectedOut: "Hello World",
		},
		{
			name:     "string_functions",
			template: "{{.text | upper}}",
			data: map[string]interface{}{
				"text": "hello",
			},
			expectedOut: "HELLO",
		},
		{
			name:     "conditional_logic",
			template: "{{if .show}}visible{{else}}hidden{{end}}",
			data: map[string]interface{}{
				"show": true,
			},
			expectedOut: "visible",
		},
		{
			name:        "invalid_template",
			template:    "{{.missing.field",
			data:        map[string]interface{}{},
			expectError: true,
		},
		{
			name:        "default_value",
			template:    "{{.missing | default \"fallback\"}}",
			data:        map[string]interface{}{},
			expectedOut: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := createTestExecutor([]worker.FlowStep{})

			result, err := executor.applyTemplate(tt.template, tt.data)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOut, result)
		})
	}
}

// TestSkipConditionEvaluation tests skip condition evaluation
func TestSkipConditionEvaluation(t *testing.T) {
	tests := []struct {
		name         string
		skipWhen     string
		stepOutputs  map[string]StepOutput
		expectedSkip bool
		expectError  bool
	}{
		{
			name:         "simple_condition_true",
			skipWhen:     "{{- eq \"test\" \"test\" -}}",
			expectedSkip: true,
		},
		{
			name:         "simple_condition_false",
			skipWhen:     "{{- eq \"test\" \"other\" -}}",
			expectedSkip: false,
		},
		{
			name:         "invalid_template",
			skipWhen:     "{{- .missing.field -}}",
			expectedSkip: false, // Invalid templates should return false
		},
		{
			name:         "non_boolean_result",
			skipWhen:     "{{- \"not-boolean\" -}}",
			expectedSkip: false, // Non-boolean should be treated as false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := worker.FlowStep{
				Name:     "test-step",
				SkipWhen: tt.skipWhen,
			}

			executor := createTestExecutor([]worker.FlowStep{step})

			shouldSkip, err := executor.evaluateSkipCondition(context.Background(), step, tt.stepOutputs)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSkip, shouldSkip)
		})
	}
}
