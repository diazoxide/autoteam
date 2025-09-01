package controlplane

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	workerapi "autoteam/api/worker"
	"autoteam/internal/config"
	"autoteam/internal/logger"
	"autoteam/internal/types"

	"go.uber.org/zap"
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
	Client     *workerapi.Client
	Status     string
	LastCheck  *time.Time
	WorkerInfo *types.WorkerInfo
}

// NewWorkerRegistry creates a new worker registry from configuration with dynamic discovery
func NewWorkerRegistry(config *config.ControlPlaneConfig) (*WorkerRegistry, error) {
	registry := &WorkerRegistry{
		workers: make(map[string]*RegisteredWorker),
	}

	// Discover workers from filesystem
	discoveredWorkers, err := DiscoverWorkers(config.WorkersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover workers from %s: %w", config.WorkersDir, err)
	}

	// Register discovered workers
	for _, worker := range discoveredWorkers {
		err := registry.RegisterWorker(worker.ID, worker.URL, worker.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to register discovered worker %s: %w", worker.ID, err)
		}
	}

	return registry, nil
}

// RegisterWorker adds a new worker to the registry
func (r *WorkerRegistry) RegisterWorker(id, url, apiKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create worker API client
	var clientOptions []workerapi.ClientOption
	if apiKey != "" {
		clientOptions = append(clientOptions, workerapi.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("X-API-Key", apiKey)
			return nil
		}))
	}

	client, err := workerapi.NewClient(url, clientOptions...)
	if err != nil {
		return fmt.Errorf("failed to create worker client: %w", err)
	}

	r.workers[id] = &RegisteredWorker{
		ID:     id,
		URL:    url,
		APIKey: apiKey,
		Client: client,
		Status: types.WorkerStatusUnknown,
	}

	return nil
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

	// Perform health check
	resp, err := worker.Client.GetHealth(ctx)
	if err != nil {
		r.updateWorkerStatus(id, types.WorkerStatusUnreachable, nil)
		log.Warn("Worker health check failed",
			zap.String("worker_id", id),
			zap.String("url", worker.URL),
			zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.updateWorkerStatus(id, types.WorkerStatusUnreachable, nil)
		return fmt.Errorf("worker health check returned status %d", resp.StatusCode)
	}

	// Try to get worker info for additional details
	infoResp, err := worker.Client.GetStatus(ctx)
	var workerInfo *types.WorkerInfo
	if err == nil && infoResp.StatusCode == 200 {
		// Parse worker info from status response if available
		// This would require parsing the JSON response, but for now we'll just mark as reachable
		defer infoResp.Body.Close()
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
