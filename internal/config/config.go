package config

import (
	"fmt"
	"os"

	"autoteam/internal/util"
	"autoteam/internal/worker"
	"gopkg.in/yaml.v3"
)

// Default configuration constants
const (
	DefaultTeamName = "autoteam"
)

type Config struct {
	Workers    []worker.Worker                   `yaml:"workers"`
	Services   map[string]map[string]interface{} `yaml:"services,omitempty"`
	Settings   worker.WorkerSettings             `yaml:"settings"`
	MCPServers map[string]worker.MCPServer       `yaml:"mcp_servers,omitempty"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	setDefaults(&config)

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Workers) == 0 {
		return fmt.Errorf("at least one worker must be configured")
	}

	// Count enabled workers
	enabledCount := 0
	for _, worker := range config.Workers {
		if worker.IsEnabled() {
			enabledCount++
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("at least one worker must be enabled")
	}

	for i, worker := range config.Workers {
		if worker.Name == "" {
			return fmt.Errorf("worker[%d].name is required", i)
		}
		// Only validate required fields for enabled workers
		if worker.IsEnabled() {
			if worker.Prompt == "" {
				return fmt.Errorf("worker[%d].prompt is required for enabled workers", i)
			}

			// Get effective settings to check flow configuration
			settings := worker.GetEffectiveSettings(config.Settings)
			if len(settings.Flow) == 0 {
				return fmt.Errorf("worker[%d].flow is required for enabled workers", i)
			}

			// Validate flow steps
			if err := validateFlow(settings.Flow); err != nil {
				return fmt.Errorf("worker[%d].flow validation failed: %w", i, err)
			}
		}
	}

	return nil
}

// validateFlow validates flow configuration
func validateFlow(flow []worker.FlowStep) error {
	if len(flow) == 0 {
		return fmt.Errorf("flow must contain at least one step")
	}

	stepNames := make(map[string]bool)
	for i, step := range flow {
		if step.Name == "" {
			return fmt.Errorf("step[%d].name is required", i)
		}
		if step.Type == "" {
			return fmt.Errorf("step[%d].type is required", i)
		}
		if stepNames[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		stepNames[step.Name] = true

		// Validate dependencies exist
		for _, dep := range step.DependsOn {
			found := false
			for _, otherStep := range flow {
				if otherStep.Name == dep {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("step %s depends on non-existent step: %s", step.Name, dep)
			}
		}
	}

	return nil
}

func setDefaults(config *Config) {
	if config.Settings.SleepDuration == nil {
		config.Settings.SleepDuration = util.IntPtr(60)
	}
	if config.Settings.TeamName == nil {
		config.Settings.TeamName = util.StringPtr(DefaultTeamName)
	}
	if config.Settings.MaxAttempts == nil {
		config.Settings.MaxAttempts = util.IntPtr(3)
	}
	// Set default service configuration if not provided
	if config.Settings.Service == nil {
		config.Settings.Service = map[string]interface{}{
			"image": "node:18.17.1",
			"user":  "developer",
		}
	}
}

func CreateSampleConfig(filename string) error {
	sampleConfig := Config{
		Workers: []worker.Worker{
			{
				Name:   "dev1",
				Prompt: "You are a developer worker responsible for implementing features and fixing bugs.",
			},
			{
				Name:   "arch1",
				Prompt: "You are an architecture worker responsible for system design and code reviews.",
				Settings: &worker.WorkerSettings{
					SleepDuration: util.IntPtr(30),
					Service: map[string]interface{}{
						"image": "python:3.11",
						"volumes": []string{
							"./custom-configs:/app/configs:ro",
							"/var/run/docker.sock:/var/run/docker.sock",
						},
						"environment": map[string]string{
							"PYTHON_PATH": "/app/custom",
							"DEBUG_MODE":  "true",
						},
					},
					Hooks: &worker.HookConfig{
						OnInit: []worker.HookCommand{
							{
								Command:     "/bin/sh",
								Args:        []string{"-c", "echo 'Agent initializing: $AGENT_NAME'"},
								Description: util.StringPtr("Log worker initialization"),
							},
						},
						OnStart: []worker.HookCommand{
							{
								Command:     "/bin/bash",
								Args:        []string{"-c", "pip install --upgrade pip && pip install requests"},
								Timeout:     util.IntPtr(60),
								ContinueOn:  util.StringPtr("always"),
								Description: util.StringPtr("Install additional Python packages"),
							},
						},
						OnStop: []worker.HookCommand{
							{
								Command:     "/bin/sh",
								Args:        []string{"-c", "echo 'Agent $AGENT_NAME shutting down gracefully'"},
								Description: util.StringPtr("Log graceful shutdown"),
							},
						},
					},
				},
			},
			{
				Name:    "devops1",
				Prompt:  "You are a DevOps worker responsible for CI/CD and infrastructure.",
				Enabled: util.BoolPtr(false), // This worker is disabled
			},
		},
		Services: map[string]map[string]interface{}{
			"postgres": {
				"image": "postgres:15",
				"environment": map[string]string{
					"POSTGRES_DB":       "autoteam_dev",
					"POSTGRES_USER":     "autoteam",
					"POSTGRES_PASSWORD": "development_password",
				},
				"ports": []string{"5432:5432"},
				"volumes": []string{
					"postgres_data:/var/lib/postgresql/data",
					"./sql/init.sql:/docker-entrypoint-initdb.d/init.sql:ro",
				},
			},
			"redis": {
				"image":   "redis:7",
				"ports":   []string{"6379:6379"},
				"volumes": []string{"redis_data:/data"},
			},
		},
		Settings: worker.WorkerSettings{
			SleepDuration: util.IntPtr(60),
			TeamName:      util.StringPtr(DefaultTeamName),
			InstallDeps:   util.BoolPtr(true),
			CommonPrompt:  util.StringPtr("Always follow coding best practices and write comprehensive tests."),
			MaxAttempts:   util.IntPtr(3),
			Service: map[string]interface{}{
				"image": "node:18.17.1",
				"user":  "developer",
			},
			Flow: []worker.FlowStep{
				{
					Name:   "collector",
					Type:   "gemini",
					Args:   []string{"--model", "gemini-2.5-flash"},
					Input:  "You are a notification collector. Get unread GitHub notifications and list them.\nUse GitHub MCP to get unread notifications.\nCRITICAL: Mark all notifications as read after collecting them.",
					Output: "{{ .stdout | trim }}",
				},
				{
					Name:      "analyzer",
					Type:      "claude",
					DependsOn: []string{"collector"},
					Input:     "{{ index .inputs 0 }}\n\nYou are the GitHub Notification Handler. Process GitHub notifications exactly like a human would.\n\nFor each notification:\n1. Read the full context (issues, PRs, comments, code)\n2. Respond naturally as a project contributor\n3. Take appropriate action (comment, review, create PR, etc.)\n4. Use GitHub MCP to publish your responses\n\nAlways be professional, helpful, and maintain high quality standards.",
				},
			},
		},
		MCPServers: map[string]worker.MCPServer{
			"memory": {
				Command: "npx",
				Args:    []string{"-y", "mcp-memory-service"},
			},
		},
	}

	data, err := yaml.Marshal(&sampleConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal sample config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}

// GetAllWorkersWithEffectiveSettings returns a slice of workers with their effective settings
func (c *Config) GetAllWorkersWithEffectiveSettings() []worker.WorkerWithSettings {
	var workers []worker.WorkerWithSettings
	for _, w := range c.Workers {
		workers = append(workers, worker.WorkerWithSettings{
			Worker:   w,
			Settings: w.GetEffectiveSettings(c.Settings),
		})
	}
	return workers
}

// GetEnabledWorkersWithEffectiveSettings returns only enabled workers with their effective settings
func (c *Config) GetEnabledWorkersWithEffectiveSettings() []worker.WorkerWithSettings {
	var workers []worker.WorkerWithSettings
	for _, w := range c.Workers {
		if w.IsEnabled() {
			workers = append(workers, worker.WorkerWithSettings{
				Worker:   w,
				Settings: w.GetEffectiveSettings(c.Settings),
			})
		}
	}
	return workers
}
