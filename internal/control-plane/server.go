package controlplane

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	controlplaneapi "autoteam/api/control-plane"
	"autoteam/internal/config"
	"autoteam/internal/logger"
	"autoteam/internal/runtime"
	"autoteam/internal/worker"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Server represents the HTTP API server for the control plane
type Server struct {
	echo      *echo.Echo
	registry  *WorkerRegistry
	port      int
	apiKey    string
	startTime time.Time
	server    *http.Server
	handlers  *Handlers
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Port   int
	APIKey string
}

// NewServer creates a new HTTP API server for the control plane
func NewServer(registry *WorkerRegistry, serverConfig ServerConfig, rt runtime.Runtime, cfg *config.Config) *Server {
	e := echo.New()
	e.HideBanner = true

	server := &Server{
		echo:      e,
		registry:  registry,
		port:      serverConfig.Port,
		apiKey:    serverConfig.APIKey,
		startTime: time.Now(),
	}

	// Create worker repository from registry's database
	workerRepo := worker.NewRepository(registry.GetDB())

	// Create handlers
	server.handlers = NewHandlers(registry, rt, cfg, workerRepo)

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server
}

// setupMiddleware configures Echo middleware
func (s *Server) setupMiddleware() {
	// CORS middleware
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"*"},
	}))

	// Rate limiting middleware (100 requests per minute per IP)
	s.echo.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(100)))

	// Logger middleware
	s.echo.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","method":"${method}","uri":"${uri}","status":${status},"latency":"${latency_human}","error":"${error}"}` + "\n",
	}))

	// Recovery middleware
	s.echo.Use(middleware.Recover())

	// API Key authentication middleware (optional)
	if s.apiKey != "" {
		s.echo.Use(s.apiKeyMiddleware)
	}
}

// apiKeyMiddleware validates API key if configured
func (s *Server) apiKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		apiKey := c.Request().Header.Get("X-API-Key")
		if apiKey == "" || apiKey != s.apiKey {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or missing API key")
		}
		return next(c)
	}
}

// setupRoutes configures API routes using generated server interface
func (s *Server) setupRoutes() {
	// Create API adapter that implements ServerInterface
	apiAdapter := &APIAdapter{handlers: s.handlers}

	// Register routes using generated RegisterHandlers
	controlplaneapi.RegisterHandlers(s.echo, apiAdapter)
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Dynamic port discovery if port is 0
	if s.port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return fmt.Errorf("failed to get dynamic port: %w", err)
		}
		s.port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	// Create HTTP server
	s.server = &http.Server{
		Addr:         ":" + strconv.Itoa(s.port),
		Handler:      s.echo,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Info("Starting control plane HTTP server",
		zap.Int("port", s.port),
		zap.String("address", fmt.Sprintf("http://localhost:%d", s.port)))

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Stopping control plane HTTP server")

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}

// Port returns the server port
func (s *Server) Port() int {
	return s.port
}

// GetURL returns the server URL
func (s *Server) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// APIAdapter adapts handlers to the generated ServerInterface
type APIAdapter struct {
	handlers *Handlers
}

// Implement all ServerInterface methods by delegating to handlers
func (a *APIAdapter) GetHealth(ctx echo.Context) error {
	return a.handlers.GetHealth(ctx)
}

func (a *APIAdapter) GetWorkers(ctx echo.Context) error {
	return a.handlers.GetWorkers(ctx)
}

func (a *APIAdapter) GetWorker(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorker(ctx, workerID)
}

func (a *APIAdapter) GetWorkerRuntime(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerRuntime(ctx, workerID)
}

// Runtime method mappings
func (a *APIAdapter) GetWorkerRuntimeHealth(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerRuntimeHealth(ctx, workerID)
}

func (a *APIAdapter) GetWorkerRuntimeStatus(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerRuntimeStatus(ctx, workerID)
}

func (a *APIAdapter) GetWorkerRuntimeConfig(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerRuntimeConfig(ctx, workerID)
}

func (a *APIAdapter) GetWorkerRuntimeLogs(ctx echo.Context, workerID string, params controlplaneapi.GetWorkerRuntimeLogsParams) error {
	return a.handlers.GetWorkerRuntimeLogs(ctx, workerID, params)
}

func (a *APIAdapter) GetWorkerRuntimeLogFile(ctx echo.Context, workerID string, filename string, params controlplaneapi.GetWorkerRuntimeLogFileParams) error {
	return a.handlers.GetWorkerRuntimeLogFile(ctx, workerID, filename, params)
}

func (a *APIAdapter) GetWorkerRuntimeFlow(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerRuntimeFlow(ctx, workerID)
}

func (a *APIAdapter) GetWorkerRuntimeFlowSteps(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerRuntimeFlowSteps(ctx, workerID)
}

func (a *APIAdapter) GetWorkerRuntimeMetrics(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerRuntimeMetrics(ctx, workerID)
}

func (a *APIAdapter) GetOpenAPISpec(ctx echo.Context) error {
	return a.handlers.GetOpenAPISpec(ctx)
}

func (a *APIAdapter) GetSwaggerUI(ctx echo.Context) error {
	return a.handlers.GetSwaggerUI(ctx)
}

// Worker action methods
func (a *APIAdapter) DeployWorker(ctx echo.Context, workerID string) error {
	return a.handlers.DeployWorker(ctx, workerID)
}

func (a *APIAdapter) StopWorker(ctx echo.Context, workerID string) error {
	return a.handlers.StopWorker(ctx, workerID)
}

func (a *APIAdapter) RestartWorker(ctx echo.Context, workerID string) error {
	return a.handlers.RestartWorker(ctx, workerID)
}

func (a *APIAdapter) PauseWorker(ctx echo.Context, workerID string) error {
	return a.handlers.PauseWorker(ctx, workerID)
}

func (a *APIAdapter) UnpauseWorker(ctx echo.Context, workerID string) error {
	return a.handlers.UnpauseWorker(ctx, workerID)
}

// CRUD operations for workers

func (a *APIAdapter) CreateWorker(ctx echo.Context) error {
	return a.handlers.CreateWorker(ctx)
}

func (a *APIAdapter) UpdateWorker(ctx echo.Context, workerID string) error {
	return a.handlers.UpdateWorker(ctx, workerID)
}

func (a *APIAdapter) DeleteWorker(ctx echo.Context, workerID string) error {
	return a.handlers.DeleteWorker(ctx, workerID)
}

func (a *APIAdapter) GetWorkerSettings(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerSettings(ctx, workerID)
}

func (a *APIAdapter) UpdateWorkerSettings(ctx echo.Context, workerID string) error {
	return a.handlers.UpdateWorkerSettings(ctx, workerID)
}
