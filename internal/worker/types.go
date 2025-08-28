package worker

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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

// FlowStep represents a single step in a dynamic flow configuration
type FlowStep struct {
	Name      string            `yaml:"name" json:"name"`                                 // Unique step name
	Type      string            `yaml:"type" json:"type"`                                 // Agent type (claude, gemini, qwen)
	Args      []string          `yaml:"args,omitempty" json:"args,omitempty"`             // Agent-specific arguments
	Env       map[string]string `yaml:"env,omitempty" json:"env,omitempty"`               // Environment variables
	DependsOn []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"` // Step dependencies
	Input     string            `yaml:"input,omitempty" json:"input,omitempty"`           // Agent input prompt (supports templates)
	Output    string            `yaml:"output,omitempty" json:"output,omitempty"`         // Output transformation template (Sprig)
	SkipWhen  string            `yaml:"skip_when,omitempty" json:"skip_when,omitempty"`   // Skip condition template (if evaluates to "true")
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
	return 0 // default - dynamic port discovery
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

// Runtime state fields - these would typically be added when the worker is instantiated
type WorkerRuntimeState struct {
	effectiveSettings WorkerSettings
	workingDir        string
	startTime         time.Time
	isRunning         bool
	lastActivity      *time.Time
}

// Runtime methods for Worker - these operate on runtime state
func (w *Worker) InitRuntime(effectiveSettings WorkerSettings) *WorkerRuntimeState {
	return &WorkerRuntimeState{
		effectiveSettings: effectiveSettings,
		workingDir:        w.GetWorkerDir(),
		startTime:         time.Now(),
		isRunning:         false,
		lastActivity:      nil,
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

// WorkerImpl provides runtime functionality for a Worker, satisfying server interface requirements
type WorkerImpl struct {
	*Worker
	*WorkerRuntimeState
}

// NewWorkerImpl creates a new WorkerImpl with runtime functionality
func NewWorkerImpl(w *Worker, settings WorkerSettings) *WorkerImpl {
	runtimeState := w.InitRuntime(settings)
	return &WorkerImpl{
		Worker:             w,
		WorkerRuntimeState: runtimeState,
	}
}

// NewWorker creates a new WorkerImpl instance (alias for NewWorkerImpl for compatibility)
func NewWorker(w *Worker, settings WorkerSettings) *WorkerImpl {
	return NewWorkerImpl(w, settings)
}

// Server interface methods for WorkerImpl
// These methods adapt the Worker and WorkerRuntimeState methods to match server expectations

// Type returns the worker type based on effective settings (with server-compatible signature)
func (wi *WorkerImpl) Type() string {
	return wi.Worker.Type(wi.effectiveSettings)
}

// Version returns worker version (with server-compatible signature)
func (wi *WorkerImpl) Version(ctx context.Context) (string, error) {
	return wi.Worker.Version()
}

// IsAvailable checks if worker is available (with server-compatible signature)
func (wi *WorkerImpl) IsAvailable(ctx context.Context) bool {
	return wi.Worker.IsAvailable(wi.effectiveSettings)
}

// GetConfig returns the underlying Worker configuration
func (wi *WorkerImpl) GetConfig() *Worker {
	return wi.Worker
}
