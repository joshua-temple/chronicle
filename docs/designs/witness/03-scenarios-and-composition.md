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

---

## Next Steps

Continue to [Execution](./04-execution.md) for execution modes, scheduling, and distributed workers.
