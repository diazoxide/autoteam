package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"autoteam/internal/config"
	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// Server represents the dashboard HTTP server
type Server struct {
	config *config.DashboardConfig
	logger *zap.Logger
}

// NewServer creates a new dashboard server
func NewServer(cfg *config.DashboardConfig) *Server {
	log, _ := logger.NewLogger(logger.InfoLevel)
	return &Server{
		config: cfg,
		logger: log,
	}
}

// Start starts the dashboard server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Serve dynamic config endpoint
	mux.HandleFunc("/config.json", s.handleConfig)

	// Serve static files with SPA fallback
	mux.Handle("/", s.handleStatic())

	s.logger.Info("Starting dashboard server",
		zap.Int("port", s.config.Port),
		zap.String("api_url", s.config.APIUrl),
	)

	return http.ListenAndServe(fmt.Sprintf(":%d", s.config.Port), mux)
}

// handleConfig serves the dynamic configuration
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"apiUrl": s.config.APIUrl,
		"title":  s.config.Title,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	
	if err := json.NewEncoder(w).Encode(config); err != nil {
		s.logger.Error("Failed to encode config", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.logger.Debug("Served dynamic config", zap.Any("config", config))
}

// handleStatic serves static files from embedded filesystem with SPA fallback
func (s *Server) handleStatic() http.Handler {
	// Create sub-filesystem from the dist directory
	distFS, err := fs.Sub(DashboardFS, "dist")
	if err != nil {
		s.logger.Error("Failed to create dist sub-filesystem", zap.Error(err))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Dashboard not available", http.StatusInternalServerError)
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Don't serve /config.json from static files (handled by dynamic endpoint)
		if path == "/config.json" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Remove leading slash for filesystem lookup
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}

		// If path is empty (root), serve index.html
		if path == "" {
			path = "index.html"
		}

		// Check if file exists
		_, err := fs.Stat(distFS, path)
		if err != nil {
			// File not found, serve index.html for SPA routing
			indexFile, indexErr := distFS.Open("index.html")
			if indexErr != nil {
				http.Error(w, "Dashboard not available", http.StatusInternalServerError)
				return
			}
			defer indexFile.Close()

			w.Header().Set("Content-Type", "text/html")
			io.Copy(w, indexFile)
			return
		}

		// Add cache headers for static assets
		if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
			w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		}

		fileServer.ServeHTTP(w, r)
	})
}

// GetPort returns the server port
func (s *Server) GetPort() int {
	return s.config.Port
}