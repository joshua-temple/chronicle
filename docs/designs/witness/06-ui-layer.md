# UI Layer

> Web UI, TUI, IDE plugins, and WYSIWYG builder.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Results & Reporting](./05-results-and-reporting.md) | [Overview](./00-overview.md) | [Daemon Service](./07-daemon-service.md) |

---

## Table of Contents

- [Architecture](#architecture)
- [Web UI](#web-ui)
  - [WYSIWYG Builder](#wysiwyg-builder)
  - [Execution View](#execution-view)
  - [Results Dashboard](#results-dashboard)
- [Terminal UI (TUI)](#terminal-ui-tui)
- [IDE Plugins](#ide-plugins)
- [Dependency Graph](#dependency-graph)
- [YAML as Source of Truth](#yaml-as-source-of-truth)

---

## Architecture

All UIs are backed by the same API. No UI has special access.

```
┌─────────────────────────────────────────────────────────────┐
│                     API Server (Go)                          │
│  REST + gRPC + WebSocket (for real-time updates)            │
├─────────────────────────────────────────────────────────────┤
│  /scenarios    - CRUD scenarios                              │
│  /components   - List discovered components                  │
│  /types        - List discovered types                       │
│  /runs         - Trigger & monitor executions               │
│  /results      - Query historical results                    │
│  /config       - Infrastructure & environment config         │
│  /ws/events    - Real-time execution stream                  │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
    ┌─────────┐         ┌─────────┐         ┌─────────┐
    │ Web UI  │         │   TUI   │         │   IDE   │
    │ (React) │         │(Bubble  │         │(VSCode/ │
    │         │         │  Tea)   │         │ GoLand) │
    └─────────┘         └─────────┘         └─────────┘
```

---

## Web UI

Full-featured visual interface for basic users.

### Features

- Visual scenario builder (drag-and-drop)
- Real-time execution view with step-by-step progress
- Results dashboard with charts/trends
- Infrastructure health overview
- Configuration forms (no YAML knowledge required)
- Type browser with dependency visualization

### WYSIWYG Builder

Drag-and-drop scenario composition:

```
┌─ Scenario Builder ────────────────────────────────────────┐
│                                                           │
│  Available Components          Canvas                     │
│  ────────────────────          ──────                     │
│  SETUP                         ┌─────────────────────┐    │
│    ├─ CreateUser               │     CreateUser      │    │
│    ├─ SeedInventory            └──────────┬──────────┘    │
│    └─ MockPaymentGateway                  │              │
│                                           ▼              │
│  TASK                          ┌─────────────────────┐    │
│    ├─ SubmitOrder              │    AddToCart        │    │
│    ├─ AddToCart                └──────────┬──────────┘    │
│    └─ ProcessPayment                      │              │
│                                           ▼              │
│  VALIDATION                    ┌─────────────────────┐    │
│    ├─ OrderExists              │   SubmitOrder       │    │
│    └─ InventoryUpdated         └──────────┬──────────┘    │
│                                           │              │
│  STEP                                     ▼              │
│    └─ FullCheckout             ┌─────────────────────┐    │
│                                │   OrderExists       │    │
│  ROLLUP                        └─────────────────────┘    │
│    └─ E2EFlow                                            │
│                                                           │
│  [Save] [Run] [Add Chaos] [Export YAML]                  │
└───────────────────────────────────────────────────────────┘
```

### Smart Composition

UI knows component dependencies and guides users:

```
┌─ Drag "CreateOrder" onto canvas ─────────────────────────┐
│                                                          │
│  ⚠️  CreateOrder requires:                               │
│      • user:User  (missing - add CreateUser?)           │
│      • cart:Cart  (missing - add SeedCart?)             │
│                                                          │
│  [Auto-add dependencies]  [Add manually]  [Cancel]      │
└──────────────────────────────────────────────────────────┘
```

### Execution View

Real-time test execution monitoring:

```
┌─ Running: checkout-flow ──────────────────────────────────┐
│                                                           │
│  Progress: ████████████░░░░░░░░  60%                     │
│                                                           │
│  ✓ CreateUser          [passed]     120ms                │
│  ✓ SeedInventory       [passed]      45ms                │
│  ✓ AddItemsToCart      [passed]     230ms                │
│  ◐ SubmitOrder         [running]    1.2s...              │
│  ○ ProcessPayment      [pending]                         │
│  ○ OrderComplete       [pending]                         │
│                                                           │
│  Live Logs:                                              │
│  ──────────                                              │
│  [INFO] Creating order for user usr_123                  │
│  [INFO] Validating cart contents                         │
│  [DEBUG] Sending to order service...                     │
│                                                           │
│  [Cancel] [Pause] [View Details]                        │
└───────────────────────────────────────────────────────────┘
```

### Results Dashboard

Historical results with trends:

```
┌─ Results Dashboard ───────────────────────────────────────┐
│                                                           │
│  Last 7 Days                    Pass Rate: 94.2%         │
│  ┌──────────────────────────────────────────────────┐    │
│  │  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓│    │
│  │  Mon   Tue   Wed   Thu   Fri   Sat   Sun        │    │
│  └──────────────────────────────────────────────────┘    │
│                                                           │
│  Recent Failures:                                        │
│  ─────────────────                                       │
│  ✗ checkout-flow (3h ago) - timeout in ProcessPayment   │
│  ✗ inventory-sync (5h ago) - assertion failed           │
│                                                           │
│  [View All Results] [Export Report] [Configure Alerts]  │
└───────────────────────────────────────────────────────────┘
```

### Dependency Graph Visualization

```
┌─ Dependency Graph ────────────────────────────────────────┐
│                                                           │
│   CreateUser ──produces──▶ user:User                     │
│        │                      │                          │
│        │                      ▼                          │
│        │              CreateOrder ──produces──▶ order    │
│        │                      │                          │
│        ▼                      ▼                          │
│   SeedInventory        ValidateOrder                     │
│        │                                                 │
│        ▼                                                 │
│   AddToCart ──requires──▶ user:User, inventory           │
│                                                           │
│  [Highlight Critical Path] [Show Parallelizable]         │
└───────────────────────────────────────────────────────────┘
```

### Type Browser

```
┌─ Types ──────────────────────────────────────────────────┐
│                                                          │
│  Name           Source                  Used By          │
│  ─────────────  ──────────────────────  ───────────────  │
│  User           tests/types/user.go     CreateUser,      │
│                                         CreateOrder      │
│  Order          tests/types/order.go    CreateOrder,     │
│                                         ValidateOrder    │
│  PaymentResult  alias: models.Payment   ProcessPayment,  │
│                                         ValidatePayment  │
│                                                          │
│  [View Schema] [Find Usages]                            │
└──────────────────────────────────────────────────────────┘
```

### Visual Debugger

Step-through debugging:

```
┌─ Debugger ────────────────────────────────────────────────┐
│                                                           │
│  ● CreateUser     ← [Breakpoint]                         │
│  ○ AddToCart                                              │
│  ○ SubmitOrder                                            │
│                                                           │
│  Context State:                  │  Component Output:     │
│  ─────────────                   │  ────────────────      │
│  user: {                         │  {                     │
│    id: "usr_123",                │    id: "usr_123",      │
│    email: "test@..."             │    email: "test@..."   │
│  }                               │  }                     │
│                                                           │
│  [Step Over] [Step Into] [Continue] [Inspect]            │
└───────────────────────────────────────────────────────────┘
```

---

## Terminal UI (TUI)

Keyboard-driven interface for advanced users.

### Main View

```
┌─ Witness ─────────────────────────────────────────────────┐
│ Scenarios          │ Execution                            │
│ ──────────         │ ──────────                           │
│ ▸ checkout-flow    │ ● CreateUser        [passed]  120ms │
│   payment-flow     │ ● SeedInventory     [passed]   45ms │
│   auth-tests       │ ◐ AddItemsToCart    [running]       │
│   inventory-sync   │ ○ SubmitOrder       [pending]       │
│                    │ ○ ValidateOrder     [pending]       │
├────────────────────┼──────────────────────────────────────┤
│ [r]un [e]dit [c]haos [q]uit │ Logs ▼                     │
│                              │ INFO: Adding item SKU-123  │
└──────────────────────────────┴────────────────────────────┘
```

### Features

- Keyboard navigation (vim-style bindings)
- Live log tailing
- Quick filters (`/` to search)
- Split panes for scenarios and execution
- Works over SSH
- Low bandwidth friendly

### Key Bindings

| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `Enter` | Select/expand |
| `r` | Run selected scenario |
| `e` | Edit scenario (opens $EDITOR) |
| `c` | Toggle chaos mode |
| `l` | Toggle log panel |
| `/` | Search/filter |
| `q` | Quit |
| `?` | Help |

---

## IDE Plugins

Deep integration for professionals.

### VSCode Extension

- Inline annotations showing discovered components
- Click-to-run individual scenarios
- Test explorer integration
- Inline results/failure details
- YAML schema validation + autocomplete
- Debugger integration

### GoLand Plugin

- Gutter icons for runnable components
- Run configurations
- Test results in tool window
- YAML support with completion
- Navigate to component definition

### Features

```go
// Gutter icon: ▶ Run | 🐛 Debug
// @witness:task name="CreateOrder" requires="user:User"
func CreateOrder(ctx witness.Context) (*Order, error) {
    // Hover shows: Last run: passed (2.3s) - 3 hours ago
    // ...
}
```

---

## Dependency Graph

### CLI Visualization

```bash
# ASCII graph of scenario dependencies
witness graph --scenario checkout-flow

# Output:
# checkout-flow
# ├── CreateUser [setup]
# │   └── produces: user:User
# ├── SeedCart [setup]
# │   ├── requires: user:User ←──┘
# │   └── produces: cart:Cart
# ├── Checkout [task]
# │   ├── requires: user:User ←──────┐
# │   ├── requires: cart:Cart ←──────┤
# │   └── produces: order:Order      │
# └── OrderCreated [validation]      │
#     └── requires: order:Order ←────┘

# Export as DOT format for Graphviz
witness graph --scenario checkout-flow --format dot > graph.dot
dot -Tpng graph.dot -o graph.png

# Export as Mermaid
witness graph --scenario checkout-flow --format mermaid
```

### Cycle Detection

```bash
witness validate --check-cycles

# Output:
# ❌ Cycle detected in component dependencies:
#    CreateOrder → ValidateStock → ReserveInventory → CreateOrder
#
#    Suggestion: Extract shared dependency or use explicit ordering
```

### Validation Rules

At discovery time:

```yaml
validation:
  dependencies:
    # Fail if any component has unresolved requires
    require_all_dependencies: true

    # Fail if cycles detected
    disallow_cycles: true

    # Warn if produces overlap (same key from multiple components)
    warn_on_shadowing: true
```

### Graph Query

```bash
# What depends on user:User?
witness graph --depends-on "user:User"

# What does CreateOrder need?
witness graph --component CreateOrder --show requires

# Find shortest path between components
witness graph --from CreateUser --to ValidateOrder
```

---

## YAML as Source of Truth

UI reads from and writes to YAML. No separate storage.

### Bidirectional Sync

```
┌─────────────────────────────────────────────────────────┐
│                      YAML Files                          │
│              (source of truth, git-versioned)            │
└──────────────────────┬──────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
         ▼             ▼             ▼
    ┌────────┐    ┌────────┐    ┌────────┐
    │  CLI   │    │  TUI   │    │ Web UI │
    └────────┘    └────────┘    └────────┘
         │             │             │
         └─────────────┼─────────────┘
                       │
                       ▼
              Read YAML → Display
              Edit in UI → Write YAML
```

### Export from Deployed Instance

```bash
# Export all configs from running instance
witness export --output ./configs/

# Pull specific scenarios
witness export --scenarios checkout-flow,payment-flow --output ./configs/

# Sync to local for git commit
witness export | git add -A && git commit -m "sync test configs"
```

---

## Next Steps

Continue to [Daemon Service](./07-daemon-service.md) for service mode, API, and scheduling.
