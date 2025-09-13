package grpc

import (
	"context"
	"net"
	"strconv"

	workerv1 "autoteam/internal/grpc/gen/proto/autoteam/worker/v1"
	"autoteam/internal/logger"
	worker "autoteam/internal/worker"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Server implements the WorkerService gRPC server
type Server struct {
	workerv1.UnimplementedWorkerServiceServer
	runtime *worker.WorkerRuntime
	apiKey  string
	port    int
	server  *grpc.Server
}

// ServerConfig contains gRPC server configuration
type ServerConfig struct {
	Port   int
	APIKey string
}

// NewServer creates a new gRPC server for worker operations
func NewServer(runtime *worker.WorkerRuntime, config ServerConfig) *Server {
	return &Server{
		runtime: runtime,
		apiKey:  config.APIKey,
		port:    config.Port,
	}
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Dynamic port discovery if port is 0
	if s.port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return err
		}
		s.port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	// Create listener
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return err
	}

	// Create gRPC server with interceptors
	opts := []grpc.ServerOption{}
	if s.apiKey != "" {
		opts = append(opts, grpc.UnaryInterceptor(s.authUnaryInterceptor))
		opts = append(opts, grpc.StreamInterceptor(s.authStreamInterceptor))
	}

	s.server = grpc.NewServer(opts...)

	// Register the service
	workerv1.RegisterWorkerServiceServer(s.server, s)

	log.Info("Starting gRPC server",
		zap.Int("port", s.port),
		zap.String("address", listener.Addr().String()))

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil {
			log.Error("gRPC server failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Stopping gRPC server")

	if s.server != nil {
		s.server.GracefulStop()
	}
	return nil
}

// Port returns the port the server is listening on
func (s *Server) Port() int {
	return s.port
}

// GetURL returns the server URL
func (s *Server) GetURL() string {
	return "grpc://localhost:" + strconv.Itoa(s.port)
}

// authUnaryInterceptor validates API key for unary RPCs
func (s *Server) authUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := s.validateAPIKey(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// authStreamInterceptor validates API key for stream RPCs
func (s *Server) authStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := s.validateAPIKey(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}

// validateAPIKey checks the API key from metadata
func (s *Server) validateAPIKey(ctx context.Context) error {
	if s.apiKey == "" {
		return nil // No auth required
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	keys := md.Get("x-api-key")
	if len(keys) == 0 || keys[0] != s.apiKey {
		return status.Errorf(codes.Unauthenticated, "invalid API key")
	}

	return nil
}
