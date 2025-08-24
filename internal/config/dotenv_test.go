package config

import (
	"os"
	"path/filepath"
	"testing"

	"autoteam/internal/testutil"

	"github.com/joho/godotenv"
)

func TestDotenvSupport(t *testing.T) {
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

	// Create a .env file with test values
	envContent := `TEST_TEAM_NAME=custom-autoteam
DEVELOPER_DEBUG=true
API_ENDPOINT=http://localhost:8080`

	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to create .env file: %v", err)
	}

	// Load the .env file
	err = godotenv.Load()
	if err != nil {
		t.Fatalf("failed to load .env file: %v", err)
	}

	// Verify environment variables are loaded
	tests := []struct {
		name     string
		envVar   string
		expected string
	}{
		{"team name", "TEST_TEAM_NAME", "custom-autoteam"},
		{"debug flag", "DEVELOPER_DEBUG", "true"},
		{"API endpoint", "API_ENDPOINT", "http://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := os.Getenv(tt.envVar)
			if actual != tt.expected {
				t.Errorf("Expected %s=%s, got %s", tt.envVar, tt.expected, actual)
			}
		})
	}

	// Test that config can be created without GitHub-specific fields
	configContent := `workers:
  - name: "developer"
    prompt: "Developer agent"
  - name: "reviewer"
    prompt: "Reviewer agent"

settings:
  team_name: "custom-team"
  flow:
    - name: step1
      type: claude
      prompt: test`

	configPath := filepath.Join(tempDir, "autoteam.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load and validate config
	cfg, err := LoadConfig("autoteam.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify basic config properties
	if len(cfg.Workers) != 2 {
		t.Errorf("Expected 2 workers, got %d", len(cfg.Workers))
	}

	if cfg.Workers[0].Name != "developer" {
		t.Errorf("Expected first worker name to be 'developer', got %s", cfg.Workers[0].Name)
	}

	if cfg.Settings.GetTeamName() != "custom-team" {
		t.Errorf("Expected team name to be 'custom-team', got %s", cfg.Settings.GetTeamName())
	}
}

func TestDotenvOptional(t *testing.T) {
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

	// Create a config file without .env file present
	configContent := `workers:
  - name: "developer"
    prompt: "Developer agent"

settings:
  team_name: "test-team"
  flow:
    - name: step1
      type: claude
      prompt: test`

	configPath := filepath.Join(tempDir, "autoteam.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load dotenv (should not fail when .env doesn't exist)
	_ = godotenv.Load()

	// Load and validate config (should work without .env file)
	cfg, err := LoadConfig("autoteam.yaml")
	if err != nil {
		t.Fatalf("failed to load config without .env file: %v", err)
	}

	// Verify basic functionality
	if len(cfg.Workers) != 1 {
		t.Errorf("Expected 1 worker, got %d", len(cfg.Workers))
	}

	if cfg.Settings.GetTeamName() != "test-team" {
		t.Errorf("Expected team name to be 'test-team', got %s", cfg.Settings.GetTeamName())
	}
}
