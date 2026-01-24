# Extensibility

> Plugin system and extension points.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| [Multi-Language](./08-multi-language.md) | [Overview](./00-overview.md) | [Test Intelligence](./10-test-intelligence.md) |

---

## Table of Contents

- [Extension Points](#extension-points)
- [Plugin Interface](#plugin-interface)
- [Built-in Plugin Types](#built-in-plugin-types)
- [Plugin Development](#plugin-development)
- [Plugin Distribution](#plugin-distribution)

---

## Extension Points

The framework provides multiple points for extension without modifying core.

| Extension Type | Purpose | Example |
|----------------|---------|---------|
| **Infrastructure Provider** | Custom infra backends | `MongoProvider`, `CockroachProvider` |
| **Results Adapter** | Custom result storage | `DatadogAdapter`, `SplunkAdapter` |
| **Notifier** | Custom alert channels | `TeamsNotifier`, `DiscordNotifier` |
| **Chaos Injector** | Custom chaos types | `CustomLatencyInjector` |
| **Report Format** | Custom output formats | `ConfluenceReport`, `PDFReport` |
| **Mock Injector** | Custom mock mechanisms | `IstioMockInjector` |
| **Discovery Scanner** | Custom annotation patterns | Alternative annotation styles |

---

## Plugin Interface

All plugins implement a base interface:

```go
// Base plugin interface
type Plugin interface {
    // Metadata
    Name() string
    Version() string
    Description() string

    // Lifecycle
    Initialize(config map[string]any) error
    Shutdown() error

    // Health
    HealthCheck(ctx context.Context) error
}
```

### Plugin Metadata

```go
type PluginMetadata struct {
    Name        string
    Version     string
    Description string
    Author      string
    License     string
    Homepage    string
    Tags        []string
}
```

---

## Built-in Plugin Types

### Infrastructure Provider Plugin

```go
type InfraProviderPlugin interface {
    Plugin

    // InfraProvider interface
    Initialize(ctx context.Context, config Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    HealthCheck(ctx context.Context) HealthReport
    Status() ProviderStatus
    Client(name string) (any, error)
}
```

### Results Adapter Plugin

```go
type ResultsAdapterPlugin interface {
    Plugin

    // Results operations
    Write(ctx context.Context, result *TestResult) error
    Query(ctx context.Context, filter ResultFilter) ([]TestResult, error)
    Delete(ctx context.Context, filter ResultFilter) error
}
```

### Notifier Plugin

```go
type NotifierPlugin interface {
    Plugin

    // Notification
    Notify(ctx context.Context, event NotificationEvent) error
    ValidateConfig(config map[string]any) error
}

type NotificationEvent struct {
    Type        string  // "failure", "success", "flaky", etc.
    Scenario    string
    Environment string
    Status      Status
    Error       *ErrorDetail
    Duration    time.Duration
    Metadata    map[string]any
}
```

### Chaos Injector Plugin

```go
type ChaosInjectorPlugin interface {
    Plugin

    // Chaos operations
    Inject(ctx context.Context, target string, config ChaosConfig) (cleanup func(), err error)
    SupportedTypes() []string
}

type ChaosConfig struct {
    Type       string
    Target     string
    Duration   time.Duration
    Parameters map[string]any
}
```

### Report Format Plugin

```go
type ReportFormatPlugin interface {
    Plugin

    // Report generation
    Generate(ctx context.Context, results []TestResult, options ReportOptions) ([]byte, error)
    ContentType() string
    FileExtension() string
}
```

### Mock Injector Plugin

```go
type MockInjectorPlugin interface {
    Plugin

    // Mock operations
    Setup(ctx context.Context, mocks map[string]any) error
    Teardown(ctx context.Context) error
    Verify(ctx context.Context, mockName string) (any, error)
}
```

---

## Plugin Development

### Example: MongoDB Provider

```go
// plugins/mongodb/provider.go
package main

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type MongoProvider struct {
    client *mongo.Client
    config map[string]any
}

func (p *MongoProvider) Name() string        { return "mongodb-provider" }
func (p *MongoProvider) Version() string     { return "1.0.0" }
func (p *MongoProvider) Description() string { return "MongoDB infrastructure provider" }

func (p *MongoProvider) Initialize(config map[string]any) error {
    p.config = config
    return nil
}

func (p *MongoProvider) Start(ctx context.Context) error {
    uri := p.config["uri"].(string)
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return err
    }
    p.client = client
    return nil
}

func (p *MongoProvider) Stop(ctx context.Context) error {
    return p.client.Disconnect(ctx)
}

func (p *MongoProvider) HealthCheck(ctx context.Context) HealthReport {
    err := p.client.Ping(ctx, nil)
    status := "healthy"
    if err != nil {
        status = "unhealthy"
    }
    return HealthReport{
        Healthy: err == nil,
        Services: map[string]ServiceHealth{
            "mongodb": {Name: "mongodb", Status: status, Error: err},
        },
    }
}

func (p *MongoProvider) Status() ProviderStatus {
    if p.client != nil {
        return ProviderStatusRunning
    }
    return ProviderStatusStopped
}

func (p *MongoProvider) Client(name string) (any, error) {
    return p.client, nil
}

func (p *MongoProvider) Shutdown() error {
    return nil
}

// Export for plugin loading
var Plugin MongoProvider
```

### Example: Teams Notifier

```go
// plugins/teams/notifier.go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
)

type TeamsNotifier struct {
    webhookURL string
}

func (n *TeamsNotifier) Name() string        { return "teams-notifier" }
func (n *TeamsNotifier) Version() string     { return "1.0.0" }
func (n *TeamsNotifier) Description() string { return "Microsoft Teams notification plugin" }

func (n *TeamsNotifier) Initialize(config map[string]any) error {
    n.webhookURL = config["webhook_url"].(string)
    return nil
}

func (n *TeamsNotifier) ValidateConfig(config map[string]any) error {
    if _, ok := config["webhook_url"]; !ok {
        return errors.New("webhook_url is required")
    }
    return nil
}

func (n *TeamsNotifier) Notify(ctx context.Context, event NotificationEvent) error {
    card := map[string]any{
        "@type":    "MessageCard",
        "summary":  fmt.Sprintf("Test %s: %s", event.Status, event.Scenario),
        "sections": []map[string]any{
            {
                "activityTitle": event.Scenario,
                "facts": []map[string]string{
                    {"name": "Status", "value": string(event.Status)},
                    {"name": "Environment", "value": event.Environment},
                    {"name": "Duration", "value": event.Duration.String()},
                },
            },
        },
    }

    body, _ := json.Marshal(card)
    req, _ := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    _, err := http.DefaultClient.Do(req)
    return err
}

func (n *TeamsNotifier) Shutdown() error { return nil }

var Plugin TeamsNotifier
```

---

## Plugin Distribution

### Local Plugin

```yaml
plugins:
  - name: mongodb-provider
    path: ./plugins/mongodb.so
    config:
      uri: mongodb://localhost:27017
```

### Remote Plugin (Registry)

```yaml
plugins:
  - name: datadog-results
    source: registry.witness.dev/datadog-adapter:1.2.0
    config:
      api_key: ${DATADOG_API_KEY}
      site: datadoghq.com
```

### Git-Based Plugin

```yaml
plugins:
  - name: custom-chaos
    source: github.com/myorg/witness-chaos-plugins@v1.0.0
    config:
      # plugin-specific config
```

### Plugin Management CLI

```bash
# List installed plugins
witness plugin list

# Install plugin
witness plugin install registry.witness.dev/datadog-adapter:1.2.0

# Update plugin
witness plugin update datadog-adapter

# Remove plugin
witness plugin remove datadog-adapter

# Verify plugin
witness plugin verify mongodb-provider
```

### Plugin Registry (Future)

```
┌─ Witness Plugin Registry ────────────────────────────────┐
│                                                          │
│  Infrastructure          Results           Notifiers     │
│  ──────────────          ───────           ─────────     │
│  ● mongodb      ★4.8     ● datadog  ★4.9   ● teams ★4.7  │
│  ● cockroach    ★4.5     ● splunk   ★4.6   ● discord★4.5 │
│  ● neo4j        ★4.2     ● elastic  ★4.8   ● opsgenie★4.6│
│  ● mssql        ★4.4     ● grafana  ★4.7   ● victorops★4.3│
│                                                          │
│  [Browse All] [Submit Plugin]                           │
└──────────────────────────────────────────────────────────┘
```

---

## Next Steps

Continue to [Test Intelligence](./10-test-intelligence.md) for data management, profiling, and flaky detection.
