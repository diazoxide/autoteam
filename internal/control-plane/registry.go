package controlplane

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"autoteam/internal/config"
	workerv1 "autoteam/internal/grpc/gen/proto/autoteam/worker/v1"
	"autoteam/internal/logger"
	"autoteam/internal/types"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

// WorkerRegistry manages worker endpoints and their clients
type WorkerRegistry struct {
	workers map[string]*RegisteredWorker
	mu      sync.RWMutex
}

// RegisteredWorker represents a worker with its client and metadata
type RegisteredWorker struct {
	ID         string
	URL        string
	APIKey     string
	Client     workerv1.WorkerServiceClient
	Conn       *grpc.ClientConn
	Status     string
	LastCheck  *time.Time
	WorkerInfo *types.WorkerInfo
}

// NewWorkerRegistry creates a new worker registry from configuration with direct API URLs
func NewWorkerRegistry(config *config.ControlPlaneConfig) (*WorkerRegistry, error) {
	registry := &WorkerRegistry{
		workers: make(map[string]*RegisteredWorker),
	}

	// Register workers from direct API URLs
	for i, apiURL := range config.WorkersAPIs {
		workerID := fmt.Sprintf("worker-%d", i+1)
		err := registry.RegisterWorker(workerID, apiURL, config.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to register worker %s: %w", workerID, err)
		}
	}

	return registry, nil
}

// parseGRPCAddress extracts host:port from a URL for gRPC connections
func parseGRPCAddress(rawURL string) (string, error) {
	// If it's already just host:port, return as-is
	if !strings.Contains(rawURL, "://") {
		return rawURL, nil
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Return host:port
	return parsedURL.Host, nil
}

// RegisterWorker adds a new worker to the registry
func (r *WorkerRegistry) RegisterWorker(id, url, apiKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Parse gRPC address from URL
	grpcAddr, err := parseGRPCAddress(url)
	if err != nil {
		return fmt.Errorf("failed to parse gRPC address from URL %s: %w", url, err)
	}

	// Create gRPC connection
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(grpcAddr, dialOptions...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to %s: %w", grpcAddr, err)
	}

	// Create gRPC client
	client := workerv1.NewWorkerServiceClient(conn)

	r.workers[id] = &RegisteredWorker{
		ID:     id,
		URL:    url,
		APIKey: apiKey,
		Client: client,
		Conn:   conn,
		Status: types.WorkerStatusUnknown,
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

	log := logger.FromContext(ctx)

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
