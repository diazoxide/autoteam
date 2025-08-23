package monitor

import (
	"context"
	"fmt"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/config"
	"autoteam/internal/flow"
	"autoteam/internal/logger"
	"autoteam/internal/task"

	"go.uber.org/zap"
)

// Config contains configuration for the monitor
type Config struct {
	SleepDuration time.Duration // Sleep duration between flow execution cycles
	TeamName      string
}

// Monitor handles flow-based agent monitoring
type Monitor struct {
	flowExecutor *flow.FlowExecutor // Dynamic flow executor
	flowSteps    []config.FlowStep  // Flow configuration
	config       Config
	worker       *config.Worker        // Worker configuration
	settings     config.WorkerSettings // Effective settings
	taskService  *task.Service         // Service for task persistence operations
	httpServer   agent.HTTPServer      // HTTP API server for monitoring
}

// New creates a new flow-based monitor instance
func New(worker *config.Worker, settings config.WorkerSettings, monitorConfig Config) *Monitor {
	// Get agent directory for task service
	agentDirectory := worker.GetWorkerDir()

	// Create flow executor with worker configuration and effective settings
	flowExecutor := flow.New(settings.Flow, settings.MCPServers, agentDirectory, worker)

	return &Monitor{
		flowExecutor: flowExecutor,
		flowSteps:    settings.Flow,
		config:       monitorConfig,
		worker:       worker,
		settings:     settings,
		taskService:  task.NewService(agentDirectory),
	}
}

// Start starts the flow-based agent processing loop
func (m *Monitor) Start(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	lgr.Info("Starting flow-based agent monitor",
		zap.Duration("sleep_duration", m.config.SleepDuration),
		zap.Int("flow_steps", len(m.flowSteps)))

	lgr.Info("Starting flow processing loop: dynamic dependency-based execution")

	// Start HTTP API server if supported
	if err := m.startHTTPServer(ctx); err != nil {
		lgr.Warn("Failed to start HTTP API server", zap.Error(err))
	}

	// Start continuous flow processing loop with sleep-based intervals
	for {
		// Check for cancellation before starting cycle
		select {
		case <-ctx.Done():
			lgr.Info("Monitor shutting down due to context cancellation")

			// Stop HTTP server
			if err := m.stopHTTPServer(ctx); err != nil {
				lgr.Warn("Failed to stop HTTP server", zap.Error(err))
			}

			return ctx.Err()
		default:
		}

		// Execute flow processing cycle
		cycleStart := time.Now()
		if err := m.processFlowCycle(ctx); err != nil {
			lgr.Warn("Failed to process flow cycle", zap.Error(err))
		}
		cycleEnd := time.Now()
		executionDuration := cycleEnd.Sub(cycleStart)

		// Log execution timing for monitoring
		lgr.Debug("Flow cycle completed",
			zap.Duration("execution_time", executionDuration),
			zap.Duration("sleep_duration", m.config.SleepDuration))

		// Sleep for the configured duration, with context cancellation check
		lgr.Debug("Sleeping before next flow cycle", zap.Duration("sleep_duration", m.config.SleepDuration))

		select {
		case <-ctx.Done():
			lgr.Info("Monitor shutting down during sleep interval")

			// Stop HTTP server
			if err := m.stopHTTPServer(ctx); err != nil {
				lgr.Warn("Failed to stop HTTP server", zap.Error(err))
			}

			return ctx.Err()
		case <-time.After(m.config.SleepDuration):
			// Continue to next cycle after sleep
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
