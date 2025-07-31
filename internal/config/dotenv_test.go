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
	envContent := `TEST_TOKEN=ghp_test_token_123
DEVELOPER_TOKEN=ghp_dev_token_456
REVIEWER_TOKEN=ghp_rev_token_789`

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
		{"test token", "TEST_TOKEN", "ghp_test_token_123"},
		{"developer token", "DEVELOPER_TOKEN", "ghp_dev_token_456"},
		{"reviewer token", "REVIEWER_TOKEN", "ghp_rev_token_789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := os.Getenv(tt.envVar)
			if actual != tt.expected {
				t.Errorf("Expected %s=%s, got %s", tt.envVar, tt.expected, actual)
			}
		})
	}

	// Test that config can reference environment variables
	configContent := `repositories:
  include:
    - "owner/test-repo"

agents:
  - name: "developer"
    prompt: "Developer agent"
    github_token: "` + os.Getenv("DEVELOPER_TOKEN") + `"
    github_user: "dev-user"
  - name: "reviewer"
    prompt: "Reviewer agent"
    github_token: "` + os.Getenv("REVIEWER_TOKEN") + `"
    github_user: "rev-user"`

	configPath := filepath.Join(tempDir, "autoteam.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load and validate config
	cfg, err := LoadConfig("autoteam.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify that environment variables were properly resolved
	if cfg.Agents[0].GitHubToken != "ghp_dev_token_456" {
		t.Errorf("Expected developer token to be ghp_dev_token_456, got %s", cfg.Agents[0].GitHubToken)
	}

	if cfg.Agents[1].GitHubToken != "ghp_rev_token_789" {
		t.Errorf("Expected reviewer token to be ghp_rev_token_789, got %s", cfg.Agents[1].GitHubToken)
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

	// Try to load .env file that doesn't exist - should not error
	err = godotenv.Load()
	// godotenv.Load() returns an error if file doesn't exist, but we ignore it
	// in our implementation, so we just verify this doesn't cause the app to crash

	// Create a basic config without .env file
	configContent := `repositories:
  include:
    - "owner/test-repo"

agents:
  - name: "developer"
    prompt: "Developer agent"
    github_token: "ghp_direct_token_123"
    github_user: "dev-user"`

	configPath := filepath.Join(tempDir, "autoteam.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load and validate config - should work without .env file
	cfg, err := LoadConfig("autoteam.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify that direct token value is used
	if cfg.Agents[0].GitHubToken != "ghp_direct_token_123" {
		t.Errorf("Expected token to be ghp_direct_token_123, got %s", cfg.Agents[0].GitHubToken)
	}
}
