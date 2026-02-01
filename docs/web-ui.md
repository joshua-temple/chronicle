# Web UI

Chronicle includes a web dashboard for visualizing test runs, monitoring progress, and exploring results.

## Accessing the Dashboard

The web UI is served by the daemon:

```bash
# Start daemon (includes web UI)
chronicle daemon

# Open in browser
open http://localhost:3000
```

Or start with a custom port:

```bash
chronicle daemon --addr :8080
# Access at http://localhost:8080
```

## Features

### Dashboard Overview

The main dashboard shows:

- **Active Runs** - Currently executing scenarios with real-time progress
- **Recent Results** - Latest test runs with pass/fail status
- **Quick Stats** - Total runs, pass rate, average duration
- **System Health** - Daemon status and infrastructure health

### Project Navigation

The sidebar provides hierarchical navigation:

```
Projects
├── my-project
│   ├── Suites
│   │   ├── smoke
│   │   └── regression
│   ├── Scenarios
│   │   ├── checkout_flow
│   │   └── login_test
│   └── Components
│       ├── Setup
│       ├── Tasks
│       └── Validations
```

### Real-Time Monitoring

Watch test execution in real-time:

- **Live Progress** - See each component as it executes
- **Duration Tracking** - Elapsed time for the run and individual components
- **Status Updates** - Instant feedback on pass/fail/skip
- **Error Details** - View errors as they occur

### Run Details

Click on any run to see:

- **Summary** - Total passed, failed, skipped
- **Timeline** - Visual flow of component execution
- **Logs** - Output from each component
- **Errors** - Stack traces and error messages
- **Duration Breakdown** - Time spent in each phase

### Scenario Browser

Explore available scenarios:

- **List View** - All scenarios with tags and descriptions
- **Dependency Graph** - Visual representation of component dependencies
- **Flow Preview** - See the execution order before running
- **Quick Run** - Start a scenario directly from the UI

### Component Explorer

Browse discovered components:

- **By Type** - Filter by setup, task, validation, etc.
- **By Tag** - Filter by component tags
- **Dependency View** - See what each component produces/requires
- **Source Link** - Jump to source code location

### Results History

Query historical results:

- **Search** - Filter by scenario, tags, date range
- **Compare** - Compare results across runs
- **Trends** - View pass/fail trends over time
- **Export** - Download results as JSON, JUnit, or HTML

## Running Tests from UI

### Start a Single Scenario

1. Navigate to **Scenarios**
2. Click on a scenario name
3. Click **Run** button
4. Optionally configure:
   - Runtime flags
   - Timeout
   - Chaos/mock profiles
5. Monitor progress in real-time

### Start a Suite

1. Navigate to **Suites**
2. Click on a suite name
3. Click **Run Suite**
4. Configure parallel execution
5. Watch all scenarios execute

### Ad-hoc Runs

1. Click **New Run** in the header
2. Select scenarios or tags
3. Configure options
4. Click **Start**

## Configuration

### Daemon Options

The web UI is automatically enabled. Configure through daemon flags:

```bash
# Change port
chronicle daemon --addr :8080

# Enable API authentication (also secures UI API calls)
chronicle daemon --api-key my-secret-key
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CHRONICLE_DAEMON_ADDR` | Daemon address |
| `CHRONICLE_API_KEY` | API key for authentication |

## Development Mode

During development, the UI connects to a local daemon:

```bash
# Terminal 1: Start daemon
chronicle daemon --watch

# Terminal 2: Start web dev server (if developing UI)
cd web && npm run dev
```

The `--watch` flag enables hot reload of configuration changes.

## API Integration

The web UI uses the same REST API documented in [Daemon API](daemon.md). You can:

- Build custom dashboards using the API
- Integrate with monitoring tools
- Create custom automation workflows

### WebSocket/SSE Events

The UI subscribes to Server-Sent Events for real-time updates:

```javascript
// The UI internally uses this pattern
const events = new EventSource('/api/v1/events');

events.addEventListener('run.started', handleRunStarted);
events.addEventListener('component.completed', handleComponentCompleted);
events.addEventListener('run.completed', handleRunCompleted);
```

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `r` | Refresh current view |
| `/` | Focus search |
| `n` | New run |
| `Esc` | Close modal |

## Best Practices

1. **Use During Development** - Real-time feedback speeds up test development
2. **Monitor CI Runs** - Point the UI at your CI daemon for visibility
3. **Explore Dependencies** - Use the graph view to understand component relationships
4. **Track Trends** - Review historical data to catch flaky tests
5. **Share Results** - Export and share reports with your team

## Troubleshooting

### UI Not Loading

1. Check daemon is running: `curl http://localhost:3000/api/v1/health`
2. Check for port conflicts
3. Verify no firewall blocking

### Stale Data

1. Check SSE connection in browser dev tools
2. Refresh the page
3. Restart daemon if events stopped

### Authentication Issues

1. Verify API key matches daemon configuration
2. Check browser console for 401 errors
3. Clear browser storage and re-authenticate
