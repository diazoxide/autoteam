package controlplane

import (
	"fmt"
	"io"
	"net/http"
	"time"

	controlplaneapi "autoteam/api/control-plane"
	workerapi "autoteam/api/worker"
	"autoteam/internal/logger"
	"autoteam/internal/types"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Handlers implements the control plane API handlers
type Handlers struct {
	registry *WorkerRegistry
}

// NewHandlers creates new control plane handlers
func NewHandlers(registry *WorkerRegistry) *Handlers {
	return &Handlers{
		registry: registry,
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
		details := types.WorkerDetails{
			ID:         id,
			URL:        worker.URL,
			Status:     worker.Status,
			LastCheck:  worker.LastCheck,
			WorkerInfo: worker.WorkerInfo,
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
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		return worker.Client.GetHealth(ctx.Request().Context())
	})
}

func (h *Handlers) GetWorkerStatus(ctx echo.Context, workerID string) error {
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		return worker.Client.GetStatus(ctx.Request().Context())
	})
}

func (h *Handlers) GetWorkerConfig(ctx echo.Context, workerID string) error {
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		return worker.Client.GetConfig(ctx.Request().Context())
	})
}

func (h *Handlers) GetWorkerLogs(ctx echo.Context, workerID string, params controlplaneapi.GetWorkerLogsParams) error {
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		// Convert control plane params to worker API params
		var workerParams workerapi.GetLogsParams
		if params.Role != nil {
			role := workerapi.GetLogsParamsRole(*params.Role)
			workerParams.Role = &role
		}
		if params.Limit != nil {
			workerParams.Limit = params.Limit
		}

		return worker.Client.GetLogs(ctx.Request().Context(), &workerParams)
	})
}

func (h *Handlers) GetWorkerLogFile(ctx echo.Context, workerID string, filename string, params controlplaneapi.GetWorkerLogFileParams) error {
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		// Convert control plane params to worker API params
		var workerParams workerapi.GetLogFileParams
		if params.Tail != nil {
			workerParams.Tail = params.Tail
		}

		return worker.Client.GetLogFile(ctx.Request().Context(), filename, &workerParams)
	})
}

func (h *Handlers) GetWorkerFlow(ctx echo.Context, workerID string) error {
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		return worker.Client.GetFlow(ctx.Request().Context())
	})
}

func (h *Handlers) GetWorkerFlowSteps(ctx echo.Context, workerID string) error {
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		return worker.Client.GetFlowSteps(ctx.Request().Context())
	})
}

func (h *Handlers) GetWorkerMetrics(ctx echo.Context, workerID string) error {
	return h.proxyRequest(ctx, workerID, func(worker *RegisteredWorker) (*http.Response, error) {
		return worker.Client.GetMetrics(ctx.Request().Context())
	})
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

// proxyRequest is a generic proxy function for worker API requests
func (h *Handlers) proxyRequest(ctx echo.Context, workerID string, requestFunc func(*RegisteredWorker) (*http.Response, error)) error {
	log := logger.FromContext(ctx.Request().Context())

	// Get worker from registry
	worker, err := h.registry.GetWorker(workerID)
	if err != nil {
		log.Warn("Worker not found", zap.String("worker_id", workerID))
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Worker not found: %s", workerID))
	}

	// Make request to worker
	resp, err := requestFunc(worker)
	if err != nil {
		log.Error("Failed to proxy request to worker",
			zap.String("worker_id", workerID),
			zap.String("worker_url", worker.URL),
			zap.Error(err))

		// Update worker status as unreachable
		h.registry.updateWorkerStatus(workerID, types.WorkerStatusUnreachable, nil)

		return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("Worker unreachable: %s", workerID))
	}
	defer resp.Body.Close()

	// Update worker status as reachable
	h.registry.updateWorkerStatus(workerID, types.WorkerStatusReachable, nil)

	// Copy status code
	ctx.Response().Status = resp.StatusCode

	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			ctx.Response().Header().Add(key, value)
		}
	}

	// Copy body
	_, err = io.Copy(ctx.Response().Writer, resp.Body)
	if err != nil {
		log.Error("Failed to copy response body",
			zap.String("worker_id", workerID),
			zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to proxy response")
	}

	log.Debug("Request proxied successfully",
		zap.String("worker_id", workerID),
		zap.Int("status_code", resp.StatusCode))

	return nil
}
