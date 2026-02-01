# Chronicle

A component-based test orchestration framework for Go. Chronicle enables building complex integration tests from reusable, composable components with automatic dependency management, infrastructure provisioning, and comprehensive reporting.

## Features

- **Component-Based Architecture** - Build tests from reusable Setup, Task, Validation, and Teardown components
- **Annotation Discovery** - Automatic component discovery using `// @chronicle:*` annotations
- **Typed Context** - Type-safe data sharing between components with generics
- **Infrastructure Management** - TestContainers integration with reuse and flush behaviors
- **Chaos Engineering** - Built-in fault injection: latency, packet loss, network partitions
- **Mock System** - Configurable mocks with matchers and conditional responses
- **CLI Tools** - Commands for discovery, validation, execution, and visualization
- **REST API Daemon** - Long-running service with hot reload and event streaming
- **Web Dashboard** - Real-time test monitoring and results visualization

## Installation

```bash
go get github.com/joshua-temple/chronicle
```

## Quick Start

### 1. Define Components

Components are Go functions with Chronicle annotations:

```go
package mytest

import "github.com/joshua-temple/chronicle/pkg/context"

// @chronicle:setup name="CreateUser" produces="user:User" teardown="DeleteUser"
func CreateUser(ctx context.Context) error {
    user := &User{ID: "user-001", Name: "Test User"}
    context.Set(ctx, "user", user)
    return nil
}

// @chronicle:task name="CreateOrder" requires="user:User" produces="order:Order"
func CreateOrder(ctx context.Context) (*Order, error) {
    user := context.Get[*User](ctx, "user")
    order := &Order{UserID: user.ID, Status: "pending"}
    context.Set(ctx, "order", order)
    return order, nil
}

// @chronicle:validation name="VerifyOrder" requires="order:Order"
func VerifyOrder(ctx context.Context, result any) error {
    order := result.(*Order)
    if order.Status != "pending" {
        return errors.New("order not in pending status")
    }
    return nil
}

// @chronicle:teardown name="DeleteUser" requires="user:User"
func DeleteUser(ctx context.Context) error {
    // Cleanup logic
    return nil
}
```

### 2. Configure Scenarios

Define scenarios in `chronicle.yaml`:

```yaml
name: order-tests
version: "1.0"

discovery:
  paths:
    - ./components

scenarios:
  - name: create_order_success
    description: User successfully creates an order
    tags: [order, happy-path]
    flow:
      - setup: CreateUser
      - task: CreateOrder
      - validation: VerifyOrder
    teardown:
      - teardown: DeleteUser
```

### 3. Run Tests

```bash
# Initialize a new project
chronicle init

# Discover components
chronicle discover

# Validate configuration
chronicle validate

# Run all scenarios
chronicle run

# Run specific scenarios
chronicle run create_order_success
chronicle run --tags smoke
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `chronicle init` | Initialize a new Chronicle project |
| `chronicle discover` | Discover and list annotated components |
| `chronicle validate` | Validate configuration and dependencies |
| `chronicle run` | Execute scenarios |
| `chronicle graph` | Visualize dependency graphs |
| `chronicle results` | Query historical test results |
| `chronicle report` | Generate reports (JSON, JUnit, HTML) |
| `chronicle daemon` | Start REST API server |

See [CLI Reference](docs/cli.md) for detailed usage.

## Programmatic API

Build scenarios programmatically for more control:

```go
import (
    "github.com/joshua-temple/chronicle/pkg/scenario"
    "github.com/joshua-temple/chronicle/pkg/execution"
)

func TestOrderFlow(t *testing.T) {
    s := scenario.NewBuilder("create_order").
        Description("User creates an order").
        Tags("order", "smoke").
        Setup("CreateUser").
        Task("CreateOrder").
        Validation("VerifyOrder").
        Teardown("DeleteUser").
        Build()

    executor := execution.NewExecutor()
    result := executor.Execute(context.Background(), s)

    if !result.IsSuccess() {
        t.Errorf("Scenario failed: %v", result.Error)
    }
}
```

## Documentation

| Document | Description |
|----------|-------------|
| [Components](docs/components.md) | Component types, annotations, context, and lifecycle |
| [Scenarios](docs/scenarios.md) | YAML configuration and programmatic builder API |
| [CLI Reference](docs/cli.md) | All commands with flags and examples |
| [Infrastructure](docs/infrastructure.md) | TestContainers, networking, and reuse strategies |
| [Chaos Testing](docs/chaos.md) | Fault injection profiles and selectors |
| [Mocking](docs/mocking.md) | Mock profiles, matchers, and responses |
| [Daemon API](docs/daemon.md) | REST API reference and authentication |
| [Web UI](docs/web-ui.md) | Dashboard features and usage |
| [Configuration](docs/configuration.md) | Complete chronicle.yaml reference |
| [Examples](docs/examples.md) | Walkthrough of example projects |

## Examples

The `examples/` directory contains working examples:

- `examples/basic/` - Simple component usage
- `examples/full-stack/` - Complete e-commerce flow with infrastructure
- `examples/infrastructure/` - TestContainers patterns
- `examples/chaos/` - Chaos engineering scenarios
