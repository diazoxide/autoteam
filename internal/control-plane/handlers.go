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

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Handlers implements the control plane API handlers
type Handlers struct {
	registry *WorkerRegistry
	runtime  runtime.Runtime
	config   *config.Config
}

// NewHandlers creates new control plane handlers
func NewHandlers(registry *WorkerRegistry, rt runtime.Runtime, cfg *config.Config) *Handlers {
	return &Handlers{
		registry: registry,
		runtime:  rt,
		config:   cfg,
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

// GetWorkers returns list of all workers
func (h *Handlers) GetWorkers(ctx echo.Context) error {
	workers := h.registry.GetAllWorkers()

	var workerDetails []types.WorkerDetails
	for id, worker := range workers {
		// Create basic worker info from database if available
		var workerInfo *types.WorkerInfo
		if worker.DBWorker != nil {
			workerInfo = &types.WorkerInfo{
				Name: worker.DBWorker.Name, // Display name for UI
				Type: "database",           // Could be enhanced to include agent type info
			}
		} else if worker.WorkerInfo != nil {
			workerInfo = worker.WorkerInfo
		}

		details := types.WorkerDetails{
			ID:         id, // Use UUID as ID for API routing
			URL:        worker.URL,
			Status:     worker.Status,
			LastCheck:  worker.LastCheck,
			WorkerInfo: workerInfo,
		}
		workerDetails = append(workerDetails, details)
	}

	response := types.WorkersResponse{
		Workers:   workerDetails,
		Total:     len(workerDetails),
		Timestamp: time.Now(),
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetWorker returns details about a specific worker
func (h *Handlers) GetWorker(ctx echo.Context, workerID string) error {
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

// Proxy handlers - forward requests to worker API
func (h *Handlers) GetWorkerHealth(ctx echo.Context, workerID string) error {
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

func (h *Handlers) GetWorkerStatus(ctx echo.Context, workerID string) error {
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

func (h *Handlers) GetWorkerConfig(ctx echo.Context, workerID string) error {
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

func (h *Handlers) GetWorkerLogs(ctx echo.Context, workerID string, params controlplaneapi.GetWorkerLogsParams) error {
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

func (h *Handlers) GetWorkerLogFile(ctx echo.Context, workerID string, filename string, params controlplaneapi.GetWorkerLogFileParams) error {
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

func (h *Handlers) GetWorkerFlow(ctx echo.Context, workerID string) error {
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

func (h *Handlers) GetWorkerFlowSteps(ctx echo.Context, workerID string) error {
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

func (h *Handlers) GetWorkerMetrics(ctx echo.Context, workerID string) error {
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
