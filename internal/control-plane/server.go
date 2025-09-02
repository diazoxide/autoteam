package controlplane

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	controlplaneapi "autoteam/api/control-plane"
	"autoteam/internal/logger"

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
func NewServer(registry *WorkerRegistry, config ServerConfig) *Server {
	e := echo.New()
	e.HideBanner = true

	server := &Server{
		echo:      e,
		registry:  registry,
		port:      config.Port,
		apiKey:    config.APIKey,
		startTime: time.Now(),
	}

	// Create handlers
	server.handlers = NewHandlers(registry)

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
		AllowMethods: []string{http.MethodGet, http.MethodOptions},
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

func (a *APIAdapter) GetWorkerHealth(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerHealth(ctx, workerID)
}

func (a *APIAdapter) GetWorkerStatus(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerStatus(ctx, workerID)
}

func (a *APIAdapter) GetWorkerConfig(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerConfig(ctx, workerID)
}

func (a *APIAdapter) GetWorkerLogs(ctx echo.Context, workerID string, params controlplaneapi.GetWorkerLogsParams) error {
	return a.handlers.GetWorkerLogs(ctx, workerID, params)
}

func (a *APIAdapter) GetWorkerLogFile(ctx echo.Context, workerID string, filename string, params controlplaneapi.GetWorkerLogFileParams) error {
	return a.handlers.GetWorkerLogFile(ctx, workerID, filename, params)
}

func (a *APIAdapter) GetWorkerFlow(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerFlow(ctx, workerID)
}

func (a *APIAdapter) GetWorkerFlowSteps(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerFlowSteps(ctx, workerID)
}

func (a *APIAdapter) GetWorkerMetrics(ctx echo.Context, workerID string) error {
	return a.handlers.GetWorkerMetrics(ctx, workerID)
}

func (a *APIAdapter) GetOpenAPISpec(ctx echo.Context) error {
	return a.handlers.GetOpenAPISpec(ctx)
}

func (a *APIAdapter) GetSwaggerUI(ctx echo.Context) error {
	return a.handlers.GetSwaggerUI(ctx)
}
