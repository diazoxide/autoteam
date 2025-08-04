package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"autoteam/internal/config"

	"gopkg.in/yaml.v3"
)

// ComposeConfig represents the structure of a Docker Compose file
type ComposeConfig struct {
	Services map[string]interface{} `yaml:"services"`
}

type Generator struct {
	fileOps *FileOperations
}

func New() *Generator {
	return &Generator{
		fileOps: NewFileOperations(),
	}
}

func (g *Generator) GenerateCompose(cfg *config.Config) error {
	// Ensure .autoteam directory exists
	if err := g.fileOps.EnsureDirectory(config.AutoTeamDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create .autoteam directory: %w", err)
	}

	// Ensure agents directories exist
	if err := g.createAgentDirectories(cfg); err != nil {
		return fmt.Errorf("failed to create agent directories: %w", err)
	}

	// Generate compose.yaml programmatically
	if err := g.generateComposeYAML(cfg); err != nil {
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	// Copy system bin directory
	if err := g.copyBinDirectory(); err != nil {
		return fmt.Errorf("failed to copy bin directory: %w", err)
	}

	return nil
}

// generateComposeYAML creates a Docker Compose YAML file programmatically
func (g *Generator) generateComposeYAML(cfg *config.Config) error {
	compose := ComposeConfig{
		Services: make(map[string]interface{}),
	}

	// Get only enabled agents with their effective settings
	agentsWithSettings := cfg.GetEnabledAgentsWithEffectiveSettings()

	for _, agentWithSettings := range agentsWithSettings {
		agent := agentWithSettings.Agent
		settings := agentWithSettings.EffectiveSettings
		serviceName := agent.GetNormalizedName()

		// Start with the service configuration from settings
		serviceConfig := make(map[string]interface{})

		// Copy all service properties from effective settings
		if settings.Service != nil {
			for key, value := range settings.Service {
				serviceConfig[key] = value
			}
		}

		// Add standard Docker Compose properties that are always needed
		serviceConfig["tty"] = true
		serviceConfig["stdin_open"] = true

		// Build volumes array
		volumes := []string{
			fmt.Sprintf("./agents/%s/codebase:/opt/autoteam/agents/%s/codebase", serviceName, serviceName),
			"./bin:/opt/autoteam/bin",
		}

		// Add any additional volumes from service config
		if existingVolumes, ok := serviceConfig["volumes"]; ok {
			if volumeSlice, ok := existingVolumes.([]string); ok {
				volumes = append(volumes, volumeSlice...)
			} else if volumeInterface, ok := existingVolumes.([]interface{}); ok {
				// Handle case where YAML unmarshals to []interface{}
				for _, v := range volumeInterface {
					if volumeStr, ok := v.(string); ok {
						volumes = append(volumes, volumeStr)
					}
				}
			}
		}
		serviceConfig["volumes"] = volumes

		// Build environment variables
		environment := make(map[string]string)

		// Add standard environment variables
		environment["IS_SANDBOX"] = "1"
		environment["GH_TOKEN"] = agent.GitHubToken
		environment["GH_USER"] = agent.GitHubUser
		environment["REPOSITORIES_INCLUDE"] = strings.Join(cfg.Repositories.Include, ",")
		if len(cfg.Repositories.Exclude) > 0 {
			environment["REPOSITORIES_EXCLUDE"] = strings.Join(cfg.Repositories.Exclude, ",")
		}
		environment["AGENT_NAME"] = agent.Name
		environment["AGENT_NORMALIZED_NAME"] = serviceName
		environment["AGENT_TYPE"] = "claude"
		environment["AGENT_PROMPT"] = agentWithSettings.GetConsolidatedPrompt(cfg)
		environment["TEAM_NAME"] = settings.TeamName
		environment["CHECK_INTERVAL"] = fmt.Sprintf("%d", settings.CheckInterval)
		environment["INSTALL_DEPS"] = fmt.Sprintf("%t", settings.InstallDeps)
		environment["ENTRYPOINT_VERSION"] = "${ENTRYPOINT_VERSION:-latest}"
		environment["MAX_RETRIES"] = "${MAX_RETRIES:-100}"
		environment["DEBUG"] = "${DEBUG:-false}"

		// Auto-inject GitHub MCP server for all agents and add any configured MCP servers
		finalMCPServers := make(map[string]config.MCPServer)

		// Always add GitHub MCP server as default
		finalMCPServers["github"] = config.MCPServer{
			Command: "/opt/autoteam/bin/github-mcp-server",
			Args:    []string{"stdio"},
			Env: map[string]string{
				"GITHUB_PERSONAL_ACCESS_TOKEN": agent.GitHubToken,
			},
		}

		// Add any additional configured MCP servers (can override github if explicitly configured)
		for name, server := range settings.MCPServers {
			finalMCPServers[name] = server
		}

		// Add MCP servers configuration to environment
		if len(finalMCPServers) > 0 {
			mcpServersJSON, err := json.Marshal(finalMCPServers)
			if err != nil {
				return fmt.Errorf("failed to marshal MCP servers for agent %s: %w", agent.Name, err)
			}
			environment["MCP_SERVERS"] = string(mcpServersJSON)
		}

		// Merge with environment from service config
		if existingEnv, ok := serviceConfig["environment"]; ok {
			if envMap, ok := existingEnv.(map[string]string); ok {
				for k, v := range envMap {
					environment[k] = v
				}
			}
		}
		serviceConfig["environment"] = environment

		// Set default entrypoint if not specified
		if _, hasEntrypoint := serviceConfig["entrypoint"]; !hasEntrypoint {
			serviceConfig["entrypoint"] = []string{"/opt/autoteam/bin/entrypoint.sh"}
		}

		compose.Services[serviceName] = serviceConfig
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&compose)
	if err != nil {
		return fmt.Errorf("failed to marshal compose config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(config.ComposeFilePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write compose.yaml file: %w", err)
	}

	return nil
}

func (g *Generator) copyBinDirectory() error {
	// Ensure agents directory exists
	if err := g.fileOps.EnsureDirectory(config.AgentsDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Remove existing directory if it exists
	if err := g.fileOps.RemoveIfExists(config.LocalBinPath); err != nil {
		return fmt.Errorf("failed to remove existing bin directory: %w", err)
	}

	// Check if system bin directory exists
	sourceDir := config.SystemBinDir
	if !g.fileOps.DirectoryExists(config.SystemBinDir) {
		// Fallback: check for old entrypoints directory for backward compatibility
		oldEntrypointsDir := "/opt/autoteam/entrypoints"
		if g.fileOps.DirectoryExists(oldEntrypointsDir) {
			sourceDir = oldEntrypointsDir
		} else {
			// Neither directory exists - create a temporary directory with a helpful message
			if err := g.fileOps.EnsureDirectory(config.LocalBinPath, config.DirPerm); err != nil {
				return fmt.Errorf("failed to create temporary bin directory: %w", err)
			}

			readmePath := filepath.Join(config.LocalBinPath, config.ReadmeFile)
			readmeContent := `# AutoTeam Binary Directory

This directory should contain all AutoTeam binaries including:
- Entrypoint scripts for different platforms
- MCP servers (github-mcp-server, etc.)
- Other runtime binaries

To install the binaries system-wide, run:
` + "```bash" + `
autoteam --install-entrypoints
` + "```" + `

This will:
1. Install all binaries for supported platforms to ` + config.SystemBinDir + `
2. Copy the binaries to this local directory during generation

Supported platforms:
- linux-amd64
- linux-arm64  
- darwin-amd64
- darwin-arm64
`

			if err := g.fileOps.WriteFileIfNotExists(readmePath, []byte(readmeContent), config.ReadmePerm); err != nil {
				return fmt.Errorf("failed to create README file: %w", err)
			}

			return nil
		}
	}

	// Copy system bin directory (or fallback entrypoints directory) to local directory
	return g.fileOps.CopyDirectory(sourceDir, config.LocalBinPath)
}

func (g *Generator) createAgentDirectories(cfg *config.Config) error {
	for _, agent := range cfg.Agents {
		// Skip disabled agents
		if !agent.IsEnabled() {
			continue
		}
		normalizedName := agent.GetNormalizedName()
		if err := g.fileOps.CreateAgentDirectoryStructure(normalizedName); err != nil {
			return fmt.Errorf("failed to create directory structure for agent %s (normalized: %s): %w", agent.Name, normalizedName, err)
		}
	}

	return nil
}
