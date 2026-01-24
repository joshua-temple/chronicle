# Results & Reporting

> Storage adapters, report formats, and notifications.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Execution](./04-execution.md) | [Overview](./00-overview.md) | [UI Layer](./06-ui-layer.md) |

---

## Table of Contents

- [Results Model](#results-model)
- [Execution Narrative](#execution-narrative)
- [Storage Adapters](#storage-adapters)
- [Report Formats](#report-formats)
- [Notifications](#notifications)

---

## Results Model

Structured representation of test execution results.

### TestResult Structure

```go
type TestResult struct {
    ID            string
    Scenario      string
    Environment   string
    StartTime     time.Time
    Duration      time.Duration
    Status        Status  // passed, failed, skipped, errored

    // Execution trace
    Steps         []StepResult

    // Chaos applied (if any)
    ChaosProfile  []string
    ChaosEvents   []ChaosEvent

    // Flags and options used
    Flags         map[string]any
    Options       []string

    // Artifacts
    Logs          []LogEntry
    Metrics       map[string]float64
    Snapshots     []Artifact  // screenshots, heap dumps, etc.

    // Failure details
    Error         *ErrorDetail
}

type StepResult struct {
    Name      string
    Type      string  // setup, task, validation, step, rollup
    Status    Status
    Duration  time.Duration
    Error     *ErrorDetail
    Children  []StepResult  // For steps/rollups
}

type ErrorDetail struct {
    Message    string
    Stack      string
    Component  string
    Expected   any
    Actual     any
}
```

---

## Execution Narrative

Beyond raw results, the framework captures a human-readable "story" of test execution that aids debugging and documentation.

### Narrative Model

```go
type ExecutionNarrative struct {
    RunID       RunID
    TraceID     TraceID
    Scenario    string
    Environment string
    StartTime   time.Time
    EndTime     time.Time

    // The story entries in execution order
    Entries     []NarrativeEntry

    // Summary statistics
    Summary     NarrativeSummary
}

type NarrativeEntry struct {
    Timestamp   time.Time
    Level       NarrativeLevel  // trace, debug, info, warn, error
    Component   string          // Which component generated this
    SpanID      string          // For trace correlation
    Action      string          // What was done
    Details     map[string]any  // Contextual data
    Duration    time.Duration   // How long it took (if applicable)
}

type NarrativeLevel string

const (
    LevelTrace NarrativeLevel = "trace"
    LevelDebug NarrativeLevel = "debug"
    LevelInfo  NarrativeLevel = "info"
    LevelWarn  NarrativeLevel = "warn"
    LevelError NarrativeLevel = "error"
)

type NarrativeSummary struct {
    TotalDuration  time.Duration
    ComponentCount int
    ErrorCount     int
    WarningCount   int
    SlowSteps      []string  // Steps exceeding threshold
}
```

### Capturing Narrative

The framework automatically captures entries during execution:

```go
// Inside a component, logging adds to narrative
func CreateUser(ctx witness.Context) error {
    ctx.Narrate(Info, "Creating test user", map[string]any{
        "email": "test@example.com",
    })

    // ... create user ...

    ctx.Narrate(Debug, "User created successfully", map[string]any{
        "user_id": user.ID,
    })

    return nil
}
```

### Automatic Narrative Capture

Framework captures key events automatically:

| Event | Level | Example |
|-------|-------|---------|
| Component start | `info` | "Starting CreateUser" |
| Component end | `info` | "Completed CreateUser in 120ms" |
| Infrastructure event | `debug` | "PostgreSQL container started" |
| Mock setup | `debug` | "Registered 3 stubs for payment-gateway" |
| Chaos injection | `warn` | "Injecting 200ms latency to payment-service" |
| Retry attempt | `warn` | "Retry 2/3 for ProcessPayment" |
| Failure | `error` | "ProcessPayment failed: timeout" |
| Recovery | `info` | "Circuit breaker reset" |

### Narrative Renderers

Narratives can be rendered in multiple formats:

**Markdown:**

```markdown
# Execution Narrative: checkout-flow

**Run ID:** run_abc123
**Trace ID:** trace_xyz789
**Duration:** 4.2s
**Status:** Failed

## Timeline

| Time | Component | Action | Duration |
|------|-----------|--------|----------|
| 0.0s | CreateUser | Starting | - |
| 0.1s | CreateUser | User created (id: usr_123) | 120ms |
| 0.1s | SeedCart | Starting | - |
| 0.3s | SeedCart | Added 3 items to cart | 180ms |
| 0.3s | Checkout | Starting | - |
| 0.5s | Checkout | ⚠️ Injecting 200ms latency | - |
| 2.8s | Checkout | ❌ Failed: timeout after 2s | 2.3s |

## Error Details

**Component:** Checkout
**Message:** timeout after 2s
**Stack:**
    at ProcessPayment (checkout.go:45)
    at Checkout (checkout.go:23)
```

**JSON:**

```json
{
  "run_id": "run_abc123",
  "trace_id": "trace_xyz789",
  "scenario": "checkout-flow",
  "entries": [
    {
      "timestamp": "2024-01-15T10:30:00Z",
      "level": "info",
      "component": "CreateUser",
      "action": "Starting",
      "duration_ms": null
    },
    {
      "timestamp": "2024-01-15T10:30:00.120Z",
      "level": "info",
      "component": "CreateUser",
      "action": "User created",
      "details": { "user_id": "usr_123" },
      "duration_ms": 120
    }
  ]
}
```

**YAML:**

```yaml
run_id: run_abc123
trace_id: trace_xyz789
scenario: checkout-flow
entries:
  - timestamp: 2024-01-15T10:30:00Z
    level: info
    component: CreateUser
    action: Starting
  - timestamp: 2024-01-15T10:30:00.120Z
    level: info
    component: CreateUser
    action: User created
    details:
      user_id: usr_123
    duration_ms: 120
```

### Configuration

```yaml
narrative:
  enabled: true
  level: info  # Minimum level to capture (trace, debug, info, warn, error)

  # Automatic capture settings
  auto_capture:
    component_lifecycle: true  # Start/end of components
    infrastructure_events: true
    mock_events: true
    chaos_events: true
    retry_attempts: true

  # Slow step detection
  slow_threshold: 5s  # Mark steps exceeding this as slow

  # Rendering
  renderers:
    - format: markdown
      output: ./narratives/{{run_id}}.md
    - format: json
      output: ./narratives/{{run_id}}.json

  # Retention
  retention: 30d
```

### CLI Access

```bash
# View narrative for latest run
witness narrative --latest

# View narrative for specific run
witness narrative --run-id run_abc123

# Export narrative
witness narrative --run-id run_abc123 --format markdown --output story.md

# Filter by level
witness narrative --run-id run_abc123 --level warn

# Follow in real-time (during execution)
witness narrative --follow
```

### Visual Debugger Integration

The narrative powers the Visual Execution Debugger in the UI:

```
┌─ Execution Debugger ───────────────────────────────────────────┐
│                                                                │
│  [▶ Play] [⏸ Pause] [⏭ Step] [⏮ Back]     Speed: [1x ▼]      │
│                                                                │
│  Timeline: ═══════●══════════════════════════════════════      │
│            0.0s   0.5s                                  4.2s   │
│                                                                │
│  ┌─ Active Component ─────────────────────────────────────┐   │
│  │  Checkout (0.5s - 2.8s)                                │   │
│  │  Status: Failed                                        │   │
│  │                                                        │   │
│  │  Narrative:                                            │   │
│  │  [0.5s] Starting checkout process                      │   │
│  │  [0.5s] ⚠️ Chaos: Injecting 200ms latency              │   │
│  │  [1.2s] Calling payment service...                     │   │
│  │  [2.8s] ❌ Timeout after 2s                            │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                │
│  Context State at 0.5s:                                        │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ user: { id: "usr_123", email: "test@example.com" }    │   │
│  │ cart: { items: 3, total: 99.99 }                      │   │
│  └────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────┘
```

---

## Storage Adapters

Pluggable storage for persisting results.

### Configuration

```yaml
results:
  adapters:
    # Local filesystem (default)
    - type: filesystem
      path: ./results
      format: json  # or: yaml
      retention: 30d
      partition_by: date  # or: scenario, environment

    # Database
    - type: postgres
      connection: ${DATABASE_URL}
      schema: witness_results
      retention: 90d

    # Cloud storage
    - type: s3
      bucket: my-test-results
      prefix: witness/
      region: us-east-1

    # Time-series (for metrics)
    - type: prometheus
      pushgateway: http://prometheus:9091
      job_name: witness

    # External systems
    - type: webhook
      url: https://my-system.com/results
      format: json
      headers:
        Authorization: Bearer ${WEBHOOK_TOKEN}

    # Multiple adapters active simultaneously
```

### Built-in Adapters

| Adapter | Description |
|---------|-------------|
| `filesystem` | Local files (JSON/YAML) |
| `postgres` | PostgreSQL database |
| `mysql` | MySQL database |
| `s3` | AWS S3 bucket |
| `gcs` | Google Cloud Storage |
| `prometheus` | Prometheus pushgateway |
| `elasticsearch` | Elasticsearch index |
| `webhook` | HTTP POST to external system |

### Adapter Interface

For custom adapters:

```go
type ResultsAdapter interface {
    Write(ctx context.Context, result *TestResult) error
    Query(ctx context.Context, filter ResultFilter) ([]TestResult, error)
    Delete(ctx context.Context, filter ResultFilter) error
}

type ResultFilter struct {
    Scenarios   []string
    Environments []string
    Status      []Status
    StartTime   *time.Time
    EndTime     *time.Time
    Tags        []string
    Limit       int
    Offset      int
}
```

---

## Report Formats

Generate reports in various formats.

### CLI Report Generation

```bash
# Generate HTML report
witness report --format html --output report.html

# Generate JUnit XML (CI integration)
witness report --format junit --output results.xml

# Generate markdown (for PRs)
witness report --format markdown --output report.md

# Diff against baseline
witness report --diff baseline.json --format markdown

# Report for specific run
witness report --run-id abc123 --format html
```

### Built-in Formats

| Format | Description | Use Case |
|--------|-------------|----------|
| `json` | Raw structured data | API consumption, further processing |
| `junit` | JUnit XML | CI/CD integration (Jenkins, GitHub Actions) |
| `html` | Interactive dashboard | Human-readable, shareable |
| `markdown` | Markdown tables | PR comments, documentation |
| `csv` | Comma-separated values | Spreadsheet analysis |
| `tap` | Test Anything Protocol | Unix tooling integration |

### HTML Report Features

```
┌─ Test Results Dashboard ────────────────────────────────────┐
│                                                             │
│  Summary: 45 passed, 3 failed, 2 skipped    Duration: 4m32s │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  [Passed] [Failed ▼] [Skipped] [All]                │   │
│  │                                                      │   │
│  │  ✗ checkout-flow (2.3s)                             │   │
│  │    └─ ProcessPayment: timeout after 30s             │   │
│  │                                                      │   │
│  │  ✗ inventory-sync (1.1s)                            │   │
│  │    └─ ValidateStock: expected 100, got 0            │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Expand All] [Collapse All] [Export JSON] [Share Link]    │
└─────────────────────────────────────────────────────────────┘
```

### Custom Report Templates

```yaml
reports:
  custom:
    - name: team-summary
      template: ./templates/team-summary.html
      output: ./reports/team-{{date}}.html
      schedule: daily

    - name: pr-comment
      template: ./templates/pr-comment.md
      variables:
        include_logs: false
        max_failures: 5
```

---

## Notifications

Alert on test results via various channels.

### Configuration

```yaml
notifications:
  channels:
    slack:
      webhook_url: ${SLACK_WEBHOOK}

    email:
      smtp_host: smtp.example.com
      smtp_port: 587
      from: witness@example.com

    pagerduty:
      routing_key: ${PAGERDUTY_KEY}

    teams:
      webhook_url: ${TEAMS_WEBHOOK}

    webhook:
      url: https://my-system.com/notify
```

### Notification Rules

```yaml
notifications:
  rules:
    # On any failure
    - on: failure
      channels: [slack]
      template: |
        🔴 *{{.Scenario}}* failed in {{.Environment}}
        Duration: {{.Duration}}
        Error: {{.Error.Message}}

    # On critical test failure
    - on: failure
      conditions:
        tags: [critical]
      channels: [slack, pagerduty]
      template: |
        🚨 CRITICAL: *{{.Scenario}}* failed
        {{.Error.Message}}

    # On success (specific schedules)
    - on: success
      conditions:
        schedules: [nightly-regression]
      channels: [slack]
      template: |
        ✅ Nightly regression passed
        {{.Passed}} passed, {{.Failed}} failed, {{.Skipped}} skipped

    # On flaky detection
    - on: flaky_detected
      channels: [slack]
      template: |
        ⚠️ Flaky test detected: *{{.Scenario}}*
        Failure rate: {{.FlakeRate}}%
```

### Severity Levels

```yaml
notifications:
  severity_mapping:
    critical:
      tags: [critical, production]
      channels: [pagerduty, slack]

    warning:
      tags: [regression]
      channels: [slack, email]

    info:
      default: true
      channels: [slack]
```

### Notification Grouping

Avoid alert fatigue:

```yaml
notifications:
  grouping:
    enabled: true
    window: 5m  # Group failures within 5 minutes
    max_per_group: 10
    template: |
      🔴 {{.Count}} tests failed in {{.Environment}}
      Scenarios: {{range .Scenarios}}{{.Name}}, {{end}}
```

---

## Next Steps

Continue to [UI Layer](./06-ui-layer.md) for Web UI, TUI, IDE plugins, and WYSIWYG builder.
