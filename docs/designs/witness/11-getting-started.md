# Getting Started

> Project structure and quickstart guide.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Test Intelligence](./10-test-intelligence.md) | [Overview](./00-overview.md) | - |

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Project Structure](#project-structure)
- [Quickstart](#quickstart)
- [CLI Reference](#cli-reference)
- [Configuration Reference](#configuration-reference)
- [Troubleshooting](#troubleshooting)
- [Migrating From Other Frameworks](#migrating-from-other-frameworks)

---

## Prerequisites

### Required

- **Go 1.21+** (required for generics support)
- **Docker** (for TestContainers support)
- **Git** (for version control integration)

### Verifying Prerequisites

```bash
# Check Go version
go version
# Should output: go version go1.21.x or higher

# Check Docker
docker --version
docker info  # Ensure daemon is running

# Check Git
git --version
```

### Optional

- **kubectl** - for Kubernetes deployment
- **Helm** - for chart-based deployment
- **make** - for build automation

---

## Installation

### CLI Installation

```bash
# macOS (Homebrew)
brew install witness

# Go install
go install github.com/witness-framework/witness@latest

# Binary download
curl -fsSL https://witness.dev/install.sh | bash

# Docker
docker pull witness/witness:latest
```

### Verify Installation

```bash
witness version
# witness v1.0.0 (go1.21, darwin/arm64)
```

---

## Project Structure

### Model A: Tests Alongside Service (Monorepo)

When your test code lives with your service:

```
my-service/
├── cmd/
│   └── server/
├── internal/
│   ├── handlers/
│   └── services/
├── pkg/
├── go.mod
├── Dockerfile
│
└── witness/                    # Witness lives in subfolder
    ├── witness.yaml            # Main configuration
    ├── environments/
    │   ├── base.yaml
    │   ├── local.yaml
    │   └── staging.yaml
    ├── scenarios/
    │   ├── checkout.yaml
    │   └── auth.yaml
    ├── chaos/
    │   └── profiles.yaml
    └── tests/
        ├── types/
        │   ├── user.go
        │   └── order.go
        ├── setup/
        │   └── users.go
        ├── tasks/
        │   └── orders.go
        ├── validations/
        │   └── orders.go
        └── steps/
            └── checkout.go
```

### Model B: Standalone Test Repo

When tests live in their own repository:

```
my-service-tests/              # Dedicated test repo
├── witness.yaml
├── environments/
│   ├── base.yaml
│   ├── local.yaml
│   └── staging.yaml
├── scenarios/
│   ├── checkout.yaml
│   └── auth.yaml
├── chaos/
│   └── profiles.yaml
└── tests/
    ├── types/
    ├── setup/
    ├── tasks/
    ├── validations/
    └── steps/
```

### Auto-Detection

```bash
# In a repo with existing go.mod/service code
witness init
# Detects existing project → creates witness/ subfolder

# In an empty directory
witness init
# Creates flat structure
```

---

## Quickstart

### 1. Initialize Project

```bash
# Create new project
mkdir my-tests && cd my-tests
witness init

# Or in existing project
cd my-service
witness init
```

### 2. Define Types

```go
// tests/types/user.go
package types

// @witness:type
type User struct {
    ID    string
    Email string
}
```

```go
// tests/types/order.go
package types

// @witness:type
type Order struct {
    ID     string
    UserID string
    Total  float64
}
```

### 3. Write Components

```go
// tests/setup/users.go
package setup

import "myproject/tests/types"

// @witness:setup name="CreateUser" produces="user:User"
func CreateUser(ctx witness.Context) error {
    user := &types.User{
        ID:    "usr_123",
        Email: "test@example.com",
    }
    witness.Set(ctx, "user", user)
    return nil
}
```

```go
// tests/tasks/orders.go
package tasks

import "myproject/tests/types"

// @witness:task name="PlaceOrder" requires="user:User" produces="order:Order"
func PlaceOrder(ctx witness.Context) (*types.Order, error) {
    user := witness.Get[*types.User](ctx, "user")

    // Call your service here
    order := &types.Order{
        ID:     "ord_456",
        UserID: user.ID,
        Total:  99.99,
    }
    return order, nil
}
```

```go
// tests/validations/orders.go
package validations

import (
    "errors"
    "myproject/tests/types"
)

// @witness:validation name="OrderCreated" requires="order:Order"
func OrderCreated(ctx witness.Context, result any) error {
    order := result.(*types.Order)
    if order.ID == "" {
        return errors.New("order ID should not be empty")
    }
    if order.Total <= 0 {
        return errors.New("order total should be positive")
    }
    return nil
}
```

### 4. Configure Infrastructure

```yaml
# witness.yaml
name: my-service-tests
version: "1.0"

discovery:
  paths:
    - ./tests

infrastructure:
  postgres:
    provider: postgres
    mode: container
    image: postgres:15

results:
  adapters:
    - type: filesystem
      path: ./results
```

### 5. Create Scenario

```yaml
# scenarios/basic.yaml
scenarios:
  - name: place-order
    description: "Test basic order placement"
    flow:
      - setup: CreateUser
      - task: PlaceOrder
      - validation: OrderCreated
```

### 6. Discover Components

```bash
witness discover

# Output:
# Types discovered:
#   User    (tests/types/user.go)
#   Order   (tests/types/order.go)
#
# Components discovered:
#   Setup:      CreateUser
#   Task:       PlaceOrder
#   Validation: OrderCreated
#
# Scenarios:
#   place-order (scenarios/basic.yaml)
```

### 7. Run Tests

```bash
witness run --scenario place-order

# Output:
# ✓ CreateUser     [passed]  12ms
# ✓ PlaceOrder     [passed]  45ms
# ✓ OrderCreated   [passed]   2ms
#
# 1 scenario, 3 steps, 0 failures
```

---

## CLI Reference

### Core Commands

| Command | Description |
|---------|-------------|
| `witness init` | Initialize new project |
| `witness discover` | Scan and list components |
| `witness run` | Execute tests |
| `witness ui` | Launch web UI |
| `witness tui` | Launch terminal UI |

### Run Options

```bash
# Run single scenario
witness run --scenario checkout-flow

# Run by tags
witness run --tags smoke,critical

# Run with flags
witness run --scenario checkout --flag new-checkout=true

# Run with options
witness run --scenario checkout --option as-admin

# Run with chaos
witness run --scenario checkout --chaos degraded-network

# Run in environment
witness run --env staging --tags integration

# Watch mode
witness watch --tags unit
```

### Management Commands

```bash
# Config management
witness config validate
witness config export --output ./backup/

# Results
witness results list --last 10
witness report --format html --output report.html

# Schedules
witness schedule list
witness schedule trigger nightly-regression

# Plugins
witness plugin list
witness plugin install mongodb-provider

# Flaky tests
witness flaky list
witness flaky quarantine some-test
```

---

## Configuration Reference

### witness.yaml

```yaml
# Project metadata
name: my-service-tests
version: "1.0"

# Component discovery
discovery:
  paths:
    - ./tests
  exclude:
    - ./tests/helpers

# Infrastructure providers
infrastructure:
  postgres:
    provider: postgres
    mode: container
    image: postgres:15
  redis:
    provider: redis
    mode: container
    image: redis:7

# Flag definitions
flags:
  definitions:
    new-checkout:
      type: boolean
      default: false
  injection:
    - method: env
      mapping:
        new-checkout: FEATURE_NEW_CHECKOUT

# Mock configuration
mocks:
  injector: wiremock
  wiremock:
    endpoint: http://localhost:8080

# Results storage
results:
  adapters:
    - type: filesystem
      path: ./results
      retention: 30d

# Notifications
notifications:
  channels:
    slack:
      webhook_url: ${SLACK_WEBHOOK}
  rules:
    - on: failure
      channels: [slack]

# Execution defaults
execution:
  timeout: 5m
  retry:
    max_attempts: 2
    backoff: exponential

# Profiling
profiling:
  enabled: true
  baseline: ./baselines/perf.json

# Flakiness detection
flakiness:
  detection:
    enabled: true
    threshold: 0.1

# Plugins
plugins:
  - name: mongodb-provider
    path: ./plugins/mongodb.so
```

### Environment Overlays

```yaml
# environments/local.yaml
environment: local

infrastructure:
  postgres:
    mode: container
  redis:
    mode: container
```

```yaml
# environments/staging.yaml
environment: staging

infrastructure:
  postgres:
    mode: external
    host: staging-db.internal
  redis:
    mode: external
    host: staging-redis.internal
```

### Scenarios

```yaml
# scenarios/checkout.yaml
scenarios:
  - name: checkout-flow
    description: "Full checkout flow"
    timeout: 30s

    flow:
      - setup: CreateUser
      - setup: SeedCart
      - task: Checkout
      - task: ProcessPayment
      - validation: OrderComplete

    chaos:
      profiles: [degraded-network]

    flags:
      new-checkout: true

    retry:
      max_attempts: 3
```

### Chaos Profiles

```yaml
# chaos/profiles.yaml
chaos_profiles:
  degraded-network:
    infrastructure:
      - type: network_latency
        target: "*"
        latency_ms: 200

  payment-outage:
    infrastructure:
      - type: service_unavailable
        target: payment-service
        duration: 10s
```

---

## Troubleshooting

### Common Issues

#### "connection refused" to infrastructure

```
Error: postgres provider: connection refused
```

**Cause:** Docker container not ready or not running.

**Solutions:**
1. Check Docker is running: `docker info`
2. Check container status: `docker ps -a`
3. Increase wait timeout in config:
   ```yaml
   infrastructure:
     postgres:
       wait_timeout: 60s
   ```

#### "component X not found"

```
Error: component "CreateUser" not found
```

**Cause:** Component not discovered.

**Solutions:**
1. Verify annotation syntax: `// @witness:setup name="CreateUser"`
2. Check discovery paths in config
3. Run `witness discover` to see what's found

#### "dependency cycle detected"

```
Error: dependency cycle: A → B → C → A
```

**Cause:** Components have circular requires/produces.

**Solution:** Refactor to break the cycle. Consider:
- Extracting shared state to a common setup
- Using explicit ordering instead of dependency inference

#### Context timeout

```
Error: context deadline exceeded
```

**Cause:** Operation took longer than configured timeout.

**Solutions:**
1. Increase timeout for slow operations
2. Check infrastructure health
3. Review component for performance issues

#### "type mismatch in context"

```
Error: type mismatch: expected *User, got *models.User
```

**Cause:** Type alias or import path mismatch.

**Solutions:**
1. Ensure consistent type imports across components
2. Use type aliases: `// @witness:type alias="User"`
3. Check produces/requires declarations match actual types

### Debug Mode

```bash
# Run with verbose logging
witness run --scenario checkout-flow --debug

# Show discovery details
witness discover --verbose

# Validate configuration
witness config validate --verbose
```

---

## Migrating From Other Frameworks

### From testify/suite

```go
// Before (testify)
type MyTestSuite struct {
    suite.Suite
    db *sql.DB
}

func (s *MyTestSuite) SetupTest() {
    s.db = setupDatabase()
}

func (s *MyTestSuite) TestCreateUser() {
    user := createUser(s.db)
    s.NotNil(user.ID)
}

// After (Witness)
// @witness:setup name="SetupDatabase" produces="db:*sql.DB"
func SetupDatabase(ctx witness.Context) error {
    db := ctx.Client("postgres").(*sql.DB)
    witness.Set(ctx, "db", db)
    return nil
}

// @witness:task name="CreateUser" requires="db:*sql.DB" produces="user:User"
func CreateUser(ctx witness.Context) (*User, error) {
    db := witness.Get[*sql.DB](ctx, "db")
    return createUser(db)
}

// @witness:validation name="UserHasID" requires="user:User"
func UserHasID(ctx witness.Context, result any) error {
    user := result.(*User)
    if user.ID == "" {
        return errors.New("user ID should not be empty")
    }
    return nil
}
```

### From Ginkgo/Gomega

```go
// Before (Ginkgo)
var _ = Describe("Checkout", func() {
    var user *User

    BeforeEach(func() {
        user = createTestUser()
    })

    It("should create an order", func() {
        order := checkout(user)
        Expect(order.ID).NotTo(BeEmpty())
    })
})

// After (Witness) - scenarios/checkout.yaml
scenarios:
  - name: checkout-creates-order
    flow:
      - setup: CreateTestUser
      - task: Checkout
      - validation: OrderHasID
```

### From go test (table-driven)

```go
// Before (go test)
func TestCheckout(t *testing.T) {
    tests := []struct {
        name     string
        currency string
        expected float64
    }{
        {"USD checkout", "USD", 99.99},
        {"EUR checkout", "EUR", 89.99},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := checkout(tt.currency)
            if result.Total != tt.expected {
                t.Errorf("got %v, want %v", result.Total, tt.expected)
            }
        })
    }
}

// After (Witness) - scenarios/checkout.yaml
scenarios:
  - name: checkout-currencies
    matrix:
      currency: [USD, EUR]
      expected: [99.99, 89.99]
    flow:
      - task: Checkout
        params:
          currency: ${{ matrix.currency }}
      - validation: CheckTotal
        params:
          expected: ${{ matrix.expected }}
```

### Migration Checklist

- [ ] Identify all test suites and their dependencies
- [ ] Map `SetupTest`/`BeforeEach` to Setup components
- [ ] Map test functions to Task components
- [ ] Map assertions to Validation components
- [ ] Extract infrastructure setup to provider config
- [ ] Convert table-driven tests to matrix scenarios
- [ ] Run `witness discover` to verify components found
- [ ] Create scenarios combining components
- [ ] Run tests and compare results with original framework

---

## Next Steps

1. Read [Core Framework](./01-core-framework.md) to understand the component model
2. Explore [Scenarios & Composition](./03-scenarios-and-composition.md) for advanced patterns
3. Set up [CI/CD Integration](./07-daemon-service.md#cicd-integration)
4. Learn about [Test Intelligence](./10-test-intelligence.md) features

---

## Resources

- [GitHub Repository](https://github.com/witness-framework/witness)
- [API Documentation](https://docs.witness.dev/api)
- [Examples Repository](https://github.com/witness-framework/examples)
- [Community Discord](https://discord.gg/witness)
