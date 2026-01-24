# Core Framework

> Component model, type system, and zero-wiring discovery.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| - | [Overview](./00-overview.md) | [Infrastructure](./02-infrastructure.md) |

---

## Table of Contents

- [Component Model](#component-model)
  - [Component Types](#component-types)
  - [Component Hierarchy](#component-hierarchy)
  - [Component Signatures](#component-signatures)
- [Typed Identifiers](#typed-identifiers)
  - [Core Identifier Types](#core-identifier-types)
  - [TraceID & Distributed Tracing](#traceid--distributed-tracing)
  - [Header Injection](#header-injection)
- [Type System](#type-system)
  - [Annotated Types](#annotated-types)
  - [Type Aliases](#type-aliases)
  - [Typed Context](#typed-context)
- [Middleware System](#middleware-system)
  - [Middleware Interface](#middleware-interface)
  - [Built-in Middleware](#built-in-middleware)
  - [Custom Middleware](#custom-middleware)
- [Discovery](#discovery)
  - [Annotation Format](#annotation-format)
  - [Dependency Declaration](#dependency-declaration)
  - [Discovery Process](#discovery-process)
- [Context & State](#context--state)

---

## Component Model

The heart of the WYSIWYG experience. Users write functions, the framework discovers and exposes them for composition.

### Component Types

| Type | Purpose | Description |
|------|---------|-------------|
| **Setup** | Stage preconditions | "Given" - prepare state before action |
| **Task** | Execute action | "When" - do something to the service |
| **Validation** | Assert outcomes | "Then" - prove something was done |
| **Step** | Reusable bundle | Combination of 2+ Setup/Task/Validation |
| **Rollup** | Higher composition | Combination of Steps/Rollups |
| **Scenario** | Test encapsulation | Complete test path with metadata |

### Component Hierarchy

```
┌──────────────────────────────────────────────────────────────┐
│  SCENARIO (encapsulates a complete test path)                │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  ROLLUP (recursive composition of rollups/steps)       │  │
│  │  ┌──────────────────────────────────────────────────┐  │  │
│  │  │  STEP (reusable combination of setup/task/val)   │  │  │
│  │  │  ┌────────────┬────────────┬────────────┐        │  │  │
│  │  │  │   SETUP    │    TASK    │ VALIDATION │        │  │  │
│  │  │  │  (Given)   │   (When)   │   (Then)   │        │  │  │
│  │  │  └────────────┴────────────┴────────────┘        │  │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### Step Flexibility

A **Step** is a combination of at least 2 of Setup/Task/Validation, but can include any number of each:

```
Step = at least 2 of (Setup | Task | Validation)
     = can have N of each in any combination

Examples:
  - Setup → Task                    (stage then act)
  - Task → Validation               (act then verify)
  - Setup → Setup → Task            (multi-stage then act)
  - Setup → Task → Validation → Validation  (full flow, multi-assert)
  - Task → Task → Validation        (chain actions, verify end state)
```

### Component Signatures

**Go:**

```go
// Setup - returns error only
type SetupFunc func(ctx witness.Context) error

// Task - returns result and error
type TaskFunc[T any] func(ctx witness.Context) (T, error)

// Validation - receives result from upstream, returns error
type ValidationFunc func(ctx witness.Context, result any) error

// Step - composite, returns error
type StepFunc func(ctx witness.Context) error

// Rollup - composite, returns error
type RollupFunc func(ctx witness.Context) error
```

---

## Typed Identifiers

The framework uses distinct types for identifiers to prevent bugs from mixing up IDs at compile time.

### Core Identifier Types

```go
// Distinct types prevent mixing up identifiers
type TestID string       // Unique test identifier
type ScenarioID string   // Unique scenario identifier
type ComponentID string  // Component registration ID
type ServiceID string    // Infrastructure service ID
type PortID string       // Service port identifier
type TagID string        // Tag for filtering/grouping
type SuiteID string      // Test suite identifier
type EnvID string        // Environment identifier
type TraceID string      // Distributed tracing identifier
type RunID string        // Execution run identifier
```

### IDRegistry

The framework maintains a registry of all active identifiers:

```go
type IDRegistry struct {
    tests      map[TestID]bool
    scenarios  map[ScenarioID]bool
    components map[ComponentID]bool
    services   map[ServiceID]bool
    traces     map[TraceID]*TraceContext
}

// Validation at registration time
func (r *IDRegistry) RegisterTest(id TestID) error {
    if r.tests[id] {
        return fmt.Errorf("duplicate test ID: %s", id)
    }
    r.tests[id] = true
    return nil
}
```

### TraceID & Distributed Tracing

Every test execution generates a `TraceID` that flows through all components and can be propagated to services under test:

```go
// TraceContext carries trace information through execution
type TraceContext struct {
    TraceID    TraceID           // Unique trace for this execution
    SpanID     string            // Current span within trace
    ParentSpan string            // Parent span (for nested components)
    Baggage    map[string]string // Key-value pairs to propagate
    StartTime  time.Time
}

// Context provides trace access
type Context interface {
    // ... other methods ...

    // Trace returns the current trace context
    Trace() *TraceContext

    // WithSpan creates a child span for nested operations
    WithSpan(name string) Context
}
```

### Header Injection

To propagate `TraceID` and other context to services under test, the framework provides header injection:

```go
// HeaderInjector configures how headers are injected into service calls
type HeaderInjector interface {
    // InjectHeaders adds trace/context headers to outgoing requests
    InjectHeaders(ctx Context, headers map[string]string) map[string]string
}

// Built-in injectors for common patterns
type W3CTraceInjector struct{}      // W3C Trace Context format
type B3Injector struct{}            // Zipkin B3 format
type JaegerInjector struct{}        // Jaeger format
type CustomInjector struct {        // User-defined format
    HeaderMapping map[string]string // TraceID -> X-My-Trace-ID
}
```

**Configuration:**

```yaml
tracing:
  enabled: true
  injector: w3c  # or: b3, jaeger, custom

  # For custom injector
  custom:
    headers:
      trace_id: X-Correlation-ID
      span_id: X-Request-ID
      baggage_prefix: X-Baggage-

  # Baggage to always propagate
  baggage:
    test_name: "{{scenario.name}}"
    environment: "{{env.name}}"
```

**Usage in Components:**

```go
// @witness:task name="CallAPI" requires="user:User"
func CallAPI(ctx witness.Context) (*Response, error) {
    // Headers are automatically injected when using the HTTP client
    client := witness.HTTPClient(ctx)  // Pre-configured with trace injection

    // Or manually inject headers
    headers := witness.InjectHeaders(ctx, map[string]string{
        "Content-Type": "application/json",
    })
    // headers now includes: X-Trace-ID, X-Span-ID, etc.

    resp, err := client.Post("/api/orders", body, headers)
    return resp, err
}
```

**Trace Propagation Flow:**

```
┌─────────────────────────────────────────────────────────────────┐
│  Test Execution (TraceID: abc-123)                              │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │ CreateUser   │───▶│ CallOrderAPI │───▶│ ValidateDB   │      │
│  │ span: span-1 │    │ span: span-2 │    │ span: span-3 │      │
│  └──────────────┘    └──────┬───────┘    └──────────────┘      │
│                             │                                   │
│                             │ HTTP Request                      │
│                             │ X-Trace-ID: abc-123               │
│                             │ X-Span-ID: span-2                 │
│                             ▼                                   │
│                    ┌────────────────┐                           │
│                    │ Order Service  │                           │
│                    │ (can correlate │                           │
│                    │  logs/traces)  │                           │
│                    └────────────────┘                           │
└─────────────────────────────────────────────────────────────────┘
```

---

## Type System

User-defined types flow through the system with compile-time safety.

### Annotated Types

Users annotate their domain types for framework discovery:

```go
// @witness:type
type User struct {
    ID    string
    Email string
    Role  string
}

// @witness:type
type Order struct {
    ID     string
    UserID string
    Items  []LineItem
    Total  float64
}

// @witness:type
type Cart struct {
    UserID string
    Items  []LineItem
}
```

### Type Aliases

For imported types from external packages:

```go
import "github.com/company/shared/models"

// @witness:type alias="PaymentResult"
type PaymentResult = models.PaymentResponse

// @witness:type alias="Customer"
type Customer = models.CustomerRecord
```

### Typed Context

Type-safe accessors for passing data between components:

```go
// Setting - type inferred
witness.Set(ctx, "user", &User{ID: "123", Email: "test@example.com"})

// Getting - compile-time type safety via generics
user := witness.Get[*User](ctx, "user")
cart := witness.Get[*Cart](ctx, "cart")
```

### Type Registry

Framework builds a registry of all discovered types:

```
Type Registry:
  User          → local (tests/types/user.go)
  Order         → local (tests/types/order.go)
  PaymentResult → alias of models.PaymentResponse
  Customer      → alias of models.CustomerRecord
```

---

## Middleware System

Middleware provides composable cross-cutting concerns without polluting component logic.

### Middleware Interface

```go
// Middleware wraps component execution
type Middleware func(next ComponentRunner) ComponentRunner

// ComponentRunner executes a component
type ComponentRunner func(ctx Context) error

// Chain applies middleware in order (first = outermost)
func Chain(middlewares ...Middleware) Middleware {
    return func(next ComponentRunner) ComponentRunner {
        for i := len(middlewares) - 1; i >= 0; i-- {
            next = middlewares[i](next)
        }
        return next
    }
}
```

### Built-in Middleware

**LoggingMiddleware** - Automatic execution logging:

```go
func LoggingMiddleware() Middleware {
    return func(next ComponentRunner) ComponentRunner {
        return func(ctx Context) error {
            ctx.Log(Info, "Starting %s", ctx.ComponentName())
            start := time.Now()

            err := next(ctx)

            duration := time.Since(start)
            if err != nil {
                ctx.Log(Error, "Failed %s after %v: %v", ctx.ComponentName(), duration, err)
            } else {
                ctx.Log(Info, "Completed %s in %v", ctx.ComponentName(), duration)
            }
            return err
        }
    }
}
```

**RetryMiddleware** - Automatic retry with backoff:

```go
func RetryMiddleware(maxRetries int, backoff BackoffStrategy) Middleware {
    return func(next ComponentRunner) ComponentRunner {
        return func(ctx Context) error {
            var lastErr error
            for attempt := 0; attempt <= maxRetries; attempt++ {
                if attempt > 0 {
                    delay := backoff.Delay(attempt)
                    ctx.Log(Debug, "Retry %d/%d after %v", attempt, maxRetries, delay)
                    time.Sleep(delay)
                }

                lastErr = next(ctx)
                if lastErr == nil {
                    return nil
                }
            }
            return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
        }
    }
}
```

**MetricsMiddleware** - Emit execution metrics:

```go
func MetricsMiddleware(collector MetricsCollector) Middleware {
    return func(next ComponentRunner) ComponentRunner {
        return func(ctx Context) error {
            start := time.Now()
            err := next(ctx)
            duration := time.Since(start)

            collector.RecordExecution(ctx.ComponentName(), duration, err == nil)
            return err
        }
    }
}
```

**TracingMiddleware** - Distributed tracing spans:

```go
func TracingMiddleware() Middleware {
    return func(next ComponentRunner) ComponentRunner {
        return func(ctx Context) error {
            spanCtx := ctx.WithSpan(ctx.ComponentName())
            defer spanCtx.Trace().End()

            return next(spanCtx)
        }
    }
}
```

**TimeoutMiddleware** - Enforce execution timeouts:

```go
func TimeoutMiddleware(timeout time.Duration) Middleware {
    return func(next ComponentRunner) ComponentRunner {
        return func(ctx Context) error {
            done := make(chan error, 1)
            go func() {
                done <- next(ctx)
            }()

            select {
            case err := <-done:
                return err
            case <-time.After(timeout):
                return fmt.Errorf("component timed out after %v", timeout)
            }
        }
    }
}
```

### Custom Middleware

Users can define custom middleware for domain-specific concerns:

```go
// @witness:middleware name="AuditMiddleware"
func AuditMiddleware(auditLog AuditLogger) Middleware {
    return func(next ComponentRunner) ComponentRunner {
        return func(ctx Context) error {
            auditLog.LogStart(ctx.Trace().TraceID, ctx.ComponentName())

            err := next(ctx)

            auditLog.LogEnd(ctx.Trace().TraceID, ctx.ComponentName(), err)
            return err
        }
    }
}
```

### Middleware Configuration

```yaml
middleware:
  # Global middleware (applies to all components)
  global:
    - logging
    - tracing
    - metrics

  # Per-scenario middleware
  scenarios:
    checkout-flow:
      - retry:
          max_attempts: 3
          backoff: exponential
      - timeout: 30s

  # Custom middleware registration
  custom:
    - name: audit
      path: ./middleware/audit.go
```

### Middleware Execution Order

```
Request Flow:
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  Logging ─▶ Tracing ─▶ Metrics ─▶ Retry ─▶ Timeout         │
│     │          │          │         │         │             │
│     │          │          │         │         ▼             │
│     │          │          │         │    ┌─────────┐        │
│     │          │          │         └───▶│Component│        │
│     │          │          │              └────┬────┘        │
│     │          │          │                   │             │
│     │          │          │◀──────────────────┘             │
│     │          │◀─────────┘        Response                 │
│     │◀─────────┘                                            │
│◀────┘                                                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Discovery

Zero-wiring philosophy: users write logic, framework discovers and wires.

### Annotation Format

```go
// @witness:<type> name="<name>" [key="value"]...

// Examples:
// @witness:type
// @witness:setup name="CreateUser" produces="user:User"
// @witness:task name="PlaceOrder" requires="user:User,cart:Cart" produces="order:Order"
// @witness:validation name="OrderValid" requires="order:Order"
// @witness:step name="CheckoutFlow" requires="user:User" produces="order:Order"
```

### Dependency Declaration

Components declare what they produce and require:

```go
// @witness:setup name="CreateUser" produces="user:User"
func CreateUser(ctx witness.Context) error {
    user := &User{ID: uuid.New(), Email: "test@example.com"}
    witness.Set(ctx, "user", user)
    return nil
}

// @witness:task name="CreateOrder" requires="user:User,cart:Cart" produces="order:Order"
func CreateOrder(ctx witness.Context) (*Order, error) {
    user := witness.Get[*User](ctx, "user")  // guaranteed to exist
    cart := witness.Get[*Cart](ctx, "cart")  // guaranteed to exist

    order := &Order{UserID: user.ID, Items: cart.Items}
    return order, nil
}

// @witness:validation name="OrderValid" requires="order:Order"
func ValidateOrder(ctx witness.Context, result any) error {
    order := result.(*Order)  // framework guarantees type
    if order.ID == "" {
        return errors.New("order ID is empty")
    }
    return nil
}
```

### Discovery Process

```
1. Scan configured paths for source files
2. Parse annotations from comments
3. Build component registry with:
   - Name
   - Type (setup/task/validation/step/rollup)
   - Produces (type:name pairs)
   - Requires (type:name pairs)
   - Source location
4. Build type registry from @witness:type annotations
5. Validate dependency graph (all requirements satisfiable)
6. Expose to UI/CLI for composition
```

### Dependency Graph

Framework builds and validates the dependency graph at discovery time:

```
Discovery Phase:
  CreateUser    → produces [user:User]
  SeedCart      → produces [cart:Cart]
  CreateOrder   → requires [user:User, cart:Cart] → produces [order:Order]
  ValidateOrder → requires [order:Order]

Composition Validation:
  If scenario has CreateOrder but no CreateUser upstream → error at config time
```

---

## Context & State

All state flows through the Context - components are stateless.

### Context Interface

```go
type Context interface {
    // Type-safe state access
    Get(key string) (any, bool)
    Set(key string, value any)

    // Typed accessors (via generics)
    // witness.Get[T](ctx, key) and witness.Set(ctx, key, value)

    // Infrastructure clients
    Client(name string) (any, error)

    // Flags
    Flag(name string) any

    // Parameters
    Param(name string) any

    // Logging
    Log(level LogLevel, msg string, args ...any)
}
```

### State Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ CreateUser  │────▶│ CreateOrder │────▶│ValidateOrder│
│             │     │             │     │             │
│ produces:   │     │ requires:   │     │ requires:   │
│   user:User │     │   user:User │     │   order     │
│             │     │   cart:Cart │     │             │
│             │     │ produces:   │     │             │
│             │     │   order     │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                    ┌──────▼──────┐
                    │   Context   │
                    │  (shared    │
                    │   state)    │
                    └─────────────┘
```

### Key Properties

- Components are **stateless** - all state flows through Context
- Components declare **dependencies** (what infra clients they need)
- Components can be **conditional** (run only if predicate passes)
- Components emit **events** for observability

---

## Next Steps

Continue to [Infrastructure](./02-infrastructure.md) for provider interfaces and environment overlays.
