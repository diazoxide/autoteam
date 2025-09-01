# Control Plane

The AutoTeam control plane provides centralized management and monitoring of all workers through a REST API.

## Features

- **Worker Discovery**: Automatically discovers workers from `.autoteam/{team-name}/workers/`
- **Health Monitoring**: Real-time health checks for all workers
- **API Proxy**: Forwards requests to worker APIs with unified interface
- **Swagger UI**: Interactive API documentation at `/docs/`

## Configuration

Enable the control plane in `autoteam.yaml`:

```yaml
control_plane:
  enabled: true
  port: 9090
```

## API Endpoints

### Control Plane
- `GET /health` - Control plane health status
- `GET /workers` - List all discovered workers
- `GET /workers/{worker-id}` - Get worker details
- `GET /openapi.yaml` - OpenAPI specification
- `GET /docs/` - Swagger UI

### Worker Proxy
- `GET /workers/{worker-id}/health` - Worker health
- `GET /workers/{worker-id}/status` - Worker status
- `GET /workers/{worker-id}/config` - Worker configuration
- `GET /workers/{worker-id}/logs` - Worker logs
- `GET /workers/{worker-id}/flow` - Worker flow
- `GET /workers/{worker-id}/metrics` - Worker metrics

## Access

After running `autoteam up`:
- API: `http://localhost:9090`
- Swagger UI: `http://localhost:9090/docs/`