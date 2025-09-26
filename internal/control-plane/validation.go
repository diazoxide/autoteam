package controlplane

import (
	"fmt"
	"net/http"
	"strings"

	"autoteam/internal/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ValidationError represents a validation error with field context
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (v ValidationErrors) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s", v.Errors[0].Message)
}

// validateWorkerID validates and parses a worker ID parameter
func validateWorkerID(workerID string) (uuid.UUID, error) {
	if strings.TrimSpace(workerID) == "" {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "Worker ID is required")
	}

	id, err := uuid.Parse(workerID)
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid worker ID format")
	}

	return id, nil
}

// validateCreateWorkerRequest validates a create worker request
func validateCreateWorkerRequest(req *types.CreateWorkerRequest) error {
	var errors []ValidationError

	// Validate name
	if req.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Worker name is required",
		})
	} else {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Worker name cannot be empty",
			})
		} else if len(name) > 100 {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Worker name must be less than 100 characters",
			})
		}
	}

	// Validate prompt
	if req.Prompt == "" {
		errors = append(errors, ValidationError{
			Field:   "prompt",
			Message: "Worker prompt is required",
		})
	} else {
		prompt := strings.TrimSpace(req.Prompt)
		if prompt == "" {
			errors = append(errors, ValidationError{
				Field:   "prompt",
				Message: "Worker prompt cannot be empty",
			})
		} else if len(prompt) < 10 {
			errors = append(errors, ValidationError{
				Field:   "prompt",
				Message: "Worker prompt must be at least 10 characters",
			})
		} else if len(prompt) > 5000 {
			errors = append(errors, ValidationError{
				Field:   "prompt",
				Message: "Worker prompt must be less than 5000 characters",
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// validateUpdateWorkerRequest validates an update worker request
func validateUpdateWorkerRequest(req *types.UpdateWorkerRequest) error {
	var errors []ValidationError

	// Validate name if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Worker name cannot be empty",
			})
		} else if len(name) > 100 {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Worker name must be less than 100 characters",
			})
		}
	}

	// Validate prompt if provided
	if req.Prompt != nil {
		prompt := strings.TrimSpace(*req.Prompt)
		if prompt == "" {
			errors = append(errors, ValidationError{
				Field:   "prompt",
				Message: "Worker prompt cannot be empty",
			})
		} else if len(prompt) < 10 {
			errors = append(errors, ValidationError{
				Field:   "prompt",
				Message: "Worker prompt must be at least 10 characters",
			})
		} else if len(prompt) > 5000 {
			errors = append(errors, ValidationError{
				Field:   "prompt",
				Message: "Worker prompt must be less than 5000 characters",
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}
