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
	Active         *bool      `json:"active,omitempty"`
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

// Control plane specific types
type ControlPlaneHealthResponse struct {
	Status        string            `json:"status"`
	Timestamp     time.Time         `json:"timestamp"`
	WorkersHealth map[string]string `json:"workers_health"`
	Message       *string           `json:"message,omitempty"`
}

type WorkersResponse struct {
	Workers   []WorkerResponse `json:"workers"`
	Total     int              `json:"total"`
	Timestamp time.Time        `json:"timestamp"`
}

type WorkerDetailsResponse struct {
	Worker    WorkerDetails `json:"worker"`
	Timestamp time.Time     `json:"timestamp"`
}

type WorkerDetails struct {
	ID         string      `json:"id"`
	URL        string      `json:"url"`
	Status     string      `json:"status"`
	LastCheck  *time.Time  `json:"last_check,omitempty"`
	WorkerInfo *WorkerInfo `json:"worker_info,omitempty"`
}

// Control plane health status constants
const (
	ControlPlaneStatusHealthy   = "healthy"
	ControlPlaneStatusDegraded  = "degraded"
	ControlPlaneStatusUnhealthy = "unhealthy"
)

// Worker connectivity status constants
const (
	WorkerStatusReachable   = "reachable"
	WorkerStatusUnreachable = "unreachable"
	WorkerStatusUnknown     = "unknown"
	WorkerStatusNotDeployed = "not_deployed"
)

// Worker CRUD API types

// CreateWorkerRequest represents the request to create a new worker
type CreateWorkerRequest struct {
	Name     string               `json:"name" validate:"required,min=1,max=255"`
	Prompt   string               `json:"prompt" validate:"required"`
	Enabled  *bool                `json:"enabled,omitempty"`
	Settings *WorkerSettingsInput `json:"settings,omitempty"`
}

// UpdateWorkerRequest represents the request to update an existing worker
type UpdateWorkerRequest struct {
	Name     *string              `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Prompt   *string              `json:"prompt,omitempty"`
	Enabled  *bool                `json:"enabled,omitempty"`
	Settings *WorkerSettingsInput `json:"settings,omitempty"`
}

// WorkerResponse represents a worker with full details
type WorkerResponse struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Prompt    string                `json:"prompt"`
	Enabled   bool                  `json:"enabled"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Settings  *WorkerSettingsOutput `json:"settings,omitempty"`
}

// WorkerSettingsResponse represents worker settings
type WorkerSettingsResponse struct {
	Settings  WorkerSettingsOutput `json:"settings"`
	Timestamp time.Time            `json:"timestamp"`
}

// UpdateWorkerSettingsRequest represents the request to update worker settings
type UpdateWorkerSettingsRequest struct {
	SleepDuration int                        `json:"sleep_duration,omitempty" validate:"omitempty,min=10,max=3600"`
	TeamName      *string                    `json:"team_name,omitempty"`
	InstallDeps   *bool                      `json:"install_deps,omitempty"`
	CommonPrompt  *string                    `json:"common_prompt,omitempty"`
	MaxAttempts   *int                       `json:"max_attempts,omitempty" validate:"omitempty,min=1,max=10"`
	Service       map[string]interface{}     `json:"service,omitempty"`
	MCPServers    map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
	Debug         *bool                      `json:"debug,omitempty"`
	Meta          map[string]interface{}     `json:"meta,omitempty"`
	Flow          []FlowStepInput            `json:"flow,omitempty"`
}

// WorkerSettingsInput represents worker settings input for create/update operations
type WorkerSettingsInput struct {
	SleepDuration int                        `json:"sleep_duration,omitempty" validate:"omitempty,min=10,max=3600"`
	TeamName      *string                    `json:"team_name,omitempty"`
	InstallDeps   *bool                      `json:"install_deps,omitempty"`
	CommonPrompt  *string                    `json:"common_prompt,omitempty"`
	MaxAttempts   *int                       `json:"max_attempts,omitempty" validate:"omitempty,min=1,max=10"`
	Service       map[string]interface{}     `json:"service,omitempty"`
	MCPServers    map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
	Debug         *bool                      `json:"debug,omitempty"`
	Meta          map[string]interface{}     `json:"meta,omitempty"`
	Flow          []FlowStepInput            `json:"flow,omitempty"`
}

// WorkerSettingsOutput represents worker settings output with full details
type WorkerSettingsOutput struct {
	ID            string                     `json:"id"`
	WorkerID      string                     `json:"worker_id"`
	SleepDuration int                        `json:"sleep_duration"`
	TeamName      string                     `json:"team_name"`
	InstallDeps   bool                       `json:"install_deps"`
	CommonPrompt  string                     `json:"common_prompt"`
	MaxAttempts   int                        `json:"max_attempts"`
	Service       map[string]interface{}     `json:"service,omitempty"`
	MCPServers    map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
	Debug         bool                       `json:"debug"`
	Meta          map[string]interface{}     `json:"meta,omitempty"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
	Flow          []FlowStepOutput           `json:"flow,omitempty"`
}

// FlowStepInput represents flow step input for create/update operations
type FlowStepInput struct {
	Name             string              `json:"name" validate:"required"`
	Type             string              `json:"type" validate:"required"`
	Args             []string            `json:"args,omitempty"`
	Env              map[string]string   `json:"env,omitempty"`
	DependsOn        []string            `json:"depends_on,omitempty"`
	Input            string              `json:"input,omitempty"`
	Output           string              `json:"output,omitempty"`
	SkipWhen         string              `json:"skip_when,omitempty"`
	DependencyPolicy string              `json:"dependency_policy,omitempty" validate:"omitempty,oneof=fail_fast all_success all_complete any_success"`
	Retry            *worker.RetryConfig `json:"retry,omitempty"`
}

// FlowStepOutput represents flow step output with full details
type FlowStepOutput struct {
	ID               string              `json:"id"`
	WorkerID         string              `json:"worker_id"`
	Name             string              `json:"name"`
	Type             string              `json:"type"`
	Order            int                 `json:"order"`
	Args             []string            `json:"args,omitempty"`
	Env              map[string]string   `json:"env,omitempty"`
	DependsOn        []string            `json:"depends_on,omitempty"`
	Input            string              `json:"input,omitempty"`
	Output           string              `json:"output,omitempty"`
	SkipWhen         string              `json:"skip_when,omitempty"`
	DependencyPolicy string              `json:"dependency_policy,omitempty"`
	Retry            *worker.RetryConfig `json:"retry,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// MCPServerConfig represents MCP server configuration
type MCPServerConfig struct {
	Command string            `json:"command" validate:"required"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}
