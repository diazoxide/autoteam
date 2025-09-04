package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/ports"
	"autoteam/internal/worker"

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
//   - ${AUTOTEAM_WORKER_NAME} -> actual worker name (e.g., "Senior Developer")
//   - ${AUTOTEAM_WORKER_DIR}  -> worker directory path (e.g., "/opt/autoteam/workers/senior_developer")
//   - ${AUTOTEAM_WORKER_NORMALIZED_NAME} -> normalized worker name (e.g., "senior_developer")
func (g *Generator) normalizeEnvironmentValue(value string, w worker.Worker) string {
	value = strings.ReplaceAll(value, "${AUTOTEAM_WORKER_NAME}", w.Name)
	value = strings.ReplaceAll(value, "${AUTOTEAM_WORKER_DIR}", w.GetWorkerDir())
	value = strings.ReplaceAll(value, "${AUTOTEAM_WORKER_NORMALIZED_NAME}", w.GetNormalizedName())
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
	if err := g.generateWorkerConfigFiles(cfg, portAllocation); err != nil {
		return fmt.Errorf("failed to generate worker config files: %w", err)
	}

	// Generate control-plane config file if control plane is enabled
	if cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		if err := g.generateControlPlaneConfig(cfg, portAllocation); err != nil {
			return fmt.Errorf("failed to generate control-plane config: %w", err)
		}
	}

	// Generate compose.yaml programmatically
	if err := g.generateComposeYAML(cfg, portAllocation); err != nil {
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	// Copy system bin directory
	if err := g.copyBinDirectory(); err != nil {
		return fmt.Errorf("failed to copy bin directory: %w", err)
	}

	// Build and copy control-plane binary if enabled
	if cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		if err := g.buildControlPlaneBinary(); err != nil {
			return fmt.Errorf("failed to build control-plane binary: %w", err)
		}
	}

	// Build and copy dashboard binary if enabled
	if cfg.Dashboard != nil && cfg.Dashboard.Enabled {
		if err := g.buildDashboardBinary(); err != nil {
			return fmt.Errorf("failed to build dashboard binary: %w", err)
		}
	}

	return nil
}

// generateComposeYAML creates a Docker Compose YAML file programmatically
func (g *Generator) generateComposeYAML(cfg *config.Config, portAllocation ports.PortAllocation) error {
	compose := ComposeConfig{
		Services: make(map[string]interface{}),
	}

	// Get only enabled workers with their effective settings
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

		// Build volumes array with team-based paths
		teamName := cfg.GetTeamName()
		volumes := []string{
			fmt.Sprintf("./%s/workers/%s:%s", teamName, serviceName, worker.GetWorkerDir()),
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
		environment["AUTOTEAM_WORKER_NAME"] = worker.Name
		environment["AUTOTEAM_WORKER_DIR"] = worker.GetWorkerDir()
		environment["AUTOTEAM_WORKER_NORMALIZED_NAME"] = worker.GetNormalizedName()

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
				// Use same port for both host and container (dynamic port discovery)
				portMappings := []string{fmt.Sprintf("%d:%d", port, port)}

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

	// Add control-plane service if enabled
	if cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		controlPlaneService := g.generateControlPlaneService(cfg)
		compose.Services["control-plane"] = controlPlaneService
	}

	// Add dashboard service if enabled
	if cfg.Dashboard != nil && cfg.Dashboard.Enabled {
		dashboardService := g.generateDashboardService(cfg)
		compose.Services["dashboard"] = dashboardService
	}

	// Add custom services from configuration
	if cfg.Services != nil {
		for serviceName, serviceConfig := range cfg.Services {
			// Check for conflicts with worker services
			if _, exists := compose.Services[serviceName]; exists {
				return fmt.Errorf("custom service '%s' conflicts with generated worker service - please choose a different name", serviceName)
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
	// Ensure .autoteam directory exists (workers directory will be created in createWorkerDirectories)
	if err := g.fileOps.EnsureDirectory(config.AutoTeamDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create .autoteam directory: %w", err)
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
			// Neither directory exists - copy from project scripts directory
			if err := g.fileOps.EnsureDirectory(config.LocalBinPath, config.DirPerm); err != nil {
				return fmt.Errorf("failed to create bin directory: %w", err)
			}

			// Copy entrypoint.sh from scripts directory
			scriptsEntrypoint := filepath.Join("scripts", config.EntrypointScript)
			localEntrypoint := filepath.Join(config.LocalBinPath, config.EntrypointScript)

			if g.fileOps.FileExists(scriptsEntrypoint) {
				if err := g.fileOps.CopyFile(scriptsEntrypoint, localEntrypoint); err != nil {
					return fmt.Errorf("failed to copy entrypoint script: %w", err)
				}
				// Make it executable
				if err := g.fileOps.SetPermissions(localEntrypoint, config.ExecutablePerm); err != nil {
					return fmt.Errorf("failed to set entrypoint script permissions: %w", err)
				}
			}

			readmePath := filepath.Join(config.LocalBinPath, config.ReadmeFile)
			readmeContent := `# AutoTeam Binary Directory

This directory contains:
- entrypoint.sh: Main container entrypoint script (copied from project scripts/)

For system-wide installation with all platform binaries, run:
` + "```bash" + `
make install
` + "```" + `

Supported platforms: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64
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
	workersDir := cfg.GetWorkersDir()
	for _, w := range cfg.Workers {
		// Skip disabled workers
		if !w.IsEnabled() {
			continue
		}
		normalizedName := w.GetNormalizedName()
		if err := g.fileOps.CreateWorkerDirectoryStructure(workersDir, normalizedName); err != nil {
			return fmt.Errorf("failed to create directory structure for worker %s (normalized: %s): %w", w.Name, normalizedName, err)
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

// generateWorkerConfigFiles creates YAML config files for each enabled worker
func (g *Generator) generateWorkerConfigFiles(cfg *config.Config, portAllocation ports.PortAllocation) error {
	for _, w := range cfg.Workers {
		// Skip disabled workers
		if !w.IsEnabled() {
			continue
		}

		settings := w.GetEffectiveSettings(cfg.Settings)
		workerWithSettings := &worker.WorkerWithSettings{Worker: w, Settings: settings}
		serviceName := w.GetNormalizedName()

		// Set HTTP port from port allocation if available
		if portAllocation != nil {
			if port, hasPort := portAllocation[serviceName]; hasPort {
				if settings.HTTPPort == nil {
					settings.HTTPPort = &port
				}
			}
		}

		// Build the worker config (now we generate worker config directly)
		workerConfig := &worker.Worker{
			Name:     w.Name,
			Prompt:   workerWithSettings.GetConsolidatedPrompt(),
			Settings: &settings,
		}

		// Create worker config directory using team-based path
		workersDir := cfg.GetWorkersDir()
		workerDir := filepath.Join(workersDir, serviceName)
		if err := os.MkdirAll(workerDir, 0755); err != nil {
			return fmt.Errorf("failed to create worker config directory %s: %w", workerDir, err)
		}

		// Write config file
		configPath := filepath.Join(workerDir, "config.yaml")
		configData, err := yaml.Marshal(workerConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal config for worker %s: %w", w.Name, err)
		}

		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", configPath, err)
		}
	}

	return nil
}

// buildControlPlaneBinary builds the Linux control-plane binary and copies it to bin directory
func (g *Generator) buildControlPlaneBinary() error {
	targetBinaryPath := filepath.Join(config.LocalBinPath, "autoteam-control-plane")

	// Try to find Linux binary first (preferred for containers)
	linuxBinaryPath := "build/autoteam-control-plane-linux-amd64"
	if g.fileOps.FileExists(linuxBinaryPath) {
		return g.fileOps.CopyFile(linuxBinaryPath, targetBinaryPath)
	}

	// Fallback to current platform binary
	if g.fileOps.FileExists("build/autoteam-control-plane") {
		return g.fileOps.CopyFile("build/autoteam-control-plane", targetBinaryPath)
	}

	return fmt.Errorf("control-plane binary not found - please run 'make build-control-plane' or 'make build-all' first")
}

// generateControlPlaneService creates the Docker Compose service configuration for control-plane
func (g *Generator) generateControlPlaneService(cfg *config.Config) map[string]interface{} {
	teamName := cfg.GetTeamName()

	service := map[string]interface{}{
		"build": map[string]interface{}{
			"context":    "./",
			"dockerfile": "../Dockerfile",
		},
		"tty":         true,
		"stdin_open":  true,
		"user":        "root",
		"working_dir": "/opt/autoteam",
		"volumes": []string{
			fmt.Sprintf("./%s/control-plane:/opt/autoteam/control-plane", teamName),
			"./bin:/opt/autoteam/bin",
		},
		"environment": map[string]string{
			"CONFIG_FILE":          "/opt/autoteam/control-plane/config.yaml",
			"CONTROL_PLANE_CONFIG": "/opt/autoteam/control-plane/config.yaml",
		},
		"entrypoint": []string{"/opt/autoteam/bin/autoteam-control-plane"},
		"command": []string{
			"--log-level", "${LOG_LEVEL:-info}",
		},
		"ports": []string{
			fmt.Sprintf("%d:%d", cfg.ControlPlane.Port, cfg.ControlPlane.Port),
		},
	}

	return service
}

// generateControlPlaneConfig creates a control-plane config file in the team-specific directory
func (g *Generator) generateControlPlaneConfig(cfg *config.Config, portAllocation ports.PortAllocation) error {
	// Create control-plane config directory
	controlPlaneDir := cfg.GetControlPlaneDir()
	if err := os.MkdirAll(controlPlaneDir, 0755); err != nil {
		return fmt.Errorf("failed to create control-plane config directory %s: %w", controlPlaneDir, err)
	}

	// Build worker API URLs from enabled workers and their allocated ports
	var workersAPIs []string
	if portAllocation != nil {
		for _, worker := range cfg.Workers {
			if worker.IsEnabled() {
				serviceName := worker.GetNormalizedName()
				if port, hasPort := portAllocation[serviceName]; hasPort {
					workerURL := fmt.Sprintf("http://%s:%d", serviceName, port)
					workersAPIs = append(workersAPIs, workerURL)
				}
			}
		}
	}

	// Build control-plane config with worker API URLs
	controlPlaneConfig := &config.ControlPlaneConfig{
		Enabled:     cfg.ControlPlane.Enabled,
		Port:        cfg.ControlPlane.Port,
		APIKey:      cfg.ControlPlane.APIKey,
		WorkersAPIs: workersAPIs,
	}

	// Write config file
	configPath := cfg.GetControlPlaneConfigPath()
	configData, err := yaml.Marshal(controlPlaneConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal control-plane config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write control-plane config file %s: %w", configPath, err)
	}

	return nil
}

// buildDashboardBinary builds the Linux dashboard binary and copies it to bin directory
func (g *Generator) buildDashboardBinary() error {
	targetBinaryPath := filepath.Join(config.LocalBinPath, "autoteam-dashboard")

	// Try to copy from build/autoteam-dashboard-linux-amd64 first (cross-compiled)
	linuxBinaryPath := "build/autoteam-dashboard-linux-amd64"
	if g.fileOps.FileExists(linuxBinaryPath) {
		return g.fileOps.CopyFile(linuxBinaryPath, targetBinaryPath)
	}

	// Fall back to current platform binary
	if g.fileOps.FileExists("build/autoteam-dashboard") {
		return g.fileOps.CopyFile("build/autoteam-dashboard", targetBinaryPath)
	}

	return fmt.Errorf("dashboard binary not found - please run 'make build-dashboard' or 'make build-all' first")
}

// generateDashboardService creates the Docker Compose service configuration for dashboard
func (g *Generator) generateDashboardService(cfg *config.Config) map[string]interface{} {
	// Determine API URL - default to control plane if available, otherwise use configured URL
	apiUrl := cfg.Dashboard.APIUrl
	if apiUrl == "" && cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		apiUrl = fmt.Sprintf("http://control-plane:%d", cfg.ControlPlane.Port)
	}
	if apiUrl == "" {
		apiUrl = "http://localhost:9090" // fallback
	}

	service := map[string]interface{}{
		"image": "alpine:latest",
		"volumes": []string{
			"./bin/autoteam-dashboard:/autoteam-dashboard:ro",
		},
		"environment": map[string]string{
			"DASHBOARD_PORT":  fmt.Sprintf("%d", cfg.Dashboard.Port),
			"API_URL":         apiUrl,
			"DASHBOARD_TITLE": cfg.Dashboard.Title,
		},
		"entrypoint": []string{"/autoteam-dashboard"},
		"ports": []string{
			fmt.Sprintf("%d:%d", cfg.Dashboard.Port, cfg.Dashboard.Port),
		},
	}

	// Add dependency on control-plane if it's enabled
	if cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		service["depends_on"] = []string{"control-plane"}
	}

	return service
}
