package server

import (
	"time"
)

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

// StatusResponse represents current agent status
type StatusResponse struct {
	Status      string       `json:"status"`
	Mode        string       `json:"mode"`
	Timestamp   time.Time    `json:"timestamp"`
	Agent       WorkerInfo   `json:"agent"`
	CurrentTask *TaskSummary `json:"current_task,omitempty"`
	Uptime      string       `json:"uptime,omitempty"`
}

// LogsResponse represents list of log files
type LogsResponse struct {
	Logs      []LogFile `json:"logs"`
	Total     int       `json:"total"`
	Timestamp time.Time `json:"timestamp"`
}

// TasksResponse represents list of tasks
type TasksResponse struct {
	Tasks     []TaskSummary `json:"tasks"`
	Total     int           `json:"total"`
	Timestamp time.Time     `json:"timestamp"`
}

// TaskResponse represents detailed task information
type TaskResponse struct {
	Task      Task      `json:"task"`
	Timestamp time.Time `json:"timestamp"`
}

// MetricsResponse represents agent performance metrics
type MetricsResponse struct {
	Metrics   WorkerMetrics `json:"metrics"`
	Timestamp time.Time    `json:"timestamp"`
}

// ConfigResponse represents sanitized agent configuration
type ConfigResponse struct {
	Config    WorkerConfig `json:"config"`
	Timestamp time.Time   `json:"timestamp"`
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

// Task represents a detailed task
type Task struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Priority      int               `json:"priority"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Platform      string            `json:"platform"`
	CompletionCmd string            `json:"completion_cmd,omitempty"`
	Context       map[string]string `json:"context,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	Status        *string           `json:"status,omitempty"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	Output        *string           `json:"output,omitempty"`
	Error         *string           `json:"error,omitempty"`
}

// TaskSummary represents a task summary
type TaskSummary struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Priority  int       `json:"priority"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	Status    *string   `json:"status,omitempty"`
}

// WorkerMetrics represents worker performance metrics
type WorkerMetrics struct {
	Uptime           *string    `json:"uptime,omitempty"`
	TasksProcessed   *int       `json:"tasks_processed,omitempty"`
	TasksSuccess     *int       `json:"tasks_success,omitempty"`
	TasksFailed      *int       `json:"tasks_failed,omitempty"`
	AvgExecutionTime *string    `json:"avg_execution_time,omitempty"`
	LastActivity     *time.Time `json:"last_activity,omitempty"`
}

// WorkerConfig represents sanitized worker configuration
type WorkerConfig struct {
	Name    *string `json:"name,omitempty"`
	Type    *string `json:"type,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	Version *string `json:"version,omitempty"`
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

// Agent status constants
const (
	AgentStatusIdle       = "idle"
	AgentStatusCollecting = "collecting"
	AgentStatusExecuting  = "executing"
	AgentStatusError      = "error"
)

// Agent mode constants
const (
	AgentModeCollector = "collector"
	AgentModeExecutor  = "executor"
	AgentModeBoth      = "both"
)

// Task status constants
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
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
