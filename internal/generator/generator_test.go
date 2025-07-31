package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"autoteam/internal/config"
	"autoteam/internal/testutil"
)

func TestGenerator_GenerateCompose(t *testing.T) {
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

	// Create template files
	composeTemplate := `services:
{{- range .Agents }}
  {{ .Name }}:
    image: {{ $.Settings.DockerImage }}
    environment:
      AGENT_NAME: {{ .Name }}
      GITHUB_REPO: {{ (index $.Repositories.Include 0) }}
      GH_TOKEN: {{ .GitHubToken }}
{{- end }}`

	entrypointTemplate := `#!/bin/bash
echo "Repository: $GITHUB_REPO"
echo "Agent: $AGENT_NAME"
echo "Check interval: ${CHECK_INTERVAL:-60}"`

	testutil.CreateTempFile(t, templatesDir, "compose.yaml.tmpl", composeTemplate)
	testutil.CreateTempFile(t, templatesDir, "entrypoint.sh.tmpl", entrypointTemplate)

	// Create test config inline
	cfg := &config.Config{
		Repositories: config.Repositories{
			Include: []string{"owner/test-repo"},
		},
		Agents: []config.Agent{
			{
				Name:        "dev1",
				Prompt:      "You are a developer agent",
				GitHubToken: "DEV1_TOKEN",
				GitHubUser:  "dev-user",
			},
			{
				Name:        "arch1",
				Prompt:      "You are an architect agent",
				GitHubToken: "ARCH1_TOKEN",
				GitHubUser:  "arch-user",
			},
		},
		Settings: config.Settings{
			DockerImage:   "node:18",
			DockerUser:    "testuser",
			CheckInterval: 30,
			TeamName:      "test-team",
			InstallDeps:   false,
		},
	}

	// Generate files
	gen := New()
	if err := gen.GenerateCompose(cfg); err != nil {
		t.Fatalf("GenerateCompose() error = %v", err)
	}

	// Verify compose.yaml was generated in .autoteam directory
	composeContent := testutil.ReadFile(t, ".autoteam/compose.yaml")

	// Check that both agents are in the compose file
	if !strings.Contains(composeContent, "dev1:") {
		t.Errorf("compose.yaml should contain dev1 service")
	}
	if !strings.Contains(composeContent, "arch1:") {
		t.Errorf("compose.yaml should contain arch1 service")
	}
	if !strings.Contains(composeContent, "node:18") {
		t.Errorf("compose.yaml should contain docker image")
	}
	if !strings.Contains(composeContent, "owner/test-repo") {
		t.Errorf("compose.yaml should contain repository URL")
	}

	// Verify .autoteam/entrypoints directory was created (entrypoint.sh is no longer generated)
	if !testutil.DirExists(".autoteam/entrypoints") {
		t.Errorf(".autoteam/entrypoints directory should be created")
	}

	// Verify agent directories were created
	agentDirs := []string{
		".autoteam/agents/dev1/codebase",
		".autoteam/agents/arch1/codebase",
	}

	for _, dir := range agentDirs {
		if !testutil.FileExists(dir) {
			t.Errorf("directory %s should be created", dir)
		}
	}
}

func TestGenerator_CreateAgentDirectories(t *testing.T) {
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

	cfg := &config.Config{
		Agents: []config.Agent{
			{Name: "test1"},
			{Name: "test2"},
		},
	}

	gen := New()
	if err := gen.createAgentDirectories(cfg); err != nil {
		t.Fatalf("createAgentDirectories() error = %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		".autoteam/agents/test1/codebase",
		".autoteam/agents/test2/codebase",
	}

	for _, dir := range expectedDirs {
		if !testutil.FileExists(dir) {
			t.Errorf("directory %s should be created", dir)
		}
	}
}

func TestGenerator_GenerateFile(t *testing.T) {
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

	// Create templates directory and template file
	templatesDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("failed to create templates directory: %v", err)
	}

	templateContent := `Repository: {{ (index .Repositories.Include 0) }}
Team: {{ .Settings.TeamName }}
{{- range .Agents }}
Agent: {{ .Name }} - {{ .Prompt }}
{{- end }}`

	testutil.CreateTempFile(t, templatesDir, "test.tmpl", templateContent)

	cfg := &config.Config{
		Repositories: config.Repositories{Include: []string{"owner/repo"}},
		Settings:     config.Settings{TeamName: "test-team"},
		Agents: []config.Agent{
			{Name: "dev1", Prompt: "Developer", GitHubToken: "DEV_TOKEN", GitHubUser: "dev-user"},
			{Name: "arch1", Prompt: "Architect", GitHubToken: "ARCH_TOKEN", GitHubUser: "arch-user"},
		},
	}

	gen := New()
	if err := gen.generateFile("test.tmpl", "output.txt", cfg); err != nil {
		t.Fatalf("generateFile() error = %v", err)
	}

	// Verify output file was created
	if !testutil.FileExists("output.txt") {
		t.Fatalf("output file should be created")
	}

	content := testutil.ReadFile(t, "output.txt")

	// Verify template was executed correctly
	expectedContent := `Repository: owner/repo
Team: test-team
Agent: dev1 - Developer
Agent: arch1 - Architect`

	if strings.TrimSpace(content) != expectedContent {
		t.Errorf("output content mismatch\ngot:\n%s\nwant:\n%s", content, expectedContent)
	}
}

func TestGenerator_GenerateFile_TemplateError(t *testing.T) {
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

	cfg := &config.Config{}
	gen := New()

	// Test non-existent template
	err = gen.generateFile("nonexistent.tmpl", "output.txt", cfg)
	if err == nil {
		t.Errorf("generateFile() should fail with non-existent template")
	}
	if !strings.Contains(err.Error(), "failed to read embedded template") {
		t.Errorf("error should mention template reading, got: %v", err)
	}
}
