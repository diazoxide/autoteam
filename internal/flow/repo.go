package flow

import (
	"context"
	"fmt"

	"autoteam/internal/database"
	"autoteam/internal/logger"
	"autoteam/internal/worker"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Repository defines the interface for flow step data operations
type Repository interface {
	// FlowStep CRUD operations
	CreateStep(ctx context.Context, step *worker.FlowStep) error
	UpdateStep(ctx context.Context, step *worker.FlowStep) error
	DeleteStep(ctx context.Context, id uuid.UUID) error
	GetStepByID(ctx context.Context, id uuid.UUID) (*worker.FlowStep, error)
	GetStepByName(ctx context.Context, workerID uuid.UUID, name string) (*worker.FlowStep, error)

	// Worker flow operations
	ListStepsByWorker(ctx context.Context, workerID uuid.UUID) ([]*worker.FlowStep, error)
	DeleteStepsByWorker(ctx context.Context, workerID uuid.UUID) error
	ReplaceWorkerFlow(ctx context.Context, workerID uuid.UUID, steps []*worker.FlowStep) error
	ReorderSteps(ctx context.Context, workerID uuid.UUID, stepIDs []uuid.UUID) error

	// Flow validation and utility
	ValidateStepDependencies(ctx context.Context, workerID uuid.UUID, step *worker.FlowStep) error
	GetMaxOrder(ctx context.Context, workerID uuid.UUID) (int, error)
	CountSteps(ctx context.Context, workerID uuid.UUID) (int64, error)
}

// repositoryImpl implements the Repository interface
type repositoryImpl struct {
	db *database.DB
}

// NewRepository creates a new flow repository
func NewRepository(db *database.DB) Repository {
	return &repositoryImpl{db: db}
}

// CreateStep creates a new flow step
func (r *repositoryImpl) CreateStep(ctx context.Context, step *worker.FlowStep) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Creating flow step",
		zap.String("worker_id", step.WorkerID.String()),
		zap.String("name", step.Name))

	if err := r.validateStep(step); err != nil {
		return fmt.Errorf("flow step validation failed: %w", err)
	}

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// If order is not set, set it to max + 1
		if step.Order == 0 {
			maxOrder, err := r.getMaxOrderTx(tx, step.WorkerID)
			if err != nil {
				return fmt.Errorf("failed to get max order: %w", err)
			}
			step.Order = maxOrder + 1
		}

		// Validate dependencies exist
		if err := r.validateStepDependenciesTx(tx, step.WorkerID, step); err != nil {
			return err
		}

		// Create step
		if err := tx.Create(step).Error; err != nil {
			return fmt.Errorf("failed to create flow step: %w", err)
		}

		lgr.Debug("Flow step created successfully",
			zap.String("id", step.ID.String()),
			zap.String("name", step.Name),
			zap.Int("order", step.Order))

		return nil
	})
}

// UpdateStep updates an existing flow step
func (r *repositoryImpl) UpdateStep(ctx context.Context, step *worker.FlowStep) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Updating flow step",
		zap.String("id", step.ID.String()),
		zap.String("name", step.Name))

	if err := r.validateStep(step); err != nil {
		return fmt.Errorf("flow step validation failed: %w", err)
	}

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Check if step exists
		var existing worker.FlowStep
		if err := tx.First(&existing, "id = ?", step.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("flow step not found: %s", step.ID)
			}
			return fmt.Errorf("failed to check step existence: %w", err)
		}

		// Validate dependencies (excluding self)
		if err := r.validateStepDependenciesTx(tx, step.WorkerID, step); err != nil {
			return err
		}

		// Update step
		if err := tx.Save(step).Error; err != nil {
			return fmt.Errorf("failed to update flow step: %w", err)
		}

		lgr.Debug("Flow step updated successfully",
			zap.String("id", step.ID.String()),
			zap.String("name", step.Name))

		return nil
	})
}

// DeleteStep deletes a flow step
func (r *repositoryImpl) DeleteStep(ctx context.Context, id uuid.UUID) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Deleting flow step", zap.String("id", id.String()))

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Get the step to check for dependencies
		var step worker.FlowStep
		if err := tx.First(&step, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("flow step not found: %s", id)
			}
			return fmt.Errorf("failed to get flow step: %w", err)
		}

		// Check if other steps depend on this one
		var dependentSteps []worker.FlowStep
		if err := tx.Where("worker_id = ? AND depends_on LIKE ?",
			step.WorkerID, "%\""+step.Name+"\"%").Find(&dependentSteps).Error; err != nil {
			return fmt.Errorf("failed to check dependencies: %w", err)
		}

		if len(dependentSteps) > 0 {
			stepNames := make([]string, len(dependentSteps))
			for i, s := range dependentSteps {
				stepNames[i] = s.Name
			}
			return fmt.Errorf("cannot delete step %s: it is a dependency for steps: %v",
				step.Name, stepNames)
		}

		// Delete step
		if err := tx.Delete(&step).Error; err != nil {
			return fmt.Errorf("failed to delete flow step: %w", err)
		}

		lgr.Debug("Flow step deleted successfully", zap.String("id", id.String()))
		return nil
	})
}

// GetStepByID retrieves a flow step by ID
func (r *repositoryImpl) GetStepByID(ctx context.Context, id uuid.UUID) (*worker.FlowStep, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Getting flow step by ID", zap.String("id", id.String()))

	var step worker.FlowStep
	err := r.db.WithContext(ctx).First(&step, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("flow step not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get flow step: %w", err)
	}

	lgr.Debug("Flow step retrieved successfully",
		zap.String("id", step.ID.String()),
		zap.String("name", step.Name))

	return &step, nil
}

// GetStepByName retrieves a flow step by name within a worker
func (r *repositoryImpl) GetStepByName(ctx context.Context, workerID uuid.UUID, name string) (*worker.FlowStep, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Getting flow step by name",
		zap.String("worker_id", workerID.String()),
		zap.String("name", name))

	var step worker.FlowStep
	err := r.db.WithContext(ctx).First(&step, "worker_id = ? AND name = ?", workerID, name).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("flow step not found: %s in worker %s", name, workerID)
		}
		return nil, fmt.Errorf("failed to get flow step: %w", err)
	}

	return &step, nil
}

// ListStepsByWorker retrieves all flow steps for a worker, ordered by execution order
func (r *repositoryImpl) ListStepsByWorker(ctx context.Context, workerID uuid.UUID) ([]*worker.FlowStep, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Listing flow steps by worker", zap.String("worker_id", workerID.String()))

	var steps []*worker.FlowStep
	err := r.db.WithContext(ctx).
		Where("worker_id = ?", workerID).
		Order("`order` ASC").
		Find(&steps).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list flow steps: %w", err)
	}

	lgr.Debug("Flow steps listed successfully",
		zap.String("worker_id", workerID.String()),
		zap.Int("count", len(steps)))

	return steps, nil
}

// DeleteStepsByWorker deletes all flow steps for a worker
func (r *repositoryImpl) DeleteStepsByWorker(ctx context.Context, workerID uuid.UUID) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Deleting all flow steps for worker", zap.String("worker_id", workerID.String()))

	result := r.db.WithContext(ctx).Delete(&worker.FlowStep{}, "worker_id = ?", workerID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete flow steps: %w", result.Error)
	}

	lgr.Debug("Flow steps deleted successfully",
		zap.String("worker_id", workerID.String()),
		zap.Int64("rows_affected", result.RowsAffected))

	return nil
}

// ReplaceWorkerFlow replaces all flow steps for a worker
func (r *repositoryImpl) ReplaceWorkerFlow(ctx context.Context, workerID uuid.UUID, steps []*worker.FlowStep) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Replacing worker flow",
		zap.String("worker_id", workerID.String()),
		zap.Int("new_steps", len(steps)))

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Delete existing steps
		if err := tx.Delete(&worker.FlowStep{}, "worker_id = ?", workerID).Error; err != nil {
			return fmt.Errorf("failed to delete existing flow steps: %w", err)
		}

		// Create new steps
		for i, step := range steps {
			step.WorkerID = workerID
			step.Order = i + 1 // Set order based on slice position

			if err := r.validateStep(step); err != nil {
				return fmt.Errorf("validation failed for step %s: %w", step.Name, err)
			}

			if err := tx.Create(step).Error; err != nil {
				return fmt.Errorf("failed to create flow step %s: %w", step.Name, err)
			}
		}

		lgr.Debug("Worker flow replaced successfully",
			zap.String("worker_id", workerID.String()),
			zap.Int("steps_created", len(steps)))

		return nil
	})
}

// ReorderSteps reorders flow steps for a worker
func (r *repositoryImpl) ReorderSteps(ctx context.Context, workerID uuid.UUID, stepIDs []uuid.UUID) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Reordering flow steps",
		zap.String("worker_id", workerID.String()),
		zap.Int("steps", len(stepIDs)))

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Update order for each step
		for i, stepID := range stepIDs {
			result := tx.Model(&worker.FlowStep{}).
				Where("id = ? AND worker_id = ?", stepID, workerID).
				Update("`order`", i+1)

			if result.Error != nil {
				return fmt.Errorf("failed to update order for step %s: %w", stepID, result.Error)
			}

			if result.RowsAffected == 0 {
				return fmt.Errorf("flow step not found or doesn't belong to worker: %s", stepID)
			}
		}

		lgr.Debug("Flow steps reordered successfully",
			zap.String("worker_id", workerID.String()),
			zap.Int("steps_reordered", len(stepIDs)))

		return nil
	})
}

// ValidateStepDependencies validates that all dependencies for a step exist
func (r *repositoryImpl) ValidateStepDependencies(ctx context.Context, workerID uuid.UUID, step *worker.FlowStep) error {
	return r.validateStepDependenciesTx(r.db.WithContext(ctx), workerID, step)
}

// GetMaxOrder returns the maximum order number for steps in a worker
func (r *repositoryImpl) GetMaxOrder(ctx context.Context, workerID uuid.UUID) (int, error) {
	return r.getMaxOrderTx(r.db.WithContext(ctx), workerID)
}

// CountSteps returns the number of steps for a worker
func (r *repositoryImpl) CountSteps(ctx context.Context, workerID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&worker.FlowStep{}).
		Where("worker_id = ?", workerID).Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count flow steps: %w", err)
	}

	return count, nil
}

// Private helper methods

func (r *repositoryImpl) validateStep(step *worker.FlowStep) error {
	if step.Name == "" {
		return fmt.Errorf("step name is required")
	}

	if step.Type == "" {
		return fmt.Errorf("step type is required")
	}

	if step.WorkerID == uuid.Nil {
		return fmt.Errorf("worker ID is required")
	}

	// Validate step type
	validTypes := map[string]bool{
		"claude": true,
		"gemini": true,
		"qwen":   true,
		"debug":  true,
	}

	if !validTypes[step.Type] {
		return fmt.Errorf("invalid step type: %s", step.Type)
	}

	return nil
}

func (r *repositoryImpl) validateStepDependenciesTx(tx *database.DB, workerID uuid.UUID, step *worker.FlowStep) error {
	if len(step.DependsOn) == 0 {
		return nil // No dependencies to validate
	}

	// Get all existing steps for this worker
	var existingSteps []worker.FlowStep
	if err := tx.Where("worker_id = ?", workerID).Find(&existingSteps).Error; err != nil {
		return fmt.Errorf("failed to get existing steps: %w", err)
	}

	// Create map of existing step names (excluding current step if updating)
	existingNames := make(map[string]bool)
	for _, existing := range existingSteps {
		if existing.ID != step.ID { // Exclude self when updating
			existingNames[existing.Name] = true
		}
	}

	// Validate each dependency exists
	for _, dep := range step.DependsOn {
		if !existingNames[dep] {
			return fmt.Errorf("dependency not found: %s", dep)
		}
	}

	return nil
}

func (r *repositoryImpl) getMaxOrderTx(tx *database.DB, workerID uuid.UUID) (int, error) {
	var maxOrder int
	err := tx.Model(&worker.FlowStep{}).
		Where("worker_id = ?", workerID).
		Select("COALESCE(MAX(`order`), 0)").
		Scan(&maxOrder).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get max order: %w", err)
	}

	return maxOrder, nil
}
