package grpc

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	workerv1 "autoteam/internal/grpc/gen/proto/autoteam/worker/v1"
	"autoteam/internal/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetHealth implements the health check RPC
func (s *Server) GetHealth(ctx context.Context, req *emptypb.Empty) (*workerv1.HealthResponse, error) {
	// Get agent info
	agentInfo := &workerv1.WorkerInfo{
		Name:    s.runtime.Name,
		Type:    s.runtime.Type(),
		Version: "1.0.0", // TODO: Get from build info
	}

	// Create health checks map
	checks := make(map[string]*workerv1.HealthCheck)

	// Add basic health check
	checks["basic"] = &workerv1.HealthCheck{
		Status:  types.HealthStatusHealthy,
		Message: nil, // Optional field
	}

	// Check if agent is available
	available := true
	agentInfo.Available = &available

	response := &workerv1.HealthResponse{
		Status:    types.HealthStatusHealthy,
		Timestamp: timestamppb.Now(),
		Agent:     agentInfo,
		Checks:    checks,
	}

	return response, nil
}

// GetStatus implements the status RPC
func (s *Server) GetStatus(ctx context.Context, req *emptypb.Empty) (*workerv1.StatusResponse, error) {
	// Get agent info
	agentInfo := &workerv1.WorkerInfo{
		Name:    s.runtime.Name,
		Type:    s.runtime.Type(),
		Version: "1.0.0", // TODO: Get from build info
	}

	// Calculate actual uptime
	uptime := s.runtime.GetUptime().String()

	// Determine actual worker status
	workerStatus := types.WorkerStatusIdle
	if s.runtime.IsRunning() {
		// Check if any step is currently active
		stepStats := s.runtime.GetAllStepStats()
		for _, stats := range stepStats {
			if stats.Active {
				workerStatus = types.WorkerStatusRunning
				break
			}
		}
	}

	response := &workerv1.StatusResponse{
		Status:    workerStatus,
		Mode:      types.WorkerModeBoth,
		Timestamp: timestamppb.Now(),
		Agent:     agentInfo,
		Uptime:    &uptime,
	}

	return response, nil
}

// ListLogs implements the list logs RPC
func (s *Server) ListLogs(ctx context.Context, req *workerv1.ListLogsRequest) (*workerv1.LogsResponse, error) {
	workingDir := s.runtime.GetWorkingDir()
	logsDir := filepath.Join(workingDir, "logs")

	var logFiles []*workerv1.LogFile

	// Walk the logs directory
	err := filepath.WalkDir(logsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking, ignore errors
		}

		if d.IsDir() {
			return nil
		}

		// Only include .log files
		if !strings.HasSuffix(d.Name(), ".log") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // Continue, ignore error files
		}

		logFile := &workerv1.LogFile{
			Filename: d.Name(),
			Size:     info.Size(),
			Modified: timestamppb.New(info.ModTime()),
		}

		// Determine role from filename if requested
		if req.Role != nil && *req.Role != "both" {
			role := determineLogRole(d.Name())
			if role != "" && role != *req.Role {
				return nil // Skip files that don't match requested role
			}
			logFile.Role = &role
		}

		logFiles = append(logFiles, logFile)
		return nil
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read logs directory: %v", err)
	}

	// Sort by modification time, newest first
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Modified.AsTime().After(logFiles[j].Modified.AsTime())
	})

	// Apply limit if specified
	if req.Limit != nil && *req.Limit > 0 && len(logFiles) > int(*req.Limit) {
		logFiles = logFiles[:*req.Limit]
	}

	response := &workerv1.LogsResponse{
		Logs:      logFiles,
		Total:     int32(len(logFiles)),
		Timestamp: timestamppb.Now(),
	}

	return response, nil
}

// GetLogFile implements the get log file RPC
func (s *Server) GetLogFile(ctx context.Context, req *workerv1.GetLogFileRequest) (*workerv1.LogFileResponse, error) {
	workingDir := s.runtime.GetWorkingDir()
	logPath := filepath.Join(workingDir, "logs", req.Filename)

	// Security check: ensure the path is within logs directory
	if !strings.HasPrefix(filepath.Clean(logPath), filepath.Join(workingDir, "logs")) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid log file path")
	}

	// Read file content
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "log file not found: %s", req.Filename)
		}
		return nil, status.Errorf(codes.Internal, "failed to read log file: %v", err)
	}

	fileContent := string(content)

	// Apply tail if specified
	if req.Tail != nil && *req.Tail > 0 {
		lines := strings.Split(fileContent, "\n")
		if len(lines) > int(*req.Tail) {
			lines = lines[len(lines)-int(*req.Tail):]
		}
		fileContent = strings.Join(lines, "\n")
	}

	response := &workerv1.LogFileResponse{
		Content: fileContent,
	}

	return response, nil
}

// StreamLogs implements the stream logs RPC
func (s *Server) StreamLogs(req *workerv1.StreamLogsRequest, stream workerv1.WorkerService_StreamLogsServer) error {
	// For now, return a simple implementation
	// TODO: Implement actual log streaming
	return status.Errorf(codes.Unimplemented, "log streaming not yet implemented")
}

// GetFlow implements the get flow RPC
func (s *Server) GetFlow(ctx context.Context, req *emptypb.Empty) (*workerv1.FlowResponse, error) {
	// Get flow steps from worker settings
	flowSteps := s.runtime.GetSettings().Flow

	flowInfo := &workerv1.FlowInfo{
		TotalSteps:   int32(len(flowSteps)),
		EnabledSteps: int32(len(flowSteps)), // TODO: Count only enabled steps
	}

	// TODO: Add execution statistics
	// lastExecution := timestamppb.Now()
	// flowInfo.LastExecution = &lastExecution
	// executionCount := int32(10)
	// flowInfo.ExecutionCount = &executionCount
	// successRate := 0.85
	// flowInfo.SuccessRate = &successRate

	response := &workerv1.FlowResponse{
		Flow:      flowInfo,
		Timestamp: timestamppb.Now(),
	}

	return response, nil
}

// GetFlowSteps implements the get flow steps RPC
func (s *Server) GetFlowSteps(ctx context.Context, req *emptypb.Empty) (*workerv1.FlowStepsResponse, error) {
	// Get flow steps from worker settings
	flowSteps := s.runtime.GetSettings().Flow

	var stepInfos []*workerv1.FlowStepInfo

	for _, step := range flowSteps {
		stepInfo := &workerv1.FlowStepInfo{
			Name:             step.Name,
			Type:             step.Type,
			Args:             step.Args,
			Env:              step.Env,
			DependsOn:        step.DependsOn,
			DependencyPolicy: &step.DependencyPolicy,
		}

		// Add optional fields
		if step.Input != "" {
			stepInfo.Input = &step.Input
		}
		if step.Output != "" {
			stepInfo.Output = &step.Output
		}
		if step.SkipWhen != "" {
			stepInfo.SkipWhen = &step.SkipWhen
		}

		// Add retry config if present
		if step.Retry != nil {
			stepInfo.Retry = &workerv1.RetryConfig{
				MaxAttempts:       int32(step.Retry.MaxAttempts),
				DelaySeconds:      int32(step.Retry.Delay),
				BackoffMultiplier: 1.5, // Default backoff multiplier
			}
		}

		// Add runtime statistics
		enabled := true
		stepInfo.Enabled = &enabled

		// Get step statistics from worker runtime
		stepStats := s.runtime.GetStepStats(step.Name)
		if stepStats != nil {
			stepInfo.Active = &stepStats.Active
			executionCount := int32(stepStats.ExecutionCount)
			successCount := int32(stepStats.SuccessCount)
			stepInfo.ExecutionCount = &executionCount
			stepInfo.SuccessCount = &successCount

			if stepStats.LastExecution != nil {
				stepInfo.LastExecution = timestamppb.New(*stepStats.LastExecution)
			}

			if stepStats.LastOutput != nil {
				stepInfo.LastOutput = stepStats.LastOutput
			}

			if stepStats.LastError != nil {
				stepInfo.LastError = stepStats.LastError
			}
		}

		stepInfos = append(stepInfos, stepInfo)
	}

	response := &workerv1.FlowStepsResponse{
		Steps:     stepInfos,
		Total:     int32(len(stepInfos)),
		Timestamp: timestamppb.Now(),
	}

	return response, nil
}

// GetMetrics implements the get metrics RPC
func (s *Server) GetMetrics(ctx context.Context, req *emptypb.Empty) (*workerv1.MetricsResponse, error) {
	// Create basic metrics
	metrics := &workerv1.WorkerMetrics{}

	// TODO: Get actual metrics from worker runtime
	uptime := "1h30m"
	metrics.Uptime = &uptime

	avgExecTime := "2.5s"
	metrics.AvgExecutionTime = &avgExecTime

	lastActivity := timestamppb.Now()
	metrics.LastActivity = lastActivity

	response := &workerv1.MetricsResponse{
		Metrics:   metrics,
		Timestamp: timestamppb.Now(),
	}

	return response, nil
}

// StreamMetrics implements the stream metrics RPC
func (s *Server) StreamMetrics(req *workerv1.StreamMetricsRequest, stream workerv1.WorkerService_StreamMetricsServer) error {
	// For now, return a simple implementation
	// TODO: Implement actual metrics streaming
	return status.Errorf(codes.Unimplemented, "metrics streaming not yet implemented")
}

// GetConfig implements the get config RPC
func (s *Server) GetConfig(ctx context.Context, req *emptypb.Empty) (*workerv1.ConfigResponse, error) {
	// Get sanitized configuration
	settings := s.runtime.GetSettings()

	config := &workerv1.WorkerConfig{}

	// Add basic config fields
	config.Name = &s.runtime.Name

	workerType := s.runtime.Type()
	config.Type = &workerType

	enabled := "true" // TODO: Get actual enabled status
	config.Enabled = &enabled

	version := "1.0.0" // TODO: Get from build info
	config.Version = &version

	teamName := settings.GetTeamName()
	if teamName != "" {
		config.TeamName = &teamName
	}

	flowSteps := int32(len(settings.Flow))
	config.FlowSteps = &flowSteps

	response := &workerv1.ConfigResponse{
		Config:    config,
		Timestamp: timestamppb.Now(),
	}

	return response, nil
}

// Helper function to determine log role from filename
func determineLogRole(filename string) string {
	filename = strings.ToLower(filename)

	if strings.Contains(filename, "collector") {
		return "collector"
	} else if strings.Contains(filename, "executor") {
		return "executor"
	}

	return ""
}
