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
- [Type System](#type-system)
  - [Annotated Types](#annotated-types)
  - [Type Aliases](#type-aliases)
  - [Typed Context](#typed-context)
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
