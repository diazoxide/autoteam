package server

import (
	"bufio"
	_ "embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"autoteam/internal/task"
	"autoteam/internal/worker"

	"github.com/labstack/echo/v4"
)

//go:embed openapi.yaml
var openAPISpec string

// Handlers contains the HTTP handlers for the worker API
type Handlers struct {
	worker      *worker.WorkerImpl
	workingDir  string
	startTime   time.Time
	taskService *task.Service
}

// NewHandlers creates a new handlers instance
func NewHandlers(wk *worker.WorkerImpl, workingDir string, startTime time.Time) *Handlers {
	return &Handlers{
		worker:      wk,
		workingDir:  workingDir,
		startTime:   startTime,
		taskService: task.NewService(workingDir),
	}
}

// GetHealth handles GET /health
func (h *Handlers) GetHealth(c echo.Context) error {
	ctx := c.Request().Context()

	// Get agent info
	agentInfo := WorkerInfo{
		Name:    h.worker.Name,
		Type:    h.worker.Type(),
		Version: "unknown",
	}

	// Get agent version
	if version, err := h.worker.Version(ctx); err == nil {
		agentInfo.Version = version
	}

	// Check agent availability
	available := h.worker.IsAvailable(ctx)
	agentInfo.Available = &available

	// Perform health checks
	checks := make(map[string]HealthCheck)

	// Agent availability check
	if available {
		checks["agent_available"] = HealthCheck{
			Status:  HealthCheckPass,
			Message: "Agent is available and ready",
		}
	} else {
		checks["agent_available"] = HealthCheck{
			Status:  HealthCheckFail,
			Message: "Agent is not available",
		}
	}

	// Working directory check
	if _, err := os.Stat(h.workingDir); err == nil {
		checks["working_directory"] = HealthCheck{
			Status:  HealthCheckPass,
			Message: "Working directory accessible",
		}
	} else {
		checks["working_directory"] = HealthCheck{
			Status:  HealthCheckFail,
			Message: fmt.Sprintf("Working directory not accessible: %v", err),
		}
	}

	// Determine overall health status
	status := HealthStatusHealthy
	for _, check := range checks {
		if check.Status == HealthCheckFail {
			status = HealthStatusUnhealthy
			break
		}
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Agent:     agentInfo,
		Checks:    checks,
	}

	return c.JSON(http.StatusOK, response)
}

// GetStatus handles GET /status
func (h *Handlers) GetStatus(c echo.Context) error {
	ctx := c.Request().Context()

	// Get worker info
	workerInfo := WorkerInfo{
		Name:    h.worker.Name,
		Type:    h.worker.Type(),
		Version: "unknown",
	}

	// Get worker version
	if version, err := h.worker.Version(ctx); err == nil {
		workerInfo.Version = version
	}

	// Check availability
	available := h.worker.IsAvailable(ctx)
	workerInfo.Available = &available

	// Get uptime from worker
	uptime := h.worker.GetUptime().String()

	// Get actual worker status
	status := WorkerStatusIdle
	if h.worker.IsRunning() {
		status = WorkerStatusRunning
	}

	response := StatusResponse{
		Status:    status,
		Mode:      WorkerModeBoth, // Workers handle both collection and execution
		Timestamp: time.Now(),
		Agent:     workerInfo,
		Uptime:    uptime,
	}

	return c.JSON(http.StatusOK, response)
}

// GetLogs handles GET /logs
func (h *Handlers) GetLogs(c echo.Context) error {
	role := c.QueryParam("role")
	limitStr := c.QueryParam("limit")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	logs, err := h.getLogFiles(role, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	response := LogsResponse{
		Logs:      logs,
		Total:     len(logs),
		Timestamp: time.Now(),
	}

	return c.JSON(http.StatusOK, response)
}

// GetLogFile handles GET /logs/{filename}
func (h *Handlers) GetLogFile(c echo.Context) error {
	filename := c.Param("filename")
	tailStr := c.QueryParam("tail")

	if filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "filename is required")
	}

	// Security check: prevent path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid filename")
	}

	logPath := filepath.Join(h.workingDir, "logs", filename)

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return echo.NewHTTPError(http.StatusNotFound, "log file not found")
	}

	// Handle tail parameter
	if tailStr != "" {
		tailLines, err := strconv.Atoi(tailStr)
		if err != nil || tailLines <= 0 || tailLines > 10000 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid tail parameter")
		}

		content, err := h.tailFile(logPath, tailLines)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		c.Response().Header().Set("Content-Type", "text/plain; charset=utf-8")
		return c.String(http.StatusOK, content)
	}

	// Serve entire file
	return c.File(logPath)
}

// GetMetrics handles GET /metrics
func (h *Handlers) GetMetrics(c echo.Context) error {
	uptime := h.worker.GetUptime().String()

	metrics := WorkerMetrics{
		Uptime: &uptime,
		// In real implementation, these would be tracked
		TasksProcessed:   intPtr(0),
		TasksSuccess:     intPtr(0),
		TasksFailed:      intPtr(0),
		AvgExecutionTime: stringPtr("0s"),
		LastActivity:     h.worker.GetLastActivity(),
	}

	response := MetricsResponse{
		Metrics:   metrics,
		Timestamp: time.Now(),
	}

	return c.JSON(http.StatusOK, response)
}

// GetConfig handles GET /config
func (h *Handlers) GetConfig(c echo.Context) error {
	ctx := c.Request().Context()

	// Get worker version
	version := "unknown"
	if v, err := h.worker.Version(ctx); err == nil {
		version = v
	}

	// Get worker configuration details
	workerConfig := h.worker.GetConfig()
	settings := h.worker.GetSettings()

	config := WorkerConfig{
		Name:      stringPtr(h.worker.Name),
		Type:      stringPtr(h.worker.Type()),
		Enabled:   stringPtr(fmt.Sprintf("%v", workerConfig.IsEnabled())),
		Version:   stringPtr(version),
		TeamName:  stringPtr(h.worker.GetTeamName()),
		FlowSteps: intPtr(len(settings.Flow)),
	}

	response := ConfigResponse{
		Config:    config,
		Timestamp: time.Now(),
	}

	return c.JSON(http.StatusOK, response)
}

// GetFlow handles GET /flow
func (h *Handlers) GetFlow(c echo.Context) error {
	// Get worker configuration to analyze flow
	settings := h.worker.GetSettings()

	flowInfo := FlowInfo{
		TotalSteps:     len(settings.Flow),
		EnabledSteps:   len(settings.Flow), // All steps are enabled by default
		LastExecution:  h.worker.GetLastActivity(),
		ExecutionCount: intPtr(0), // TODO: Track actual execution count
		SuccessRate:    nil,       // TODO: Track success rate
	}

	response := FlowResponse{
		Flow:      flowInfo,
		Timestamp: time.Now(),
	}

	return c.JSON(http.StatusOK, response)
}

// GetFlowSteps handles GET /flow/steps
func (h *Handlers) GetFlowSteps(c echo.Context) error {
	// Get worker configuration to get flow steps
	settings := h.worker.GetSettings()

	var stepInfos []FlowStepInfo
	for _, step := range settings.Flow {
		stepInfo := FlowStepInfo{
			FlowStep: step, // Embed original FlowStep directly
			FlowStepRuntime: FlowStepRuntime{
				Enabled:        boolPtr(true), // All steps are enabled by default
				LastExecution:  nil,           // TODO: Track per-step execution times
				ExecutionCount: intPtr(0),     // TODO: Track per-step execution count
				SuccessCount:   intPtr(0),     // TODO: Track per-step success count
				LastOutput:     nil,           // TODO: Track last output
				LastError:      nil,           // TODO: Track last error
			},
		}
		stepInfos = append(stepInfos, stepInfo)
	}

	response := FlowStepsResponse{
		Steps:     stepInfos,
		Total:     len(stepInfos),
		Timestamp: time.Now(),
	}

	return c.JSON(http.StatusOK, response)
}

// getLogFiles retrieves log files based on role filter and limit
func (h *Handlers) getLogFiles(role string, limit int) ([]LogFile, error) {
	logsDir := filepath.Join(h.workingDir, "logs")

	// Check if logs directory exists
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return []LogFile{}, nil
	}

	var logFiles []LogFile

	err := filepath.WalkDir(logsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only include .log files
		if !strings.HasSuffix(d.Name(), ".log") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // Skip files we can't stat
		}

		// Determine role based on filename or directory
		fileRole := h.determineLogRole(d.Name(), path)

		// Apply role filter
		if role != "" && role != fileRole {
			return nil
		}

		logFile := LogFile{
			Filename: d.Name(),
			Size:     info.Size(),
			Modified: info.ModTime(),
		}

		if fileRole != "" {
			logFile.Role = &fileRole
		}

		logFiles = append(logFiles, logFile)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by modification time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Modified.After(logFiles[j].Modified)
	})

	// Apply limit
	if limit > 0 && len(logFiles) > limit {
		logFiles = logFiles[:limit]
	}

	return logFiles, nil
}

// determineLogRole determines the role based on log filename or path
func (h *Handlers) determineLogRole(filename, path string) string {
	// Check if path contains collector or executor subdirectory
	if strings.Contains(path, "/collector/") {
		return LogRoleCollector
	}
	if strings.Contains(path, "/executor/") {
		return LogRoleExecutor
	}

	// Check filename patterns (fallback)
	lower := strings.ToLower(filename)
	if strings.Contains(lower, "collector") {
		return LogRoleCollector
	}
	if strings.Contains(lower, "executor") {
		return LogRoleExecutor
	}

	return LogRoleBoth
}

// tailFile reads the last n lines from a file
func (h *Handlers) tailFile(filepath string, lines int) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var result []string
	scanner := bufio.NewScanner(file)

	// Read all lines
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Get last n lines
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}

	result = allLines[start:]
	return strings.Join(result, "\n"), nil
}

// GetOpenAPISpec handles GET /openapi.yaml and /openapi
func (h *Handlers) GetOpenAPISpec(c echo.Context) error {
	// Replace hardcoded server URL with actual request host
	actualURL := "http://" + c.Request().Host
	spec := strings.ReplaceAll(openAPISpec, "http://localhost:8080", actualURL)

	c.Response().Header().Set("Content-Type", "application/x-yaml")
	return c.String(http.StatusOK, spec)
}

// GetSwaggerUI handles GET /docs/ - serves basic Swagger UI
func (h *Handlers) GetSwaggerUI(c echo.Context) error {
	// Simple Swagger UI HTML that loads the OpenAPI spec
	swaggerHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>AutoTeam Worker API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js" crossorigin></script>
    <script>
        window.onload = () => {
            window.ui = SwaggerUIBundle({
                url: window.location.origin + '/openapi.yaml',
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.presets.standalone,
                ],
            });
        };
    </script>
</body>
</html>`

	c.Response().Header().Set("Content-Type", "text/html")
	return c.String(http.StatusOK, swaggerHTML)
}

// Utility functions for pointer creation
func stringPtr(s string) *string     { return &s }
func intPtr(i int) *int              { return &i }
func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }

// stringPtrIfNotEmpty returns a pointer to string if not empty, nil otherwise
func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
