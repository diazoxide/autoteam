package controlplane

import (
	"context"
	"fmt"
	"sync"
	"time"

	"autoteam/internal/config"
	"autoteam/internal/database"
	workerv1 "autoteam/internal/grpc/gen/proto/autoteam/worker/v1"
	"autoteam/internal/logger"
	"autoteam/internal/types"
	"autoteam/internal/worker"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

// WorkerRegistry manages worker endpoints and their clients
type WorkerRegistry struct {
	workers    map[string]*RegisteredWorker
	mu         sync.RWMutex
	db         *database.DB
	workerRepo worker.Repository
}

// RegisteredWorker represents a worker with its client and metadata
type RegisteredWorker struct {
	ID         string
	UUID       uuid.UUID
	Name       string
	URL        string
	APIKey     string
	Client     workerv1.WorkerServiceClient
	Conn       *grpc.ClientConn
	Status     string
	LastCheck  *time.Time
	WorkerInfo *types.WorkerInfo
	DBWorker   *worker.Worker // Worker information from database
}

// NewWorkerRegistry creates a new worker registry from database
func NewWorkerRegistry(db *database.DB) (*WorkerRegistry, error) {
	workerRepo := worker.NewRepository(db)

	registry := &WorkerRegistry{
		workers:    make(map[string]*RegisteredWorker),
		db:         db,
		workerRepo: workerRepo,
	}

	// Load workers from database
	err := registry.loadWorkersFromDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to load workers from database: %w", err)
	}

	return registry, nil
}

// GetDB returns the database instance
func (r *WorkerRegistry) GetDB() *database.DB {
	return r.db
}

// loadWorkersFromDatabase loads workers from the database
func (r *WorkerRegistry) loadWorkersFromDatabase() error {
	ctx := context.Background()
	log := logger.FromContext(ctx)

	// Get all workers from database
	dbWorkers, err := r.workerRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workers from database: %w", err)
	}

	log.Info("Loading workers from database", zap.Int("worker_count", len(dbWorkers)))

	// For now, we'll set workers as not deployed (no gRPC connection)
	// In a real scenario, workers would need to register their endpoints somehow
	for _, dbWorker := range dbWorkers {
		workerID := dbWorker.ID.String()

		registeredWorker := &RegisteredWorker{
			ID:       workerID,
			UUID:     dbWorker.ID,
			Name:     dbWorker.Name,
			Status:   types.WorkerStatusNotDeployed, // Default status for database workers
			DBWorker: dbWorker,
		}

		// If worker has deployment info or endpoint, try to connect
		// For now, we'll mark all database workers as not deployed
		// TODO: Add deployment endpoint configuration to worker model

		r.workers[workerID] = registeredWorker

		log.Debug("Loaded worker from database",
			zap.String("worker_id", workerID),
			zap.String("worker_name", dbWorker.Name),
			zap.Bool("enabled", dbWorker.Enabled))
	}

	return nil
}

// Close gracefully closes all worker connections
func (r *WorkerRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, worker := range r.workers {
		if worker.Conn != nil {
			worker.Conn.Close()
		}
	}
	return nil
}

// createContext creates a context with gRPC metadata for API key authentication
func (r *WorkerRegistry) createContext(ctx context.Context, apiKey string) context.Context {
	if apiKey != "" {
		md := metadata.Pairs("x-api-key", apiKey)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

// GetWorker returns a worker by ID
func (r *WorkerRegistry) GetWorker(id string) (*RegisteredWorker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	worker, exists := r.workers[id]
	if !exists {
		return nil, fmt.Errorf("worker not found: %s", id)
	}

	return worker, nil
}

// GetAllWorkers returns all registered workers
func (r *WorkerRegistry) GetAllWorkers() map[string]*RegisteredWorker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]*RegisteredWorker, len(r.workers))
	for id, worker := range r.workers {
		result[id] = worker
	}

	return result
}

// CheckWorkerHealth performs health check on a specific worker
func (r *WorkerRegistry) CheckWorkerHealth(ctx context.Context, id string) error {
	worker, err := r.GetWorker(id)
	if err != nil {
		return err
	}

	// Add nil check to prevent panic
	if worker == nil {
		return fmt.Errorf("worker is nil for id: %s", id)
	}

	log := logger.FromContext(ctx)

	// Skip health check for database-only workers (no client connection)
	if worker.Client == nil {
		log.Debug("Skipping health check for database-only worker",
			zap.String("worker_id", id),
			zap.String("worker_name", worker.Name),
			zap.String("status", worker.Status))
		return nil
	}

	// Create context with authentication
	grpcCtx := r.createContext(ctx, worker.APIKey)

	// Perform health check
	resp, err := worker.Client.GetHealth(grpcCtx, &emptypb.Empty{})
	if err != nil {
		r.updateWorkerStatus(id, types.WorkerStatusUnreachable, nil)
		log.Warn("Worker health check failed",
			zap.String("worker_id", id),
			zap.String("url", worker.URL),
			zap.Error(err))
		return err
	}

	// Try to get worker info for additional details
	statusResp, err := worker.Client.GetStatus(grpcCtx, &emptypb.Empty{})
	var workerInfo *types.WorkerInfo
	if err == nil && statusResp != nil {
		// Convert gRPC status response to WorkerInfo
		workerInfo = &types.WorkerInfo{
			Name: resp.Agent.Name,
			Type: resp.Agent.Type,
		}
	}

	r.updateWorkerStatus(id, types.WorkerStatusReachable, workerInfo)
	log.Debug("Worker health check successful",
		zap.String("worker_id", id),
		zap.String("url", worker.URL))

	return nil
}

// updateWorkerStatus updates worker status and metadata
func (r *WorkerRegistry) updateWorkerStatus(id, status string, workerInfo *types.WorkerInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if worker, exists := r.workers[id]; exists {
		now := time.Now()
		worker.Status = status
		worker.LastCheck = &now
		worker.WorkerInfo = workerInfo
	}
}

// PerformHealthChecks runs health checks on all workers
func (r *WorkerRegistry) PerformHealthChecks(ctx context.Context) {
	log := logger.FromContext(ctx)
	workers := r.GetAllWorkers()

	log.Debug("Performing health checks", zap.Int("worker_count", len(workers)))

	// Check all workers concurrently
	var wg sync.WaitGroup
	for id := range workers {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			_ = r.CheckWorkerHealth(ctx, workerID) // Error is already logged in CheckWorkerHealth
		}(id)
	}

	wg.Wait()
	log.Debug("Health checks completed")
}

// GetWorkerCount returns the total number of registered workers
func (r *WorkerRegistry) GetWorkerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.workers)
}

// GetHealthyWorkerCount returns the number of healthy workers
func (r *WorkerRegistry) GetHealthyWorkerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, worker := range r.workers {
		if worker.Status == types.WorkerStatusReachable {
			count++
		}
	}
	return count
}

// UpdateWorkerEndpoint updates the endpoint for a database worker and connects to it
func (r *WorkerRegistry) UpdateWorkerEndpoint(workerID, endpoint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	worker, exists := r.workers[workerID]
	if !exists {
		return fmt.Errorf("worker not found: %s", workerID)
	}

	// Close existing connection if any
	if worker.Conn != nil {
		worker.Conn.Close()
	}

	// Create gRPC connection to the worker
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(endpoint, dialOptions...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to %s: %w", endpoint, err)
	}

	// Create gRPC client
	client := workerv1.NewWorkerServiceClient(conn)

	// Update worker with endpoint and connection
	worker.URL = endpoint
	worker.Client = client
	worker.Conn = conn
	worker.Status = types.WorkerStatusUnknown // Will be updated by health check

	return nil
}

// DeployDatabaseWorkers deploys all enabled workers from database
func (r *WorkerRegistry) DeployDatabaseWorkers(ctx context.Context, runtime interface{}, cfg *config.Config) error {
	log := logger.FromContext(ctx)

	// Get runtime interface for deployment
	rt, ok := runtime.(interface {
		DeployWorker(ctx context.Context, worker worker.Worker, settings worker.WorkerSettings, cfg *config.Config) error
	})
	if !ok {
		return fmt.Errorf("runtime does not support worker deployment")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	deployedCount := 0
	for workerID, registeredWorker := range r.workers {
		if registeredWorker.DBWorker == nil {
			continue // Skip non-database workers
		}

		dbWorker := registeredWorker.DBWorker
		if !dbWorker.Enabled {
			log.Debug("Skipping disabled worker", zap.String("worker_id", workerID), zap.String("worker_name", dbWorker.Name))
			continue
		}

		log.Info("Deploying database worker", zap.String("worker_id", workerID), zap.String("worker_name", dbWorker.Name))

		// Create effective settings by merging database worker settings with global settings
		effectiveSettings := cfg.Settings
		if dbWorker.Settings != nil {
			// Merge database worker settings with global settings (database settings override global ones)
			if dbWorker.Settings.TeamName != "" {
				effectiveSettings.TeamName = dbWorker.Settings.TeamName
			}
			if dbWorker.Settings.SleepDuration != 0 {
				effectiveSettings.SleepDuration = dbWorker.Settings.SleepDuration
			}
			effectiveSettings.Debug = dbWorker.Settings.Debug
		}

		// Use flow configuration from database worker (FlowSteps), not global settings
		if len(dbWorker.FlowSteps) > 0 {
			effectiveSettings.Flow = dbWorker.FlowSteps
			log.Debug("Using flow configuration from database worker",
				zap.String("worker_id", workerID),
				zap.Int("database_flow_steps", len(dbWorker.FlowSteps)))
		} else {
			log.Warn("No flow steps found in database worker, using global flow as fallback",
				zap.String("worker_id", workerID),
				zap.Int("global_flow_steps", len(effectiveSettings.Flow)))
		}

		// Deploy worker using runtime
		if err := rt.DeployWorker(ctx, *dbWorker, effectiveSettings, cfg); err != nil {
			log.Error("Failed to deploy database worker",
				zap.String("worker_id", workerID),
				zap.String("worker_name", dbWorker.Name),
				zap.Error(err))
			continue // Continue with other workers
		}

		// Generate worker endpoint based on container naming convention
		containerName := fmt.Sprintf("%s-%s", cfg.GetTeamName(), dbWorker.GetNormalizedName())
		workerEndpoint := fmt.Sprintf("%s:8080", containerName)

		// Register the endpoint
		if err := r.updateWorkerEndpointUnsafe(workerID, workerEndpoint); err != nil {
			log.Error("Failed to register worker endpoint",
				zap.String("worker_id", workerID),
				zap.String("endpoint", workerEndpoint),
				zap.Error(err))
			continue
		}

		deployedCount++
		log.Info("Database worker deployed successfully",
			zap.String("worker_id", workerID),
			zap.String("worker_name", dbWorker.Name),
			zap.String("endpoint", workerEndpoint))
	}

	log.Info("Database workers deployment completed", zap.Int("deployed_count", deployedCount))
	return nil
}

// updateWorkerEndpointUnsafe updates worker endpoint without locking (internal use)
func (r *WorkerRegistry) updateWorkerEndpointUnsafe(workerID, endpoint string) error {
	worker, exists := r.workers[workerID]
	if !exists {
		return fmt.Errorf("worker not found: %s", workerID)
	}

	// Close existing connection if any
	if worker.Conn != nil {
		worker.Conn.Close()
	}

	// Create gRPC connection to the worker
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(endpoint, dialOptions...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to %s: %w", endpoint, err)
	}

	// Create gRPC client
	client := workerv1.NewWorkerServiceClient(conn)

	// Update worker with endpoint and connection
	worker.URL = endpoint
	worker.Client = client
	worker.Conn = conn
	worker.Status = types.WorkerStatusUnknown // Will be updated by health check

	return nil
}

// RefreshFromDatabase refreshes the worker registry from database
func (r *WorkerRegistry) RefreshFromDatabase(ctx context.Context) error {
	log := logger.FromContext(ctx)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Get all workers from database
	dbWorkers, err := r.workerRepo.List(ctx)
	if err != nil {
		log.Error("Failed to list workers from database during refresh", zap.Error(err))
		return fmt.Errorf("failed to list workers from database: %w", err)
	}

	log.Debug("Refreshing workers from database", zap.Int("worker_count", len(dbWorkers)))

	// Track which workers exist in database
	dbWorkerIDs := make(map[string]bool)

	// Update or add workers from database
	for _, dbWorker := range dbWorkers {
		workerID := dbWorker.ID.String()
		dbWorkerIDs[workerID] = true

		existingWorker, exists := r.workers[workerID]
		if exists {
			// Update existing worker with new database information
			existingWorker.Name = dbWorker.Name
			existingWorker.DBWorker = dbWorker

			log.Debug("Updated existing worker from database",
				zap.String("worker_id", workerID),
				zap.String("worker_name", dbWorker.Name))
		} else {
			// Add new worker from database
			registeredWorker := &RegisteredWorker{
				ID:       workerID,
				UUID:     dbWorker.ID,
				Name:     dbWorker.Name,
				Status:   types.WorkerStatusNotDeployed,
				DBWorker: dbWorker,
			}

			r.workers[workerID] = registeredWorker

			log.Debug("Added new worker from database",
				zap.String("worker_id", workerID),
				zap.String("worker_name", dbWorker.Name))
		}
	}

	// Remove workers that no longer exist in database
	for workerID, worker := range r.workers {
		if worker.DBWorker != nil && !dbWorkerIDs[workerID] {
			// Close connection if exists
			if worker.Conn != nil {
				worker.Conn.Close()
			}
			delete(r.workers, workerID)

			log.Debug("Removed deleted worker from registry",
				zap.String("worker_id", workerID),
				zap.String("worker_name", worker.Name))
		}
	}

	log.Info("Worker registry refreshed from database",
		zap.Int("total_workers", len(r.workers)),
		zap.Int("database_workers", len(dbWorkers)))

	return nil
}
