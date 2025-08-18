package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
	"autoteam/internal/flow"
	"autoteam/internal/logger"
	"autoteam/internal/task"

	"go.uber.org/zap"
)

// Config contains configuration for the monitor
type Config struct {
	CheckInterval time.Duration
	TeamName      string
}

// Monitor handles flow-based agent monitoring
type Monitor struct {
	flowExecutor *flow.FlowExecutor // Dynamic flow executor
	flowSteps    []config.FlowStep  // Flow configuration
	config       Config
	globalConfig *entrypoint.Config
	taskService  *task.Service    // Service for task persistence operations
	httpServer   agent.HTTPServer // HTTP API server for monitoring
}

// New creates a new flow-based monitor instance
func New(flowSteps []config.FlowStep, monitorConfig Config, globalConfig *entrypoint.Config) *Monitor {
	// Get agent directory for task service
	agentNormalizedName := strings.ToLower(strings.ReplaceAll(globalConfig.Agent.Name, " ", "_"))
	agentDirectory := fmt.Sprintf("/opt/autoteam/agents/%s", agentNormalizedName)

	// Create flow executor
	flowExecutor := flow.New(flowSteps, globalConfig.MCPServers, agentDirectory)

	return &Monitor{
		flowExecutor: flowExecutor,
		flowSteps:    flowSteps,
		config:       monitorConfig,
		globalConfig: globalConfig,
		taskService:  task.NewService(agentDirectory),
	}
}

// Start starts the flow-based agent processing loop
func (m *Monitor) Start(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	lgr.Info("Starting flow-based agent monitor",
		zap.Duration("check_interval", m.config.CheckInterval),
		zap.Int("flow_steps", len(m.flowSteps)))

	lgr.Info("Starting flow processing loop: dynamic dependency-based execution")

	// Start HTTP API server if supported
	if err := m.startHTTPServer(ctx); err != nil {
		lgr.Warn("Failed to start HTTP API server", zap.Error(err))
	}

	// Start continuous flow processing loop
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lgr.Info("Monitor shutting down due to context cancellation")

			// Stop HTTP server
			if err := m.stopHTTPServer(ctx); err != nil {
				lgr.Warn("Failed to stop HTTP server", zap.Error(err))
			}

			return ctx.Err()

		case <-ticker.C:
			// Execute flow processing cycle
			if err := m.processFlowCycle(ctx); err != nil {
				lgr.Warn("Failed to process flow cycle", zap.Error(err))
			}
		}
	}
}

// processFlowCycle executes one cycle of the flow-based architecture
func (m *Monitor) processFlowCycle(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Starting flow processing cycle")

	// Execute the flow
	result, err := m.flowExecutor.Execute(ctx)
	if err != nil {
		lgr.Error("Flow execution failed", zap.Error(err))
		return fmt.Errorf("flow execution failed: %w", err)
	}

	if !result.Success {
		lgr.Warn("Flow execution completed with errors",
			zap.Int("steps_executed", len(result.Steps)),
			zap.Error(result.Error))
		return result.Error
	}

	lgr.Info("Flow execution completed successfully",
		zap.Int("steps_executed", len(result.Steps)))

	// Log step outputs for debugging
	for _, stepOutput := range result.Steps {
		lgr.Debug("Flow step output",
			zap.String("step_name", stepOutput.Name),
			zap.Int("stdout_length", len(stepOutput.Stdout)),
			zap.Int("stderr_length", len(stepOutput.Stderr)))
	}

	return nil
}

// startHTTPServer starts the HTTP API server for agent monitoring
func (m *Monitor) startHTTPServer(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("HTTP API server not implemented for flow mode")
	return nil
}

// stopHTTPServer stops the HTTP API server
func (m *Monitor) stopHTTPServer(ctx context.Context) error {
	if m.httpServer != nil && m.httpServer.IsRunning() {
		lgr := logger.FromContext(ctx)
		lgr.Info("Stopping HTTP API server")
		return m.httpServer.Stop(ctx)
	}
	return nil
}
