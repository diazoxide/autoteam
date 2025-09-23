package database

import (
	"context"
	"fmt"
	"time"

	"autoteam/internal/logger"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DB wraps gorm.DB with additional functionality
type DB struct {
	*gorm.DB
}

// Connection manages database connections
type Connection struct {
	db     *DB
	config Config
}

// NewConnection creates a new database connection
func NewConnection(config Config) (*Connection, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	conn := &Connection{
		config: config,
	}

	if err := conn.connect(); err != nil {
		return nil, err
	}

	return conn, nil
}

// connect establishes the database connection
func (c *Connection) connect() error {
	var dialector gorm.Dialector

	switch c.config.Type {
	case DatabaseTypeSQLite:
		dialector = sqlite.Open(c.config.DSN)
	case DatabaseTypePostgreSQL:
		dialector = postgres.Open(c.config.DSN)
	default:
		return fmt.Errorf("unsupported database type: %s", c.config.Type)
	}

	// Configure GORM
	config := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent), // We'll use our own logger
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		DisableForeignKeyConstraintWhenMigrating: false,
	}

	db, err := gorm.Open(dialector, config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	c.db = &DB{DB: db}
	return nil
}

// GetDB returns the database instance
func (c *Connection) GetDB() *DB {
	return c.db
}

// Close closes the database connection
func (c *Connection) Close() error {
	if c.db != nil {
		sqlDB, err := c.db.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Health checks database health
func (c *Connection) Health(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("database connection not established")
	}

	sqlDB, err := c.db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	return sqlDB.PingContext(ctx)
}

// WithContext returns a DB instance with context
func (db *DB) WithContext(ctx context.Context) *DB {
	return &DB{DB: db.DB.WithContext(ctx)}
}

// Transaction executes a function within a database transaction
func (db *DB) Transaction(ctx context.Context, fn func(*DB) error) error {
	lgr := logger.FromContext(ctx)

	return db.DB.Transaction(func(tx *gorm.DB) error {
		txDB := &DB{DB: tx}

		start := time.Now()
		err := fn(txDB)
		duration := time.Since(start)

		if err != nil {
			lgr.Debug("Transaction rolled back",
				zap.Error(err),
				zap.Duration("duration", duration))
			return err
		}

		lgr.Debug("Transaction committed",
			zap.Duration("duration", duration))
		return nil
	})
}

// UUID generates a new UUID
func UUID() uuid.UUID {
	return uuid.New()
}

// ParseUUID parses a UUID string
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
