package worker

import (
	"fmt"
)

// GetEffectiveSettingsFromDB returns the effective settings for a worker from database,
// merging global settings with worker-specific overrides
func (w *Worker) GetEffectiveSettingsFromDB(globalSettings WorkerSettings) WorkerSettings {
	// Start with defaults
	effective := WorkerSettings{
		SleepDuration: 60,
		TeamName:      "autoteam",
		InstallDeps:   false,
		CommonPrompt:  "",
		MaxAttempts:   3,
		Debug:         false,
		Service:       make(JSONMap),
		MCPServers:    make(MCPServersMap),
		Meta:          make(JSONMap),
	}

	// Apply global settings first
	if globalSettings.SleepDuration > 0 {
		effective.SleepDuration = globalSettings.SleepDuration
	}
	if globalSettings.TeamName != "" {
		effective.TeamName = globalSettings.TeamName
	}
	effective.InstallDeps = globalSettings.InstallDeps
	if globalSettings.CommonPrompt != "" {
		effective.CommonPrompt = globalSettings.CommonPrompt
	}
	if globalSettings.MaxAttempts > 0 {
		effective.MaxAttempts = globalSettings.MaxAttempts
	}
	effective.Debug = globalSettings.Debug

	// Merge service configurations
	if globalSettings.Service != nil {
		effective.Service = copyJSONMap(globalSettings.Service)
	}

	// Merge MCP server configurations
	if globalSettings.MCPServers != nil {
		effective.MCPServers = copyMCPServersMap(globalSettings.MCPServers)
	}

	// Merge meta configuration
	if globalSettings.Meta != nil {
		effective.Meta = copyJSONMap(globalSettings.Meta)
	}

	// Apply worker-specific settings if they exist
	if w.Settings != nil {
		// Override with worker-specific settings where provided
		if w.Settings.SleepDuration > 0 {
			effective.SleepDuration = w.Settings.SleepDuration
		}
		if w.Settings.TeamName != "" {
			effective.TeamName = w.Settings.TeamName
		}
		effective.InstallDeps = w.Settings.InstallDeps
		if w.Settings.CommonPrompt != "" {
			effective.CommonPrompt = w.Settings.CommonPrompt
		}
		if w.Settings.MaxAttempts > 0 {
			effective.MaxAttempts = w.Settings.MaxAttempts
		}
		effective.Debug = w.Settings.Debug

		// Merge service configurations
		if w.Settings.Service != nil {
			effective.Service = mergeJSONMap(effective.Service, w.Settings.Service)
		}

		// Merge MCP server configurations
		if w.Settings.MCPServers != nil {
			effective.MCPServers = mergeMCPServersMap(effective.MCPServers, w.Settings.MCPServers)
		}

		// Merge meta configuration
		if w.Settings.Meta != nil {
			effective.Meta = mergeJSONMap(effective.Meta, w.Settings.Meta)
		}

		// Copy hooks if present
		if w.Settings.Hooks != nil {
			effective.Hooks = w.Settings.Hooks
		}
	}

	return effective
}

// Helper functions for copying and merging custom types
func copyJSONMap(source JSONMap) JSONMap {
	if source == nil {
		return make(JSONMap)
	}
	result := make(JSONMap, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

func copyMCPServersMap(source MCPServersMap) MCPServersMap {
	if source == nil {
		return make(MCPServersMap)
	}
	result := make(MCPServersMap, len(source))
	for k, v := range source {
		result[k] = MCPServer{
			Command: v.Command,
			Args:    append([]string{}, v.Args...),
			Env:     copyStringMap(v.Env),
		}
	}
	return result
}

func copyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

func mergeJSONMap(base, override JSONMap) JSONMap {
	if base == nil {
		base = make(JSONMap)
	}
	result := copyJSONMap(base)

	for k, v := range override {
		result[k] = v
	}
	return result
}

func mergeMCPServersMap(base, override MCPServersMap) MCPServersMap {
	if base == nil {
		base = make(MCPServersMap)
	}
	result := copyMCPServersMap(base)

	for k, v := range override {
		result[k] = v
	}
	return result
}

// ValidateWorker validates a worker configuration
func ValidateWorker(w *Worker) error {
	if w.Name == "" {
		return fmt.Errorf("worker name is required")
	}

	if w.Settings != nil {
		if w.Settings.SleepDuration < 0 {
			return fmt.Errorf("sleep duration cannot be negative")
		}
		if w.Settings.MaxAttempts < 1 {
			return fmt.Errorf("max attempts must be at least 1")
		}
	}

	// Validate flow steps
	stepNames := make(map[string]bool)
	for i, step := range w.FlowSteps {
		if step.Name == "" {
			return fmt.Errorf("flow step %d must have a name", i)
		}
		if stepNames[step.Name] {
			return fmt.Errorf("duplicate flow step name: %s", step.Name)
		}
		stepNames[step.Name] = true

		if step.Type == "" {
			return fmt.Errorf("flow step %s must have a type", step.Name)
		}
	}

	return nil
}
