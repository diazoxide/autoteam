package embedded

import (
	"os"
	"path/filepath"
	"strings"
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

func TestBinaryTypeConstants(t *testing.T) {
	tests := []struct {
		binaryType BinaryType
		expected   string
	}{
		{Worker, "worker"},
		{ControlPlane, "control-plane"},
		{Dashboard, "dashboard"},
	}

	for _, tt := range tests {
		if string(tt.binaryType) != tt.expected {
			t.Errorf("BinaryType constant mismatch: got %s, want %s", tt.binaryType, tt.expected)
		}
	}
}

func TestPlatformString(t *testing.T) {
	tests := []struct {
		platform Platform
		expected string
	}{
		{Platform{OS: "linux", Arch: "amd64"}, "linux-amd64"},
		{Platform{OS: "darwin", Arch: "arm64"}, "darwin-arm64"},
		{Platform{OS: "windows", Arch: "386"}, "windows-386"},
	}

	for _, tt := range tests {
		result := tt.platform.String()
		if result != tt.expected {
			t.Errorf("Platform.String() = %s, want %s", result, tt.expected)
		}
	}
}

func TestListEmbeddedBinaries(t *testing.T) {
	binaries, err := ListEmbeddedBinaries()
	if err != nil {
		t.Errorf("ListEmbeddedBinaries failed: %v", err)
	}

	// Should not crash even if no binaries are embedded
	t.Logf("Found %d embedded binaries", len(binaries))
	for _, binary := range binaries {
		if !strings.HasPrefix(binary, "binaries/autoteam-") {
			t.Errorf("Unexpected binary path format: %s", binary)
		}
	}
}

func TestIsBinaryAvailable(t *testing.T) {
	// Test with current platform (should work regardless of embedded binaries)
	currentPlatform := GetCurrentPlatform()

	binaryTypes := []BinaryType{Worker, ControlPlane, Dashboard}
	for _, binaryType := range binaryTypes {
		available := IsBinaryAvailable(binaryType, currentPlatform)
		t.Logf("Binary %s available for %s: %v", binaryType, currentPlatform.String(), available)
		// Don't assert true/false since it depends on build state
	}
}

func TestGetAvailablePlatforms(t *testing.T) {
	binaryTypes := []BinaryType{Worker, ControlPlane, Dashboard}

	for _, binaryType := range binaryTypes {
		platforms, err := GetAvailablePlatforms(binaryType)
		if err != nil {
			t.Errorf("GetAvailablePlatforms(%s) failed: %v", binaryType, err)
			continue
		}

		t.Logf("Available platforms for %s: %d", binaryType, len(platforms))
		for _, platform := range platforms {
			if platform.OS == "" || platform.Arch == "" {
				t.Errorf("Invalid platform returned: %+v", platform)
			}
		}
	}
}

func TestExtractBinaryToTemp(t *testing.T) {
	// Test with current platform
	currentPlatform := GetCurrentPlatform()

	// Only test if binary is available (to avoid test failures in different build contexts)
	if IsBinaryAvailable(Worker, currentPlatform) {
		tempPath, err := ExtractBinaryToTemp(Worker, currentPlatform)
		if err != nil {
			t.Errorf("ExtractBinaryToTemp failed: %v", err)
		} else {
			defer os.Remove(tempPath) // Clean up

			// Verify temp file exists and is executable
			info, err := os.Stat(tempPath)
			if err != nil {
				t.Errorf("Temp binary does not exist: %v", err)
			} else if info.Mode()&0111 == 0 {
				t.Error("Temp binary is not executable")
			}

			t.Logf("Successfully extracted binary to temp: %s", tempPath)
		}
	} else {
		t.Logf("Worker binary not available for %s, skipping temp extraction test", currentPlatform.String())
	}
}

func TestIsScriptAvailable(t *testing.T) {
	available := IsScriptAvailable(EntrypointScript)
	t.Logf("Entrypoint script available: %v", available)

	// Test should not fail regardless of availability
}

func TestExtractAllBinariesForPlatform(t *testing.T) {
	tempDir := t.TempDir()

	// Test with Linux AMD64 (most common platform)
	linuxPlatform := Platform{OS: "linux", Arch: "amd64"}

	err := ExtractAllBinariesForPlatform(linuxPlatform, tempDir)
	if err != nil {
		// This might fail if no embedded binaries are available, which is OK for testing
		t.Logf("ExtractAllBinariesForPlatform failed (expected in some build contexts): %v", err)
		return
	}

	// Check that files were extracted
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Errorf("Failed to read temp directory: %v", err)
		return
	}

	t.Logf("Extracted %d files for platform %s", len(files), linuxPlatform.String())
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "autoteam-") {
			t.Errorf("Unexpected file extracted: %s", file.Name())
		}
	}
}
