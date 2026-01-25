# Chronicle

A component-based test orchestration framework for Go.

Chronicle enables building complex integration tests from reusable, composable components with automatic dependency management, infrastructure provisioning, and comprehensive reporting.

## Features

- **Component-Based Architecture**: Build tests from reusable Setup, Task, Validation, and Teardown components
- **Annotation Discovery**: Automatic component discovery using `// @chronicle:*` annotations
- **Typed Context**: Type-safe data sharing between components with generics
- **Infrastructure Management**: TestContainers integration with reuse behavior
- **Chaos Engineering**: Built-in support for network latency, failures, and application chaos
- **Mock System**: Configurable mocks with matchers and conditional behavior
- **CLI Tools**: Commands for discovery, validation, execution, and visualization
- **REST API Daemon**: Long-running service with hot reload and event streaming

## Installation

```bash
go get github.com/joshua-temple/chronicle
```

## Quick Start

### 1. Define Components

```go
package mytest

import (
    "github.com/joshua-temple/chronicle/pkg/core"
)

// @chronicle:setup
// @produces user:*User
// @teardown CleanupUser
func CreateUser(ctx core.Context) error {
    user := &User{ID: "user-001", Name: "Test User"}
    ctx.Set("user", user)
    return nil
}

// @chronicle:task
// @requires user:*User
// @produces order:*Order
func CreateOrder(ctx core.Context) error {
    user, _ := ctx.Get("user")
    order := &Order{UserID: user.(*User).ID, Status: "pending"}
    ctx.Set("order", order)
    return nil
}

// @chronicle:validation
// @requires order:*Order
func VerifyOrderCreated(ctx core.Context) error {
    order, _ := ctx.Get("order")
    if order.(*Order).Status != "pending" {
        return errors.New("order not in pending status")
    }
    return nil
}

// @chronicle:teardown
// @requires user:*User
func CleanupUser(ctx core.Context) error {
    // Clean up test user
    return nil
}
```

### 2. Configure Scenario (YAML)

```yaml
# chronicle.yaml
name: "order-tests"
version: "1.0"

discovery:
  paths:
    - "./components"

scenarios:
  create_order_success:
    description: "User successfully creates an order"
    tags: ["order", "happy-path"]
    flow:
      - name: CreateUser
        type: setup
      - name: CreateOrder
        type: task
      - name: VerifyOrderCreated
        type: validation
      - name: CleanupUser
        type: teardown
```

### 3. Or Build Programmatically

```go
import (
    "github.com/joshua-temple/chronicle/pkg/scenario"
    "github.com/joshua-temple/chronicle/pkg/execution"
)

func TestOrderFlow(t *testing.T) {
    executor := execution.NewExecutor()

    // Register components
    executor.RegisterComponent(core.NewComponent("CreateUser", core.ComponentSetup).
        WithFunc(CreateUser).
        WithProduces("user", "*User"))

    executor.RegisterComponent(core.NewComponent("CreateOrder", core.ComponentTask).
        WithFunc(CreateOrder).
        WithRequires("user", "*User").
        WithProduces("order", "*Order"))

    // Build and execute scenario
    s := scenario.NewBuilder("create_order_success").
        Description("User successfully creates an order").
        Setup("CreateUser").
        Task("CreateOrder").
        Validation("VerifyOrderCreated").
        Teardown("CleanupUser").
        Build()

    result := executor.Execute(context.Background(), s)
    if !result.IsSuccess() {
        t.Errorf("Scenario failed: %v", result.Error)
    }
}
```

## CLI Usage

```bash
# Discover components in your codebase
chronicle discover

# Validate configuration
chronicle validate

# Run scenarios
chronicle run                           # Run all
chronicle run --tags smoke              # Run by tag
chronicle run --scenario my_scenario    # Run specific scenario

# Visualize dependencies
chronicle graph --format mermaid

# View results
chronicle results list
chronicle results show <run-id>
chronicle results export <run-id> --format html
```

## Daemon Mode

Run Chronicle as a long-running service:

```bash
# Start daemon with REST API
chronicle daemon --addr :8080 --watch

# API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/scenarios
curl -X POST http://localhost:8080/api/v1/runs \
  -H "X-API-Key: your-api-key" \
  -d '{"scenario_name": "my_scenario"}'
```

## Configuration

### Infrastructure

```yaml
infrastructure:
  providers:
    postgres:
      type: testcontainers
      image: postgres:15
      ports: ["5432:5432"]
      env:
        POSTGRES_PASSWORD: test
      wait:
        type: port
        port: 5432
```

### Chaos Profiles

```yaml
chaos_profiles:
  network_latency:
    latency:
      probability: 0.1
      min_delay: 100ms
      max_delay: 500ms
```

### Mock Profiles

```yaml
mock_profiles:
  payment_declined:
    matchers:
      - endpoint: "/api/payments"
        method: POST
        response:
          status: 402
          body: '{"error": "payment_declined"}'
```

## Package Structure

```
pkg/
├── chaos/         # Chaos engineering profiles
├── cli/           # CLI commands
├── config/        # YAML configuration loading
├── context/       # Typed context implementation
├── core/          # Core types (Component, TraceContext)
├── daemon/        # REST API server
├── discovery/     # AST-based annotation parser
├── execution/     # Scenario executor
├── infrastructure/# TestContainers providers
├── middleware/    # Composable middleware
├── mock/          # Mock system
├── results/       # Results storage and reporting
└── scenario/      # Scenario model and builder
```

## Examples

See the `examples/` directory:

- `examples/basic/` - Simple component usage
- `examples/full-stack/` - Complete e-commerce flow with all features
- `examples/infrastructure/` - TestContainers usage

## Documentation

- [Design Documents](docs/designs/witness/) - Architecture and design decisions
- [API Reference](docs/api/) - Package documentation

## License

This project is licensed under the terms specified in the LICENSE file.
