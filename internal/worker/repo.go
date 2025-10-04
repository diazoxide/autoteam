package worker

import (
	"context"
	"fmt"

	"autoteam/internal/database"
	"autoteam/internal/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Repository defines the interface for worker data operations
type Repository interface {
	// Worker CRUD operations
	Create(ctx context.Context, worker *Worker) error
	Update(ctx context.Context, worker *Worker) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Worker, error)
	GetByName(ctx context.Context, name string) (*Worker, error)
	List(ctx context.Context, filters ...Filter) ([]*Worker, error)
	ListEnabled(ctx context.Context) ([]*Worker, error)

	// Worker settings operations
	CreateOrUpdateSettings(ctx context.Context, settings *WorkerSettings) error
	GetSettingsByWorkerID(ctx context.Context, workerID uuid.UUID) (*WorkerSettings, error)
	DeleteSettings(ctx context.Context, workerID uuid.UUID) error

	// Flow step operations
	CreateFlowSteps(ctx context.Context, steps []FlowStep) error
	UpdateFlowSteps(ctx context.Context, workerID uuid.UUID, steps []FlowStep) error
	DeleteFlowStepsByWorkerID(ctx context.Context, workerID uuid.UUID) error
	GetFlowStepsByWorkerID(ctx context.Context, workerID uuid.UUID) ([]FlowStep, error)

	// Utility operations
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	Count(ctx context.Context, filters ...Filter) (int64, error)
}

// Filter represents a query filter for workers
type Filter struct {
	Field    string
	Operator string
	Value    interface{}
}

// repositoryImpl implements the Repository interface
type repositoryImpl struct {
	db *database.DB
}

// NewRepository creates a new worker repository
func NewRepository(db *database.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create creates a new worker
func (r *repositoryImpl) Create(ctx context.Context, worker *Worker) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Creating worker", zap.String("name", worker.Name))

	if err := ValidateWorker(worker); err != nil {
		return fmt.Errorf("worker validation failed: %w", err)
	}

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Create worker
		if err := tx.Create(worker).Error; err != nil {
			return fmt.Errorf("failed to create worker: %w", err)
		}

		lgr.Debug("Worker created successfully",
			zap.String("id", worker.ID.String()),
			zap.String("name", worker.Name))

		return nil
	})
}

// Update updates an existing worker
func (r *repositoryImpl) Update(ctx context.Context, worker *Worker) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Updating worker",
		zap.String("id", worker.ID.String()),
		zap.String("name", worker.Name))

	if err := ValidateWorker(worker); err != nil {
		return fmt.Errorf("worker validation failed: %w", err)
	}

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Check if worker exists
		var existing Worker
		if err := tx.First(&existing, "id = ?", worker.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("worker not found: %s", worker.ID)
			}
			return fmt.Errorf("failed to check worker existence: %w", err)
		}

		// Update worker
		if err := tx.Save(worker).Error; err != nil {
			return fmt.Errorf("failed to update worker: %w", err)
		}

		lgr.Debug("Worker updated successfully",
			zap.String("id", worker.ID.String()),
			zap.String("name", worker.Name))

		return nil
	})
}

// Delete soft deletes a worker
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Deleting worker", zap.String("id", id.String()))

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Soft delete worker (GORM will handle the DeletedAt field)
		result := tx.Delete(&Worker{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("failed to delete worker: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("worker not found: %s", id)
		}

		lgr.Debug("Worker deleted successfully", zap.String("id", id.String()))
		return nil
	})
}

// GetByID retrieves a worker by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*Worker, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Getting worker by ID", zap.String("id", id.String()))

	var worker Worker
	err := r.db.WithContext(ctx).
		Preload("Settings").
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("flow_steps.`order` ASC")
		}).
		First(&worker, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("worker not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	lgr.Debug("Worker retrieved successfully",
		zap.String("id", worker.ID.String()),
		zap.String("name", worker.Name))

	return &worker, nil
}

// GetByName retrieves a worker by name
func (r *repositoryImpl) GetByName(ctx context.Context, name string) (*Worker, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Getting worker by name", zap.String("name", name))

	var worker Worker
	err := r.db.WithContext(ctx).
		Preload("Settings").
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("flow_steps.`order` ASC")
		}).
		First(&worker, "name = ?", name).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("worker not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	lgr.Debug("Worker retrieved successfully",
		zap.String("id", worker.ID.String()),
		zap.String("name", worker.Name))

	return &worker, nil
}

// List retrieves all workers with optional filters
func (r *repositoryImpl) List(ctx context.Context, filters ...Filter) ([]*Worker, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Listing workers", zap.Int("filters", len(filters)))

	query := r.db.WithContext(ctx).
		Preload("Settings").
		Preload("FlowSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("flow_steps.[order] ASC")
		})

	// Apply filters
	for _, filter := range filters {
		query = applyFilter(query, filter)
	}

	var workers []*Worker
	if err := query.Find(&workers).Error; err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}

	lgr.Debug("Workers listed successfully", zap.Int("count", len(workers)))
	return workers, nil
}

// ListEnabled retrieves all enabled workers
func (r *repositoryImpl) ListEnabled(ctx context.Context) ([]*Worker, error) {
	return r.List(ctx, Filter{
		Field:    "enabled",
		Operator: "=",
		Value:    true,
	})
}

// CreateOrUpdateSettings creates or updates worker settings
func (r *repositoryImpl) CreateOrUpdateSettings(ctx context.Context, settings *WorkerSettings) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Creating/updating worker settings",
		zap.String("worker_id", settings.WorkerID.String()))

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Check if settings already exist
		var existing WorkerSettings
		err := tx.First(&existing, "worker_id = ?", settings.WorkerID).Error

		if err == gorm.ErrRecordNotFound {
			// Create new settings
			if createErr := tx.Create(settings).Error; createErr != nil {
				return fmt.Errorf("failed to create worker settings: %w", createErr)
			}
			lgr.Debug("Worker settings created successfully")
		} else if err != nil {
			return fmt.Errorf("failed to check existing settings: %w", err)
		} else {
			// Update existing settings
			settings.ID = existing.ID // Keep the same ID
			if updateErr := tx.Save(settings).Error; updateErr != nil {
				return fmt.Errorf("failed to update worker settings: %w", updateErr)
			}
			lgr.Debug("Worker settings updated successfully")
		}

		return nil
	})
}

// GetSettingsByWorkerID retrieves settings for a worker
func (r *repositoryImpl) GetSettingsByWorkerID(ctx context.Context, workerID uuid.UUID) (*WorkerSettings, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Getting worker settings", zap.String("worker_id", workerID.String()))

	var settings WorkerSettings
	err := r.db.WithContext(ctx).First(&settings, "worker_id = ?", workerID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("worker settings not found for worker: %s", workerID)
		}
		return nil, fmt.Errorf("failed to get worker settings: %w", err)
	}

	return &settings, nil
}

// DeleteSettings deletes worker settings
func (r *repositoryImpl) DeleteSettings(ctx context.Context, workerID uuid.UUID) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Deleting worker settings", zap.String("worker_id", workerID.String()))

	result := r.db.WithContext(ctx).Delete(&WorkerSettings{}, "worker_id = ?", workerID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete worker settings: %w", result.Error)
	}

	lgr.Debug("Worker settings deleted successfully",
		zap.String("worker_id", workerID.String()),
		zap.Int64("rows_affected", result.RowsAffected))

	return nil
}

// Exists checks if a worker exists by ID
func (r *repositoryImpl) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Worker{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check worker existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByName checks if a worker exists by name
func (r *repositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Worker{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check worker existence by name: %w", err)
	}
	return count > 0, nil
}

// Count returns the total count of workers with optional filters
func (r *repositoryImpl) Count(ctx context.Context, filters ...Filter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Worker{})

	// Apply filters
	for _, filter := range filters {
		query = applyFilter(query, filter)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count workers: %w", err)
	}

	return count, nil
}

// applyFilter applies a filter to a GORM query
func applyFilter(query *gorm.DB, filter Filter) *gorm.DB {
	switch filter.Operator {
	case "=", "eq":
		return query.Where(filter.Field+" = ?", filter.Value)
	case "!=", "neq":
		return query.Where(filter.Field+" != ?", filter.Value)
	case ">", "gt":
		return query.Where(filter.Field+" > ?", filter.Value)
	case ">=", "gte":
		return query.Where(filter.Field+" >= ?", filter.Value)
	case "<", "lt":
		return query.Where(filter.Field+" < ?", filter.Value)
	case "<=", "lte":
		return query.Where(filter.Field+" <= ?", filter.Value)
	case "like":
		return query.Where(filter.Field+" LIKE ?", filter.Value)
	case "in":
		return query.Where(filter.Field+" IN ?", filter.Value)
	case "not_in":
		return query.Where(filter.Field+" NOT IN ?", filter.Value)
	case "is_null":
		return query.Where(filter.Field + " IS NULL")
	case "is_not_null":
		return query.Where(filter.Field + " IS NOT NULL")
	default:
		// Default to equals
		return query.Where(filter.Field+" = ?", filter.Value)
	}
}

// CreateFlowSteps creates multiple flow steps for a worker
func (r *repositoryImpl) CreateFlowSteps(ctx context.Context, steps []FlowStep) error {
	lgr := logger.FromContext(ctx)

	if len(steps) == 0 {
		return nil
	}

	lgr.Debug("Creating flow steps", zap.Int("count", len(steps)))

	// Use a transaction to ensure all steps are created or none
	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		for _, step := range steps {
			if err := tx.Create(&step).Error; err != nil {
				lgr.Error("Failed to create flow step", zap.Error(err), zap.String("step_name", step.Name))
				return fmt.Errorf("failed to create flow step %s: %w", step.Name, err)
			}
		}
		return nil
	})
}

// UpdateFlowSteps replaces all flow steps for a worker
func (r *repositoryImpl) UpdateFlowSteps(ctx context.Context, workerID uuid.UUID, steps []FlowStep) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Updating flow steps", zap.String("worker_id", workerID.String()), zap.Int("count", len(steps)))

	return r.db.WithContext(ctx).Transaction(ctx, func(tx *database.DB) error {
		// Delete existing flow steps
		if err := tx.Where("worker_id = ?", workerID).Delete(&FlowStep{}).Error; err != nil {
			lgr.Error("Failed to delete existing flow steps", zap.Error(err))
			return fmt.Errorf("failed to delete existing flow steps: %w", err)
		}

		// Create new flow steps
		for _, step := range steps {
			if err := tx.Create(&step).Error; err != nil {
				lgr.Error("Failed to create flow step", zap.Error(err), zap.String("step_name", step.Name))
				return fmt.Errorf("failed to create flow step %s: %w", step.Name, err)
			}
		}
		return nil
	})
}

// DeleteFlowStepsByWorkerID deletes all flow steps for a worker
func (r *repositoryImpl) DeleteFlowStepsByWorkerID(ctx context.Context, workerID uuid.UUID) error {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Deleting flow steps by worker ID", zap.String("worker_id", workerID.String()))

	err := r.db.WithContext(ctx).Where("worker_id = ?", workerID).Delete(&FlowStep{}).Error
	if err != nil {
		lgr.Error("Failed to delete flow steps", zap.Error(err))
		return fmt.Errorf("failed to delete flow steps for worker %s: %w", workerID.String(), err)
	}

	lgr.Debug("Deleted flow steps successfully")
	return nil
}

// GetFlowStepsByWorkerID gets all flow steps for a worker
func (r *repositoryImpl) GetFlowStepsByWorkerID(ctx context.Context, workerID uuid.UUID) ([]FlowStep, error) {
	lgr := logger.FromContext(ctx)
	lgr.Debug("Getting flow steps by worker ID", zap.String("worker_id", workerID.String()))

	var flowSteps []FlowStep
	err := r.db.WithContext(ctx).Where("worker_id = ?", workerID).Order("`order` ASC").Find(&flowSteps).Error
	if err != nil {
		lgr.Error("Failed to get flow steps", zap.Error(err))
		return nil, fmt.Errorf("failed to get flow steps for worker %s: %w", workerID.String(), err)
	}

	lgr.Debug("Retrieved flow steps successfully", zap.Int("count", len(flowSteps)))
	return flowSteps, nil
}
