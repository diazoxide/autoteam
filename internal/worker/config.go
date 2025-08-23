package worker

import (
	"autoteam/internal/config"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadWorkerFromFile loads worker configuration directly from a YAML file
func LoadWorkerFromFile(configPath string) (*config.Worker, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var worker config.Worker
	if err := yaml.Unmarshal(data, &worker); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Validate worker configuration
	if worker.Name == "" {
		return nil, fmt.Errorf("worker name is required")
	}

	return &worker, nil
}
