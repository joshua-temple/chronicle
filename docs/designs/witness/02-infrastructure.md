# Infrastructure

> Provider interfaces, built-in providers, and environment overlays.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Core Framework](./01-core-framework.md) | [Overview](./00-overview.md) | [Scenarios & Composition](./03-scenarios-and-composition.md) |

---

## Table of Contents

- [Three-Layer Model](#three-layer-model)
- [Provider Interface](#provider-interface)
- [Built-in Providers](#built-in-providers)
- [Client Auto-Exposure](#client-auto-exposure)
- [Environment Overlays](#environment-overlays)
- [Custom Providers](#custom-providers)

---

## Three-Layer Model

Infrastructure from the user's perspective has three layers:

```
┌─────────────────────────────────────────────────────────────┐
│  LAYER 3: Services Under Test                               │
│  (Your application services - what you're actually testing) │
│  Examples: API Gateway, Order Service, Auth Service         │
├─────────────────────────────────────────────────────────────┤
│  LAYER 2: Seed/Staging/Setup                                │
│  (User-provided plumbing that provisions Layer 1)           │
│  Examples: Create Kafka topics, DynamoDB tables, seed data  │
├─────────────────────────────────────────────────────────────┤
│  LAYER 1: Infrastructure                                    │
│  (Base infra the test suite leans against)                  │
│  Examples: PostgreSQL, Redis, Kafka, LocalStack, etc.       │
└─────────────────────────────────────────────────────────────┘
```

The framework manages Layer 1. Users implement Layer 2 via Setup components. Layer 3 is what you're testing.

---

## Provider Interface

The framework is unopinionated about HOW infrastructure comes up. It provides an interface; users (or built-in providers) implement it.

```go
type InfraProvider interface {
    // Lifecycle
    Initialize(ctx context.Context, config Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error

    // Health & Status
    HealthCheck(ctx context.Context) HealthReport
    Status() ProviderStatus

    // Client Access (auto-exposed to user logic)
    Client(name string) (any, error)
}
```

### HealthReport

```go
type HealthReport struct {
    Healthy  bool
    Services map[string]ServiceHealth
}

type ServiceHealth struct {
    Name    string
    Status  string  // "healthy", "unhealthy", "starting", "stopped"
    Latency time.Duration
    Error   error
}
```

---

## Built-in Providers

Framework ships convenience providers for common infrastructure. Users just provide config.

### Available Providers

| Provider | Description |
|----------|-------------|
| `postgres` | PostgreSQL database |
| `redis` | Redis cache/store |
| `kafka` | Apache Kafka |
| `dynamodb` | DynamoDB (via LocalStack or real) |
| `localstack` | AWS LocalStack for S3, SQS, SNS, etc. |
| `mysql` | MySQL database |
| `mongodb` | MongoDB |
| `elasticsearch` | Elasticsearch |
| `rabbitmq` | RabbitMQ |

### Configuration

```yaml
infrastructure:
  postgres:
    provider: postgres
    config:
      # External mode - connect to existing
      mode: external
      host: localhost
      port: 5432
      database: testdb
      username: postgres
      password: ${POSTGRES_PASSWORD}

      # OR container mode - spin up via testcontainers
      mode: container
      image: postgres:15
      database: testdb

  redis:
    provider: redis
    config:
      mode: container
      image: redis:7

  kafka:
    provider: kafka
    config:
      mode: container
      image: confluentinc/cp-kafka:7.5.0
```

---

## Client Auto-Exposure

When a provider starts, it registers clients that become available to all components:

```go
// User's component automatically gets access
// @witness:task name="QueryOrders"
func QueryOrders(ctx witness.Context) (*Orders, error) {
    // Clients injected automatically based on infrastructure config
    db := ctx.Client("postgres").(*sql.DB)
    redis := ctx.Client("redis").(*redis.Client)

    // Use clients...
    rows, err := db.Query("SELECT * FROM orders")
    // ...
}
```

### Client Types by Provider

| Provider | Client Type |
|----------|-------------|
| `postgres` | `*sql.DB` |
| `redis` | `*redis.Client` |
| `kafka` | `*kafka.Writer` / `*kafka.Reader` |
| `dynamodb` | `*dynamodb.Client` |
| `mongodb` | `*mongo.Client` |

---

## Environment Overlays

Same tests, different environments. Config switches between local/deployed without code changes.

### Overlay Structure

```
configs/
├── base.yaml        # Shared configuration
├── local.yaml       # Local development overlay
├── staging.yaml     # Staging environment overlay
└── production.yaml  # Production overlay
```

### Base Configuration

```yaml
# base.yaml - shared across all environments
scenarios:
  - name: checkout-flow
    flow:
      - setup: CreateUser
      - step: CheckoutAndPay
      - validation: OrderComplete

chaos_profiles:
  degraded-network:
    infrastructure:
      - type: network_latency
        latency_ms: 200
```

### Local Overlay

```yaml
# local.yaml
environment: local

infrastructure:
  postgres:
    mode: container
    image: postgres:15
  redis:
    mode: container
    image: redis:7
  kafka:
    mode: container
    image: confluentinc/cp-kafka:7.5.0
```

### Staging Overlay

```yaml
# staging.yaml
environment: staging

infrastructure:
  postgres:
    mode: external
    host: staging-db.internal
    port: 5432
  redis:
    mode: external
    host: staging-redis.internal
  kafka:
    mode: external
    brokers:
      - staging-kafka-1:9092
      - staging-kafka-2:9092
```

### Production Overlay

```yaml
# production.yaml
environment: production

infrastructure:
  postgres:
    mode: external
    host: prod-db.internal
    read_only: true  # Safety constraint
  redis:
    mode: external
    host: prod-redis.internal
  kafka:
    mode: external
    brokers:
      - prod-kafka-1:9092
      - prod-kafka-2:9092

# Restrict which scenarios can run in production
allowed_scenarios:
  tags: [smoke, readonly]
```

### Runtime Selection

```bash
# Local development
witness run --env local

# CI against staging
witness run --env staging

# Production smoke tests
witness run --env production --tags smoke
```

### Overlay Inheritance

```
base.yaml → local.yaml (extends base, overrides infra)
          → staging.yaml (extends base, overrides infra)
          → production.yaml (extends base, overrides infra + constraints)
```

---

## Custom Providers

Users implement the `InfraProvider` interface for custom infrastructure:

```go
// Custom provider for proprietary system
type MyCustomProvider struct {
    client *myclient.Client
}

func (p *MyCustomProvider) Initialize(ctx context.Context, config Config) error {
    endpoint := config["endpoint"].(string)
    p.client = myclient.New(endpoint)
    return nil
}

func (p *MyCustomProvider) Start(ctx context.Context) error {
    return p.client.Connect(ctx)
}

func (p *MyCustomProvider) Stop(ctx context.Context) error {
    return p.client.Disconnect(ctx)
}

func (p *MyCustomProvider) HealthCheck(ctx context.Context) HealthReport {
    err := p.client.Ping(ctx)
    return HealthReport{
        Healthy: err == nil,
        Services: map[string]ServiceHealth{
            "mycustom": {Status: "healthy"},
        },
    }
}

func (p *MyCustomProvider) Status() ProviderStatus {
    return ProviderStatusRunning
}

func (p *MyCustomProvider) Client(name string) (any, error) {
    return p.client, nil
}
```

### Registering Custom Provider

```yaml
infrastructure:
  mycustom:
    provider: custom
    path: ./providers/mycustom.go
    config:
      endpoint: http://mycustom:8080
```

---

## Next Steps

Continue to [Scenarios & Composition](./03-scenarios-and-composition.md) for scenario structure, chaos engineering, flags, options, and mocking.
