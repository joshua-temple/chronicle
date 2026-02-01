# CLI Reference

Chronicle provides a command-line interface for discovering components, running tests, and managing results.

## Global Flags

These flags work with all commands:

| Flag | Description |
|------|-------------|
| `--config <file>` | Config file (default: `./chronicle.yaml`) |
| `-v, --verbose` | Verbose output |
| `--no-color` | Disable colored output |

## Commands

### init

Initialize a new Chronicle project.

```bash
chronicle init
```

Creates a `chronicle.yaml` configuration file with sensible defaults.

---

### discover

Discover and list annotated components.

```bash
chronicle discover [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-p, --paths <paths>` | Paths to scan (default: from config) |
| `-t, --type <type>` | Filter by type (setup, task, validation, etc.) |
| `-T, --tags <tags>` | Filter by tags |
| `-f, --format <fmt>` | Output format: table, json, yaml |
| `--show-deps` | Show component dependencies |
| `--types-only` | Only show discovered types |

**Examples:**

```bash
# List all components
chronicle discover

# Show only setup components
chronicle discover --type setup

# Filter by tag
chronicle discover --tags smoke

# Show dependencies
chronicle discover --show-deps

# JSON output
chronicle discover --format json
```

---

### validate

Validate configuration and components.

```bash
chronicle validate [flags]
```

Checks:
- Valid YAML syntax
- All dependencies can be satisfied
- No circular dependencies
- Scenarios reference valid components

**Flags:**

| Flag | Description |
|------|-------------|
| `--check-cycles` | Check for circular dependencies (default: true) |
| `--check-deps` | Check dependency satisfaction (default: true) |
| `--strict` | Treat warnings as errors |
| `-s, --scenario <name>` | Validate specific scenario only |

**Examples:**

```bash
# Validate everything
chronicle validate

# Strict mode (fail on warnings)
chronicle validate --strict

# Validate specific scenario
chronicle validate --scenario checkout_flow
```

---

### run

Execute scenarios.

```bash
chronicle run [scenario...] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-T, --tags <tags>` | Run scenarios with matching tags |
| `-X, --exclude-tags <tags>` | Exclude scenarios with tags |
| `-S, --suite <name>` | Run a predefined suite |
| `-F, --flag <key=value>` | Set runtime flag |
| `--option <name>` | Enable option bundle |
| `--chaos <profile>` | Enable chaos profile |
| `--mock <profile>` | Enable mock profile |
| `-t, --timeout <duration>` | Global timeout (default: 30m) |
| `--parallel <n>` | Parallel scenario count (default: 1) |
| `--fail-fast` | Stop on first failure |
| `-o, --output <dir>` | Output directory for results |
| `-f, --format <fmt>` | Output format: text, json, junit |
| `--dry-run` | Show what would run |
| `--list-suites` | List available suites |
| `--daemon` | Run via daemon (auto-starts if needed) |

**Examples:**

```bash
# Run all scenarios
chronicle run

# Run specific scenarios
chronicle run login_test checkout_test

# Run by tag
chronicle run --tags smoke
chronicle run --tags integration --exclude-tags slow

# Run a suite
chronicle run --suite regression

# Parallel execution
chronicle run --parallel 4 --fail-fast

# With chaos and mocks
chronicle run --chaos network_latency --mock payment_failed

# Set flags
chronicle run --flag environment=staging --flag debug=true

# Dry run
chronicle run --dry-run

# Custom output
chronicle run --output ./results --format junit
```

---

### graph

Visualize dependency graphs.

```bash
chronicle graph [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --scenario <name>` | Show graph for specific scenario |
| `-c, --component <name>` | Show graph for specific component |
| `-f, --format <fmt>` | Output format: ascii, dot, mermaid |
| `--depends-on <key>` | Show components depending on key |
| `--show-requires` | Show only what component requires |
| `--show-produces` | Show only what component produces |
| `--reverse` | Show reverse dependencies |

**Examples:**

```bash
# Full dependency graph
chronicle graph

# Mermaid format for documentation
chronicle graph --format mermaid

# Graphviz DOT format
chronicle graph --format dot > deps.dot
dot -Tpng deps.dot -o deps.png

# Component-specific
chronicle graph --component CreateOrder
chronicle graph --component CreateOrder --reverse

# Scenario flow
chronicle graph --scenario checkout_flow
```

---

### results

Query historical test results.

```bash
chronicle results <subcommand> [flags]
```

**Subcommands:**

#### results list

```bash
chronicle results list [flags]
```

| Flag | Description |
|------|-------------|
| `-n, --limit <n>` | Max results to show (default: 20) |
| `--since <time>` | Show results since (e.g., 24h, 7d, 2024-01-01) |
| `-f, --format <fmt>` | Output format: table, json |

#### results show

```bash
chronicle results show <run-id>
```

Shows detailed information about a specific run.

#### results delete

```bash
chronicle results delete <run-id...>
```

Delete one or more results.

**Examples:**

```bash
# List recent results
chronicle results list

# List last 50 results
chronicle results list --limit 50

# Results from last 24 hours
chronicle results list --since 24h

# Show specific run
chronicle results show abc123

# Delete results
chronicle results delete abc123 def456
```

---

### report

Generate reports from test results.

```bash
chronicle report [run-id] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-f, --format <fmt>` | Report format: text, json, junit, html, markdown |
| `-o, --output <file>` | Output file (default: stdout) |
| `--latest` | Use most recent run |

**Examples:**

```bash
# Generate JUnit report
chronicle report --latest --format junit --output results.xml

# Generate HTML report
chronicle report abc123 --format html --output report.html

# JSON to stdout
chronicle report --latest --format json
```

---

### daemon

Start the REST API server.

```bash
chronicle daemon [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--addr <address>` | Listen address (default: :3000) |
| `--watch` | Watch for config changes |
| `--api-key <key>` | API key for authentication |

**Examples:**

```bash
# Start daemon
chronicle daemon

# Custom port
chronicle daemon --addr :8080

# With hot reload
chronicle daemon --watch

# With API key
chronicle daemon --api-key my-secret-key
```

See [Daemon API](daemon.md) for REST API documentation.

---

### version

Show version information.

```bash
chronicle version
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CHRONICLE_CONFIG` | Default config file path |
| `CHRONICLE_VERBOSE` | Enable verbose mode |
| `CHRONICLE_NO_COLOR` | Disable colors |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Validation error |
| 4 | Test failure |
