// Package embedded provides functionality for managing embedded binaries and scripts
// in the AutoTeam single-binary distribution system.
//
// This package uses Go's embed package to include all necessary binaries
// (worker, control-plane, dashboard) and scripts (entrypoint.sh) within
// the main autoteam binary, eliminating external file dependencies.
package embedded

import (
	"fmt"

	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// Manager provides high-level functionality for managing embedded assets
type Manager struct {
	log *zap.Logger
}

// NewManager creates a new embedded assets manager
func NewManager() *Manager {
	log, _ := logger.NewLogger(logger.InfoLevel)
	return &Manager{
		log: log,
	}
}

// ExtractAllForPlatform extracts all binaries and scripts for a specific platform to a directory
func (m *Manager) ExtractAllForPlatform(platform Platform, destDir string) error {
	m.log.Info("Extracting all embedded assets for platform",
		zap.String("platform", platform.String()),
		zap.String("dest_dir", destDir))

	// Extract all binaries for the platform
	if err := ExtractAllBinariesForPlatform(platform, destDir); err != nil {
		return fmt.Errorf("failed to extract binaries: %w", err)
	}

	// Extract entrypoint script
	if IsScriptAvailable(EntrypointScript) {
		scriptPath := fmt.Sprintf("%s/%s", destDir, EntrypointScript)
		if err := ExtractScript(EntrypointScript, scriptPath); err != nil {
			return fmt.Errorf("failed to extract entrypoint script: %w", err)
		}

		m.log.Info("Extracted entrypoint script", zap.String("path", scriptPath))
	}

	m.log.Info("Successfully extracted all embedded assets",
		zap.String("platform", platform.String()),
		zap.String("dest_dir", destDir))

	return nil
}

// GetInfo returns information about embedded assets
func (m *Manager) GetInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Get available binaries
	binaries, err := ListEmbeddedBinaries()
	if err != nil {
		return nil, fmt.Errorf("failed to list embedded binaries: %w", err)
	}

	info["binaries"] = binaries
	info["total_binaries"] = len(binaries)

	// Check script availability
	info["entrypoint_available"] = IsScriptAvailable(EntrypointScript)

	// Get available platforms for each binary type
	binaryTypes := []BinaryType{Worker, ControlPlane, Dashboard}
	platformInfo := make(map[string][]Platform)

	for _, binaryType := range binaryTypes {
		platforms, err := GetAvailablePlatforms(binaryType)
		if err != nil {
			return nil, fmt.Errorf("failed to get platforms for %s: %w", binaryType, err)
		}
		platformInfo[string(binaryType)] = platforms
	}

	info["platforms"] = platformInfo

	return info, nil
}
