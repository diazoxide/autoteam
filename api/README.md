# AutoTeam API Organization

This directory contains all API specifications and generated code for AutoTeam's various services.

## Structure

```
api/
├── README.md                    # This file
├── worker/                      # Worker Management API
│   ├── openapi.yaml            # OpenAPI 3.0 specification
│   ├── server.cfg.yaml         # oapi-codegen server config
│   ├── client.cfg.yaml         # oapi-codegen client config  
│   ├── generate.go             # Code generation directives
│   └── server.gen.go           # Generated server interface & types
├── management/                  # (Future) Management API
│   └── ...
├── monitoring/                  # (Future) Monitoring & Metrics API
│   └── ...
└── auth/                       # (Future) Authentication API
    └── ...
```

## APIs

### Worker API (`api/worker/`)
**Status**: ✅ Implemented

The Worker API provides endpoints for monitoring and managing individual AutoTeam workers:

- **Health Monitoring**: `/health` - Worker health checks and availability
- **Status Information**: `/status` - Current worker operational status  
- **Log Management**: `/logs` - Access to worker log files
- **Flow Configuration**: `/flow`, `/flow/steps` - Worker flow configuration and execution
- **Metrics**: `/metrics` - Performance metrics and statistics
- **Configuration**: `/config` - Sanitized worker configuration
- **Documentation**: `/docs/`, `/openapi.yaml` - API documentation

**Technologies**: 
- OpenAPI 3.0 specification
- oapi-codegen for code generation
- Echo framework for HTTP routing
- Go types and interfaces

### Future APIs

#### Management API (`api/management/`)
**Status**: 🔄 Planned

Central management API for:
- Team configuration management  
- Worker orchestration
- Global settings and policies
- Multi-worker coordination

#### Monitoring API (`api/monitoring/`)  
**Status**: 🔄 Planned

Centralized monitoring and observability:
- Aggregate metrics from all workers
- Health dashboards
- Alert management
- Performance analytics

#### Authentication API (`api/auth/`)
**Status**: 🔄 Planned  

Authentication and authorization:
- User management
- API key management
- Role-based access control
- Single sign-on integration

## Code Generation

Each API directory contains its own code generation setup:

```bash
# Generate all API code
make codegen

# Generate specific API (from api/{name}/ directory)
go generate .
```

### Configuration

- `openapi.yaml`: OpenAPI 3.0 specification
- `server.cfg.yaml`: Server code generation configuration  
- `client.cfg.yaml`: Client code generation configuration
- `generate.go`: Go generate directives

### Generated Files

- `server.gen.go`: Server interfaces, types, and route handlers
- `client.gen.go`: Client interfaces and methods (when needed)

## Best Practices

### API Design
- Follow OpenAPI 3.0 standards
- Use consistent naming conventions
- Include comprehensive parameter validation
- Provide detailed error responses
- Document all endpoints with examples

### Code Generation
- Keep generated code in version control
- Regenerate code after spec changes
- Use adapter pattern for existing implementations  
- Maintain backward compatibility where possible

### Versioning
- Use semantic versioning for API specs
- Consider API versioning strategy for breaking changes
- Document migration paths for API changes

## Usage

### Adding a New API

1. Create new directory: `api/{name}/`
2. Add OpenAPI specification: `openapi.yaml`
3. Configure code generation: `server.cfg.yaml`, `client.cfg.yaml`  
4. Add generation directive: `generate.go`
5. Update main Makefile if needed
6. Generate and integrate code
7. Update this README

### Modifying Existing APIs

1. Update OpenAPI specification  
2. Regenerate code: `go generate .`
3. Update adapters/implementations as needed
4. Test changes thoroughly
5. Update documentation

## Integration

APIs integrate with AutoTeam services through:

- **Server Package**: `internal/server/` - HTTP server implementation
- **Adapter Pattern**: Bridges generated interfaces with existing handlers
- **Build System**: Makefile targets for code generation
- **Configuration**: Service-specific settings and routing