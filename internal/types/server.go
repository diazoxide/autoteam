package types

import (
	"time"

	"autoteam/internal/worker"
)

// NOTE: This file contains types used by the server API responses.
// These types are referenced by the OpenAPI-generated code via x-go-type extensions
// and used by the server handlers to construct API responses.

// HealthResponse represents agent health status
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Agent     WorkerInfo             `json:"agent"`
	Checks    map[string]HealthCheck `json:"checks,omitempty"`
}

// HealthCheck represents individual health check result
type HealthCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// StatusResponse represents current worker status
type StatusResponse struct {
	Status    string     `json:"status"`
	Mode      string     `json:"mode"`
	Timestamp time.Time  `json:"timestamp"`
	Agent     WorkerInfo `json:"agent"`
	Uptime    string     `json:"uptime,omitempty"`
}

// LogsResponse represents list of log files
type LogsResponse struct {
	Logs      []LogFile `json:"logs"`
	Total     int       `json:"total"`
	Timestamp time.Time `json:"timestamp"`
}

// MetricsResponse represents agent performance metrics
type MetricsResponse struct {
	Metrics   WorkerMetrics `json:"metrics"`
	Timestamp time.Time     `json:"timestamp"`
}

// ConfigResponse represents sanitized agent configuration
type ConfigResponse struct {
	Config    WorkerConfig `json:"config"`
	Timestamp time.Time    `json:"timestamp"`
}

// WorkerInfo contains basic worker information
type WorkerInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	Available *bool  `json:"available,omitempty"`
}

// LogFile represents a log file entry
type LogFile struct {
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	Role     *string   `json:"role,omitempty"`
}

// WorkerMetrics represents worker performance metrics
type WorkerMetrics struct {
	Uptime           *string    `json:"uptime,omitempty"`
	AvgExecutionTime *string    `json:"avg_execution_time,omitempty"`
	LastActivity     *time.Time `json:"last_activity,omitempty"`
}

// WorkerConfig represents sanitized worker configuration
type WorkerConfig struct {
	Name      *string `json:"name,omitempty"`
	Type      *string `json:"type,omitempty"`
	Enabled   *string `json:"enabled,omitempty"`
	Version   *string `json:"version,omitempty"`
	TeamName  *string `json:"team_name,omitempty"`
	FlowSteps *int    `json:"flow_steps,omitempty"`
}

// FlowResponse represents flow configuration and status
type FlowResponse struct {
	Flow      FlowInfo  `json:"flow"`
	Timestamp time.Time `json:"timestamp"`
}

// FlowStepsResponse represents detailed flow step information
type FlowStepsResponse struct {
	Steps     []FlowStepInfo `json:"steps"`
	Total     int            `json:"total"`
	Timestamp time.Time      `json:"timestamp"`
}

// FlowInfo contains flow summary information
type FlowInfo struct {
	TotalSteps     int        `json:"total_steps"`
	EnabledSteps   int        `json:"enabled_steps"`
	LastExecution  *time.Time `json:"last_execution,omitempty"`
	ExecutionCount *int       `json:"execution_count,omitempty"`
	SuccessRate    *float64   `json:"success_rate,omitempty"`
}

// FlowStepRuntime contains runtime execution information for a flow step
type FlowStepRuntime struct {
	Enabled        *bool      `json:"enabled,omitempty"`
	LastExecution  *time.Time `json:"last_execution,omitempty"`
	ExecutionCount *int       `json:"execution_count,omitempty"`
	SuccessCount   *int       `json:"success_count,omitempty"`
	LastOutput     *string    `json:"last_output,omitempty"`
	LastError      *string    `json:"last_error,omitempty"`
}

// FlowStepInfo represents detailed information about a flow step using composition
type FlowStepInfo struct {
	worker.FlowStep `json:",inline"` // Embed original FlowStep with inline JSON
	FlowStepRuntime `json:",inline"` // Embed runtime fields inline
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error     string    `json:"error"`
	Code      *string   `json:"code,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Health status constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
)

// Worker status constants
const (
	WorkerStatusIdle    = "idle"
	WorkerStatusRunning = "running"
	WorkerStatusError   = "error"
)

// Worker mode constants
const (
	WorkerModeBoth = "both"
)

// Health check status constants
const (
	HealthCheckPass = "pass"
	HealthCheckFail = "fail"
)

// Log role constants
const (
	LogRoleCollector = "collector"
	LogRoleExecutor  = "executor"
	LogRoleBoth      = "both"
)
