# Chronicle Standalone UI Design

> Local web interface for editing Chronicle configuration and building scenarios.

## Overview

The `chronicle ui` command launches a local HTTP server that serves the React SPA with file system APIs for editing `chronicle.yaml` and building scenarios visually.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  chronicle ui                        │
├─────────────────────────────────────────────────────┤
│  /api/local/config     → Read/write chronicle.yaml  │
│  /api/local/discover   → Run component discovery    │
│  /api/local/validate   → Validate configuration     │
│  /api/local/project    → Project info               │
│  /*                    → Static files (React SPA)   │
└─────────────────────────────────────────────────────┘
```

### Modes

The React app operates in two modes:

| Mode | Command | Features |
|------|---------|----------|
| **Standalone** | `chronicle ui` | Config editor, scenario builder, component browser |
| **Daemon** | `chronicle daemon` | Dashboard, runs, results, monitoring |

Mode is detected at startup by checking which API endpoints respond.

## CLI Command

```bash
chronicle ui [flags]

Flags:
  --port int       Port to serve on (default 3000)
  --dir string     Project directory (default ".")
  --no-browser     Don't open browser automatically
```

## Local API Endpoints

### Configuration

**GET /api/local/config**

Returns the parsed `chronicle.yaml` as JSON.

```json
{
  "version": "1",
  "scenarios": [...],
  "infrastructure": {...},
  "chaos": {...},
  "mocks": {...}
}
```

**PUT /api/local/config**

Accepts JSON, writes back to `chronicle.yaml`. Validates before saving.

Request body: Same structure as GET response.

Response: `200 OK` or `400 Bad Request` with validation errors.

**POST /api/local/config/validate**

Validates config without saving.

Request body: Same structure as config.

Response:
```json
{
  "valid": true,
  "errors": [],
  "warnings": ["scenario 'test' has no teardown"]
}
```

### Discovery

**POST /api/local/discover**

Runs component discovery on the project directory.

Response:
```json
{
  "components": [
    {
      "name": "CreateUser",
      "type": "setup",
      "description": "Creates a test user",
      "tags": ["user", "setup"],
      "produces": ["user"],
      "requires": [],
      "source_file": "components/user.go"
    }
  ],
  "discovered_at": "2026-01-25T10:30:00Z"
}
```

**GET /api/local/components**

Returns cached discovery results from last scan.

### Project

**GET /api/local/project**

Returns project information.

```json
{
  "directory": "/path/to/project",
  "config_file": "chronicle.yaml",
  "config_exists": true,
  "last_modified": "2026-01-25T10:00:00Z"
}
```

## Frontend Pages

### Config Editor (`/config`)

Tabbed interface for editing `chronicle.yaml`:

- **General**: Version, global settings
- **Scenarios**: List of scenarios with inline editing
- **Infrastructure**: Provider configuration
- **Chaos**: Chaos profile definitions
- **Mocks**: Mock profile definitions

Features:
- Form-based editing (no raw YAML)
- Real-time validation
- Save with confirmation
- "View YAML" preview toggle

### Scenario Builder (`/scenarios/:name/edit`)

Visual editor for individual scenarios:

- **Metadata**: Name, description, tags, timeout, parallel count
- **Flow Editor**: Ordered list of steps
  - Add step from component picker
  - Remove/reorder steps
  - Configure step options (timeout, condition)
- **Flow Preview**: Read-only graph visualization

The component picker shows discovered components filtered by type.

### Components Browser (`/components`)

Reuses existing Components page with additions:
- "Refresh" button to re-run discovery
- Shows discovery timestamp
- Loading state during scan

## Mode Detection

On app load:

```typescript
async function detectMode(): Promise<'standalone' | 'daemon' | 'disconnected'> {
  try {
    await fetch('/api/local/project')
    return 'standalone'
  } catch {
    try {
      await fetch('/api/v1/health')
      return 'daemon'
    } catch {
      return 'disconnected'
    }
  }
}
```

The app renders different navigation and routes based on mode.

## Implementation

### Go Backend

| File | Purpose |
|------|---------|
| `pkg/cli/ui.go` | CLI command definition |
| `pkg/ui/server.go` | HTTP server setup |
| `pkg/ui/handlers.go` | API endpoint handlers |
| `pkg/ui/config.go` | Config read/write with YAML preservation |

### Frontend

| File | Purpose |
|------|---------|
| `web/src/api/local.ts` | Local API client functions |
| `web/src/hooks/useLocalConfig.ts` | Config state with React Query |
| `web/src/hooks/useLocalDiscover.ts` | Discovery state |
| `web/src/stores/mode.ts` | Mode detection Zustand store |
| `web/src/pages/ConfigEditor.tsx` | Main config editor page |
| `web/src/pages/ScenarioEditor.tsx` | Scenario builder page |
| `web/src/components/config/*.tsx` | Config section editors |
| `web/src/components/scenarios/FlowEditor.tsx` | Flow step list editor |
| `web/src/components/scenarios/FlowPreview.tsx` | Graph visualization |

## Future Enhancements

- **Visual drag-and-drop builder**: Full canvas-based scenario editor
- **Multi-file support**: Edit scenarios in separate YAML files
- **Live preview**: See changes reflected in real-time
- **Undo/redo**: Edit history with keyboard shortcuts
