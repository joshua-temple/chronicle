# Infrastructure

Chronicle manages test infrastructure through providers. The primary provider is TestContainers, which manages Docker containers for databases, message queues, and other services.

## Configuration

Configure infrastructure in `chronicle.yaml`:

```yaml
infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
    ports:
      - container: 5432
        host: 5432
    env:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: testdb
    health_check:
      command: ["pg_isready", "-U", "test"]
      interval: 1s
      timeout: 5s
      retries: 30

  redis:
    provider: testcontainers
    image: redis:7
    ports:
      - container: 6379
    health_check:
      endpoint: "redis://localhost:6379"
```

## Provider Options

### Basic Container

```yaml
infrastructure:
  myservice:
    provider: testcontainers
    image: myimage:tag
    ports:
      - container: 8080
        host: 8080        # Optional: specific host port
        protocol: tcp     # Optional: tcp (default) or udp
    env:
      KEY: value
    volumes:
      - source: ./data
        target: /data
        read_only: false
```

### Health Checks

Three ways to check container health:

```yaml
# Command-based
health_check:
  command: ["pg_isready", "-U", "postgres"]
  interval: 1s
  timeout: 5s
  retries: 30

# HTTP endpoint
health_check:
  endpoint: "http://localhost:8080/health"
  interval: 2s
  timeout: 10s
  retries: 15

# Port check (default)
health_check:
  interval: 1s
  timeout: 30s
  retries: 30
```

### Resource Limits

```yaml
infrastructure:
  heavy_service:
    provider: testcontainers
    image: resource-heavy:latest
    resources:
      memory: 2g
      cpu: "1.5"
```

### Dependencies

Start containers in order:

```yaml
infrastructure:
  database:
    provider: testcontainers
    image: postgres:15
    # ...

  app:
    provider: testcontainers
    image: myapp:latest
    depends_on: [database]  # Starts after database is healthy
```

## Reuse Behavior

Control how containers are managed between test runs:

```yaml
infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
    reuse:
      enabled: true
      ttl: 1h          # Keep alive for 1 hour
      key: "pg-main"   # Unique identifier for this container
```

### Reuse Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `always_fresh` | Destroy and recreate each run | Maximum isolation |
| `flush` | Keep container, flush data | Fast startup, clean state |
| `full` | Keep container and data | Debugging, sequential tests |

```yaml
reuse:
  enabled: true
  behavior: flush  # Options: always_fresh, flush, full
```

### Flush Strategies

When using `flush` behavior, configure how data is reset:

```yaml
infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
    reuse:
      enabled: true
      behavior: flush
    flush:
      strategy: truncate  # Truncate all tables
      exclude: [migrations, schema_versions]  # Keep these tables
```

## Docker Compose Integration

Use existing docker-compose files:

```yaml
infrastructure:
  stack:
    provider: testcontainers
    compose_file: ./docker-compose.test.yml
    services: [postgres, redis]  # Optional: specific services only
```

## Networking

### Shared Network

Containers can communicate by name when on a shared network:

```yaml
infrastructure:
  network:
    name: chronicle-network

  postgres:
    provider: testcontainers
    image: postgres:15
    # Accessible at hostname "postgres" by other containers

  app:
    provider: testcontainers
    image: myapp:latest
    env:
      DATABASE_URL: postgres://postgres:5432/testdb
```

### Custom Networks

```yaml
infrastructure:
  custom_net:
    provider: network
    driver: bridge

  service:
    provider: testcontainers
    image: myimage:latest
    network: custom_net
```

## Provider Interface

Infrastructure providers implement the `Provider` interface:

```go
type Provider interface {
    Name() string
    Initialize(ctx context.Context, config map[string]any) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    HealthCheck(ctx context.Context) HealthReport
    Status() ProviderStatus
    Client(name string) (any, error)
}
```

### Getting Clients

Access infrastructure clients in your components:

```go
// @chronicle:setup name="SetupDB" produces="db:*sql.DB"
func SetupDB(ctx context.Context) error {
    // Get the database connection from infrastructure
    db := infrastructure.Client[*sql.DB]("postgres")
    context.Set(ctx, "db", db)
    return nil
}
```

### FlushableProvider

Providers supporting state reset implement:

```go
type FlushableProvider interface {
    Provider
    Flush(ctx context.Context) error
    FlushWithConfig(ctx context.Context, config FlushConfig) error
}
```

### NetworkAwareProvider

Providers supporting Docker networks implement:

```go
type NetworkAwareProvider interface {
    Provider
    SetNetwork(networkName string)
    Network() string
}
```

## Built-in Providers

### TestContainers

General-purpose container provider:

```yaml
infrastructure:
  any_service:
    provider: testcontainers
    image: any/image:tag
```

### Common Patterns

#### PostgreSQL

```yaml
infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
    ports:
      - container: 5432
    env:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: testdb
    health_check:
      command: ["pg_isready", "-U", "test"]
    reuse:
      enabled: true
      behavior: flush
    flush:
      strategy: truncate
```

#### Redis

```yaml
infrastructure:
  redis:
    provider: testcontainers
    image: redis:7-alpine
    ports:
      - container: 6379
    health_check:
      command: ["redis-cli", "ping"]
    reuse:
      enabled: true
      behavior: flush
    flush:
      strategy: flushdb
```

#### Kafka

```yaml
infrastructure:
  kafka:
    provider: testcontainers
    image: confluentinc/cp-kafka:7.5.0
    ports:
      - container: 9092
    env:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
    depends_on: [zookeeper]
```

#### Elasticsearch

```yaml
infrastructure:
  elasticsearch:
    provider: testcontainers
    image: elasticsearch:8.11.0
    ports:
      - container: 9200
    env:
      discovery.type: single-node
      xpack.security.enabled: "false"
    health_check:
      endpoint: "http://localhost:9200/_cluster/health"
```

## Isolation Levels

Control test isolation:

| Level | Description |
|-------|-------------|
| `none` | Tests share state |
| `data` | Flush data between tests |
| `schema` | Separate schemas per test |
| `instance` | Separate containers per test |

```yaml
infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
    isolation: data  # Flush data between tests
```

## Best Practices

1. **Use Reuse** - Enable container reuse for faster test cycles
2. **Health Checks** - Always configure appropriate health checks
3. **Explicit Ports** - Let TestContainers assign random ports to avoid conflicts
4. **Flush Data** - Use flush behavior for clean state without restart overhead
5. **Dependencies** - Use `depends_on` for ordered startup
6. **Networks** - Use shared networks for container-to-container communication
