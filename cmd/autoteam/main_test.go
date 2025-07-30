package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"autoteam/internal/config"
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
{{- range .Agents }}
  {{ .Name }}:
    image: {{ $.Settings.DockerImage }}
    environment:
      AGENT_NAME: {{ .Name }}
      GITHUB_REPO: {{ $.Repository.URL }}
{{- end }}`

	entrypointTemplate := `#!/bin/bash
echo "Test entrypoint"`

	testutil.CreateTempFile(t, templatesDir, "compose.yaml.tmpl", composeTemplate)
	testutil.CreateTempFile(t, templatesDir, "entrypoint.sh.tmpl", entrypointTemplate)

	// Create test config
	testConfig := `repository:
  url: "owner/test-repo"
agents:
  - name: "dev1"
    prompt: "Test agent"
    github_token: "TEST_TOKEN"`

	testutil.CreateTempFile(t, tempDir, "autoteam.yaml", testConfig)

	// Create a mock CLI command
	cmd := &cli.Command{}
	ctx := context.Background()

	// Since we're not using the CLI framework here, we need to load config manually for this test
	cfg, err := config.LoadConfig("autoteam.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	ctx = context.WithValue(ctx, configContextKey, cfg)

	// Test generate command
	err = generateCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("generateCommand() error = %v", err)
	}

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
	if !strings.Contains(composeContent, "owner/test-repo") {
		t.Errorf("compose.yaml should contain repository URL")
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

	// Don't create autoteam.yaml
	cmd := &cli.Command{}
	ctx := context.Background()

	err = generateCommand(ctx, cmd)
	if err == nil {
		t.Errorf("generateCommand() should fail when config is not available")
	}

	if !strings.Contains(err.Error(), "config not available in context") {
		t.Errorf("error should mention config not available, got: %v", err)
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
	if !strings.Contains(content, "repository:") {
		t.Errorf("autoteam.yaml should contain repository section")
	}
	if !strings.Contains(content, "agents:") {
		t.Errorf("autoteam.yaml should contain agents section")
	}
	if !strings.Contains(content, "dev1") {
		t.Errorf("autoteam.yaml should contain dev1 agent")
	}
	if !strings.Contains(content, "arch1") {
		t.Errorf("autoteam.yaml should contain arch1 agent")
	}
}

func TestRunDockerCompose(t *testing.T) {
	// Test with a command that should always be available
	ctx := context.Background()
	err := runDockerCompose(ctx, "--version")

	// We expect this to either succeed (if docker-compose is installed)
	// or fail with a specific error (if docker-compose is not found)
	// But it should not panic
	if err != nil {
		// Check that it's a reasonable error (command not found, etc.)
		if !strings.Contains(err.Error(), "docker-compose") {
			t.Logf("docker-compose not available for testing: %v", err)
		}
	}
}

// Integration test that simulates the full CLI workflow
func TestCLIIntegration(t *testing.T) {
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

	// Create template files
	composeTemplate := `services:
{{- range .Agents }}
  {{ .Name }}:
    image: {{ $.Settings.DockerImage }}
{{- end }}`

	entrypointTemplate := `#!/bin/bash
echo "Integration test"`

	testutil.CreateTempFile(t, templatesDir, "compose.yaml.tmpl", composeTemplate)
	testutil.CreateTempFile(t, templatesDir, "entrypoint.sh.tmpl", entrypointTemplate)

	ctx := context.Background()
	cmd := &cli.Command{}

	// Step 1: Initialize config
	err = initCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("initCommand() error = %v", err)
	}

	// Step 2: Load config after init created the file
	cfg, err := config.LoadConfig("autoteam.yaml")
	if err != nil {
		t.Fatalf("failed to load config after init: %v", err)
	}
	ctx = context.WithValue(ctx, configContextKey, cfg)

	// Step 3: Generate compose files
	err = generateCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("generateCommand() error = %v", err)
	}

	// Verify all expected files exist
	expectedFiles := []string{
		"autoteam.yaml",
		".autoteam/compose.yaml",
		".autoteam/agents/dev1/codebase",
		".autoteam/agents/arch1/codebase",
	}

	for _, file := range expectedFiles {
		if !testutil.FileExists(file) {
			t.Errorf("file/directory %s should exist after CLI workflow", file)
		}
	}

	// Verify .autoteam/entrypoints directory exists (even if empty due to no system installation)
	if !testutil.DirExists(".autoteam/entrypoints") {
		t.Errorf(".autoteam/entrypoints directory should be created")
	}
}
