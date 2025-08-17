package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/ports"

	"gopkg.in/yaml.v3"
)

// ComposeConfig represents the structure of a Docker Compose file
type ComposeConfig struct {
	Services map[string]interface{} `yaml:"services"`
	Volumes  map[string]interface{} `yaml:"volumes,omitempty"`
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
	return g.GenerateComposeWithPorts(cfg, nil)
}

func (g *Generator) GenerateComposeWithPorts(cfg *config.Config, portAllocation ports.PortAllocation) error {
	// Ensure .autoteam directory exists
	if err := g.fileOps.EnsureDirectory(config.AutoTeamDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create .autoteam directory: %w", err)
	}

	// Ensure agents directories exist
	if err := g.createAgentDirectories(cfg); err != nil {
		return fmt.Errorf("failed to create agent directories: %w", err)
	}

	// Generate compose.yaml programmatically
	if err := g.generateComposeYAML(cfg, portAllocation); err != nil {
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	// Copy system bin directory
	if err := g.copyBinDirectory(); err != nil {
		return fmt.Errorf("failed to copy bin directory: %w", err)
	}

	return nil
}

// generateComposeYAML creates a Docker Compose YAML file programmatically
func (g *Generator) generateComposeYAML(cfg *config.Config, portAllocation ports.PortAllocation) error {
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
			fmt.Sprintf("./agents/%s:/opt/autoteam/agents/%s", serviceName, serviceName),
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
		environment["AGENT_NAME"] = agent.Name
		environment["AGENT_NORMALIZED_NAME"] = serviceName
		environment["AGENT_TYPE"] = "claude"
		environment["AGENT_PROMPT"] = agentWithSettings.GetConsolidatedPrompt(cfg)
		environment["TEAM_NAME"] = settings.GetTeamName()
		environment["CHECK_INTERVAL"] = fmt.Sprintf("%d", settings.GetCheckInterval())
		environment["INSTALL_DEPS"] = fmt.Sprintf("%t", settings.GetInstallDeps())

		// Two-Layer Agent Architecture Configuration
		if settings.CollectorAgent != nil {
			environment["COLLECTOR_AGENT_TYPE"] = settings.CollectorAgent.Type
			if len(settings.CollectorAgent.Args) > 0 {
				environment["COLLECTOR_AGENT_ARGS"] = strings.Join(settings.CollectorAgent.Args, ",")
			}
			if len(settings.CollectorAgent.Env) > 0 {
				var envPairs []string
				for k, v := range settings.CollectorAgent.Env {
					envPairs = append(envPairs, k+"="+v)
				}
				environment["COLLECTOR_AGENT_ENV"] = strings.Join(envPairs, ",")
			}
			if settings.CollectorAgent.Prompt != nil {
				environment["COLLECTOR_AGENT_PROMPT"] = *settings.CollectorAgent.Prompt
			}
		}
		if settings.ExecutionAgent != nil {
			environment["EXECUTION_AGENT_TYPE"] = settings.ExecutionAgent.Type
			if len(settings.ExecutionAgent.Args) > 0 {
				environment["EXECUTION_AGENT_ARGS"] = strings.Join(settings.ExecutionAgent.Args, ",")
			}
			if len(settings.ExecutionAgent.Env) > 0 {
				var envPairs []string
				for k, v := range settings.ExecutionAgent.Env {
					envPairs = append(envPairs, k+"="+v)
				}
				environment["EXECUTION_AGENT_ENV"] = strings.Join(envPairs, ",")
			}
			if settings.ExecutionAgent.Prompt != nil {
				environment["EXECUTION_AGENT_PROMPT"] = *settings.ExecutionAgent.Prompt
			}
		}
		environment["ENTRYPOINT_VERSION"] = "${ENTRYPOINT_VERSION:-latest}"
		environment["MAX_RETRIES"] = "${MAX_RETRIES:-100}"
		environment["DEBUG"] = "${DEBUG:-false}"

		// Auto-inject GitHub MCP server for all agents and add any configured MCP servers
		finalMCPServers := make(map[string]config.MCPServer)

		// Add configured MCP servers
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
			// Handle both map[string]string and map[string]interface{} cases
			if envMap, ok := existingEnv.(map[string]string); ok {
				for k, v := range envMap {
					environment[k] = v
				}
			} else if envMapInterface, ok := existingEnv.(map[string]interface{}); ok {
				for k, v := range envMapInterface {
					if vStr, ok := v.(string); ok {
						environment[k] = vStr
					}
				}
			}
		}
		serviceConfig["environment"] = environment

		// Set default entrypoint if not specified
		if _, hasEntrypoint := serviceConfig["entrypoint"]; !hasEntrypoint {
			serviceConfig["entrypoint"] = []string{"/opt/autoteam/bin/entrypoint.sh"}
		}

		// Add port mapping if ports are allocated
		if portAllocation != nil {
			if port, hasPort := portAllocation[serviceName]; hasPort {
				// Add port mapping: host:container (8080 is the default container port)
				ports := []string{fmt.Sprintf("%d:8080", port)}

				// Merge with existing ports if any
				if existingPorts, ok := serviceConfig["ports"]; ok {
					if portSlice, ok := existingPorts.([]string); ok {
						ports = append(ports, portSlice...)
					} else if portInterface, ok := existingPorts.([]interface{}); ok {
						for _, p := range portInterface {
							if portStr, ok := p.(string); ok {
								ports = append(ports, portStr)
							}
						}
					}
				}
				serviceConfig["ports"] = ports
			}
		}

		compose.Services[serviceName] = serviceConfig
	}

	// Add custom services from configuration
	if cfg.Services != nil {
		for serviceName, serviceConfig := range cfg.Services {
			// Check for conflicts with agent services
			if _, exists := compose.Services[serviceName]; exists {
				return fmt.Errorf("custom service '%s' conflicts with generated agent service - please choose a different name", serviceName)
			}

			// Add custom service directly to compose
			compose.Services[serviceName] = serviceConfig
		}
	}

	// Auto-detect and create named volume definitions
	namedVolumes := g.detectNamedVolumes(compose.Services)
	if len(namedVolumes) > 0 {
		compose.Volumes = make(map[string]interface{})
		for volumeName := range namedVolumes {
			// Create empty volume definition (external volumes can be customized later)
			compose.Volumes[volumeName] = map[string]interface{}{}
		}
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

// detectNamedVolumes scans all services for named volume references and returns a set of volume names
func (g *Generator) detectNamedVolumes(services map[string]interface{}) map[string]bool {
	namedVolumes := make(map[string]bool)
	// Regex to match named volumes (e.g., "postgres_data:/var/lib/postgresql/data")
	// Named volumes don't start with ./ or / (those are bind mounts)
	namedVolumeRegex := regexp.MustCompile(`^([^/.][^:/]*):`)

	for _, serviceConfig := range services {
		if serviceMap, ok := serviceConfig.(map[string]interface{}); ok {
			if volumes, exists := serviceMap["volumes"]; exists {
				// Handle both []string and []interface{} volume formats
				if volumeSlice, ok := volumes.([]string); ok {
					for _, volume := range volumeSlice {
						if matches := namedVolumeRegex.FindStringSubmatch(volume); matches != nil {
							namedVolumes[matches[1]] = true
						}
					}
				} else if volumeInterface, ok := volumes.([]interface{}); ok {
					for _, v := range volumeInterface {
						if volumeStr, ok := v.(string); ok {
							if matches := namedVolumeRegex.FindStringSubmatch(volumeStr); matches != nil {
								namedVolumes[matches[1]] = true
							}
						}
					}
				}
			}
		}
	}

	return namedVolumes
}
