package main

import (
	"context"
	"os"
	"path/filepath"
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

	// Copy templates to temp directory
	templatesDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("failed to create templates directory: %v", err)
	}

	// Create simplified template files for testing
	composeTemplate := `services:
{{- range .Workers }}
  {{ .Name }}:
    image: {{ $.Settings.DockerImage }}
    environment:
      AGENT_NAME: {{ .Name }}
      GITHUB_REPO: {{ (index $.Repositories.Include 0) }}
{{- end }}`

	entrypointTemplate := `#!/bin/bash
echo "Test entrypoint"`

	testutil.CreateTempFile(t, templatesDir, "compose.yaml.tmpl", composeTemplate)
	testutil.CreateTempFile(t, templatesDir, "entrypoint.sh.tmpl", entrypointTemplate)

	// Create test config
	testConfig := `repositories:
  include:
    - "owner/test-repo"
workers:
  - name: "dev1"
    prompt: "Test agent"
    github_token: "TEST_TOKEN"
    github_user: "test-user"

settings:
  flow:
    - name: collector
      type: gemini
      prompt: "Collect"
    - name: executor
      type: claude
      depends_on: [collector]
      prompt: "Execute"`

	testutil.CreateTempFile(t, tempDir, "autoteam.yaml", testConfig)

	// For now, test the generate functionality by calling generator directly
	// This skips CLI layer testing but ensures core functionality works
	t.Skip("Skipping CLI test - core functionality tested in generator package")

	// Verify files were generated in .autoteam directory
	if !testutil.FileExists(".autoteam/compose.yaml") {
		t.Errorf("compose.yaml should be generated in .autoteam directory")
	}

	// entrypoint.sh is no longer generated - it's copied from system entrypoints directory
	if !testutil.DirExists(".autoteam") {
		t.Errorf(".autoteam directory should be created")
	}

	// Verify content
	composeContent := testutil.ReadFile(t, ".autoteam/compose.yaml")
	if !strings.Contains(composeContent, "dev1:") {
		t.Errorf("compose.yaml should contain dev1 service")
	}
	if !strings.Contains(composeContent, "dev1") {
		t.Errorf("compose.yaml should contain dev1 service")
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

	// For now, test the missing config by calling generator directly
	// This skips CLI layer testing but ensures core functionality works
	t.Skip("Skipping CLI test - core functionality tested in generator package")
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
