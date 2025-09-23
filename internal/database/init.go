package database

import (
	"context"
	"fmt"

	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// Initialize creates a database connection and returns it
// Models should be migrated separately to avoid import cycles
func InitializeConnection(ctx context.Context, config Config) (*Connection, error) {
	lgr := logger.FromContext(ctx)
	lgr.Info("Initializing database", zap.String("type", string(config.Type)))

	// Create connection
	conn, err := NewConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	lgr.Info("Database connection established successfully")
	return conn, nil
}

// MigrateModels runs migrations for the given models
func MigrateModels(ctx context.Context, conn *Connection, models ...interface{}) error {
	lgr := logger.FromContext(ctx)
	lgr.Info("Running database migrations", zap.Int("models", len(models)))

	migrator := NewMigrator(conn.GetDB())

	if err := migrator.Migrate(ctx, models...); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Create indexes
	lgr.Info("Creating database indexes")
	if err := migrator.CreateIndexes(ctx); err != nil {
		lgr.Warn("Failed to create some indexes", zap.Error(err))
		// Don't fail on index creation errors
	}

	lgr.Info("Database migrations completed successfully")
	return nil
}

// GetDefaultConfig returns default database configuration for development
func GetDefaultConfig() Config {
	return Config{
		Type: DatabaseTypeSQLite,
		DSN:  "./autoteam.db",
	}
}
