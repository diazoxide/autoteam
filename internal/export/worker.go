package export

import (
	"context"
	"fmt"
	"os"

	"autoteam/internal/flow"
	"autoteam/internal/logger"
	"autoteam/internal/worker"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// WorkerExportConfig represents the YAML export format
type WorkerExportConfig struct {
	Workers []worker.Worker `yaml:"workers"`
}

// ExportWorkersToYAML exports all workers to a YAML file
func ExportWorkersToYAML(ctx context.Context, workerRepo worker.Repository, filePath string) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Exporting workers to YAML", zap.String("file", filePath))

	// Get all workers from database
	workers, err := workerRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workers: %w", err)
	}

	lgr.Debug("Retrieved workers for export", zap.Int("count", len(workers)))

	// Convert to export format
	exportConfig := WorkerExportConfig{
		Workers: make([]worker.Worker, len(workers)),
	}

	for i, w := range workers {
		exportConfig.Workers[i] = *w

		// For YAML export, we need to populate the Flow field in settings
		// from the separate FlowSteps
		if w.Settings != nil && len(w.FlowSteps) > 0 {
			exportConfig.Workers[i].Settings.Flow = w.FlowSteps
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(exportConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	lgr.Info("Workers exported successfully",
		zap.String("file", filePath),
		zap.Int("workers", len(workers)))

	return nil
}

// ExportWorkerToYAML exports a single worker to YAML
func ExportWorkerToYAML(ctx context.Context, workerRepo worker.Repository, workerID string, filePath string) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Exporting worker to YAML",
		zap.String("worker_id", workerID),
		zap.String("file", filePath))

	// Parse UUID
	id, err := worker.ParseUUID(workerID)
	if err != nil {
		return fmt.Errorf("invalid worker ID: %w", err)
	}

	// Get worker from database
	w, err := workerRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get worker: %w", err)
	}

	// For single worker export, populate Flow field in settings
	if w.Settings != nil && len(w.FlowSteps) > 0 {
		w.Settings.Flow = w.FlowSteps
	}

	// Marshal to YAML
	data, err := yaml.Marshal(w)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	lgr.Info("Worker exported successfully",
		zap.String("worker_id", workerID),
		zap.String("name", w.Name),
		zap.String("file", filePath))

	return nil
}

// ExportWorkersToYAMLString exports workers to YAML string (for API responses)
func ExportWorkersToYAMLString(ctx context.Context, workerRepo worker.Repository) (string, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Exporting workers to YAML string")

	// Get all workers from database
	workers, err := workerRepo.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list workers: %w", err)
	}

	// Convert to export format
	exportConfig := WorkerExportConfig{
		Workers: make([]worker.Worker, len(workers)),
	}

	for i, w := range workers {
		exportConfig.Workers[i] = *w

		// Populate Flow field in settings from FlowSteps
		if w.Settings != nil && len(w.FlowSteps) > 0 {
			exportConfig.Workers[i].Settings.Flow = w.FlowSteps
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(exportConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	lgr.Debug("Workers exported to YAML string successfully",
		zap.Int("workers", len(workers)))

	return string(data), nil
}

// ExportWorkerToYAMLString exports a single worker to YAML string
func ExportWorkerToYAMLString(ctx context.Context, workerRepo worker.Repository, workerID string) (string, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Exporting worker to YAML string", zap.String("worker_id", workerID))

	// Parse UUID
	id, err := worker.ParseUUID(workerID)
	if err != nil {
		return "", fmt.Errorf("invalid worker ID: %w", err)
	}

	// Get worker from database
	w, err := workerRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get worker: %w", err)
	}

	// Populate Flow field in settings
	if w.Settings != nil && len(w.FlowSteps) > 0 {
		w.Settings.Flow = w.FlowSteps
	}

	// Marshal to YAML
	data, err := yaml.Marshal(w)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	lgr.Debug("Worker exported to YAML string successfully",
		zap.String("worker_id", workerID),
		zap.String("name", w.Name))

	return string(data), nil
}

// ImportOptions configures import behavior
type ImportOptions struct {
	Replace    bool // If true, clear database before importing
	SkipErrors bool // If true, continue importing even if some workers fail
	DryRun     bool // If true, validate but don't actually import
}

// ImportWorkersFromYAML imports workers from a YAML file
func ImportWorkersFromYAML(ctx context.Context, workerRepo worker.Repository, flowRepo flow.Repository, filePath string, opts ImportOptions) (*ImportResult, error) {
	lgr := logger.FromContext(ctx)
	lgr.Info("Importing workers from YAML",
		zap.String("file", filePath),
		zap.Bool("replace", opts.Replace),
		zap.Bool("skip_errors", opts.SkipErrors),
		zap.Bool("dry_run", opts.DryRun))

	// Read YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	return ImportWorkersFromYAMLData(ctx, workerRepo, flowRepo, data, opts)
}

// ImportWorkersFromYAMLData imports workers from YAML data
func ImportWorkersFromYAMLData(ctx context.Context, workerRepo worker.Repository, flowRepo flow.Repository, data []byte, opts ImportOptions) (*ImportResult, error) {
	lgr := logger.FromContext(ctx)

	// Try parsing as WorkerExportConfig first (multiple workers)
	var exportConfig WorkerExportConfig
	err := yaml.Unmarshal(data, &exportConfig)
	if err != nil {
		// Try parsing as single Worker
		var singleWorker worker.Worker
		err = yaml.Unmarshal(data, &singleWorker)
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
		exportConfig.Workers = []worker.Worker{singleWorker}
	}

	lgr.Debug("Parsed YAML data", zap.Int("workers", len(exportConfig.Workers)))

	result := &ImportResult{
		TotalWorkers:  len(exportConfig.Workers),
		ImportedCount: 0,
		SkippedCount:  0,
		ErrorCount:    0,
		Errors:        make([]ImportError, 0),
	}

	// Validate all workers first
	for i, w := range exportConfig.Workers {
		if err := worker.ValidateWorker(&w); err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, ImportError{
				WorkerIndex: i,
				WorkerName:  w.Name,
				Error:       fmt.Sprintf("validation failed: %v", err),
			})

			if !opts.SkipErrors {
				return result, fmt.Errorf("worker validation failed for %s: %w", w.Name, err)
			}
		}
	}

	if opts.DryRun {
		lgr.Info("Dry run completed - validation successful", zap.Int("workers", len(exportConfig.Workers)))
		result.ImportedCount = len(exportConfig.Workers) - result.ErrorCount
		return result, nil
	}

	// Clear database if replace option is set
	if opts.Replace {
		lgr.Info("Replacing existing workers")
		existingWorkers, err := workerRepo.List(ctx)
		if err != nil {
			return result, fmt.Errorf("failed to list existing workers: %w", err)
		}

		for _, existing := range existingWorkers {
			if err := workerRepo.Delete(ctx, existing.ID); err != nil {
				lgr.Error("Failed to delete existing worker",
					zap.String("id", existing.ID.String()),
					zap.String("name", existing.Name),
					zap.Error(err))
			}
		}
	}

	// Import workers
	for i, w := range exportConfig.Workers {
		lgr.Debug("Importing worker",
			zap.Int("index", i+1),
			zap.String("name", w.Name))

		if err := importSingleWorker(ctx, workerRepo, flowRepo, &w); err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, ImportError{
				WorkerIndex: i,
				WorkerName:  w.Name,
				Error:       err.Error(),
			})

			if !opts.SkipErrors {
				return result, fmt.Errorf("failed to import worker %s: %w", w.Name, err)
			}

			lgr.Error("Failed to import worker",
				zap.String("name", w.Name),
				zap.Error(err))

			result.SkippedCount++
		} else {
			result.ImportedCount++
			lgr.Debug("Worker imported successfully", zap.String("name", w.Name))
		}
	}

	lgr.Info("Import completed",
		zap.Int("total", result.TotalWorkers),
		zap.Int("imported", result.ImportedCount),
		zap.Int("skipped", result.SkippedCount),
		zap.Int("errors", result.ErrorCount))

	return result, nil
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	TotalWorkers  int           `json:"total_workers"`
	ImportedCount int           `json:"imported_count"`
	SkippedCount  int           `json:"skipped_count"`
	ErrorCount    int           `json:"error_count"`
	Errors        []ImportError `json:"errors,omitempty"`
}

// ImportError represents an error that occurred during import
type ImportError struct {
	WorkerIndex int    `json:"worker_index"`
	WorkerName  string `json:"worker_name"`
	Error       string `json:"error"`
}

// importSingleWorker imports a single worker and its flow steps
func importSingleWorker(ctx context.Context, workerRepo worker.Repository, flowRepo flow.Repository, w *worker.Worker) error {
	lgr := logger.FromContext(ctx)

	// Check if worker with same name already exists
	existing, err := workerRepo.GetByName(ctx, w.Name)
	if err == nil {
		// Worker exists, update it
		lgr.Debug("Worker already exists, updating", zap.String("name", w.Name))
		w.ID = existing.ID

		// Update worker
		if err := workerRepo.Update(ctx, w); err != nil {
			return fmt.Errorf("failed to update existing worker: %w", err)
		}
	} else {
		// Worker doesn't exist, create new one
		lgr.Debug("Creating new worker", zap.String("name", w.Name))
		w.ID = worker.UUID() // Generate new UUID

		if err := workerRepo.Create(ctx, w); err != nil {
			return fmt.Errorf("failed to create worker: %w", err)
		}
	}

	// Handle settings
	if w.Settings != nil {
		w.Settings.WorkerID = w.ID
		w.Settings.ID = worker.UUID() // Generate new UUID for settings

		if err := workerRepo.CreateOrUpdateSettings(ctx, w.Settings); err != nil {
			return fmt.Errorf("failed to create/update worker settings: %w", err)
		}

		// Handle flow steps from settings.Flow
		if len(w.Settings.Flow) > 0 {
			lgr.Debug("Importing flow steps from settings",
				zap.String("worker", w.Name),
				zap.Int("steps", len(w.Settings.Flow)))

			// Delete existing flow steps
			if err := flowRepo.DeleteStepsByWorker(ctx, w.ID); err != nil {
				return fmt.Errorf("failed to delete existing flow steps: %w", err)
			}

			// Create new flow steps
			steps := make([]*worker.FlowStep, len(w.Settings.Flow))
			for i, step := range w.Settings.Flow {
				steps[i] = &worker.FlowStep{
					ID:               worker.UUID(),
					WorkerID:         w.ID,
					Name:             step.Name,
					Type:             step.Type,
					Order:            i + 1,
					Args:             step.Args,
					Env:              step.Env,
					DependsOn:        step.DependsOn,
					Input:            step.Input,
					Output:           step.Output,
					SkipWhen:         step.SkipWhen,
					DependencyPolicy: step.DependencyPolicy,
					Retry:            step.Retry,
				}
			}

			if err := flowRepo.ReplaceWorkerFlow(ctx, w.ID, steps); err != nil {
				return fmt.Errorf("failed to create flow steps: %w", err)
			}
		}
	}

	// Handle flow steps directly (for cases where FlowSteps is populated)
	if len(w.FlowSteps) > 0 {
		lgr.Debug("Importing direct flow steps",
			zap.String("worker", w.Name),
			zap.Int("steps", len(w.FlowSteps)))

		// Delete existing flow steps
		if err := flowRepo.DeleteStepsByWorker(ctx, w.ID); err != nil {
			return fmt.Errorf("failed to delete existing flow steps: %w", err)
		}

		// Create new flow steps
		steps := make([]*worker.FlowStep, len(w.FlowSteps))
		for i, step := range w.FlowSteps {
			steps[i] = &worker.FlowStep{
				ID:               worker.UUID(),
				WorkerID:         w.ID,
				Name:             step.Name,
				Type:             step.Type,
				Order:            i + 1,
				Args:             step.Args,
				Env:              step.Env,
				DependsOn:        step.DependsOn,
				Input:            step.Input,
				Output:           step.Output,
				SkipWhen:         step.SkipWhen,
				DependencyPolicy: step.DependencyPolicy,
				Retry:            step.Retry,
			}
		}

		if err := flowRepo.ReplaceWorkerFlow(ctx, w.ID, steps); err != nil {
			return fmt.Errorf("failed to create flow steps: %w", err)
		}
	}

	lgr.Debug("Worker imported successfully",
		zap.String("name", w.Name),
		zap.String("id", w.ID.String()))

	return nil
}
