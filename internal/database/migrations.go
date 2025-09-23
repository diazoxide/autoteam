package database

import (
	"context"
	"fmt"

	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// Migrator handles database migrations
type Migrator struct {
	db *DB
}

// NewMigrator creates a new migrator
func NewMigrator(db *DB) *Migrator {
	return &Migrator{db: db}
}

// Migrate runs auto-migration for all models
func (m *Migrator) Migrate(ctx context.Context, models ...interface{}) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Starting database migration", zap.Int("models", len(models)))

	for i, model := range models {
		modelName := fmt.Sprintf("%T", model)
		lgr.Debug("Migrating model",
			zap.String("model", modelName),
			zap.Int("index", i+1),
			zap.Int("total", len(models)))

		if err := m.db.AutoMigrate(model); err != nil {
			lgr.Error("Failed to migrate model",
				zap.String("model", modelName),
				zap.Error(err))
			return fmt.Errorf("failed to migrate model %s: %w", modelName, err)
		}
	}

	lgr.Info("Database migration completed successfully")
	return nil
}

// DropTables drops all tables for the given models (use with caution)
func (m *Migrator) DropTables(ctx context.Context, models ...interface{}) error {
	lgr := logger.FromContext(ctx)
	lgr.Warn("Dropping database tables", zap.Int("models", len(models)))

	for _, model := range models {
		modelName := fmt.Sprintf("%T", model)
		lgr.Debug("Dropping table for model", zap.String("model", modelName))

		if err := m.db.Migrator().DropTable(model); err != nil {
			lgr.Error("Failed to drop table",
				zap.String("model", modelName),
				zap.Error(err))
			return fmt.Errorf("failed to drop table for model %s: %w", modelName, err)
		}
	}

	lgr.Info("Tables dropped successfully")
	return nil
}

// HasTable checks if a table exists for the given model
func (m *Migrator) HasTable(model interface{}) bool {
	return m.db.Migrator().HasTable(model)
}

// CreateIndexes creates indexes for better performance
func (m *Migrator) CreateIndexes(ctx context.Context) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Creating database indexes")

	// Define indexes that should be created
	indexes := []struct {
		table   string
		columns []string
		name    string
	}{
		// Worker indexes
		{"workers", []string{"name"}, "idx_workers_name"},
		{"workers", []string{"enabled"}, "idx_workers_enabled"},
		{"workers", []string{"created_at"}, "idx_workers_created_at"},

		// Worker settings indexes
		{"worker_settings", []string{"worker_id"}, "idx_worker_settings_worker_id"},

		// Flow steps indexes
		{"flow_steps", []string{"worker_id"}, "idx_flow_steps_worker_id"},
		{"flow_steps", []string{"worker_id", "`order`"}, "idx_flow_steps_worker_order"},
		{"flow_steps", []string{"name"}, "idx_flow_steps_name"},
		{"flow_steps", []string{"type"}, "idx_flow_steps_type"},
	}

	for _, idx := range indexes {
		lgr.Debug("Creating index",
			zap.String("table", idx.table),
			zap.String("name", idx.name),
			zap.Strings("columns", idx.columns))

		// Check if index already exists
		if m.db.Migrator().HasIndex(idx.table, idx.name) {
			lgr.Debug("Index already exists, skipping", zap.String("name", idx.name))
			continue
		}

		// Create index using raw SQL since GORM's index creation is limited
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
			idx.name, idx.table, joinColumns(idx.columns))

		if err := m.db.Exec(sql).Error; err != nil {
			lgr.Error("Failed to create index",
				zap.String("name", idx.name),
				zap.Error(err))
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}

		lgr.Debug("Index created successfully", zap.String("name", idx.name))
	}

	lgr.Info("Database indexes created successfully")
	return nil
}

// joinColumns joins column names with commas
func joinColumns(columns []string) string {
	if len(columns) == 0 {
		return ""
	}
	if len(columns) == 1 {
		return columns[0]
	}

	result := columns[0]
	for i := 1; i < len(columns); i++ {
		result += ", " + columns[i]
	}
	return result
}

// GetMigrationInfo returns information about current migration status
func (m *Migrator) GetMigrationInfo(ctx context.Context, models ...interface{}) (map[string]interface{}, error) {
	lgr := logger.FromContext(ctx)
	info := make(map[string]interface{})

	for _, model := range models {
		modelName := fmt.Sprintf("%T", model)
		hasTable := m.db.Migrator().HasTable(model)

		lgr.Debug("Checking migration status",
			zap.String("model", modelName),
			zap.Bool("has_table", hasTable))

		info[modelName] = map[string]interface{}{
			"has_table": hasTable,
		}
	}

	return info, nil
}
