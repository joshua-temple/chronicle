# Daemon Service

> Service mode, API, and scheduling for deployed environments.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [UI Layer](./06-ui-layer.md) | [Overview](./00-overview.md) | [Multi-Language](./08-multi-language.md) |

---

## Table of Contents

- [Service Architecture](#service-architecture)
- [API Reference](#api-reference)
- [Deployment Modes](#deployment-modes)
- [CI/CD Integration](#cicd-integration)
- [Observability](#observability)

---

## Service Architecture

When deployed, Witness runs as a long-lived service with full orchestration capabilities.

```
┌─────────────────────────────────────────────────────────────┐
│                    Witness Service                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  API Server │  │  Scheduler  │  │  Executor   │         │
│  │  (REST/gRPC)│  │  (Cron/     │  │  (Workers)  │         │
│  │             │  │   Random)   │  │             │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         │                │                │                 │
│         └────────────────┼────────────────┘                 │
│                          │                                  │
│                    ┌─────▼─────┐                            │
│                    │ Event Bus │                            │
│                    └─────┬─────┘                            │
│                          │                                  │
│         ┌────────────────┼────────────────┐                 │
│         │                │                │                 │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐         │
│  │   Config    │  │   Results   │  │  Notifier   │         │
│  │   Store     │  │   Store     │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Components

| Component | Responsibility |
|-----------|----------------|
| **API Server** | External interface (trigger runs, query results, manage config) |
| **Scheduler** | Cron jobs, random intervals, watch for triggers |
| **Executor** | Worker pool that runs scenarios |
| **Event Bus** | Internal pub/sub for decoupled communication |
| **Config Store** | Manages YAML configs (file or remote) |
| **Results Store** | Persists test results via adapters |
| **Notifier** | Sends alerts on configured conditions |

---

## API Reference

### Runs

```
POST   /api/v1/runs              # Trigger a run
GET    /api/v1/runs              # List runs
GET    /api/v1/runs/:id          # Get run status
GET    /api/v1/runs/:id/stream   # WebSocket for live updates
DELETE /api/v1/runs/:id          # Cancel a run
```

#### Trigger a Run

```bash
POST /api/v1/runs
Content-Type: application/json

{
  "scenarios": ["checkout-flow"],
  "environment": "staging",
  "flags": {
    "new-checkout-flow": true
  },
  "options": ["with-slow-network"],
  "chaos": {
    "profiles": ["degraded-network"]
  }
}
```

Response:
```json
{
  "id": "run_abc123",
  "status": "running",
  "scenarios": ["checkout-flow"],
  "started_at": "2024-01-15T10:30:00Z"
}
```

### Scenarios

```
GET    /api/v1/scenarios         # List scenarios
GET    /api/v1/scenarios/:name   # Get scenario details
POST   /api/v1/scenarios         # Create scenario (saves to YAML)
PUT    /api/v1/scenarios/:name   # Update scenario
DELETE /api/v1/scenarios/:name   # Delete scenario
```

### Components

```
GET    /api/v1/components        # List discovered components
GET    /api/v1/components/:name  # Get component details
GET    /api/v1/types             # List discovered types
```

### Results

```
GET    /api/v1/results           # Query results (filters, pagination)
GET    /api/v1/results/:id       # Get specific result
DELETE /api/v1/results/:id       # Delete result
```

Query parameters:
- `scenario` - Filter by scenario name
- `environment` - Filter by environment
- `status` - Filter by status (passed, failed, skipped)
- `from` / `to` - Time range
- `limit` / `offset` - Pagination

### Config

```
GET    /api/v1/config/export     # Export all YAML configs
POST   /api/v1/config/import     # Import YAML configs
GET    /api/v1/config/validate   # Validate configuration
```

### Health & Metrics

```
GET    /api/v1/health            # Service health
GET    /api/v1/metrics           # Prometheus metrics
GET    /api/v1/status            # Detailed status
```

---

## Deployment Modes

### Standalone (Simple)

Single instance for small deployments:

```yaml
service:
  mode: standalone
  host: 0.0.0.0
  port: 8080
  workers: 4
```

### Distributed (Scale Out)

Multiple workers with coordination:

```yaml
service:
  mode: distributed
  coordinator:
    type: redis
    url: redis://coordinator:6379

  api:
    replicas: 2
    port: 8080

  workers:
    min: 2
    max: 20
    autoscale: true
    autoscale_metric: queue_depth
    autoscale_threshold: 10
```

### Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: witness
spec:
  replicas: 2
  template:
    spec:
      containers:
        - name: witness
          image: witness:latest
          ports:
            - containerPort: 8080
          env:
            - name: WITNESS_MODE
              value: distributed
            - name: WITNESS_COORDINATOR
              value: redis://redis:6379
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "500m"
```

---

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run smoke tests
        run: |
          curl -X POST https://witness.internal/api/v1/runs \
            -H "Authorization: Bearer ${{ secrets.WITNESS_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d '{
              "scenarios": ["*"],
              "tags": ["smoke"],
              "environment": "staging"
            }' \
            -o run.json

          RUN_ID=$(jq -r '.id' run.json)

          # Poll for completion
          while true; do
            STATUS=$(curl -s -H "Authorization: Bearer ${{ secrets.WITNESS_TOKEN }}" \
              https://witness.internal/api/v1/runs/$RUN_ID | jq -r '.status')

            if [ "$STATUS" = "completed" ]; then break; fi
            if [ "$STATUS" = "failed" ]; then exit 1; fi
            sleep 10
          done

      - name: Download results
        run: |
          curl -H "Authorization: Bearer ${{ secrets.WITNESS_TOKEN }}" \
            https://witness.internal/api/v1/runs/$RUN_ID/report?format=junit \
            -o results.xml

      - name: Upload test results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: results.xml
```

### GitLab CI

```yaml
# .gitlab-ci.yml
integration_tests:
  stage: test
  script:
    - |
      RUN_ID=$(curl -X POST $WITNESS_URL/api/v1/runs \
        -H "Authorization: Bearer $WITNESS_TOKEN" \
        -d '{"tags": ["integration"], "environment": "staging"}' \
        | jq -r '.id')

      witness wait --run-id $RUN_ID --timeout 10m
      witness report --run-id $RUN_ID --format junit --output results.xml

  artifacts:
    reports:
      junit: results.xml
```

### CLI Helper

```bash
# Wait for run completion
witness wait --run-id <id> --timeout 10m

# Get exit code based on result
witness result --run-id <id> --exit-code

# One-liner for CI
witness run --tags smoke --env staging --wait --timeout 10m
```

---

## Observability

### Metrics (Prometheus)

```yaml
observability:
  metrics:
    enabled: true
    endpoint: /metrics
    format: prometheus

    # Custom labels
    labels:
      service: witness
      environment: ${ENVIRONMENT}
```

Available metrics:
- `witness_runs_total` - Total runs by status
- `witness_run_duration_seconds` - Run duration histogram
- `witness_scenarios_total` - Scenarios by status
- `witness_step_duration_seconds` - Step duration histogram
- `witness_workers_active` - Active workers
- `witness_queue_depth` - Pending runs in queue

### Tracing (OpenTelemetry)

```yaml
observability:
  tracing:
    enabled: true
    exporter: otlp
    endpoint: otel-collector:4317

    # Sampling
    sampling:
      strategy: ratio
      ratio: 0.1  # 10% of traces
```

### Logging

```yaml
observability:
  logging:
    level: info  # debug, info, warn, error
    format: json  # or: text
    output: stdout

    # Structured fields
    fields:
      service: witness
      version: ${VERSION}
```

### Health Checks

```yaml
health:
  liveness:
    path: /health/live
    interval: 10s

  readiness:
    path: /health/ready
    interval: 5s
    checks:
      - coordinator
      - config_store
      - results_store
```

---

## Next Steps

Continue to [Multi-Language](./08-multi-language.md) for Go, Python, Java SDKs and portability.
