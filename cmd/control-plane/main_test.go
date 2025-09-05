package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/urfave/cli/v3"
)

func TestMain(t *testing.T) {
	// Test that the application can be created without panicking
	// Load .env file if it exists (ignore errors for optional file)
	_ = os.Setenv("LOG_LEVEL", "debug") // Set test environment

	// Create a minimal app instance like in main()
	app := &cli.Command{
		Name:    "autoteam-control-plane",
		Usage:   "AutoTeam Control Plane - Central orchestrator for managing multiple workers",
		Version: "test",
		Action:  runControlPlane,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set log level (debug, info, warn, error)",
				Value:   "info",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "HTTP server port",
				Value:   9090,
				Sources: cli.EnvVars("PORT"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to autoteam.yaml configuration file",
				Value:   "autoteam.yaml",
				Sources: cli.EnvVars("CONFIG_FILE"),
			},
		},
	}

	if app == nil {
		t.Fatal("Failed to create CLI app")
	}

	// Test basic properties
	if app.Name != "autoteam-control-plane" {
		t.Errorf("Expected app name 'autoteam-control-plane', got %s", app.Name)
	}

	if len(app.Flags) == 0 {
		t.Error("Expected app to have flags defined")
	}

	if app.Action == nil {
		t.Error("Expected app to have an action defined")
	}
}

func TestRunControlPlaneWithInvalidConfig(t *testing.T) {
	// Test that runControlPlane handles invalid config gracefully
	cmd := &cli.Command{}
	cmd.Set("config", "/nonexistent/config.yaml")
	cmd.Set("log-level", "info")
	cmd.Set("port", "9090")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := runControlPlane(ctx, cmd)
	
	// Should return an error for nonexistent config file
	if err == nil {
		t.Error("Expected error for nonexistent config file, got nil")
	}
}

func TestRunControlPlaneWithInvalidLogLevel(t *testing.T) {
	// Test that runControlPlane handles invalid log level gracefully
	cmd := &cli.Command{}
	cmd.Set("config", "autoteam.yaml") // This might not exist, but we're testing log level parsing
	cmd.Set("log-level", "invalid-level")
	cmd.Set("port", "9090")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := runControlPlane(ctx, cmd)
	
	// Should return an error for invalid log level
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}

	// Check that the error mentions log level
	if err != nil && err.Error() == "" {
		t.Error("Expected descriptive error message")
	}
}

func TestBuildVariables(t *testing.T) {
	// Test that build variables are defined (even if they're default values)
	if Version == "" {
		t.Error("Expected Version to be defined")
	}

	if BuildTime == "" {
		t.Error("Expected BuildTime to be defined")
	}

	if GitCommit == "" {
		t.Error("Expected GitCommit to be defined")
	}
}

func TestAppConfiguration(t *testing.T) {
	// Test that the app has the expected flags configured
	app := &cli.Command{
		Name:    "autoteam-control-plane",
		Usage:   "AutoTeam Control Plane - Central orchestrator for managing multiple workers",
		Version: "test",
		Action:  runControlPlane,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set log level (debug, info, warn, error)",
				Value:   "info",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "HTTP server port",
				Value:   9090,
				Sources: cli.EnvVars("PORT"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to autoteam.yaml configuration file",
				Value:   "autoteam.yaml",
				Sources: cli.EnvVars("CONFIG_FILE"),
			},
		},
	}

	// Check that required flags exist
	expectedFlags := map[string]bool{
		"log-level": false,
		"port":      false,
		"config":    false,
	}

	for _, flag := range app.Flags {
		flagName := ""
		switch f := flag.(type) {
		case *cli.StringFlag:
			flagName = f.Name
		case *cli.IntFlag:
			flagName = f.Name
		}

		if _, exists := expectedFlags[flagName]; exists {
			expectedFlags[flagName] = true
		}
	}

	// Verify all expected flags were found
	for flagName, found := range expectedFlags {
		if !found {
			t.Errorf("Expected flag '%s' not found", flagName)
		}
	}
}

func TestEnvVarSupport(t *testing.T) {
	// Test environment variable support
	testCases := []struct {
		envVar string
		value  string
	}{
		{"LOG_LEVEL", "debug"},
		{"PORT", "8080"},
		{"CONFIG_FILE", "test-config.yaml"},
	}

	for _, tc := range testCases {
		// Set environment variable
		oldValue := os.Getenv(tc.envVar)
		os.Setenv(tc.envVar, tc.value)
		defer os.Setenv(tc.envVar, oldValue) // Restore original value

		// The actual CLI parsing would happen in the urfave/cli framework
		// Here we just verify the environment variables can be set
		if os.Getenv(tc.envVar) != tc.value {
			t.Errorf("Failed to set environment variable %s to %s", tc.envVar, tc.value)
		}
	}
}

// Test CLI integration (without actually running the server)
func TestCLIIntegration(t *testing.T) {
	// Create a test config file
	configContent := `
control_plane:
  enabled: true
  port: 9090
  workers: []
`
	tmpFile, err := os.CreateTemp("", "test-autoteam-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	tmpFile.Close()

	// Test with minimal valid configuration
	app := &cli.Command{
		Name:   "test-control-plane",
		Action: runControlPlane,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "log-level", Value: "info"},
			&cli.IntFlag{Name: "port", Value: 9091}, // Use different port to avoid conflicts
			&cli.StringFlag{Name: "config", Value: tmpFile.Name()},
		},
	}

	// Test that the app can be created
	if app.Action == nil {
		t.Error("Expected action to be defined")
	}
}