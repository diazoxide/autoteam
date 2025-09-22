package embedded

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"autoteam/internal/logger"
	"go.uber.org/zap"
)

// Embedded scripts
//
//go:embed scripts/*
var embeddedScripts embed.FS

// ScriptType represents the type of script
type ScriptType string

const (
	EntrypointScript ScriptType = "entrypoint.sh"
)

// GetEmbeddedScriptPath returns the embedded file path for a script
func GetEmbeddedScriptPath(scriptType ScriptType) string {
	return fmt.Sprintf("scripts/%s", scriptType)
}

// ExtractScript extracts an embedded script to a destination path
func ExtractScript(scriptType ScriptType, destPath string) error {
	log := logger.NewLogger(logger.InfoLevel)

	embeddedPath := GetEmbeddedScriptPath(scriptType)

	log.Debug("Extracting embedded script",
		zap.String("type", string(scriptType)),
		zap.String("embedded_path", embeddedPath),
		zap.String("dest_path", destPath))

	// Read the embedded script
	data, err := embeddedScripts.ReadFile(embeddedPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded script %s: %w", embeddedPath, err)
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Write the script to destination with executable permissions
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write script to %s: %w", destPath, err)
	}

	log.Debug("Successfully extracted embedded script",
		zap.String("dest_path", destPath),
		zap.Int("size_bytes", len(data)))

	return nil
}

// IsScriptAvailable checks if a specific script is available in the embedded files
func IsScriptAvailable(scriptType ScriptType) bool {
	embeddedPath := GetEmbeddedScriptPath(scriptType)

	_, err := embeddedScripts.ReadFile(embeddedPath)
	return err == nil
}