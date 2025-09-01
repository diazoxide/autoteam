# AutoTeam Control Plane API

The Control Plane API provides a centralized interface for managing and monitoring multiple AutoTeam workers. It acts as a proxy/gateway that forwards requests to individual worker APIs.

## Overview

- **Read-only operations**: Only monitoring and status endpoints are supported
- **Worker management**: Discovery and health monitoring of configured workers
- **Proxy functionality**: All worker API endpoints are accessible via the control plane
- **Configuration**: Configured via standard `autoteam.yaml` file

## API Endpoints

### Worker Management
- `GET /workers` - List all configured workers
- `GET /workers/{worker_id}` - Get specific worker details
- `GET /workers/{worker_id}/health` - Worker health check

### Proxied Worker Endpoints
All worker API endpoints are available with the `workers/{worker_id}` prefix:
- `GET /workers/{worker_id}/status` - Worker operational status
- `GET /workers/{worker_id}/config` - Worker configuration
- `GET /workers/{worker_id}/logs` - Worker log files
- `GET /workers/{worker_id}/logs/{filename}` - Download specific log file
- `GET /workers/{worker_id}/flow` - Worker flow configuration
- `GET /workers/{worker_id}/flow/steps` - Detailed flow step information
- `GET /workers/{worker_id}/metrics` - Worker performance metrics

### Documentation
- `GET /openapi.yaml` - OpenAPI specification
- `GET /docs/` - Swagger UI documentation

## Configuration

Add control plane configuration to your `autoteam.yaml`:

```yaml
# Existing worker configuration
workers:
  - name: "Senior Developer"
    # ... worker config

# Control plane configuration
control_plane:
  enabled: true
  port: 9090
  api_key: "optional-api-key"
  workers:
    - id: "senior-dev"
      url: "http://senior-dev:8080"
      api_key: "worker-api-key"
    - id: "junior-dev"
      url: "http://junior-dev:8081"
```

## Usage

### Start Control Plane
```bash
autoteam-control-plane --config-file autoteam.yaml
```

### Example API Calls
```bash
# List all workers
curl http://localhost:9090/workers

# Get specific worker status
curl http://localhost:9090/workers/senior-dev/status

# Get worker logs
curl http://localhost:9090/workers/senior-dev/logs

# Control plane health check
curl http://localhost:9090/health
```

## Features

- **Health Monitoring**: Automatic health checks for all configured workers
- **Service Discovery**: Workers are discovered from configuration
- **Circuit Breaking**: Unreachable workers are marked and excluded from requests
- **Request Proxying**: Transparent forwarding of requests to worker APIs
- **Aggregated Health**: Control plane health reflects the status of all workers

## Generated Code

This API uses OpenAPI code generation:
- `server.gen.go` - Generated server interfaces and types
- `client.gen.go` - Generated client code
- Run `go generate` to regenerate after OpenAPI spec changes