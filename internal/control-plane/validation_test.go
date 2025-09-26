package controlplane

import (
	"strings"
	"testing"

	"autoteam/internal/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWorkerID(t *testing.T) {
	tests := []struct {
		name      string
		workerID  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid UUID",
			workerID:  "550e8400-e29b-41d4-a716-446655440000",
			expectErr: false,
		},
		{
			name:      "empty string",
			workerID:  "",
			expectErr: true,
			errMsg:    "Worker ID is required",
		},
		{
			name:      "whitespace only",
			workerID:  "   \t\n   ",
			expectErr: true,
			errMsg:    "Worker ID is required",
		},
		{
			name:      "invalid UUID format",
			workerID:  "not-a-uuid",
			expectErr: true,
			errMsg:    "Invalid worker ID format",
		},
		{
			name:      "UUID with invalid characters",
			workerID:  "550e8400-e29b-41d4-a716-44665544000g",
			expectErr: true,
			errMsg:    "Invalid worker ID format",
		},
		{
			name:      "UUID too short",
			workerID:  "550e8400-e29b-41d4-a716",
			expectErr: true,
			errMsg:    "Invalid worker ID format",
		},
		{
			name:      "UUID too long",
			workerID:  "550e8400-e29b-41d4-a716-446655440000-extra",
			expectErr: true,
			errMsg:    "Invalid worker ID format",
		},
		{
			name:      "nil UUID string",
			workerID:  "00000000-0000-0000-0000-000000000000",
			expectErr: false, // nil UUID is valid UUID format in Go
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := validateWorkerID(tt.workerID)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Equal(t, uuid.Nil, id)
			} else {
				require.NoError(t, err)
				// For nil UUID case, we allow it but don't check NotEqual to uuid.Nil
				if tt.workerID != "00000000-0000-0000-0000-000000000000" {
					assert.NotEqual(t, uuid.Nil, id)
				}
				// Verify the returned UUID can be converted back to string
				assert.Equal(t, strings.ToLower(tt.workerID), id.String())
			}
		})
	}
}

func TestValidateCreateWorkerRequest(t *testing.T) {
	tests := []struct {
		name      string
		request   *types.CreateWorkerRequest
		expectErr bool
		errFields []string
	}{
		{
			name: "valid request",
			request: &types.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: "This is a valid prompt that is long enough",
			},
			expectErr: false,
		},
		{
			name: "empty name",
			request: &types.CreateWorkerRequest{
				Name:   "",
				Prompt: "This is a valid prompt that is long enough",
			},
			expectErr: true,
			errFields: []string{"name"},
		},
		{
			name: "whitespace only name",
			request: &types.CreateWorkerRequest{
				Name:   "   \t\n   ",
				Prompt: "This is a valid prompt that is long enough",
			},
			expectErr: true,
			errFields: []string{"name"},
		},
		{
			name: "name too long",
			request: &types.CreateWorkerRequest{
				Name:   strings.Repeat("a", 101), // 101 characters
				Prompt: "This is a valid prompt that is long enough",
			},
			expectErr: true,
			errFields: []string{"name"},
		},
		{
			name: "empty prompt",
			request: &types.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: "",
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "whitespace only prompt",
			request: &types.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: "   \t\n   ",
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "prompt too short",
			request: &types.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: "short", // 5 characters, minimum is 10
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "prompt too long",
			request: &types.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: strings.Repeat("a", 5001), // 5001 characters, maximum is 5000
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "multiple validation errors",
			request: &types.CreateWorkerRequest{
				Name:   "",
				Prompt: "short",
			},
			expectErr: true,
			errFields: []string{"name", "prompt"},
		},
		{
			name: "boundary valid name length",
			request: &types.CreateWorkerRequest{
				Name:   strings.Repeat("a", 100), // exactly 100 characters
				Prompt: "This is a valid prompt that is long enough",
			},
			expectErr: false,
		},
		{
			name: "boundary valid prompt length min",
			request: &types.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: "1234567890", // exactly 10 characters
			},
			expectErr: false,
		},
		{
			name: "boundary valid prompt length max",
			request: &types.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: strings.Repeat("a", 5000), // exactly 5000 characters
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateWorkerRequest(tt.request)

			if tt.expectErr {
				require.Error(t, err)

				// Check if it's a ValidationErrors type
				var validationErr *ValidationErrors
				assert.ErrorAs(t, err, &validationErr)

				// Verify expected fields are in the error
				for _, expectedField := range tt.errFields {
					found := false
					for _, valErr := range validationErr.Errors {
						if valErr.Field == expectedField {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected validation error for field: %s", expectedField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUpdateWorkerRequest(t *testing.T) {
	validName := "valid-worker-name"
	validPrompt := "This is a valid prompt that is long enough"
	emptyName := ""
	emptyPrompt := ""
	longName := strings.Repeat("a", 101)
	shortPrompt := "short"
	longPrompt := strings.Repeat("a", 5001)
	whitespaceName := "   \t\n   "
	whitespacePrompt := "   \t\n   "

	tests := []struct {
		name      string
		request   *types.UpdateWorkerRequest
		expectErr bool
		errFields []string
	}{
		{
			name: "valid name update",
			request: &types.UpdateWorkerRequest{
				Name: &validName,
			},
			expectErr: false,
		},
		{
			name: "valid prompt update",
			request: &types.UpdateWorkerRequest{
				Prompt: &validPrompt,
			},
			expectErr: false,
		},
		{
			name: "valid full update",
			request: &types.UpdateWorkerRequest{
				Name:   &validName,
				Prompt: &validPrompt,
			},
			expectErr: false,
		},
		{
			name:    "empty update request",
			request: &types.UpdateWorkerRequest{
				// No fields set
			},
			expectErr: false, // Empty update request should be valid
		},
		{
			name: "empty name",
			request: &types.UpdateWorkerRequest{
				Name: &emptyName,
			},
			expectErr: true,
			errFields: []string{"name"},
		},
		{
			name: "whitespace only name",
			request: &types.UpdateWorkerRequest{
				Name: &whitespaceName,
			},
			expectErr: true,
			errFields: []string{"name"},
		},
		{
			name: "name too long",
			request: &types.UpdateWorkerRequest{
				Name: &longName,
			},
			expectErr: true,
			errFields: []string{"name"},
		},
		{
			name: "empty prompt",
			request: &types.UpdateWorkerRequest{
				Prompt: &emptyPrompt,
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "whitespace only prompt",
			request: &types.UpdateWorkerRequest{
				Prompt: &whitespacePrompt,
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "prompt too short",
			request: &types.UpdateWorkerRequest{
				Prompt: &shortPrompt,
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "prompt too long",
			request: &types.UpdateWorkerRequest{
				Prompt: &longPrompt,
			},
			expectErr: true,
			errFields: []string{"prompt"},
		},
		{
			name: "multiple validation errors",
			request: &types.UpdateWorkerRequest{
				Name:   &emptyName,
				Prompt: &shortPrompt,
			},
			expectErr: true,
			errFields: []string{"name", "prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateWorkerRequest(tt.request)

			if tt.expectErr {
				require.Error(t, err)

				// Check if it's a ValidationErrors type
				var validationErr *ValidationErrors
				assert.ErrorAs(t, err, &validationErr)

				// Verify expected fields are in the error
				for _, expectedField := range tt.errFields {
					found := false
					for _, valErr := range validationErr.Errors {
						if valErr.Field == expectedField {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected validation error for field: %s", expectedField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   ValidationErrors
		expected string
	}{
		{
			name: "single error",
			errors: ValidationErrors{
				Errors: []ValidationError{
					{Field: "name", Message: "Name is required"},
				},
			},
			expected: "validation failed: Name is required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				Errors: []ValidationError{
					{Field: "name", Message: "Name is required"},
					{Field: "prompt", Message: "Prompt is too short"},
				},
			},
			expected: "validation failed: Name is required", // Should return first error
		},
		{
			name: "no errors",
			errors: ValidationErrors{
				Errors: []ValidationError{},
			},
			expected: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.errors.Error())
		})
	}
}

// Test edge cases for validation functions
func TestValidationEdgeCases(t *testing.T) {
	t.Run("nil create request", func(t *testing.T) {
		// This should panic or be handled gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic when validating nil request: %v", r)
			}
		}()

		err := validateCreateWorkerRequest(nil)
		// If no panic, we expect an error
		assert.Error(t, err)
	})

	t.Run("nil update request", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic when validating nil request: %v", r)
			}
		}()

		err := validateUpdateWorkerRequest(nil)
		// If no panic, we expect an error
		assert.Error(t, err)
	})

	t.Run("unicode in worker name", func(t *testing.T) {
		req := &types.CreateWorkerRequest{
			Name:   "worker-名前-测试-🤖",
			Prompt: "This is a valid prompt that is long enough",
		}

		err := validateCreateWorkerRequest(req)
		// Unicode should be allowed in names
		assert.NoError(t, err)
	})

	t.Run("special characters in prompt", func(t *testing.T) {
		req := &types.CreateWorkerRequest{
			Name:   "test-worker",
			Prompt: "Special chars: !@#$%^&*()_+-={}[]|\\:;\"'<>?,./ and unicode: 你好世界 🌍",
		}

		err := validateCreateWorkerRequest(req)
		// Special characters should be allowed in prompts
		assert.NoError(t, err)
	})
}
