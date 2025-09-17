package main

import (
	"context"
	"fmt"
	"log"

	"autoteam/internal/database"
	"autoteam/internal/export"
	"autoteam/internal/flow"
	"autoteam/internal/logger"
	"autoteam/internal/worker"
)

func main() {
	fmt.Println("Testing database import from autoteam.debug.yaml...")

	// Setup logging context
	ctx, err := logger.SetupContext(context.Background(), logger.InfoLevel)
	if err != nil {
		log.Fatalf("Failed to setup logger: %v", err)
	}

	// Initialize database with SQLite
	dbConfig := database.Config{
		Type: database.DatabaseTypeSQLite,
		DSN:  "./autoteam-debug.db",
	}

	conn, err := database.InitializeConnection(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer conn.Close()

	// Run migrations
	fmt.Println("Running migrations...")
	if migrateErr := database.MigrateModels(ctx, conn,
		&worker.Worker{},
		&worker.WorkerSettings{},
		&worker.FlowStep{},
	); migrateErr != nil {
		log.Fatalf("Failed to migrate database: %v", migrateErr)
	}

	// Create repositories
	workerRepo := worker.NewRepository(conn.GetDB())
	flowRepo := flow.NewRepository(conn.GetDB())

	// Import workers from YAML file directly
	fmt.Println("Importing workers from YAML...")
	importResult, err := export.ImportWorkersFromYAML(ctx, workerRepo, flowRepo, "autoteam.debug.yaml", export.ImportOptions{
		DryRun:     false,
		Replace:    true,
		SkipErrors: false,
	})

	if err != nil {
		log.Fatalf("Failed to import workers: %v", err)
	}

	fmt.Printf("Import completed successfully!\n")
	fmt.Printf("- Total workers: %d\n", importResult.TotalWorkers)
	fmt.Printf("- Workers imported: %d\n", importResult.ImportedCount)
	fmt.Printf("- Workers skipped: %d\n", importResult.SkippedCount)
	fmt.Printf("- Error count: %d\n", importResult.ErrorCount)

	for _, err := range importResult.Errors {
		fmt.Printf("  - Error: %v\n", err)
	}

	// List imported workers
	fmt.Println("\nListing imported workers...")
	workers, err := workerRepo.List(ctx)
	if err != nil {
		log.Fatalf("Failed to list workers: %v", err)
	}

	fmt.Printf("Found %d workers:\n", len(workers))
	for _, w := range workers {
		fmt.Printf("- %s (ID: %s, Enabled: %t)\n", w.Name, w.ID.String(), w.Enabled)

		// List flow steps for each worker
		steps, err := flowRepo.ListStepsByWorker(ctx, w.ID)
		if err != nil {
			fmt.Printf("  Error getting flow steps: %v\n", err)
			continue
		}

		fmt.Printf("  Flow steps (%d):\n", len(steps))
		for _, step := range steps {
			fmt.Printf("    - %s (Type: %s, Order: %d)\n", step.Name, step.Type, step.Order)
		}
	}

	fmt.Println("\nDatabase import test completed successfully!")
}
