# Multi-Language

> Go, Python, Java SDKs and portability strategy.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Daemon Service](./07-daemon-service.md) | [Overview](./00-overview.md) | [Extensibility](./09-extensibility.md) |

---

## Table of Contents

- [Portability Strategy](#portability-strategy)
- [Go SDK](#go-sdk)
- [Python SDK](#python-sdk)
- [Java SDK](#java-sdk)
- [Shared Components](#shared-components)
- [Polyglot Execution](#polyglot-execution)

---

## Portability Strategy

Go is the proof-of-concept. The architecture enables ports to other languages.

```
┌─────────────────────────────────────────────────────────────┐
│                    Language-Agnostic Core                    │
│  (Concepts, YAML schema, API contracts, protocols)          │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
   ┌─────────┐           ┌─────────┐          ┌─────────┐
   │   Go    │           │ Python  │          │  Java   │
   │   SDK   │           │   SDK   │          │   SDK   │
   └─────────┘           └─────────┘          └─────────┘
```

### What's Portable (Language-Agnostic)

- YAML configuration schema
- API contracts (OpenAPI/gRPC definitions)
- Concepts: Setup/Task/Validation/Step/Rollup/Scenario
- Type annotation patterns
- Chaos profiles
- Results format

### What's Language-Specific

- Annotation/decorator syntax
- Discovery mechanism
- Type system integration
- Client libraries

---

## Go SDK

The reference implementation.

### Annotations

```go
// @witness:type
type User struct {
    ID    string
    Email string
}

// @witness:setup name="CreateUser" produces="user:User"
func CreateUser(ctx witness.Context) error {
    user := &User{ID: uuid.New(), Email: "test@example.com"}
    witness.Set(ctx, "user", user)
    return nil
}

// @witness:task name="CreateOrder" requires="user:User,cart:Cart" produces="order:Order"
func CreateOrder(ctx witness.Context) (*Order, error) {
    user := witness.Get[*User](ctx, "user")
    cart := witness.Get[*Cart](ctx, "cart")
    return &Order{UserID: user.ID, Items: cart.Items}, nil
}

// @witness:validation name="OrderValid" requires="order:Order"
func ValidateOrder(ctx witness.Context, result any) error {
    order := result.(*Order)
    if order.ID == "" {
        return errors.New("order ID should not be empty")
    }
    return nil
}
```

### Discovery

- AST parsing of Go source files
- Comment annotation extraction
- Reflection for type information
- Build-time code generation (optional)

### Type Safety

```go
// Generics for type-safe context access
user := witness.Get[*User](ctx, "user")
cart := witness.Get[*Cart](ctx, "cart")

// Compile-time type checking
witness.Set(ctx, "order", order)
```

---

## Python SDK

Pythonic interface using decorators and type hints.

### Decorators

```python
from witness import witness, Context

@witness.type
class User:
    id: str
    email: str

@witness.type
class Order:
    id: str
    user_id: str
    items: list
    total: float

@witness.setup(name="CreateUser", produces=["user:User"])
def create_user(ctx: Context) -> None:
    user = User(id=str(uuid.uuid4()), email="test@example.com")
    ctx.set("user", user)

@witness.task(name="CreateOrder", requires=["user:User", "cart:Cart"], produces=["order:Order"])
def create_order(ctx: Context) -> Order:
    user = ctx.get(User, "user")
    cart = ctx.get(Cart, "cart")
    return Order(user_id=user.id, items=cart.items)

@witness.validation(name="OrderValid", requires=["order:Order"])
def validate_order(ctx: Context, result: Any) -> None:
    order = result
    assert order.id, "order ID should not be empty"
```

### Discovery

- Decorator registration
- Python `inspect` module
- Type hint introspection
- Module scanning

### Type Safety

```python
# Type hints + runtime validation
user: User = ctx.get(User, "user")
cart: Cart = ctx.get(Cart, "cart")

# Optional runtime type checking
ctx.set("order", order, validate=True)
```

---

## Java SDK

Annotation-based approach using standard Java patterns.

### Annotations

```java
package com.example.tests;

import io.witness.*;

@WitnessType
public class User {
    private String id;
    private String email;
    // getters, setters, etc.
}

@WitnessType
public class Order {
    private String id;
    private String userId;
    private List<Item> items;
    private double total;
}

public class UserSetup {

    @WitnessSetup(name = "CreateUser", produces = "user:User")
    public void createUser(WitnessContext ctx) {
        User user = new User(UUID.randomUUID().toString(), "test@example.com");
        ctx.set("user", user);
    }
}

public class OrderTasks {

    @WitnessTask(name = "CreateOrder", requires = {"user:User", "cart:Cart"}, produces = "order:Order")
    public Order createOrder(WitnessContext ctx) {
        User user = ctx.get(User.class, "user");
        Cart cart = ctx.get(Cart.class, "cart");
        return new Order(user.getId(), cart.getItems());
    }
}

public class OrderValidations {

    @WitnessValidation(name = "OrderValid", requires = "order:Order")
    public void validateOrder(WitnessContext ctx, Object result) {
        Order order = (Order) result;
        if (order.getId() == null || order.getId().isEmpty()) {
            throw new AssertionError("order ID should not be empty");
        }
    }
}
```

### Discovery

- Annotation processing (compile-time)
- Classpath scanning (runtime)
- Reflection for type information

### Type Safety

```java
// Generic context methods
User user = ctx.get(User.class, "user");
Cart cart = ctx.get(Cart.class, "cart");

// Type-safe set
ctx.set("order", order);
```

---

## Shared Components

### YAML Schema (Same Across Languages)

```yaml
# This YAML works with Go, Python, and Java implementations
scenarios:
  - name: checkout-flow
    flow:
      - setup: CreateUser
      - setup: SeedCart
      - task: CreateOrder
      - validation: OrderValid

infrastructure:
  postgres:
    provider: postgres
    config:
      mode: container
      image: postgres:15

chaos_profiles:
  degraded-network:
    infrastructure:
      - type: network_latency
        latency_ms: 200
```

### API Contracts

```protobuf
// witness.proto - shared gRPC definitions
service WitnessService {
  rpc TriggerRun(TriggerRunRequest) returns (RunResponse);
  rpc GetRunStatus(GetRunStatusRequest) returns (RunStatus);
  rpc ListScenarios(ListScenariosRequest) returns (ScenariosResponse);
}

message TriggerRunRequest {
  repeated string scenarios = 1;
  string environment = 2;
  map<string, string> flags = 3;
  repeated string options = 4;
}
```

### Results Format

```json
{
  "id": "run_abc123",
  "scenario": "checkout-flow",
  "status": "passed",
  "duration_ms": 2340,
  "steps": [
    {
      "name": "CreateUser",
      "type": "setup",
      "status": "passed",
      "duration_ms": 120
    }
  ]
}
```

---

## Polyglot Execution

Mix languages in the same test suite.

### Architecture

```
┌──────────────────┐     ┌──────────────────┐
│  Witness Service │     │   Worker Pool    │
│   (Orchestrator) │────▶│  ┌────────────┐  │
│                  │     │  │ Go Worker  │  │
│  - Language      │     │  ├────────────┤  │
│    agnostic      │     │  │ Python     │  │
│  - Coordinates   │     │  │ Worker     │  │
│    via API       │     │  ├────────────┤  │
│                  │     │  │ Java       │  │
│                  │     │  │ Worker     │  │
└──────────────────┘     │  └────────────┘  │
                         └──────────────────┘
```

### Configuration

```yaml
execution:
  workers:
    - language: go
      path: ./tests/go
      count: 2

    - language: python
      path: ./tests/python
      venv: ./tests/python/.venv
      count: 2

    - language: java
      path: ./tests/java
      classpath: ./tests/java/target/classes
      count: 1
```

### Scenario Assignment

```yaml
scenarios:
  # Runs on Go worker
  - name: checkout-flow
    language: go
    flow: [...]

  # Runs on Python worker
  - name: ml-validation
    language: python
    flow: [...]

  # Auto-detect based on component location
  - name: mixed-flow
    flow:
      - setup: CreateUser          # Go
      - task: RunMLPrediction      # Python
      - validation: ValidateOutput # Go
```

### Cross-Language Data

Data flows through the coordinator:

```
Go Worker                 Coordinator              Python Worker
    │                          │                          │
    │  produces user:User      │                          │
    │─────────────────────────▶│                          │
    │                          │  user:User available     │
    │                          │─────────────────────────▶│
    │                          │                          │
    │                          │  requires user:User      │
    │                          │◀─────────────────────────│
    │                          │                          │
```

Types are serialized (JSON/protobuf) for cross-language transfer.

---

## Next Steps

Continue to [Extensibility](./09-extensibility.md) for plugin system and extension points.
