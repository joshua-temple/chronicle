# Components

Components are the building blocks of Chronicle tests. Each component is a Go function with annotations that describe its role, dependencies, and outputs.

## Component Types

Chronicle supports six component types, following a Given-When-Then pattern:

| Type | Purpose | Annotation |
|------|---------|------------|
| **Setup** | Prepare state before actions (Given) | `@chronicle:setup` |
| **Task** | Execute actions (When) | `@chronicle:task` |
| **Validation** | Assert outcomes (Then) | `@chronicle:validation` |
| **Step** | Bundle of 2+ Setup/Task/Validation | `@chronicle:step` |
| **Rollup** | Higher-order composition of Steps | `@chronicle:rollup` |
| **Teardown** | Cleanup (runs even on failure) | `@chronicle:teardown` |

## Annotations

Annotate your functions to make them discoverable:

```go
// @chronicle:setup name="CreateUser" produces="user:User" teardown="DeleteUser"
// @chronicle:description "Creates a test user"
// @chronicle:tags setup,user,auth
func CreateUser(ctx context.Context) error {
    // ...
}
```

### Available Annotations

| Annotation | Description | Example |
|------------|-------------|---------|
| `@chronicle:<type>` | Component type (required) | `@chronicle:setup` |
| `name="..."` | Component name | `name="CreateUser"` |
| `produces="key:Type"` | What this component outputs | `produces="user:User"` |
| `requires="key:Type"` | What this component needs | `requires="user:User"` |
| `teardown="Name"` | Paired teardown component | `teardown="DeleteUser"` |
| `@chronicle:description` | Human-readable description | `"Creates a test user"` |
| `@chronicle:tags` | Comma-separated tags | `setup,user,auth` |
| `@chronicle:owner` | Team or person responsible | `platform-team` |
| `@chronicle:version` | Component version | `1.0.0` |
| `@chronicle:deprecated` | Mark as deprecated | `"Use CreateUserV2 instead"` |

### Multiple Dependencies

Specify multiple produces/requires with comma separation:

```go
// @chronicle:task name="ProcessOrder" requires="user:User,cart:Cart" produces="order:Order,receipt:Receipt"
func ProcessOrder(ctx context.Context) (*Order, error) {
    // ...
}
```

## Function Signatures

Each component type has an expected function signature:

### Setup

```go
// SetupFunc prepares state and returns only an error
func(ctx context.Context) error
```

### Task

```go
// TaskFunc executes an action and returns a result
func(ctx context.Context) (T, error)  // Generic result type
```

### Validation

```go
// ValidationFunc receives the upstream result and validates it
func(ctx context.Context, result any) error
```

### Teardown

```go
// TeardownFunc cleans up and returns only an error
func(ctx context.Context) error
```

### Step / Rollup

```go
// Composite components return only an error
func(ctx context.Context) error
```

## Context

The context provides type-safe data sharing between components.

### Setting Values

```go
import "github.com/joshua-temple/chronicle/pkg/context"

func CreateUser(ctx context.Context) error {
    user := &User{ID: "usr_123"}
    context.Set(ctx, "user", user)  // Store with key "user"
    return nil
}
```

### Getting Values

```go
func CreateOrder(ctx context.Context) (*Order, error) {
    // Type-safe retrieval with generics
    user := context.Get[*User](ctx, "user")
    if user == nil {
        return nil, errors.New("user not found")
    }
    // ...
}
```

### Context Keys

Keys should match your `produces`/`requires` declarations:

```go
// @chronicle:setup produces="user:User,config:Config"
func Setup(ctx context.Context) error {
    context.Set(ctx, "user", user)     // key: "user"
    context.Set(ctx, "config", cfg)    // key: "config"
    return nil
}
```

## Dependencies

Chronicle automatically resolves dependencies based on produces/requires:

```
CreateUser (produces: user)
    └── CreateOrder (requires: user, produces: order)
            └── VerifyOrder (requires: order)
```

### Dependency Validation

The `chronicle validate` command checks:

- All required dependencies can be satisfied
- No circular dependencies exist
- Types match between producers and consumers

### Viewing Dependencies

```bash
# Show full dependency graph
chronicle graph

# Show what a component requires
chronicle graph --component CreateOrder --show-requires

# Show what depends on a component
chronicle graph --component CreateUser --reverse
```

## Lifecycle

### Execution Order

1. **Setup** components run first, in dependency order
2. **Task** components run after their dependencies are satisfied
3. **Validation** components run after the task they validate
4. **Teardown** components run last, even if tests fail

### Paired Teardowns

Setup components can declare a paired teardown:

```go
// @chronicle:setup name="CreateUser" teardown="DeleteUser"
func CreateUser(ctx context.Context) error { ... }

// @chronicle:teardown name="DeleteUser"
func DeleteUser(ctx context.Context) error { ... }
```

The teardown runs automatically when the scenario ends.

### Teardown Order

Teardowns run in reverse order of their paired setups:

```
Setup: A → B → C
Teardown: C → B → A
```

## Programmatic Components

Create components programmatically without annotations:

```go
import "github.com/joshua-temple/chronicle/pkg/core"

comp := core.NewComponent("CreateUser", core.ComponentSetup).
    WithProduces("user", "User").
    WithTeardown("DeleteUser").
    WithDescription("Creates a test user").
    WithTags("user", "auth").
    WithFunc(func(ctx context.Context) error {
        // ...
        return nil
    })
```

### Component Builder Methods

| Method | Description |
|--------|-------------|
| `WithProduces(key, type)` | Add output dependency |
| `WithRequires(key, type)` | Add input dependency |
| `WithTeardown(name)` | Set paired teardown |
| `WithDescription(desc)` | Set description |
| `WithTags(tags...)` | Add tags |
| `WithOwner(owner)` | Set owner |
| `WithVersion(version)` | Set version |
| `WithDeprecated(msg, sunset)` | Mark deprecated |
| `WithFunc(fn)` | Bind the function |

## Best Practices

1. **Single Responsibility** - Each component should do one thing
2. **Clear Dependencies** - Explicitly declare all requires/produces
3. **Pair Teardowns** - Always clean up resources you create
4. **Use Tags** - Tag components for filtering (`smoke`, `integration`, etc.)
5. **Document** - Use `@chronicle:description` for clarity
6. **Type Safety** - Use generics with `context.Get[T]` for type-safe retrieval
