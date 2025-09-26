package controlplane

import (
	"autoteam/internal/types"
	"autoteam/internal/worker"

	"github.com/google/uuid"
)

// convertWorkerToResponse converts a domain worker entity to API response format
func (h *Handlers) convertWorkerToResponse(w *worker.Worker) *types.WorkerResponse {
	if w == nil {
		return nil
	}

	response := &types.WorkerResponse{
		ID:        w.ID.String(),
		Name:      w.Name,
		Prompt:    w.Prompt,
		Enabled:   w.Enabled,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}

	// Add settings if available
	if w.Settings != nil {
		response.Settings = convertWorkerSettingsToOutput(w.Settings, w.FlowSteps)
	}

	return response
}

// convertWorkerSettingsToOutput converts domain settings to API output format
func convertWorkerSettingsToOutput(settings *worker.WorkerSettings, flowSteps []worker.FlowStep) *types.WorkerSettingsOutput {
	if settings == nil {
		return nil
	}

	// Convert MCP servers
	mcpServers := make(map[string]types.MCPServerConfig)
	for name, server := range settings.MCPServers {
		mcpServers[name] = types.MCPServerConfig{
			Command: server.Command,
			Args:    server.Args,
			Env:     server.Env,
		}
	}

	output := &types.WorkerSettingsOutput{
		ID:            settings.ID.String(),
		WorkerID:      settings.WorkerID.String(),
		SleepDuration: settings.SleepDuration,
		TeamName:      settings.TeamName,
		InstallDeps:   settings.InstallDeps,
		CommonPrompt:  settings.CommonPrompt,
		MaxAttempts:   settings.MaxAttempts,
		Service:       settings.Service,
		MCPServers:    mcpServers,
		Debug:         settings.Debug,
		Meta:          settings.Meta,
		CreatedAt:     settings.CreatedAt,
		UpdatedAt:     settings.UpdatedAt,
		Flow:          make([]types.FlowStepOutput, len(flowSteps)),
	}

	// Convert flow steps
	for i, step := range flowSteps {
		output.Flow[i] = types.FlowStepOutput{
			ID:               step.ID.String(),
			WorkerID:         step.WorkerID.String(),
			Name:             step.Name,
			Type:             step.Type,
			Order:            step.Order,
			Args:             step.Args,
			Env:              step.Env,
			DependsOn:        step.DependsOn,
			Input:            step.Input,
			Output:           step.Output,
			SkipWhen:         step.SkipWhen,
			DependencyPolicy: step.DependencyPolicy,
			Retry:            step.Retry,
			CreatedAt:        step.CreatedAt,
			UpdatedAt:        step.UpdatedAt,
		}
	}

	return output
}

// createWorkerFromRequest creates a domain worker entity from API request
func createWorkerFromRequest(req *types.CreateWorkerRequest) *worker.Worker {
	w := &worker.Worker{
		ID:      uuid.New(),
		Name:    req.Name,
		Prompt:  req.Prompt,
		Enabled: true, // default to enabled
	}

	if req.Enabled != nil {
		w.Enabled = *req.Enabled
	}

	return w
}

// createWorkerSettingsFromRequest creates domain settings from API request
func createWorkerSettingsFromRequest(req *types.CreateWorkerRequest, workerID uuid.UUID) (*worker.WorkerSettings, []worker.FlowStep) {
	if req.Settings == nil {
		return nil, nil
	}

	// Create settings with defaults
	settings := &worker.WorkerSettings{
		ID:            uuid.New(),
		WorkerID:      workerID,
		SleepDuration: req.Settings.SleepDuration,
		TeamName:      "autoteam", // default
		InstallDeps:   false,      // default
		MaxAttempts:   3,          // default
		Debug:         false,      // default
		MCPServers:    make(worker.MCPServersMap),
		Service:       make(map[string]interface{}),
		Meta:          make(map[string]interface{}),
	}

	// Set optional fields
	if req.Settings.TeamName != nil {
		settings.TeamName = *req.Settings.TeamName
	}
	if req.Settings.InstallDeps != nil {
		settings.InstallDeps = *req.Settings.InstallDeps
	}
	if req.Settings.MaxAttempts != nil {
		settings.MaxAttempts = *req.Settings.MaxAttempts
	}
	if req.Settings.CommonPrompt != nil {
		settings.CommonPrompt = *req.Settings.CommonPrompt
	}
	if req.Settings.Debug != nil {
		settings.Debug = *req.Settings.Debug
	}

	// Convert MCP servers
	if req.Settings.MCPServers != nil {
		for name, server := range req.Settings.MCPServers {
			settings.MCPServers[name] = worker.MCPServer{
				Command: server.Command,
				Args:    server.Args,
				Env:     server.Env,
			}
		}
	}

	// Convert service config
	if req.Settings.Service != nil {
		settings.Service = req.Settings.Service
	}

	// Convert meta
	if req.Settings.Meta != nil {
		settings.Meta = req.Settings.Meta
	}

	// Create flow steps
	var flowSteps []worker.FlowStep
	if req.Settings.Flow != nil {
		flowSteps = make([]worker.FlowStep, len(req.Settings.Flow))
		for i, step := range req.Settings.Flow {
			flowStep := worker.FlowStep{
				ID:               uuid.New(),
				WorkerID:         workerID,
				Name:             step.Name,
				Type:             step.Type,
				Order:            i,
				Args:             worker.StringSlice(step.Args),
				Env:              worker.StringMap(step.Env),
				DependsOn:        worker.StringSlice(step.DependsOn),
				Input:            step.Input,
				Output:           step.Output,
				SkipWhen:         step.SkipWhen,
				DependencyPolicy: step.DependencyPolicy,
				Retry:            step.Retry,
			}
			flowSteps[i] = flowStep
		}
	}

	return settings, flowSteps
}

// updateWorkerFromRequest updates a domain worker entity from API request
func updateWorkerFromRequest(w *worker.Worker, req *types.UpdateWorkerRequest) {
	if req.Name != nil {
		w.Name = *req.Name
	}
	if req.Prompt != nil {
		w.Prompt = *req.Prompt
	}
	if req.Enabled != nil {
		w.Enabled = *req.Enabled
	}
}

// updateWorkerSettingsFromRequest updates worker settings from API request
func updateWorkerSettingsFromRequest(req *types.WorkerSettingsInput, workerID uuid.UUID) *worker.WorkerSettings {
	if req == nil {
		return nil
	}

	// Create settings with defaults
	settings := &worker.WorkerSettings{
		ID:            uuid.New(),
		WorkerID:      workerID,
		SleepDuration: req.SleepDuration,
		TeamName:      "autoteam", // default
		InstallDeps:   false,      // default
		MaxAttempts:   3,          // default
		Debug:         false,      // default
		MCPServers:    make(worker.MCPServersMap),
		Service:       make(map[string]interface{}),
		Meta:          make(map[string]interface{}),
	}

	// Set optional fields
	if req.TeamName != nil {
		settings.TeamName = *req.TeamName
	}
	if req.InstallDeps != nil {
		settings.InstallDeps = *req.InstallDeps
	}
	if req.MaxAttempts != nil {
		settings.MaxAttempts = *req.MaxAttempts
	}
	if req.CommonPrompt != nil {
		settings.CommonPrompt = *req.CommonPrompt
	}
	if req.Debug != nil {
		settings.Debug = *req.Debug
	}

	// Convert MCP servers
	if req.MCPServers != nil {
		for name, server := range req.MCPServers {
			settings.MCPServers[name] = worker.MCPServer{
				Command: server.Command,
				Args:    server.Args,
				Env:     server.Env,
			}
		}
	}

	// Convert service config
	if req.Service != nil {
		settings.Service = req.Service
	}

	// Convert meta
	if req.Meta != nil {
		settings.Meta = req.Meta
	}

	return settings
}
