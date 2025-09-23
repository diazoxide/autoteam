package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"autoteam/internal/config"
	"autoteam/internal/embedded"
	"autoteam/internal/logger"
	"autoteam/internal/worker"

	"github.com/containerd/errdefs"
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
	var clientOpts []client.Opt

	// Configure Docker host
	if dockerHost, ok := runtimeConfig["docker_host"].(string); ok && dockerHost != "" {
		clientOpts = append(clientOpts, client.WithHost(dockerHost))
	}

	// Configure API version
	if apiVersion, ok := runtimeConfig["api_version"].(string); ok && apiVersion != "" {
		clientOpts = append(clientOpts, client.WithVersion(apiVersion))
	} else {
		// Default to API version negotiation
		clientOpts = append(clientOpts, client.WithAPIVersionNegotiation())
	}

	// Configure TLS settings
	if tlsVerify, ok := runtimeConfig["tls_verify"].(bool); ok && tlsVerify {
		if certPath, ok := runtimeConfig["cert_path"].(string); ok && certPath != "" {
			clientOpts = append(clientOpts, client.WithTLSClientConfig(certPath, "", ""))
		}
	}

	// Configure timeout
	if timeoutSeconds, ok := runtimeConfig["timeout"].(float64); ok && timeoutSeconds > 0 {
		timeout := time.Duration(timeoutSeconds) * time.Second
		clientOpts = append(clientOpts, client.WithTimeout(timeout))
	}

	// If no custom configuration is provided, use environment variables
	if len(clientOpts) == 0 {
		clientOpts = append(clientOpts, client.FromEnv, client.WithAPIVersionNegotiation())
	}

	cli, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Get default image name from config or use default
	imageName := "alpine:latest"
	if configImageName, ok := runtimeConfig["default_image"].(string); ok && configImageName != "" {
		imageName = configImageName
	}

	return &DockerRuntime{
		client:    cli,
		config:    runtimeConfig,
		imageName: imageName,
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

// RestartWorker restarts a specific worker container
func (d *DockerRuntime) RestartWorker(ctx context.Context, workerName string, cfg *config.Config) error {
	log := logger.FromContext(ctx)
	containerName := d.getWorkerContainerNameByName(workerName, cfg)

	timeout := int(30) // 30 seconds
	if err := d.client.ContainerRestart(ctx, containerName, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to restart worker container %s: %w", containerName, err)
	}

	log.Info("Worker restarted successfully", zap.String("worker", workerName), zap.String("container", containerName))
	return nil
}

// PauseWorker pauses a specific worker container
func (d *DockerRuntime) PauseWorker(ctx context.Context, workerName string, cfg *config.Config) error {
	log := logger.FromContext(ctx)
	containerName := d.getWorkerContainerNameByName(workerName, cfg)

	if err := d.client.ContainerPause(ctx, containerName); err != nil {
		return fmt.Errorf("failed to pause worker container %s: %w", containerName, err)
	}

	log.Info("Worker paused successfully", zap.String("worker", workerName), zap.String("container", containerName))
	return nil
}

// UnpauseWorker unpauses a specific worker container
func (d *DockerRuntime) UnpauseWorker(ctx context.Context, workerName string, cfg *config.Config) error {
	log := logger.FromContext(ctx)
	containerName := d.getWorkerContainerNameByName(workerName, cfg)

	if err := d.client.ContainerUnpause(ctx, containerName); err != nil {
		return fmt.Errorf("failed to unpause worker container %s: %w", containerName, err)
	}

	log.Info("Worker unpaused successfully", zap.String("worker", workerName), zap.String("container", containerName))
	return nil
}

// GetWorkerStatus returns the status of a specific worker
func (d *DockerRuntime) GetWorkerStatus(ctx context.Context, workerName string, cfg *config.Config) (*ServiceStatus, error) {
	containerName := d.getWorkerContainerNameByName(workerName, cfg)

	inspect, err := d.client.ContainerInspect(ctx, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return &ServiceStatus{
				Name:   workerName,
				Status: "not_deployed",
				Health: "unknown",
			}, nil
		}
		return nil, fmt.Errorf("failed to inspect worker container %s: %w", containerName, err)
	}

	// Parse the created timestamp from string to time.Time
	createdTime, err := time.Parse(time.RFC3339, inspect.Created)
	if err != nil {
		// Fallback to zero time if parsing fails
		createdTime = time.Time{}
	}

	status := &ServiceStatus{
		Name:      workerName,
		Status:    d.getContainerStatusString(inspect.State),
		Health:    d.getContainerHealthString(inspect.State),
		CreatedAt: createdTime,
		Image:     inspect.Config.Image,
	}

	// Add port mappings if available
	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Ports != nil {
		for containerPort, hostBindings := range inspect.NetworkSettings.Ports {
			for _, binding := range hostBindings {
				if binding.HostPort != "" {
					status.Ports = append(status.Ports, fmt.Sprintf("%s:%s", binding.HostPort, containerPort))
				}
			}
		}
	}

	return status, nil
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
	teamName := config.DefaultTeamName
	if cfg != nil {
		teamName = cfg.GetTeamName()
	}
	return fmt.Sprintf("%s-network", teamName)
}

func (d *DockerRuntime) getDockerSocketPath() string {
	if socketPath, ok := d.config["docker_socket"].(string); ok && socketPath != "" {
		return socketPath
	}
	// Default Docker socket path
	return "/var/run/docker.sock"
}

func (d *DockerRuntime) getHostWorkingDirectory() string {
	// Check for environment variable set by control plane container
	if hostDir := os.Getenv("HOST_WORKING_DIR"); hostDir != "" {
		return hostDir
	}
	// Check configuration
	if hostDir, ok := d.config["host_working_dir"].(string); ok && hostDir != "" {
		return hostDir
	}
	// Default to current working directory
	if currentDir, err := os.Getwd(); err == nil {
		return currentDir
	}
	// Fallback to /opt/autoteam if we can't get working directory
	return "/opt/autoteam"
}

func (d *DockerRuntime) getWorkerContainerName(w worker.Worker, cfg *config.Config) string {
	teamName := config.DefaultTeamName
	if cfg != nil {
		teamName = cfg.GetTeamName()
	}
	return fmt.Sprintf("%s-%s", teamName, w.GetNormalizedName())
}

func (d *DockerRuntime) getWorkerContainerNameByName(workerName string, cfg *config.Config) string {
	// Normalize the worker name the same way as in worker package
	normalized := strings.ToLower(strings.ReplaceAll(workerName, " ", "_"))
	teamName := config.DefaultTeamName
	if cfg != nil {
		teamName = cfg.GetTeamName()
	}
	return fmt.Sprintf("%s-%s", teamName, normalized)
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

	log.Info("Ensuring binaries are available for container deployment")

	// Create local .autoteam/bin directory if it doesn't exist
	if err := os.MkdirAll(".autoteam/bin", 0755); err != nil {
		return fmt.Errorf("failed to create .autoteam/bin directory: %w", err)
	}

	// Check if we have embedded binaries available
	hasEmbeddedBinaries := false
	containerPlatforms := []embedded.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "linux", Arch: "386"},
		{OS: "linux", Arch: "arm"},
	}

	for _, platform := range containerPlatforms {
		if embedded.IsBinaryAvailable(embedded.Worker, platform) {
			hasEmbeddedBinaries = true
			break
		}
	}

	if !hasEmbeddedBinaries {
		log.Info("No embedded binaries available in this build - checking for built binaries to extract")

		// First, check if binaries already exist in .autoteam/bin
		existingBinaries := 0
		for _, platform := range containerPlatforms {
			binaryName := embedded.GetBinaryName(embedded.Worker, platform)
			if _, err := os.Stat(fmt.Sprintf(".autoteam/bin/%s", binaryName)); err == nil {
				existingBinaries++
			}
		}

		if existingBinaries > 0 {
			log.Info("Found existing binaries in .autoteam/bin", zap.Int("count", existingBinaries))
			return nil
		}

		// Try to extract binaries from build/ directory
		log.Info("Extracting binaries from build/ directory")
		extractedCount := 0

		binaryTypes := []embedded.BinaryType{
			embedded.Worker,
			embedded.ControlPlane,
			embedded.Dashboard,
		}

		for _, platform := range containerPlatforms {
			for _, binaryType := range binaryTypes {
				binaryName := embedded.GetBinaryName(binaryType, platform)
				buildPath := fmt.Sprintf("build/%s", binaryName)
				destPath := fmt.Sprintf(".autoteam/bin/%s", binaryName)

				// Check if binary exists in build/ directory
				if _, err := os.Stat(buildPath); err == nil {
					// Copy binary to .autoteam/bin/
					if err := copyFile(buildPath, destPath); err != nil {
						log.Warn("Failed to copy binary from build directory",
							zap.String("source", buildPath),
							zap.String("dest", destPath),
							zap.Error(err))
						continue
					}

					// Make binary executable
					if err := os.Chmod(destPath, 0755); err != nil {
						log.Warn("Failed to make binary executable",
							zap.String("path", destPath),
							zap.Error(err))
					}

					extractedCount++
					log.Debug("Extracted binary from build directory",
						zap.String("type", string(binaryType)),
						zap.String("platform", platform.String()),
						zap.String("source", buildPath),
						zap.String("dest", destPath))
				}
			}
		}

		// Extract entrypoint script if it exists
		entrypointSource := "scripts/entrypoint.sh"
		entrypointDest := ".autoteam/bin/entrypoint.sh"
		if _, err := os.Stat(entrypointSource); err == nil {
			if err := copyFile(entrypointSource, entrypointDest); err == nil {
				if err := os.Chmod(entrypointDest, 0755); err == nil {
					extractedCount++
					log.Debug("Extracted entrypoint script", zap.String("dest", entrypointDest))
				}
			}
		}

		if extractedCount == 0 {
			return fmt.Errorf("no binaries found in build/ directory and no embedded binaries available.\n" +
				"Solutions:\n" +
				"  1. Build binaries first: make build-worker-all build-control-plane-all build-dashboard-all\n" +
				"  2. Build with embedded binaries: make build-embedded && use autoteam-embedded command\n" +
				"  3. Use the installation script which includes all required binaries")
		}

		// Create generic symlinks for container compatibility
		// Containers expect generic names like "autoteam-worker", but we have platform-specific names
		genericLinksCreated := 0
		for _, binaryType := range binaryTypes {
			// Find the amd64 version as the default (most common)
			platformBinaryName := embedded.GetBinaryName(binaryType, embedded.Platform{OS: "linux", Arch: "amd64"})
			genericBinaryName := fmt.Sprintf("autoteam-%s", binaryType)

			platformPath := fmt.Sprintf(".autoteam/bin/%s", platformBinaryName)
			genericPath := fmt.Sprintf(".autoteam/bin/%s", genericBinaryName)

			// Check if platform-specific binary exists
			if _, err := os.Stat(platformPath); err == nil {
				// Create a copy with generic name (symlinks don't work well with Docker volumes)
				if err := copyFile(platformPath, genericPath); err == nil {
					if err := os.Chmod(genericPath, 0755); err == nil {
						genericLinksCreated++
						log.Debug("Created generic binary copy",
							zap.String("type", string(binaryType)),
							zap.String("source", platformPath),
							zap.String("dest", genericPath))
					}
				}
			}
		}

		log.Info("Successfully extracted binaries from build directory",
			zap.Int("extracted_files", extractedCount),
			zap.Int("generic_copies", genericLinksCreated))
		return nil
	}

	log.Info("Extracting embedded binaries for container deployment")

	// Extract all binary types for Linux platforms
	binaryTypes := []embedded.BinaryType{
		embedded.Worker,
		embedded.ControlPlane,
		embedded.Dashboard,
	}

	extractedCount := 0
	for _, platform := range containerPlatforms {
		for _, binaryType := range binaryTypes {
			if !embedded.IsBinaryAvailable(binaryType, platform) {
				log.Debug("Binary not available for platform",
					zap.String("type", string(binaryType)),
					zap.String("platform", platform.String()))
				continue
			}

			binaryName := embedded.GetBinaryName(binaryType, platform)
			localPath := fmt.Sprintf(".autoteam/bin/%s", binaryName)

			// Check if binary already exists and skip if it does
			if _, err := os.Stat(localPath); err == nil {
				log.Debug("Binary already exists, skipping extraction",
					zap.String("binary", binaryName))
				continue
			}

			// Extract embedded binary to local bin directory
			if err := embedded.ExtractBinary(binaryType, platform, localPath); err != nil {
				log.Warn("Failed to extract embedded binary",
					zap.String("type", string(binaryType)),
					zap.String("platform", platform.String()),
					zap.Error(err))
				continue
			}

			extractedCount++
			log.Debug("Extracted embedded binary",
				zap.String("type", string(binaryType)),
				zap.String("platform", platform.String()),
				zap.String("path", localPath))
		}
	}

	// Extract entrypoint script
	entrypointPath := ".autoteam/bin/entrypoint.sh"
	if _, err := os.Stat(entrypointPath); os.IsNotExist(err) {
		if embedded.IsScriptAvailable(embedded.EntrypointScript) {
			if err := embedded.ExtractScript(embedded.EntrypointScript, entrypointPath); err != nil {
				log.Warn("Failed to extract entrypoint script", zap.Error(err))
			} else {
				extractedCount++
				log.Debug("Extracted entrypoint script", zap.String("path", entrypointPath))
			}
		} else {
			log.Warn("Entrypoint script not available in embedded assets")
		}
	}

	// Create generic symlinks for container compatibility (embedded path)
	// Containers expect generic names like "autoteam-worker", but we have platform-specific names
	genericLinksCreated := 0
	for _, binaryType := range binaryTypes {
		// Find the amd64 version as the default (most common)
		platformBinaryName := embedded.GetBinaryName(binaryType, embedded.Platform{OS: "linux", Arch: "amd64"})
		genericBinaryName := fmt.Sprintf("autoteam-%s", binaryType)

		platformPath := fmt.Sprintf(".autoteam/bin/%s", platformBinaryName)
		genericPath := fmt.Sprintf(".autoteam/bin/%s", genericBinaryName)

		// Check if platform-specific binary exists
		if _, err := os.Stat(platformPath); err == nil {
			// Create a copy with generic name (symlinks don't work well with Docker volumes)
			if err := copyFile(platformPath, genericPath); err == nil {
				if err := os.Chmod(genericPath, 0755); err == nil {
					genericLinksCreated++
					log.Debug("Created generic binary copy from embedded",
						zap.String("type", string(binaryType)),
						zap.String("source", platformPath),
						zap.String("dest", genericPath))
				}
			}
		}
	}

	log.Info("Embedded binary extraction completed",
		zap.Int("extracted_files", extractedCount),
		zap.Int("generic_copies", genericLinksCreated))

	return nil
}

func copyFile(src, dst string) error {
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

	// Workers now load configuration directly from database - no config files needed

	// Generate control plane config
	if cfg.ControlPlane != nil && cfg.ControlPlane.Enabled {
		if err := d.generateControlPlaneConfig(cfg); err != nil {
			return fmt.Errorf("failed to generate control plane config: %w", err)
		}
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

	// Note: Workers are now managed through database, not through config file
	// Control plane will load workers from database at runtime

	// Create control plane config (workers loaded from database, not config)
	controlPlaneConfig := map[string]interface{}{
		"enabled": cfg.ControlPlane.Enabled,
		"port":    cfg.ControlPlane.Port,
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
		"AUTOTEAM_WORKER_ID":              w.ID.String(),
		"AUTOTEAM_WORKER_NAME":            w.Name,
		"AUTOTEAM_WORKER_NORMALIZED_NAME": w.GetNormalizedName(),
		"DEBUG":                           debugValue,
		"LOG_LEVEL":                       logLevelValue,
		"GRPC_PORT":                       "8080",
	}

	// Prepare worker configuration as JSON to pass to worker
	workerConfig := map[string]interface{}{
		"worker":   w,
		"settings": settings,
	}

	// Marshal configuration to JSON
	configJSON, err := json.Marshal(workerConfig)
	if err != nil {
		// Log error and continue with empty config
		log := logger.FromContext(context.Background())
		log.Error("Failed to marshal worker configuration to JSON", zap.Error(err))
		configJSON = []byte("{}")
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

	// Build volumes with absolute paths - workers don't need database access
	hostDir := d.getHostWorkingDirectory()
	volumes := []string{
		fmt.Sprintf("%s/.autoteam/bin:/opt/autoteam/bin", hostDir),
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
	user := "root" // default
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
		Entrypoint:    []string{"/opt/autoteam/bin/autoteam-worker"},
		Command:       []string{"--config-json", string(configJSON)},
		WorkingDir:    "/opt/autoteam",
		User:          user,
		NetworkName:   d.getNetworkName(cfg),
		RestartPolicy: "unless-stopped",
	}
}

func (d *DockerRuntime) buildControlPlaneContainerConfig(cfg *config.Config) *ContainerConfig {
	teamName := cfg.GetTeamName()

	hostDir := d.getHostWorkingDirectory()
	environment := map[string]string{
		"CONTROL_PLANE_CONFIG": "/opt/autoteam/control-plane/config.yaml",
		"HOST_WORKING_DIR":     hostDir, // Pass host working directory to control plane container
	}

	// Add database configuration if available
	if cfg.Settings.Database != nil {
		if cfg.Settings.Database.Type != "" {
			environment["DATABASE_TYPE"] = string(cfg.Settings.Database.Type)
		}
		if cfg.Settings.Database.DSN != "" {
			// Map the DSN to the container path
			environment["DATABASE_DSN"] = "/opt/autoteam/autoteam-debug.db"
		}
	}

	volumes := []string{
		fmt.Sprintf("%s/.autoteam/%s/control-plane:/opt/autoteam/control-plane", hostDir, teamName),
		fmt.Sprintf("%s/.autoteam/bin:/opt/autoteam/bin", hostDir),
		fmt.Sprintf("%s:/var/run/docker.sock", d.getDockerSocketPath()), // Docker socket for container management
		fmt.Sprintf("%s:/opt/autoteam/host", hostDir),                   // Mount host directory to access config files
	}

	// Add database file volume mount if database configuration exists
	if cfg.Settings.Database != nil && cfg.Settings.Database.DSN != "" {
		// Extract database file path and mount it
		volumes = append(volumes, fmt.Sprintf("%s/autoteam-debug.db:/opt/autoteam/autoteam-debug.db", hostDir))
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
		Command:       []string{"--config", "/opt/autoteam/host/autoteam.debug.yaml", "--log-level", "info"},
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
		fmt.Sprintf("%s/.autoteam/bin/autoteam-dashboard:/autoteam-dashboard:ro", currentDir),
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

// getContainerStatusString converts Docker container state to readable status
func (d *DockerRuntime) getContainerStatusString(state *container.State) string {
	if state.Running {
		if state.Paused {
			return "paused"
		}
		return "running"
	}
	if state.Dead {
		return "dead"
	}
	if state.Restarting {
		return "restarting"
	}
	if state.ExitCode != 0 {
		return "error"
	}
	return "stopped"
}

// getContainerHealthString converts Docker container health to readable status
func (d *DockerRuntime) getContainerHealthString(state *container.State) string {
	if state.Health != nil {
		switch state.Health.Status {
		case "healthy":
			return "healthy"
		case "unhealthy":
			return "unhealthy"
		case "starting":
			return "starting"
		default:
			return "unknown"
		}
	}
	if state.Running && !state.Paused {
		return "healthy"
	}
	return "unknown"
}
