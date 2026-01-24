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
