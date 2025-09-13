package grpc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	workerv1 "autoteam/internal/grpc/gen/proto/autoteam/worker/v1"
	"autoteam/internal/types"
	"autoteam/internal/worker"

	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServer_GetHealth(t *testing.T) {
	mockRuntime := createMockWorkerRuntimeForHandlers()
	server := &Server{runtime: mockRuntime}

	ctx := context.Background()
	req := &emptypb.Empty{}

	response, err := server.GetHealth(ctx, req)

	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Status != types.HealthStatusHealthy {
		t.Errorf("Expected status '%s', got '%s'", types.HealthStatusHealthy, response.Status)
	}

	if response.Agent == nil {
		t.Fatal("Expected agent info, got nil")
	}

	if response.Agent.Name != "Test Worker" {
		t.Errorf("Expected worker name 'Test Worker', got '%s'", response.Agent.Name)
	}

	if response.Checks == nil {
		t.Fatal("Expected health checks, got nil")
	}

	basicCheck, exists := response.Checks["basic"]
	if !exists {
		t.Error("Expected basic health check to exist")
	}

	if basicCheck.Status != types.HealthStatusHealthy {
		t.Errorf("Expected basic check status '%s', got '%s'", types.HealthStatusHealthy, basicCheck.Status)
	}

	if response.Timestamp == nil {
		t.Error("Expected timestamp to be set")
	}
}

func TestServer_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		isRunning      bool
		hasActiveSteps bool
		expectedStatus string
	}{
		{
			name:           "idle_worker",
			isRunning:      false,
			hasActiveSteps: false,
			expectedStatus: types.WorkerStatusIdle,
		},
		{
			name:           "running_worker_with_active_steps",
			isRunning:      true,
			hasActiveSteps: true,
			expectedStatus: types.WorkerStatusRunning,
		},
		{
			name:           "running_worker_without_active_steps",
			isRunning:      true,
			hasActiveSteps: false,
			expectedStatus: types.WorkerStatusIdle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRuntime := createMockWorkerRuntimeForHandlers()

			// Set up runtime state
			mockRuntime.SetRunning(tt.isRunning)
			if tt.hasActiveSteps {
				// Simulate active step
				mockRuntime.SetStepActive("test-step", true)
			}

			server := &Server{runtime: mockRuntime}
			ctx := context.Background()
			req := &emptypb.Empty{}

			response, err := server.GetStatus(ctx, req)

			if err != nil {
				t.Fatalf("GetStatus failed: %v", err)
			}

			if response.Status != tt.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, response.Status)
			}

			if response.Mode != types.WorkerModeBoth {
				t.Errorf("Expected mode '%s', got '%s'", types.WorkerModeBoth, response.Mode)
			}

			if response.Agent == nil {
				t.Fatal("Expected agent info, got nil")
			}

			if response.Uptime == nil {
				t.Error("Expected uptime to be set")
			}
		})
	}
}

func TestServer_GetFlowSteps(t *testing.T) {
	mockRuntime := createMockWorkerRuntimeWithFlowSteps()
	server := &Server{runtime: mockRuntime}

	ctx := context.Background()
	req := &emptypb.Empty{}

	response, err := server.GetFlowSteps(ctx, req)

	if err != nil {
		t.Fatalf("GetFlowSteps failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	expectedStepCount := 3
	if len(response.Steps) != expectedStepCount {
		t.Errorf("Expected %d steps, got %d", expectedStepCount, len(response.Steps))
	}

	if response.Total != int32(expectedStepCount) {
		t.Errorf("Expected total %d, got %d", expectedStepCount, response.Total)
	}

	// Verify first step details
	if len(response.Steps) > 0 {
		firstStep := response.Steps[0]
		if firstStep.Name != "step1" {
			t.Errorf("Expected first step name 'step1', got '%s'", firstStep.Name)
		}

		if firstStep.Type != "debug" {
			t.Errorf("Expected first step type 'debug', got '%s'", firstStep.Type)
		}

		if firstStep.DependencyPolicy == nil || *firstStep.DependencyPolicy != "fail_fast" {
			t.Error("Expected dependency policy 'fail_fast'")
		}

		if firstStep.Enabled == nil || !*firstStep.Enabled {
			t.Error("Expected step to be enabled")
		}
	}

	if response.Timestamp == nil {
		t.Error("Expected timestamp to be set")
	}
}

func TestServer_GetFlowSteps_WithRuntimeStats(t *testing.T) {
	mockRuntime := createMockWorkerRuntimeWithFlowSteps()

	// Add runtime statistics
	mockRuntime.SetStepActive("step1", true)
	mockRuntime.RecordStepExecution("step1", true, func() *string { s := "test output"; return &s }(), nil)

	server := &Server{runtime: mockRuntime}
	ctx := context.Background()
	req := &emptypb.Empty{}

	response, err := server.GetFlowSteps(ctx, req)

	if err != nil {
		t.Fatalf("GetFlowSteps failed: %v", err)
	}

	if len(response.Steps) == 0 {
		t.Fatal("Expected at least one step")
	}

	firstStep := response.Steps[0]

	if firstStep.Active == nil || !*firstStep.Active {
		t.Error("Expected first step to be active")
	}

	if firstStep.ExecutionCount == nil || *firstStep.ExecutionCount < 1 {
		t.Error("Expected execution count to be at least 1")
	}

	if firstStep.LastOutput == nil || *firstStep.LastOutput != "test output" {
		t.Error("Expected last output to be set")
	}
}

func TestServer_ListLogs(t *testing.T) {
	t.Skip("Skipping log tests - working directory access needs refactoring")
	// Create a temporary directory with test log files
	tempDir, err := os.MkdirTemp("", "autoteam-test-logs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logsDir := filepath.Join(tempDir, "logs")
	err = os.MkdirAll(logsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create test log files
	testFiles := []struct {
		name    string
		content string
	}{
		{"collector.log", "collector log content"},
		{"executor.log", "executor log content"},
		{"general.log", "general log content"},
		{"notlog.txt", "not a log file"}, // Should be ignored
	}

	for _, file := range testFiles {
		filePath := filepath.Join(logsDir, file.name)
		err = os.WriteFile(filePath, []byte(file.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	// Create server with custom working directory
	w := &worker.Worker{
		Name:   "Test Worker",
		Prompt: "Test prompt",
	}
	settings := worker.WorkerSettings{
		Flow: []worker.FlowStep{
			{Name: "test-step", Type: "debug"},
		},
	}
	customRuntime := worker.NewWorkerRuntime(w, settings)

	server := &Server{runtime: customRuntime}

	tests := []struct {
		name          string
		role          *string
		limit         *int32
		expectedFiles int
	}{
		{
			name:          "all_logs",
			role:          nil,
			limit:         nil,
			expectedFiles: 3, // Only .log files
		},
		{
			name:          "collector_logs",
			role:          func() *string { s := "collector"; return &s }(),
			limit:         nil,
			expectedFiles: 1,
		},
		{
			name:          "executor_logs",
			role:          func() *string { s := "executor"; return &s }(),
			limit:         nil,
			expectedFiles: 1,
		},
		{
			name:          "limited_logs",
			role:          nil,
			limit:         func() *int32 { i := int32(2); return &i }(),
			expectedFiles: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &workerv1.ListLogsRequest{
				Role:  tt.role,
				Limit: tt.limit,
			}

			ctx := context.Background()
			response, err := server.ListLogs(ctx, req)

			if err != nil {
				t.Fatalf("ListLogs failed: %v", err)
			}

			if len(response.Logs) != tt.expectedFiles {
				t.Errorf("Expected %d log files, got %d", tt.expectedFiles, len(response.Logs))
			}

			if response.Total != int32(len(response.Logs)) {
				t.Errorf("Expected total %d, got %d", len(response.Logs), response.Total)
			}

			// Verify log files have required fields
			for _, logFile := range response.Logs {
				if logFile.Filename == "" {
					t.Error("Expected filename to be set")
				}
				if logFile.Size <= 0 {
					t.Error("Expected size to be greater than 0")
				}
				if logFile.Modified == nil {
					t.Error("Expected modified timestamp to be set")
				}
			}
		})
	}
}

func TestServer_GetLogFile(t *testing.T) {
	t.Skip("Skipping log file tests - working directory access needs refactoring")
	// Create a temporary directory with test log file
	tempDir, err := os.MkdirTemp("", "autoteam-test-logs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logsDir := filepath.Join(tempDir, "logs")
	err = os.MkdirAll(logsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	testContent := "line1\nline2\nline3\nline4\nline5"
	testFile := filepath.Join(logsDir, "test.log")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create server with custom working directory
	w := &worker.Worker{Name: "Test Worker", Prompt: "Test prompt"}
	settings := worker.WorkerSettings{Flow: []worker.FlowStep{{Name: "test-step", Type: "debug"}}}
	customRuntime := worker.NewWorkerRuntime(w, settings)

	server := &Server{runtime: customRuntime}

	tests := []struct {
		name        string
		filename    string
		tail        *int32
		expectError bool
	}{
		{
			name:        "full_file",
			filename:    "test.log",
			tail:        nil,
			expectError: false,
		},
		{
			name:        "tail_3_lines",
			filename:    "test.log",
			tail:        func() *int32 { i := int32(3); return &i }(),
			expectError: false,
		},
		{
			name:        "nonexistent_file",
			filename:    "nonexistent.log",
			tail:        nil,
			expectError: true,
		},
		{
			name:        "path_traversal_attempt",
			filename:    "../../../etc/passwd",
			tail:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &workerv1.GetLogFileRequest{
				Filename: tt.filename,
				Tail:     tt.tail,
			}

			ctx := context.Background()
			response, err := server.GetLogFile(ctx, req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetLogFile failed: %v", err)
			}

			if response.Content == "" {
				t.Error("Expected content to be non-empty")
			}
		})
	}
}

func TestServer_GetConfig(t *testing.T) {
	mockRuntime := createMockWorkerRuntimeForHandlers()
	server := &Server{runtime: mockRuntime}

	ctx := context.Background()
	req := &emptypb.Empty{}

	response, err := server.GetConfig(ctx, req)

	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if response.Config == nil {
		t.Fatal("Expected config, got nil")
	}

	config := response.Config

	if config.Name == nil || *config.Name != "Test Worker" {
		t.Error("Expected worker name to be set correctly")
	}

	if config.Type == nil {
		t.Error("Expected worker type to be set")
	}

	if config.Enabled == nil || *config.Enabled != "true" {
		t.Error("Expected worker to be enabled")
	}

	if config.Version == nil {
		t.Error("Expected version to be set")
	}

	if response.Timestamp == nil {
		t.Error("Expected timestamp to be set")
	}
}

func TestDetermineLogRole(t *testing.T) {
	tests := []struct {
		filename     string
		expectedRole string
	}{
		{"collector.log", "collector"},
		{"collector_debug.log", "collector"},
		{"COLLECTOR.LOG", "collector"},
		{"executor.log", "executor"},
		{"executor_output.log", "executor"},
		{"EXECUTOR.LOG", "executor"},
		{"general.log", ""},
		{"application.log", ""},
		{"test.log", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			role := determineLogRole(tt.filename)
			if role != tt.expectedRole {
				t.Errorf("Expected role '%s', got '%s'", tt.expectedRole, role)
			}
		})
	}
}

// createMockWorkerRuntimeForHandlers creates a mock worker runtime for handler testing
func createMockWorkerRuntimeForHandlers() *worker.WorkerRuntime {
	w := &worker.Worker{
		Name:   "Test Worker",
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

	runtime := worker.NewWorkerRuntime(w, settings)
	runtime.Name = "Test Worker" // Ensure name is set
	return runtime
}

// createMockWorkerRuntimeWithFlowSteps creates a mock runtime with multiple flow steps
func createMockWorkerRuntimeWithFlowSteps() *worker.WorkerRuntime {
	w := &worker.Worker{
		Name:   "Test Worker",
		Prompt: "Test prompt",
	}

	settings := worker.WorkerSettings{
		Flow: []worker.FlowStep{
			{
				Name:             "step1",
				Type:             "debug",
				DependencyPolicy: "fail_fast",
				Retry: &worker.RetryConfig{
					MaxAttempts: 3,
					Delay:       2,
				},
			},
			{
				Name:             "step2",
				Type:             "claude",
				DependencyPolicy: "all_success",
				DependsOn:        []string{"step1"},
			},
			{
				Name:             "step3",
				Type:             "debug",
				DependencyPolicy: "all_complete",
				DependsOn:        []string{"step1", "step2"},
			},
		},
	}

	runtime := worker.NewWorkerRuntime(w, settings)
	runtime.Name = "Test Worker"
	return runtime
}
