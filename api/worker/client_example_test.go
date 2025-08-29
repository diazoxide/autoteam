package worker_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"autoteam/api/worker"
)

// Example showing how to use the generated client
func ExampleClient() {
	// Create a client for the worker API
	client, err := worker.NewClient("http://localhost:8080")
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// Get worker health status
	ctx := context.Background()
	response, err := client.GetHealth(ctx)
	if err != nil {
		fmt.Printf("Error getting health: %v\n", err)
		return
	}

	if response.StatusCode == http.StatusOK {
		fmt.Println("Worker is healthy!")
	}
}

// Example test demonstrating client usage
func TestClientGeneration(t *testing.T) {
	// Verify client can be created without errors
	client, err := worker.NewClient("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	t.Log("✓ Client generation successful")
}
