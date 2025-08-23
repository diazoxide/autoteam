package server

import (
	"bufio"
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

	"github.com/labstack/echo/v4"
)

// Handlers contains the HTTP handlers for the worker API
type Handlers struct {
	worker      WorkerInterface
	workingDir  string
	startTime   time.Time
	taskService *task.Service
}

// NewHandlers creates a new handlers instance
func NewHandlers(wk WorkerInterface, workingDir string, startTime time.Time) *Handlers {
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
		Name:    h.worker.Name(),
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

	// Get agent info
	agentInfo := WorkerInfo{
		Name:    h.worker.Name(),
		Type:    h.worker.Type(),
		Version: "unknown",
	}

	// Get agent version
	if version, err := h.worker.Version(ctx); err == nil {
		agentInfo.Version = version
	}

	// Check availability
	available := h.worker.IsAvailable(ctx)
	agentInfo.Available = &available

	// Calculate uptime
	uptime := time.Since(h.startTime).String()

	// For now, status is always idle - in real implementation this would track actual agent state
	response := StatusResponse{
		Status:    AgentStatusIdle,
		Mode:      AgentModeBoth,
		Timestamp: time.Now(),
		Agent:     agentInfo,
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

// GetCollectorLogs handles GET /logs/collector
func (h *Handlers) GetCollectorLogs(c echo.Context) error {
	limitStr := c.QueryParam("limit")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	logs, err := h.getLogFiles(LogRoleCollector, limit)
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

// GetExecutorLogs handles GET /logs/executor
func (h *Handlers) GetExecutorLogs(c echo.Context) error {
	limitStr := c.QueryParam("limit")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	logs, err := h.getLogFiles(LogRoleExecutor, limit)
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

// GetTasks handles GET /tasks
func (h *Handlers) GetTasks(c echo.Context) error {
	// For now, return empty tasks list - in real implementation this would query actual tasks
	response := TasksResponse{
		Tasks:     []TaskSummary{},
		Total:     0,
		Timestamp: time.Now(),
	}

	return c.JSON(http.StatusOK, response)
}

// GetTask handles GET /tasks/{id}
func (h *Handlers) GetTask(c echo.Context) error {
	id := c.Param("id")

	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "task id is required")
	}

	// For now, return not found - in real implementation this would query actual task
	return echo.NewHTTPError(http.StatusNotFound, "task not found")
}

// GetMetrics handles GET /metrics
func (h *Handlers) GetMetrics(c echo.Context) error {
	uptime := time.Since(h.startTime).String()

	metrics := WorkerMetrics{
		Uptime: &uptime,
		// In real implementation, these would be tracked
		TasksProcessed:   intPtr(0),
		TasksSuccess:     intPtr(0),
		TasksFailed:      intPtr(0),
		AvgExecutionTime: stringPtr("0s"),
		LastActivity:     timePtr(h.startTime),
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

	// Get agent version
	version := "unknown"
	if v, err := h.worker.Version(ctx); err == nil {
		version = v
	}

	config := WorkerConfig{
		Name:    stringPtr(h.worker.Name()),
		Type:    stringPtr(h.worker.Type()),
		Enabled: boolPtr(true),
		Version: stringPtr(version),
	}

	response := ConfigResponse{
		Config:    config,
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

// Utility functions for pointer creation
func stringPtr(s string) *string     { return &s }
func intPtr(i int) *int              { return &i }
func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }
