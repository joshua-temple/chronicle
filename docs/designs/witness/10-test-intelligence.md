# Test Intelligence

> Data management, profiling, flaky detection, and impact analysis.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Extensibility](./09-extensibility.md) | [Overview](./00-overview.md) | [Getting Started](./11-getting-started.md) |

---

## Table of Contents

- [Test Data Management](#test-data-management)
- [Contract Testing](#contract-testing)
- [Performance Profiling](#performance-profiling)
- [Flaky Test Detection](#flaky-test-detection)
- [Test Impact Analysis](#test-impact-analysis)

---

## Test Data Management

Manage test data across runs with fixtures, generators, and snapshots.

### Fixtures

Static test data loaded from files:

```yaml
data:
  fixtures:
    - name: standard-user-set
      source: fixtures/users.json
      refresh: per_suite  # or: per_test, manual

    - name: product-catalog
      source: fixtures/products.yaml
      refresh: per_suite

    - name: payment-methods
      source: fixtures/payments.json
      refresh: manual  # Only reload when explicitly requested
```

### Generators

Dynamic test data generation:

```yaml
data:
  generators:
    - name: random-user
      type: faker
      schema:
        id: "{{uuid}}"
        email: "{{internet.email}}"
        name: "{{name.full}}"
        created_at: "{{date.recent}}"

    - name: random-order
      type: faker
      schema:
        id: "{{uuid}}"
        amount: "{{finance.amount}}"
        currency: "{{finance.currencyCode}}"
        items:
          count: "{{random.number min=1 max=10}}"
          template:
            sku: "SKU-{{random.alphanumeric length=8}}"
            quantity: "{{random.number min=1 max=5}}"
```

### Using Data in Components

```go
// @witness:setup name="LoadUsers" produces="users:[]User"
func LoadUsers(ctx witness.Context) error {
    users := witness.Fixture[[]User](ctx, "standard-user-set")
    witness.Set(ctx, "users", users)
    return nil
}

// @witness:setup name="CreateRandomUser" produces="user:User"
func CreateRandomUser(ctx witness.Context) error {
    user := witness.Generate[User](ctx, "random-user")
    witness.Set(ctx, "user", user)
    return nil
}
```

### Snapshots

Compare outputs against known-good baselines:

```yaml
data:
  snapshots:
    enabled: true
    directory: ./snapshots
    compare_on:
      - order.total
      - order.items.length
      - response.status
    update_mode: manual  # or: auto
```

```go
// @witness:validation name="OrderMatchesSnapshot"
func OrderMatchesSnapshot(ctx witness.Context, result any) error {
    order := result.(*Order)
    return witness.MatchSnapshot(ctx, "order-output", order)
}
```

```bash
# Update snapshots
witness run --scenario checkout-flow --update-snapshots

# Compare against snapshots
witness run --scenario checkout-flow  # Fails if mismatch
```

---

## Contract Testing

Ensure services comply with agreed contracts.

### Consumer-Driven Contracts

```yaml
contracts:
  - provider: payment-service
    consumer: checkout-service
    pact: ./contracts/payment.pact.json

  - provider: inventory-service
    consumer: order-service
    pact: ./contracts/inventory.pact.json
```

### Record/Replay

Record real interactions for later replay:

```yaml
mocks:
  payment-gateway:
    mode: record_replay
    record_path: ./recordings/payment
    fallback: stub  # If no recording matches
```

```bash
# Record mode - capture real responses
witness run --scenario checkout-flow --record-mocks

# Replay mode - use recorded responses
witness run --scenario checkout-flow --replay-mocks
```

### Contract Verification

```bash
# Verify provider against contracts
witness contract verify --provider payment-service

# Verify consumer expectations
witness contract verify --consumer checkout-service
```

---

## Performance Profiling

Track performance over time and detect regressions.

### Configuration

```yaml
profiling:
  enabled: true
  baseline: ./baselines/perf.json

  thresholds:
    warn: 1.5x   # 50% slower than baseline
    fail: 2.0x   # 100% slower than baseline

  track:
    - component_duration
    - scenario_duration
    - infra_latency
    - memory_usage

  aggregation:
    strategy: p95  # or: mean, median, p99
```

### Baseline Management

```bash
# Create baseline from current run
witness run --scenario checkout-flow --save-baseline

# Compare against baseline
witness run --scenario checkout-flow --compare-baseline

# Update baseline
witness baseline update --scenario checkout-flow
```

### Profiling Report

```
┌─ Performance Report ──────────────────────────────────────┐
│                                                           │
│  Scenario: checkout-flow                                  │
│  Baseline: 2024-01-10                                     │
│                                                           │
│  Component          Baseline    Current    Delta          │
│  ─────────────────  ────────    ───────    ─────          │
│  CreateUser         120ms       125ms      +4%            │
│  SeedInventory       45ms        48ms      +7%            │
│  SubmitOrder        230ms       450ms      +96% ⚠️        │
│  ProcessPayment     180ms       190ms      +6%            │
│  ValidateOrder       15ms        16ms      +7%            │
│                                                           │
│  Total              590ms       829ms      +41% ⚠️        │
│                                                           │
│  ⚠️  Performance regression detected                      │
│     SubmitOrder: 450ms (baseline: 230ms) - 1.96x slower  │
└───────────────────────────────────────────────────────────┘
```

---

## Flaky Test Detection

Automatically detect and manage flaky tests.

### Configuration

```yaml
flakiness:
  detection:
    enabled: true
    window: 50_runs        # Look at last 50 runs
    threshold: 0.1         # 10% failure rate = flaky
    min_runs: 10           # Minimum runs before flagging

  handling:
    quarantine: true       # Isolate flaky tests
    auto_retry: 3          # Retry flaky tests more
    notify: [slack:#flaky] # Alert on new flaky tests

  dashboard: true          # Track flakiness trends
```

### Flaky Test Handling

```yaml
scenarios:
  - name: checkout-flow
    # Automatically added when detected as flaky
    flaky:
      detected: true
      first_seen: 2024-01-15
      failure_rate: 0.12
      quarantined: true

    # Override quarantine
    quarantine_override: false
```

### CLI Commands

```bash
# List flaky tests
witness flaky list

# Quarantine a test
witness flaky quarantine checkout-flow

# Un-quarantine
witness flaky restore checkout-flow

# View flakiness report
witness flaky report --format html
```

### Flakiness Dashboard

```
┌─ Flaky Tests Dashboard ───────────────────────────────────┐
│                                                           │
│  Flaky Tests: 3                    Quarantined: 2         │
│                                                           │
│  Scenario              Failure Rate   Last 7 Days         │
│  ─────────────────────  ────────────  ───────────         │
│  checkout-flow          12%           ▓▓░▓░▓▓░░░         │
│  payment-retry          8%            ░▓░░▓░░▓░░         │
│  inventory-sync         15%           ▓░▓▓░▓▓░▓░         │
│                                                           │
│  [View Details] [Un-quarantine All] [Export Report]      │
└───────────────────────────────────────────────────────────┘
```

---

## Test Impact Analysis

Determine which tests to run based on code changes.

### Configuration

```yaml
impact_analysis:
  enabled: true

  # Map code paths to components
  mappings:
    - paths: ["internal/orders/**"]
      components: [CreateOrder, ValidateOrder, OrderComplete]

    - paths: ["internal/payments/**"]
      components: [ProcessPayment, ValidatePayment]

    - paths: ["internal/users/**"]
      components: [CreateUser, ValidateUser]

  # Auto-discover from imports (optional)
  auto_discover: true
```

### Usage

```bash
# Run only affected tests based on git diff
witness run --impact-analysis

# Preview affected tests
witness impact --preview

# Impact against specific commit
witness run --impact-analysis --base main
```

### Output

```
┌─ Impact Analysis ─────────────────────────────────────────┐
│                                                           │
│  Changed Files:                                           │
│    • internal/orders/service.go                          │
│    • internal/orders/repository.go                       │
│                                                           │
│  Affected Components:                                     │
│    • CreateOrder                                         │
│    • ValidateOrder                                       │
│                                                           │
│  Scenarios to Run:                                       │
│    • checkout-flow                                       │
│    • order-cancellation                                  │
│                                                           │
│  Skipping: 13 scenarios (not affected)                   │
│                                                           │
│  Running 2 of 15 scenarios                               │
└───────────────────────────────────────────────────────────┘
```

### Integration with CI

```yaml
# GitHub Actions
- name: Run affected tests
  run: |
    witness run --impact-analysis --base ${{ github.event.pull_request.base.sha }}
```

---

## AI-Assisted Features (Future)

### Test Generation

```bash
# Generate scenarios from OpenAPI spec
witness generate --from-openapi ./api/openapi.yaml

# Suggest tests for uncovered code
witness suggest --coverage

# Generate edge case tests
witness generate --edge-cases --component CreateOrder
```

### Natural Language Explanation

```bash
# Explain what a scenario does
witness explain --scenario checkout-flow

# Output:
# This scenario tests the complete checkout flow:
# 1. Creates a test user with random email
# 2. Seeds the inventory with 100 items
# 3. Adds items to the user's cart
# 4. Submits the order to the order service
# 5. Validates the order was created successfully
# 6. Verifies inventory was decremented
```

---

## Next Steps

Continue to [Getting Started](./11-getting-started.md) for project structure and quickstart guide.
