package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"autoteam/internal/worker"

	"github.com/labstack/echo/v4"
)

func createTestHandlers() *Handlers {
	// Create a minimal worker config for testing
	cfg := &worker.Worker{
		Name:   "test-worker",
		Prompt: "Test prompt",
	}

	settings := worker.WorkerSettings{
		Flow: []worker.FlowStep{
			{
				Name: "test-step",
				Type: "debug",
			},
		},
	}

	wk := worker.NewWorkerRuntime(cfg, settings)
	return NewHandlers(wk, "/tmp/test", time.Now())
}

func TestNewHandlers(t *testing.T) {
	cfg := &worker.Worker{
		Name:   "test-worker",
		Prompt: "Test prompt",
	}

	settings := worker.WorkerSettings{
		Flow: []worker.FlowStep{
			{
				Name: "test-step",
				Type: "debug",
			},
		},
	}

	wk := worker.NewWorkerRuntime(cfg, settings)
	workingDir := "/tmp/test"
	startTime := time.Now()

	handlers := NewHandlers(wk, workingDir, startTime)

	if handlers == nil {
		t.Fatal("Expected handlers to be created, got nil")
	}

	if handlers.worker != wk {
		t.Error("Expected worker to be set correctly")
	}

	if handlers.workingDir != workingDir {
		t.Error("Expected workingDir to be set correctly")
	}

	if handlers.startTime != startTime {
		t.Error("Expected startTime to be set correctly")
	}
}

func TestGetHealth(t *testing.T) {
	e := echo.New()
	handlers := createTestHandlers()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetHealth(c)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	// Check that response contains expected fields
	body := rec.Body.String()
	expectedFields := []string{
		"status",
		"timestamp",
		"agent",
		"checks",
	}

	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Expected response to contain field %s, body: %s", field, body)
		}
	}
}

func TestGetStatus(t *testing.T) {
	e := echo.New()
	handlers := createTestHandlers()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetStatus(c)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	// Check that response contains expected status fields
	body := rec.Body.String()
	expectedFields := []string{
		"status",
		"mode",
		"timestamp",
		"agent",
		"uptime",
	}

	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Expected response to contain field %s, body: %s", field, body)
		}
	}
}

func TestGetOpenAPISpec(t *testing.T) {
	e := echo.New()
	handlers := createTestHandlers()

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetOpenAPISpec(c)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Check content type for YAML
	contentType := rec.Header().Get("Content-Type")
	expectedContentType := "application/x-yaml"
	if contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check that response contains OpenAPI spec
	body := rec.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty OpenAPI spec response")
	}

	// Should contain some typical OpenAPI keywords
	openAPIKeywords := []string{"openapi", "info", "paths"}
	foundKeywords := 0
	for _, keyword := range openAPIKeywords {
		if strings.Contains(strings.ToLower(body), keyword) {
			foundKeywords++
		}
	}

	if foundKeywords == 0 {
		t.Error("Expected OpenAPI spec to contain typical OpenAPI keywords")
	}
}

func TestGetLogs(t *testing.T) {
	e := echo.New()
	handlers := createTestHandlers()

	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.GetLogs(c)

	// This might return an error if no log files exist, which is OK for testing
	if err != nil {
		// Check if it's a "no log files found" type error, which is acceptable
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d or %d, got %d", http.StatusOK, http.StatusNotFound, rec.Code)
		}
	} else {
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// If successful, should return JSON
		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
	}
}

func TestGetLogsWithQuery(t *testing.T) {
	e := echo.New()
	handlers := createTestHandlers()

	tests := []struct {
		name        string
		query       string
		expectError bool
	}{
		{
			name:        "with limit parameter",
			query:       "?limit=10",
			expectError: false,
		},
		{
			name:        "with search parameter",
			query:       "?search=test",
			expectError: false,
		},
		{
			name:        "with level parameter",
			query:       "?level=info",
			expectError: false,
		},
		{
			name:        "with multiple parameters",
			query:       "?limit=5&search=error&level=error",
			expectError: false,
		},
		{
			name:        "with invalid limit",
			query:       "?limit=invalid",
			expectError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/logs"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handlers.GetLogs(c)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			// Status should be OK or NotFound (if no logs exist)
			if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
				t.Errorf("Expected status code %d or %d, got %d", http.StatusOK, http.StatusNotFound, rec.Code)
			}
		})
	}
}

func TestHandlersIntegration(t *testing.T) {
	// Test that all handlers can be created and called without panicking
	handlers := createTestHandlers()
	e := echo.New()

	endpoints := []struct {
		method  string
		path    string
		handler echo.HandlerFunc
	}{
		{"GET", "/health", handlers.GetHealth},
		{"GET", "/status", handlers.GetStatus},
		{"GET", "/openapi.yaml", handlers.GetOpenAPISpec},
		{"GET", "/logs", handlers.GetLogs},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Should not panic
			err := endpoint.handler(c)

			// Error is acceptable for some endpoints (like logs if no files exist)
			// but we shouldn't panic
			if err != nil && rec.Code >= 500 {
				t.Errorf("Handler returned server error: %v, status: %d", err, rec.Code)
			}
		})
	}
}
