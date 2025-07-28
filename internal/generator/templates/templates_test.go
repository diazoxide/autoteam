package templates

import (
	"os"
	"strings"
	"testing"
	"text/template"

	"auto-team/internal/config"
)

func TestComposeTemplate(t *testing.T) {
	// Read the actual compose template
	templateContent, err := os.ReadFile("compose.yaml.tmpl")
	if err != nil {
		t.Fatalf("failed to read compose template: %v", err)
	}

	// Parse template
	tmpl, err := template.New("compose").Parse(string(templateContent))
	if err != nil {
		t.Fatalf("failed to parse compose template: %v", err)
	}

	// Test data
	cfg := &config.Config{
		Repository: config.Repository{
			URL:        "owner/test-repo",
			MainBranch: "main",
		},
		Agents: []config.Agent{
			{
				Name:           "dev1",
				Prompt:         "You are a developer",
				GitHubTokenEnv: "DEV1_TOKEN",
				CommonPrompt:   "Follow best practices",
			},
			{
				Name:           "arch1",
				Prompt:         "You are an architect",
				GitHubTokenEnv: "ARCH1_TOKEN",
			},
		},
		Settings: config.Settings{
			DockerImage:   "node:18.17.1",
			DockerUser:    "developer",
			CheckInterval: 60,
			TeamName:      "test-team",
			InstallDeps:   true,
		},
	}

	// Create template data with agents that have effective settings (same as generator)
	templateData := struct {
		*config.Config
		AgentsWithSettings []config.AgentWithSettings
	}{
		Config:             cfg,
		AgentsWithSettings: cfg.GetAllAgentsWithEffectiveSettings(),
	}

	// Execute template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		t.Fatalf("failed to execute compose template: %v", err)
	}

	result := buf.String()

	// Verify generated content contains expected elements
	expectedContent := []string{
		"services:",
		"dev1:",
		"arch1:",
		"image: node:18.17.1",
		"AGENT_NAME: dev1",
		"AGENT_NAME: arch1",
		"AGENT_TYPE: claude",
		"GITHUB_REPO: owner/test-repo",
		"GH_TOKEN: ${DEV1_TOKEN}",
		"GH_TOKEN: ${ARCH1_TOKEN}",
		"TEAM_NAME: test-team",
		"CHECK_INTERVAL: 60",
		"INSTALL_DEPS: true",
		"ENTRYPOINT_VERSION: ${ENTRYPOINT_VERSION:-latest}",
		"MAX_RETRIES: 100",
		"DEBUG: false",
		"/home/developer/test-team/codebase",
		"/home/developer/.claude",
		"--binary autoteam-entrypoint",
		"exec /tmp/autoteam-entrypoint",
		"IS_SANDBOX: 1",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("compose template should contain %q, but it doesn't.\nGenerated content:\n%s", expected, result)
		}
	}

	// Verify structure - should have both agent services
	lines := strings.Split(result, "\n")
	dev1Found := false
	arch1Found := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "dev1:" {
			dev1Found = true
		}
		if line == "arch1:" {
			arch1Found = true
		}
	}

	if !dev1Found {
		t.Errorf("compose template should generate dev1 service")
	}
	if !arch1Found {
		t.Errorf("compose template should generate arch1 service")
	}
}

func TestEntrypointTemplate(t *testing.T) {
	// Read the actual entrypoint template
	templateContent, err := os.ReadFile("entrypoint.sh.tmpl")
	if err != nil {
		t.Fatalf("failed to read entrypoint template: %v", err)
	}

	// Parse template
	tmpl, err := template.New("entrypoint").Parse(string(templateContent))
	if err != nil {
		t.Fatalf("failed to parse entrypoint template: %v", err)
	}

	// Test data - minimal config since entrypoint uses env vars
	cfg := &config.Config{
		Settings: config.Settings{
			CheckInterval: 30,
		},
	}

	// Execute template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, cfg); err != nil {
		t.Fatalf("failed to execute entrypoint template: %v", err)
	}

	result := buf.String()

	// Verify generated script contains expected elements
	expectedContent := []string{
		"#!/bin/bash",
		"if [ \"$INSTALL_DEPS\" == \"true\" ]; then",
		"npm install -g @anthropic-ai/claude-code",
		"gh auth status",
		"git config --global credential.helper store",
		"gh repo clone ${GITHUB_REPO}",
		"gh_my_pending_list()",
		"run_claude()",
		"check_issues_and_prs()",
		"sleep ${CHECK_INTERVAL:-60}",
		"echo \"$(date): Checking for pending items...\"",
		"claude --dangerously-skip-permissions",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("entrypoint template should contain %q, but it doesn't.\nGenerated content:\n%s", expected, result)
		}
	}

	// Verify it starts with shebang
	if !strings.HasPrefix(result, "#!/bin/bash") {
		t.Errorf("entrypoint script should start with #!/bin/bash")
	}

	// Verify it contains the main loop
	if !strings.Contains(result, "while true; do") {
		t.Errorf("entrypoint script should contain main monitoring loop")
	}
}

func TestTemplatesSyntax(t *testing.T) {
	templates := []string{
		"compose.yaml.tmpl",
		"entrypoint.sh.tmpl",
	}

	for _, templateFile := range templates {
		t.Run(templateFile, func(t *testing.T) {
			// Read template
			content, err := os.ReadFile(templateFile)
			if err != nil {
				t.Fatalf("failed to read template %s: %v", templateFile, err)
			}

			// Try to parse template
			_, err = template.New(templateFile).Parse(string(content))
			if err != nil {
				t.Errorf("template %s has syntax errors: %v", templateFile, err)
			}
		})
	}
}

func TestComposeTemplateWithMinimalConfig(t *testing.T) {
	templateContent, err := os.ReadFile("compose.yaml.tmpl")
	if err != nil {
		t.Fatalf("failed to read compose template: %v", err)
	}

	tmpl, err := template.New("compose").Parse(string(templateContent))
	if err != nil {
		t.Fatalf("failed to parse compose template: %v", err)
	}

	// Minimal config with defaults
	cfg := &config.Config{
		Repository: config.Repository{
			URL:        "owner/repo",
			MainBranch: "main",
		},
		Agents: []config.Agent{
			{
				Name:           "single-agent",
				Prompt:         "Test",
				GitHubTokenEnv: "TOKEN",
			},
		},
		Settings: config.Settings{
			DockerImage:   "node:18.17.1",
			DockerUser:    "developer",
			TeamName:      "auto-team",
			CheckInterval: 60,
			InstallDeps:   false,
		},
	}

	// Create template data with agents that have effective settings (same as generator)
	templateData := struct {
		*config.Config
		AgentsWithSettings []config.AgentWithSettings
	}{
		Config:             cfg,
		AgentsWithSettings: cfg.GetAllAgentsWithEffectiveSettings(),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		t.Fatalf("failed to execute compose template with minimal config: %v", err)
	}

	result := buf.String()

	// Should work with single agent
	if !strings.Contains(result, "single-agent:") {
		t.Errorf("should contain single-agent service")
	}

	// Should handle false boolean correctly
	if !strings.Contains(result, "INSTALL_DEPS: false") {
		t.Errorf("should contain INSTALL_DEPS: false")
	}
}

func TestComposeTemplatePromptEscaping(t *testing.T) {
	templateContent, err := os.ReadFile("compose.yaml.tmpl")
	if err != nil {
		t.Fatalf("failed to read compose template: %v", err)
	}

	tmpl, err := template.New("compose").Parse(string(templateContent))
	if err != nil {
		t.Fatalf("failed to parse compose template: %v", err)
	}

	// Config with special characters in prompts
	cfg := &config.Config{
		Repository: config.Repository{URL: "owner/repo"},
		Agents: []config.Agent{
			{
				Name:           "test",
				Prompt:         "You are a \"special\" agent with 'quotes' and $variables",
				GitHubTokenEnv: "TOKEN",
				CommonPrompt:   "Follow \"best practices\" and don't break things",
			},
		},
		Settings: config.Settings{
			DockerImage: "node:test",
			DockerUser:  "test",
			TeamName:    "test",
		},
	}

	// Create template data with agents that have effective settings (same as generator)
	templateData := struct {
		*config.Config
		AgentsWithSettings []config.AgentWithSettings
	}{
		Config:             cfg,
		AgentsWithSettings: cfg.GetAllAgentsWithEffectiveSettings(),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		t.Fatalf("failed to execute template with special characters: %v", err)
	}

	result := buf.String()

	// Should properly escape quotes in YAML
	if !strings.Contains(result, `AGENT_PROMPT: "You are a \"special\" agent with 'quotes' and $variables"`) {
		t.Errorf("should properly escape quotes in agent prompt")
	}

	if !strings.Contains(result, `COMMON_PROMPT: "Follow \"best practices\" and don't break things"`) {
		t.Errorf("should properly escape quotes in common prompt")
	}
}

func TestComposeTemplateWithAgentSpecificSettings(t *testing.T) {
	tmpl, err := template.ParseFiles("compose.yaml.tmpl")
	if err != nil {
		t.Fatalf("failed to parse compose template: %v", err)
	}

	cfg := &config.Config{
		Repository: config.Repository{
			URL: "owner/test-repo",
		},
		Agents: []config.Agent{
			{
				Name:           "dev1",
				Prompt:         "You are a developer",
				GitHubTokenEnv: "DEV1_TOKEN",
				Settings:       nil, // Uses global settings
			},
			{
				Name:           "python-dev",
				Prompt:         "You are a Python developer",
				GitHubTokenEnv: "PYTHON_DEV_TOKEN",
				Settings: &config.AgentSettings{
					DockerImage:   stringPtr("python:3.11"),
					DockerUser:    stringPtr("pythonista"),
					CheckInterval: intPtr(30),
					InstallDeps:   boolPtr(false),
				},
			},
		},
		Settings: config.Settings{
			DockerImage:   "node:18.17.1",
			DockerUser:    "developer",
			CheckInterval: 60,
			TeamName:      "test-team",
			InstallDeps:   true,
		},
	}

	// Create template data with agents that have effective settings (same as generator)
	templateData := struct {
		*config.Config
		AgentsWithSettings []config.AgentWithSettings
	}{
		Config:             cfg,
		AgentsWithSettings: cfg.GetAllAgentsWithEffectiveSettings(),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		t.Fatalf("failed to execute template with agent-specific settings: %v", err)
	}

	result := buf.String()

	// Verify dev1 uses global settings
	if !strings.Contains(result, "image: node:18.17.1") {
		t.Error("dev1 should use global docker image")
	}
	if !strings.Contains(result, "/home/developer/test-team/codebase") {
		t.Error("dev1 should use global docker user and team name")
	}
	if !strings.Contains(result, "CHECK_INTERVAL: 60") {
		t.Error("dev1 should use global check interval")
	}
	if !strings.Contains(result, "INSTALL_DEPS: true") {
		t.Error("dev1 should use global install deps setting")
	}

	// Verify python-dev uses overridden settings
	if !strings.Contains(result, "image: python:3.11") {
		t.Error("python-dev should use overridden docker image")
	}
	if !strings.Contains(result, "/home/pythonista/test-team/codebase") {
		t.Error("python-dev should use overridden docker user")
	}
	if !strings.Contains(result, "CHECK_INTERVAL: 30") {
		t.Error("python-dev should use overridden check interval")
	}
	if !strings.Contains(result, "INSTALL_DEPS: false") {
		t.Error("python-dev should use overridden install deps setting")
	}

	// Verify both agents are present
	if !strings.Contains(result, "dev1:") {
		t.Error("Template should contain dev1 service")
	}
	if !strings.Contains(result, "python-dev:") {
		t.Error("Template should contain python-dev service")
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func TestComposeTemplateWithCustomVolumesAndEntrypoints(t *testing.T) {
	tmpl, err := template.ParseFiles("compose.yaml.tmpl")
	if err != nil {
		t.Fatalf("failed to parse compose template: %v", err)
	}

	cfg := &config.Config{
		Repository: config.Repository{
			URL: "owner/test-repo",
		},
		Agents: []config.Agent{
			{
				Name:           "standard-agent",
				Prompt:         "You are a standard agent",
				GitHubTokenEnv: "STANDARD_TOKEN",
				Settings:       nil, // Uses default entrypoint
			},
			{
				Name:           "custom-agent",
				Prompt:         "You are a custom agent",
				GitHubTokenEnv: "CUSTOM_TOKEN",
				Settings: &config.AgentSettings{
					DockerImage: stringPtr("python:3.11"),
					Volumes: []string{
						"./custom-configs:/app/configs:ro",
						"/var/run/docker.sock:/var/run/docker.sock",
						"./data:/data",
					},
					Entrypoint: stringPtr("/app/custom-entrypoint.sh"),
					Environment: map[string]string{
						"PYTHON_PATH": "/app/custom",
						"DEBUG_MODE":  "true",
						"API_KEY":     "secret-key",
					},
				},
			},
		},
		Settings: config.Settings{
			DockerImage:   "node:18.17.1",
			DockerUser:    "developer",
			CheckInterval: 60,
			TeamName:      "test-team",
			InstallDeps:   true,
			Environment: map[string]string{
				"GLOBAL_VAR": "global_value",
				"DEBUG_MODE": "false", // Should be overridden by agent
			},
		},
	}

	// Create template data with agents that have effective settings
	templateData := struct {
		*config.Config
		AgentsWithSettings []config.AgentWithSettings
	}{
		Config:             cfg,
		AgentsWithSettings: cfg.GetAllAgentsWithEffectiveSettings(),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		t.Fatalf("failed to execute template with custom settings: %v", err)
	}

	result := buf.String()

	// Verify standard-agent uses default entrypoint
	if !strings.Contains(result, "curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/install.sh") {
		t.Error("standard-agent should use default autoteam-entrypoint installation")
	}

	// Verify custom-agent uses custom entrypoint
	if !strings.Contains(result, "entrypoint: /app/custom-entrypoint.sh") {
		t.Error("custom-agent should use custom entrypoint")
	}

	// Verify custom volumes are present
	expectedVolumes := []string{
		"./custom-configs:/app/configs:ro",
		"/var/run/docker.sock:/var/run/docker.sock",
		"./data:/data",
	}
	for _, vol := range expectedVolumes {
		if !strings.Contains(result, "- "+vol) {
			t.Errorf("custom-agent should include volume: %s", vol)
		}
	}

	// Verify custom environment variables
	if !strings.Contains(result, `PYTHON_PATH: "/app/custom"`) {
		t.Error("custom-agent should have PYTHON_PATH environment variable")
	}
	if !strings.Contains(result, `DEBUG_MODE: "true"`) {
		t.Error("custom-agent should have DEBUG_MODE overridden to true")
	}
	if !strings.Contains(result, `API_KEY: "secret-key"`) {
		t.Error("custom-agent should have API_KEY environment variable")
	}
	if !strings.Contains(result, `GLOBAL_VAR: "global_value"`) {
		t.Error("custom-agent should inherit GLOBAL_VAR from global settings")
	}

	// Verify both agents are present
	if !strings.Contains(result, "standard-agent:") {
		t.Error("Template should contain standard-agent service")
	}
	if !strings.Contains(result, "custom-agent:") {
		t.Error("Template should contain custom-agent service")
	}

	// Verify Docker images
	if !strings.Contains(result, "image: node:18.17.1") {
		t.Error("standard-agent should use global docker image")
	}
	if !strings.Contains(result, "image: python:3.11") {
		t.Error("custom-agent should use custom docker image")
	}
}
