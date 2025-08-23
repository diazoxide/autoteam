package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"autoteam/internal/logger"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// WorkerInterface defines the minimal interface needed by the HTTP server
type WorkerInterface interface {
	Name() string
	Type() string
	IsAvailable(ctx context.Context) bool
	Version(ctx context.Context) (string, error)
}

// Server represents the HTTP API server for a worker
type Server struct {
	echo       *echo.Echo
	worker     WorkerInterface
	port       int
	apiKey     string
	workingDir string
	startTime  time.Time
	server     *http.Server
	handlers   *Handlers
}

// Config contains server configuration
type Config struct {
	Port       int
	APIKey     string
	WorkingDir string
}

// NewServer creates a new HTTP API server for the given worker
func NewServer(wk WorkerInterface, config Config) *Server {
	e := echo.New()
	e.HideBanner = true

	server := &Server{
		echo:       e,
		worker:     wk,
		port:       config.Port,
		apiKey:     config.APIKey,
		workingDir: config.WorkingDir,
		startTime:  time.Now(),
	}

	// Create handlers
	server.handlers = NewHandlers(wk, server.workingDir, server.startTime)

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

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	api := s.echo.Group("/")

	// Health endpoints
	api.GET("health", s.handlers.GetHealth)
	api.GET("status", s.handlers.GetStatus)

	// Log endpoints
	api.GET("logs", s.handlers.GetLogs)
	api.GET("logs/collector", s.handlers.GetCollectorLogs)
	api.GET("logs/executor", s.handlers.GetExecutorLogs)
	api.GET("logs/:filename", s.handlers.GetLogFile)

	// Task endpoints
	api.GET("tasks", s.handlers.GetTasks)
	api.GET("tasks/:id", s.handlers.GetTask)

	// Metrics endpoint
	api.GET("metrics", s.handlers.GetMetrics)

	// Configuration endpoint
	api.GET("config", s.handlers.GetConfig)

	// Documentation redirect
	api.GET("docs", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/")
	})

	// Static docs serving (for Swagger UI if needed)
	// Note: In production, you might want to embed static files or serve from CDN
	api.Static("docs/", "docs/")
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	s.server = &http.Server{
		Addr:         ":" + strconv.Itoa(s.port),
		Handler:      s.echo,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	lgr.Info("Starting HTTP API server",
		zap.String("agent", s.worker.Name()),
		zap.String("type", s.worker.Type()),
		zap.Int("port", s.port),
		zap.String("address", s.server.Addr))

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lgr.Error("HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	if s.server == nil {
		return nil
	}

	lgr.Info("Stopping HTTP API server",
		zap.String("agent", s.worker.Name()),
		zap.Int("port", s.port))

	// Create context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		lgr.Error("Error during server shutdown", zap.Error(err))
		return err
	}

	lgr.Info("HTTP API server stopped successfully")
	return nil
}

// Port returns the server port
func (s *Server) Port() int {
	return s.port
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	return s.server != nil
}

// GetURL returns the base URL for the server
func (s *Server) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// GetDocsURL returns the documentation URL
func (s *Server) GetDocsURL() string {
	return fmt.Sprintf("http://localhost:%d/docs/", s.port)
}
