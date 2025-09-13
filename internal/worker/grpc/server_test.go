package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"autoteam/internal/worker"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestNewServer(t *testing.T) {
	mockRuntime := createMockWorkerRuntime()
	config := ServerConfig{
		Port:   8080,
		APIKey: "test-api-key",
	}

	server := NewServer(mockRuntime, config)

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	if server.runtime != mockRuntime {
		t.Error("Expected runtime to be set correctly")
	}

	if server.apiKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", server.apiKey)
	}

	if server.port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.port)
	}
}

func TestServer_Port(t *testing.T) {
	server := &Server{port: 9090}
	
	if server.Port() != 9090 {
		t.Errorf("Expected port 9090, got %d", server.Port())
	}
}

func TestServer_GetURL(t *testing.T) {
	server := &Server{port: 8080}
	
	expected := "grpc://localhost:8080"
	if server.GetURL() != expected {
		t.Errorf("Expected URL '%s', got '%s'", expected, server.GetURL())
	}
}

func TestServer_DynamicPortDiscovery(t *testing.T) {
	mockRuntime := createMockWorkerRuntime()
	config := ServerConfig{
		Port:   0, // Dynamic port discovery
		APIKey: "",
	}

	server := NewServer(mockRuntime, config)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(ctx)

	// Port should be assigned dynamically
	if server.Port() == 0 {
		t.Error("Expected port to be assigned dynamically")
	}

	// URL should reflect the assigned port
	expectedPrefix := "grpc://localhost:"
	if len(server.GetURL()) <= len(expectedPrefix) {
		t.Errorf("Expected URL to have dynamic port, got '%s'", server.GetURL())
	}
}

func TestServer_AuthValidation(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		requestAPIKey  string
		expectError    bool
		expectedCode   codes.Code
	}{
		{
			name:           "no_auth_required",
			apiKey:         "",
			requestAPIKey:  "",
			expectError:    false,
		},
		{
			name:           "valid_api_key",
			apiKey:         "secret-key",
			requestAPIKey:  "secret-key",
			expectError:    false,
		},
		{
			name:           "invalid_api_key",
			apiKey:         "secret-key",
			requestAPIKey:  "wrong-key",
			expectError:    true,
			expectedCode:   codes.Unauthenticated,
		},
		{
			name:           "missing_api_key",
			apiKey:         "secret-key",
			requestAPIKey:  "",
			expectError:    true,
			expectedCode:   codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{apiKey: tt.apiKey}
			
			ctx := context.Background()
			if tt.requestAPIKey != "" {
				md := metadata.Pairs("x-api-key", tt.requestAPIKey)
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			err := server.validateAPIKey(ctx)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("Expected gRPC status error, got %v", err)
					return
				}
				
				if st.Code() != tt.expectedCode {
					t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestServer_AuthInterceptors(t *testing.T) {
	mockRuntime := createMockWorkerRuntime()
	config := ServerConfig{
		Port:   0,
		APIKey: "test-secret",
	}

	server := NewServer(mockRuntime, config)

	// Test unary interceptor
	t.Run("unary_interceptor", func(t *testing.T) {
		ctx := context.Background()
		md := metadata.Pairs("x-api-key", "test-secret")
		ctx = metadata.NewIncomingContext(ctx, md)

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "success", nil
		}

		result, err := server.authUnaryInterceptor(ctx, nil, nil, handler)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if !called {
			t.Error("Expected handler to be called")
		}
		
		if result != "success" {
			t.Errorf("Expected 'success', got %v", result)
		}
	})

	// Test stream interceptor
	t.Run("stream_interceptor", func(t *testing.T) {
		ctx := context.Background()
		md := metadata.Pairs("x-api-key", "test-secret")
		ctx = metadata.NewIncomingContext(ctx, md)

		mockStream := &mockServerStream{ctx: ctx}

		called := false
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			called = true
			return nil
		}

		err := server.authStreamInterceptor(nil, mockStream, nil, handler)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if !called {
			t.Error("Expected handler to be called")
		}
	})
}

// createMockWorkerRuntime creates a mock worker runtime for testing
func createMockWorkerRuntime() *worker.WorkerRuntime {
	w := &worker.Worker{
		Name:   "Test Worker",
		Prompt: "Test prompt",
	}

	settings := worker.WorkerSettings{
		Flow: []worker.FlowStep{
			{
				Name: "test-step",
				Type: "debug",
			},
		},
	}

	return worker.NewWorkerRuntime(w, settings)
}

// mockServerStream implements grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(interface{}) error {
	return nil
}

func (m *mockServerStream) RecvMsg(interface{}) error {
	return nil
}