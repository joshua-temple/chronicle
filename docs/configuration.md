# Configuration Reference

Chronicle is configured through `chronicle.yaml`. This document covers all available options.

## Minimal Configuration

```yaml
name: my-project
version: "1.0"

discovery:
  paths:
    - ./

scenarios: []
```

## Full Configuration Schema

```yaml
# Project identification
name: my-project
version: "1.0"

# Component discovery
discovery:
  paths:
    - ./components
    - ./tests
  exclude:
    - ./vendor
    - ./*_test.go

# Infrastructure providers
infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
    ports:
      - container: 5432
        host: 5432
    env:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
    health_check:
      command: ["pg_isready"]
      interval: 1s
      timeout: 30s
      retries: 30
    reuse:
      enabled: true
      ttl: 1h
    resources:
      memory: 512m
      cpu: "0.5"

# Test scenarios
scenarios:
  - name: scenario_name
    description: What this scenario tests
    timeout: 5m
    tags: [smoke, integration]
    flow:
      - setup: ComponentName
      - task: ComponentName
      - validation: ComponentName
    teardown:
      - teardown: ComponentName
    flags:
      key: value
    options: [option_name]
    chaos_profiles: [profile_name]
    mock_profiles: [profile_name]
    skip_if:
      - env: CI
        reason: Skip in CI
    skip_unless:
      - flag: run_slow
        reason: Requires --flag run_slow
    matrix:
      param: [value1, value2]
    extends: base_scenario
    abstract: false

# Predefined test suites
suites:
  smoke:
    description: Quick smoke tests
    scenarios: [test_a, test_b]
    tags: [smoke]
    exclude_tags: [slow]
    parallel: 4
    fail_fast: true

# Chaos engineering profiles
chaos_profiles:
  network_latency:
    description: Simulate slow network
    network:
      latency:
        enabled: true
        min: 100ms
        max: 500ms
        jitter: 0.1
      packet_loss:
        enabled: false
        percentage: 0
      partition:
        enabled: false
        duration: 0s
        targets: []
    resource:
      cpu:
        enabled: false
        percentage: 0
        duration: 0s
      memory:
        enabled: false
        percentage: 0
        duration: 0s
      io:
        enabled: false
        percentage: 0
        duration: 0s
    custom:
      before: []
      during: []
      after: []

# Mock profiles
mock_profiles:
  payment_declined:
    description: Simulate payment failure
    services:
      - name: payment-api
        type: http
        rules:
          - match:
              method: POST
              path: /api/payments
              headers:
                Content-Type: application/json
              body: ""
              body_json:
                amount: 0
            response:
              status: 402
              headers:
                Content-Type: application/json
              body: '{"error": "declined"}'
              file: ""
            delay: 0s
        fallback:
          action: error
          status: 500
          body: '{"error": "unexpected"}'
        passthrough: false

# Runtime flags
flags:
  definitions:
    environment:
      type: string
      default: local
      description: Target environment
      choices: [local, staging, production]
    debug:
      type: bool
      default: false
      description: Enable debug logging
    retries:
      type: int
      default: 3
      description: Number of retries
  defaults:
    environment: local
    debug: false

# Named option bundles
options:
  verbose:
    description: Enable verbose output
    flags:
      debug: true
      log_level: debug
    middleware: [logging]
    tags: [verbose]

# Reusable bundles
bundles:
  infrastructure:
    database: [postgres, redis]
  flags:
    production: [environment=production, strict=true]
  options:
    full_logging: [verbose, trace]
  middleware:
    standard: [logging, metrics]

# Secret management
secrets:
  provider: env  # env, file, vault
  path: ""
  mapping:
    DB_PASSWORD: POSTGRES_PASSWORD
  fallback_to_env: true
  vault:
    address: ""
    token: ""
    path: ""

# Execution settings
execution:
  parallelism: 1
  default_timeout: 5m
  retry:
    max_retries: 0
    backoff:
      type: exponential
      initial: 1s
      max: 30s
      multiplier: 2
      jitter: true
  teardown_mode: always  # always, on_failure, never
  fail_fast: false
  shuffle_scenarios: false

# Results storage and reporting
results:
  storage:
    type: file  # file, s3, gcs, database
    path: .chronicle/results
    bucket: ""
    region: ""
    config: {}
  reports:
    - format: junit
      path: ./reports/junit.xml
    - format: html
      path: ./reports/report.html
  retention:
    days: 30
    count: 100
    cleanup: true

# Notifications
notifications:
  slack:
    enabled: false
    webhook_url: ""
    channel: "#testing"
    on_events: [complete, failure]
  email:
    enabled: false
    smtp_server: ""
    from: ""
    to: []
    on_events: [failure]
  webhook:
    enabled: false
    url: ""
    headers: {}
    on_events: [start, complete, failure]
```

## Section Reference

### discovery

Configure where Chronicle looks for annotated components.

| Field | Type | Description |
|-------|------|-------------|
| `paths` | []string | Directories to scan |
| `exclude` | []string | Patterns to exclude |

### infrastructure

Configure test infrastructure. See [Infrastructure](infrastructure.md).

| Field | Type | Description |
|-------|------|-------------|
| `provider` | string | Provider type (testcontainers) |
| `image` | string | Docker image |
| `ports` | []PortConfig | Port mappings |
| `env` | map[string]string | Environment variables |
| `volumes` | []VolumeConfig | Volume mounts |
| `health_check` | HealthCheckConfig | Health check configuration |
| `reuse` | ReuseConfig | Reuse behavior |
| `depends_on` | []string | Dependencies |
| `resources` | ResourcesConfig | Resource limits |
| `compose_file` | string | Docker Compose file path |
| `services` | []string | Services from compose file |

### scenarios

Define test scenarios. See [Scenarios](scenarios.md).

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Scenario name (required) |
| `description` | string | Human-readable description |
| `timeout` | duration | Execution timeout |
| `tags` | []string | Tags for filtering |
| `flow` | []FlowItem | Execution flow |
| `teardown` | []FlowItem | Teardown flow |
| `flags` | map[string]any | Runtime flags |
| `options` | []string | Option bundles to enable |
| `chaos_profiles` | []string | Chaos profiles to apply |
| `mock_profiles` | []string | Mock profiles to apply |
| `skip_if` | []Condition | Skip conditions |
| `skip_unless` | []Condition | Skip-unless conditions |
| `matrix` | map[string][]any | Matrix parameters |
| `extends` | string | Parent scenario |
| `abstract` | bool | Cannot be run directly |

### suites

Group scenarios into suites.

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Suite description |
| `scenarios` | []string | Explicit scenario list |
| `tags` | []string | Include scenarios with tags |
| `exclude_tags` | []string | Exclude scenarios with tags |
| `parallel` | int | Parallel execution count |
| `fail_fast` | bool | Stop on first failure |

### chaos_profiles

Configure chaos engineering. See [Chaos Testing](chaos.md).

### mock_profiles

Configure mocking. See [Mocking](mocking.md).

### flags

Define runtime flags and their defaults.

| Field | Type | Description |
|-------|------|-------------|
| `definitions.<name>.type` | string | Type: bool, string, int, float, []string |
| `definitions.<name>.default` | any | Default value |
| `definitions.<name>.description` | string | Flag description |
| `definitions.<name>.required` | bool | Is required |
| `definitions.<name>.choices` | []string | Valid choices |
| `defaults` | map[string]any | Default flag values |

### execution

Configure test execution behavior.

| Field | Type | Description |
|-------|------|-------------|
| `parallelism` | int | Concurrent scenarios |
| `default_timeout` | duration | Default timeout |
| `retry.max_retries` | int | Retry count |
| `retry.backoff.type` | string | constant, exponential, linear |
| `retry.backoff.initial` | duration | Initial backoff |
| `retry.backoff.max` | duration | Max backoff |
| `retry.backoff.multiplier` | float | Exponential multiplier |
| `retry.backoff.jitter` | bool | Add randomness |
| `teardown_mode` | string | always, on_failure, never |
| `fail_fast` | bool | Stop on first failure |
| `shuffle_scenarios` | bool | Randomize order |

### results

Configure results storage and reporting.

| Field | Type | Description |
|-------|------|-------------|
| `storage.type` | string | file, s3, gcs, database |
| `storage.path` | string | Local storage path |
| `storage.bucket` | string | Cloud bucket name |
| `reports` | []ReportConfig | Report configurations |
| `retention.days` | int | Keep results for N days |
| `retention.count` | int | Keep last N results |
| `retention.cleanup` | bool | Auto-cleanup old results |

### notifications

Configure notifications on test events.

| Field | Type | Description |
|-------|------|-------------|
| `slack.enabled` | bool | Enable Slack |
| `slack.webhook_url` | string | Webhook URL |
| `slack.channel` | string | Channel name |
| `slack.on_events` | []string | Events to notify |
| `email.enabled` | bool | Enable email |
| `email.smtp_server` | string | SMTP server |
| `email.from` | string | From address |
| `email.to` | []string | Recipients |
| `webhook.enabled` | bool | Enable webhook |
| `webhook.url` | string | Webhook URL |
| `webhook.headers` | map[string]string | Custom headers |

## Environment Variables

Configuration values can reference environment variables:

```yaml
infrastructure:
  postgres:
    env:
      POSTGRES_PASSWORD: ${DB_PASSWORD}
```

Or use the `secrets` section for managed secrets.

## Config File Locations

Chronicle looks for configuration in:

1. `--config` flag value
2. `./chronicle.yaml`
3. `./chronicle.yml`
4. Parent directories (up to 3 levels)

## Validation

Validate your configuration:

```bash
chronicle validate
```

This checks:
- YAML syntax
- Required fields
- Reference validity (scenarios reference real components)
- Dependency satisfaction
- Circular dependencies
