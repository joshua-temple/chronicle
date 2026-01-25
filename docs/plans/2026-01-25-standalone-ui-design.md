# Chronicle Standalone UI Design

> Multi-project control center for Chronicle test framework.

## Overview

Extend Chronicle's UI to support a standalone "control center" mode that manages multiple projects, detects running daemon instances, and can launch the framework on demand.

**Goals:**
- Developer workstation: Switch between multiple local projects seamlessly
- Team dashboard: Central visibility into test status across projects
- Zero friction: Auto-discover projects, remember preferences, one-click launch

**Non-goals:**
- Network discovery (mDNS) - too fragile, platform-dependent
- Electron/native app - adds distribution complexity
- Result caching - better fetched fresh from daemon

---

## Architecture

### Deployment Model

Single binary, two modes via subcommand:

```bash
chronicle ui              # Single-project mode (current behavior)
chronicle ui --standalone # Multi-project control center
```

The React SPA detects mode via `/api/standalone/mode` endpoint and renders accordingly.

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Chronicle Standalone UI                       │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Project   │  │   Daemon    │  │   Process   │              │
│  │  Registry   │  │  Connector  │  │  Launcher   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                      │
│         ▼                ▼                ▼                      │
│  ~/.chronicle/     Health checks    spawn chronicle              │
│  projects.json     + SSE streams    daemon processes             │
└─────────────────────────────────────────────────────────────────┘
                           │
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │  Project A │  │  Project B │  │  Project C │
    │  (local)   │  │  (local)   │  │  (remote)  │
    │  :8080     │  │  stopped   │  │  :9090     │
    └────────────┘  └────────────┘  └────────────┘
```

---

## Project Registry

### Storage Location

```
~/.chronicle/projects.json
```

### Schema

```json
{
  "version": 1,
  "projects": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "my-service",
      "path": "/Users/josh/code/my-service",
      "remoteUrl": null,
      "addedAt": "2026-01-20T08:00:00Z",
      "lastOpened": "2026-01-25T10:30:00Z",
      "lastScenarios": ["smoke-test", "full-integration"],
      "preferences": {
        "defaultTab": "scenarios",
        "scenarioFilter": "tag:critical"
      }
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "name": "team-dashboard",
      "path": null,
      "remoteUrl": "https://chronicle.internal.company.com:8080",
      "addedAt": "2026-01-22T14:00:00Z",
      "lastOpened": "2026-01-24T16:45:00Z",
      "lastScenarios": [],
      "preferences": {}
    }
  ],
  "settings": {
    "autoDiscover": true,
    "pollIntervalMs": 30000,
    "activePollIntervalMs": 5000
  }
}
```

### Auto-Discovery

When `autoDiscover` is enabled, scan for Chronicle projects on startup:

1. Check current directory for `chronicle.yaml`
2. Check recent git repositories (parse `~/.gitconfig` or shell history)
3. Check common development paths (`~/code/**/chronicle.yaml`, `~/projects/**/chronicle.yaml`)

Auto-discovered projects are added to registry with `autoDiscovered: true` flag. Users can pin or remove them.

---

## Daemon Status Detection

### Health Check Strategy

**Smart polling** with three tiers:

| Context | Interval | Method |
|---------|----------|--------|
| Active project | 5s | SSE stream (push) |
| Background projects | 30s | HTTP GET /health |
| Inactive (tab hidden) | 60s | HTTP GET /health |

### Status States

```typescript
type DaemonStatus =
  | { state: 'unknown' }           // Never checked
  | { state: 'stopped' }           // Health check failed
  | { state: 'starting' }          // Launch initiated, waiting
  | { state: 'running', port: number, version: string }
  | { state: 'unhealthy', error: string }
```

### Health Endpoint

Daemon exposes `/health` returning:

```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "2h34m",
  "scenarios": 12,
  "lastRun": "2026-01-25T10:15:00Z"
}
```

---

## Process Launcher

### Launch Flow

When user clicks "Launch" on a stopped project:

1. **Validate** - Check project path exists, `chronicle.yaml` present
2. **Find port** - Use configured port or find available one
3. **Spawn** - Execute `chronicle daemon --port <port>` in project directory
4. **Monitor** - Poll `/health` until responsive (timeout: 30s)
5. **Connect** - Establish SSE stream, update status to "running"

### Implementation

```go
type Launcher struct {
    processes map[string]*exec.Cmd  // projectID -> process
    mu        sync.RWMutex
}

func (l *Launcher) Launch(ctx context.Context, project Project) error {
    cmd := exec.CommandContext(ctx, "chronicle", "daemon", "--port", port)
    cmd.Dir = project.Path
    cmd.Stdout = // capture for logs
    cmd.Stderr = // capture for logs

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start daemon: %w", err)
    }

    l.processes[project.ID] = cmd

    // Wait for health
    return l.waitForHealth(ctx, project, 30*time.Second)
}

func (l *Launcher) Stop(ctx context.Context, projectID string) error {
    cmd, ok := l.processes[projectID]
    if !ok {
        return nil
    }

    // Graceful shutdown via signal
    cmd.Process.Signal(os.Interrupt)

    // Wait with timeout
    done := make(chan error)
    go func() { done <- cmd.Wait() }()

    select {
    case <-time.After(10 * time.Second):
        cmd.Process.Kill()
    case <-done:
    }

    delete(l.processes, projectID)
    return nil
}
```

---

## UI Design

### Project Selector View

```
┌─────────────────────────────────────────────────────────────┐
│  Chronicle Control Center                      [+ Add Project] │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ ● my-service                    Running on :8080         │ │
│  │   /Users/josh/code/my-service   Last run: 2 min ago      │ │
│  │                                           [Open] [Stop]  │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ ○ payment-api                   Stopped                  │ │
│  │   /Users/josh/code/payment-api  Last run: 3 days ago     │ │
│  │                                         [Open] [Launch]  │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ ● team-dashboard                Running (remote)         │ │
│  │   https://chronicle.internal    12 scenarios             │ │
│  │                                                  [Open]  │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ─────────────────────────────────────────────────────────── │
│  Recently discovered:                                         │
│  ○ user-service  /Users/josh/code/user-svc     [Add] [Ignore] │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Project Detail View

Once a project is opened, show the existing Chronicle UI (scenarios, runs, components, config) with a "← Back to Projects" link.

### Add Project Modal

```
┌────────────────────────────────────────┐
│  Add Project                       [X] │
├────────────────────────────────────────┤
│                                        │
│  ○ Local project                       │
│    Path: [________________________] 📁 │
│                                        │
│  ○ Remote daemon                       │
│    URL:  [________________________]    │
│                                        │
│  Name: [________________________]      │
│        (auto-detected from config)     │
│                                        │
│              [Cancel]  [Add Project]   │
└────────────────────────────────────────┘
```

---

## API Endpoints

### New Endpoints (Standalone Mode)

```
GET  /api/standalone/mode          # Returns { mode: "standalone" | "single" }
GET  /api/standalone/projects      # List all registered projects with status
POST /api/standalone/projects      # Add a project
DELETE /api/standalone/projects/:id # Remove a project
PUT  /api/standalone/projects/:id  # Update project preferences

POST /api/standalone/projects/:id/launch  # Start daemon for project
POST /api/standalone/projects/:id/stop    # Stop daemon for project
GET  /api/standalone/projects/:id/health  # Check specific project health

GET  /api/standalone/discover      # Trigger auto-discovery scan
```

### Response Examples

```json
// GET /api/standalone/projects
{
  "projects": [
    {
      "id": "...",
      "name": "my-service",
      "path": "/Users/josh/code/my-service",
      "status": {
        "state": "running",
        "port": 8080,
        "version": "0.1.0"
      },
      "lastOpened": "2026-01-25T10:30:00Z"
    }
  ]
}

// POST /api/standalone/projects/:id/launch
{
  "success": true,
  "port": 8081,
  "pid": 12345
}
```

---

## Implementation Plan

### Phase 1: Core Infrastructure
1. Create `pkg/ui/standalone/` package
2. Implement ProjectRegistry with file persistence
3. Add `--standalone` flag to `chronicle ui` command
4. Create `/api/standalone/*` endpoints

### Phase 2: Daemon Management
5. Implement health checker with smart polling
6. Implement process launcher
7. Add SSE forwarding for active project

### Phase 3: React UI
8. Create ProjectSelector component
9. Add mode detection to App.tsx
10. Implement project cards with status indicators
11. Add/remove project modals

### Phase 4: Auto-Discovery
12. Implement directory scanner
13. Git history integration (optional)
14. "Recently discovered" UI section

### Phase 5: Polish
15. Keyboard shortcuts (Cmd+1/2/3 for projects)
16. Preferences persistence
17. Error handling and edge cases

---

## Open Questions (Decided)

| Question | Decision |
|----------|----------|
| Separate repo? | No - same repo, standalone artifact |
| Discovery method? | Auto-discover local + manual remote |
| Polling strategy? | Smart polling (active=5s, background=30s) |
| Memory scope? | Moderate (projects + last scenarios + preferences) |
| Packaging? | Subcommand: `chronicle ui --standalone` |

---

## Success Criteria

1. User can add local and remote projects
2. Projects show live status (running/stopped)
3. One-click launch starts daemon and connects
4. Switching projects preserves context (last scenarios, filters)
5. Auto-discovery finds Chronicle projects without manual setup
6. Works for both individual developers and team dashboards
