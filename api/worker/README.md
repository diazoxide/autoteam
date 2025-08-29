# Worker API

The Worker API provides HTTP endpoints for monitoring and managing individual AutoTeam workers.

## Overview

This API allows external systems to:
- Monitor worker health and status
- Access worker logs and metrics  
- View worker configuration and flow definitions
- Get real-time performance data

## Endpoints

### Health & Status
- `GET /health` - Comprehensive health check with availability status
- `GET /status` - Current operational status and mode information

### Logs & Debugging  
- `GET /logs` - List available log files with filtering options
- `GET /logs/{filename}` - Download specific log file (with tail support)

### Configuration & Flow
- `GET /config` - Sanitized worker configuration
- `GET /flow` - Flow summary and execution status  
- `GET /flow/steps` - Detailed information about each flow step

### Metrics & Performance
- `GET /metrics` - Worker performance metrics and statistics

### Documentation
- `GET /docs/` - Interactive Swagger UI documentation
- `GET /openapi.yaml` - OpenAPI 3.0 specification

## API Features

### Authentication
- Optional API key authentication via `X-API-Key` header
- Falls back to unauthenticated access if no key configured

### Error Handling
- Consistent error response format
- Proper HTTP status codes
- Detailed error messages with timestamps

### Parameter Validation
- Query parameter validation (role, limit, tail)
- Path parameter validation (filename patterns)
- Type-safe parameter binding

### Response Format
- JSON responses with timestamps
- Consistent wrapper objects
- Proper content-type headers

## Code Generation

This API is generated from the OpenAPI specification using oapi-codegen:

```bash
# Generate both server and client code
go generate .

# Or use make target
make codegen
```

### Files

- `openapi.yaml` - OpenAPI 3.0 specification (source of truth)
- `server.cfg.yaml` - Server code generation configuration
- `client.cfg.yaml` - Client code generation configuration
- `generate.go` - Go generate directives  
- `server.gen.go` - Generated server interface and types
- `client.gen.go` - Generated client interface and types

## Integration

### Server Implementation
The generated `ServerInterface` is implemented via an adapter pattern in `internal/server/adapter.go` which bridges to the existing handlers in `internal/server/handlers.go`.

### Route Registration
Routes are automatically registered using the generated `RegisterHandlers` function:

```go
worker.RegisterHandlers(echoInstance, serverImplementation)
```

### Type Safety
All request/response types are generated from the OpenAPI spec, providing compile-time type safety and automatic parameter validation.

## Development

### Modifying the API

1. Update `openapi.yaml` specification
2. Regenerate code: `go generate .`  
3. Update adapter if needed: `internal/server/adapter.go`
4. Test changes thoroughly
5. Update documentation

### Testing

The API can be tested using:
- Generated Swagger UI at `/docs/`  
- Direct HTTP requests to endpoints
- Generated client code for programmatic access

### Client Usage

Use the generated client for type-safe API calls:

```go
import "autoteam/api/worker"

// Create client
client, err := worker.NewClient("http://localhost:8080")
if err != nil {
    log.Fatal(err)
}

// Get worker health
ctx := context.Background()
response, err := client.GetHealth(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Monitoring

Workers expose their own API for self-monitoring, creating a distributed observability model where each worker provides its own metrics and status information.