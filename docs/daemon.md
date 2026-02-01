# Daemon API

Chronicle can run as a long-running daemon with a REST API. This enables remote test execution, real-time monitoring, and integration with CI/CD pipelines.

## Starting the Daemon

```bash
# Start with defaults (port 3000)
chronicle daemon

# Custom port
chronicle daemon --addr :8080

# With config file watching
chronicle daemon --watch

# With API key authentication
chronicle daemon --api-key my-secret-key
```

## Authentication

The daemon supports API key authentication:

```bash
# Start with API key
chronicle daemon --api-key my-secret-key
```

Include the key in requests:

```bash
curl -H "X-API-Key: my-secret-key" http://localhost:3000/api/v1/scenarios
```

## API Reference

Base URL: `http://localhost:3000/api/v1`

### Health Check

Check daemon health (no authentication required).

```
GET /api/v1/health
```

**Response:**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h30m"
}
```

---

### Scenarios

#### List Scenarios

```
GET /api/v1/scenarios
```

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `tags` | Filter by tags (comma-separated) |
| `exclude_tags` | Exclude by tags |

**Response:**

```json
{
  "scenarios": [
    {
      "name": "checkout_flow",
      "description": "Complete checkout process",
      "tags": ["smoke", "checkout"],
      "timeout": "5m"
    }
  ]
}
```

#### Get Scenario

```
GET /api/v1/scenarios/{name}
```

**Response:**

```json
{
  "name": "checkout_flow",
  "description": "Complete checkout process",
  "tags": ["smoke", "checkout"],
  "timeout": "5m",
  "flow": [
    {"type": "setup", "name": "CreateUser"},
    {"type": "task", "name": "AddToCart"},
    {"type": "validation", "name": "VerifyCart"}
  ]
}
```

---

### Runs

#### Start a Run

```
POST /api/v1/runs
```

**Request Body:**

```json
{
  "scenario_name": "checkout_flow",
  "flags": {
    "environment": "staging"
  },
  "timeout": "10m"
}
```

**Response:**

```json
{
  "id": "run_abc123",
  "status": "running",
  "scenario_name": "checkout_flow",
  "started_at": "2024-01-15T10:30:00Z"
}
```

#### Start Batch Run

Run multiple scenarios:

```
POST /api/v1/runs/batch
```

**Request Body:**

```json
{
  "scenarios": ["test_a", "test_b"],
  "tags": ["smoke"],
  "exclude_tags": ["slow"],
  "parallel": 4,
  "fail_fast": true,
  "flags": {
    "environment": "staging"
  }
}
```

Or run a suite:

```json
{
  "suite": "regression",
  "parallel": 2
}
```

**Response:**

```json
{
  "id": "run_batch_xyz",
  "status": "running",
  "scenarios": ["test_a", "test_b", "test_c"],
  "started_at": "2024-01-15T10:30:00Z"
}
```

#### List Runs

```
GET /api/v1/runs
```

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `status` | Filter by status (running, completed, failed) |
| `limit` | Max results (default: 20) |
| `since` | Results after timestamp |

**Response:**

```json
{
  "runs": [
    {
      "id": "run_abc123",
      "status": "completed",
      "scenario_name": "checkout_flow",
      "duration": "45s",
      "started_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### Get Run

```
GET /api/v1/runs/{id}
```

**Response:**

```json
{
  "id": "run_abc123",
  "status": "completed",
  "scenario_name": "checkout_flow",
  "duration": "45s",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:30:45Z",
  "results": {
    "passed": 5,
    "failed": 0,
    "skipped": 0
  },
  "flow_results": [
    {
      "name": "CreateUser",
      "type": "setup",
      "status": "passed",
      "duration": "120ms"
    }
  ]
}
```

#### Cancel Run

```
DELETE /api/v1/runs/{id}
```

**Response:**

```json
{
  "id": "run_abc123",
  "status": "cancelled"
}
```

---

### Suites

#### List Suites

```
GET /api/v1/suites
```

**Response:**

```json
{
  "suites": [
    {
      "name": "smoke",
      "description": "Quick smoke tests",
      "scenario_count": 5
    },
    {
      "name": "regression",
      "description": "Full regression suite",
      "scenario_count": 50
    }
  ]
}
```

#### Get Suite

```
GET /api/v1/suites/{name}
```

**Response:**

```json
{
  "name": "smoke",
  "description": "Quick smoke tests",
  "scenarios": ["login_test", "checkout_basic"],
  "tags": ["smoke"],
  "parallel": 4,
  "fail_fast": true
}
```

---

### Components

#### List Components

```
GET /api/v1/components
```

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `type` | Filter by type (setup, task, etc.) |
| `tags` | Filter by tags |

**Response:**

```json
{
  "components": [
    {
      "name": "CreateUser",
      "type": "setup",
      "produces": [{"key": "user", "type": "User"}],
      "requires": [],
      "tags": ["user", "auth"]
    }
  ]
}
```

#### Get Component

```
GET /api/v1/components/{name}
```

**Response:**

```json
{
  "name": "CreateUser",
  "type": "setup",
  "description": "Creates a test user",
  "produces": [{"key": "user", "type": "User"}],
  "requires": [],
  "teardown": "DeleteUser",
  "tags": ["user", "auth"],
  "source_file": "components/user.go",
  "source_line": 15
}
```

---

### Results

#### List Results

```
GET /api/v1/results
```

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `limit` | Max results (default: 20) |
| `since` | Results after timestamp |

#### Get Result

```
GET /api/v1/results/{id}
```

#### Delete Result

```
DELETE /api/v1/results/{id}
```

---

### Configuration

#### Get Config

```
GET /api/v1/config
```

Returns the current configuration (sanitized, no secrets).

#### Reload Config

```
POST /api/v1/config/reload
```

Reloads configuration from disk. Used with `--watch` for hot reload.

---

### Events (SSE)

Subscribe to real-time events:

```
GET /api/v1/events
```

This is a Server-Sent Events (SSE) endpoint. Events are streamed as they occur.

**Event Types:**

| Event | Description |
|-------|-------------|
| `run.started` | A run has started |
| `run.completed` | A run has completed |
| `run.failed` | A run has failed |
| `component.started` | A component started executing |
| `component.completed` | A component completed |
| `component.failed` | A component failed |
| `config.reloaded` | Configuration was reloaded |

**Example Event:**

```
event: run.started
data: {"id": "run_abc123", "scenario": "checkout_flow", "timestamp": "2024-01-15T10:30:00Z"}

event: component.completed
data: {"run_id": "run_abc123", "component": "CreateUser", "duration": "120ms"}

event: run.completed
data: {"id": "run_abc123", "status": "passed", "duration": "45s"}
```

**Client Example:**

```javascript
const events = new EventSource('http://localhost:3000/api/v1/events');

events.addEventListener('run.started', (e) => {
  const data = JSON.parse(e.data);
  console.log('Run started:', data.id);
});

events.addEventListener('run.completed', (e) => {
  const data = JSON.parse(e.data);
  console.log('Run completed:', data.id, data.status);
});
```

## Using from CLI

The CLI can delegate to the daemon:

```bash
# Run via daemon (auto-starts if needed)
chronicle run --daemon checkout_flow

# The daemon persists between runs for faster execution
```

## Client Library

Use the Go client for programmatic access:

```go
import "github.com/joshua-temple/chronicle/pkg/daemon/client"

// Create client
c := client.New("http://localhost:3000", client.WithAPIKey("my-key"))

// Start a run
resp, err := c.RunScenario(ctx, &client.RunRequest{
    ScenarioName: "checkout_flow",
    Flags: map[string]any{
        "environment": "staging",
    },
})

// Wait for completion
result, err := c.WaitForRun(ctx, resp.ID, time.Second)
```

## Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o chronicle ./cmd/chronicle

FROM alpine:latest
COPY --from=builder /app/chronicle /usr/local/bin/
EXPOSE 3000
CMD ["chronicle", "daemon", "--addr", ":3000"]
```

```bash
docker run -p 3000:3000 -v $(pwd):/app chronicle daemon
```

## Best Practices

1. **Use API Keys** - Always enable authentication in production
2. **Monitor Events** - Subscribe to SSE for real-time visibility
3. **Set Timeouts** - Configure appropriate timeouts for long-running tests
4. **Use Hot Reload** - Enable `--watch` for development
5. **Health Checks** - Monitor `/api/v1/health` in your infrastructure
