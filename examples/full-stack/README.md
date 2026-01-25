# Chronicle Full-Stack Example

This example demonstrates all Chronicle features using an e-commerce order flow scenario.

## Features Demonstrated

- **Component Types**: Setup, Task, Validation, and Teardown components
- **Annotations**: Using `@chronicle:` annotations for component discovery
- **Dependencies**: Using `@requires` and `@produces` for data flow
- **Scenario Composition**: Building complex test flows from reusable components
- **Scenario Inheritance**: Using `extends` for DRY scenario definitions
- **Mock Profiles**: Configuring different external service behaviors
- **Chaos Profiles**: Testing resilience under adverse conditions
- **Conditional Execution**: Running scenarios based on flags
- **Matrix Testing**: Running scenarios with different parameter combinations
- **Parallel Execution**: Running independent validations in parallel

## Directory Structure

```
full-stack/
├── chronicle.yaml          # Configuration file
├── components/
│   ├── setup.go           # Setup components (CreateTestUser, CreateTestProduct)
│   ├── tasks.go           # Task components (AddItemToCart, ProcessPayment, etc.)
│   ├── validations.go     # Validation components (VerifyOrderCreated, etc.)
│   └── teardown.go        # Teardown components (CleanupTestOrder, etc.)
├── example_test.go        # Go test file demonstrating programmatic usage
└── README.md              # This file
```

## Running the Example

### Using the CLI

```bash
# Discover components
chronicle discover

# Validate configuration
chronicle validate

# Run all scenarios
chronicle run

# Run specific scenarios by tag
chronicle run --tags smoke
chronicle run --tags order,happy-path

# Run with specific flags
chronicle run --flag environment=staging

# Run with chaos enabled
chronicle run --chaos network_latency

# Run with mocks enabled
chronicle run --mock payment_declined

# Dry run to see what would be executed
chronicle run --dry-run

# Generate dependency graph
chronicle graph --format dot > graph.dot
```

### Using Go Tests

```bash
# Run all tests
go test ./examples/full-stack/... -v

# Run specific test
go test ./examples/full-stack/... -v -run TestCompleteOrderFlow
```

### Using the Daemon

```bash
# Start the daemon
chronicle daemon --addr :8080 --watch

# In another terminal, interact with the API
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/scenarios
curl -X POST http://localhost:8080/api/v1/runs \
  -H "X-API-Key: <your-api-key>" \
  -d '{"scenario_name":"complete_order_success"}'
```

## Scenarios

### Happy Path Scenarios

- **complete_order_success**: Full order flow with successful payment
- **order_with_notifications**: Order flow including notification sending

### Error Handling Scenarios

- **payment_declined_flow**: Tests behavior when payment is declined
- **out_of_stock_flow**: Tests behavior when item is unavailable

### Resilience Scenarios

- **order_with_network_latency**: Order processing under network latency

### Matrix Scenarios

- **payment_provider_matrix**: Tests order flow with multiple payment providers

### Comprehensive Scenarios

- **comprehensive_order_validation**: Full validation with parallel checks

## Configuration

The `chronicle.yaml` file demonstrates:

- **Discovery paths**: Where to find component files
- **Infrastructure**: Mock database, cache, and message queue configuration
- **Execution settings**: Timeouts, parallelism, fail-fast behavior
- **Results storage**: Where to store test results
- **Flags**: Default values and bundles for different environments
- **Chaos profiles**: Network latency, database failures, high load simulation
- **Mock profiles**: Happy path, payment declined, out of stock scenarios
- **Scenarios**: Complete scenario definitions with inheritance and conditions

## Component Annotations

Components use annotations for discovery:

```go
// @chronicle:setup
// @produces user:*User
// @teardown CleanupTestUser
// @description Creates a test user for the order flow
func CreateTestUser(ctx core.Context) error {
    // ...
}
```

Available annotations:
- `@chronicle:setup` - Setup component
- `@chronicle:task` - Task component
- `@chronicle:validation` - Validation component
- `@chronicle:teardown` - Teardown component
- `@chronicle:step` - Step component (bundle of other components)
- `@chronicle:rollup` - Rollup component (higher-order composition)
- `@requires key:Type` - Declares a dependency
- `@produces key:Type` - Declares what the component produces
- `@teardown ComponentName` - Pairs setup with teardown
- `@description Text` - Human-readable description
- `@tags tag1,tag2` - Tags for filtering
- `@owner team-name` - Team responsible for the component

## Best Practices Demonstrated

1. **Separation of Concerns**: Each component has a single responsibility
2. **Explicit Dependencies**: All data dependencies are declared
3. **Reusable Components**: Components can be composed into different scenarios
4. **Graceful Cleanup**: Teardown runs even when tests fail
5. **Clear Documentation**: Components have descriptions and tags
6. **Testable Design**: Components are pure functions with no hidden state
