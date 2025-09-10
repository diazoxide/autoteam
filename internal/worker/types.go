package worker

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Worker represents a worker configuration
type Worker struct {
	Name     string          `yaml:"name"`
	Prompt   string          `yaml:"prompt"`
	Enabled  *bool           `yaml:"enabled,omitempty"`
	Settings *WorkerSettings `yaml:"settings,omitempty"`
}

// WorkerSettings represents worker-specific settings and configuration
type WorkerSettings struct {
	SleepDuration *int                   `yaml:"sleep_duration,omitempty"`
	TeamName      *string                `yaml:"team_name,omitempty"`
	InstallDeps   *bool                  `yaml:"install_deps,omitempty"`
	CommonPrompt  *string                `yaml:"common_prompt,omitempty"`
	MaxAttempts   *int                   `yaml:"max_attempts,omitempty"`
	HTTPPort      *int                   `yaml:"http_port,omitempty"`
	Service       map[string]interface{} `yaml:"service,omitempty"`
	MCPServers    map[string]MCPServer   `yaml:"mcp_servers,omitempty"`
	Hooks         *HookConfig            `yaml:"hooks,omitempty"`
	Debug         *bool                  `yaml:"debug,omitempty"`
	Meta          map[string]interface{} `yaml:"meta,omitempty"`
	// Dynamic Flow Configuration
	Flow []FlowStep `yaml:"flow"`
}

// RetryConfig defines retry behavior for a flow step
type RetryConfig struct {
	MaxAttempts int    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"` // Default: 1 (no retry)
	Delay       int    `yaml:"delay,omitempty" json:"delay,omitempty"`               // Seconds between retries
	Backoff     string `yaml:"backoff,omitempty" json:"backoff,omitempty"`           // "fixed", "exponential", "linear"
	MaxDelay    int    `yaml:"max_delay,omitempty" json:"max_delay,omitempty"`       // Max delay for backoff strategies
}

// FlowStep represents a single step in a dynamic flow configuration
type FlowStep struct {
	Name             string       `yaml:"name" json:"name"`                                           // Unique step name
	Type             string       `yaml:"type" json:"type"`                                           // Agent type (claude, gemini, qwen)
	Args             []string     `yaml:"args,omitempty" json:"args,omitempty"`                       // Agent-specific arguments
	Env              map[string]string `yaml:"env,omitempty" json:"env,omitempty"`                    // Environment variables
	DependsOn        []string     `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`           // Step dependencies
	Input            string       `yaml:"input,omitempty" json:"input,omitempty"`                     // Agent input prompt (supports templates)
	Output           string       `yaml:"output,omitempty" json:"output,omitempty"`                   // Output transformation template (Sprig)
	SkipWhen         string       `yaml:"skip_when,omitempty" json:"skip_when,omitempty"`             // Skip condition template (if evaluates to "true")
	DependencyPolicy string       `yaml:"dependency_policy,omitempty" json:"dependency_policy,omitempty"` // "fail_fast", "all_success", "all_complete", "any_success"
	Retry            *RetryConfig `yaml:"retry,omitempty" json:"retry,omitempty"`                     // Retry configuration
}

// MCPServer represents a Model Context Protocol server configuration
type MCPServer struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// HookConfig represents worker lifecycle hook-driven script execution configuration
type HookConfig struct {
	OnInit  []HookCommand `yaml:"on_init,omitempty"`  // Before worker initialization
	OnStart []HookCommand `yaml:"on_start,omitempty"` // When worker starts monitoring
	OnStop  []HookCommand `yaml:"on_stop,omitempty"`  // When worker stops
	OnError []HookCommand `yaml:"on_error,omitempty"` // When worker encounters errors
}

// HookCommand represents a command to execute on a worker lifecycle hook
type HookCommand struct {
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	WorkingDir  *string           `yaml:"working_dir,omitempty"`
	Timeout     *int              `yaml:"timeout,omitempty"`     // timeout in seconds
	ContinueOn  *string           `yaml:"continue_on,omitempty"` // "success", "error", "always"
	Description *string           `yaml:"description,omitempty"`
}

// WorkerWithSettings combines a worker with its effective settings
type WorkerWithSettings struct {
	Worker   Worker
	Settings WorkerSettings
}

// GetConsolidatedPrompt returns the worker prompt combined with common prompt
func (wws *WorkerWithSettings) GetConsolidatedPrompt() string {
	var promptParts []string

	// Add worker-specific prompt
	if wws.Worker.Prompt != "" {
		promptParts = append(promptParts, wws.Worker.Prompt)
	}

	// Add common prompt
	if wws.Settings.CommonPrompt != nil && *wws.Settings.CommonPrompt != "" {
		promptParts = append(promptParts, *wws.Settings.CommonPrompt)
	}

	if len(promptParts) == 0 {
		return ""
	}

	return strings.Join(promptParts, "\n\n")
}

// normalizeWorkerName converts worker names to snake_case for use in service names and paths
func normalizeWorkerName(name string) string {
	// Replace any non-alphanumeric characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	normalized := reg.ReplaceAllString(name, "_")

	// Convert to lowercase
	normalized = strings.ToLower(normalized)

	// Remove leading/trailing underscores
	normalized = strings.Trim(normalized, "_")

	// Replace multiple consecutive underscores with single underscore
	multiUnderscoreReg := regexp.MustCompile(`_+`)
	normalized = multiUnderscoreReg.ReplaceAllString(normalized, "_")

	return normalized
}

// GetNormalizedName returns the normalized worker name suitable for service names and paths
func (w *Worker) GetNormalizedName() string {
	return normalizeWorkerName(w.Name)
}

// GetNormalizedNameWithVariation returns the normalized worker name with a variation (e.g., collector, executor)
// for two-layer architecture using subdirectory structure
func (w *Worker) GetNormalizedNameWithVariation(variation string) string {
	normalizedName := normalizeWorkerName(w.Name)
	if variation == "" {
		return normalizedName
	}
	return fmt.Sprintf("%s/%s", normalizedName, variation)
}

// GetWorkerDir returns the worker directory path for use in configurations and volume mounts
func (w *Worker) GetWorkerDir() string {
	return fmt.Sprintf("/opt/autoteam/workers/%s", w.GetNormalizedName())
}

// GetWorkerSubDir returns the worker subdirectory path for a specific variation (e.g., collector, executor)
func (w *Worker) GetWorkerSubDir(variation string) string {
	if variation == "" {
		return w.GetWorkerDir()
	}
	return fmt.Sprintf("%s/%s", w.GetWorkerDir(), variation)
}

// IsEnabled returns true if the worker is enabled (default is true)
func (w *Worker) IsEnabled() bool {
	if w.Enabled == nil {
		return true
	}
	return *w.Enabled
}

// Helper methods to get values with defaults for WorkerSettings
func (s *WorkerSettings) GetSleepDuration() int {
	if s.SleepDuration != nil {
		return *s.SleepDuration
	}
	return 60 // default
}

func (s *WorkerSettings) GetTeamName() string {
	if s.TeamName != nil {
		return *s.TeamName
	}
	return "autoteam" // default
}

func (s *WorkerSettings) GetHTTPPort() int {
	if s.HTTPPort != nil {
		return *s.HTTPPort
	}
	return 8080 // default fixed port for all workers
}

func (s *WorkerSettings) GetInstallDeps() bool {
	if s.InstallDeps != nil {
		return *s.InstallDeps
	}
	return false // default
}

func (s *WorkerSettings) GetCommonPrompt() string {
	if s.CommonPrompt != nil {
		return *s.CommonPrompt
	}
	return "" // default
}

func (s *WorkerSettings) GetMaxAttempts() int {
	if s.MaxAttempts != nil {
		return *s.MaxAttempts
	}
	return 3 // default
}

func (s *WorkerSettings) GetDebug() bool {
	if s.Debug != nil {
		return *s.Debug
	}
	return false // default
}

// StepStats tracks execution statistics for a single flow step
type StepStats struct {
	Enabled              bool       `json:"enabled"`
	Active               bool       `json:"active"`
	LastExecution        *time.Time `json:"last_execution,omitempty"`
	LastExecutionSuccess *bool      `json:"last_execution_success,omitempty"`
	ExecutionCount       int        `json:"execution_count"`
	SuccessCount         int        `json:"success_count"`
	LastOutput           *string    `json:"last_output,omitempty"`
	LastError            *string    `json:"last_error,omitempty"`
	RetryAttempt         int        `json:"retry_attempt"`          // Current retry attempt (0 = first try)
	TotalRetries         int        `json:"total_retries"`          // Total retry attempts made
	LastRetryTime        *time.Time `json:"last_retry_time,omitempty"`
	NextRetryTime        *time.Time `json:"next_retry_time,omitempty"` // When next retry will occur
}

// FlowStats tracks overall flow execution statistics
type FlowStats struct {
	ExecutionCount int        `json:"execution_count"`
	SuccessCount   int        `json:"success_count"`
	LastExecution  *time.Time `json:"last_execution,omitempty"`
}

// Runtime state fields - these would typically be added when the worker is instantiated
type WorkerRuntimeState struct {
	effectiveSettings WorkerSettings
	workingDir        string
	startTime         time.Time
	isRunning         bool
	lastActivity      *time.Time
	flowStats         FlowStats
	stepStats         map[string]*StepStats
	stepStatsMutex    sync.Mutex // Protects stepStats map and individual StepStats fields
}

// Runtime methods for Worker - these operate on runtime state
func (w *Worker) InitRuntime(effectiveSettings WorkerSettings) *WorkerRuntimeState {
	stepStats := make(map[string]*StepStats)

	// Initialize step stats for all flow steps
	for _, step := range effectiveSettings.Flow {
		stepStats[step.Name] = &StepStats{
			Enabled:        true,
			Active:         false,
			ExecutionCount: 0,
			SuccessCount:   0,
		}
	}

	return &WorkerRuntimeState{
		effectiveSettings: effectiveSettings,
		workingDir:        w.GetWorkerDir(),
		startTime:         time.Now(),
		isRunning:         false,
		lastActivity:      nil,
		flowStats: FlowStats{
			ExecutionCount: 0,
			SuccessCount:   0,
		},
		stepStats: stepStats,
	}
}

// Type returns the worker type based on the primary flow step
func (w *Worker) Type(effectiveSettings WorkerSettings) string {
	if len(effectiveSettings.Flow) > 0 {
		return effectiveSettings.Flow[0].Type
	}
	return "unknown"
}

// IsAvailable checks if the worker is available
func (w *Worker) IsAvailable(effectiveSettings WorkerSettings) bool {
	return len(effectiveSettings.Flow) > 0
}

// Version returns worker version information
func (w *Worker) Version() (string, error) {
	return "1.0.0", nil
}

// Runtime state methods
func (rs *WorkerRuntimeState) GetSettings() WorkerSettings {
	return rs.effectiveSettings
}

func (rs *WorkerRuntimeState) GetWorkingDir() string {
	return rs.workingDir
}

func (rs *WorkerRuntimeState) GetTeamName() string {
	return rs.effectiveSettings.GetTeamName()
}

func (rs *WorkerRuntimeState) GetUptime() time.Duration {
	return time.Since(rs.startTime)
}

func (rs *WorkerRuntimeState) IsRunning() bool {
	return rs.isRunning
}

func (rs *WorkerRuntimeState) GetLastActivity() *time.Time {
	return rs.lastActivity
}

func (rs *WorkerRuntimeState) SetRunning(running bool) {
	rs.isRunning = running
	if running {
		now := time.Now()
		rs.lastActivity = &now
	}
}

func (rs *WorkerRuntimeState) UpdateLastActivity() {
	now := time.Now()
	rs.lastActivity = &now
}

// Flow and step statistics access methods
func (rs *WorkerRuntimeState) GetFlowStats() FlowStats {
	return rs.flowStats
}

func (rs *WorkerRuntimeState) GetStepStats(stepName string) *StepStats {
	rs.stepStatsMutex.Lock()
	defer rs.stepStatsMutex.Unlock()

	return rs.stepStats[stepName]
}

func (rs *WorkerRuntimeState) GetAllStepStats() map[string]*StepStats {
	rs.stepStatsMutex.Lock()
	defer rs.stepStatsMutex.Unlock()

	// Return a copy of the map to avoid race conditions
	result := make(map[string]*StepStats, len(rs.stepStats))
	for k, v := range rs.stepStats {
		result[k] = v
	}
	return result
}

// Methods to update step statistics
func (rs *WorkerRuntimeState) SetStepActive(stepName string, active bool) {
	rs.stepStatsMutex.Lock()
	defer rs.stepStatsMutex.Unlock()

	if stats, exists := rs.stepStats[stepName]; exists {
		stats.Active = active
	}
}

func (rs *WorkerRuntimeState) RecordStepExecution(stepName string, success bool, output *string, errorMsg *string) {
	rs.stepStatsMutex.Lock()
	defer rs.stepStatsMutex.Unlock()

	if stats, exists := rs.stepStats[stepName]; exists {
		now := time.Now()
		stats.LastExecution = &now
		stats.LastExecutionSuccess = &success
		stats.ExecutionCount++

		if success {
			stats.SuccessCount++
			// Clear last error on successful execution
			stats.LastError = nil
		} else {
			// Only set error if execution failed
			if errorMsg != nil {
				stats.LastError = errorMsg
			}
		}

		if output != nil {
			// Truncate output if too long
			truncated := *output
			if len(truncated) > 500 {
				truncated = truncated[:500] + "..."
			}
			stats.LastOutput = &truncated
		}
	}
}

// Method to update flow statistics
func (rs *WorkerRuntimeState) RecordFlowExecution(success bool) {
	now := time.Now()
	rs.flowStats.LastExecution = &now
	rs.flowStats.ExecutionCount++
	if success {
		rs.flowStats.SuccessCount++
	}
}

// WorkerRuntime provides runtime functionality for a Worker, satisfying server interface requirements
type WorkerRuntime struct {
	*Worker
	*WorkerRuntimeState
}

// NewWorkerRuntime creates a new WorkerRuntime with runtime functionality
func NewWorkerRuntime(w *Worker, settings WorkerSettings) *WorkerRuntime {
	runtimeState := w.InitRuntime(settings)
	return &WorkerRuntime{
		Worker:             w,
		WorkerRuntimeState: runtimeState,
	}
}

// Server interface methods for WorkerRuntime
// These methods adapt the Worker and WorkerRuntimeState methods to match server expectations

// Type returns the worker type based on effective settings (with server-compatible signature)
func (wi *WorkerRuntime) Type() string {
	return wi.Worker.Type(wi.effectiveSettings)
}

// Version returns worker version (with server-compatible signature)
func (wi *WorkerRuntime) Version(ctx context.Context) (string, error) {
	return wi.Worker.Version()
}

// IsAvailable checks if worker is available (with server-compatible signature)
func (wi *WorkerRuntime) IsAvailable(ctx context.Context) bool {
	return wi.Worker.IsAvailable(wi.effectiveSettings)
}

// GetConfig returns the underlying Worker configuration
func (wi *WorkerRuntime) GetConfig() *Worker {
	return wi.Worker
}

// Additional methods for WorkerRuntime to support handler functionality
func (wi *WorkerRuntime) GetFlowStats() FlowStats {
	return wi.WorkerRuntimeState.GetFlowStats()
}

func (wi *WorkerRuntime) GetStepStats(stepName string) *StepStats {
	return wi.WorkerRuntimeState.GetStepStats(stepName)
}

func (wi *WorkerRuntime) GetAllStepStats() map[string]*StepStats {
	return wi.WorkerRuntimeState.GetAllStepStats()
}

func (wi *WorkerRuntime) SetStepActive(stepName string, active bool) {
	wi.WorkerRuntimeState.SetStepActive(stepName, active)
}

func (wi *WorkerRuntime) RecordStepExecution(stepName string, success bool, output *string, errorMsg *string) {
	wi.WorkerRuntimeState.RecordStepExecution(stepName, success, output, errorMsg)
}

func (wi *WorkerRuntime) RecordFlowExecution(success bool) {
	wi.WorkerRuntimeState.RecordFlowExecution(success)
}
