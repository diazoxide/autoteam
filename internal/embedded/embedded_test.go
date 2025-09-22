package embedded

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCurrentPlatform(t *testing.T) {
	platform := GetCurrentPlatform()

	if platform.OS == "" {
		t.Error("Platform OS should not be empty")
	}

	if platform.Arch == "" {
		t.Error("Platform Arch should not be empty")
	}

	t.Logf("Current platform: %s", platform.String())
}

func TestGetBinaryName(t *testing.T) {
	platform := Platform{OS: "linux", Arch: "amd64"}

	tests := []struct {
		binaryType BinaryType
		expected   string
	}{
		{Worker, "autoteam-worker-linux-amd64"},
		{ControlPlane, "autoteam-control-plane-linux-amd64"},
		{Dashboard, "autoteam-dashboard-linux-amd64"},
	}

	for _, tt := range tests {
		result := GetBinaryName(tt.binaryType, platform)
		if result != tt.expected {
			t.Errorf("GetBinaryName(%s, %s) = %s, want %s",
				tt.binaryType, platform.String(), result, tt.expected)
		}
	}
}

func TestGetEmbeddedBinaryPath(t *testing.T) {
	platform := Platform{OS: "linux", Arch: "amd64"}

	path := GetEmbeddedBinaryPath(Worker, platform)
	expected := "binaries/autoteam-worker-linux-amd64"

	if path != expected {
		t.Errorf("GetEmbeddedBinaryPath() = %s, want %s", path, expected)
	}
}

func TestGetEmbeddedScriptPath(t *testing.T) {
	path := GetEmbeddedScriptPath(EntrypointScript)
	expected := "scripts/entrypoint.sh"

	if path != expected {
		t.Errorf("GetEmbeddedScriptPath() = %s, want %s", path, expected)
	}
}

func TestManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Error("NewManager() should not return nil")
	}

	// Test GetInfo (this will work even without embedded files)
	info, err := manager.GetInfo()
	if err != nil {
		t.Errorf("GetInfo() returned error: %v", err)
	}

	if info == nil {
		t.Error("GetInfo() should not return nil")
	}

	t.Logf("Embedded info: %+v", info)
}

func TestExtractBinaryToTempWhenAvailable(t *testing.T) {
	platform := GetCurrentPlatform()

	// Test extraction for current platform if binary is available
	if IsBinaryAvailable(Worker, platform) {
		tempPath, err := ExtractBinaryToTemp(Worker, platform)
		if err != nil {
			t.Errorf("ExtractBinaryToTemp() returned error: %v", err)
		} else {
			defer os.Remove(tempPath) // Clean up

			// Check if file exists and is executable
			if _, err := os.Stat(tempPath); err != nil {
				t.Errorf("Extracted binary does not exist: %v", err)
			}

			t.Logf("Successfully extracted binary to: %s", tempPath)
		}
	} else {
		t.Logf("Worker binary not available for platform %s, skipping extraction test", platform.String())
	}
}

func TestExtractAllForPlatformWhenAvailable(t *testing.T) {
	platform := GetCurrentPlatform()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "autoteam-embedded-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager()

	// Try to extract all assets for current platform
	err = manager.ExtractAllForPlatform(platform, tempDir)

	// Check if any assets were extracted
	files, _ := filepath.Glob(filepath.Join(tempDir, "*"))

	if err != nil && len(files) == 0 {
		t.Logf("No embedded assets available for platform %s, skipping extraction test", platform.String())
	} else if err != nil {
		t.Errorf("ExtractAllForPlatform() returned error: %v", err)
	} else {
		t.Logf("Successfully extracted %d assets to %s", len(files), tempDir)
		for _, file := range files {
			t.Logf("  - %s", filepath.Base(file))
		}
	}
}