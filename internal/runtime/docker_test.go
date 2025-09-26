package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"autoteam/internal/config"
	"autoteam/internal/logger"
)

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (src, dst string)
		expectError bool
	}{
		{
			name: "successful file copy",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				src := filepath.Join(tempDir, "source.txt")
				dst := filepath.Join(tempDir, "dest.txt")

				content := "test file content"
				if err := os.WriteFile(src, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				return src, dst
			},
			expectError: false,
		},
		{
			name: "source file does not exist",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				src := filepath.Join(tempDir, "nonexistent.txt")
				dst := filepath.Join(tempDir, "dest.txt")
				return src, dst
			},
			expectError: true,
		},
		{
			name: "destination directory does not exist",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				src := filepath.Join(tempDir, "source.txt")
				dst := filepath.Join(tempDir, "nonexistent", "dest.txt")

				content := "test file content"
				if err := os.WriteFile(src, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				return src, dst
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup(t)

			err := copyFile(src, dst)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file was copied correctly
			srcContent, err := os.ReadFile(src)
			if err != nil {
				t.Fatalf("Failed to read source file: %v", err)
			}

			dstContent, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("Failed to read destination file: %v", err)
			}

			if string(srcContent) != string(dstContent) {
				t.Errorf("File content mismatch. Source: %s, Dest: %s",
					string(srcContent), string(dstContent))
			}
		})
	}
}

func TestEnsureBinaries_WithBuildDirectory(t *testing.T) {
	// Setup test environment
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	// Create build directory with test binaries
	buildDir := "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}

	// Create test binaries for Linux platforms
	testBinaries := []string{
		"autoteam-worker-linux-amd64",
		"autoteam-control-plane-linux-amd64",
		"autoteam-dashboard-linux-amd64",
		"autoteam-worker-linux-arm64",
		"autoteam-control-plane-linux-arm64",
		"autoteam-dashboard-linux-arm64",
	}

	for _, binary := range testBinaries {
		binaryPath := filepath.Join(buildDir, binary)
		content := fmt.Sprintf("fake binary content for %s", binary)
		if err := os.WriteFile(binaryPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to create test binary %s: %v", binary, err)
		}
	}

	// Create entrypoint script
	scriptsDir := "scripts"
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("Failed to create scripts directory: %v", err)
	}

	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")
	entrypointContent := "#!/bin/sh\necho 'test entrypoint'"
	if err := os.WriteFile(entrypointPath, []byte(entrypointContent), 0755); err != nil {
		t.Fatalf("Failed to create entrypoint script: %v", err)
	}

	// Create Docker runtime and test ensureBinaries
	runtime := &DockerRuntime{}
	ctx, err := logger.SetupContext(context.Background(), logger.InfoLevel)
	if err != nil {
		t.Fatalf("Failed to setup logger context: %v", err)
	}

	err = runtime.ensureBinaries(ctx)
	if err != nil {
		t.Fatalf("ensureBinaries failed: %v", err)
	}

	// Verify .autoteam/bin directory was created
	binDir := ".autoteam/bin"
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		t.Error(".autoteam/bin directory was not created")
	}

	// Verify platform-specific binaries were extracted
	for _, binary := range testBinaries {
		binaryPath := filepath.Join(binDir, binary)
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Errorf("Platform-specific binary %s was not extracted", binary)
		}
	}

	// Verify generic binaries were created
	genericBinaries := []string{
		"autoteam-worker",
		"autoteam-control-plane",
		"autoteam-dashboard",
	}

	for _, binary := range genericBinaries {
		binaryPath := filepath.Join(binDir, binary)
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Errorf("Generic binary %s was not created", binary)
		}

		// Verify it's executable
		info, err := os.Stat(binaryPath)
		if err != nil {
			t.Errorf("Failed to stat generic binary %s: %v", binary, err)
		} else if info.Mode()&0111 == 0 {
			t.Errorf("Generic binary %s is not executable", binary)
		}
	}

	// Verify entrypoint script was extracted
	entrypointDest := filepath.Join(binDir, "entrypoint.sh")
	if _, err := os.Stat(entrypointDest); os.IsNotExist(err) {
		t.Error("Entrypoint script was not extracted")
	}
}

func TestEnsureBinaries_NoEmbeddedNoBuild(t *testing.T) {
	// Setup test environment with no build directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	// Create Docker runtime and test ensureBinaries
	runtime := &DockerRuntime{}
	ctx, err := logger.SetupContext(context.Background(), logger.InfoLevel)
	if err != nil {
		t.Fatalf("Failed to setup logger context: %v", err)
	}

	err = runtime.ensureBinaries(ctx)

	// Should return error when no embedded binaries and no build directory
	if err == nil {
		t.Error("Expected error when no embedded binaries and no build directory")
	}

	// Error message should contain helpful solutions
	errMsg := err.Error()
	expectedMessages := []string{
		"no binaries found in build/ directory",
		"make build-worker-all",
		"make build-embedded",
	}

	for _, msg := range expectedMessages {
		if !contains(errMsg, msg) {
			t.Errorf("Error message should contain '%s', got: %s", msg, errMsg)
		}
	}
}

func TestEnsureBinaries_ExistingBinaries(t *testing.T) {
	// Setup test environment
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	// Create .autoteam/bin directory with existing binaries
	binDir := ".autoteam/bin"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create existing worker binary
	existingBinary := filepath.Join(binDir, "autoteam-worker-linux-amd64")
	if err := os.WriteFile(existingBinary, []byte("existing binary"), 0755); err != nil {
		t.Fatalf("Failed to create existing binary: %v", err)
	}

	// Create build directory with some test binaries to simulate real environment
	buildDir := "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}

	// Create a test binary in build directory
	buildBinary := filepath.Join(buildDir, "autoteam-worker-linux-amd64")
	if err := os.WriteFile(buildBinary, []byte("new binary from build"), 0755); err != nil {
		t.Fatalf("Failed to create build binary: %v", err)
	}

	// Create Docker runtime and test ensureBinaries
	runtime := &DockerRuntime{}
	ctx, err := logger.SetupContext(context.Background(), logger.InfoLevel)
	if err != nil {
		t.Fatalf("Failed to setup logger context: %v", err)
	}

	err = runtime.ensureBinaries(ctx)
	if err != nil {
		t.Fatalf("ensureBinaries failed with existing binaries: %v", err)
	}

	// Verify binary still exists and was updated with newer content
	if _, err := os.Stat(existingBinary); os.IsNotExist(err) {
		t.Error("Binary was unexpectedly removed")
	}

	// Verify binary was updated with content from build directory
	content, err := os.ReadFile(existingBinary)
	if err != nil {
		t.Fatalf("Failed to read updated binary: %v", err)
	}

	if string(content) != "new binary from build" {
		t.Errorf("Binary was not updated. Expected 'new binary from build', got '%s'", string(content))
	}
}

func TestDockerRuntime_GetHostWorkingDirectory(t *testing.T) {
	runtime := &DockerRuntime{}

	// Test with default working directory
	dir := runtime.getHostWorkingDirectory()
	if dir == "" {
		t.Error("Host working directory should not be empty")
	}

	// Should return current directory or configured directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	if dir != wd {
		t.Logf("Host working directory (%s) differs from current directory (%s), which is acceptable", dir, wd)
	}
}

func TestDockerRuntime_Integration(t *testing.T) {
	// This test verifies the Docker runtime can be created and basic operations work
	cfg := &config.Config{
		Deployments: &config.DeploymentConfig{
			Runtime: "docker",
		},
	}

	runtime := &DockerRuntime{}

	// Test that we can create container configs (without actually creating containers)
	if cfg.ControlPlane == nil {
		cfg.ControlPlane = &config.ControlPlaneConfig{
			Enabled: true,
			Port:    9090,
		}
	}

	containerConfig := runtime.buildControlPlaneContainerConfig(cfg)

	if containerConfig == nil {
		t.Error("Container config should not be nil")
	}

	if containerConfig.Entrypoint[0] != "/opt/autoteam/bin/autoteam-control-plane" {
		t.Errorf("Expected entrypoint to use generic binary name, got: %s", containerConfig.Entrypoint[0])
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
