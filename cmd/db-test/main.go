package main

import (
	"context"
	"fmt"
	"log"

	"autoteam/internal/database"
	"autoteam/internal/logger"
	"autoteam/internal/worker"
)

func main() {
	fmt.Println("Testing database integration...")

	// Setup logging context
	ctx, err := logger.SetupContext(context.Background(), logger.InfoLevel)
	if err != nil {
		log.Fatalf("Failed to setup logger: %v", err)
	}

	// Initialize database
	dbConfig := database.Config{
		Type: database.DatabaseTypeSQLite,
		DSN:  ":memory:", // Use in-memory database for testing
	}

	conn, err := database.InitializeConnection(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer conn.Close()

	// Run migrations
	fmt.Println("Running migrations...")
	if err := database.MigrateModels(ctx, conn,
		&worker.Worker{},
		&worker.WorkerSettings{},
		&worker.FlowStep{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create repository
	fmt.Println("Creating repository...")
	workerRepo := worker.NewRepository(conn.GetDB())

	// Test creating a worker
	fmt.Println("Creating test worker...")
	testWorker := &worker.Worker{
		Name:    "test-worker",
		Prompt:  "Test worker for database integration",
		Enabled: true,
	}

	if err := workerRepo.Create(ctx, testWorker); err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	fmt.Printf("Worker created with ID: %s\n", testWorker.ID.String())

	// Test listing workers
	fmt.Println("Listing workers...")
	workers, err := workerRepo.List(ctx)
	if err != nil {
		log.Fatalf("Failed to list workers: %v", err)
	}

	fmt.Printf("Found %d workers:\n", len(workers))
	for _, w := range workers {
		fmt.Printf("- %s (ID: %s, Enabled: %t)\n", w.Name, w.ID.String(), w.Enabled)
	}

	// Test getting worker by ID
	fmt.Println("Getting worker by ID...")
	retrievedWorker, err := workerRepo.GetByID(ctx, testWorker.ID)
	if err != nil {
		log.Fatalf("Failed to get worker by ID: %v", err)
	}

	fmt.Printf("Retrieved worker: %s\n", retrievedWorker.Name)

	fmt.Println("Database integration test completed successfully!")
}
