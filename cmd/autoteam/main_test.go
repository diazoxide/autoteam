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
	// Create a temporary directory for the test
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

	// Create test config for new runtime architecture
	testConfig := `team_name: "test-team"
deployments:
  runtime: "docker"
workers:
  - name: "Test Developer"
    prompt: "Test agent for development"
    enabled: true
    settings:
      service:
        image: "autoteam:latest"`

	testutil.CreateTempFile(t, tempDir, "autoteam.yaml", testConfig)

	// Test the generate command with new CLI structure
	cmd := &cli.Command{}
	cmd.Set("config-file", "autoteam.yaml")
	ctx := context.Background()

	err = generateCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("generateCommand() error = %v", err)
	}

	// Verify team directory structure was created
	if !testutil.DirExists(".autoteam/test-team") {
		t.Errorf(".autoteam/test-team directory should be created")
	}

	// Verify worker directory was created
	if !testutil.DirExists(".autoteam/test-team/workers") {
		t.Errorf("workers directory should be created")
	}

	// Verify bin directory exists
	if !testutil.DirExists("bin") {
		t.Errorf("bin directory should be created")
	}
}

func TestGenerateCommand_MissingConfig(t *testing.T) {
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

	// Test generate command with missing config file
	cmd := &cli.Command{}
	cmd.Set("config-file", "nonexistent.yaml")
	ctx := context.Background()

	err = generateCommand(ctx, cmd)
	if err == nil {
		t.Errorf("generateCommand() should fail with missing config file")
	}
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

	// Verify content contains expected sample data
	content := testutil.ReadFile(t, "autoteam.yaml")
	if !strings.Contains(content, "workers:") {
		t.Errorf("autoteam.yaml should contain workers section")
	}
	if !strings.Contains(content, "dev1") {
		t.Errorf("autoteam.yaml should contain dev1 worker")
	}
	if !strings.Contains(content, "arch1") {
		t.Errorf("autoteam.yaml should contain arch1 agent")
	}
}
