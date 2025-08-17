package monitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/config"
	"autoteam/internal/entrypoint"
	"autoteam/internal/logger"
	"autoteam/internal/task"

	"go.uber.org/zap"
)

// Config contains configuration for the two-layer monitor
type Config struct {
	CheckInterval time.Duration
	TeamName      string
}

// Monitor handles the two-layer agent monitoring loop
type Monitor struct {
	taskCollectionAgent agent.Agent // First Layer - uses CollectorAgent config
	taskExecutionAgent  agent.Agent // Second Layer - uses Agent config
	config              Config
	globalConfig        *entrypoint.Config
	firstLayerConfig    *config.AgentConfig // First Layer configuration with custom prompt
	secondLayerConfig   *config.AgentConfig // Second Layer configuration with custom prompt
	taskService         *task.Service       // Service for task persistence operations
	httpServer          agent.HTTPServer    // HTTP API server for monitoring
}

// New creates a new two-layer monitor instance
func New(collectionAgent, executionAgent agent.Agent, monitorConfig Config, globalConfig *entrypoint.Config) *Monitor {
	// Get agent directory for task service
	agentNormalizedName := strings.ToLower(strings.ReplaceAll(globalConfig.Agent.Name, " ", "_"))
	agentDirectory := fmt.Sprintf("/opt/autoteam/agents/%s", agentNormalizedName)

	return &Monitor{
		taskCollectionAgent: collectionAgent,
		taskExecutionAgent:  executionAgent,
		config:              monitorConfig,
		globalConfig:        globalConfig,
		firstLayerConfig:    nil, // Will be set via SetLayerConfigs if custom prompts are configured
		secondLayerConfig:   nil, // Will be set via SetLayerConfigs if custom prompts are configured
		taskService:         task.NewService(agentDirectory),
	}
}

// SetLayerConfigs sets the layer configurations for custom prompt support
func (m *Monitor) SetLayerConfigs(firstLayer, secondLayer *config.AgentConfig) {
	m.firstLayerConfig = firstLayer
	m.secondLayerConfig = secondLayer
}

// Start starts the two-layer agent processing loop
func (m *Monitor) Start(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Starting two-layer agent monitor",
		zap.Duration("check_interval", m.config.CheckInterval),
		zap.String("collection_agent", m.taskCollectionAgent.Type()),
		zap.String("execution_agent", m.taskExecutionAgent.Type()))

	// Repository access handled via MCP servers

	lgr.Info("Starting two-layer processing loop: task collection -> task execution")

	// Configure agents if they support configuration
	if err := m.configureAgents(ctx); err != nil {
		lgr.Warn("Failed to configure agents", zap.Error(err))
	}

	// Start HTTP API server if supported
	if err := m.startHTTPServer(ctx); err != nil {
		lgr.Warn("Failed to start HTTP API server", zap.Error(err))
	}

	// Start continuous two-layer processing loop
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
			// Execute two-layer processing cycle
			if err := m.processTwoLayerCycle(ctx); err != nil {
				lgr.Warn("Failed to process two-layer cycle", zap.Error(err))
			}
		}
	}
}

// processTwoLayerCycle executes one cycle of the two-layer architecture
func (m *Monitor) processTwoLayerCycle(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Starting two-layer processing cycle")

	// Layer 1: Collect tasks using collector agent
	tasks, err := m.collectTasksWithCollectorAgent(ctx)
	if err != nil {
		return fmt.Errorf("task collection failed: %w", err)
	}

	if tasks.IsEmpty() {
		lgr.Debug("No tasks found by aggregation agent")
		return nil
	}

	lgr.Info("Tasks collected by aggregation agent",
		zap.Int("task_count", tasks.Count()),
		zap.String("agent_type", m.taskCollectionAgent.Type()))

	// Layer 2: Execute highest priority task using execution agent
	return m.executeHighestPriorityTask(ctx, tasks)
}

// collectTasksWithCollectorAgent uses the first layer agent to collect tasks
func (m *Monitor) collectTasksWithCollectorAgent(ctx context.Context) (*task.TaskList, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Collecting tasks with collector agent", zap.String("agent_type", m.taskCollectionAgent.Type()))

	// Use custom prompt if configured, otherwise fallback to default first layer prompt
	var basePrompt string
	if m.firstLayerConfig != nil && m.firstLayerConfig.Prompt != nil {
		basePrompt = *m.firstLayerConfig.Prompt
		lgr.Debug("Using custom first layer prompt from configuration")
	} else {
		basePrompt = task.FirstLayerPrompt
		lgr.Debug("Using default first layer prompt")
	}

	// Add TODO_LIST format instruction to the prompt
	prompt := basePrompt + `

Return list of tasks in this exact format:

TODO_LIST: ["task text", "task text", ...]

IMPORTANT: 
- Use exactly "TODO_LIST: " followed by a JSON array
- If no tasks found, return: TODO_LIST: []
- Include the TODO_LIST format in your response regardless of other output`

	// Setup run options for task collection
	// Use agent-specific collector directory for first layer
	workingDirectory := m.getLayerWorkingDirectory("collector")

	runOptions := agent.RunOptions{
		MaxRetries:       1,
		ContinueMode:     false,
		WorkingDirectory: workingDirectory,
	}

	// Define file paths - tasks.json in common agent directory, output.txt in layer-specific directory
	outputFilePath := fmt.Sprintf("%s/output.txt", workingDirectory)

	// Log the working directory for debugging
	lgr.Info("Executing task collection with aggregation agent",
		zap.String("working_directory", workingDirectory),
		zap.String("agent_type", m.taskCollectionAgent.Type()))

	// Execute aggregation agent and capture stdout
	output, err := m.taskCollectionAgent.Run(ctx, prompt, runOptions)

	// Always save the stdout to output.txt for debugging (even on failure)
	if output != nil {
		if saveErr := m.saveAgentOutput(ctx, outputFilePath, output.Stdout, output.Stderr); saveErr != nil {
			lgr.Warn("Failed to save agent output", zap.Error(saveErr))
		}
	}

	if err != nil {
		lgr.Error("Task collection agent failed", zap.Error(err))
		// Do not create empty tasks.json - preserve existing tasks
		return nil, fmt.Errorf("aggregation agent execution failed: %w", err)
	}

	// Parse tasks from agent stdout
	newTasksJSON, err := task.ParseTasksFromStdout(output.Stdout)
	if err != nil {
		lgr.Error("Failed to parse tasks from stdout",
			zap.Error(err),
			zap.String("stdout", output.Stdout))

		// Do not create empty tasks.json - preserve existing tasks
		// Return empty task list to continue processing without losing existing data
		return task.CreateEmptyTaskList(), nil
	}

	// Use task service to merge and save tasks
	mergedTasksJSON, err := m.taskService.AddNewTasksAndSave(ctx, newTasksJSON)
	if err != nil {
		lgr.Warn("Failed to merge and save tasks", zap.Error(err))
		mergedTasksJSON = newTasksJSON
	}

	// Convert TasksJSON to TaskList for compatibility with existing code
	taskList := m.taskService.ConvertToTaskList(mergedTasksJSON)

	lgr.Info("Task collection completed successfully",
		zap.String("agent_type", m.taskCollectionAgent.Type()),
		zap.Int("tasks_count", taskList.Count()),
		zap.Int("todo_count", mergedTasksJSON.TodoCount()),
		zap.Int("done_count", mergedTasksJSON.DoneCount()),
		zap.Int("new_tasks_added", newTasksJSON.TodoCount()),
		zap.String("tasks_file", m.taskService.GetTasksPath()),
		zap.String("stdout_preview", truncateString(output.Stdout, 200)))

	return taskList, nil
}

// saveAgentOutput saves agent stdout and stderr to a file for debugging
func (m *Monitor) saveAgentOutput(ctx context.Context, filePath, stdout, stderr string) error {
	lgr := logger.FromContext(ctx)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create output content with timestamp and sections
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	content := fmt.Sprintf("=== Agent Output - %s ===\n\n", timestamp)

	// Add stdout section
	content += "=== STDOUT ===\n"
	if stdout != "" {
		content += stdout
	} else {
		content += "(empty)\n"
	}
	content += "\n\n"

	// Add stderr section
	content += "=== STDERR ===\n"
	if stderr != "" {
		content += stderr
	} else {
		content += "(empty)\n"
	}
	content += "\n"

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write agent output file: %w", err)
	}

	lgr.Debug("Agent output saved successfully",
		zap.String("path", filePath),
		zap.Int("stdout_length", len(stdout)),
		zap.Int("stderr_length", len(stderr)))

	return nil
}

// truncateString truncates a string to maxLength with "..." suffix if needed
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

// executeHighestPriorityTask uses the second layer agent to execute a task
func (m *Monitor) executeHighestPriorityTask(ctx context.Context, tasks *task.TaskList) error {
	lgr := logger.FromContext(ctx)

	// Get the highest priority task
	highestPriorityTask := tasks.GetHighestPriorityTask()
	if highestPriorityTask == nil {
		lgr.Debug("No tasks to execute")
		return nil
	}

	lgr.Info("Executing task with execution agent",
		zap.String("task_id", highestPriorityTask.ID),
		zap.String("task_type", highestPriorityTask.Type),
		zap.Int("priority", highestPriorityTask.Priority),
		zap.String("agent_type", m.taskExecutionAgent.Type()))

	// Repository operations handled via MCP servers
	// Use custom prompt if configured, otherwise fallback to building task-specific prompt
	var prompt string
	if m.secondLayerConfig != nil && m.secondLayerConfig.Prompt != nil {
		// Use custom second layer prompt with task description appended
		prompt = *m.secondLayerConfig.Prompt + "\n\n## Task Description\n" + highestPriorityTask.Description
		lgr.Debug("Using custom second layer prompt from configuration")
	} else {
		// Build task-specific prompt for execution using the task description
		prompt = task.BuildSecondLayerPrompt(highestPriorityTask.Description)
		lgr.Debug("Using default second layer prompt")
	}

	// Get working directory for execution agent (second layer)
	workingDir := m.getLayerWorkingDirectory("executor")

	// Setup run options for task execution with streaming logs
	runOptions := agent.RunOptions{
		MaxRetries:       3, // More retries for execution
		ContinueMode:     false,
		WorkingDirectory: workingDir,
	}

	// Define output file path in the executor directory (backward compatibility)
	executorOutputPath := fmt.Sprintf("%s/output.txt", workingDir)

	// Log the working directory for debugging
	lgr.Info("Executing task with execution agent",
		zap.String("working_directory", workingDir),
		zap.String("agent_type", m.taskExecutionAgent.Type()))

	// Execute the task with the execution agent using streaming logs
	output, err := m.executeWithStreamingLogs(ctx, prompt, runOptions, highestPriorityTask.Description)

	// Always save the stdout to output.txt for debugging (even on failure) - backward compatibility
	if output != nil {
		if saveErr := m.saveAgentOutput(ctx, executorOutputPath, output.Stdout, output.Stderr); saveErr != nil {
			lgr.Warn("Failed to save executor agent output", zap.Error(saveErr))
		}
	}

	if err != nil {
		lgr.Error("Task execution agent failed",
			zap.String("task_id", highestPriorityTask.ID),
			zap.String("task_type", highestPriorityTask.Type),
			zap.Error(err))
		return fmt.Errorf("execution agent failed for task %s: %w", highestPriorityTask.ID, err)
	}

	lgr.Info("Task executed successfully",
		zap.String("task_id", highestPriorityTask.ID),
		zap.String("task_type", highestPriorityTask.Type),
		zap.String("agent_type", m.taskExecutionAgent.Type()))

	// Mark task as completed using task service
	if err := m.taskService.MarkTaskCompleted(ctx, highestPriorityTask.Description); err != nil {
		lgr.Warn("Failed to mark task as completed in tasks.json",
			zap.Error(err),
			zap.String("task_description", highestPriorityTask.Description))
	}

	return nil
}

// configureAgents configures both agents if they support configuration
func (m *Monitor) configureAgents(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// Configure task collection agent
	if configurable, ok := m.taskCollectionAgent.(agent.Configurable); ok {
		lgr.Debug("Configuring task collection agent", zap.String("agent_type", m.taskCollectionAgent.Type()))
		if err := configurable.Configure(ctx); err != nil {
			lgr.Warn("Failed to configure task collection agent", zap.Error(err))
		}
	}

	// Configure task execution agent
	if configurable, ok := m.taskExecutionAgent.(agent.Configurable); ok {
		lgr.Debug("Configuring task execution agent", zap.String("agent_type", m.taskExecutionAgent.Type()))
		if err := configurable.Configure(ctx); err != nil {
			lgr.Warn("Failed to configure task execution agent", zap.Error(err))
		}
	}

	return nil
}

// getLayerWorkingDirectory returns the layer-specific working directory
func (m *Monitor) getLayerWorkingDirectory(layer string) string {
	// Normalize agent name consistently using the same logic as config.Agent.GetNormalizedName()
	agentNormalizedName := strings.ToLower(strings.ReplaceAll(m.globalConfig.Agent.Name, " ", "_"))
	return fmt.Sprintf("/opt/autoteam/agents/%s/%s", agentNormalizedName, layer)
}

// executeWithStreamingLogs executes the agent with streaming logs to a task-specific file
func (m *Monitor) executeWithStreamingLogs(ctx context.Context, prompt string, runOptions agent.RunOptions, taskDescription string) (*agent.AgentOutput, error) {
	lgr := logger.FromContext(ctx)

	// Create streaming logger for the executor working directory
	streamingLogger := task.NewStreamingLogger(runOptions.WorkingDirectory)

	// Create log file for this task
	logFile, err := streamingLogger.CreateLogFile(ctx, taskDescription)
	if err != nil {
		lgr.Warn("Failed to create streaming log file, proceeding without streaming logs", zap.Error(err))
		// Fall back to regular execution without streaming logs
		return m.taskExecutionAgent.Run(ctx, prompt, runOptions)
	}
	defer logFile.Close()

	lgr.Info("Created streaming log file for task",
		zap.String("task_description", taskDescription),
		zap.String("normalized_name", task.NormalizeTaskText(taskDescription)),
		zap.String("log_file", logFile.Name()))

	// Note: The current agent interface doesn't support streaming output redirection
	// For now, we'll execute the agent normally and then write the output to the log file
	// This maintains backward compatibility while adding the streaming log functionality

	// Execute the agent normally
	output, err := m.taskExecutionAgent.Run(ctx, prompt, runOptions)

	// Stream the output to the log file immediately after execution
	if output != nil {
		// Write stdout section
		if _, writeErr := logFile.WriteString("=== AGENT STDOUT ===\n"); writeErr != nil {
			lgr.Warn("Failed to write to streaming log file", zap.Error(writeErr))
		}
		if output.Stdout != "" {
			if _, writeErr := logFile.WriteString(output.Stdout); writeErr != nil {
				lgr.Warn("Failed to write stdout to streaming log file", zap.Error(writeErr))
			}
		} else {
			if _, writeErr := logFile.WriteString("(empty)\n"); writeErr != nil {
				lgr.Warn("Failed to write stdout to streaming log file", zap.Error(writeErr))
			}
		}
		if _, writeErr := logFile.WriteString("\n\n"); writeErr != nil {
			lgr.Warn("Failed to write to streaming log file", zap.Error(writeErr))
		}

		// Write stderr section
		if _, writeErr := logFile.WriteString("=== AGENT STDERR ===\n"); writeErr != nil {
			lgr.Warn("Failed to write to streaming log file", zap.Error(writeErr))
		}
		if output.Stderr != "" {
			if _, writeErr := logFile.WriteString(output.Stderr); writeErr != nil {
				lgr.Warn("Failed to write stderr to streaming log file", zap.Error(writeErr))
			}
		} else {
			if _, writeErr := logFile.WriteString("(empty)\n"); writeErr != nil {
				lgr.Warn("Failed to write stderr to streaming log file", zap.Error(writeErr))
			}
		}
		if _, writeErr := logFile.WriteString("\n"); writeErr != nil {
			lgr.Warn("Failed to write to streaming log file", zap.Error(writeErr))
		}

		// Write completion timestamp
		completionTime := time.Now().Format("2006-01-02 15:04:05")
		if _, writeErr := logFile.WriteString(fmt.Sprintf("=== Task Completed - %s ===\n", completionTime)); writeErr != nil {
			lgr.Warn("Failed to write completion timestamp to streaming log file", zap.Error(writeErr))
		}

		lgr.Info("Successfully wrote agent output to streaming log file",
			zap.String("log_file", logFile.Name()),
			zap.Int("stdout_length", len(output.Stdout)),
			zap.Int("stderr_length", len(output.Stderr)))
	}

	return output, err
}

// startHTTPServer starts the HTTP API server for agent monitoring
func (m *Monitor) startHTTPServer(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// Try to use execution agent first (primary agent for HTTP server)
	if httpCapable, ok := m.taskExecutionAgent.(agent.HTTPServerCapable); ok {
		agentNormalizedName := strings.ToLower(strings.ReplaceAll(m.globalConfig.Agent.Name, " ", "_"))
		workingDir := fmt.Sprintf("/opt/autoteam/agents/%s", agentNormalizedName)

		// Default port 8080, can be configured via environment variable if needed
		port := 8080
		apiKey := os.Getenv(strings.ToUpper(agentNormalizedName) + "_API_KEY")

		lgr.Info("Starting HTTP API server",
			zap.String("agent", m.taskExecutionAgent.Name()),
			zap.Int("port", port),
			zap.Bool("api_key_configured", apiKey != ""))

		m.httpServer = httpCapable.CreateHTTPServer(workingDir, port, apiKey)

		if err := m.httpServer.Start(ctx); err != nil {
			lgr.Error("Failed to start HTTP API server", zap.Error(err))
			return err
		}

		lgr.Info("HTTP API server started successfully",
			zap.String("url", m.httpServer.GetURL()),
			zap.String("docs_url", m.httpServer.GetDocsURL()))

		return nil
	}

	lgr.Debug("Agent does not support HTTP server capability")
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
