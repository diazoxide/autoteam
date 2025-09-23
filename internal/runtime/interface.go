package runtime

import (
	"context"
	"time"

	"autoteam/internal/config"
	"autoteam/internal/worker"
)

// ServiceStatus represents the status of a deployed service
type ServiceStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // running, stopped, error, etc.
	Health    string    `json:"health"` // healthy, unhealthy, unknown
	CreatedAt time.Time `json:"created_at"`
	Image     string    `json:"image,omitempty"`
	Ports     []string  `json:"ports,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// ContainerConfig represents configuration for a container deployment
type ContainerConfig struct {
	Name           string
	Image          string
	Environment    map[string]string
	Volumes        []string
	Ports          []string
	WorkingDir     string
	Entrypoint     []string
	Command        []string
	User           string
	NetworkName    string
	RestartPolicy  string
	ResourceLimits *config.ResourceLimits
}

// Runtime defines the interface for container runtime implementations
type Runtime interface {
	// Initialize sets up the runtime environment (networks, volumes, etc.)
	Initialize(ctx context.Context, cfg *config.Config) error

	// DeployWorker deploys a worker container with the given configuration
	DeployWorker(ctx context.Context, worker worker.Worker, settings worker.WorkerSettings, cfg *config.Config) error

	// DeployControlPlane deploys the control plane service
	DeployControlPlane(ctx context.Context, cfg *config.Config) error

	// DeployDashboard deploys the dashboard service
	DeployDashboard(ctx context.Context, cfg *config.Config) error

	// DeployService deploys a custom service from the services configuration
	DeployService(ctx context.Context, name string, serviceConfig map[string]interface{}, cfg *config.Config) error

	// StopWorker stops a specific worker container
	StopWorker(ctx context.Context, workerName string, cfg *config.Config) error

	// RestartWorker restarts a specific worker container
	RestartWorker(ctx context.Context, workerName string, cfg *config.Config) error

	// PauseWorker pauses a specific worker container
	PauseWorker(ctx context.Context, workerName string, cfg *config.Config) error

	// UnpauseWorker unpauses a specific worker container
	UnpauseWorker(ctx context.Context, workerName string, cfg *config.Config) error

	// GetWorkerStatus returns the status of a specific worker
	GetWorkerStatus(ctx context.Context, workerName string, cfg *config.Config) (*ServiceStatus, error)

	// StopControlPlane stops the control plane service
	StopControlPlane(ctx context.Context, cfg *config.Config) error

	// StopDashboard stops the dashboard service
	StopDashboard(ctx context.Context, cfg *config.Config) error

	// StopService stops a custom service
	StopService(ctx context.Context, serviceName string, cfg *config.Config) error

	// StopAll stops all running containers for the team
	StopAll(ctx context.Context, cfg *config.Config) error

	// GetStatus returns the status of all deployed services
	GetStatus(ctx context.Context, cfg *config.Config) ([]ServiceStatus, error)

	// GetLogs retrieves logs from a specific service
	GetLogs(ctx context.Context, serviceName string, lines int, cfg *config.Config) ([]string, error)

	// Cleanup removes all resources (containers, networks, volumes) for the team
	Cleanup(ctx context.Context, cfg *config.Config) error
}

// Factory creates a runtime instance based on the configuration
func NewRuntime(deploymentConfig *config.DeploymentConfig) (Runtime, error) {
	switch deploymentConfig.Runtime {
	case "docker":
		return NewDockerRuntime(deploymentConfig.Config)
	default:
		// For now, default to docker runtime
		return NewDockerRuntime(deploymentConfig.Config)
	}
}
