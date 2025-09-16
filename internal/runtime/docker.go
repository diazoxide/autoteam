package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"autoteam/internal/config"
	"autoteam/internal/logger"
	"autoteam/internal/worker"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// DockerRuntime implements the Runtime interface using Docker API
type DockerRuntime struct {
	client    *client.Client
	config    map[string]interface{}
	imageName string
}

// NewDockerRuntime creates a new Docker runtime instance
func NewDockerRuntime(runtimeConfig map[string]interface{}) (Runtime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerRuntime{
		client:    cli,
		config:    runtimeConfig,
		imageName: "autoteam:latest", // Default image name
	}, nil
}

// Initialize sets up the Docker runtime environment
func (d *DockerRuntime) Initialize(ctx context.Context, cfg *config.Config) error {
	log := logger.FromContext(ctx)

	// Create team-specific network
	networkName := d.getNetworkName(cfg)
	if err := d.ensureNetwork(ctx, networkName); err != nil {
		return fmt.Errorf("failed to ensure network %s: %w", networkName, err)
	}

	// Ensure local bin directory has required binaries
	if err := d.ensureBinaries(ctx); err != nil {
		return fmt.Errorf("failed to ensure binaries: %w", err)
	}

	// Generate config files for workers and control plane
	if err := d.generateConfigFiles(cfg); err != nil {
		return fmt.Errorf("failed to generate config files: %w", err)
	}

	log.Info("Docker runtime initialized",
		zap.String("network", networkName),
		zap.String("image", d.imageName),
		zap.String("team", cfg.GetTeamName()))

	return nil
}

// DeployWorker deploys a worker container
func (d *DockerRuntime) DeployWorker(ctx context.Context, w worker.Worker, settings worker.WorkerSettings, cfg *config.Config) error {
	log := logger.FromContext(ctx)
	containerName := d.getWorkerContainerName(w, cfg)

	// Stop existing container if it exists
	if err := d.stopContainer(ctx, containerName); err != nil {
		log.Warn("Failed to stop existing worker container", zap.Error(err), zap.String("worker", w.Name))
	}

	// Prepare container configuration
	containerConfig := d.buildWorkerContainerConfig(w, settings, cfg)

	// Ensure the worker's image exists locally
	if err := d.ensureImage(ctx, containerConfig.Image); err != nil {
		return fmt.Errorf("failed to ensure worker image %s: %w", containerConfig.Image, err)
	}

	// Create and start container
	if err := d.createAndStartContainer(ctx, containerName, containerConfig); err != nil {
		return fmt.Errorf("failed to deploy worker %s: %w", w.Name, err)
	}

	log.Info("Worker deployed successfully", zap.String("worker", w.Name), zap.String("container", containerName))
	return nil
}

// DeployControlPlane deploys the control plane service
func (d *DockerRuntime) DeployControlPlane(ctx context.Context, cfg *config.Config) error {
	if cfg.ControlPlane == nil || !cfg.ControlPlane.Enabled {
		return nil
	}

	log := logger.FromContext(ctx)
	containerName := d.getControlPlaneContainerName(cfg)

	// Stop existing container if it exists
	if err := d.stopContainer(ctx, containerName); err != nil {
		log.Warn("Failed to stop existing control plane container", zap.Error(err))
	}

	// Prepare container configuration
	containerConfig := d.buildControlPlaneContainerConfig(cfg)

	// Create and start container
	if err := d.createAndStartContainer(ctx, containerName, containerConfig); err != nil {
		return fmt.Errorf("failed to deploy control plane: %w", err)
	}

	log.Info("Control plane deployed successfully", zap.String("container", containerName))
	return nil
}

// DeployDashboard deploys the dashboard service
func (d *DockerRuntime) DeployDashboard(ctx context.Context, cfg *config.Config) error {
	if cfg.Dashboard == nil || !cfg.Dashboard.Enabled {
		return nil
	}

	log := logger.FromContext(ctx)
	containerName := d.getDashboardContainerName(cfg)

	// Stop existing container if it exists
	if err := d.stopContainer(ctx, containerName); err != nil {
		log.Warn("Failed to stop existing dashboard container", zap.Error(err))
	}

	// Prepare container configuration
	containerConfig := d.buildDashboardContainerConfig(cfg)

	// Create and start container
	if err := d.createAndStartContainer(ctx, containerName, containerConfig); err != nil {
		return fmt.Errorf("failed to deploy dashboard: %w", err)
	}

	log.Info("Dashboard deployed successfully", zap.String("container", containerName))
	return nil
}

// DeployService deploys a custom service
func (d *DockerRuntime) DeployService(ctx context.Context, name string, serviceConfig map[string]interface{}, cfg *config.Config) error {
	log := logger.FromContext(ctx)
	containerName := d.getServiceContainerName(name, cfg)

	// Stop existing container if it exists
	if err := d.stopContainer(ctx, containerName); err != nil {
		log.Warn("Failed to stop existing service container", zap.Error(err), zap.String("service", name))
	}

	// Prepare container configuration
	containerConfig := d.buildServiceContainerConfig(name, serviceConfig, cfg)

	// Create and start container
	if err := d.createAndStartContainer(ctx, containerName, containerConfig); err != nil {
		return fmt.Errorf("failed to deploy service %s: %w", name, err)
	}

	log.Info("Service deployed successfully", zap.String("service", name), zap.String("container", containerName))
	return nil
}

// StopWorker stops a specific worker container
func (d *DockerRuntime) StopWorker(ctx context.Context, workerName string, cfg *config.Config) error {
	containerName := d.getWorkerContainerNameByName(workerName, cfg)
	return d.stopContainer(ctx, containerName)
}

// StopControlPlane stops the control plane service
func (d *DockerRuntime) StopControlPlane(ctx context.Context, cfg *config.Config) error {
	containerName := d.getControlPlaneContainerName(cfg)
	return d.stopContainer(ctx, containerName)
}

// StopDashboard stops the dashboard service
func (d *DockerRuntime) StopDashboard(ctx context.Context, cfg *config.Config) error {
	containerName := d.getDashboardContainerName(cfg)
	return d.stopContainer(ctx, containerName)
}

// StopService stops a custom service
func (d *DockerRuntime) StopService(ctx context.Context, serviceName string, cfg *config.Config) error {
	containerName := d.getServiceContainerName(serviceName, cfg)
	return d.stopContainer(ctx, containerName)
}

// StopAll stops all containers for the team
func (d *DockerRuntime) StopAll(ctx context.Context, cfg *config.Config) error {
	log := logger.FromContext(ctx)
	teamName := cfg.GetTeamName()

	// List all containers with team label
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var errors []string
	for _, cont := range containers {
		// Check if container belongs to this team
		if teamLabel, exists := cont.Labels["autoteam.team"]; exists && teamLabel == teamName {
			log.Debug("Stopping team container", zap.String("container", cont.Names[0]), zap.String("team", teamName))

			timeout := 30 // 30 seconds timeout
			if err := d.client.ContainerStop(ctx, cont.ID, container.StopOptions{Timeout: &timeout}); err != nil {
				errors = append(errors, fmt.Sprintf("failed to stop container %s: %v", cont.Names[0], err))
				continue
			}

			// Remove the container
			if err := d.client.ContainerRemove(ctx, cont.ID, container.RemoveOptions{Force: true}); err != nil {
				errors = append(errors, fmt.Sprintf("failed to remove container %s: %v", cont.Names[0], err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping containers: %s", strings.Join(errors, "; "))
	}

	log.Info("All team containers stopped", zap.String("team", teamName))
	return nil
}

// GetStatus returns the status of all deployed services
func (d *DockerRuntime) GetStatus(ctx context.Context, cfg *config.Config) ([]ServiceStatus, error) {
	teamName := cfg.GetTeamName()
	var services []ServiceStatus

	// List all containers with team label
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		// Check if container belongs to this team
		if teamLabel, exists := cont.Labels["autoteam.team"]; exists && teamLabel == teamName {
			status := ServiceStatus{
				Name:      strings.TrimPrefix(cont.Names[0], "/"),
				Status:    cont.State,
				CreatedAt: time.Unix(cont.Created, 0),
				Image:     cont.Image,
			}

			// Map Docker ports
			for _, port := range cont.Ports {
				if port.PublicPort > 0 {
					status.Ports = append(status.Ports, fmt.Sprintf("%d:%d", port.PublicPort, port.PrivatePort))
				}
			}

			// Set health status based on container state
			switch cont.State {
			case "running":
				status.Health = "healthy"
			case "exited":
				status.Health = "unhealthy"
			default:
				status.Health = "unknown"
			}

			services = append(services, status)
		}
	}

	return services, nil
}

// GetLogs retrieves logs from a specific service
func (d *DockerRuntime) GetLogs(ctx context.Context, serviceName string, lines int, cfg *config.Config) ([]string, error) {
	containerName := fmt.Sprintf("%s-%s", cfg.GetTeamName(), serviceName)

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(lines),
	}

	reader, err := d.client.ContainerLogs(ctx, containerName, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for container %s: %w", containerName, err)
	}
	defer reader.Close()

	logBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	// Split logs into lines
	logLines := strings.Split(string(logBytes), "\n")

	// Remove empty last line if present
	if len(logLines) > 0 && logLines[len(logLines)-1] == "" {
		logLines = logLines[:len(logLines)-1]
	}

	return logLines, nil
}

// Cleanup removes all resources for the team
func (d *DockerRuntime) Cleanup(ctx context.Context, cfg *config.Config) error {
	log := logger.FromContext(ctx)
	teamName := cfg.GetTeamName()

	// Stop all containers first
	if err := d.StopAll(ctx, cfg); err != nil {
		log.Warn("Failed to stop all containers during cleanup", zap.Error(err))
	}

	// Remove the team network
	networkName := d.getNetworkName(cfg)
	if err := d.client.NetworkRemove(ctx, networkName); err != nil {
		log.Warn("Failed to remove network during cleanup", zap.Error(err), zap.String("network", networkName))
	}

	log.Info("Runtime cleanup completed", zap.String("team", teamName))
	return nil
}

// Helper methods

func (d *DockerRuntime) getNetworkName(cfg *config.Config) string {
	if networkName, ok := d.config["network_name"].(string); ok {
		return networkName
	}
	return fmt.Sprintf("%s-network", cfg.GetTeamName())
}

func (d *DockerRuntime) getWorkerContainerName(w worker.Worker, cfg *config.Config) string {
	return fmt.Sprintf("%s-%s", cfg.GetTeamName(), w.GetNormalizedName())
}

func (d *DockerRuntime) getWorkerContainerNameByName(workerName string, cfg *config.Config) string {
	// Normalize the worker name the same way as in worker package
	normalized := strings.ToLower(strings.ReplaceAll(workerName, " ", "_"))
	return fmt.Sprintf("%s-%s", cfg.GetTeamName(), normalized)
}

func (d *DockerRuntime) getControlPlaneContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-control-plane", cfg.GetTeamName())
}

func (d *DockerRuntime) getDashboardContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-dashboard", cfg.GetTeamName())
}

func (d *DockerRuntime) getServiceContainerName(serviceName string, cfg *config.Config) string {
	return fmt.Sprintf("%s-%s", cfg.GetTeamName(), serviceName)
}

func (d *DockerRuntime) ensureNetwork(ctx context.Context, networkName string) error {
	// Check if network already exists
	networks, err := d.client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			return nil // Network already exists
		}
	}

	// Create network
	_, err = d.client.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"autoteam.managed": "true",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", networkName, err)
	}

	return nil
}

func (d *DockerRuntime) ensureImage(ctx context.Context, imageName string) error {
	log := logger.FromContext(ctx)

	// Check if image exists locally
	images, err := d.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				log.Debug("Docker image already exists", zap.String("image", imageName))
				return nil // Image already exists
			}
		}
	}

	// Image not found locally, try to pull it
	log.Info("Docker image not found locally, pulling from registry",
		zap.String("image", imageName))

	reader, err := d.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read the pull output to ensure it completes
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read image pull output: %w", err)
	}

	log.Info("Successfully pulled Docker image", zap.String("image", imageName))
	return nil
}

func (d *DockerRuntime) ensureBinaries(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Create local bin directory if it doesn't exist
	if err := os.MkdirAll("bin", 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// List of required files to copy from system to local bin
	systemBinDir := "/opt/autoteam/bin"
	requiredFiles := []string{
		"entrypoint.sh",
		"autoteam-worker-linux-amd64",
		"autoteam-worker-linux-arm64",
		"autoteam-worker-darwin-amd64",
		"autoteam-worker-darwin-arm64",
	}

	// Add control plane and dashboard binaries from build directory
	buildBinaries := map[string]string{
		"autoteam-control-plane": "build/autoteam-control-plane-linux-amd64",
		"autoteam-dashboard":     "build/autoteam-dashboard-linux-amd64",
	}

	for binary, buildPath := range buildBinaries {
		localPath := fmt.Sprintf("bin/%s", binary)

		// Copy from build directory if it exists and local is outdated or missing
		if buildInfo, err := os.Stat(buildPath); err == nil {
			shouldCopy := true
			if localInfo, err := os.Stat(localPath); err == nil {
				if localInfo.ModTime().After(buildInfo.ModTime()) {
					shouldCopy = false
				}
			}

			if shouldCopy {
				if err := d.copyFile(buildPath, localPath); err != nil {
					log.Warn("Failed to copy build binary", zap.String("binary", binary), zap.Error(err))
				} else {
					log.Debug("Copied build binary to local bin", zap.String("binary", binary))
				}
			}
		}
	}

	for _, file := range requiredFiles {
		systemPath := fmt.Sprintf("%s/%s", systemBinDir, file)
		localPath := fmt.Sprintf("bin/%s", file)

		// Check if local file already exists and is newer than system file
		if localInfo, err := os.Stat(localPath); err == nil {
			if systemInfo, err := os.Stat(systemPath); err == nil {
				if localInfo.ModTime().After(systemInfo.ModTime()) {
					log.Debug("Local binary is up to date", zap.String("file", file))
					continue
				}
			}
		}

		// Copy file from system to local
		if err := d.copyFile(systemPath, localPath); err != nil {
			log.Warn("Failed to copy binary", zap.String("file", file), zap.Error(err))
			continue
		}

		log.Debug("Copied binary to local bin", zap.String("file", file))
	}

	return nil
}

func (d *DockerRuntime) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy permissions
	if info, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, info.Mode()) // Ignore error - not critical
	}

	return nil
}

func (d *DockerRuntime) generateConfigFiles(cfg *config.Config) error {
	teamName := cfg.GetTeamName()

	// Create team directory structure under .autoteam
	teamDir := fmt.Sprintf("./.autoteam/%s", teamName)
	if err := os.MkdirAll(teamDir, 0755); err != nil {
		return fmt.Errorf("failed to create team directory: %w", err)
	}

	// Generate worker configs
	for _, w := range cfg.Workers {
		if err := d.generateWorkerConfig(w, cfg); err != nil {
			return fmt.Errorf("failed to generate config for worker %s: %w", w.Name, err)
		}
	}

	// Generate control plane config
	if cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		if err := d.generateControlPlaneConfig(cfg); err != nil {
			return fmt.Errorf("failed to generate control plane config: %w", err)
		}
	}

	return nil
}

func (d *DockerRuntime) generateWorkerConfig(w worker.Worker, cfg *config.Config) error {
	teamName := cfg.GetTeamName()
	workerNormalizedName := strings.ToLower(strings.ReplaceAll(w.Name, " ", "_"))

	// Create worker-specific directory under .autoteam
	workerDir := fmt.Sprintf("./.autoteam/%s/workers/%s", teamName, workerNormalizedName)
	if err := os.MkdirAll(workerDir, 0755); err != nil {
		return fmt.Errorf("failed to create worker directory: %w", err)
	}

	// Create worker-specific config file containing just this worker's configuration
	configPath := fmt.Sprintf("%s/config.yaml", workerDir)
	configData, err := yaml.Marshal(w)
	if err != nil {
		return fmt.Errorf("failed to marshal worker config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write worker config file: %w", err)
	}

	return nil
}

func (d *DockerRuntime) generateControlPlaneConfig(cfg *config.Config) error {
	teamName := cfg.GetTeamName()

	// Create control plane directory under .autoteam
	controlPlaneDir := fmt.Sprintf("./.autoteam/%s/control-plane", teamName)
	if err := os.MkdirAll(controlPlaneDir, 0755); err != nil {
		return fmt.Errorf("failed to create control plane directory: %w", err)
	}

	// Generate worker URLs using actual container names (gRPC format)
	var workersAPIs []string
	for _, w := range cfg.Workers {
		containerName := d.getWorkerContainerName(w, cfg)
		workerURL := fmt.Sprintf("%s:8080", containerName)
		workersAPIs = append(workersAPIs, workerURL)
	}

	// Create control plane config with correct worker URLs
	controlPlaneConfig := map[string]interface{}{
		"enabled":      cfg.ControlPlane.Enabled,
		"port":         cfg.ControlPlane.Port,
		"workers_apis": workersAPIs,
	}

	// Create config file
	configPath := fmt.Sprintf("%s/config.yaml", controlPlaneDir)
	configData, err := yaml.Marshal(controlPlaneConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal control plane config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write control plane config file: %w", err)
	}

	return nil
}

func (d *DockerRuntime) buildWorkerContainerConfig(w worker.Worker, settings worker.WorkerSettings, cfg *config.Config) *ContainerConfig {
	teamName := cfg.GetTeamName()
	workerDir := w.GetWorkerDir()

	// Build environment variables with proper defaults
	debugValue := os.Getenv("DEBUG")
	if debugValue == "" {
		debugValue = "false"
	}
	logLevelValue := os.Getenv("LOG_LEVEL")
	if logLevelValue == "" {
		logLevelValue = "info"
	}

	environment := map[string]string{
		"CONFIG_FILE":                     fmt.Sprintf("%s/config.yaml", workerDir),
		"AUTOTEAM_WORKER_NAME":            w.Name,
		"AUTOTEAM_WORKER_DIR":             workerDir,
		"AUTOTEAM_WORKER_NORMALIZED_NAME": w.GetNormalizedName(),
		"DEBUG":                           debugValue,
		"LOG_LEVEL":                       logLevelValue,
		"GRPC_PORT":                       "8080",
	}

	// Merge with settings environment
	if settings.Service != nil {
		if env, ok := settings.Service["environment"]; ok {
			if envMap, ok := env.(map[string]interface{}); ok {
				for k, v := range envMap {
					if vStr, ok := v.(string); ok {
						environment[k] = vStr
					}
				}
			} else if envMapStr, ok := env.(map[string]string); ok {
				for k, v := range envMapStr {
					environment[k] = v
				}
			}
		}
	}

	// Build volumes with absolute paths
	currentDir, err := os.Getwd()
	if err != nil {
		return nil
	}
	volumes := []string{
		fmt.Sprintf("%s/.autoteam/%s/workers/%s:%s", currentDir, teamName, w.GetNormalizedName(), workerDir),
		fmt.Sprintf("%s/bin:/opt/autoteam/bin", currentDir),
	}

	// Add custom volumes from settings
	if settings.Service != nil {
		if vols, ok := settings.Service["volumes"]; ok {
			if volSlice, ok := vols.([]string); ok {
				volumes = append(volumes, volSlice...)
			} else if volInterface, ok := vols.([]interface{}); ok {
				for _, v := range volInterface {
					if vStr, ok := v.(string); ok {
						volumes = append(volumes, vStr)
					}
				}
			}
		}
	}

	// Get image from settings or use default
	imageName := d.imageName
	if settings.Service != nil {
		if img, ok := settings.Service["image"].(string); ok && img != "" {
			imageName = img
		}
	}

	// Get user from settings
	user := "developer" // default
	if settings.Service != nil {
		if u, ok := settings.Service["user"].(string); ok && u != "" {
			user = u
		}
	}

	return &ContainerConfig{
		Name:          d.getWorkerContainerName(w, cfg),
		Image:         imageName,
		Environment:   environment,
		Volumes:       volumes,
		Entrypoint:    []string{"/opt/autoteam/bin/entrypoint.sh"},
		WorkingDir:    "/opt/autoteam",
		User:          user,
		NetworkName:   d.getNetworkName(cfg),
		RestartPolicy: "unless-stopped",
	}
}

func (d *DockerRuntime) buildControlPlaneContainerConfig(cfg *config.Config) *ContainerConfig {
	teamName := cfg.GetTeamName()

	environment := map[string]string{
		"CONTROL_PLANE_CONFIG": "/opt/autoteam/control-plane/config.yaml",
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil
	}
	volumes := []string{
		fmt.Sprintf("%s/.autoteam/%s/control-plane:/opt/autoteam/control-plane", currentDir, teamName),
		fmt.Sprintf("%s/bin:/opt/autoteam/bin", currentDir),
	}

	ports := []string{
		fmt.Sprintf("%d:%d", cfg.ControlPlane.Port, cfg.ControlPlane.Port),
	}

	return &ContainerConfig{
		Name:          d.getControlPlaneContainerName(cfg),
		Image:         "alpine:latest",
		Environment:   environment,
		Volumes:       volumes,
		Ports:         ports,
		Entrypoint:    []string{"/opt/autoteam/bin/autoteam-control-plane"},
		Command:       []string{"--log-level", "info"},
		WorkingDir:    "/opt/autoteam",
		User:          "root",
		NetworkName:   d.getNetworkName(cfg),
		RestartPolicy: "unless-stopped",
	}
}

func (d *DockerRuntime) buildDashboardContainerConfig(cfg *config.Config) *ContainerConfig {
	// Determine API URL
	apiUrl := cfg.Dashboard.APIUrl
	if apiUrl == "" && cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		apiUrl = fmt.Sprintf("http://%s:%d", d.getControlPlaneContainerName(cfg), cfg.ControlPlane.Port)
	}

	environment := map[string]string{
		"DASHBOARD_PORT":  fmt.Sprintf("%d", cfg.Dashboard.Port),
		"API_URL":         apiUrl,
		"DASHBOARD_TITLE": cfg.Dashboard.Title,
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil
	}
	volumes := []string{
		fmt.Sprintf("%s/bin/autoteam-dashboard:/autoteam-dashboard:ro", currentDir),
	}

	ports := []string{
		fmt.Sprintf("%d:%d", cfg.Dashboard.Port, cfg.Dashboard.Port),
	}

	return &ContainerConfig{
		Name:          d.getDashboardContainerName(cfg),
		Image:         "alpine:latest",
		Environment:   environment,
		Volumes:       volumes,
		Ports:         ports,
		Entrypoint:    []string{"/autoteam-dashboard"},
		NetworkName:   d.getNetworkName(cfg),
		RestartPolicy: "unless-stopped",
	}
}

func (d *DockerRuntime) buildServiceContainerConfig(name string, serviceConfig map[string]interface{}, cfg *config.Config) *ContainerConfig {
	containerConfig := &ContainerConfig{
		Name:          d.getServiceContainerName(name, cfg),
		NetworkName:   d.getNetworkName(cfg),
		RestartPolicy: "unless-stopped",
	}

	// Map service config fields to container config
	if image, ok := serviceConfig["image"].(string); ok {
		containerConfig.Image = image
	}

	if env, ok := serviceConfig["environment"]; ok {
		environment := make(map[string]string)
		if envMap, ok := env.(map[string]interface{}); ok {
			for k, v := range envMap {
				if vStr, ok := v.(string); ok {
					environment[k] = vStr
				}
			}
		} else if envMapStr, ok := env.(map[string]string); ok {
			environment = envMapStr
		}
		containerConfig.Environment = environment
	}

	if vols, ok := serviceConfig["volumes"]; ok {
		var volumes []string
		if volSlice, ok := vols.([]string); ok {
			volumes = volSlice
		} else if volInterface, ok := vols.([]interface{}); ok {
			for _, v := range volInterface {
				if vStr, ok := v.(string); ok {
					volumes = append(volumes, vStr)
				}
			}
		}
		containerConfig.Volumes = volumes
	}

	if ports, ok := serviceConfig["ports"]; ok {
		var portMappings []string
		if portSlice, ok := ports.([]string); ok {
			portMappings = portSlice
		} else if portInterface, ok := ports.([]interface{}); ok {
			for _, p := range portInterface {
				if pStr, ok := p.(string); ok {
					portMappings = append(portMappings, pStr)
				}
			}
		}
		containerConfig.Ports = portMappings
	}

	if workingDir, ok := serviceConfig["working_dir"].(string); ok {
		containerConfig.WorkingDir = workingDir
	}

	if user, ok := serviceConfig["user"].(string); ok {
		containerConfig.User = user
	}

	return containerConfig
}

func (d *DockerRuntime) stopContainer(ctx context.Context, containerName string) error {
	log := logger.FromContext(ctx)

	// Check if container exists
	containers, err := d.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var containerID string
	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				containerID = cont.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		log.Debug("Container not found, skipping stop", zap.String("container", containerName))
		return nil // Container doesn't exist
	}

	// Stop container with timeout
	timeout := 30 // 30 seconds timeout
	if err := d.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerName, err)
	}

	// Remove container
	if err := d.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerName, err)
	}

	log.Debug("Container stopped and removed", zap.String("container", containerName))
	return nil
}

func (d *DockerRuntime) createAndStartContainer(ctx context.Context, containerName string, config *ContainerConfig) error {
	log := logger.FromContext(ctx)

	// Build Docker container config
	containerConfig := &container.Config{
		Image:      config.Image,
		Env:        d.mapToEnvSlice(config.Environment),
		WorkingDir: config.WorkingDir,
		User:       config.User,
		Labels: map[string]string{
			"autoteam.managed": "true",
			"autoteam.team":    containerName[:strings.LastIndex(containerName, "-")], // Extract team name
		},
		Tty:       true,
		OpenStdin: true,
	}

	if len(config.Entrypoint) > 0 {
		containerConfig.Entrypoint = config.Entrypoint
	}
	if len(config.Command) > 0 {
		containerConfig.Cmd = config.Command
	}

	// Build host config
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(config.RestartPolicy)},
	}

	// Add volume mounts
	hostConfig.Binds = append(hostConfig.Binds, config.Volumes...)

	// Add port mappings
	for _, portMapping := range config.Ports {
		// Parse "host:container" format
		parts := strings.Split(portMapping, ":")
		if len(parts) == 2 {
			hostPort := parts[0]
			containerPort := parts[1]

			// Set exposed ports
			if containerConfig.ExposedPorts == nil {
				containerConfig.ExposedPorts = make(nat.PortSet)
			}
			port, _ := nat.NewPort("tcp", containerPort)
			containerConfig.ExposedPorts[port] = struct{}{}

			// Set port bindings
			if hostConfig.PortBindings == nil {
				hostConfig.PortBindings = make(nat.PortMap)
			}
			hostConfig.PortBindings[port] = []nat.PortBinding{
				{HostPort: hostPort},
			}
		}
	}

	// Create container
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", containerName, err)
	}

	// Connect to network
	if config.NetworkName != "" {
		if err := d.client.NetworkConnect(ctx, config.NetworkName, resp.ID, nil); err != nil {
			log.Warn("Failed to connect container to network", zap.Error(err), zap.String("container", containerName), zap.String("network", config.NetworkName))
		}
	}

	// Start container
	if err := d.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerName, err)
	}

	log.Debug("Container created and started", zap.String("container", containerName), zap.String("id", resp.ID))
	return nil
}

func (d *DockerRuntime) mapToEnvSlice(env map[string]string) []string {
	var envSlice []string
	for k, v := range env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return envSlice
}
