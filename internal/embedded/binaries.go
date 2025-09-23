package embedded

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"autoteam/internal/logger"
	"go.uber.org/zap"
)

// Embedded binaries - these will be populated by the build system
//
//go:embed binaries/*
var embeddedBinaries embed.FS

// BinaryType represents the type of binary
type BinaryType string

const (
	Worker       BinaryType = "worker"
	ControlPlane BinaryType = "control-plane"
	Dashboard    BinaryType = "dashboard"
)

// Platform represents the target platform
type Platform struct {
	OS   string
	Arch string
}

// String returns the platform as "os-arch" format
func (p Platform) String() string {
	return fmt.Sprintf("%s-%s", p.OS, p.Arch)
}

// GetCurrentPlatform returns the current runtime platform
func GetCurrentPlatform() Platform {
	return Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// GetBinaryName returns the expected binary filename for a given type and platform
func GetBinaryName(binaryType BinaryType, platform Platform) string {
	return fmt.Sprintf("autoteam-%s-%s", binaryType, platform.String())
}

// GetEmbeddedBinaryPath returns the embedded file path for a binary
func GetEmbeddedBinaryPath(binaryType BinaryType, platform Platform) string {
	return fmt.Sprintf("binaries/%s", GetBinaryName(binaryType, platform))
}

// ListEmbeddedBinaries returns a list of all embedded binaries
func ListEmbeddedBinaries() ([]string, error) {
	var binaries []string

	err := fs.WalkDir(embeddedBinaries, "binaries", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasPrefix(d.Name(), "autoteam-") {
			binaries = append(binaries, path)
		}

		return nil
	})

	return binaries, err
}

// ExtractBinary extracts an embedded binary to a destination path
func ExtractBinary(binaryType BinaryType, platform Platform, destPath string) error {
	log, _ := logger.NewLogger(logger.InfoLevel)

	embeddedPath := GetEmbeddedBinaryPath(binaryType, platform)

	log.Debug("Extracting embedded binary",
		zap.String("type", string(binaryType)),
		zap.String("platform", platform.String()),
		zap.String("embedded_path", embeddedPath),
		zap.String("dest_path", destPath))

	// Read the embedded binary
	data, err := embeddedBinaries.ReadFile(embeddedPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded binary %s: %w", embeddedPath, err)
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Write the binary to destination with executable permissions
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write binary to %s: %w", destPath, err)
	}

	log.Debug("Successfully extracted embedded binary",
		zap.String("dest_path", destPath),
		zap.Int("size_bytes", len(data)))

	return nil
}

// ExtractBinaryToTemp extracts an embedded binary to a temporary file and returns the path
func ExtractBinaryToTemp(binaryType BinaryType, platform Platform) (string, error) {
	log, _ := logger.NewLogger(logger.InfoLevel)

	// Create temporary file
	binaryName := GetBinaryName(binaryType, platform)
	tempFile, err := os.CreateTemp("", fmt.Sprintf("%s-*", binaryName))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFile.Close() // Close file handle, we'll overwrite it

	tempPath := tempFile.Name()

	// Extract to temporary location
	if err := ExtractBinary(binaryType, platform, tempPath); err != nil {
		os.Remove(tempPath) // Clean up on error
		return "", err
	}

	log.Debug("Extracted binary to temporary location",
		zap.String("type", string(binaryType)),
		zap.String("platform", platform.String()),
		zap.String("temp_path", tempPath))

	return tempPath, nil
}

// IsBinaryAvailable checks if a specific binary is available in the embedded files
func IsBinaryAvailable(binaryType BinaryType, platform Platform) bool {
	embeddedPath := GetEmbeddedBinaryPath(binaryType, platform)

	_, err := embeddedBinaries.ReadFile(embeddedPath)
	return err == nil
}

// GetAvailablePlatforms returns all platforms for which a specific binary type is available
func GetAvailablePlatforms(binaryType BinaryType) ([]Platform, error) {
	var platforms []Platform

	err := fs.WalkDir(embeddedBinaries, "binaries", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Parse filename: autoteam-{type}-{os}-{arch}
		filename := d.Name()
		expectedPrefix := fmt.Sprintf("autoteam-%s-", binaryType)

		if strings.HasPrefix(filename, expectedPrefix) {
			// Extract platform part
			platformPart := strings.TrimPrefix(filename, expectedPrefix)
			parts := strings.Split(platformPart, "-")

			if len(parts) >= 2 {
				platform := Platform{
					OS:   parts[0],
					Arch: strings.Join(parts[1:], "-"), // Handle arch like "arm64" or complex ones
				}
				platforms = append(platforms, platform)
			}
		}

		return nil
	})

	return platforms, err
}

// ExtractAllBinariesForPlatform extracts all binary types for a specific platform to a directory
func ExtractAllBinariesForPlatform(platform Platform, destDir string) error {
	log, _ := logger.NewLogger(logger.InfoLevel)

	binaryTypes := []BinaryType{Worker, ControlPlane, Dashboard}

	for _, binaryType := range binaryTypes {
		if !IsBinaryAvailable(binaryType, platform) {
			log.Warn("Binary not available for platform",
				zap.String("type", string(binaryType)),
				zap.String("platform", platform.String()))
			continue
		}

		binaryName := GetBinaryName(binaryType, platform)
		destPath := filepath.Join(destDir, binaryName)

		if err := ExtractBinary(binaryType, platform, destPath); err != nil {
			return fmt.Errorf("failed to extract %s for %s: %w", binaryType, platform.String(), err)
		}

		log.Info("Extracted binary",
			zap.String("type", string(binaryType)),
			zap.String("platform", platform.String()),
			zap.String("dest_path", destPath))
	}

	return nil
}
