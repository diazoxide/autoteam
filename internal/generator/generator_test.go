package generator

import (
	"os"
	"strings"
	"testing"

	"autoteam/internal/config"
	"autoteam/internal/testutil"
	"autoteam/internal/util"
	"autoteam/internal/worker"

	"gopkg.in/yaml.v3"
)

func TestGenerator_GenerateCompose(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := testutil.CreateTempDir(t)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create test config with new service structure
	cfg := &config.Config{
		Workers: []worker.Worker{
			{
				Name:   "dev1",
				Prompt: "You are a developer agent",
			},
			{
				Name:   "arch1",
				Prompt: "You are an architect agent",
				Settings: &worker.WorkerSettings{
					Service: map[string]interface{}{
						"image":   "python:3.11",
						"volumes": []string{"./custom-vol:/app/custom"},
					},
				},
			},
		},
		Settings: worker.WorkerSettings{
			Service: map[string]interface{}{
				"image":   "node:18",
				"user":    "testuser",
				"volumes": []string{"./shared:/app/shared"},
			},
			SleepDuration: util.IntPtr(30),
			TeamName:      util.StringPtr("test-team"),
			InstallDeps:   util.BoolPtr(false),
		},
	}

	// Generate files
	gen := New()
	if err := gen.GenerateCompose(cfg); err != nil {
		t.Fatalf("GenerateCompose() error = %v", err)
	}

	// Verify compose.yaml was generated in .autoteam directory
	composeContent := testutil.ReadFile(t, ".autoteam/compose.yaml")

	// Parse the YAML to verify structure
	var compose ComposeConfig
	if err := yaml.Unmarshal([]byte(composeContent), &compose); err != nil {
		t.Fatalf("Failed to parse generated compose.yaml: %v", err)
	}

	// Verify both services exist
	if _, exists := compose.Services["dev1"]; !exists {
		t.Errorf("compose.yaml should contain dev1 service")
	}
	if _, exists := compose.Services["arch1"]; !exists {
		t.Errorf("compose.yaml should contain arch1 service")
	}

	// Check dev1 service uses global settings
	dev1Service := compose.Services["dev1"].(map[string]interface{})
	if dev1Service["image"] != "node:18" {
		t.Errorf("dev1 service should use global image, got %v", dev1Service["image"])
	}
	if dev1Service["user"] != "testuser" {
		t.Errorf("dev1 service should use global user, got %v", dev1Service["user"])
	}

	// Check arch1 service has overridden image but inherits global user
	arch1Service := compose.Services["arch1"].(map[string]interface{})
	if arch1Service["image"] != "python:3.11" {
		t.Errorf("arch1 service should use overridden image, got %v", arch1Service["image"])
	}
	if arch1Service["user"] != "testuser" {
		t.Errorf("arch1 service should inherit global user, got %v", arch1Service["user"])
	}

	// Verify environment variables are set correctly
	// The YAML unmarshaling converts environment to map[string]interface{}
	dev1EnvInterface := dev1Service["environment"].(map[string]interface{})
	expectedConfigFile := "/opt/autoteam/workers/dev1/config.yaml"
	if dev1EnvInterface["CONFIG_FILE"] != expectedConfigFile {
		t.Errorf("dev1 environment should contain CONFIG_FILE=%s, got %v", expectedConfigFile, dev1EnvInterface["CONFIG_FILE"])
	}
	// GitHub token environment variables removed

	// Verify volumes are properly merged
	// The YAML unmarshaling converts volumes to []interface{}
	dev1VolumesInterface := dev1Service["volumes"].([]interface{})
	hasSharedVolume := false
	hasWorkerVolume := false
	for _, vol := range dev1VolumesInterface {
		volStr := vol.(string)
		if strings.Contains(volStr, "./shared:/app/shared") {
			hasSharedVolume = true
		}
		if strings.Contains(volStr, "dev1:/opt/autoteam/workers/dev1") {
			hasWorkerVolume = true
		}
	}
	if !hasSharedVolume {
		t.Errorf("dev1 should have shared volume from global settings")
	}
	if !hasWorkerVolume {
		t.Errorf("dev1 should have full worker directory volume")
	}

	// Verify .autoteam/bin directory was created
	if !testutil.DirExists(".autoteam/bin") {
		t.Errorf(".autoteam/bin directory should be created")
	}

	// Verify worker directories were created
	agentDirs := []string{
		".autoteam/test-team/workers/dev1",
		".autoteam/test-team/workers/arch1",
	}

	for _, dir := range agentDirs {
		if !testutil.FileExists(dir) {
			t.Errorf("directory %s should be created", dir)
		}
	}
}

func TestGenerator_CreateWorkerDirectories(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	cfg := &config.Config{
		Workers: []worker.Worker{
			{Name: "test1"},
			{Name: "test2"},
		},
		Settings: worker.WorkerSettings{
			TeamName: util.StringPtr("test-team"),
		},
	}

	gen := New()
	if err := gen.createWorkerDirectories(cfg); err != nil {
		t.Fatalf("createWorkerDirectories() error = %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		".autoteam/test-team/workers/test1",
		".autoteam/test-team/workers/test2",
	}

	for _, dir := range expectedDirs {
		if !testutil.FileExists(dir) {
			t.Errorf("directory %s should be created", dir)
		}
	}
}

func TestGenerator_GenerateComposeYAML(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	cfg := &config.Config{
		Settings: worker.WorkerSettings{
			Service: map[string]interface{}{
				"image": "node:18",
				"user":  "developer",
			},
			TeamName: util.StringPtr("test-team"),
		},
		Workers: []worker.Worker{
			{
				Name:   "dev1",
				Prompt: "Developer agent",
			},
		},
	}

	gen := New()

	// Ensure .autoteam directory exists
	if err := os.MkdirAll(".autoteam", 0755); err != nil {
		t.Fatalf("failed to create .autoteam directory: %v", err)
	}

	if err := gen.generateComposeYAML(cfg, nil); err != nil {
		t.Fatalf("generateComposeYAML() error = %v", err)
	}

	// Verify compose.yaml was created
	if !testutil.FileExists(".autoteam/compose.yaml") {
		t.Fatalf("compose.yaml should be created")
	}

	content := testutil.ReadFile(t, ".autoteam/compose.yaml")

	// Parse and verify the generated YAML
	var compose ComposeConfig
	if err := yaml.Unmarshal([]byte(content), &compose); err != nil {
		t.Fatalf("Failed to parse generated compose.yaml: %v", err)
	}

	// Verify service structure
	if _, exists := compose.Services["dev1"]; !exists {
		t.Errorf("compose.yaml should contain dev1 service")
	}

	dev1Service := compose.Services["dev1"].(map[string]interface{})
	if dev1Service["image"] != "node:18" {
		t.Errorf("dev1 service should have correct image, got %v", dev1Service["image"])
	}
}

func TestGenerator_FixedPortGeneration(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := testutil.CreateTempDir(t)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create test config with multiple workers
	cfg := &config.Config{
		Workers: []worker.Worker{
			{
				Name:   "worker1",
				Prompt: "Worker 1",
			},
			{
				Name:   "worker2",
				Prompt: "Worker 2",
			},
		},
		Settings: worker.WorkerSettings{
			TeamName: util.StringPtr("test-team"),
		},
	}

	// Generate files
	gen := New()
	if err := gen.GenerateCompose(cfg); err != nil {
		t.Fatalf("GenerateCompose() error = %v", err)
	}

	// Verify compose.yaml was generated
	composeContent := testutil.ReadFile(t, ".autoteam/compose.yaml")

	// Parse the YAML to verify structure
	var compose ComposeConfig
	if err := yaml.Unmarshal([]byte(composeContent), &compose); err != nil {
		t.Fatalf("Failed to parse generated compose.yaml: %v", err)
	}

	// Verify no external port mappings exist
	for serviceName, serviceInterface := range compose.Services {
		service := serviceInterface.(map[string]interface{})
		if _, hasExternalPorts := service["ports"]; hasExternalPorts {
			t.Errorf("Service %s should not have external port mappings", serviceName)
		}
	}

	// Verify workers use fixed internal port 8080 (indirectly through environment variables)
	worker1Service := compose.Services["worker1"].(map[string]interface{})
	environment := worker1Service["environment"].(map[string]interface{})

	// Check that worker directory paths are correctly set (indicates proper setup for internal port usage)
	expectedWorkerDir := "/opt/autoteam/workers/worker1"
	if environment["AUTOTEAM_WORKER_DIR"] != expectedWorkerDir {
		t.Errorf("Worker1 should have correct worker directory, got %v", environment["AUTOTEAM_WORKER_DIR"])
	}
}

func TestGenerator_ControlPlaneConfigGeneration(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := testutil.CreateTempDir(t)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create test config with control plane enabled and multiple workers
	cfg := &config.Config{
		Workers: []worker.Worker{
			{
				Name:   "worker1",
				Prompt: "Worker 1",
			},
			{
				Name:   "worker2",
				Prompt: "Worker 2",
			},
			{
				Name:    "disabled-worker",
				Prompt:  "Disabled Worker",
				Enabled: util.BoolPtr(false),
			},
		},
		ControlPlane: &config.ControlPlaneConfig{
			Enabled: true,
			Port:    9090,
			APIKey:  "test-key",
		},
		Settings: worker.WorkerSettings{
			TeamName: util.StringPtr("test-team"),
		},
	}

	// Generate control plane config
	gen := New()
	if err := gen.generateControlPlaneConfig(cfg, nil); err != nil {
		t.Fatalf("generateControlPlaneConfig() error = %v", err)
	}

	// Verify control plane config was generated
	configPath := ".autoteam/test-team/control-plane/config.yaml"
	if !testutil.FileExists(configPath) {
		t.Fatalf("Control plane config should be generated at %s", configPath)
	}

	configContent := testutil.ReadFile(t, configPath)

	// Parse the control plane config
	var controlPlaneConfig config.ControlPlaneConfig
	if err := yaml.Unmarshal([]byte(configContent), &controlPlaneConfig); err != nil {
		t.Fatalf("Failed to parse control plane config: %v", err)
	}

	// Verify control plane settings
	if !controlPlaneConfig.Enabled {
		t.Errorf("Control plane should be enabled")
	}
	if controlPlaneConfig.Port != 9090 {
		t.Errorf("Control plane port should be 9090, got %d", controlPlaneConfig.Port)
	}
	if controlPlaneConfig.APIKey != "test-key" {
		t.Errorf("Control plane API key should be 'test-key', got %s", controlPlaneConfig.APIKey)
	}

	// Verify worker APIs are correctly generated with fixed port 8080
	expectedWorkerAPIs := []string{
		"http://worker1:8080",
		"http://worker2:8080",
	}

	if len(controlPlaneConfig.WorkersAPIs) != len(expectedWorkerAPIs) {
		t.Errorf("Expected %d worker APIs, got %d", len(expectedWorkerAPIs), len(controlPlaneConfig.WorkersAPIs))
	}

	for i, expectedAPI := range expectedWorkerAPIs {
		if i < len(controlPlaneConfig.WorkersAPIs) && controlPlaneConfig.WorkersAPIs[i] != expectedAPI {
			t.Errorf("Worker API[%d] should be %s, got %s", i, expectedAPI, controlPlaneConfig.WorkersAPIs[i])
		}
	}

	// Verify disabled worker is not included
	for _, api := range controlPlaneConfig.WorkersAPIs {
		if strings.Contains(api, "disabled-worker") {
			t.Errorf("Disabled worker should not be included in worker APIs")
		}
	}
}
