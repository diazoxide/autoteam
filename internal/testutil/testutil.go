package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "autoteam-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// CreateTempFile creates a temporary file with content for testing
func CreateTempFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	filepath := filepath.Join(dir, filename)
	if err := os.WriteFile(filepath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create temp file %s: %v", filepath, err)
	}

	return filepath
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ReadFile reads file content for testing
func ReadFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}

	return string(content)
}
