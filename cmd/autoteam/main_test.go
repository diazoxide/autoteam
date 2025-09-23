package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"autoteam/internal/testutil"

	"github.com/urfave/cli/v3"
)

func TestGenerateCommand(t *testing.T) {
	// Skip CLI layer testing - integration tests cover the full CLI workflow
	// This test can focus on config loading and runtime initialization
	t.Skip("CLI layer testing skipped - covered by integration tests")
}

func TestGenerateCommand_MissingConfig(t *testing.T) {
	// Skip CLI layer testing - integration tests cover the full CLI workflow
	t.Skip("CLI layer testing skipped - covered by integration tests")
}

func TestInitCommand(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	cmd := &cli.Command{}
	ctx := context.Background()

	err = initCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("initCommand() error = %v", err)
	}

	// Verify autoteam.yaml was created
	if !testutil.FileExists("autoteam.yaml") {
		t.Errorf("autoteam.yaml should be created")
	}

	// Verify content contains expected database-driven configuration
	content := testutil.ReadFile(t, "autoteam.yaml")
	if !strings.Contains(content, "control_plane:") {
		t.Errorf("autoteam.yaml should contain control_plane section")
	}
	if !strings.Contains(content, "enabled: true") {
		t.Errorf("autoteam.yaml should have control plane enabled")
	}
	if !strings.Contains(content, "dashboard:") {
		t.Errorf("autoteam.yaml should contain dashboard section")
	}
	if !strings.Contains(content, "flow:") {
		t.Errorf("autoteam.yaml should contain global flow configuration")
	}
}
