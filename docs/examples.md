# Examples

The `examples/` directory contains working examples demonstrating Chronicle's features.

## Directory Structure

```
examples/
├── basic/           # Simple component usage
├── full-stack/      # Complete e-commerce flow
├── infrastructure/  # TestContainers patterns
├── chaos/           # Chaos engineering scenarios
├── distributed/     # Distributed system testing
└── mocks/           # Mock service examples
```

## Basic Example

**Location:** `examples/basic/`

A minimal example showing core Chronicle concepts.

### Components

```go
// components.go

// @chronicle:setup name="CreateUser" produces="user:User" teardown="DeleteUser"
func CreateUser(ctx context.Context) error {
    user := &User{ID: "usr_123", Email: "test@example.com"}
    context.Set(ctx, "user", user)
    return nil
}

// @chronicle:teardown name="DeleteUser" requires="user:User"
func DeleteUser(ctx context.Context) error {
    user := context.Get[*User](ctx, "user")
    // cleanup logic
    return nil
}

// @chronicle:task name="CreateOrder" requires="user:User" produces="order:Order"
func CreateOrder(ctx context.Context) (*Order, error) {
    user := context.Get[*User](ctx, "user")
    order := &Order{ID: "ord_456", UserID: user.ID}
    context.Set(ctx, "order", order)
    return order, nil
}

// @chronicle:validation name="OrderValid" requires="order:Order"
func OrderValid(ctx context.Context, result any) error {
    order := result.(*Order)
    if order.ID == "" {
        return errors.New("order ID required")
    }
    return nil
}
```

### Configuration

```yaml
# chronicle.yaml
name: basic-example
version: "1.0"

discovery:
  paths:
    - ./

scenarios:
  - name: create_order
    description: Basic order creation flow
    flow:
      - setup: CreateUser
      - task: CreateOrder
      - validation: OrderValid
    teardown:
      - teardown: DeleteUser
```

### Running

```bash
cd examples/basic
chronicle discover
chronicle validate
chronicle run
```

---

## Full-Stack Example

**Location:** `examples/full-stack/`

A complete e-commerce testing scenario with database, caching, and external service mocks.

### Infrastructure

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
      POSTGRES_DB: ecommerce
    health_check:
      command: ["pg_isready", "-U", "test"]
    reuse:
      enabled: true
      behavior: flush

  redis:
    provider: testcontainers
    image: redis:7-alpine
    ports:
      - container: 6379
    health_check:
      command: ["redis-cli", "ping"]
    reuse:
      enabled: true
```

### Scenarios

```yaml
scenarios:
  - name: complete_checkout
    description: Full checkout flow with payment
    tags: [checkout, integration]
    timeout: 5m
    flow:
      - setup: SetupDatabase
      - setup: CreateUser
      - setup: CreateProduct
      - task: AddToCart
      - task: ApplyCoupon
      - task: ProcessPayment
      - validation: VerifyOrder
      - validation: VerifyInventory
    teardown:
      - teardown: CleanupOrder
      - teardown: CleanupUser

  - name: checkout_payment_failure
    description: Checkout with declined payment
    tags: [checkout, error-handling]
    mock_profiles: [payment_declined]
    flow:
      - setup: SetupDatabase
      - setup: CreateUser
      - setup: CreateProduct
      - task: AddToCart
      - task: ProcessPayment
      - validation: VerifyPaymentError
```

### Running

```bash
cd examples/full-stack
chronicle run
chronicle run --tags checkout
chronicle run checkout_payment_failure
```

---

## Infrastructure Example

**Location:** `examples/infrastructure/`

Demonstrates various TestContainers patterns.

### Multiple Databases

```yaml
infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
    # ...

  mysql:
    provider: testcontainers
    image: mysql:8
    env:
      MYSQL_ROOT_PASSWORD: test
      MYSQL_DATABASE: testdb

  mongodb:
    provider: testcontainers
    image: mongo:6
```

### Message Queues

```yaml
infrastructure:
  kafka:
    provider: testcontainers
    image: confluentinc/cp-kafka:7.5.0
    depends_on: [zookeeper]
    # ...

  rabbitmq:
    provider: testcontainers
    image: rabbitmq:3-management
    ports:
      - container: 5672
      - container: 15672
```

### Docker Compose Integration

```yaml
infrastructure:
  stack:
    provider: testcontainers
    compose_file: ./docker-compose.test.yml
    services: [api, database, cache]
```

---

## Chaos Example

**Location:** `examples/chaos/`

Demonstrates fault injection for resilience testing.

### Profiles

```yaml
chaos_profiles:
  slow_database:
    network:
      latency:
        enabled: true
        min: 500ms
        max: 2s

  network_partition:
    network:
      partition:
        enabled: true
        duration: 10s
        targets: [cache]

  high_load:
    resource:
      cpu:
        enabled: true
        percentage: 80
        duration: 30s
```

### Test Scenarios

```yaml
scenarios:
  - name: test_timeout_handling
    chaos_profiles: [slow_database]
    timeout: 5s
    flow:
      - task: QueryDatabase
      - validation: VerifyTimeout

  - name: test_cache_fallback
    chaos_profiles: [network_partition]
    flow:
      - task: FetchWithCache
      - validation: VerifyFallbackBehavior

  - name: test_under_load
    chaos_profiles: [high_load]
    flow:
      - task: ProcessRequests
      - validation: VerifyPerformance
```

---

## Running Examples

### Prerequisites

1. Go 1.21+
2. Docker (for TestContainers)
3. Chronicle CLI installed

### Quick Start

```bash
# Clone and navigate to examples
cd examples/basic

# Discover components
chronicle discover

# Validate configuration
chronicle validate

# Run all scenarios
chronicle run

# Run with verbose output
chronicle run -v

# Run specific scenario
chronicle run create_order
```

### Common Commands

```bash
# View dependency graph
chronicle graph --format mermaid

# Dry run (show what would execute)
chronicle run --dry-run

# Run with chaos
chronicle run --chaos network_latency

# Generate report
chronicle report --latest --format html --output report.html
```

## Creating Your Own

1. **Start Simple** - Copy `examples/basic/` as a template
2. **Add Components** - Create annotated functions
3. **Configure** - Define scenarios in `chronicle.yaml`
4. **Validate** - Run `chronicle validate`
5. **Execute** - Run `chronicle run`

### Template

```bash
# Create new project
mkdir my-tests && cd my-tests
chronicle init

# Create component file
cat > components.go << 'EOF'
package main

import "github.com/joshua-temple/chronicle/pkg/context"

// @chronicle:setup name="Setup"
func Setup(ctx context.Context) error {
    return nil
}

// @chronicle:task name="DoSomething"
func DoSomething(ctx context.Context) (any, error) {
    return "result", nil
}

// @chronicle:validation name="Verify"
func Verify(ctx context.Context, result any) error {
    return nil
}
EOF

# Edit chronicle.yaml to add scenario
# Then run
chronicle discover
chronicle validate
chronicle run
```
