package generator

import (
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

// normalizeEnvironmentValue replaces AutoTeam placeholder variables with actual runtime values.
// Supported placeholders:
//   - ${AUTOTEAM_AGENT_NAME} -> actual agent name (e.g., "Senior Developer")
//   - ${AUTOTEAM_WORKER_DIR}  -> agent directory path (e.g., "/opt/autoteam/workers/senior_developer")
//   - ${AUTOTEAM_AGENT_NORMALIZED_NAME} -> normalized agent name (e.g., "senior_developer")
func (g *Generator) normalizeEnvironmentValue(value string, worker config.Worker) string {
	value = strings.ReplaceAll(value, "${AUTOTEAM_AGENT_NAME}", worker.Name)
	value = strings.ReplaceAll(value, "${AUTOTEAM_WORKER_DIR}", worker.GetWorkerDir())
	value = strings.ReplaceAll(value, "${AUTOTEAM_AGENT_NORMALIZED_NAME}", worker.GetNormalizedName())
	return value
}

func (g *Generator) GenerateCompose(cfg *config.Config) error {
	return g.GenerateComposeWithPorts(cfg, nil)
}

func (g *Generator) GenerateComposeWithPorts(cfg *config.Config, portAllocation ports.PortAllocation) error {
	// Ensure .autoteam directory exists
	if err := g.fileOps.EnsureDirectory(config.AutoTeamDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create .autoteam directory: %w", err)
	}

	// Ensure worker directories exist
	if err := g.createWorkerDirectories(cfg); err != nil {
		return fmt.Errorf("failed to create worker directories: %w", err)
	}

	// Generate worker config files
	if err := g.generateAgentConfigFiles(cfg); err != nil {
		return fmt.Errorf("failed to generate agent config files: %w", err)
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
	workersWithSettings := cfg.GetEnabledWorkersWithEffectiveSettings()

	for _, workerWithSettings := range workersWithSettings {
		worker := workerWithSettings.Worker
		settings := workerWithSettings.Settings
		serviceName := worker.GetNormalizedName()

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
			fmt.Sprintf("./workers/%s:%s", serviceName, worker.GetWorkerDir()),
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

		// Build environment variables - now we only need the config file path
		environment := make(map[string]string)

		// Set the path to the worker's config file
		environment["CONFIG_FILE"] = fmt.Sprintf("%s/config.yaml", worker.GetWorkerDir())

		// Set AutoTeam worker runtime variables with consistent AUTOTEAM_ prefix
		environment["AUTOTEAM_AGENT_NAME"] = worker.Name
		environment["AUTOTEAM_WORKER_DIR"] = worker.GetWorkerDir()
		environment["AUTOTEAM_AGENT_NORMALIZED_NAME"] = worker.GetNormalizedName()

		// Keep some optional runtime variables that can be overridden
		environment["DEBUG"] = "${DEBUG:-false}"
		environment["LOG_LEVEL"] = "${LOG_LEVEL:-info}"

		// Merge with environment from service config and normalize placeholder variables
		if existingEnv, ok := serviceConfig["environment"]; ok {
			// Handle both map[string]string and map[string]interface{} cases
			if envMap, ok := existingEnv.(map[string]string); ok {
				for k, v := range envMap {
					environment[k] = g.normalizeEnvironmentValue(v, worker)
				}
			} else if envMapInterface, ok := existingEnv.(map[string]interface{}); ok {
				for k, v := range envMapInterface {
					if vStr, ok := v.(string); ok {
						environment[k] = g.normalizeEnvironmentValue(vStr, worker)
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
				portMappings := []string{fmt.Sprintf("%d:8080", port)}

				// Merge with existing ports if any
				if existingPorts, ok := serviceConfig["ports"]; ok {
					if portSlice, ok := existingPorts.([]string); ok {
						portMappings = append(portMappings, portSlice...)
					} else if portInterface, ok := existingPorts.([]interface{}); ok {
						for _, p := range portInterface {
							if portStr, ok := p.(string); ok {
								portMappings = append(portMappings, portStr)
							}
						}
					}
				}
				serviceConfig["ports"] = portMappings
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
	// Ensure workers directory exists
	if err := g.fileOps.EnsureDirectory(config.WorkersDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create workers directory: %w", err)
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
make install
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

func (g *Generator) createWorkerDirectories(cfg *config.Config) error {
	for _, worker := range cfg.Workers {
		// Skip disabled agents
		if !worker.IsEnabled() {
			continue
		}
		normalizedName := worker.GetNormalizedName()
		if err := g.fileOps.CreateWorkerDirectoryStructure(normalizedName); err != nil {
			return fmt.Errorf("failed to create directory structure for worker %s (normalized: %s): %w", worker.Name, normalizedName, err)
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

// generateAgentConfigFiles creates YAML config files for each enabled agent
func (g *Generator) generateAgentConfigFiles(cfg *config.Config) error {
	for _, worker := range cfg.Workers {
		// Skip disabled agents
		if !worker.IsEnabled() {
			continue
		}

		settings := worker.GetEffectiveSettings(cfg.Settings)
		workerWithSettings := &config.WorkerWithSettings{Worker: worker, Settings: settings}
		serviceName := worker.GetNormalizedName()

		// Build the worker config (now we generate worker config directly)
		workerConfig := &config.Worker{
			Name:       worker.Name,
			Prompt:     workerWithSettings.GetConsolidatedPrompt(cfg),
			Settings:   &settings,
			MCPServers: worker.MCPServers,
		}

		// Create worker config directory
		workerDir := filepath.Join(config.WorkersDir, serviceName)
		if err := os.MkdirAll(workerDir, 0755); err != nil {
			return fmt.Errorf("failed to create worker config directory %s: %w", workerDir, err)
		}

		// Write config file
		configPath := filepath.Join(workerDir, "config.yaml")
		configData, err := yaml.Marshal(workerConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal config for worker %s: %w", worker.Name, err)
		}

		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", configPath, err)
		}
	}

	return nil
}
