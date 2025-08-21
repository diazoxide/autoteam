package config

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Default configuration constants
const (
	DefaultTeamName = "autoteam"
)

type Config struct {
	Workers    []Worker                          `yaml:"workers"`
	Services   map[string]map[string]interface{} `yaml:"services,omitempty"`
	Settings   AgentSettings                     `yaml:"settings"`
	MCPServers map[string]MCPServer              `yaml:"mcp_servers,omitempty"`
}

type Worker struct {
	Name       string               `yaml:"name"`
	Prompt     string               `yaml:"prompt"`
	Enabled    *bool                `yaml:"enabled,omitempty"`
	Settings   *AgentSettings       `yaml:"settings,omitempty"`
	MCPServers map[string]MCPServer `yaml:"mcp_servers,omitempty"`
}

type AgentSettings struct {
	SleepDuration *int                   `yaml:"sleep_duration,omitempty"`
	TeamName      *string                `yaml:"team_name,omitempty"`
	InstallDeps   *bool                  `yaml:"install_deps,omitempty"`
	CommonPrompt  *string                `yaml:"common_prompt,omitempty"`
	MaxAttempts   *int                   `yaml:"max_attempts,omitempty"`
	Service       map[string]interface{} `yaml:"service,omitempty"`
	MCPServers    map[string]MCPServer   `yaml:"mcp_servers,omitempty"`
	Hooks         *HookConfig            `yaml:"hooks,omitempty"`
	// Dynamic Flow Configuration
	Flow []FlowStep `yaml:"flow"`
}

// FlowStep represents a single step in a dynamic flow configuration
type FlowStep struct {
	Name         string            `yaml:"name"`                   // Unique step name
	Type         string            `yaml:"type"`                   // Agent type (claude, gemini, qwen)
	Args         []string          `yaml:"args,omitempty"`         // Agent-specific arguments
	Env          map[string]string `yaml:"env,omitempty"`          // Environment variables
	DependsOn    []string          `yaml:"depends_on,omitempty"`   // Step dependencies
	Prompt       string            `yaml:"prompt,omitempty"`       // Step-specific prompt
	SkipWhen     string            `yaml:"skip_when,omitempty"`    // Skip condition template (if evaluates to "true")
	Transformers *Transformers     `yaml:"transformers,omitempty"` // Input/output transformers
}

// Transformers defines template-based data transformation for flow steps
type Transformers struct {
	Input  string `yaml:"input,omitempty"`  // Input transformation template (Sprig)
	Output string `yaml:"output,omitempty"` // Output transformation template (Sprig)
}

// MCPServer represents a Model Context Protocol server configuration
type MCPServer struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// AgentConfig represents unified agent configuration structure
type AgentConfig struct {
	Type   string            `yaml:"type"`
	Args   []string          `yaml:"args,omitempty"`
	Env    map[string]string `yaml:"env,omitempty"`
	Prompt *string           `yaml:"prompt,omitempty"`
}

// HookConfig represents agent lifecycle hook-driven script execution configuration
type HookConfig struct {
	OnInit  []HookCommand `yaml:"on_init,omitempty"`  // Before agent initialization
	OnStart []HookCommand `yaml:"on_start,omitempty"` // When agent starts monitoring
	OnStop  []HookCommand `yaml:"on_stop,omitempty"`  // When agent stops
	OnError []HookCommand `yaml:"on_error,omitempty"` // When agent encounters errors
}

// HookCommand represents a command to execute on an agent lifecycle hook
type HookCommand struct {
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	WorkingDir  *string           `yaml:"working_dir,omitempty"`
	Timeout     *int              `yaml:"timeout,omitempty"`     // timeout in seconds
	ContinueOn  *string           `yaml:"continue_on,omitempty"` // "success", "error", "always"
	Description *string           `yaml:"description,omitempty"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	setDefaults(&config)

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Workers) == 0 {
		return fmt.Errorf("at least one worker must be configured")
	}

	// Count enabled workers
	enabledCount := 0
	for _, worker := range config.Workers {
		if worker.IsEnabled() {
			enabledCount++
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("at least one worker must be enabled")
	}

	for i, worker := range config.Workers {
		if worker.Name == "" {
			return fmt.Errorf("worker[%d].name is required", i)
		}
		// Only validate required fields for enabled workers
		if worker.IsEnabled() {
			if worker.Prompt == "" {
				return fmt.Errorf("worker[%d].prompt is required for enabled workers", i)
			}

			// Get effective settings to check flow configuration
			settings := worker.GetEffectiveSettings(config.Settings)
			if len(settings.Flow) == 0 {
				return fmt.Errorf("worker[%d].flow is required for enabled workers", i)
			}

			// Validate flow steps
			if err := validateFlow(settings.Flow); err != nil {
				return fmt.Errorf("agent[%d].flow validation failed: %w", i, err)
			}
		}
	}

	return nil
}

// validateFlow validates flow configuration
func validateFlow(flow []FlowStep) error {
	if len(flow) == 0 {
		return fmt.Errorf("flow must contain at least one step")
	}

	stepNames := make(map[string]bool)
	for i, step := range flow {
		if step.Name == "" {
			return fmt.Errorf("step[%d].name is required", i)
		}
		if step.Type == "" {
			return fmt.Errorf("step[%d].type is required", i)
		}
		if stepNames[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		stepNames[step.Name] = true

		// Validate dependencies exist
		for _, dep := range step.DependsOn {
			found := false
			for _, otherStep := range flow {
				if otherStep.Name == dep {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("step %s depends on non-existent step: %s", step.Name, dep)
			}
		}
	}

	return nil
}

func setDefaults(config *Config) {
	if config.Settings.SleepDuration == nil {
		config.Settings.SleepDuration = IntPtr(60)
	}
	if config.Settings.TeamName == nil {
		config.Settings.TeamName = StringPtr(DefaultTeamName)
	}
	if config.Settings.MaxAttempts == nil {
		config.Settings.MaxAttempts = IntPtr(3)
	}
	// Set default service configuration if not provided
	if config.Settings.Service == nil {
		config.Settings.Service = map[string]interface{}{
			"image": "node:18.17.1",
			"user":  "developer",
		}
	}
}

func CreateSampleConfig(filename string) error {
	sampleConfig := Config{
		Workers: []Worker{
			{
				Name:   "dev1",
				Prompt: "You are a developer agent responsible for implementing features and fixing bugs.",
			},
			{
				Name:   "arch1",
				Prompt: "You are an architecture agent responsible for system design and code reviews.",
				Settings: &AgentSettings{
					SleepDuration: IntPtr(30),
					Service: map[string]interface{}{
						"image": "python:3.11",
						"volumes": []string{
							"./custom-configs:/app/configs:ro",
							"/var/run/docker.sock:/var/run/docker.sock",
						},
						"environment": map[string]string{
							"PYTHON_PATH": "/app/custom",
							"DEBUG_MODE":  "true",
						},
					},
					Hooks: &HookConfig{
						OnInit: []HookCommand{
							{
								Command:     "/bin/sh",
								Args:        []string{"-c", "echo 'Agent initializing: $AGENT_NAME'"},
								Description: StringPtr("Log agent initialization"),
							},
						},
						OnStart: []HookCommand{
							{
								Command:     "/bin/bash",
								Args:        []string{"-c", "pip install --upgrade pip && pip install requests"},
								Timeout:     IntPtr(60),
								ContinueOn:  StringPtr("always"),
								Description: StringPtr("Install additional Python packages"),
							},
						},
						OnStop: []HookCommand{
							{
								Command:     "/bin/sh",
								Args:        []string{"-c", "echo 'Agent $AGENT_NAME shutting down gracefully'"},
								Description: StringPtr("Log graceful shutdown"),
							},
						},
					},
				},
			},
			{
				Name:    "devops1",
				Prompt:  "You are a DevOps agent responsible for CI/CD and infrastructure.",
				Enabled: BoolPtr(false), // This agent is disabled
			},
		},
		Services: map[string]map[string]interface{}{
			"postgres": {
				"image": "postgres:15",
				"environment": map[string]string{
					"POSTGRES_DB":       "autoteam_dev",
					"POSTGRES_USER":     "autoteam",
					"POSTGRES_PASSWORD": "development_password",
				},
				"ports": []string{"5432:5432"},
				"volumes": []string{
					"postgres_data:/var/lib/postgresql/data",
					"./sql/init.sql:/docker-entrypoint-initdb.d/init.sql:ro",
				},
			},
			"redis": {
				"image":   "redis:7",
				"ports":   []string{"6379:6379"},
				"volumes": []string{"redis_data:/data"},
			},
		},
		Settings: AgentSettings{
			SleepDuration: IntPtr(60),
			TeamName:      StringPtr(DefaultTeamName),
			InstallDeps:   BoolPtr(true),
			CommonPrompt:  StringPtr("Always follow coding best practices and write comprehensive tests."),
			MaxAttempts:   IntPtr(3),
			Service: map[string]interface{}{
				"image": "node:18.17.1",
				"user":  "developer",
			},
			Flow: []FlowStep{
				{
					Name:   "collector",
					Type:   "gemini",
					Args:   []string{"--model", "gemini-2.5-flash"},
					Prompt: "You are a notification collector. Get unread GitHub notifications and list them.\nUse GitHub MCP to get unread notifications.\nCRITICAL: Mark all notifications as read after collecting them.",
					Transformers: &Transformers{
						Output: "{{ .stdout | trim }}",
					},
				},
				{
					Name:      "analyzer",
					Type:      "claude",
					DependsOn: []string{"collector"},
					Prompt:    "You are the GitHub Notification Handler. Process GitHub notifications exactly like a human would.\n\nFor each notification:\n1. Read the full context (issues, PRs, comments, code)\n2. Respond naturally as a project contributor\n3. Take appropriate action (comment, review, create PR, etc.)\n4. Use GitHub MCP to publish your responses\n\nAlways be professional, helpful, and maintain high quality standards.",
					Transformers: &Transformers{
						Input: "{{ index .inputs 0 }}",
					},
				},
			},
		},
		MCPServers: map[string]MCPServer{
			"memory": {
				Command: "npx",
				Args:    []string{"-y", "mcp-memory-service"},
			},
		},
	}

	data, err := yaml.Marshal(&sampleConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal sample config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}

// mergeServiceConfigs merges global and agent service configurations
// Agent service properties override global ones, with special handling for maps and arrays
func mergeServiceConfigs(global, agent map[string]interface{}) map[string]interface{} {
	if global == nil && agent == nil {
		return nil
	}
	if global == nil {
		return copyServiceConfig(agent)
	}
	if agent == nil {
		return copyServiceConfig(global)
	}

	// Start with a copy of global config
	result := copyServiceConfig(global)

	// Override/merge with agent config
	for key, agentValue := range agent {
		globalValue, exists := result[key]

		// If key doesn't exist in global, just add it
		if !exists {
			result[key] = agentValue
			continue
		}

		// Universal map merging - merge any map-type values recursively
		if merged := tryMergeAsMapRecursive(globalValue, agentValue); merged != nil {
			result[key] = merged
			continue
		}

		// For all other properties (including arrays like volumes, ports), agent replaces global
		result[key] = agentValue
	}

	return result
}

// tryMergeAsMapRecursive attempts to merge two values as maps recursively using golang maps package
// Returns the merged map if successful, nil if values aren't compatible maps
func tryMergeAsMapRecursive(globalValue, agentValue interface{}) interface{} {
	// Try map[string]string first (most common for environment, labels, etc.)
	if globalMap, ok := globalValue.(map[string]string); ok {
		if agentMap, ok := agentValue.(map[string]string); ok {
			// Use maps.Clone for efficient copying, then merge
			merged := maps.Clone(globalMap)
			maps.Copy(merged, agentMap) // Agent values override global
			return merged
		}
	}

	// Try map[string]interface{} (common after YAML unmarshaling) with recursive merging
	if globalMap, ok := globalValue.(map[string]interface{}); ok {
		if agentMap, ok := agentValue.(map[string]interface{}); ok {
			// Use maps.Clone for efficient copying
			merged := maps.Clone(globalMap)

			// Recursively merge/override with agent values
			for k, agentVal := range agentMap {
				globalVal, exists := merged[k]

				if !exists {
					// Key doesn't exist in global, just add it (deep copy)
					merged[k] = deepCopyValue(agentVal)
				} else {
					// Try recursive merge for nested maps
					if recursiveMerged := tryMergeAsMapRecursive(globalVal, agentVal); recursiveMerged != nil {
						merged[k] = recursiveMerged
					} else {
						// Not mergeable maps, agent value replaces global (deep copy)
						merged[k] = deepCopyValue(agentVal)
					}
				}
			}
			return merged
		}
	}

	// Try mixed map types - convert map[string]string to map[string]interface{}
	if globalMap, ok := globalValue.(map[string]string); ok {
		if agentMap, ok := agentValue.(map[string]interface{}); ok {
			// Convert global to interface{} map and clone
			merged := make(map[string]interface{})
			for k, v := range globalMap {
				merged[k] = v
			}

			// Recursively merge/override with agent values
			for k, agentVal := range agentMap {
				globalVal, exists := merged[k]

				if !exists {
					merged[k] = deepCopyValue(agentVal)
				} else {
					if recursiveMerged := tryMergeAsMapRecursive(globalVal, agentVal); recursiveMerged != nil {
						merged[k] = recursiveMerged
					} else {
						merged[k] = deepCopyValue(agentVal)
					}
				}
			}
			return merged
		}
	}

	// Try reverse mixed types - convert map[string]interface{} to accommodate map[string]string
	if globalMap, ok := globalValue.(map[string]interface{}); ok {
		if agentMap, ok := agentValue.(map[string]string); ok {
			// Use maps.Clone for efficient copying
			merged := maps.Clone(globalMap)

			// Override with agent values (convert to interface{})
			for k, v := range agentMap {
				merged[k] = v
			}
			return merged
		}
	}

	// Values aren't compatible maps
	return nil
}

// deepCopyValue creates a deep copy of various value types
func deepCopyValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]string:
		return maps.Clone(v) // Use efficient maps.Clone
	case map[string]interface{}:
		// Deep copy for nested interface{} maps
		copied := make(map[string]interface{})
		for k, val := range v {
			copied[k] = deepCopyValue(val) // Recursive deep copy
		}
		return copied
	case []string:
		// Use slices.Clone if available, otherwise manual copy
		copied := make([]string, len(v))
		copy(copied, v)
		return copied
	case []interface{}:
		// Deep copy for slice of interfaces
		copied := make([]interface{}, len(v))
		for i, val := range v {
			copied[i] = deepCopyValue(val) // Recursive deep copy
		}
		return copied
	default:
		// For primitive types (string, int, bool, etc.), direct assignment is fine
		return value
	}
}

// copyServiceConfig creates a deep copy of a service configuration map
func copyServiceConfig(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range source {
		// Special handling for map types (like environment)
		if envMap, ok := value.(map[string]string); ok {
			newEnvMap := make(map[string]string)
			for k, v := range envMap {
				newEnvMap[k] = v
			}
			result[key] = newEnvMap
		} else if strSlice, ok := value.([]string); ok {
			// Copy string slices (like volumes)
			newSlice := make([]string, len(strSlice))
			copy(newSlice, strSlice)
			result[key] = newSlice
		} else {
			// For other types, direct assignment (should be safe for scalars)
			result[key] = value
		}
	}
	return result
}

// mergeMCPServers merges MCP server configurations from global settings, agent settings, and agent-level MCP servers
// Priority: agent-level MCPServers > agent.settings.MCPServers > global settings MCPServers
func mergeMCPServers(globalMCPServers, agentSettingsMCPServers, agentMCPServers map[string]MCPServer) map[string]MCPServer {
	if globalMCPServers == nil && agentSettingsMCPServers == nil && agentMCPServers == nil {
		return nil
	}

	result := make(map[string]MCPServer)

	// Start with global MCP servers
	for name, server := range globalMCPServers {
		result[name] = copyMCPServer(server)
	}

	// Override with agent settings MCP servers
	for name, server := range agentSettingsMCPServers {
		result[name] = copyMCPServer(server)
	}

	// Override with agent-level MCP servers (highest priority)
	for name, server := range agentMCPServers {
		result[name] = copyMCPServer(server)
	}

	return result
}

// copyMCPServer creates a deep copy of an MCPServer
func copyMCPServer(server MCPServer) MCPServer {
	copied := MCPServer{
		Command: server.Command,
	}

	// Copy args slice
	if server.Args != nil {
		copied.Args = make([]string, len(server.Args))
		copy(copied.Args, server.Args)
	}

	// Copy env map using maps.Clone
	if server.Env != nil {
		copied.Env = maps.Clone(server.Env)
	}

	return copied
}

// copyHookConfig creates a deep copy of a HookConfig
func copyHookConfig(source *HookConfig) *HookConfig {
	if source == nil {
		return nil
	}

	copied := &HookConfig{}

	// Copy each hook command slice
	if source.OnInit != nil {
		copied.OnInit = copyHookCommands(source.OnInit)
	}
	if source.OnStart != nil {
		copied.OnStart = copyHookCommands(source.OnStart)
	}
	if source.OnStop != nil {
		copied.OnStop = copyHookCommands(source.OnStop)
	}
	if source.OnError != nil {
		copied.OnError = copyHookCommands(source.OnError)
	}

	return copied
}

// copyHookCommands creates a deep copy of a slice of HookCommand
func copyHookCommands(source []HookCommand) []HookCommand {
	if source == nil {
		return nil
	}

	copied := make([]HookCommand, len(source))
	for i, cmd := range source {
		copied[i] = HookCommand{
			Command: cmd.Command,
		}

		// Copy args slice
		if cmd.Args != nil {
			copied[i].Args = make([]string, len(cmd.Args))
			copy(copied[i].Args, cmd.Args)
		}

		// Copy env map
		if cmd.Env != nil {
			copied[i].Env = maps.Clone(cmd.Env)
		}

		// Copy optional fields
		if cmd.WorkingDir != nil {
			copied[i].WorkingDir = StringPtr(*cmd.WorkingDir)
		}
		if cmd.Timeout != nil {
			copied[i].Timeout = IntPtr(*cmd.Timeout)
		}
		if cmd.ContinueOn != nil {
			copied[i].ContinueOn = StringPtr(*cmd.ContinueOn)
		}
		if cmd.Description != nil {
			copied[i].Description = StringPtr(*cmd.Description)
		}
	}

	return copied
}

// mergeHookConfigs merges hook configurations with agent-level overriding global
func mergeHookConfigs(global, agentLevel *HookConfig) *HookConfig {
	if agentLevel != nil {
		return copyHookConfig(agentLevel)
	}
	if global != nil {
		return copyHookConfig(global)
	}
	return nil
}

// copyAgentSettings creates a deep copy of an AgentSettings
func copyAgentSettings(source AgentSettings) AgentSettings {
	copied := AgentSettings{}

	if source.SleepDuration != nil {
		copied.SleepDuration = IntPtr(*source.SleepDuration)
	}
	if source.TeamName != nil {
		copied.TeamName = StringPtr(*source.TeamName)
	}
	if source.InstallDeps != nil {
		copied.InstallDeps = BoolPtr(*source.InstallDeps)
	}
	if source.CommonPrompt != nil {
		copied.CommonPrompt = StringPtr(*source.CommonPrompt)
	}
	if source.MaxAttempts != nil {
		copied.MaxAttempts = IntPtr(*source.MaxAttempts)
	}

	// Copy service configuration
	if source.Service != nil {
		copied.Service = copyServiceConfig(source.Service)
	}

	// Copy MCP servers
	if source.MCPServers != nil {
		copied.MCPServers = make(map[string]MCPServer)
		for k, v := range source.MCPServers {
			copied.MCPServers[k] = copyMCPServer(v)
		}
	}

	// Copy hooks configuration
	copied.Hooks = copyHookConfig(source.Hooks)

	// Copy flow configuration
	if len(source.Flow) > 0 {
		copied.Flow = make([]FlowStep, len(source.Flow))
		copy(copied.Flow, source.Flow)
	}

	return copied
}

// GetEffectiveSettings returns the effective settings for an agent,
// merging global settings with agent-specific overrides
func (w *Worker) GetEffectiveSettings(globalSettings AgentSettings) AgentSettings {
	effective := copyAgentSettings(globalSettings) // Start with copy of global settings

	// Always merge MCP servers, even if agent settings is nil
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, nil, w.MCPServers)

	if w.Settings == nil {
		return effective
	}

	// Override with worker-specific settings where provided
	if w.Settings.SleepDuration != nil {
		effective.SleepDuration = w.Settings.SleepDuration
	}
	if w.Settings.TeamName != nil {
		effective.TeamName = w.Settings.TeamName
	}
	if w.Settings.InstallDeps != nil {
		effective.InstallDeps = w.Settings.InstallDeps
	}
	if w.Settings.CommonPrompt != nil {
		effective.CommonPrompt = w.Settings.CommonPrompt
	}
	if w.Settings.MaxAttempts != nil {
		effective.MaxAttempts = w.Settings.MaxAttempts
	}

	// Merge service configurations
	if len(w.Settings.Service) > 0 {
		effective.Service = mergeServiceConfigs(globalSettings.Service, w.Settings.Service)
	}

	// Merge MCP server configurations
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, w.Settings.MCPServers, w.MCPServers)

	// Merge hooks configuration
	effective.Hooks = mergeHookConfigs(globalSettings.Hooks, w.Settings.Hooks)

	// Merge flow configuration - agent settings override global
	if len(w.Settings.Flow) > 0 {
		effective.Flow = make([]FlowStep, len(w.Settings.Flow))
		copy(effective.Flow, w.Settings.Flow)
	} else if len(globalSettings.Flow) > 0 {
		effective.Flow = make([]FlowStep, len(globalSettings.Flow))
		copy(effective.Flow, globalSettings.Flow)
	}

	return effective
}

// GetAllWorkersWithEffectiveSettings returns a slice of workers with their effective settings
func (c *Config) GetAllWorkersWithEffectiveSettings() []WorkerWithSettings {
	var workers []WorkerWithSettings
	for _, worker := range c.Workers {
		workers = append(workers, WorkerWithSettings{
			Worker:   worker,
			Settings: worker.GetEffectiveSettings(c.Settings),
		})
	}
	return workers
}

// GetEnabledWorkersWithEffectiveSettings returns only enabled workers with their effective settings
func (c *Config) GetEnabledWorkersWithEffectiveSettings() []WorkerWithSettings {
	var workers []WorkerWithSettings
	for _, worker := range c.Workers {
		if worker.IsEnabled() {
			workers = append(workers, WorkerWithSettings{
				Worker:   worker,
				Settings: worker.GetEffectiveSettings(c.Settings),
			})
		}
	}
	return workers
}

type WorkerWithSettings struct {
	Worker   Worker
	Settings AgentSettings
}

// GetConsolidatedPrompt returns the worker prompt combined with common prompt
func (wws *WorkerWithSettings) GetConsolidatedPrompt(cfg *Config) string {
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

// normalizeAgentName converts agent names to snake_case for use in service names and paths
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

// GetNormalizedName returns the normalized agent name suitable for service names and paths
func (w *Worker) GetNormalizedName() string {
	return normalizeWorkerName(w.Name)
}

// GetNormalizedNameWithVariation returns the normalized agent name with a variation (e.g., collector, executor)
// for two-layer architecture using subdirectory structure
func (w *Worker) GetNormalizedNameWithVariation(variation string) string {
	normalizedName := normalizeWorkerName(w.Name)
	if variation == "" {
		return normalizedName
	}
	return fmt.Sprintf("%s/%s", normalizedName, variation)
}

// GetAgentDir returns the agent directory path for use in configurations and volume mounts
func (w *Worker) GetWorkerDir() string {
	return fmt.Sprintf("/opt/autoteam/agents/%s", w.GetNormalizedName())
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

// StringPtr returns a pointer to the given string value. Suitable for optional string parameters or configurations.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int value. Suitable for optional int parameters or configurations.
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to the given boolean value.
func BoolPtr(b bool) *bool {
	return &b
}

// Helper methods to get values with defaults for AgentSettings
func (s *AgentSettings) GetSleepDuration() int {
	if s.SleepDuration != nil {
		return *s.SleepDuration
	}
	return 60 // default
}

func (s *AgentSettings) GetTeamName() string {
	if s.TeamName != nil {
		return *s.TeamName
	}
	return DefaultTeamName // default
}

func (s *AgentSettings) GetInstallDeps() bool {
	if s.InstallDeps != nil {
		return *s.InstallDeps
	}
	return false // default
}

func (s *AgentSettings) GetCommonPrompt() string {
	if s.CommonPrompt != nil {
		return *s.CommonPrompt
	}
	return "" // default
}

func (s *AgentSettings) GetMaxAttempts() int {
	if s.MaxAttempts != nil {
		return *s.MaxAttempts
	}
	return 3 // default
}
