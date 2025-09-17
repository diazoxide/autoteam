package config

import (
	"fmt"
	"os"

	"autoteam/internal/worker"

	"gopkg.in/yaml.v3"
)

// Default configuration constants
const (
	DefaultTeamName = "autoteam"
)

type Config struct {
	Services     map[string]map[string]interface{} `yaml:"services,omitempty"`
	Settings     worker.WorkerSettings             `yaml:"settings"`
	MCPServers   map[string]worker.MCPServer       `yaml:"mcp_servers,omitempty"`
	ControlPlane *ControlPlaneConfig               `yaml:"control_plane,omitempty"`
	Dashboard    *DashboardConfig                  `yaml:"dashboard,omitempty"`
	Deployments  *DeploymentConfig                 `yaml:"deployments,omitempty"`
}

// ControlPlaneConfig represents the control plane configuration
type ControlPlaneConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	APIKey  string `yaml:"api_key,omitempty"`
}

// DashboardConfig represents the dashboard configuration
type DashboardConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	APIUrl  string `yaml:"api_url,omitempty"`
	Title   string `yaml:"title,omitempty"`
}

// DeploymentConfig represents the deployment runtime configuration
type DeploymentConfig struct {
	Runtime string                 `yaml:"runtime"`
	Config  map[string]interface{} `yaml:"config,omitempty"`
}

// DockerConfig represents Docker-specific deployment configuration
type DockerConfig struct {
	NetworkName    string            `yaml:"network_name,omitempty"`
	BuildArgs      map[string]string `yaml:"build_args,omitempty"`
	RegistryConfig *RegistryConfig   `yaml:"registry,omitempty"`
	ResourceLimits *ResourceLimits   `yaml:"resource_limits,omitempty"`
	EnableBuildx   bool              `yaml:"enable_buildx,omitempty"`
	DockerfilePath string            `yaml:"dockerfile_path,omitempty"`
}

// RegistryConfig represents Docker registry configuration
type RegistryConfig struct {
	URL      string `yaml:"url,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// ResourceLimits represents container resource limits
type ResourceLimits struct {
	Memory string `yaml:"memory,omitempty"`
	CPUs   string `yaml:"cpus,omitempty"`
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
	// Workers now come exclusively from database, no config validation needed
	// Database configuration is handled separately by control plane
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
	if config.Settings.SleepDuration == 0 {
		config.Settings.SleepDuration = 60
	}
	if config.Settings.TeamName == "" {
		config.Settings.TeamName = DefaultTeamName
	}
	if config.Settings.MaxAttempts == 0 {
		config.Settings.MaxAttempts = 3
	}
	// Set default service configuration if not provided
	if config.Settings.Service == nil {
		config.Settings.Service = map[string]interface{}{
			"image": "node:18.17.1",
			"user":  "developer",
		}
	}

	// Set control plane defaults if enabled
	if config.ControlPlane != nil && config.ControlPlane.Enabled {
		if config.ControlPlane.Port == 0 {
			config.ControlPlane.Port = 9090
		}
	}

	// Set deployment defaults
	if config.Deployments == nil {
		config.Deployments = &DeploymentConfig{
			Runtime: "docker",
			Config: map[string]interface{}{
				"network_name": fmt.Sprintf("%s-network", config.GetTeamName()),
			},
		}
	}
	if config.Deployments.Runtime == "" {
		config.Deployments.Runtime = "docker"
	}
}

func CreateSampleConfig(filename string) error {
	sampleConfig := Config{
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
			SleepDuration: 60,
			TeamName:      DefaultTeamName,
			InstallDeps:   true,
			CommonPrompt:  "Always follow coding best practices and write comprehensive tests.",
			MaxAttempts:   3,
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
		ControlPlane: &ControlPlaneConfig{
			Enabled: true, // Enable control plane for database workers
			Port:    9090,
		},
		Dashboard: &DashboardConfig{
			Enabled: true,
			Port:    8081,
			APIUrl:  "http://localhost:9090",
			Title:   "AutoTeam Dashboard",
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

// GetTeamName returns the team name from settings, or default if not set
func (c *Config) GetTeamName() string {
	if c.Settings.TeamName != "" {
		return c.Settings.TeamName
	}
	return DefaultTeamName
}

// GetWorkersDir returns the team-specific workers directory path
func (c *Config) GetWorkersDir() string {
	return fmt.Sprintf(WorkersBaseDir, c.GetTeamName())
}

// GetControlPlaneDir returns the team-specific control-plane directory path
func (c *Config) GetControlPlaneDir() string {
	return fmt.Sprintf(ControlPlaneBaseDir, c.GetTeamName())
}

// GetControlPlaneConfigPath returns the team-specific control-plane config file path
func (c *Config) GetControlPlaneConfigPath() string {
	return fmt.Sprintf("%s/config.yaml", c.GetControlPlaneDir())
}
