# Scenarios & Composition

> Scenario structure, chaos engineering, flags, options, and mocking.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Infrastructure](./02-infrastructure.md) | [Overview](./00-overview.md) | [Execution](./04-execution.md) |

---

## Table of Contents

- [Scenario Structure](#scenario-structure)
  - [Flow Keyword](#flow-keyword)
  - [Scenario Inheritance](#scenario-inheritance)
  - [Parameterized Scenarios](#parameterized-scenarios)
  - [Conditional Execution](#conditional-execution)
- [Chaos Engineering](#chaos-engineering)
  - [Infrastructure Chaos](#infrastructure-chaos)
  - [Application Chaos](#application-chaos)
  - [Chaos Profiles](#chaos-profiles)
- [Flags](#flags)
  - [Flag Definition](#flag-definition)
  - [Flag Injection](#flag-injection)
  - [Flag Matrix](#flag-matrix)
- [Options](#options)
  - [Option Definition](#option-definition)
  - [Option Mutations](#option-mutations)
  - [Option Composition](#option-composition)
- [Mocking](#mocking)
  - [Mock Definition](#mock-definition)
  - [Mock Injector Interface](#mock-injector-interface)
  - [Mock Profiles](#mock-profiles)
  - [Mock Verification](#mock-verification)
- [Bundle Registry](#bundle-registry)
  - [Infrastructure Bundles](#infrastructure-bundles)
  - [Flag Bundles](#flag-bundles)
  - [Option Bundles](#option-bundles)
  - [Middleware Bundles](#middleware-bundles)

---

## Scenario Structure

Scenarios are the test encapsulation layer. They define execution paths.

### Basic Scenario

```yaml
scenarios:
  - name: checkout-flow
    description: "Complete checkout with payment processing"
    timeout: 30s

    flow:
      - setup: CreateUser
      - setup: SeedInventory
      - step: AddItemsToCart
      - task: SubmitOrder
      - validation: OrderCreated
      - validation: InventoryDecremented
```

### Flow Keyword

The `flow` keyword supports any component type - not just atomics:

```yaml
flow:
  - setup: CreateUser           # atomic
  - task: Checkout              # atomic
  - validation: OrderComplete   # atomic
  - step: AddAndValidateCart    # bundles 2+ atomics
  - rollup: FullE2ECheckout     # bundles steps/rollups
```

### UI Expanded View

```
│  Collapsed:                    Expanded:                  │
│  ───────────                   ──────────                 │
│  ● CreateUser                  ● CreateUser               │
│  ● SeedCart                    ● SeedCart                 │
│  ● CheckoutAndProcessPayment   ● CheckoutAndProcessPayment│
│                                  ├─ Checkout              │
│                                  ├─ ProcessPayment        │
│                                  └─ OrderComplete         │
```

### Scenario Inheritance

DRY for similar scenarios:

```yaml
scenarios:
  - name: checkout-base
    abstract: true  # Can't run directly
    flow:
      - setup: CreateUser
      - setup: SeedInventory
      - step: AddItemsToCart

  - name: checkout-credit-card
    extends: checkout-base
    flow:
      - task: PayWithCreditCard
      - validation: PaymentComplete

  - name: checkout-paypal
    extends: checkout-base
    flow:
      - task: PayWithPayPal
      - validation: PaymentComplete
```

### Parameterized Scenarios (Matrix)

Run same scenario with different inputs:

```yaml
scenarios:
  - name: checkout-multi-currency
    matrix:
      currency: [USD, EUR, GBP, JPY]
      quantity: [1, 10, 100]

    flow:
      - setup: CreateUser
      - task: AddToCart
        params:
          quantity: ${{ matrix.quantity }}
      - task: Checkout
        params:
          currency: ${{ matrix.currency }}
      - validation: OrderTotal

# Generates: 4 currencies × 3 quantities = 12 test runs
```

### Conditional Execution

#### Skip Conditions

```yaml
scenarios:
  - name: production-smoke
    skip_if:
      - condition: env.SKIP_SMOKE == "true"
        reason: "Smoke tests disabled via environment"
      - condition: flags.maintenance_mode == true
        reason: "System in maintenance mode"
    flow: [...]

  - name: database-migration
    skip_unless:
      - condition: env.DATABASE_URL is set
        reason: "Requires DATABASE_URL"
      - condition: env.ENVIRONMENT in ["staging", "production"]
        reason: "Only runs in staging/production"
    flow: [...]
```

#### Condition Syntax

| Expression | Meaning |
|------------|---------|
| `env.VAR == "value"` | Environment variable equals |
| `env.VAR is set` | Environment variable exists |
| `env.VAR is empty` | Environment variable is empty or unset |
| `env.VAR in ["a", "b"]` | Environment variable in list |
| `flags.name == true` | Flag value check |
| `time.hour >= 9 and time.hour < 17` | Time-based (business hours) |
| `weekday in ["mon", "tue", "wed"]` | Day-based |

#### Runtime Skip

```go
// @witness:setup name="ConditionalSetup"
func ConditionalSetup(ctx witness.Context) error {
    if someCondition {
        return witness.Skip("Condition not met: %v", reason)
    }
    // Continue with setup
}
```

---

## Chaos Engineering

Chaos testing at both infrastructure and application layers.

### Infrastructure Chaos

| Type | Description |
|------|-------------|
| `network_latency` | Inject delay to service calls |
| `network_partition` | Block traffic between services |
| `service_unavailable` | Kill/pause a service |
| `resource_exhaustion` | Limit CPU/memory |
| `disk_full` | Simulate storage issues |

### Application Chaos

| Type | Description |
|------|-------------|
| `invalid_input` | Malformed/unexpected values |
| `boundary_value` | Edge cases (0, max, negative) |
| `injection` | SQL/XSS/command injection attempts |
| `timing` | Race conditions, out-of-order |
| `state_corruption` | Invalid state transitions |

### Inline Chaos

```yaml
scenarios:
  - name: checkout-under-stress
    flow:
      - setup: CreateUser
      - task: SubmitOrder
      - validation: OrderCreated

    chaos:
      enabled: true
      infrastructure:
        - type: network_latency
          target: payment-service
          latency_ms: 500
          probability: 0.3
        - type: service_unavailable
          target: redis
          duration: 5s
      application:
        - type: invalid_input
          target: SubmitOrder
          generator: malformed_credit_card
        - type: boundary_value
          target: AddItemsToCart
          field: quantity
          values: [0, -1, 999999]
```

### Chaos Profiles

Define reusable chaos configurations:

```yaml
chaos_profiles:
  degraded-network:
    infrastructure:
      - type: network_latency
        target: "*"
        latency_ms: 200
        probability: 0.5

  payment-outage:
    infrastructure:
      - type: service_unavailable
        target: payment-service
        duration: 10s

  fuzz-inputs:
    application:
      - type: invalid_input
        generator: random_unicode
      - type: boundary_value
        fields: ["*numeric*"]
        values: [0, -1, MAX_INT]
```

### Using Profiles

```yaml
scenarios:
  - name: checkout-under-stress
    flow: [...]
    chaos:
      profiles: [degraded-network, fuzz-inputs]  # Compose multiple

  - name: checkout-payment-failure
    flow: [...]
    chaos:
      profiles: [payment-outage]
```

---

## Flags

Feature flag injection into services under test.

### Flag Definition

```yaml
flags:
  definitions:
    new-checkout-flow:
      type: boolean
      default: false

    payment-provider:
      type: enum
      values: [stripe, braintree, adyen]
      default: stripe

    rate-limit-threshold:
      type: number
      default: 100
```

### Flag Injection

How flags get to your services:

```yaml
flags:
  injection:
    # Environment variables
    - method: env
      mapping:
        new-checkout-flow: FEATURE_NEW_CHECKOUT
        payment-provider: PAYMENT_PROVIDER

    # HTTP call to flag service
    - method: api
      endpoint: http://flagservice:8080/flags
      auth: ${FLAG_SERVICE_TOKEN}

    # Config file mount
    - method: file
      path: /etc/service/flags.json
      format: json
```

### Using Flags in Scenarios

```yaml
scenarios:
  - name: checkout-new-flow
    flags:
      new-checkout-flow: true
      payment-provider: stripe
    flow:
      - setup: CreateUser
      - task: CheckoutWithNewFlow
      - validation: OrderComplete
```

### Flag Matrix

Test all flag combinations:

```yaml
scenarios:
  - name: checkout-flag-matrix
    flag_matrix:
      new-checkout-flow: [true, false]
      payment-provider: [stripe, braintree]
    # Generates 2×2 = 4 scenario runs
    flow: [...]
```

### CLI Override

```bash
witness run --scenario checkout \
  --flag new-checkout-flow=true \
  --flag payment-provider=adyen
```

### Flag-Aware Components

```go
// @witness:task name="ProcessPayment" requires="order:Order"
func ProcessPayment(ctx witness.Context) (*PaymentResult, error) {
    provider := witness.Flag[string](ctx, "payment-provider")

    switch provider {
    case "stripe":
        return processStripe(ctx)
    case "braintree":
        return processBraintree(ctx)
    }
    return nil, errors.New("unknown provider")
}
```

---

## Options

Scenario mutations/variants without duplication.

### Option Definition

```yaml
options:
  definitions:
    with-slow-network:
      description: "Simulate degraded network"
      applies:
        chaos:
          profiles: [degraded-network]

    as-admin-user:
      description: "Run as admin instead of regular user"
      applies:
        steps:
          replace:
            - from: CreateUser
              to: CreateAdminUser

    with-empty-cart:
      description: "Start with empty cart"
      applies:
        steps:
          remove: [SeedCart]

    high-volume:
      description: "Test with large quantities"
      applies:
        params:
          quantity: 10000

    skip-payment:
      description: "Mock payment step"
      applies:
        steps:
          replace:
            - from: ProcessPayment
              to: MockPayment
```

### Option Mutations

| Mutation Type | Description | Example |
|---------------|-------------|---------|
| `steps.replace` | Swap one component for another | CreateUser → CreateAdminUser |
| `steps.remove` | Remove a step | Remove SeedCart |
| `steps.insert_before` | Add step before another | Add LoginStep before Checkout |
| `steps.insert_after` | Add step after another | Add AuditLog after Payment |
| `params` | Override parameters | quantity: 10000 |
| `chaos` | Apply chaos profiles | Add network latency |
| `flags` | Set flag values | new-checkout-flow: true |
| `timeout` | Adjust timeout | timeout: 5m |
| `skip_validations` | Skip certain validations | Skip InventoryCheck |

### Using Options

```yaml
scenarios:
  - name: checkout-flow
    flow:
      - setup: CreateUser
      - setup: SeedCart
      - task: Checkout
      - task: ProcessPayment
      - validation: OrderComplete

  # Variant defined inline
  - name: checkout-admin
    extends: checkout-flow
    options: [as-admin-user]

  # Compose multiple options
  - name: checkout-stress-test
    extends: checkout-flow
    options: [as-admin-user, high-volume, with-slow-network]
```

### CLI Option Application

```bash
# Apply options at runtime
witness run --scenario checkout-flow \
  --option with-slow-network \
  --option high-volume

# Mix flags and options
witness run --scenario checkout-flow \
  --flag new-checkout-flow=true \
  --option with-slow-network
```

### Option Composition Rules

```yaml
options:
  definitions:
    option-a:
      conflicts_with: [option-b]  # Can't combine
      requires: [option-c]         # Must also apply

    option-b:
      exclusive_group: payment-mocks  # Only one from group

    mock-stripe:
      exclusive_group: payment-mocks
```

### Flags vs Options

| Aspect | Flags | Options |
|--------|-------|---------|
| **Purpose** | Control service behavior | Mutate test structure |
| **Scope** | Injected into service under test | Modify scenario execution |
| **Persistence** | Lives in service runtime | Lives in test framework |
| **Example** | "Enable new checkout feature" | "Run as admin user" |

---

## Mocking

Scenario-scoped mocks for services under test.

### Mock Definition

Framework is schema-agnostic - users define mock format based on their injector:

```yaml
scenarios:
  - name: checkout-with-mocks
    mocks:
      payment-gateway:
        # Opaque to framework - passed directly to user's injector
        # User defines schema based on their injector implementation
        type: wiremock
        stubs:
          - request: { method: POST, url: /charge }
            response: { status: 200, body: { id: "ch_123" } }

      order-service:
        type: grpc-mock
        proto: ./protos/order.proto
        responses:
          GetOrder:
            return: { id: "123", status: "pending" }

    flow:
      - setup: CreateUser
      - step: CheckoutFlow
      - validation: PaymentRecorded
```

### Mock Injector Interface

Framework provides the interface, users implement injection mechanism:

```go
type MockInjector interface {
    // Framework passes raw config - injector interprets it
    Setup(ctx context.Context, mocks map[string]any) error
    Teardown(ctx context.Context) error
    Verify(ctx context.Context, mockName string) (any, error)
}
```

### User Implementation Examples

```go
// WireMock-based injection
type WireMockInjector struct {
    endpoint string
}

func (w *WireMockInjector) Setup(ctx context.Context, mocks map[string]any) error {
    for name, config := range mocks {
        // POST to WireMock admin API
        w.registerStubs(name, config)
    }
    return nil
}

// In-process mock server
type InProcessMockInjector struct {
    servers map[string]*httptest.Server
}

func (i *InProcessMockInjector) Setup(ctx context.Context, mocks map[string]any) error {
    for name, config := range mocks {
        server := i.createMockServer(config)
        i.servers[name] = server
        os.Setenv(name+"_URL", server.URL)
    }
    return nil
}
```

### Registering Injector

```yaml
mocks:
  injector: wiremock  # or: in-process, custom

  wiremock:
    endpoint: http://wiremock:8080

  custom:
    path: ./mocks/injector.go
```

### Mock Profiles

Reusable mock configurations:

```yaml
mock_profiles:
  payment-always-succeeds:
    payment-gateway:
      type: wiremock
      stubs:
        - request: { method: POST, path: /v1/charges }
          response: { status: 200, body: { status: succeeded } }

  payment-always-fails:
    payment-gateway:
      type: wiremock
      stubs:
        - request: { method: POST, path: /v1/charges }
          response: { status: 402, body: { error: declined } }
```

### Using Mock Profiles

```yaml
scenarios:
  - name: checkout-payment-failure
    mock_profiles: [payment-always-fails]
    flow: [...]

  - name: checkout-success
    mock_profiles: [payment-always-succeeds]
    chaos:
      profiles: [degraded-network]  # Mocks + chaos compose
    flow: [...]
```

### Mock Verification

#### Verification Modes

```go
type MockVerificationMode string

const (
    // Strict - all registered stubs must be called exactly once
    VerificationStrict MockVerificationMode = "strict"

    // Lenient - stubs may be called any number of times (including zero)
    VerificationLenient MockVerificationMode = "lenient"

    // AtLeastOnce - all stubs must be called at least once
    VerificationAtLeastOnce MockVerificationMode = "at_least_once"
)
```

#### Verification Configuration

```yaml
mocks:
  verification:
    mode: at_least_once  # strict | lenient | at_least_once
    on_failure: fail     # fail | warn | ignore
    run_at: scenario_end # scenario_end | test_end | teardown

  payment-gateway:
    stubs:
      - request: { method: POST, path: /charge }
        response: { status: 200 }
        # Per-stub verification override
        expected_calls: 1  # Exactly once
        # Or: expected_calls: "1-3"  # Between 1 and 3
        # Or: expected_calls: "1+"   # At least once
```

#### Verification Results

```go
type MockVerificationResult struct {
    MockName      string
    Stub          string
    ExpectedCalls string
    ActualCalls   int
    Passed        bool
    UnmatchedCalls []UnmatchedCall  // Calls that didn't match any stub
}
```

#### Programmatic Verification

```go
// @witness:validation name="VerifyPaymentCalls" requires="order:Order"
func VerifyPaymentCalls(ctx witness.Context, result any) error {
    verification, err := witness.VerifyMock(ctx, "payment-gateway")
    if err != nil {
        return err
    }

    if verification.CallCount("/charge") != 1 {
        return fmt.Errorf("expected 1 charge call, got %d",
            verification.CallCount("/charge"))
    }

    return nil
}
```

#### Unmatched Call Handling

```yaml
mocks:
  unmatched_calls:
    action: record  # record | fail | passthrough

    # If passthrough, forward to real service
    passthrough:
      payment-gateway: https://real-payment.internal
```

---

## Bundle Registry

Pre-packaged, reusable configurations for common patterns. Bundles provide named collections of infrastructure, flags, options, and middleware.

### Infrastructure Bundles

Named combinations of infrastructure services:

```yaml
bundles:
  infrastructure:
    standard-web-stack:
      description: "Common web application infrastructure"
      services:
        - postgres:
            provider: postgres
            config:
              image: postgres:15
        - redis:
            provider: redis
            config:
              image: redis:7

    event-driven-stack:
      description: "Event-driven microservices infrastructure"
      services:
        - kafka:
            provider: kafka
            config:
              image: confluentinc/cp-kafka:7.5.0
        - zookeeper:
            provider: zookeeper
            config:
              image: confluentinc/cp-zookeeper:7.5.0
        - schema-registry:
            provider: schema-registry
            config:
              image: confluentinc/cp-schema-registry:7.5.0

    aws-local:
      description: "LocalStack for AWS services"
      services:
        - localstack:
            provider: localstack
            config:
              services: [s3, sqs, dynamodb, lambda]
```

**Using Infrastructure Bundles:**

```yaml
infrastructure:
  bundle: standard-web-stack  # Use pre-defined bundle
  additional:                  # Add more services
    - elasticsearch:
        provider: elasticsearch
```

### Flag Bundles

Named groups of related feature flags:

```yaml
bundles:
  flags:
    new-checkout-experience:
      description: "All flags for new checkout"
      values:
        new-checkout-flow: true
        instant-payment: true
        one-click-buy: true

    legacy-mode:
      description: "Disable all new features"
      values:
        new-checkout-flow: false
        instant-payment: false
        one-click-buy: false
        new-inventory-system: false

    payment-v2:
      description: "New payment processing flags"
      values:
        payment-provider: stripe-v2
        payment-retry-enabled: true
        payment-timeout-ms: 30000
```

**Using Flag Bundles:**

```yaml
scenarios:
  - name: checkout-new-experience
    flag_bundles: [new-checkout-experience, payment-v2]
    flow: [...]

  - name: checkout-legacy
    flag_bundles: [legacy-mode]
    flow: [...]
```

**CLI:**

```bash
witness run --scenario checkout --flag-bundle new-checkout-experience
```

### Option Bundles

Named groups of scenario mutations:

```yaml
bundles:
  options:
    admin-testing:
      description: "Run scenarios as admin user"
      options:
        - as-admin-user
        - with-elevated-permissions
        - bypass-rate-limit

    stress-testing:
      description: "High volume with chaos"
      options:
        - high-volume
        - with-slow-network
        - extended-timeout

    minimal-setup:
      description: "Skip optional setup steps"
      options:
        - skip-analytics
        - skip-notifications
        - mock-external-services
```

**Using Option Bundles:**

```yaml
scenarios:
  - name: checkout-admin-stress
    option_bundles: [admin-testing, stress-testing]
    flow: [...]
```

### Middleware Bundles

Named groups of middleware configurations:

```yaml
bundles:
  middleware:
    observability:
      description: "Full observability stack"
      middleware:
        - logging
        - tracing
        - metrics

    resilience:
      description: "Retry and timeout handling"
      middleware:
        - retry:
            max_attempts: 3
            backoff: exponential
        - timeout: 30s
        - circuit-breaker:
            threshold: 5

    debugging:
      description: "Verbose debugging"
      middleware:
        - logging:
            level: debug
        - tracing:
            verbose: true
        - slow-step-detection:
            threshold: 5s
```

**Using Middleware Bundles:**

```yaml
execution:
  middleware_bundles: [observability, resilience]

  scenarios:
    checkout-debugging:
      middleware_bundles: [debugging]  # Override with debug bundle
```

### Bundle Composition

Bundles can extend other bundles:

```yaml
bundles:
  infrastructure:
    base-services:
      services:
        - postgres
        - redis

    full-stack:
      extends: base-services
      services:
        - kafka
        - elasticsearch
```

### Bundle Registry API

```go
type BundleRegistry struct {
    Infrastructure map[string]InfrastructureBundle
    Flags          map[string]FlagBundle
    Options        map[string]OptionBundle
    Middleware     map[string]MiddlewareBundle
}

// Register custom bundles programmatically
func (r *BundleRegistry) RegisterInfrastructure(name string, bundle InfrastructureBundle)
func (r *BundleRegistry) RegisterFlags(name string, bundle FlagBundle)
func (r *BundleRegistry) RegisterOptions(name string, bundle OptionBundle)
func (r *BundleRegistry) RegisterMiddleware(name string, bundle MiddlewareBundle)

// Resolve bundles to concrete configuration
func (r *BundleRegistry) ResolveInfrastructure(names []string) ([]Service, error)
func (r *BundleRegistry) ResolveFlags(names []string) (map[string]any, error)
func (r *BundleRegistry) ResolveOptions(names []string) ([]Option, error)
func (r *BundleRegistry) ResolveMiddleware(names []string) ([]Middleware, error)
```

### Built-in Bundles

Framework ships with common bundles:

| Category | Bundle Name | Contents |
|----------|-------------|----------|
| Infrastructure | `postgres-redis` | PostgreSQL + Redis |
| Infrastructure | `kafka-stack` | Kafka + Zookeeper + Schema Registry |
| Infrastructure | `aws-local` | LocalStack with common services |
| Flags | `all-features-on` | Enable all feature flags |
| Flags | `all-features-off` | Disable all feature flags |
| Options | `stress-test` | High volume + chaos |
| Options | `quick-test` | Skip slow steps |
| Middleware | `observability` | Logging + Tracing + Metrics |
| Middleware | `resilience` | Retry + Timeout + Circuit Breaker |

---

## Next Steps

Continue to [Execution](./04-execution.md) for execution modes, scheduling, and distributed workers.
