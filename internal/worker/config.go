package worker

import (
	"fmt"
	"maps"
	"os"

	"autoteam/internal/util"

	"gopkg.in/yaml.v3"
)

// LoadWorkerFromFile loads worker configuration directly from a YAML file
func LoadWorkerFromFile(configPath string) (*Worker, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var worker Worker
	if err := yaml.Unmarshal(data, &worker); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Validate worker configuration
	if worker.Name == "" {
		return nil, fmt.Errorf("worker name is required")
	}

	return &worker, nil
}

// GetEffectiveSettings returns the effective settings for a worker,
// merging global settings with worker-specific overrides
func (w *Worker) GetEffectiveSettings(globalSettings WorkerSettings) WorkerSettings {
	effective := copyWorkerSettings(globalSettings) // Start with copy of global settings

	// Always merge MCP servers, even if worker settings is nil
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, nil)

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
	if w.Settings.HTTPPort != nil {
		effective.HTTPPort = w.Settings.HTTPPort
	}

	// Merge service configurations
	if len(w.Settings.Service) > 0 {
		effective.Service = mergeServiceConfigs(globalSettings.Service, w.Settings.Service)
	}

	// Merge MCP server configurations
	effective.MCPServers = mergeMCPServers(globalSettings.MCPServers, w.Settings.MCPServers)

	// Merge hooks configuration
	effective.Hooks = mergeHookConfigs(globalSettings.Hooks, w.Settings.Hooks)

	// Override debug flag
	if w.Settings.Debug != nil {
		effective.Debug = w.Settings.Debug
	}

	// Merge meta configuration using universal map merging
	if len(w.Settings.Meta) > 0 {
		if merged := tryMergeAsMapRecursive(globalSettings.Meta, w.Settings.Meta); merged != nil {
			effective.Meta = merged.(map[string]interface{})
		} else {
			effective.Meta = deepCopyValue(w.Settings.Meta).(map[string]interface{})
		}
	}

	// Merge flow configuration - worker settings override global
	if len(w.Settings.Flow) > 0 {
		effective.Flow = make([]FlowStep, len(w.Settings.Flow))
		copy(effective.Flow, w.Settings.Flow)
	} else if len(globalSettings.Flow) > 0 {
		effective.Flow = make([]FlowStep, len(globalSettings.Flow))
		copy(effective.Flow, globalSettings.Flow)
	}

	return effective
}

// mergeServiceConfigs merges global and worker service configurations
// Worker service properties override global ones, with special handling for maps and arrays
func mergeServiceConfigs(global, worker map[string]interface{}) map[string]interface{} {
	if global == nil && worker == nil {
		return nil
	}
	if global == nil {
		return copyServiceConfig(worker)
	}
	if worker == nil {
		return copyServiceConfig(global)
	}

	// Start with a copy of global config
	result := copyServiceConfig(global)

	// Override/merge with worker config
	for key, workerValue := range worker {
		globalValue, exists := result[key]

		// If key doesn't exist in global, just add it
		if !exists {
			result[key] = workerValue
			continue
		}

		// Universal map merging - merge any map-type values recursively
		if merged := tryMergeAsMapRecursive(globalValue, workerValue); merged != nil {
			result[key] = merged
			continue
		}

		// For all other properties (including arrays like volumes, ports), worker replaces global
		result[key] = workerValue
	}

	return result
}

// tryMergeAsMapRecursive attempts to merge two values as maps recursively using golang maps package
// Returns the merged map if successful, nil if values aren't compatible maps
func tryMergeAsMapRecursive(globalValue, workerValue interface{}) interface{} {
	// Try map[string]string first (most common for environment, labels, etc.)
	if globalMap, ok := globalValue.(map[string]string); ok {
		if workerMap, ok := workerValue.(map[string]string); ok {
			// Use maps.Clone for efficient copying, then merge
			merged := maps.Clone(globalMap)
			maps.Copy(merged, workerMap) // Agent values override global
			return merged
		}
	}

	// Try map[string]interface{} (common after YAML unmarshaling) with recursive merging
	if globalMap, ok := globalValue.(map[string]interface{}); ok {
		if workerMap, ok := workerValue.(map[string]interface{}); ok {
			var merged map[string]interface{}

			// Handle nil global map
			if globalMap == nil {
				merged = make(map[string]interface{})
			} else {
				// Use maps.Clone for efficient copying
				merged = maps.Clone(globalMap)
			}

			// Recursively merge/override with worker values
			for k, workerVal := range workerMap {
				globalVal, exists := merged[k]

				if !exists {
					// Key doesn't exist in global, just add it (deep copy)
					merged[k] = deepCopyValue(workerVal)
				} else {
					// Try recursive merge for nested maps
					if recursiveMerged := tryMergeAsMapRecursive(globalVal, workerVal); recursiveMerged != nil {
						merged[k] = recursiveMerged
					} else {
						// Not mergeable maps, worker value replaces global (deep copy)
						merged[k] = deepCopyValue(workerVal)
					}
				}
			}
			return merged
		}
	}

	// Handle case where global is nil but worker is a map
	if globalValue == nil {
		if workerMap, ok := workerValue.(map[string]interface{}); ok {
			return deepCopyValue(workerMap)
		}
	}

	// Try mixed map types - convert map[string]string to map[string]interface{}
	if globalMap, ok := globalValue.(map[string]string); ok {
		if workerMap, ok := workerValue.(map[string]interface{}); ok {
			// Convert global to interface{} map and clone
			merged := make(map[string]interface{})
			for k, v := range globalMap {
				merged[k] = v
			}

			// Recursively merge/override with worker values
			for k, workerVal := range workerMap {
				globalVal, exists := merged[k]

				if !exists {
					merged[k] = deepCopyValue(workerVal)
				} else {
					if recursiveMerged := tryMergeAsMapRecursive(globalVal, workerVal); recursiveMerged != nil {
						merged[k] = recursiveMerged
					} else {
						merged[k] = deepCopyValue(workerVal)
					}
				}
			}
			return merged
		}
	}

	// Try reverse mixed types - convert map[string]interface{} to accommodate map[string]string
	if globalMap, ok := globalValue.(map[string]interface{}); ok {
		if workerMap, ok := workerValue.(map[string]string); ok {
			// Use maps.Clone for efficient copying
			merged := maps.Clone(globalMap)

			// Override with worker values (convert to interface{})
			for k, v := range workerMap {
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

// mergeMCPServers merges MCP server configurations from global settings and worker settings
// Priority: worker.settings.MCPServers > global settings MCPServers
func mergeMCPServers(globalMCPServers, workerSettingsMCPServers map[string]MCPServer) map[string]MCPServer {
	if globalMCPServers == nil && workerSettingsMCPServers == nil {
		return nil
	}

	result := make(map[string]MCPServer)

	// Start with global MCP servers
	for name, server := range globalMCPServers {
		result[name] = copyMCPServer(server)
	}

	// Override with worker settings MCP servers
	for name, server := range workerSettingsMCPServers {
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
			copied[i].WorkingDir = util.StringPtr(*cmd.WorkingDir)
		}
		if cmd.Timeout != nil {
			copied[i].Timeout = util.IntPtr(*cmd.Timeout)
		}
		if cmd.ContinueOn != nil {
			copied[i].ContinueOn = util.StringPtr(*cmd.ContinueOn)
		}
		if cmd.Description != nil {
			copied[i].Description = util.StringPtr(*cmd.Description)
		}
	}

	return copied
}

// mergeHookConfigs merges hook configurations with worker-level overriding global
func mergeHookConfigs(global, workerLevel *HookConfig) *HookConfig {
	if workerLevel != nil {
		return copyHookConfig(workerLevel)
	}
	if global != nil {
		return copyHookConfig(global)
	}
	return nil
}

// copyWorkerSettings creates a deep copy of a WorkerSettings
func copyWorkerSettings(source WorkerSettings) WorkerSettings {
	copied := WorkerSettings{}

	if source.SleepDuration != nil {
		copied.SleepDuration = util.IntPtr(*source.SleepDuration)
	}
	if source.TeamName != nil {
		copied.TeamName = util.StringPtr(*source.TeamName)
	}
	if source.InstallDeps != nil {
		copied.InstallDeps = util.BoolPtr(*source.InstallDeps)
	}
	if source.CommonPrompt != nil {
		copied.CommonPrompt = util.StringPtr(*source.CommonPrompt)
	}
	if source.MaxAttempts != nil {
		copied.MaxAttempts = util.IntPtr(*source.MaxAttempts)
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

	// Copy debug flag
	if source.Debug != nil {
		copied.Debug = util.BoolPtr(*source.Debug)
	}

	// Copy meta configuration
	if source.Meta != nil {
		copied.Meta = deepCopyValue(source.Meta).(map[string]interface{})
	}

	// Copy flow configuration
	if len(source.Flow) > 0 {
		copied.Flow = make([]FlowStep, len(source.Flow))
		copy(copied.Flow, source.Flow)
	}

	return copied
}
