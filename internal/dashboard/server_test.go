package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"autoteam/internal/config"
)

func TestNewServer(t *testing.T) {
	cfg := &config.DashboardConfig{
		Enabled: true,
		Port:    8081,
		APIUrl:  "http://localhost:9090",
		Title:   "Test Dashboard",
	}

	server := NewServer(cfg)
	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	if server.config != cfg {
		t.Error("Expected config to be set correctly")
	}

	if server.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestGetPort(t *testing.T) {
	cfg := &config.DashboardConfig{
		Port: 9999,
	}

	server := NewServer(cfg)
	if server.GetPort() != 9999 {
		t.Errorf("Expected port 9999, got %d", server.GetPort())
	}
}

func TestHandleConfig(t *testing.T) {
	cfg := &config.DashboardConfig{
		Enabled: true,
		Port:    8081,
		APIUrl:  "http://test-api:9090",
		Title:   "Test Dashboard Title",
	}

	server := NewServer(cfg)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request should return config",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request should return config",
			method:         "POST",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/config.json", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(server.handleConfig)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, status)
			}

			if tt.expectedStatus == http.StatusOK {
				// Check content type
				expectedContentType := "application/json"
				if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
					t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
				}

				// Check cache control header
				expectedCacheControl := "no-cache, no-store, must-revalidate"
				if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != expectedCacheControl {
					t.Errorf("Expected cache control %s, got %s", expectedCacheControl, cacheControl)
				}

				// Parse and validate JSON response
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Expected valid JSON response, got error: %v", err)
				}

				// Check response fields
				if apiUrl, ok := response["apiUrl"].(string); !ok || apiUrl != cfg.APIUrl {
					t.Errorf("Expected apiUrl %s, got %v", cfg.APIUrl, response["apiUrl"])
				}

				if title, ok := response["title"].(string); !ok || title != cfg.Title {
					t.Errorf("Expected title %s, got %v", cfg.Title, response["title"])
				}
			}
		})
	}
}

func TestHandleStatic(t *testing.T) {
	cfg := &config.DashboardConfig{
		Enabled: true,
		Port:    8081,
		APIUrl:  "http://localhost:9090",
		Title:   "Test Dashboard",
	}

	server := NewServer(cfg)
	handler := server.handleStatic()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "config.json should be blocked from static handler",
			path:           "/config.json",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "root path should serve index.html",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
		},
		{
			name:           "non-existent path should serve index.html (SPA fallback)",
			path:           "/some/non-existent/path",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, status)
			}

			if tt.expectedType != "" {
				contentType := rr.Header().Get("Content-Type")
				if !strings.HasPrefix(contentType, tt.expectedType) {
					t.Errorf("Expected content type to start with %s, got %s", tt.expectedType, contentType)
				}
			}
		})
	}
}

func TestServer_Integration(t *testing.T) {
	cfg := &config.DashboardConfig{
		Enabled: true,
		Port:    8081,
		APIUrl:  "http://integration-test:9090",
		Title:   "Integration Test Dashboard",
	}

	server := NewServer(cfg)

	// Test that server can handle both config and static requests
	mux := http.NewServeMux()
	mux.HandleFunc("/config.json", server.handleConfig)
	mux.Handle("/", server.handleStatic())

	// Test config endpoint
	req, _ := http.NewRequest("GET", "/config.json", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Config endpoint failed with status %d", rr.Code)
	}

	// Test static endpoint
	req, _ = http.NewRequest("GET", "/", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Static endpoint failed with status %d", rr.Code)
	}
}
