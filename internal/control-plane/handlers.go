package controlplane

import (
	"fmt"
	"net/http"
	"time"

	controlplaneapi "autoteam/api/control-plane"
	"autoteam/internal/config"
	workerv1 "autoteam/internal/grpc/gen/proto/autoteam/worker/v1"
	"autoteam/internal/logger"
	"autoteam/internal/runtime"
	"autoteam/internal/types"
	"autoteam/internal/worker"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Handlers implements the control plane API handlers
type Handlers struct {
	registry   *WorkerRegistry
	runtime    runtime.Runtime
	config     *config.Config
	workerRepo worker.Repository
}

// NewHandlers creates new control plane handlers
func NewHandlers(registry *WorkerRegistry, rt runtime.Runtime, cfg *config.Config, workerRepo worker.Repository) *Handlers {
	return &Handlers{
		registry:   registry,
		runtime:    rt,
		config:     cfg,
		workerRepo: workerRepo,
	}
}

// GetHealth implements control plane health check
func (h *Handlers) GetHealth(ctx echo.Context) error {
	log := logger.FromContext(ctx.Request().Context())

	// Perform health checks on all workers
	h.registry.PerformHealthChecks(ctx.Request().Context())

	// Determine overall health status
	workers := h.registry.GetAllWorkers()
	workersHealth := make(map[string]string)
	healthyCount := 0

	for id, worker := range workers {
		workersHealth[id] = worker.Status
		if worker.Status == types.WorkerStatusReachable {
			healthyCount++
		}
	}

	// Determine overall status
	var status string
	var message *string
	totalWorkers := len(workers)

	if healthyCount == totalWorkers {
		status = types.ControlPlaneStatusHealthy
	} else if healthyCount > 0 {
		status = types.ControlPlaneStatusDegraded
		msg := fmt.Sprintf("%d of %d workers are healthy", healthyCount, totalWorkers)
		message = &msg
	} else {
		status = types.ControlPlaneStatusUnhealthy
		msg := "No workers are reachable"
		message = &msg
	}

	response := types.ControlPlaneHealthResponse{
		Status:        status,
		Timestamp:     time.Now(),
		WorkersHealth: workersHealth,
		Message:       message,
	}

	log.Debug("Control plane health check completed",
		zap.String("status", status),
		zap.Int("healthy_workers", healthyCount),
		zap.Int("total_workers", totalWorkers))

	return ctx.JSON(http.StatusOK, response)
}

// GetWorkers returns list of all workers (CRUD data from database)
func (h *Handlers) GetWorkers(ctx echo.Context) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get all workers from database (CRUD data)
	dbWorkers, err := h.workerRepo.List(ctx.Request().Context())
	if err != nil {
		log.Error("Failed to get workers from database", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get workers")
	}

	// Convert to API response format
	var workerResponses []types.WorkerResponse
	for _, dbWorker := range dbWorkers {
		// Get settings for this worker
		settings, err := h.workerRepo.GetSettingsByWorkerID(ctx.Request().Context(), dbWorker.ID)
		if err != nil {
			log.Warn("Failed to get settings for worker", zap.String("worker_id", dbWorker.ID.String()), zap.Error(err))
			settings = nil // Let converter handle nil settings
		}

		// Get flow steps for this worker
		flowSteps, err := h.workerRepo.GetFlowStepsByWorkerID(ctx.Request().Context(), dbWorker.ID)
		if err != nil {
			log.Warn("Failed to get flow steps for worker", zap.String("worker_id", dbWorker.ID.String()), zap.Error(err))
			flowSteps = []worker.FlowStep{} // Default to empty flow steps
		}

		// Set flow steps for conversion
		dbWorker.FlowSteps = flowSteps
		dbWorker.Settings = settings

		// Use converter utility
		if response := h.convertWorkerToResponse(dbWorker); response != nil {
			workerResponses = append(workerResponses, *response)
		}
	}

	response := types.WorkersResponse{
		Workers:   workerResponses,
		Total:     len(workerResponses),
		Timestamp: time.Now(),
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetWorker returns details about a specific worker
func (h *Handlers) GetWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Validate and parse worker ID
	id, err := validateWorkerID(workerID)
	if err != nil {
		return err
	}

	// Get worker from database (CRUD data)
	dbWorker, err := h.workerRepo.GetByID(ctx.Request().Context(), id)
	if err != nil {
		log.Error("Failed to get worker from database", zap.Error(err))
		if err.Error() == fmt.Sprintf("worker not found: %s", id) {
			return echo.NewHTTPError(http.StatusNotFound, "Worker not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get worker")
	}

	// Get settings from database
	settings, err := h.workerRepo.GetSettingsByWorkerID(ctx.Request().Context(), id)
	if err != nil {
		log.Warn("Failed to get worker settings", zap.Error(err))
		settings = nil // Let converter handle nil settings
	}

	// Get flow steps
	flowSteps, err := h.workerRepo.GetFlowStepsByWorkerID(ctx.Request().Context(), id)
	if err != nil {
		log.Warn("Failed to get flow steps", zap.Error(err))
		flowSteps = []worker.FlowStep{} // Default to empty flow steps
	}

	// Set flow steps and settings for conversion
	dbWorker.FlowSteps = flowSteps
	dbWorker.Settings = settings

	// Use converter utility
	response := h.convertWorkerToResponse(dbWorker)
	if response == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert worker response")
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetWorkerRuntime returns runtime details about a specific worker from the registry
func (h *Handlers) GetWorkerRuntime(ctx echo.Context, workerID string) error {
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	details := types.WorkerDetails{
		ID:         workerID,
		URL:        worker.URL,
		Status:     worker.Status,
		LastCheck:  worker.LastCheck,
		WorkerInfo: worker.WorkerInfo,
	}

	response := types.WorkerDetailsResponse{
		Worker:    details,
		Timestamp: time.Now(),
	}

	return ctx.JSON(http.StatusOK, response)
}

// Runtime handlers - forward requests to worker API (require running containers)
func (h *Handlers) GetWorkerRuntimeHealth(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Make gRPC call
	resp, err := worker.Client.GetHealth(grpcCtx, &emptypb.Empty{})
	if err != nil {
		log.Error("Failed to get worker health",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetWorkerRuntimeStatus(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Make gRPC call
	resp, err := worker.Client.GetStatus(grpcCtx, &emptypb.Empty{})
	if err != nil {
		log.Error("Failed to get worker status",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetWorkerRuntimeConfig(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Make gRPC call
	resp, err := worker.Client.GetConfig(grpcCtx, &emptypb.Empty{})
	if err != nil {
		log.Error("Failed to get worker config",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetWorkerRuntimeLogs(ctx echo.Context, workerID string, params controlplaneapi.GetWorkerRuntimeLogsParams) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Convert control plane params to gRPC request
	req := &workerv1.ListLogsRequest{}
	if params.Role != nil {
		roleStr := string(*params.Role)
		req.Role = &roleStr
	}
	if params.Limit != nil {
		limitInt32 := int32(*params.Limit)
		req.Limit = &limitInt32
	}

	// Make gRPC call
	resp, err := worker.Client.ListLogs(grpcCtx, req)
	if err != nil {
		log.Error("Failed to get worker logs",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetWorkerRuntimeLogFile(ctx echo.Context, workerID string, filename string, params controlplaneapi.GetWorkerRuntimeLogFileParams) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Convert control plane params to gRPC request
	req := &workerv1.GetLogFileRequest{
		Filename: filename,
	}
	if params.Tail != nil {
		tailInt32 := int32(*params.Tail)
		req.Tail = &tailInt32
	}

	// Make gRPC call
	resp, err := worker.Client.GetLogFile(grpcCtx, req)
	if err != nil {
		log.Error("Failed to get worker log file",
			zap.String("worker_id", workerID),
			zap.String("filename", filename),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetWorkerRuntimeFlow(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Make gRPC call
	resp, err := worker.Client.GetFlow(grpcCtx, &emptypb.Empty{})
	if err != nil {
		log.Error("Failed to get worker flow",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetWorkerRuntimeFlowSteps(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Make gRPC call
	resp, err := worker.Client.GetFlowSteps(grpcCtx, &emptypb.Empty{})
	if err != nil {
		log.Error("Failed to get worker flow steps",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) GetWorkerRuntimeMetrics(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if worker has gRPC client connection
	if worker.Client == nil {
		log.Warn("Worker has no gRPC connection", zap.String("worker_id", workerID), zap.String("status", worker.Status))
		return echo.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("Worker not deployed: %s", workerID))
	}

	// Create context with authentication
	grpcCtx := h.registry.createContext(ctx.Request().Context(), worker.APIKey)

	// Make gRPC call
	resp, err := worker.Client.GetMetrics(grpcCtx, &emptypb.Empty{})
	if err != nil {
		log.Error("Failed to get worker metrics",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)
		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Convert gRPC response to JSON
	return ctx.JSON(http.StatusOK, resp)
}

// GetOpenAPISpec returns the control plane OpenAPI specification
func (h *Handlers) GetOpenAPISpec(ctx echo.Context) error {
	spec, err := controlplaneapi.GetSwagger()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get OpenAPI spec")
	}

	yamlBytes, err := spec.MarshalJSON()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal OpenAPI spec")
	}

	return ctx.Blob(http.StatusOK, "application/x-yaml", yamlBytes)
}

// GetSwaggerUI serves the Swagger UI documentation
func (h *Handlers) GetSwaggerUI(ctx echo.Context) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>AutoTeam Control Plane API</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: '/openapi.yaml',
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.presets.standalone
            ]
        });
    </script>
</body>
</html>`

	return ctx.HTML(http.StatusOK, html)
}

// Worker action handlers

// DeployWorker deploys a specific worker
func (h *Handlers) DeployWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Check if we have database worker info
	if worker.DBWorker == nil {
		log.Warn("Worker has no database info for deployment", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Worker %s cannot be deployed: no configuration found", workerID))
	}

	// Start with global settings as base
	effectiveSettings := h.config.Settings
	if worker.DBWorker.Settings != nil {
		// Merge database worker settings with global settings (database settings override global ones)
		// Keep global settings as base and only override specific fields from database worker settings
		if worker.DBWorker.Settings.TeamName != "" {
			effectiveSettings.TeamName = worker.DBWorker.Settings.TeamName
		}
		if worker.DBWorker.Settings.SleepDuration != 0 {
			effectiveSettings.SleepDuration = worker.DBWorker.Settings.SleepDuration
		}
		// Always use the database worker's debug setting
		effectiveSettings.Debug = worker.DBWorker.Settings.Debug
	}

	// Use flow configuration from database worker (FlowSteps), not global settings
	if len(worker.DBWorker.FlowSteps) > 0 {
		// Assign database flow steps directly (both FlowSteps and Flow are []FlowStep)
		effectiveSettings.Flow = worker.DBWorker.FlowSteps
		log.Debug("Using flow configuration from database worker",
			zap.String("worker_id", workerID),
			zap.Int("database_flow_steps", len(worker.DBWorker.FlowSteps)))
	} else {
		log.Warn("No flow steps found in database worker, using global flow as fallback",
			zap.String("worker_id", workerID),
			zap.Int("global_flow_steps", len(effectiveSettings.Flow)))
	}

	// Debug: Log the effective settings flow configuration
	log.Info("Deploying worker with effective settings",
		zap.String("worker_id", workerID),
		zap.Int("flow_steps_count", len(effectiveSettings.Flow)),
		zap.Bool("debug", effectiveSettings.Debug))

	// Deploy worker using runtime
	if err := h.runtime.DeployWorker(ctx.Request().Context(), *worker.DBWorker, effectiveSettings, h.config); err != nil {
		log.Error("Failed to deploy worker", zap.String("worker_id", workerID), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to deploy worker: %s", err.Error()))
	}

	log.Info("Worker deployed successfully", zap.String("worker_id", workerID))
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Worker deployed successfully",
		"worker_id": workerID,
		"timestamp": time.Now(),
	})
}

// StopWorker stops a specific worker
func (h *Handlers) StopWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Stop worker using runtime
	if err := h.runtime.StopWorker(ctx.Request().Context(), worker.Name, h.config); err != nil {
		log.Error("Failed to stop worker", zap.String("worker_id", workerID), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to stop worker: %s", err.Error()))
	}

	log.Info("Worker stopped successfully", zap.String("worker_id", workerID))
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Worker stopped successfully",
		"worker_id": workerID,
		"timestamp": time.Now(),
	})
}

// RestartWorker restarts a specific worker
func (h *Handlers) RestartWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Restart worker using runtime
	if err := h.runtime.RestartWorker(ctx.Request().Context(), worker.Name, h.config); err != nil {
		log.Error("Failed to restart worker", zap.String("worker_id", workerID), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to restart worker: %s", err.Error()))
	}

	log.Info("Worker restarted successfully", zap.String("worker_id", workerID))
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Worker restarted successfully",
		"worker_id": workerID,
		"timestamp": time.Now(),
	})
}

// PauseWorker pauses a specific worker
func (h *Handlers) PauseWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Pause worker using runtime
	if err := h.runtime.PauseWorker(ctx.Request().Context(), worker.Name, h.config); err != nil {
		log.Error("Failed to pause worker", zap.String("worker_id", workerID), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to pause worker: %s", err.Error()))
	}

	log.Info("Worker paused successfully", zap.String("worker_id", workerID))
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Worker paused successfully",
		"worker_id": workerID,
		"timestamp": time.Now(),
	})
}

// UnpauseWorker unpauses a specific worker
func (h *Handlers) UnpauseWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Unpause worker using runtime
	if err := h.runtime.UnpauseWorker(ctx.Request().Context(), worker.Name, h.config); err != nil {
		log.Error("Failed to unpause worker", zap.String("worker_id", workerID), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to unpause worker: %s", err.Error()))
	}

	log.Info("Worker unpaused successfully", zap.String("worker_id", workerID))
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Worker unpaused successfully",
		"worker_id": workerID,
		"timestamp": time.Now(),
	})
}

// CRUD handlers for workers

// CreateWorker creates a new worker
func (h *Handlers) CreateWorker(ctx echo.Context) error {
	log := logger.FromContext(ctx.Request().Context())

	var req types.CreateWorkerRequest
	if err := ctx.Bind(&req); err != nil {
		log.Warn("Invalid create worker request", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate request
	if err := validateCreateWorkerRequest(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Create worker entity
	w := createWorkerFromRequest(&req)

	// Check if worker name already exists
	exists, err := h.workerRepo.ExistsByName(ctx.Request().Context(), req.Name)
	if err != nil {
		log.Error("Failed to check worker name existence", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check worker name")
	}
	if exists {
		return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("worker with name '%s' already exists", req.Name))
	}

	// Create settings and flow steps if provided
	settings, flowSteps := createWorkerSettingsFromRequest(&req, w.ID)

	// Create worker in database
	err = h.workerRepo.Create(ctx.Request().Context(), w)
	if err != nil {
		log.Error("Failed to create worker", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create worker")
	}

	// Create settings if provided
	if settings != nil {
		err = h.workerRepo.CreateOrUpdateSettings(ctx.Request().Context(), settings)
		if err != nil {
			log.Error("Failed to create worker settings", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create worker settings")
		}
	}

	// Create flow steps if provided
	if len(flowSteps) > 0 {
		err = h.workerRepo.CreateFlowSteps(ctx.Request().Context(), flowSteps)
		if err != nil {
			log.Error("Failed to create flow steps", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create flow steps")
		}
	}

	// Get the created worker with settings
	createdWorker, err := h.workerRepo.GetByID(ctx.Request().Context(), w.ID)
	if err != nil {
		log.Error("Failed to retrieve created worker", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve created worker")
	}

	// Convert to response format
	response := h.convertWorkerToResponse(createdWorker)

	// Register worker with registry for deployment
	if h.registry != nil {
		if refreshErr := h.registry.RefreshFromDatabase(ctx.Request().Context()); refreshErr != nil {
			log.Warn("Failed to refresh registry from database", zap.Error(refreshErr))
		}
	}

	log.Info("Worker created successfully", zap.String("worker_id", createdWorker.ID.String()), zap.String("name", createdWorker.Name))
	return ctx.JSON(http.StatusCreated, response)
}

// UpdateWorker updates an existing worker
func (h *Handlers) UpdateWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Validate worker ID
	id, err := validateWorkerID(workerID)
	if err != nil {
		return err
	}

	var req types.UpdateWorkerRequest
	if bindErr := ctx.Bind(&req); bindErr != nil {
		log.Warn("Invalid update worker request", zap.Error(bindErr))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate request
	if validationErr := validateUpdateWorkerRequest(&req); validationErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, validationErr.Error())
	}

	// Get existing worker
	existingWorker, err := h.workerRepo.GetByID(ctx.Request().Context(), id)
	if err != nil {
		log.Error("Failed to get existing worker", zap.Error(err))
		if err.Error() == fmt.Sprintf("worker not found: %s", id) {
			return echo.NewHTTPError(http.StatusNotFound, "Worker not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get worker")
	}

	// Check if new name conflicts with existing workers
	if req.Name != nil && *req.Name != existingWorker.Name {
		exists, checkErr := h.workerRepo.ExistsByName(ctx.Request().Context(), *req.Name)
		if checkErr != nil {
			log.Error("Failed to check worker name existence", zap.Error(checkErr))
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check worker name")
		}
		if exists {
			return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("worker with name '%s' already exists", *req.Name))
		}
	}

	// Update worker fields
	updateWorkerFromRequest(existingWorker, &req)

	// Update worker in database
	err = h.workerRepo.Update(ctx.Request().Context(), existingWorker)
	if err != nil {
		log.Error("Failed to update worker", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update worker")
	}

	// Update settings if provided
	if req.Settings != nil {
		settings := updateWorkerSettingsFromRequest(req.Settings, id)
		if settings != nil {
			// Update settings using repository
			err = h.workerRepo.CreateOrUpdateSettings(ctx.Request().Context(), settings)
			if err != nil {
				log.Error("Failed to update worker settings", zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update worker settings")
			}
		}
	}

	// Get the updated worker with settings
	updatedWorker, err := h.workerRepo.GetByID(ctx.Request().Context(), id)
	if err != nil {
		log.Error("Failed to retrieve updated worker", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve updated worker")
	}

	// Convert to response format
	response := h.convertWorkerToResponse(updatedWorker)

	// Refresh registry and potentially redeploy if worker is running
	if h.registry != nil {
		if refreshErr := h.registry.RefreshFromDatabase(ctx.Request().Context()); refreshErr != nil {
			log.Warn("Failed to refresh registry from database", zap.Error(refreshErr))
		}
	}

	log.Info("Worker updated successfully", zap.String("worker_id", id.String()))
	return ctx.JSON(http.StatusOK, response)
}

// DeleteWorker deletes a worker
func (h *Handlers) DeleteWorker(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Validate worker ID
	id, err := validateWorkerID(workerID)
	if err != nil {
		return err
	}

	// Check if worker exists
	existingWorker, err := h.workerRepo.GetByID(ctx.Request().Context(), id)
	if err != nil {
		if err.Error() == fmt.Sprintf("worker not found: %s", id) {
			return echo.NewHTTPError(http.StatusNotFound, "Worker not found")
		}
		log.Error("Failed to check worker existence", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check worker existence")
	}

	// Stop worker if it's running
	workerName := existingWorker.Name
	if err := h.runtime.StopWorker(ctx.Request().Context(), workerName, h.config); err != nil {
		log.Warn("Failed to stop worker before deletion", zap.Error(err), zap.String("worker_name", workerName))
		// Continue with deletion even if stop fails
	}

	// Delete worker from database
	if err := h.workerRepo.Delete(ctx.Request().Context(), id); err != nil {
		log.Error("Failed to delete worker", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete worker")
	}

	// Remove from registry
	if h.registry != nil {
		if refreshErr := h.registry.RefreshFromDatabase(ctx.Request().Context()); refreshErr != nil {
			log.Warn("Failed to refresh registry from database", zap.Error(refreshErr))
		}
	}

	log.Info("Worker deleted successfully", zap.String("worker_id", id.String()), zap.String("name", workerName))
	return ctx.NoContent(http.StatusNoContent)
}

// GetWorkerSettings returns detailed worker settings
func (h *Handlers) GetWorkerSettings(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Parse worker ID
	id, err := uuid.Parse(workerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid worker ID")
	}

	// Get worker with settings
	w, err := h.workerRepo.GetByID(ctx.Request().Context(), id)
	if err != nil {
		if err.Error() == fmt.Sprintf("worker not found: %s", id) {
			return echo.NewHTTPError(http.StatusNotFound, "Worker not found")
		}
		log.Error("Failed to get worker", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get worker")
	}

	if w.Settings == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Worker settings not found")
	}

	// Convert to response format
	settingsOutput := convertWorkerSettingsToOutput(w.Settings, w.FlowSteps)

	response := types.WorkerSettingsResponse{
		Settings:  *settingsOutput,
		Timestamp: time.Now(),
	}

	return ctx.JSON(http.StatusOK, response)
}

// UpdateWorkerSettings updates worker settings
func (h *Handlers) UpdateWorkerSettings(ctx echo.Context, workerID string) error {
	log := logger.FromContext(ctx.Request().Context())

	// Validate and parse worker ID
	id, err := validateWorkerID(workerID)
	if err != nil {
		return err
	}

	var req controlplaneapi.UpdateWorkerSettingsRequest
	if bindErr := ctx.Bind(&req); bindErr != nil {
		log.Warn("Invalid update worker settings request", zap.Error(bindErr))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Check if worker exists
	_, err = h.workerRepo.GetByID(ctx.Request().Context(), id)
	if err != nil {
		if err.Error() == fmt.Sprintf("worker not found: %s", id) {
			return echo.NewHTTPError(http.StatusNotFound, "Worker not found")
		}
		log.Error("Failed to check worker existence", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check worker existence")
	}

	// Create settings object with default values
	settings := &worker.WorkerSettings{
		InstallDeps: false,
		MaxAttempts: 3,
		Debug:       false,
	}

	// Handle pointer fields
	if req.SleepDuration != nil {
		settings.SleepDuration = *req.SleepDuration
	}

	// Set default values if not provided
	if req.TeamName != nil {
		settings.TeamName = *req.TeamName
	} else {
		settings.TeamName = "autoteam"
	}

	if req.InstallDeps != nil {
		settings.InstallDeps = *req.InstallDeps
	}

	if req.CommonPrompt != nil {
		settings.CommonPrompt = *req.CommonPrompt
	}

	if req.MaxAttempts != nil {
		settings.MaxAttempts = *req.MaxAttempts
	}

	if req.Debug != nil {
		settings.Debug = *req.Debug
	}

	if req.Service != nil {
		settings.Service = worker.JSONMap(*req.Service)
	}

	if req.McpServers != nil {
		mcpServers := make(worker.MCPServersMap)
		for k, v := range *req.McpServers {
			mcpServers[k] = worker.MCPServer{
				Command: v.Command,
			}
			// Handle optional pointer fields in MCP server
			if v.Args != nil {
				mcpServers[k] = worker.MCPServer{
					Command: v.Command,
					Args:    *v.Args,
				}
			}
			if v.Env != nil {
				server := mcpServers[k]
				server.Env = *v.Env
				mcpServers[k] = server
			}
		}
		settings.MCPServers = mcpServers
	}

	if req.Meta != nil {
		settings.Meta = worker.JSONMap(*req.Meta)
	}

	// Set worker ID for settings
	settings.WorkerID = id

	// Update settings using repository
	err = h.workerRepo.CreateOrUpdateSettings(ctx.Request().Context(), settings)
	if err != nil {
		log.Error("Failed to update worker settings", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update worker settings")
	}

	// Get updated settings
	updatedSettings, err := h.workerRepo.GetSettingsByWorkerID(ctx.Request().Context(), id)
	if err != nil {
		log.Error("Failed to get updated settings", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get updated settings")
	}

	// Handle flow steps update if provided
	var updatedFlowSteps []worker.FlowStep
	if req.Flow != nil {
		// Convert from API types to database types
		flowSteps := make([]worker.FlowStep, len(*req.Flow))
		for i, step := range *req.Flow {
			flowSteps[i] = worker.FlowStep{
				WorkerID: id,
				Name:     step.Name,
				Type:     step.Type,
				Order:    i, // Set order based on array position
				Env:      make(map[string]string),
			}

			// Handle optional pointer fields
			if step.Args != nil {
				flowSteps[i].Args = *step.Args
			}
			if step.DependsOn != nil {
				flowSteps[i].DependsOn = *step.DependsOn
			}
			if step.Input != nil {
				flowSteps[i].Input = *step.Input
			}
			if step.Output != nil {
				flowSteps[i].Output = *step.Output
			}
			if step.SkipWhen != nil {
				flowSteps[i].SkipWhen = *step.SkipWhen
			}
			if step.DependencyPolicy != nil {
				flowSteps[i].DependencyPolicy = string(*step.DependencyPolicy)
			}

			// Copy env from API request (already map[string]string)
			if step.Env != nil {
				for k, v := range *step.Env {
					flowSteps[i].Env[k] = v
				}
			}

			// Handle retry configuration
			if step.Retry != nil {
				retryConfig := &worker.RetryConfig{}
				if step.Retry.MaxAttempts != nil {
					retryConfig.MaxAttempts = *step.Retry.MaxAttempts
				}
				if step.Retry.Delay != nil {
					retryConfig.Delay = *step.Retry.Delay
				}
				if step.Retry.Backoff != nil {
					retryConfig.Backoff = string(*step.Retry.Backoff)
				}
				if step.Retry.MaxDelay != nil {
					retryConfig.MaxDelay = *step.Retry.MaxDelay
				}
				flowSteps[i].Retry = retryConfig
			}
		}

		// Update flow steps using repository
		err = h.workerRepo.UpdateFlowSteps(ctx.Request().Context(), id, flowSteps)
		if err != nil {
			log.Error("Failed to update flow steps", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update flow steps")
		}

		updatedFlowSteps = flowSteps
		log.Info("Flow steps updated successfully",
			zap.String("worker_id", id.String()),
			zap.Int("steps_count", len(flowSteps)))
	} else {
		// Get existing flow steps if no update provided
		existingFlowSteps, err := h.workerRepo.GetFlowStepsByWorkerID(ctx.Request().Context(), id)
		if err != nil {
			log.Error("Failed to get existing flow steps", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get existing flow steps")
		}
		updatedFlowSteps = existingFlowSteps
	}

	// Convert to response format with updated flow steps
	settingsOutput := convertWorkerSettingsToOutput(updatedSettings, updatedFlowSteps)

	response := types.WorkerSettingsResponse{
		Settings:  *settingsOutput,
		Timestamp: time.Now(),
	}

	// Refresh registry and potentially redeploy if worker is running
	if h.registry != nil {
		if refreshErr := h.registry.RefreshFromDatabase(ctx.Request().Context()); refreshErr != nil {
			log.Warn("Failed to refresh registry from database", zap.Error(refreshErr))
		}
	}

	log.Info("Worker settings updated successfully", zap.String("worker_id", id.String()))
	return ctx.JSON(http.StatusOK, response)
}
