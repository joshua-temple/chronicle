# Execution

> Execution modes, scheduling, and distributed workers.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Scenarios & Composition](./03-scenarios-and-composition.md) | [Overview](./00-overview.md) | [Results & Reporting](./05-results-and-reporting.md) |

---

## Table of Contents

- [Execution Modes](#execution-modes)
- [CLI Usage](#cli-usage)
- [Scheduling](#scheduling)
- [Distributed Execution](#distributed-execution)
- [Retry Policies](#retry-policies)

---

## Execution Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| **Adhoc** | Run single test/scenario on demand | Local dev, debugging |
| **Subset** | Run filtered tests (by tag, name, pattern) | PR checks, focused testing |
| **Full Suite** | Run everything | Nightly regression |
| **Scheduled** | Cron-based execution | Continuous validation |
| **Random Interval** | Randomized timing | Chaos/soak testing |
| **Triggered** | API-initiated | CI/CD integration |
| **Watch** | Re-run on file changes | Local dev loop |

---

## CLI Usage

### Adhoc Execution

```bash
# Run single scenario
witness run --scenario checkout-flow

# Run specific scenario with flags
witness run --scenario checkout-flow --flag new-checkout-flow=true
```

### Subset Execution

```bash
# By tags
witness run --tags "auth,critical" --exclude-tags "slow"

# By pattern
witness run --match "payment-*"

# By tag and pattern
witness run --tags smoke --match "checkout-*"
```

### Full Suite

```bash
# Run everything
witness run --all

# Run all in specific environment
witness run --all --env staging
```

### Watch Mode

```bash
# Re-run on file changes (local dev)
witness watch --tags unit

# Watch specific scenarios
witness watch --scenario checkout-flow
```

### Environment Selection

```bash
# Local development (default)
witness run --env local

# Against staging
witness run --env staging --tags integration

# Production smoke tests
witness run --env production --tags smoke
```

### Combined Examples

```bash
# Run checkout scenarios with new feature flag in staging
witness run --match "checkout-*" \
  --env staging \
  --flag new-checkout-flow=true \
  --option with-slow-network

# Run all critical tests except slow ones
witness run --tags critical --exclude-tags slow --env staging

# Matrix test with specific currencies
witness run --scenario checkout-multi-currency \
  --matrix-filter "currency=USD,EUR"
```

---

## Scheduling

For deployed service mode, tests can run on schedules.

### Cron-Based Scheduling

```yaml
schedules:
  - name: hourly-smoke
    cron: "0 * * * *"  # Every hour
    scenarios:
      tags: [smoke]
    on_failure:
      notify: [slack:#alerts, pagerduty]

  - name: nightly-regression
    cron: "0 2 * * *"  # 2 AM daily
    scenarios:
      tags: [regression]
    chaos:
      profiles: [degraded-network]
    on_success:
      notify: [slack:#test-results]
    on_failure:
      notify: [slack:#alerts, email:oncall@example.com]

  - name: weekly-full-suite
    cron: "0 3 * * 0"  # 3 AM Sundays
    scenarios:
      all: true
    timeout: 4h
```

### Random Interval Scheduling

For chaos/soak testing with unpredictable timing:

```yaml
schedules:
  - name: chaos-monkey
    interval:
      min: 15m
      max: 4h
    scenarios:
      random: true  # Pick random scenarios
      count: 5      # Run 5 at a time
    chaos:
      profiles: [random]  # Random chaos injection

  - name: continuous-validation
    interval:
      min: 5m
      max: 30m
    scenarios:
      tags: [critical]
    environment: production
```

### Schedule Management

```bash
# List schedules
witness schedule list

# Enable/disable schedule
witness schedule enable nightly-regression
witness schedule disable chaos-monkey

# Trigger scheduled job manually
witness schedule trigger hourly-smoke

# View schedule history
witness schedule history nightly-regression --last 10
```

---

## Distributed Execution

Fan out tests across multiple workers for parallel execution.

### Configuration

```yaml
execution:
  mode: distributed
  coordinator: redis://coordinator:6379

  workers:
    min: 2
    max: 20
    autoscale: true
    autoscale_threshold: 0.8  # Scale up at 80% utilization

  strategy: round-robin  # or: least-loaded, affinity

  parallelism:
    scenarios: true    # Different scenarios in parallel
    within_test: false # Steps within scenario sequential
```

### Worker Architecture

```
┌──────────────────┐     ┌──────────────────┐
│  Witness Service │     │   Worker Pool    │
│   (Coordinator)  │────▶│  ┌────────────┐  │
│                  │     │  │  Worker 1  │  │
│  - Job queue     │     │  ├────────────┤  │
│  - Result agg    │     │  │  Worker 2  │  │
│  - Health checks │     │  ├────────────┤  │
│                  │     │  │  Worker N  │  │
└──────────────────┘     │  └────────────┘  │
                         └──────────────────┘
```

### Worker Affinity

Route specific tests to specific workers:

```yaml
execution:
  affinity:
    rules:
      # GPU-intensive tests to GPU workers
      - match:
          tags: [gpu-required]
        workers:
          labels: [gpu]

      # High-memory tests to large workers
      - match:
          tags: [memory-intensive]
        workers:
          labels: [large]
```

### Distributed Scheduling

```yaml
schedules:
  - name: full-regression
    cron: "0 2 * * *"
    scenarios:
      all: true
    execution:
      mode: distributed
      workers:
        min: 10
        max: 50
      timeout_per_scenario: 5m
```

---

## Retry Policies

Handle transient failures with configurable retries.

### Scenario-Level Retry

```yaml
scenarios:
  - name: checkout-flow
    flow: [...]
    retry:
      max_attempts: 3
      backoff: exponential
      initial_delay: 1s
      max_delay: 30s
      retry_on:
        - network_error
        - timeout
```

### Global Retry Policy

```yaml
execution:
  retry:
    default:
      max_attempts: 2
      backoff: fixed
      delay: 5s

    # Override for specific scenarios
    overrides:
      - match:
          tags: [flaky]
        policy:
          max_attempts: 5
          backoff: exponential
```

### Backoff Strategies

| Strategy | Description |
|----------|-------------|
| `fixed` | Same delay between retries |
| `exponential` | Delay doubles each retry |
| `linear` | Delay increases linearly |
| `random` | Random delay within bounds |

### Retry Conditions

```yaml
retry:
  retry_on:
    - network_error       # Connection failures
    - timeout             # Timeout exceeded
    - service_unavailable # 503 responses
    - rate_limited        # 429 responses

  never_retry_on:
    - assertion_failure   # Test logic failures
    - validation_error    # Validation failures
```

---

## Next Steps

Continue to [Results & Reporting](./05-results-and-reporting.md) for storage adapters, report formats, and notifications.
