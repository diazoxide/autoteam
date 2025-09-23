package database

import (
	"fmt"
	"strings"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	DatabaseTypeSQLite     DatabaseType = "sqlite"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
)

// Config represents database configuration
type Config struct {
	Type DatabaseType `yaml:"type" json:"type"`
	DSN  string       `yaml:"dsn" json:"dsn"`
}

// DefaultConfig returns default database configuration
func DefaultConfig() Config {
	return Config{
		Type: DatabaseTypeSQLite,
		DSN:  "./autoteam.db",
	}
}

// Validate validates the database configuration
func (c *Config) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("database type is required")
	}

	if c.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	switch c.Type {
	case DatabaseTypeSQLite:
		// SQLite DSN can be just a file path
		return nil
	case DatabaseTypePostgreSQL:
		// Basic PostgreSQL DSN validation
		if !strings.Contains(c.DSN, "://") {
			return fmt.Errorf("invalid PostgreSQL DSN format")
		}
		return nil
	default:
		return fmt.Errorf("unsupported database type: %s", c.Type)
	}
}

// GetDialectorName returns the GORM dialector name for the database type
func (c *Config) GetDialectorName() string {
	switch c.Type {
	case DatabaseTypeSQLite:
		return "sqlite"
	case DatabaseTypePostgreSQL:
		return "postgres"
	default:
		return string(c.Type)
	}
}
