# Witness: Modular Testing Framework

> A language-agnostic, unopinionated testing framework that provides WYSIWYG test composition with UI, YAML configuration, and chaos engineering capabilities.

---

## Navigation

| Previous | Up | Next |
|----------|----|----- |
| - | - | [Core Framework](./01-core-framework.md) |

---

## Table of Contents

### This Document
- [Vision](#vision)
- [Core Philosophy](#core-philosophy)
- [Architecture Overview](#architecture-overview)
- [Feature Summary](#feature-summary)

### Child Documents
1. [Core Framework](./01-core-framework.md) - Component model, type system, discovery
2. [Infrastructure](./02-infrastructure.md) - Provider interfaces, built-ins, environment overlays
3. [Scenarios & Composition](./03-scenarios-and-composition.md) - Scenarios, chaos, flags, options, mocks
4. [Execution](./04-execution.md) - Execution modes, scheduling, distributed workers
5. [Results & Reporting](./05-results-and-reporting.md) - Storage adapters, report formats, notifications
6. [UI Layer](./06-ui-layer.md) - Web UI, TUI, IDE plugins, WYSIWYG builder
7. [Daemon Service](./07-daemon-service.md) - Service mode, API, scheduling
8. [Multi-Language](./08-multi-language.md) - Go, Python, Java SDKs, portability
9. [Extensibility](./09-extensibility.md) - Plugin system, extension points
10. [Test Intelligence](./10-test-intelligence.md) - Data management, profiling, flaky detection
11. [Getting Started](./11-getting-started.md) - Project structure, quickstart

---

## Vision

Witness is a modular testing framework that provides:

- **Three-layer infrastructure abstraction** (Infra → Seed/Setup → Service)
- **WYSIWYG test composition** via UI with YAML fallback
- **Environment overlays** (testcontainers locally, standalone in deployed environments)
- **Chaos testing** at infrastructure AND application layers
- **Pluggable storage** for results with file-based configurations

### Target Audiences

| Audience | Interface | Description |
|----------|-----------|-------------|
| Basic users | Web UI | Visual scenario builder, point-and-click |
| Advanced users | TUI | Keyboard-driven, works over SSH |
| Professionals | IDE plugins | Inline integration, debugger support |

### Trajectory

Internal tool → Open source framework → Potential enterprise offering

---

## Core Philosophy

The framework is an **orchestrator**, not an implementor. It answers "what runs when and how do I know it worked?" - never "how should you do X."

### Key Principles

1. **Zero-Wiring** - Users write logic, provide config. Framework discovers, wires, and executes.
2. **Unopinionated** - Framework doesn't dictate HOW to do things, only orchestrates.
3. **YAML as Source of Truth** - UI reads/writes YAML. No separate storage.
4. **Composition Over Inheritance** - Build complex tests from simple, reusable components.
5. **Environment Agnostic** - Same tests run locally or against live environments.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    UI Layer                                  │
│  (Web UI │ TUI │ IDE Plugin │ CLI)                          │
├─────────────────────────────────────────────────────────────┤
│                    API Layer                                 │
│  (REST/gRPC - same API backs all UIs)                       │
├─────────────────────────────────────────────────────────────┤
│                  Orchestration Engine                        │
│  (Scheduler │ Executor │ Event Bus │ State Manager)         │
├─────────────────────────────────────────────────────────────┤
│                  Component Registry                          │
│  (Setup │ Task │ Validation │ Step │ Rollup │ Scenario)     │
├─────────────────────────────────────────────────────────────┤
│               Infrastructure Abstraction                     │
│  (Provider Interface │ Client Factory │ Health Checks)      │
├─────────────────────────────────────────────────────────────┤
│                  Storage Adapters                            │
│  (Config Loader │ Results Writers │ Artifact Store)         │
└─────────────────────────────────────────────────────────────┘
```

### Infrastructure Layers (User Perspective)

```
┌─────────────────────────────────────────────────────────────┐
│  LAYER 3: Services Under Test                               │
│  (Your application services - what you're actually testing) │
├─────────────────────────────────────────────────────────────┤
│  LAYER 2: Seed/Staging/Setup                                │
│  (User-provided plumbing that provisions Layer 1)           │
├─────────────────────────────────────────────────────────────┤
│  LAYER 1: Infrastructure                                    │
│  (Base infra the test suite leans against)                  │
└─────────────────────────────────────────────────────────────┘
```

---

## Feature Summary

### Core Framework
- Zero-wiring discovery via annotations
- Component model: Setup → Task → Validation → Step → Rollup → Scenario
- Typed context with declared produces/requires dependencies
- User-annotated types (including aliases for imports)
- `flow` keyword supports any component type
- Typed identifiers (TestID, ScenarioID, ComponentID, TraceID, etc.)
- Composable middleware system for cross-cutting concerns

### Infrastructure
- Three-layer model (Infra → Seed/Setup → Services)
- Built-in providers for common infra (Postgres, Redis, Kafka, etc.)
- Provider interface for custom implementations
- Auto-exposed clients to components
- Reuse behavior (always fresh, flush between tests, full reuse)
- Isolation levels (data, schema, instance)

### Chaos Engineering
- Infrastructure chaos (latency, partitions, outages, resource limits)
- Application chaos (invalid inputs, boundary values, injection)
- Reusable chaos profiles with aliases
- Composable profiles per scenario

### Flags & Options
- Feature flag injection into services under test
- Flag matrix for testing combinations
- Options for scenario mutations/variants
- Composable flags + options + chaos

### Mocking
- Scenario-scoped mock definitions
- Framework-agnostic mock schema (user defines)
- MockInjector interface (user implements injection mechanism)
- Reusable mock profiles

### Bundle Registry
- Infrastructure bundles (pre-packaged service combinations)
- Flag bundles (related feature flags grouped)
- Option bundles (scenario mutation presets)
- Middleware bundles (cross-cutting concern sets)

### Configuration
- YAML as source of truth
- Environment overlays (local, staging, production)
- UI reads/writes YAML
- Export from deployed instances

### Execution
- Adhoc, subset, full suite, scheduled, random interval, triggered, watch
- Distributed workers (optional)
- Retries with backoff strategies

### Test Intelligence
- Test data management (fixtures, generators, snapshots)
- Contract testing / service mocking (record/replay, pacts)
- Performance profiling with baselines
- Flaky test detection and quarantine
- Test impact analysis (run affected tests only)
- Scenario inheritance and matrix parameterization

### Distributed Tracing
- TraceID propagation through all execution
- Header injection for service correlation (W3C, B3, Jaeger, custom)
- Context baggage for test metadata propagation
- Span hierarchy for component nesting

### Results & Reporting
- Pluggable storage adapters
- Multiple report formats
- Configurable notifications
- Execution narrative (human-readable execution story)
- Multiple narrative renderers (Markdown, JSON, YAML)

### UI Layer
- Web UI, TUI, IDE plugins
- Visual WYSIWYG builder
- Dependency graph visualization
- Execution debugger (step-through)

### Daemon/Service Mode
- Scheduler (cron, random intervals)
- API for CI/CD integration
- Distributed worker pool
- Full observability

### Multi-Language
- Go PoC with portable architecture
- Python, Java SDKs follow same patterns
- Polyglot execution support

### Extensibility
- Plugin system for providers, adapters, notifiers, chaos injectors
- Plugin registry (future)

---

## Next Steps

Continue to [Core Framework](./01-core-framework.md) for detailed component model and type system documentation.
