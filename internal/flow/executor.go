package flow

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"autoteam/internal/agent"
	"autoteam/internal/logger"
	"autoteam/internal/worker"

	"github.com/Masterminds/sprig/v3"
	"go.uber.org/zap"
)

// FlowExecutor executes dynamic flows with dependency resolution
type FlowExecutor struct {
	Steps         []worker.FlowStep
	Agents        map[string]agent.Agent
	MCPServers    map[string]worker.MCPServer
	WorkingDir    string
	Worker        *worker.Worker        // Worker configuration for template context
	WorkerRuntime *worker.WorkerRuntime // Runtime for step tracking (optional)
}

// StepOutput represents the output of a flow step
type StepOutput struct {
	Name     string
	Stdout   string
	Stderr   string
	Skipped  bool // Indicates if the step was skipped due to skip_when condition
	Failed   bool // Indicates if the step failed after all retries
	Canceled bool // Indicates if the step was canceled due to fail_fast policy
}

// FlowResult represents the result of executing a flow
type FlowResult struct {
	Steps   []StepOutput
	Success bool
	Error   error
}

// New creates a new FlowExecutor with the given steps and worker configuration
func New(steps []worker.FlowStep, mcpServers map[string]worker.MCPServer, workingDir string, worker *worker.Worker) *FlowExecutor {
	return &FlowExecutor{
		Steps:      steps,
		Agents:     make(map[string]agent.Agent),
		MCPServers: mcpServers,
		WorkingDir: workingDir,
		Worker:     worker,
	}
}

// SetWorkerRuntime sets the worker runtime for step tracking
func (fe *FlowExecutor) SetWorkerRuntime(workerRuntime *worker.WorkerRuntime) {
	fe.WorkerRuntime = workerRuntime
}

// Execute runs the flow with dependency resolution and parallel execution
func (fe *FlowExecutor) Execute(ctx context.Context) (*FlowResult, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Starting flow execution", zap.Int("total_steps", len(fe.Steps)))

	// Validate flow configuration
	if err := fe.validateFlow(); err != nil {
		return nil, fmt.Errorf("flow validation failed: %w", err)
	}

	// Resolve dependencies and create execution levels for parallel execution
	dependencyLevels, err := fe.resolveDependencyLevels()
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	lgr.Debug("Flow dependency resolution completed",
		zap.Int("levels", len(dependencyLevels)),
		zap.Any("execution_levels", dependencyLevels))

	// Create agents for each step
	if err := fe.createAgents(ctx); err != nil {
		return nil, fmt.Errorf("agent creation failed: %w", err)
	}

	// Execute steps level by level with parallel execution within each level
	stepOutputs := make(map[string]StepOutput)
	stepOutputsMutex := sync.RWMutex{}
	var allStepOutputs []StepOutput

	for levelIndex, level := range dependencyLevels {
		lgr.Debug("Processing execution level",
			zap.Int("level", levelIndex+1),
			zap.Int("total_levels", len(dependencyLevels)),
			zap.Strings("steps", level))

		// Execute all steps in this level in parallel
		levelOutputs, err := fe.executeLevel(ctx, level, stepOutputs, &stepOutputsMutex)
		if err != nil {
			// Add partial results and return error
			allStepOutputs = append(allStepOutputs, levelOutputs...)
			return &FlowResult{Steps: allStepOutputs, Success: false, Error: err}, err
		}

		// Store outputs from this level
		stepOutputsMutex.Lock()
		for _, output := range levelOutputs {
			stepOutputs[output.Name] = output
		}
		allStepOutputs = append(allStepOutputs, levelOutputs...)
		stepOutputsMutex.Unlock()

		lgr.Debug("Level execution completed",
			zap.Int("level", levelIndex+1),
			zap.Int("steps_completed", len(levelOutputs)))
	}

	lgr.Info("Flow execution completed", zap.Int("steps_executed", len(allStepOutputs)), zap.Bool("success", true))
	return &FlowResult{Steps: allStepOutputs, Success: true}, nil
}

// executeLevel executes all steps in a level in parallel
func (fe *FlowExecutor) executeLevel(ctx context.Context, stepNames []string, stepOutputs map[string]StepOutput, stepOutputsMutex *sync.RWMutex) ([]StepOutput, error) {
	lgr := logger.FromContext(ctx)

	if len(stepNames) == 0 {
		return []StepOutput{}, nil
	}

	// For single step, execute directly without goroutines
	if len(stepNames) == 1 {
		stepName := stepNames[0]
		step := fe.getStepByName(stepName)
		if step == nil {
			return nil, fmt.Errorf("step not found: %s", stepName)
		}

		lgr.Debug("Executing single step", zap.String("step_name", step.Name))

		stepOutputsMutex.RLock()
		stepOutputsCopy := make(map[string]StepOutput)
		for k, v := range stepOutputs {
			stepOutputsCopy[k] = v
		}
		stepOutputsMutex.RUnlock()

		output, err := fe.executeStep(ctx, *step, stepOutputsCopy)
		if err != nil {
			return []StepOutput{{
				Name:     step.Name,
				Stdout:   "",
				Stderr:   err.Error(),
				Skipped:  false,
				Failed:   true,
				Canceled: false,
			}}, err
		}

		return []StepOutput{*output}, nil
	}

	// For multiple steps, execute in parallel
	lgr.Debug("Executing parallel steps", zap.Int("count", len(stepNames)), zap.Strings("steps", stepNames))

	// Check if any step has fail_fast policy to enable context cancellation
	hasFailFastPolicy := false
	for _, stepName := range stepNames {
		if step := fe.getStepByName(stepName); step != nil {
			policy := step.DependencyPolicy
			if policy == "" || policy == "fail_fast" {
				hasFailFastPolicy = true
				break
			}
		}
	}

	// Create cancellable context if fail_fast policy is present
	execCtx := ctx
	var cancel context.CancelFunc
	if hasFailFastPolicy {
		execCtx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	type stepResult struct {
		output StepOutput
		err    error
	}

	resultChan := make(chan stepResult, len(stepNames))
	var wg sync.WaitGroup

	// Start parallel execution
	for _, stepName := range stepNames {
		wg.Add(1)
		go func(stepName string) {
			defer wg.Done()

			step := fe.getStepByName(stepName)
			if step == nil {
				resultChan <- stepResult{
					output: StepOutput{
						Name:     stepName,
						Stdout:   "",
						Stderr:   "step not found",
						Skipped:  false,
						Failed:   true,
						Canceled: false,
					},
					err: fmt.Errorf("step not found: %s", stepName),
				}
				return
			}

			lgr.Debug("Starting parallel step", zap.String("step_name", step.Name))

			// Create a copy of stepOutputs for thread safety
			stepOutputsMutex.RLock()
			stepOutputsCopy := make(map[string]StepOutput)
			for k, v := range stepOutputs {
				stepOutputsCopy[k] = v
			}
			stepOutputsMutex.RUnlock()

			// Execute the step with potentially cancellable context
			output, err := fe.executeStep(execCtx, *step, stepOutputsCopy)

			// Check if context was canceled
			if execCtx.Err() != nil {
				lgr.Info("Step canceled due to context cancellation",
					zap.String("step_name", step.Name),
					zap.Error(execCtx.Err()))
				resultChan <- stepResult{
					output: StepOutput{
						Name:     step.Name,
						Stdout:   "",
						Stderr:   "canceled due to fail_fast policy",
						Skipped:  false,
						Failed:   false,
						Canceled: true,
					},
					err: execCtx.Err(),
				}
				return
			}

			if err != nil {
				lgr.Error("Step failed in parallel execution",
					zap.String("step_name", step.Name),
					zap.Error(err),
					zap.String("error_type", fmt.Sprintf("%T", err)))

				// Cancel other parallel steps if this step has fail_fast policy
				policy := step.DependencyPolicy
				if (policy == "" || policy == "fail_fast") && cancel != nil {
					lgr.Info("Canceling other parallel steps due to fail_fast policy",
						zap.String("failed_step", step.Name))
					cancel()
				}

				resultChan <- stepResult{
					output: StepOutput{
						Name:     step.Name,
						Stdout:   "",
						Stderr:   err.Error(),
						Skipped:  false,
						Failed:   true,
						Canceled: false,
					},
					err: err,
				}
				return
			}

			lgr.Debug("Parallel step completed",
				zap.String("step_name", step.Name),
				zap.Int("output_size", len(output.Stdout)))

			resultChan <- stepResult{output: *output, err: nil}
		}(stepName)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	var outputs []StepOutput
	var firstError error

	for result := range resultChan {
		outputs = append(outputs, result.output)
		if result.err != nil && firstError == nil {
			firstError = result.err
		}
	}

	// Handle different dependency policies
	if firstError != nil {
		// Check if any step has fail_fast policy (default behavior)
		for _, output := range outputs {
			step := fe.getStepByName(output.Name)
			if step != nil {
				policy := step.DependencyPolicy
				if policy == "" || policy == "fail_fast" {
					// Fail fast - return immediately
					return outputs, firstError
				}
			}
		}

		// No fail_fast policy found, continue with partial results
		lgr.Warn("Level completed with failures, but no fail_fast policies",
			zap.Int("total_outputs", len(outputs)),
			zap.Error(firstError))
	}

	return outputs, nil
}

// validateFlow validates the flow configuration
func (fe *FlowExecutor) validateFlow() error {
	if len(fe.Steps) == 0 {
		return fmt.Errorf("flow must contain at least one step")
	}

	stepNames := make(map[string]bool)
	for _, step := range fe.Steps {
		if step.Name == "" {
			return fmt.Errorf("step name is required")
		}
		if step.Type == "" {
			return fmt.Errorf("step type is required for step: %s", step.Name)
		}
		if stepNames[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		stepNames[step.Name] = true

		// Validate dependencies exist
		for _, dep := range step.DependsOn {
			if !stepNames[dep] && !fe.stepExistsInFlow(dep) {
				return fmt.Errorf("step %s depends on non-existent step: %s", step.Name, dep)
			}
		}
	}

	return nil
}

// stepExistsInFlow checks if a step name exists in the flow
func (fe *FlowExecutor) stepExistsInFlow(stepName string) bool {
	for _, step := range fe.Steps {
		if step.Name == stepName {
			return true
		}
	}
	return false
}

// resolveDependencyLevels groups steps by dependency levels for parallel execution
func (fe *FlowExecutor) resolveDependencyLevels() ([][]string, error) {
	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize graph and in-degree count
	for _, step := range fe.Steps {
		graph[step.Name] = []string{}
		inDegree[step.Name] = 0
	}

	// Build edges and calculate in-degrees
	for _, step := range fe.Steps {
		for _, dep := range step.DependsOn {
			graph[dep] = append(graph[dep], step.Name)
			inDegree[step.Name]++
		}
	}

	var levels [][]string
	remainingSteps := len(fe.Steps)

	// Process steps level by level
	for remainingSteps > 0 {
		var currentLevel []string

		// Find all nodes with no incoming edges (ready to execute)
		for stepName, degree := range inDegree {
			if degree == 0 {
				currentLevel = append(currentLevel, stepName)
			}
		}

		// Check if we found any steps for this level
		if len(currentLevel) == 0 {
			return nil, fmt.Errorf("circular dependency detected in flow")
		}

		// Sort current level for deterministic ordering
		sort.Strings(currentLevel)
		levels = append(levels, currentLevel)

		// Remove current level steps and update in-degrees
		for _, stepName := range currentLevel {
			// Mark as processed by setting in-degree to -1
			inDegree[stepName] = -1
			remainingSteps--

			// Reduce in-degree for dependent steps
			for _, neighbor := range graph[stepName] {
				if inDegree[neighbor] > 0 {
					inDegree[neighbor]--
				}
			}
		}
	}

	return levels, nil
}

// createAgents creates agent instances for each step in the flow
func (fe *FlowExecutor) createAgents(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	for _, step := range fe.Steps {
		// Skip agent creation if agent already exists (for testing)
		if _, exists := fe.Agents[step.Name]; exists {
			continue
		}

		// Create agent config from step
		agentConfig := agent.AgentConfig{
			Type: step.Type,
			Args: step.Args,
			Env:  step.Env,
		}

		// Create agent with working directory + step name for proper MCP config paths
		// Extract just the directory name from workingDir (e.g., "senior_developer" from "/opt/autoteam/workers/senior_developer")
		baseName := filepath.Base(fe.WorkingDir)
		fullAgentName := fmt.Sprintf("%s/%s", baseName, step.Name)
		stepAgent, err := agent.CreateAgent(agentConfig, fullAgentName, fe.MCPServers)
		if err != nil {
			return fmt.Errorf("failed to create agent for step %s: %w", step.Name, err)
		}

		fe.Agents[step.Name] = stepAgent

		// Configure MCP servers if the agent supports configuration
		if configurable, ok := stepAgent.(agent.Configurable); ok {
			lgr.Debug("Configuring MCP servers for agent",
				zap.String("step_name", step.Name),
				zap.String("agent_type", step.Type),
				zap.Int("mcp_servers", len(fe.MCPServers)))

			if err := configurable.Configure(ctx); err != nil {
				return fmt.Errorf("failed to configure MCP servers for step %s: %w", step.Name, err)
			}

			lgr.Debug("MCP servers configured successfully",
				zap.String("step_name", step.Name))
		} else {
			lgr.Debug("Agent does not support MCP configuration",
				zap.String("step_name", step.Name),
				zap.String("agent_type", step.Type))
		}

		lgr.Debug("Agent created for step",
			zap.String("step_name", step.Name),
			zap.String("agent_type", step.Type))
	}

	return nil
}

// executeStep executes a single flow step
func (fe *FlowExecutor) executeStep(ctx context.Context, step worker.FlowStep, previousOutputs map[string]StepOutput) (*StepOutput, error) {
	lgr := logger.FromContext(ctx)

	// Check dependency policy first
	canExecute, reason := fe.evaluateDependencyPolicy(step, previousOutputs)
	if !canExecute {
		lgr.Info("Step skipped due to dependency policy",
			zap.String("step_name", step.Name),
			zap.String("dependency_policy", step.DependencyPolicy),
			zap.String("reason", reason))

		// Return skipped output
		return &StepOutput{
			Name:     step.Name,
			Stdout:   "",
			Stderr:   reason,
			Skipped:  true,
			Failed:   false,
			Canceled: false,
		}, nil
	}

	// Check skip condition
	shouldSkip, err := fe.evaluateSkipCondition(ctx, step, previousOutputs)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate skip condition for step %s: %w", step.Name, err)
	}

	if shouldSkip {
		lgr.Info("Step skipped",
			zap.String("step_name", step.Name),
			zap.String("skip_condition", step.SkipWhen))

		// Return empty output for skipped step
		return &StepOutput{
			Name:     step.Name,
			Stdout:   "",
			Stderr:   "",
			Skipped:  true,
			Failed:   false,
			Canceled: false,
		}, nil
	}

	// Get agent for this step
	stepAgent, exists := fe.Agents[step.Name]
	if !exists {
		// Debug: list all available agents
		var availableAgents []string
		for agentName := range fe.Agents {
			availableAgents = append(availableAgents, agentName)
		}
		lgr.Error("Agent not found for step",
			zap.String("step_name", step.Name),
			zap.Strings("available_agents", availableAgents))
		return nil, fmt.Errorf("agent not found for step: %s", step.Name)
	}

	// Prepare input data for template processing
	inputData := fe.prepareInputData(step, previousOutputs)

	// Process input field as template if it contains template syntax
	prompt := step.Input
	if step.Input != "" {
		transformedInput, transformErr := fe.applyTemplate(step.Input, inputData)
		if transformErr != nil {
			lgr.Warn("Input template processing failed, using original input",
				zap.String("step_name", step.Name),
				zap.String("input_template", step.Input),
				zap.Error(transformErr))
		} else {
			prompt = transformedInput
		}
	}

	// Mark step as active
	if fe.WorkerRuntime != nil {
		fe.WorkerRuntime.SetStepActive(step.Name, true)
		defer fe.WorkerRuntime.SetStepActive(step.Name, false)
	}

	// Log step execution start
	lgr.Info("Executing step",
		zap.String("step_name", step.Name),
		zap.String("agent_type", step.Type))

	lgr.Debug("Step prompt details",
		zap.String("step_name", step.Name),
		zap.String("prompt", prompt))

	// Set up run options
	runOptions := agent.RunOptions{
		MaxRetries:       1,
		ContinueMode:     false,
		WorkingDirectory: fmt.Sprintf("%s/%s", fe.WorkingDir, step.Name),
	}

	// Determine retry configuration
	maxAttempts := 1
	if step.Retry != nil && step.Retry.MaxAttempts > 0 {
		maxAttempts = step.Retry.MaxAttempts
	}

	// Execute agent with retry logic
	var output *agent.AgentOutput
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Update retry statistics
		if fe.WorkerRuntime != nil {
			stats := fe.WorkerRuntime.GetStepStats(step.Name)
			if stats != nil {
				stats.RetryAttempt = attempt - 1
				if attempt > 1 {
					now := time.Now()
					stats.LastRetryTime = &now
					stats.TotalRetries = attempt - 1
				}
			}
		}

		// Log retry attempt
		if attempt > 1 {
			lgr.Info("Retrying step execution",
				zap.String("step_name", step.Name),
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", maxAttempts),
				zap.String("backoff_strategy", step.Retry.Backoff))
		}

		// Execute the agent
		output, lastErr = stepAgent.Run(ctx, prompt, runOptions)
		if lastErr == nil {
			// Success - exit retry loop
			break
		}

		// If this isn't the last attempt, calculate delay and wait
		if attempt < maxAttempts {
			delay := fe.calculateRetryDelay(step.Retry, attempt)
			if delay > 0 {
				// Set next retry time for status tracking
				if fe.WorkerRuntime != nil {
					stats := fe.WorkerRuntime.GetStepStats(step.Name)
					if stats != nil {
						nextRetry := time.Now().Add(delay)
						stats.NextRetryTime = &nextRetry
					}
				}

				lgr.Info("Waiting before retry",
					zap.String("step_name", step.Name),
					zap.Duration("delay", delay),
					zap.Int("next_attempt", attempt+1))

				time.Sleep(delay)
			}
		}
	}

	// Check if all attempts failed
	if lastErr != nil {
		// Record failed execution statistics
		if fe.WorkerRuntime != nil {
			errorMsg := lastErr.Error()
			fe.WorkerRuntime.RecordStepExecution(step.Name, false, nil, &errorMsg)
		}
		return nil, fmt.Errorf("agent execution failed for step %s after %d attempts: %w", step.Name, maxAttempts, lastErr)
	}

	// Log agent completion
	lgr.Debug("Agent execution completed",
		zap.String("step_name", step.Name),
		zap.String("agent_type", step.Type),
		zap.String("stdout", output.Stdout),
		zap.String("stderr", output.Stderr),
	)

	// Apply output transformation if specified
	stdout := output.Stdout
	if step.Output != "" {
		templateData := map[string]interface{}{
			"stdout": output.Stdout,
			"stderr": output.Stderr,
		}

		transformedOutput, err := fe.applyTemplate(step.Output, templateData)
		if err != nil {
			lgr.Warn("Output transformation failed, using raw output",
				zap.String("step_name", step.Name),
				zap.String("output_template", step.Output),
				zap.Error(err))
		} else {
			stdout = transformedOutput
			lgr.Debug("Output transformed",
				zap.String("step_name", step.Name),
				zap.String("output", stdout),
			)
		}
	}

	// Log step completion
	lgr.Info("Step completed",
		zap.String("step_name", step.Name),
		zap.Bool("success", true))

	// Record step execution statistics directly
	if fe.WorkerRuntime != nil {
		success := output.Stderr == ""
		var outputPtr *string
		if stdout != "" {
			outputPtr = &stdout
		}
		var errorPtr *string
		if output.Stderr != "" {
			errorPtr = &output.Stderr
		}
		fe.WorkerRuntime.RecordStepExecution(step.Name, success, outputPtr, errorPtr)
	}

	return &StepOutput{
		Name:     step.Name,
		Stdout:   stdout,
		Stderr:   output.Stderr,
		Skipped:  false,
		Failed:   false, // Success case
		Canceled: false,
	}, nil
}

// prepareInputData prepares template data for input transformation
func (fe *FlowExecutor) prepareInputData(step worker.FlowStep, previousOutputs map[string]StepOutput) map[string]interface{} {
	// Collect inputs from dependencies
	var inputs []string
	for _, dep := range step.DependsOn {
		if output, exists := previousOutputs[dep]; exists {
			inputs = append(inputs, output.Stdout)
		}
	}

	return map[string]interface{}{
		"inputs": inputs,
		"step":   step,
		"flow":   fe,
	}
}

// evaluateSkipCondition evaluates a skip condition template and returns true if step should be skipped
func (fe *FlowExecutor) evaluateSkipCondition(ctx context.Context, step worker.FlowStep, previousOutputs map[string]StepOutput) (bool, error) {
	if step.SkipWhen == "" {
		return false, nil // No skip condition defined
	}

	lgr := logger.FromContext(ctx)

	// Prepare input data for skip condition evaluation (same as input transformers)
	inputData := fe.prepareInputData(step, previousOutputs)

	lgr.Debug("Evaluating skip condition",
		zap.String("step_name", step.Name),
		zap.String("skip_when", step.SkipWhen),
		zap.Any("input_data", inputData))

	// Evaluate the skip condition template
	result, err := fe.applyTemplate(step.SkipWhen, inputData)
	if err != nil {
		lgr.Warn("Skip condition template execution failed, assuming step should not be skipped",
			zap.String("step_name", step.Name),
			zap.String("skip_when", step.SkipWhen),
			zap.Error(err))
		return false, nil // Don't skip if template fails
	}

	// Trim whitespace and check if result is "true"
	shouldSkip := strings.TrimSpace(result) == "true"

	lgr.Debug("Skip condition evaluated",
		zap.String("step_name", step.Name),
		zap.String("result", result),
		zap.Bool("will_skip", shouldSkip))

	return shouldSkip, nil
}

// applyTemplate applies a Sprig template to the given data
func (fe *FlowExecutor) applyTemplate(templateStr string, data interface{}) (string, error) {
	// Create template with Sprig functions
	tmpl, err := template.New("transform").Funcs(sprig.FuncMap()).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("template parsing failed: %w", err)
	}

	// Execute template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

// getStepByName finds a step by its name
func (fe *FlowExecutor) getStepByName(name string) *worker.FlowStep {
	for _, step := range fe.Steps {
		if step.Name == name {
			return &step
		}
	}
	return nil
}

// evaluateDependencyPolicy checks if a step can execute based on its dependency policy
func (fe *FlowExecutor) evaluateDependencyPolicy(step worker.FlowStep, completedSteps map[string]StepOutput) (bool, string) {
	policy := step.DependencyPolicy
	if policy == "" {
		policy = "fail_fast" // Default policy
	}

	if len(step.DependsOn) == 0 {
		return true, "" // No dependencies means step can run
	}

	// Check each dependency exists
	var dependencyResults []StepOutput
	for _, dep := range step.DependsOn {
		if output, exists := completedSteps[dep]; exists {
			dependencyResults = append(dependencyResults, output)
		} else {
			return false, fmt.Sprintf("dependency %s has not completed yet", dep)
		}
	}

	switch policy {
	case "fail_fast":
		// Stop immediately if any dependency failed
		for _, dep := range dependencyResults {
			if dep.Failed {
				return false, fmt.Sprintf("dependency %s failed and policy is fail_fast", dep.Name)
			}
		}
		return true, ""

	case "all_success":
		// All dependencies must succeed
		for _, dep := range dependencyResults {
			if dep.Failed || dep.Skipped {
				return false, fmt.Sprintf("dependency %s did not succeed (failed=%v, skipped=%v)", dep.Name, dep.Failed, dep.Skipped)
			}
		}
		return true, ""

	case "all_complete":
		// All dependencies must be complete (success or failure doesn't matter)
		// Since we already verified all dependencies exist in completedSteps, this is always true
		return true, ""

	case "any_success":
		// At least one dependency must succeed
		hasSuccess := false
		for _, dep := range dependencyResults {
			if !dep.Failed && !dep.Skipped {
				hasSuccess = true
				break
			}
		}
		if !hasSuccess {
			return false, "no dependencies succeeded and policy is any_success"
		}
		return true, ""

	default:
		// Unknown policy, default to fail_fast behavior
		for _, dep := range dependencyResults {
			if dep.Failed {
				return false, fmt.Sprintf("dependency %s failed and unknown policy %s defaults to fail_fast", dep.Name, policy)
			}
		}
		return true, ""
	}
}

// calculateRetryDelay calculates the delay for a retry attempt based on the backoff strategy
func (fe *FlowExecutor) calculateRetryDelay(retry *worker.RetryConfig, attemptNumber int) time.Duration {
	if retry == nil || retry.Delay == 0 {
		return 0
	}

	baseDelay := time.Duration(retry.Delay) * time.Second
	maxDelay := time.Duration(retry.MaxDelay) * time.Second
	if maxDelay == 0 {
		maxDelay = time.Duration(300) * time.Second // Default max delay of 5 minutes
	}

	var delay time.Duration
	switch retry.Backoff {
	case "exponential":
		// Exponential backoff: delay * 2^(attempt-1)
		exp := 1
		for i := 1; i < attemptNumber; i++ {
			exp *= 2
		}
		delay = baseDelay * time.Duration(exp)
	case "linear":
		// Linear backoff: delay * attempt
		delay = baseDelay * time.Duration(attemptNumber)
	default: // "fixed" or empty
		// Fixed delay
		delay = baseDelay
	}

	// Cap at max delay
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}
