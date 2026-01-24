# Witness Implementation Plan

> Autonomous implementation guide for replacing Chronicle with Witness.
> Execute with: `/super` pointing to this document with `--dangerously-skip-permissions`

---

## Overview

**Objective:** Implement the complete Witness testing framework as designed in `docs/designs/witness/`, replacing the existing Chronicle codebase.

**Branch:** `feature/witness-implementation`

**Approach:**
- Small, conventional commits (`feat:`, `test:`, `refactor:`, `docs:`)
- Test-first development where practical
- Example-driven validation (examples/ folder demonstrates all features)
- Use `ralph` loops for iterative refinement within phases

**Success Criteria:**
- All design features implemented
- Comprehensive test coverage (target: 80%+)
- All examples pass
- `go test ./...` passes
- `golangci-lint run` passes
- Design docs remain accurate (update if implementation diverges)

---

## Context Control

> **Critical:** This implementation spans multiple context windows. Follow these rules to prevent context rot.

### Progress Tracking

Maintain `docs/designs/witness/PROGRESS.md` as the single source of truth for implementation state:

```markdown
# Witness Implementation Progress

## Current Phase: [X]
## Current Step: [X.Y]
## Status: [in_progress | blocked | completed]

## Completed Phases
- [x] Phase 0: Setup (commit: abc1234)
- [ ] Phase 1: Core Framework
  - [x] 1.1 Typed Identifiers (commit: def5678)
  - [ ] 1.2 Component Types
  ...

## Blocking Issues
- None

## Next Action
[Specific next step to take]
```

### Phase Isolation Rules

1. **One phase per session** - Complete one phase before starting another
2. **Commit after each step** - Every numbered step (1.1, 1.2, etc.) gets its own commit
3. **Update PROGRESS.md after every commit** - This is the context restoration point
4. **Each phase is self-contained** - Reference design docs, not previous conversation

### Context Restoration Protocol

When starting a new session or after context compaction:

```
1. Read PROGRESS.md to understand current state
2. Read the design doc for current phase (e.g., 01-core-framework.md)
3. Read relevant existing code files for current step
4. Continue from "Next Action" in PROGRESS.md
```

### Checkpoint Commands

After completing each step, run:

```bash
# 1. Verify tests pass
go test ./pkg/witness/... -v

# 2. Verify lint passes
golangci-lint run ./pkg/witness/...

# 3. Commit the step
git add -A && git commit -m "feat(scope): description"

# 4. Update PROGRESS.md
# Mark step complete, update "Next Action"
```

### State Preservation Files

These files persist knowledge across context windows:

| File | Purpose |
|------|---------|
| `PROGRESS.md` | Current implementation state |
| `go.mod` | Dependencies and module structure |
| `*_test.go` | Expected behavior documentation |
| Design docs | Authoritative specifications |

### Anti-Patterns to Avoid

- **Don't** reference "earlier in this conversation" - use files instead
- **Don't** assume knowledge from previous sessions - re-read relevant files
- **Don't** batch multiple steps into one commit - granular commits aid restoration
- **Don't** skip PROGRESS.md updates - this causes context rot
- **Don't** implement ahead of the current phase - stay focused

---

## Pre-Implementation Setup

### Step 0.1: Create Feature Branch

```bash
git checkout -b feature/witness-implementation
```

### Step 0.2: Clean Slate

Remove existing Chronicle implementation but preserve:
- `go.mod` / `go.sum` (will update dependencies)
- `.github/` workflows
- `docs/designs/witness/` (the design docs)
- `CLAUDE.md`

```bash
# Remove old implementation
rm -rf pkg/ internal/ cmd/ examples/ testutil/

# Create new structure
mkdir -p \
  cmd/witness \
  pkg/witness/core \
  pkg/witness/context \
  pkg/witness/discovery \
  pkg/witness/infrastructure \
  pkg/witness/scenario \
  pkg/witness/execution \
  pkg/witness/results \
  pkg/witness/middleware \
  pkg/witness/chaos \
  pkg/witness/mocks \
  pkg/witness/config \
  pkg/witness/cli \
  examples/basic \
  examples/infrastructure \
  examples/chaos \
  examples/mocks \
  examples/distributed \
  examples/full-stack
```

Commit: `feat: initialize witness project structure`

---

## Phase 1: Core Framework

**Reference:** `docs/designs/witness/01-core-framework.md`

### 1.1 Typed Identifiers

**File:** `pkg/witness/core/identifiers.go`

Implement:
```go
type TestID string
type ScenarioID string
type ComponentID string
type ServiceID string
type TraceID string
type RunID string
type SpanID string
```

Include:
- `IDRegistry` for tracking active IDs
- `NewTestID()`, `NewTraceID()` generators
- Validation methods

**Test File:** `pkg/witness/core/identifiers_test.go`
- Test ID generation uniqueness
- Test registry duplicate detection
- Test validation

Commit: `feat(core): add typed identifiers and ID registry`

### 1.2 Component Types

**File:** `pkg/witness/core/components.go`

Implement:
```go
type ComponentType string

const (
    ComponentSetup      ComponentType = "setup"
    ComponentTask       ComponentType = "task"
    ComponentValidation ComponentType = "validation"
    ComponentStep       ComponentType = "step"
    ComponentRollup     ComponentType = "rollup"
    ComponentTeardown   ComponentType = "teardown"
)

type Component struct {
    ID          ComponentID
    Name        string
    Type        ComponentType
    Version     string
    Produces    []Dependency
    Requires    []Dependency
    Teardown    string // paired teardown component
    Description string
    Tags        []string
    Owner       string
    Deprecated  string
    Sunset      string
}

type Dependency struct {
    Key  string
    Type string
    Description string
}
```

**Test File:** `pkg/witness/core/components_test.go`

Commit: `feat(core): add component type definitions`

### 1.3 Context Implementation

**File:** `pkg/witness/context/context.go`

Implement:
```go
type Context interface {
    context.Context

    // State management
    Get(key string) (any, bool)
    Set(key string, value any)

    // Infrastructure
    Client(name string) (any, error)

    // Flags & Params
    Flag(name string) any
    Param(name string) any

    // Tracing
    Trace() *TraceContext
    WithSpan(name string) Context

    // Child contexts
    Child(name string) Context

    // Logging
    Log(level LogLevel, msg string, args ...any)
    Narrate(level NarrativeLevel, action string, details map[string]any)

    // Teardown context
    FailureReason() error
    PartialResults() map[string]any
}

// Package-level generic accessors
func Get[T any](ctx Context, key string) T
func Set[T any](ctx Context, key string, value T)
```

Include:
- Thread-safety documentation (not thread-safe, use Child())
- Size limits enforcement
- Trace context propagation

**Test File:** `pkg/witness/context/context_test.go`
- Test Get/Set with various types
- Test generic accessors
- Test child context isolation
- Test size limits

Commit: `feat(context): implement typed context with generics`

### 1.4 Annotation Discovery

**File:** `pkg/witness/discovery/parser.go`

Implement AST-based annotation parser:
```go
type AnnotationParser struct {
    paths []string
}

func (p *AnnotationParser) Discover() (*Registry, error)

// Parses:
// @witness:type
// @witness:setup name="X" produces="key:Type" teardown="Y"
// @witness:task name="X" requires="key:Type" produces="key:Type"
// @witness:validation name="X" requires="key:Type"
// @witness:teardown name="X" requires="key:Type"
// @witness:step name="X"
// @witness:rollup name="X"
// @witness:middleware name="X"
// @witness:description "text"
// @witness:tags tag1,tag2
// @witness:owner team-name
// @witness:version "1"
// @witness:deprecated "message" sunset="date"
```

**File:** `pkg/witness/discovery/registry.go`

```go
type Registry struct {
    Components map[ComponentID]*Component
    Types      map[string]*TypeInfo
    Middleware map[string]*MiddlewareInfo
}

func (r *Registry) Validate() error // Check all requires can be satisfied
func (r *Registry) DetectCycles() []Cycle
func (r *Registry) DependencyGraph() *Graph
```

**Test File:** `pkg/witness/discovery/parser_test.go`
- Test parsing each annotation type
- Test multi-line descriptions
- Test error cases (malformed annotations)

**Test File:** `pkg/witness/discovery/registry_test.go`
- Test validation
- Test cycle detection
- Test dependency graph building

Commit: `feat(discovery): implement AST-based annotation parser`
Commit: `feat(discovery): add component registry with validation`

### 1.5 Middleware System

**File:** `pkg/witness/middleware/middleware.go`

```go
type Middleware func(next Runner) Runner
type Runner func(ctx Context) error

func Chain(middlewares ...Middleware) Middleware

// Built-in middleware
func Logging() Middleware
func Tracing() Middleware
func Metrics(collector MetricsCollector) Middleware
func Retry(config RetryConfig) Middleware
func Timeout(d time.Duration) Middleware
```

**Test File:** `pkg/witness/middleware/middleware_test.go`
- Test chain ordering
- Test each built-in middleware
- Test error propagation

Commit: `feat(middleware): implement composable middleware system`

### Example: Basic Component Discovery

**File:** `examples/basic/types.go`
```go
package basic

// @witness:type
type User struct {
    ID    string
    Email string
}

// @witness:type
type Order struct {
    ID     string
    UserID string
    Total  float64
}
```

**File:** `examples/basic/components.go`
```go
package basic

import "github.com/joshua-temple/witness/pkg/witness"

// @witness:setup name="CreateUser" produces="user:User" teardown="DeleteUser"
// @witness:description "Creates a test user for the scenario"
// @witness:tags setup,user
func CreateUser(ctx witness.Context) error {
    user := &User{ID: "usr_123", Email: "test@example.com"}
    witness.Set(ctx, "user", user)
    return nil
}

// @witness:teardown name="DeleteUser" requires="user:User"
func DeleteUser(ctx witness.Context) error {
    user := witness.Get[*User](ctx, "user")
    // cleanup logic
    _ = user
    return nil
}

// @witness:task name="CreateOrder" requires="user:User" produces="order:Order"
// @witness:description "Creates an order for the user"
func CreateOrder(ctx witness.Context) (*Order, error) {
    user := witness.Get[*User](ctx, "user")
    order := &Order{ID: "ord_456", UserID: user.ID, Total: 99.99}
    return order, nil
}

// @witness:validation name="OrderValid" requires="order:Order"
// @witness:description "Validates the order was created correctly"
func OrderValid(ctx witness.Context, result any) error {
    order := result.(*Order)
    if order.ID == "" {
        return errors.New("order ID should not be empty")
    }
    if order.Total <= 0 {
        return errors.New("order total should be positive")
    }
    return nil
}
```

**File:** `examples/basic/basic_test.go`
```go
package basic_test

func TestDiscovery(t *testing.T) {
    parser := discovery.NewParser("./")
    registry, err := parser.Discover()
    require.NoError(t, err)

    assert.Contains(t, registry.Components, "CreateUser")
    assert.Contains(t, registry.Components, "CreateOrder")
    assert.Contains(t, registry.Components, "OrderValid")
    assert.Contains(t, registry.Types, "User")
    assert.Contains(t, registry.Types, "Order")
}
```

Commit: `feat(examples): add basic component discovery example`

### Phase 1 Completion Checkpoint

Before proceeding to Phase 2:

```bash
# Verify all Phase 1 tests pass
go test ./pkg/witness/core/... ./pkg/witness/context/... ./pkg/witness/discovery/... ./pkg/witness/middleware/... ./examples/basic/... -v

# Verify lint passes
golangci-lint run ./pkg/witness/...

# Update PROGRESS.md
# - Mark all Phase 1 steps complete
# - Set Current Phase: 2
# - Set Next Action: "Start Phase 2: YAML Configuration"
```

**Files that should exist:**
- `pkg/witness/core/identifiers.go` + `_test.go`
- `pkg/witness/core/components.go` + `_test.go`
- `pkg/witness/context/context.go` + `_test.go`
- `pkg/witness/discovery/parser.go` + `_test.go`
- `pkg/witness/discovery/registry.go` + `_test.go`
- `pkg/witness/middleware/middleware.go` + `_test.go`
- `examples/basic/types.go`, `components.go`, `basic_test.go`

---

## Phase 2: Configuration & Scenarios

**Reference:** `docs/designs/witness/03-scenarios-and-composition.md`

### 2.1 YAML Configuration

**File:** `pkg/witness/config/loader.go`

```go
type Config struct {
    Name           string
    Version        string
    Discovery      DiscoveryConfig
    Infrastructure map[string]InfraConfig
    Scenarios      []ScenarioConfig
    ChaosProfiles  map[string]ChaosProfile
    MockProfiles   map[string]MockProfile
    Flags          FlagsConfig
    Options        map[string]OptionConfig
    Bundles        BundlesConfig
    Secrets        SecretsConfig
    Execution      ExecutionConfig
    Results        ResultsConfig
    Notifications  NotificationsConfig
}

func Load(paths ...string) (*Config, error)
func LoadWithOverlay(base string, overlay string) (*Config, error)
```

**File:** `pkg/witness/config/schema.go`
- Define all config structs matching design YAML schemas

**File:** `pkg/witness/config/validation.go`
- Validate configuration
- Check required fields
- Validate references (scenario references existing components)

**Test File:** `pkg/witness/config/loader_test.go`
- Test loading valid configs
- Test overlay merging
- Test validation errors

Commit: `feat(config): implement YAML configuration loader`
Commit: `feat(config): add environment overlay support`

### 2.2 Scenario Model

**File:** `pkg/witness/scenario/scenario.go`

```go
type Scenario struct {
    Name        string
    Description string
    Timeout     time.Duration
    Tags        []string

    // Flow
    Flow        []FlowItem
    TeardownFlow []FlowItem

    // Modifiers
    Flags       map[string]any
    Options     []string
    ChaosProfiles []string
    MockProfiles  []string

    // Conditions
    SkipIf      []Condition
    SkipUnless  []Condition

    // Matrix
    Matrix      map[string][]any

    // Inheritance
    Extends     string
    Abstract    bool
}

type FlowItem struct {
    Type       ComponentType
    Name       string
    Timeout    time.Duration
    DependsOn  []string
    Params     map[string]any
    Parallel   []FlowItem // for parallel blocks
}
```

**File:** `pkg/witness/scenario/builder.go`
- Fluent builder API for programmatic scenario creation

**File:** `pkg/witness/scenario/resolver.go`
- Resolve scenario from config + registry
- Handle inheritance
- Expand matrix parameters
- Apply options/flags/chaos

**Test File:** `pkg/witness/scenario/scenario_test.go`

Commit: `feat(scenario): implement scenario model and builder`
Commit: `feat(scenario): add inheritance and matrix expansion`

### 2.3 Conditional Execution

**File:** `pkg/witness/scenario/conditions.go`

```go
type Condition struct {
    Expression string
    Reason     string
}

func (c *Condition) Evaluate(env map[string]string, flags map[string]any) (bool, error)

// Expression parser for:
// env.VAR == "value"
// env.VAR is set
// env.VAR is empty
// env.VAR in ["a", "b"]
// flags.name == true
// time.hour >= 9
// weekday in ["mon", "tue"]
```

**Test File:** `pkg/witness/scenario/conditions_test.go`

Commit: `feat(scenario): add conditional execution support`

### Example: Scenario Configuration

**File:** `examples/basic/scenarios/checkout.yaml`
```yaml
scenarios:
  - name: basic-checkout
    description: "Basic checkout flow demonstrating core features"
    timeout: 30s
    tags: [smoke, checkout]

    flow:
      - setup: CreateUser
      - task: CreateOrder
      - validation: OrderValid

    teardown:
      mode: always
      order: reverse

  - name: checkout-conditional
    description: "Checkout with conditional execution"
    skip_unless:
      - condition: env.RUN_CHECKOUT_TESTS == "true"
        reason: "Checkout tests disabled"
    flow:
      - setup: CreateUser
      - task: CreateOrder
      - validation: OrderValid
```

**File:** `examples/basic/scenarios_test.go`
```go
func TestScenarioLoading(t *testing.T) {
    cfg, err := config.Load("./scenarios/checkout.yaml")
    require.NoError(t, err)

    assert.Len(t, cfg.Scenarios, 2)
    assert.Equal(t, "basic-checkout", cfg.Scenarios[0].Name)
}
```

Commit: `feat(examples): add scenario configuration examples`

### Phase 2 Completion Checkpoint

```bash
go test ./pkg/witness/config/... ./pkg/witness/scenario/... -v
golangci-lint run ./pkg/witness/config/... ./pkg/witness/scenario/...
# Update PROGRESS.md - set Current Phase: 3
```

**Files that should exist:**
- `pkg/witness/config/loader.go`, `schema.go`, `validation.go` + tests
- `pkg/witness/scenario/scenario.go`, `builder.go`, `resolver.go`, `conditions.go` + tests

---

## Phase 3: Infrastructure

**Reference:** `docs/designs/witness/02-infrastructure.md`

### 3.1 Provider Interface

**File:** `pkg/witness/infrastructure/provider.go`

```go
type Provider interface {
    Name() string
    Initialize(ctx context.Context, config map[string]any) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    HealthCheck(ctx context.Context) HealthReport
    Status() ProviderStatus
    Client(name string) (any, error)
}

type FlushableProvider interface {
    Provider
    Flush(ctx context.Context) error
    FlushWithConfig(ctx context.Context, config FlushConfig) error
}

type HealthReport struct {
    Healthy  bool
    Services map[string]ServiceHealth
}

type ProviderStatus int
const (
    StatusStopped ProviderStatus = iota
    StatusStarting
    StatusRunning
    StatusUnhealthy
)
```

**File:** `pkg/witness/infrastructure/manager.go`

```go
type Manager struct {
    providers map[string]Provider
    reuse     ReuseBehavior
    isolation IsolationLevel
}

func (m *Manager) Start(ctx context.Context) error
func (m *Manager) Stop(ctx context.Context) error
func (m *Manager) Reset(ctx context.Context) error // Between scenarios
func (m *Manager) Client(service, name string) (any, error)
```

Commit: `feat(infrastructure): implement provider interface and manager`

### 3.2 TestContainers Provider

**File:** `pkg/witness/infrastructure/testcontainers/provider.go`

```go
type TestContainersProvider struct {
    containers map[string]testcontainers.Container
    config     Config
}

// Implement Provider interface
```

**File:** `pkg/witness/infrastructure/testcontainers/postgres.go`
**File:** `pkg/witness/infrastructure/testcontainers/redis.go`
**File:** `pkg/witness/infrastructure/testcontainers/kafka.go`

Built-in provider configurations.

**Test File:** `pkg/witness/infrastructure/testcontainers/provider_test.go`

Commit: `feat(infrastructure): add TestContainers provider implementation`
Commit: `feat(infrastructure): add built-in Postgres, Redis, Kafka providers`

### 3.3 Secret Management

**File:** `pkg/witness/config/secrets.go`

```go
type SecretProvider interface {
    Resolve(ctx context.Context, key string) (string, error)
    Watch(ctx context.Context, key string, callback func(string)) error
}

type EnvSecretProvider struct{}
type VaultSecretProvider struct{ ... }

func ResolveVariables(cfg *Config, provider SecretProvider) error
```

**Test File:** `pkg/witness/config/secrets_test.go`

Commit: `feat(config): add secret management and variable resolution`

### 3.4 Reuse Behavior

**File:** `pkg/witness/infrastructure/reuse.go`

```go
type ReuseBehavior int
const (
    AlwaysFresh ReuseBehavior = iota
    ReuseWithFlush
    FullReuse
)

type IsolationLevel int
const (
    NoIsolation IsolationLevel = iota
    DataIsolation
    SchemaIsolation
    InstanceIsolation
)
```

Commit: `feat(infrastructure): add reuse behavior and isolation levels`

### Example: Infrastructure

**File:** `examples/infrastructure/witness.yaml`
```yaml
name: infrastructure-example
version: "1.0"

discovery:
  paths: [./]

infrastructure:
  postgres:
    provider: postgres
    config:
      mode: container
      image: postgres:15
    reuse: flush
    flush:
      strategy: truncate
      tables: [users, orders]

  redis:
    provider: redis
    config:
      mode: container
      image: redis:7
    reuse: flush
    flush:
      strategy: flushdb

secrets:
  provider: env
  fallback_to_env: true
```

**File:** `examples/infrastructure/components.go`
```go
// @witness:setup name="SeedDatabase" produces="db_ready:bool"
func SeedDatabase(ctx witness.Context) error {
    db := ctx.Client("postgres").(*sql.DB)
    _, err := db.Exec("INSERT INTO users (id, email) VALUES ($1, $2)", "usr_1", "test@example.com")
    witness.Set(ctx, "db_ready", err == nil)
    return err
}

// @witness:task name="QueryUser" requires="db_ready:bool" produces="user:User"
func QueryUser(ctx witness.Context) (*User, error) {
    db := ctx.Client("postgres").(*sql.DB)
    var user User
    err := db.QueryRow("SELECT id, email FROM users WHERE id = $1", "usr_1").Scan(&user.ID, &user.Email)
    return &user, err
}
```

**File:** `examples/infrastructure/infrastructure_test.go`

Commit: `feat(examples): add infrastructure example with Postgres and Redis`

---

## Phase 4: Execution Engine

**Reference:** `docs/designs/witness/04-execution.md`

### 4.1 Executor

**File:** `pkg/witness/execution/executor.go`

```go
type Executor struct {
    registry    *discovery.Registry
    infra       *infrastructure.Manager
    middleware  []middleware.Middleware
    results     results.Writer
}

func (e *Executor) Run(ctx context.Context, scenario *scenario.Scenario) (*Result, error)
func (e *Executor) RunComponent(ctx Context, component *Component) (any, error)
```

**File:** `pkg/witness/execution/dag.go`

```go
type DAGExecutor struct {
    maxParallel int
    strategy    DAGStrategy // breadth_first | depth_first
}

func (d *DAGExecutor) Execute(ctx context.Context, flow []FlowItem) error
func (d *DAGExecutor) BuildGraph(flow []FlowItem, registry *Registry) (*Graph, error)
```

**Test File:** `pkg/witness/execution/executor_test.go`
**Test File:** `pkg/witness/execution/dag_test.go`

Commit: `feat(execution): implement core executor`
Commit: `feat(execution): add DAG-based parallel execution`

### 4.2 Timeout Handling

**File:** `pkg/witness/execution/timeout.go`

```go
type TimeoutConfig struct {
    Suite     time.Duration
    Scenario  time.Duration
    Component time.Duration
    Warning   time.Duration
}

func WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc)
func CheckDeadline(ctx context.Context) error
```

Commit: `feat(execution): add hierarchical timeout handling`

### 4.3 Teardown Handling

**File:** `pkg/witness/execution/teardown.go`

```go
type TeardownRunner struct {
    mode            TeardownMode // always, on_failure, on_success, never
    order           TeardownOrder // reverse, declared, parallel
    continueOnError bool
}

func (t *TeardownRunner) Run(ctx Context, teardowns []FlowItem, failure error) []error
```

Commit: `feat(execution): add teardown execution with failure handling`

### 4.4 Runner Integration

**File:** `pkg/witness/execution/runner.go`

Integrate all execution components:
```go
type Runner struct {
    config     *config.Config
    registry   *discovery.Registry
    infra      *infrastructure.Manager
    executor   *Executor
    middleware middleware.Middleware
}

func NewRunner(cfg *config.Config) (*Runner, error)
func (r *Runner) RunScenario(ctx context.Context, name string) (*Result, error)
func (r *Runner) RunAll(ctx context.Context, filter Filter) ([]*Result, error)
```

Commit: `feat(execution): add integrated runner`

### Phase 4 Completion Checkpoint (Core Runnable)

> **Milestone:** At this point, the core framework can discover, configure, and execute scenarios.

```bash
# Full test suite for core
go test ./pkg/witness/... -v
golangci-lint run ./pkg/witness/...

# Update PROGRESS.md - set Current Phase: 5
```

**Integration test:** The basic example should now be executable (even without results storage).

---

## Phase 5: Results & Reporting

**Reference:** `docs/designs/witness/05-results-and-reporting.md`

### 5.1 Results Model

**File:** `pkg/witness/results/model.go`

```go
type Result struct {
    ID          RunID
    ScenarioID  ScenarioID
    TraceID     TraceID
    Status      Status
    StartTime   time.Time
    Duration    time.Duration
    Environment string

    Steps       []StepResult
    Error       *ErrorDetail

    // Modifiers used
    Flags       map[string]any
    Options     []string
    ChaosProfiles []string

    // Artifacts
    Logs        []LogEntry
    Metrics     map[string]float64
    Narrative   *Narrative
}

type ErrorDetail struct {
    Category   ErrorCategory
    Message    string
    Stack      string
    Component  string
    Expected   any
    Actual     any
    Suggestion string
    Retryable  bool
}

type ErrorCategory string
const (
    CategoryAssertion      ErrorCategory = "assertion"
    CategoryPrecondition   ErrorCategory = "precondition"
    CategoryInfrastructure ErrorCategory = "infrastructure"
    CategoryNetwork        ErrorCategory = "network"
    CategoryTimeout        ErrorCategory = "timeout"
    CategoryFramework      ErrorCategory = "framework"
    CategoryConfiguration  ErrorCategory = "configuration"
    CategoryDependency     ErrorCategory = "dependency"
    CategoryExternal       ErrorCategory = "external"
    CategoryUnknown        ErrorCategory = "unknown"
)
```

Commit: `feat(results): implement results model with error classification`

### 5.2 Narrative

**File:** `pkg/witness/results/narrative.go`

```go
type Narrative struct {
    RunID   RunID
    TraceID TraceID
    Entries []NarrativeEntry
    Summary NarrativeSummary
}

type NarrativeEntry struct {
    Timestamp time.Time
    Level     NarrativeLevel
    Component string
    SpanID    string
    Action    string
    Details   map[string]any
    Duration  time.Duration
}

type NarrativeRenderer interface {
    Render(n *Narrative) ([]byte, error)
}

type MarkdownRenderer struct{}
type JSONRenderer struct{}
type YAMLRenderer struct{}
```

Commit: `feat(results): add execution narrative with multiple renderers`

### 5.3 Storage Adapters

**File:** `pkg/witness/results/adapter.go`

```go
type Writer interface {
    Write(ctx context.Context, result *Result) error
}

type Reader interface {
    Query(ctx context.Context, filter Filter) ([]*Result, error)
    Get(ctx context.Context, id RunID) (*Result, error)
}

type Adapter interface {
    Writer
    Reader
    Delete(ctx context.Context, filter Filter) error
}
```

**File:** `pkg/witness/results/adapters/filesystem.go`
**File:** `pkg/witness/results/adapters/memory.go`

Commit: `feat(results): add storage adapter interface and implementations`

### 5.4 Report Formats

**File:** `pkg/witness/results/reports/junit.go`
**File:** `pkg/witness/results/reports/html.go`
**File:** `pkg/witness/results/reports/markdown.go`
**File:** `pkg/witness/results/reports/json.go`

Commit: `feat(results): add report generators (JUnit, HTML, Markdown, JSON)`

---

## Phase 6: Chaos & Mocks

**Reference:** `docs/designs/witness/03-scenarios-and-composition.md`

### 6.1 Chaos Profiles

**File:** `pkg/witness/chaos/chaos.go`

```go
type ChaosInjector interface {
    Inject(ctx context.Context, target string, config ChaosConfig) (cleanup func(), err error)
    SupportedTypes() []string
}

type ChaosConfig struct {
    Type        string
    Target      string
    Duration    time.Duration
    Probability float64
    Parameters  map[string]any
}

// Infrastructure chaos
type NetworkLatencyInjector struct{}
type ServiceUnavailableInjector struct{}

// Application chaos
type InvalidInputInjector struct{}
type BoundaryValueInjector struct{}
```

**File:** `pkg/witness/chaos/profile.go`

```go
type Profile struct {
    Name           string
    Infrastructure []ChaosConfig
    Application    []ChaosConfig
}

type Manager struct {
    injectors map[string]ChaosInjector
    active    []func() // cleanup functions
}
```

Commit: `feat(chaos): implement chaos injection framework`
Commit: `feat(chaos): add network latency and service unavailable injectors`

### 6.2 Mock System

**File:** `pkg/witness/mocks/injector.go`

```go
type MockInjector interface {
    Setup(ctx context.Context, mocks map[string]any) error
    Teardown(ctx context.Context) error
    Verify(ctx context.Context, mockName string) (*VerificationResult, error)
}

type VerificationResult struct {
    MockName       string
    ExpectedCalls  string
    ActualCalls    int
    Passed         bool
    UnmatchedCalls []UnmatchedCall
}
```

**File:** `pkg/witness/mocks/wiremock/injector.go`
**File:** `pkg/witness/mocks/inprocess/injector.go`

Commit: `feat(mocks): implement mock injector interface`
Commit: `feat(mocks): add WireMock and in-process mock injectors`

### Example: Chaos & Mocks

**File:** `examples/chaos/witness.yaml`
```yaml
chaos_profiles:
  degraded-network:
    infrastructure:
      - type: network_latency
        target: "*"
        latency_ms: 200
        probability: 0.5

  service-outage:
    infrastructure:
      - type: service_unavailable
        target: payment-service
        duration: 5s

scenarios:
  - name: checkout-under-chaos
    flow:
      - setup: CreateUser
      - task: CreateOrder
      - validation: OrderValid
    chaos:
      profiles: [degraded-network]
```

**File:** `examples/mocks/witness.yaml`
```yaml
mocks:
  injector: inprocess
  verification:
    mode: at_least_once
    on_failure: fail

mock_profiles:
  payment-success:
    payment-gateway:
      stubs:
        - request: { method: POST, path: /charge }
          response: { status: 200, body: { id: "ch_123" } }

scenarios:
  - name: checkout-mocked
    mock_profiles: [payment-success]
    flow:
      - setup: CreateUser
      - task: ProcessPayment
      - validation: PaymentRecorded
```

Commit: `feat(examples): add chaos and mock examples`

### Phase 6 Completion Checkpoint (Full Feature Set)

> **Milestone:** All core features implemented. Chaos and mocks working.

```bash
go test ./pkg/witness/... ./examples/... -v
golangci-lint run ./...

# Update PROGRESS.md - set Current Phase: 7
```

**All framework features complete.** Remaining phases add CLI/API interfaces.

---

## Phase 7: CLI

**Reference:** `docs/designs/witness/11-getting-started.md`

### 7.1 Core Commands

**File:** `cmd/witness/main.go`
**File:** `pkg/witness/cli/cli.go`

Commands:
```
witness init              # Initialize project
witness discover          # Show discovered components
witness validate          # Validate configuration
witness run               # Run scenarios
witness graph             # Show dependency graph
witness results           # Query results
witness report            # Generate reports
witness docs              # Generate documentation
witness serve             # Start daemon mode
witness api-key           # Manage API keys
witness config            # Config management
witness flaky             # Flaky test management
```

**File:** `pkg/witness/cli/run.go`
```go
// witness run --scenario X --tags Y --env Z --flag A=B --option C
```

**File:** `pkg/witness/cli/discover.go`
**File:** `pkg/witness/cli/graph.go`
**File:** `pkg/witness/cli/validate.go`

Commit: `feat(cli): add core CLI commands`
Commit: `feat(cli): add run command with filtering`
Commit: `feat(cli): add discover, graph, and validate commands`

### 7.2 Graph Visualization

**File:** `pkg/witness/cli/graph.go`

```go
func GraphASCII(scenario string, registry *Registry) string
func GraphDOT(scenario string, registry *Registry) string
func GraphMermaid(scenario string, registry *Registry) string
```

Commit: `feat(cli): add dependency graph visualization`

---

## Phase 8: Daemon & API

**Reference:** `docs/designs/witness/07-daemon-service.md`

### 8.1 API Server

**File:** `pkg/witness/daemon/server.go`

```go
type Server struct {
    runner *execution.Runner
    config *config.Config
    auth   *Auth
}

// REST endpoints
// POST   /api/v1/runs
// GET    /api/v1/runs
// GET    /api/v1/runs/:id
// DELETE /api/v1/runs/:id
// GET    /api/v1/scenarios
// GET    /api/v1/components
// GET    /api/v1/results
// GET    /api/v1/health
// POST   /api/v1/config/reload
```

Commit: `feat(daemon): implement REST API server`

### 8.2 Authentication

**File:** `pkg/witness/daemon/auth.go`

```go
type Auth struct {
    method AuthMethod // jwt, api_key, oauth2
    roles  map[string][]string
}

func (a *Auth) Middleware() func(http.Handler) http.Handler
func (a *Auth) CreateAPIKey(name string, scopes []string) (string, error)
func (a *Auth) RevokeAPIKey(key string) error
```

Commit: `feat(daemon): add authentication and authorization`

### 8.3 Event Bus

**File:** `pkg/witness/daemon/eventbus.go`

```go
type EventBus interface {
    Publish(event Event) error
    Subscribe(topic string, handler func(Event)) error
}

type EmbeddedEventBus struct{}
type RedisEventBus struct{}
```

Commit: `feat(daemon): add event bus for internal communication`

### 8.4 Hot Reload

**File:** `pkg/witness/daemon/watcher.go`

```go
type ConfigWatcher struct {
    paths    []string
    debounce time.Duration
    onChange func(*config.Config)
}

func (w *ConfigWatcher) Start(ctx context.Context) error
```

Commit: `feat(daemon): add configuration hot reload`

### Phase 8 Completion Checkpoint (Complete System)

> **Milestone:** Full system with CLI and daemon API.

```bash
# Test everything
go test ./... -v
golangci-lint run ./...

# Verify CLI works
go build -o witness ./cmd/witness
./witness --help
./witness discover --help

# Update PROGRESS.md - set Current Phase: 9
```

**System is feature-complete.** Phase 9-10 are validation and polish.

---

## Phase 9: Full Stack Example

### 9.1 Complete Example

**File:** `examples/full-stack/`

Create a comprehensive example demonstrating:
- Multiple component types (setup, task, validation, teardown)
- Infrastructure (Postgres, Redis)
- Chaos profiles
- Mock profiles
- Environment overlays (local.yaml, staging.yaml)
- Matrix scenarios
- Conditional execution
- Custom middleware
- All CLI commands work

```
examples/full-stack/
├── witness.yaml
├── environments/
│   ├── base.yaml
│   ├── local.yaml
│   └── staging.yaml
├── scenarios/
│   ├── checkout.yaml
│   ├── payment.yaml
│   └── inventory.yaml
├── chaos/
│   └── profiles.yaml
├── mocks/
│   └── profiles.yaml
├── components/
│   ├── types.go
│   ├── setup.go
│   ├── tasks.go
│   ├── validations.go
│   ├── teardown.go
│   └── middleware.go
└── full_stack_test.go
```

**File:** `examples/full-stack/full_stack_test.go`
```go
func TestFullStackDiscovery(t *testing.T) { ... }
func TestFullStackExecution(t *testing.T) { ... }
func TestFullStackWithChaos(t *testing.T) { ... }
func TestFullStackWithMocks(t *testing.T) { ... }
func TestFullStackCLI(t *testing.T) { ... }
```

Commit: `feat(examples): add comprehensive full-stack example`

---

## Phase 10: Documentation & Polish

### 10.1 Update Design Docs

Review all design docs and update any that diverged during implementation.

Commit: `docs: update design docs to match implementation`

### 10.2 Generate Documentation

- Generate component catalog from examples
- Generate API documentation

Commit: `docs: generate API and component documentation`

### 10.3 Update Gap Analysis

Update `gap-analysis.md` to reflect:
- All Chronicle → Witness alignment complete
- All design features implemented
- Coverage metrics

Commit: `docs: update gap analysis - implementation complete`

### 10.4 Final Testing

```bash
# Run all tests
go test ./... -v -race -cover

# Run linter
golangci-lint run

# Run examples
cd examples/basic && go test ./...
cd examples/infrastructure && go test ./...
cd examples/chaos && go test ./...
cd examples/mocks && go test ./...
cd examples/full-stack && go test ./...
```

Commit: `test: ensure all tests pass with coverage`

---

## Execution Notes

### Using Ralph Loops

For complex phases, use ralph loops:

```
/ralph-loop
Objective: Implement Phase 1 Core Framework
Iterate until:
- All core types implemented
- All tests pass
- Example works
```

### Commit Conventions

```
feat(scope): description     # New feature
test(scope): description     # Adding tests
fix(scope): description      # Bug fix
refactor(scope): description # Code restructuring
docs(scope): description     # Documentation
chore(scope): description    # Maintenance
```

Scopes: `core`, `context`, `discovery`, `config`, `scenario`, `infrastructure`, `execution`, `results`, `middleware`, `chaos`, `mocks`, `cli`, `daemon`, `examples`

### Progress Tracking

After each phase, update:
1. Run tests: `go test ./...`
2. Run linter: `golangci-lint run`
3. Verify examples work
4. Commit with appropriate message

### If Blocked

If implementation reveals design issues:
1. Document the issue
2. Propose design change
3. Update design doc
4. Continue implementation

---

## Success Checklist

- [ ] All Phase 1-10 complete
- [ ] `go test ./...` passes with 80%+ coverage
- [ ] `golangci-lint run` passes
- [ ] All examples work
- [ ] CLI commands functional
- [ ] Daemon mode works
- [ ] Design docs accurate
- [ ] Gap analysis updated
- [ ] Ready for PR to main
