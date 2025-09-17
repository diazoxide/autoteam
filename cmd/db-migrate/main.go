package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"autoteam/internal/database"
	"autoteam/internal/export"
	"autoteam/internal/flow"
	"autoteam/internal/logger"
	"autoteam/internal/worker"

	"go.uber.org/zap"
)

func main() {
	var (
		yamlFile = flag.String("yaml", "autoteam.yaml", "YAML config file to import")
		dbPath   = flag.String("db", "./autoteam.db", "SQLite database path")
		replace  = flag.Bool("replace", false, "Replace existing workers in database")
		dryRun   = flag.Bool("dry-run", false, "Validate YAML but don't import to database")
	)
	flag.Parse()

	ctx := context.Background()
	lgr, err := logger.NewLogger(logger.InfoLevel)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	ctx = logger.WithLogger(ctx, lgr)

	lgr.Info("Starting database migration",
		zap.String("yaml_file", *yamlFile),
		zap.String("db_path", *dbPath),
		zap.Bool("replace", *replace),
		zap.Bool("dry_run", *dryRun))

	// Initialize database
	dbConfig := database.Config{
		Type: database.DatabaseTypeSQLite,
		DSN:  *dbPath,
	}

	conn, err := database.InitializeConnection(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer conn.Close()

	// Run migrations
	if err := database.MigrateModels(ctx, conn,
		&worker.Worker{},
		&worker.WorkerSettings{},
		&worker.FlowStep{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create repositories
	workerRepo := worker.NewRepository(conn.GetDB())
	flowRepo := flow.NewRepository(conn.GetDB())

	// Import from YAML
	opts := export.ImportOptions{
		Replace:    *replace,
		SkipErrors: false,
		DryRun:     *dryRun,
	}

	result, err := export.ImportWorkersFromYAML(ctx, workerRepo, flowRepo, *yamlFile, opts)
	if err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	// Display results
	fmt.Printf("\n=== Import Results ===\n")
	fmt.Printf("Total workers: %d\n", result.TotalWorkers)
	fmt.Printf("Imported: %d\n", result.ImportedCount)
	fmt.Printf("Skipped: %d\n", result.SkippedCount)
	fmt.Printf("Errors: %d\n", result.ErrorCount)

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, e := range result.Errors {
			fmt.Printf("- Worker %s (index %d): %s\n", e.WorkerName, e.WorkerIndex, e.Error)
		}
	}

	if !*dryRun {
		// Verify import by listing workers
		workers, err := workerRepo.List(ctx)
		if err != nil {
			log.Fatalf("Failed to list workers after import: %v", err)
		}

		fmt.Printf("\n=== Workers in Database ===\n")
		for _, w := range workers {
			fmt.Printf("- %s (ID: %s, Enabled: %t)\n", w.Name, w.ID.String(), w.Enabled)
			if len(w.FlowSteps) > 0 {
				fmt.Printf("  Flow steps: %d\n", len(w.FlowSteps))
			}
		}
	}

	fmt.Printf("\nMigration completed successfully!\n")
}
