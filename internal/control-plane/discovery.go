package controlplane

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"autoteam/internal/worker"

	"gopkg.in/yaml.v3"
)

// DiscoveredWorker represents a worker discovered from the filesystem
type DiscoveredWorker struct {
	ID     string
	Name   string
	URL    string
	APIKey string
	Config *worker.Worker
}

// DiscoverWorkers scans the workers directory and discovers worker configurations
func DiscoverWorkers(workersDir string) ([]DiscoveredWorker, error) {
	var discoveredWorkers []DiscoveredWorker

	// Check if workers directory exists
	if _, err := os.Stat(workersDir); os.IsNotExist(err) {
		return discoveredWorkers, nil // No workers directory, return empty list
	}

	// Read workers directory
	entries, err := os.ReadDir(workersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workers directory %s: %w", workersDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workerID := entry.Name()
		configPath := filepath.Join(workersDir, workerID, "config.yaml")

		// Check if config.yaml exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue // Skip directories without config.yaml
		}

		// Load worker configuration
		workerConfig, err := loadWorkerConfig(configPath)
		if err != nil {
			// Log warning but continue with other workers
			continue
		}

		// Extract HTTP port from worker settings
		httpPort := 8080 // Default port
		if workerConfig.Settings != nil && workerConfig.Settings.HTTPPort != nil {
			httpPort = *workerConfig.Settings.HTTPPort
		}

		// Create discovered worker
		discovered := DiscoveredWorker{
			ID:     workerID,
			Name:   workerConfig.Name,
			URL:    fmt.Sprintf("http://localhost:%d", httpPort),
			APIKey: "", // Workers currently don't use API keys
			Config: workerConfig,
		}

		discoveredWorkers = append(discoveredWorkers, discovered)
	}

	return discoveredWorkers, nil
}

// loadWorkerConfig loads a worker configuration from a YAML file
func loadWorkerConfig(configPath string) (*worker.Worker, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read worker config file %s: %w", configPath, err)
	}

	var workerConfig worker.Worker
	if err := yaml.Unmarshal(data, &workerConfig); err != nil {
		return nil, fmt.Errorf("failed to parse worker config YAML %s: %w", configPath, err)
	}

	return &workerConfig, nil
}

// GetWorkerIDFromName generates a worker ID from a worker name (for consistency)
func GetWorkerIDFromName(name string) string {
	// Convert to lowercase and replace spaces with underscores
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "_")
	// Remove any non-alphanumeric characters except underscores and hyphens
	var result strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
