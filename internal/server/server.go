package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	workerapi "autoteam/api/worker"
	"autoteam/internal/logger"
	"autoteam/internal/worker"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Server represents the HTTP API server for a worker
type Server struct {
	echo       *echo.Echo
	worker     *worker.WorkerRuntime
	port       int
	apiKey     string
	workingDir string
	startTime  time.Time
	server     *http.Server
	handlers   *Handlers
	apiAdapter *APIAdapter
}

// Config contains server configuration
type Config struct {
	Port       int
	APIKey     string
	WorkingDir string
}

// NewServer creates a new HTTP API server for the given worker
func NewServer(wk *worker.WorkerRuntime, config Config) *Server {
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

	// Create handlers and API adapter
	server.handlers = NewHandlers(wk, server.workingDir, server.startTime)
	server.apiAdapter = NewAPIAdapter(server.handlers)

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

// setupRoutes configures API routes using generated OpenAPI routes
func (s *Server) setupRoutes() {
	// Use the generated RegisterHandlers to set up all routes
	workerapi.RegisterHandlers(s.echo, s.apiAdapter)

	// Add legacy routes for backward compatibility
	s.echo.GET("openapi", s.handlers.GetOpenAPISpec) // Alternative endpoint
	s.echo.GET("docs", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/")
	})
}

// Start starts the HTTP server with dynamic port discovery if port is 0
func (s *Server) Start(ctx context.Context) error {
	lgr := logger.FromContext(ctx)

	// Use dynamic port discovery if port is 0
	addr := ":" + strconv.Itoa(s.port)
	if s.port == 0 {
		addr = ":0" // Let OS choose available port
	}

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.echo,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	lgr.Debug("Starting HTTP API server",
		zap.String("worker", s.worker.Name),
		zap.String("type", s.worker.Type()),
		zap.Int("requested_port", s.port),
		zap.String("address", s.server.Addr))

	// Start server and discover the actual port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Update port with discovered port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		s.port = tcpAddr.Port
		lgr.Debug("HTTP server port discovered", zap.Int("actual_port", s.port))
	}

	// Start server in goroutine with the listener
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
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

	lgr.Debug("Stopping HTTP API server",
		zap.String("agent", s.worker.Name),
		zap.Int("port", s.port))

	// Create context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		lgr.Error("Error during server shutdown", zap.Error(err))
		return err
	}

	lgr.Debug("HTTP API server stopped")
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
