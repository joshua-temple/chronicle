# Scenarios

Scenarios define the execution flow of components. You can configure them in YAML or build them programmatically.

## YAML Configuration

Define scenarios in `chronicle.yaml`:

```yaml
scenarios:
  - name: user_creates_order
    description: A user successfully creates an order
    tags: [order, happy-path, smoke]
    timeout: 5m
    flow:
      - setup: CreateUser
      - setup: CreateCart
      - task: AddItemToCart
      - task: CreateOrder
      - validation: VerifyOrderCreated
    teardown:
      - teardown: DeleteOrder
      - teardown: DeleteUser
```

## Flow Items

Each flow item specifies a component type and name:

```yaml
flow:
  - setup: ComponentName       # Setup component
  - task: ComponentName        # Task component
  - validation: ComponentName  # Validation component
  - step: ComponentName        # Step (composite)
  - rollup: ComponentName      # Rollup (composite)
```

### Flow Item Options

```yaml
flow:
  - task: ProcessPayment
    timeout: 30s              # Override default timeout
    params:                    # Pass parameters
      amount: 99.99
      currency: USD
    depends_on: [CreateOrder]  # Explicit dependencies
    parallel: true             # Can run in parallel
```

## Scenario Options

### Tags

Filter scenarios by tags:

```yaml
scenarios:
  - name: smoke_test
    tags: [smoke, quick]
    # ...
```

Run with: `chronicle run --tags smoke`

### Timeout

Set scenario-level timeout:

```yaml
scenarios:
  - name: long_running_test
    timeout: 30m
    # ...
```

### Conditional Execution

Skip scenarios based on conditions:

```yaml
scenarios:
  - name: production_only
    skip_if:
      - env: CI
        reason: "Skip in CI environment"
    skip_unless:
      - flag: run_production_tests
        reason: "Requires --flag run_production_tests"
    # ...
```

### Matrix Testing

Run scenarios with multiple parameter combinations:

```yaml
scenarios:
  - name: test_payment_providers
    matrix:
      provider: [stripe, paypal, square]
      currency: [USD, EUR, GBP]
    flow:
      - task: ProcessPayment
        params:
          provider: ${{ matrix.provider }}
          currency: ${{ matrix.currency }}
```

This creates 9 scenario variations (3 providers × 3 currencies).

### Inheritance

Extend base scenarios:

```yaml
scenarios:
  - name: base_user_flow
    abstract: true  # Cannot be run directly
    flow:
      - setup: CreateUser
    teardown:
      - teardown: DeleteUser

  - name: user_places_order
    extends: base_user_flow
    flow:
      - task: CreateOrder
      - validation: VerifyOrder
```

## Chaos and Mock Profiles

Apply chaos or mock profiles to scenarios:

```yaml
scenarios:
  - name: test_with_latency
    chaos_profiles: [network_latency]
    flow:
      - task: CallExternalAPI
      - validation: VerifyTimeout

  - name: test_payment_failure
    mock_profiles: [payment_declined]
    flow:
      - task: ProcessPayment
      - validation: VerifyPaymentFailed
```

## Runtime Flags

Set flags for conditional behavior:

```yaml
scenarios:
  - name: configurable_test
    flags:
      use_cache: true
      log_level: debug
    flow:
      - task: ProcessWithOptions
```

Override at runtime: `chronicle run --flag use_cache=false`

## Suites

Group scenarios into suites:

```yaml
suites:
  smoke:
    description: Quick smoke tests
    scenarios: [login_test, basic_order]
    parallel: 4
    fail_fast: true

  integration:
    description: Full integration tests
    tags: [integration]  # Include all with this tag
    exclude_tags: [slow]  # Exclude these
    parallel: 2

  regression:
    description: Complete regression suite
    scenarios: [test_a, test_b, test_c]
```

Run a suite: `chronicle run --suite smoke`

## Programmatic Builder

Build scenarios in Go code:

```go
import "github.com/joshua-temple/chronicle/pkg/scenario"

s := scenario.NewBuilder("user_creates_order").
    Description("A user successfully creates an order").
    Tags("order", "happy-path").
    Timeout(5 * time.Minute).
    Setup("CreateUser").
    Setup("CreateCart").
    Task("AddItemToCart").
    Task("CreateOrder").
    Validation("VerifyOrderCreated").
    Teardown("DeleteOrder").
    Teardown("DeleteUser").
    Build()
```

### Builder Methods

| Method | Description |
|--------|-------------|
| `Description(desc)` | Set scenario description |
| `Timeout(duration)` | Set execution timeout |
| `Tags(tags...)` | Add tags |
| `Extends(parent)` | Inherit from another scenario |
| `Abstract()` | Mark as abstract (not runnable) |
| `Setup(name)` | Add setup component |
| `SetupWithTimeout(name, d)` | Add setup with timeout |
| `Task(name)` | Add task component |
| `TaskWithParams(name, params)` | Add task with parameters |
| `Validation(name)` | Add validation component |
| `Step(name)` | Add step component |
| `Rollup(name)` | Add rollup component |
| `Teardown(name)` | Add teardown component |
| `Flow(item)` | Add custom flow item |
| `Parallel(items...)` | Add parallel execution block |
| `Flag(key, value)` | Set a runtime flag |
| `Flags(map)` | Set multiple flags |
| `Options(names...)` | Enable option bundles |
| `ChaosProfiles(names...)` | Apply chaos profiles |
| `MockProfiles(names...)` | Apply mock profiles |
| `SkipIf(expr, reason)` | Add skip condition |
| `SkipUnless(expr, reason)` | Add skip-unless condition |
| `Matrix(key, values)` | Add matrix parameter |
| `Build()` | Return the scenario |
| `MustBuild()` | Return scenario, panic on error |

### Parallel Execution

Run components in parallel:

```go
s := scenario.NewBuilder("parallel_tasks").
    Setup("CommonSetup").
    Parallel(
        scenario.NewFlowItem(core.ComponentTask, "TaskA"),
        scenario.NewFlowItem(core.ComponentTask, "TaskB"),
        scenario.NewFlowItem(core.ComponentTask, "TaskC"),
    ).
    Validation("VerifyAll").
    Build()
```

### Custom Flow Items

Create flow items with full control:

```go
item := scenario.NewFlowItem(core.ComponentTask, "ProcessOrder").
    WithTimeout(30 * time.Second).
    WithParams(map[string]any{
        "retry": true,
        "maxAttempts": 3,
    }).
    WithDependsOn("CreateUser", "CreateCart")

s := scenario.NewBuilder("custom_flow").
    Flow(item).
    Build()
```

## Execution

### Running Scenarios

```bash
# Run all scenarios
chronicle run

# Run specific scenarios by name
chronicle run user_creates_order checkout_flow

# Run by tag
chronicle run --tags smoke
chronicle run --tags integration --exclude-tags slow

# Run a suite
chronicle run --suite regression

# Parallel execution
chronicle run --parallel 4

# Stop on first failure
chronicle run --fail-fast

# Dry run (show what would execute)
chronicle run --dry-run
```

### Runtime Modifiers

```bash
# Set flags
chronicle run --flag environment=staging --flag debug=true

# Enable option bundles
chronicle run --option verbose_logging

# Apply chaos profiles
chronicle run --chaos network_latency

# Apply mock profiles
chronicle run --mock payment_declined

# Set timeout
chronicle run --timeout 1h
```

## Best Practices

1. **Name Clearly** - Use descriptive names like `user_checkout_with_coupon`
2. **Tag Consistently** - Use standard tags: `smoke`, `integration`, `regression`
3. **Use Suites** - Group related scenarios for easier execution
4. **Set Timeouts** - Always set appropriate timeouts
5. **Test Failures** - Include scenarios with chaos/mocks to test error paths
6. **Keep Focused** - Each scenario should test one user journey
