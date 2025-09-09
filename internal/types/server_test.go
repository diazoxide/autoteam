package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHealthResponse(t *testing.T) {
	timestamp := time.Now()
	checks := map[string]HealthCheck{
		"database": {
			Status:  "healthy",
			Message: "Connection successful",
		},
		"redis": {
			Status:  "unhealthy",
			Message: "Connection failed",
		},
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: timestamp,
		Agent: WorkerInfo{
			Name:    "test-agent",
			Type:    "claude",
			Version: "1.0.0",
		},
		Checks: checks,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal HealthResponse: %v", err)
	}

	// Test JSON deserialization
	var unmarshaled HealthResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal HealthResponse: %v", err)
	}

	// Verify fields
	if unmarshaled.Status != response.Status {
		t.Errorf("Expected status %s, got %s", response.Status, unmarshaled.Status)
	}

	if unmarshaled.Agent.Name != response.Agent.Name {
		t.Errorf("Expected agent name %s, got %s", response.Agent.Name, unmarshaled.Agent.Name)
	}

	if len(unmarshaled.Checks) != len(response.Checks) {
		t.Errorf("Expected %d checks, got %d", len(response.Checks), len(unmarshaled.Checks))
	}
}

func TestHealthCheck(t *testing.T) {
	check := HealthCheck{
		Status:  "healthy",
		Message: "All systems operational",
	}

	jsonData, err := json.Marshal(check)
	if err != nil {
		t.Errorf("Failed to marshal HealthCheck: %v", err)
	}

	var unmarshaled HealthCheck
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal HealthCheck: %v", err)
	}

	if unmarshaled.Status != check.Status {
		t.Errorf("Expected status %s, got %s", check.Status, unmarshaled.Status)
	}

	if unmarshaled.Message != check.Message {
		t.Errorf("Expected message %s, got %s", check.Message, unmarshaled.Message)
	}
}

func TestStatusResponse(t *testing.T) {
	timestamp := time.Now()
	response := StatusResponse{
		Status:    "running",
		Mode:      "production",
		Timestamp: timestamp,
		Agent: WorkerInfo{
			Name:    "production-agent",
			Type:    "gemini",
			Version: "2.1.0",
		},
		Uptime: "2h30m45s",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal StatusResponse: %v", err)
	}

	var unmarshaled StatusResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal StatusResponse: %v", err)
	}

	if unmarshaled.Status != response.Status {
		t.Errorf("Expected status %s, got %s", response.Status, unmarshaled.Status)
	}

	if unmarshaled.Mode != response.Mode {
		t.Errorf("Expected mode %s, got %s", response.Mode, unmarshaled.Mode)
	}

	if unmarshaled.Uptime != response.Uptime {
		t.Errorf("Expected uptime %s, got %s", response.Uptime, unmarshaled.Uptime)
	}
}

func TestLogsResponse(t *testing.T) {
	timestamp := time.Now()
	logs := []LogFile{
		{
			Filename: "app.log",
			Size:     1024,
			Modified: timestamp.Add(-time.Hour),
		},
		{
			Filename: "error.log",
			Size:     2048,
			Modified: timestamp.Add(-30 * time.Minute),
		},
	}

	response := LogsResponse{
		Logs:      logs,
		Total:     len(logs),
		Timestamp: timestamp,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal LogsResponse: %v", err)
	}

	var unmarshaled LogsResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal LogsResponse: %v", err)
	}

	if len(unmarshaled.Logs) != len(response.Logs) {
		t.Errorf("Expected %d logs, got %d", len(response.Logs), len(unmarshaled.Logs))
	}

	if unmarshaled.Total != response.Total {
		t.Errorf("Expected total %d, got %d", response.Total, unmarshaled.Total)
	}

	// Check first log entry
	if len(unmarshaled.Logs) > 0 {
		expectedLog := response.Logs[0]
		actualLog := unmarshaled.Logs[0]

		if actualLog.Filename != expectedLog.Filename {
			t.Errorf("Expected log filename %s, got %s", expectedLog.Filename, actualLog.Filename)
		}

		if actualLog.Size != expectedLog.Size {
			t.Errorf("Expected log size %d, got %d", expectedLog.Size, actualLog.Size)
		}
	}
}

func TestMetricsResponse(t *testing.T) {
	timestamp := time.Now()
	uptime := "2h30m"
	avgExec := "2.5s"
	metrics := WorkerMetrics{
		Uptime:           &uptime,
		AvgExecutionTime: &avgExec,
		LastActivity:     &timestamp,
	}

	response := MetricsResponse{
		Metrics:   metrics,
		Timestamp: timestamp,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal MetricsResponse: %v", err)
	}

	var unmarshaled MetricsResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal MetricsResponse: %v", err)
	}

	if unmarshaled.Metrics.Uptime == nil || *unmarshaled.Metrics.Uptime != *response.Metrics.Uptime {
		t.Errorf("Expected uptime %s, got %v", *response.Metrics.Uptime, unmarshaled.Metrics.Uptime)
	}

	if unmarshaled.Metrics.AvgExecutionTime == nil || *unmarshaled.Metrics.AvgExecutionTime != *response.Metrics.AvgExecutionTime {
		t.Errorf("Expected avg execution time %s, got %v", *response.Metrics.AvgExecutionTime, unmarshaled.Metrics.AvgExecutionTime)
	}
}

func TestWorkerInfo(t *testing.T) {
	info := WorkerInfo{
		Name:    "test-worker",
		Type:    "claude",
		Version: "1.2.3",
	}

	jsonData, err := json.Marshal(info)
	if err != nil {
		t.Errorf("Failed to marshal WorkerInfo: %v", err)
	}

	var unmarshaled WorkerInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal WorkerInfo: %v", err)
	}

	if unmarshaled.Name != info.Name {
		t.Errorf("Expected name %s, got %s", info.Name, unmarshaled.Name)
	}

	if unmarshaled.Type != info.Type {
		t.Errorf("Expected type %s, got %s", info.Type, unmarshaled.Type)
	}

	if unmarshaled.Version != info.Version {
		t.Errorf("Expected version %s, got %s", info.Version, unmarshaled.Version)
	}
}

func TestLogFile(t *testing.T) {
	timestamp := time.Now()
	logFile := LogFile{
		Filename: "test.log",
		Size:     4096,
		Modified: timestamp,
	}

	jsonData, err := json.Marshal(logFile)
	if err != nil {
		t.Errorf("Failed to marshal LogFile: %v", err)
	}

	var unmarshaled LogFile
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal LogFile: %v", err)
	}

	if unmarshaled.Filename != logFile.Filename {
		t.Errorf("Expected filename %s, got %s", logFile.Filename, unmarshaled.Filename)
	}

	if unmarshaled.Size != logFile.Size {
		t.Errorf("Expected size %d, got %d", logFile.Size, unmarshaled.Size)
	}
}

func TestWorkerMetrics(t *testing.T) {
	timestamp := time.Now()
	uptime := "2h30m"
	avgExec := "1.5s"
	metrics := WorkerMetrics{
		Uptime:           &uptime,
		AvgExecutionTime: &avgExec,
		LastActivity:     &timestamp,
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		t.Errorf("Failed to marshal WorkerMetrics: %v", err)
	}

	var unmarshaled WorkerMetrics
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal WorkerMetrics: %v", err)
	}

	if unmarshaled.Uptime == nil || *unmarshaled.Uptime != *metrics.Uptime {
		t.Errorf("Expected uptime %s, got %v", *metrics.Uptime, unmarshaled.Uptime)
	}

	if unmarshaled.AvgExecutionTime == nil || *unmarshaled.AvgExecutionTime != *metrics.AvgExecutionTime {
		t.Errorf("Expected avg execution time %s, got %v", *metrics.AvgExecutionTime, unmarshaled.AvgExecutionTime)
	}

	if unmarshaled.LastActivity == nil {
		t.Error("Expected last activity to be set")
	}
}

// Test edge cases and error conditions
func TestEmptyStructsSerialization(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"Empty HealthResponse", HealthResponse{}},
		{"Empty StatusResponse", StatusResponse{}},
		{"Empty LogsResponse", LogsResponse{}},
		{"Empty MetricsResponse", MetricsResponse{}},
		{"Empty WorkerInfo", WorkerInfo{}},
		{"Empty LogFile", LogFile{}},
		{"Empty WorkerMetrics", WorkerMetrics{}},
		{"Empty HealthCheck", HealthCheck{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.value)
			if err != nil {
				t.Errorf("Failed to marshal %s: %v", tt.name, err)
			}

			if len(jsonData) == 0 {
				t.Errorf("Expected non-empty JSON for %s", tt.name)
			}

			// Should be valid JSON
			var result map[string]interface{}
			err = json.Unmarshal(jsonData, &result)
			if err != nil {
				t.Errorf("Invalid JSON generated for %s: %v", tt.name, err)
			}
		})
	}
}
