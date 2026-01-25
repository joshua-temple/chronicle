# Standalone UI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement multi-project control center for Chronicle UI with daemon detection and auto-launch.

**Architecture:** Extend existing `pkg/ui/` with standalone mode, add project registry, health checker, and process launcher. React SPA gets new ProjectSelector view.

**Tech Stack:** Go (backend), React/TypeScript (frontend), JSON file storage

---

## Phase 1: Core Infrastructure

### Task 1: Project Registry Types

**Files:**
- Create: `pkg/ui/standalone/registry.go`
- Create: `pkg/ui/standalone/registry_test.go`

**Step 1: Write failing test**

```go
// pkg/ui/standalone/registry_test.go
package standalone

import (
    "os"
    "path/filepath"
    "testing"
)

func TestRegistry_AddProject(t *testing.T) {
    dir := t.TempDir()
    r := NewRegistry(filepath.Join(dir, "projects.json"))

    p := Project{
        Name: "test-project",
        Path: "/path/to/project",
    }

    if err := r.Add(p); err != nil {
        t.Fatalf("Add() error = %v", err)
    }

    projects := r.List()
    if len(projects) != 1 {
        t.Errorf("List() returned %d projects, want 1", len(projects))
    }

    if projects[0].Name != "test-project" {
        t.Errorf("Name = %q, want %q", projects[0].Name, "test-project")
    }

    if projects[0].ID == "" {
        t.Error("ID should be generated")
    }
}

func TestRegistry_Persistence(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "projects.json")

    r1 := NewRegistry(path)
    r1.Add(Project{Name: "persist-test", Path: "/test"})

    // Create new registry instance, should load from file
    r2 := NewRegistry(path)
    projects := r2.List()

    if len(projects) != 1 {
        t.Errorf("Persistence failed: got %d projects, want 1", len(projects))
    }
}
```

**Step 2: Implement registry**

```go
// pkg/ui/standalone/registry.go
package standalone

import (
    "encoding/json"
    "os"
    "sync"
    "time"

    "github.com/google/uuid"
)

type Project struct {
    ID            string            `json:"id"`
    Name          string            `json:"name"`
    Path          string            `json:"path,omitempty"`
    RemoteURL     string            `json:"remoteUrl,omitempty"`
    AddedAt       time.Time         `json:"addedAt"`
    LastOpened    time.Time         `json:"lastOpened,omitempty"`
    LastScenarios []string          `json:"lastScenarios,omitempty"`
    Preferences   map[string]string `json:"preferences,omitempty"`
    AutoDiscovered bool             `json:"autoDiscovered,omitempty"`
}

type Settings struct {
    AutoDiscover        bool `json:"autoDiscover"`
    PollIntervalMs      int  `json:"pollIntervalMs"`
    ActivePollIntervalMs int `json:"activePollIntervalMs"`
}

type registryData struct {
    Version  int       `json:"version"`
    Projects []Project `json:"projects"`
    Settings Settings  `json:"settings"`
}

type Registry struct {
    path string
    mu   sync.RWMutex
    data registryData
}

func NewRegistry(path string) *Registry {
    r := &Registry{
        path: path,
        data: registryData{
            Version:  1,
            Projects: []Project{},
            Settings: Settings{
                AutoDiscover:         true,
                PollIntervalMs:       30000,
                ActivePollIntervalMs: 5000,
            },
        },
    }
    r.load()
    return r
}

func (r *Registry) load() error {
    r.mu.Lock()
    defer r.mu.Unlock()

    data, err := os.ReadFile(r.path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }

    return json.Unmarshal(data, &r.data)
}

func (r *Registry) save() error {
    data, err := json.MarshalIndent(r.data, "", "  ")
    if err != nil {
        return err
    }

    dir := filepath.Dir(r.path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    return os.WriteFile(r.path, data, 0644)
}

func (r *Registry) Add(p Project) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if p.ID == "" {
        p.ID = uuid.New().String()
    }
    if p.AddedAt.IsZero() {
        p.AddedAt = time.Now()
    }

    r.data.Projects = append(r.data.Projects, p)
    return r.save()
}

func (r *Registry) List() []Project {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return append([]Project{}, r.data.Projects...)
}

func (r *Registry) Get(id string) (Project, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    for _, p := range r.data.Projects {
        if p.ID == id {
            return p, true
        }
    }
    return Project{}, false
}

func (r *Registry) Remove(id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    for i, p := range r.data.Projects {
        if p.ID == id {
            r.data.Projects = append(r.data.Projects[:i], r.data.Projects[i+1:]...)
            return r.save()
        }
    }
    return nil
}

func (r *Registry) Update(p Project) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    for i, existing := range r.data.Projects {
        if existing.ID == p.ID {
            r.data.Projects[i] = p
            return r.save()
        }
    }
    return nil
}

func (r *Registry) Settings() Settings {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.data.Settings
}
```

**Step 3: Run tests**

```bash
go test ./pkg/ui/standalone/... -v
```

**Step 4: Commit**

```bash
git add pkg/ui/standalone/
git commit -m "feat(ui): add project registry for standalone mode"
```

---

### Task 2: Daemon Status Types

**Files:**
- Create: `pkg/ui/standalone/status.go`
- Create: `pkg/ui/standalone/status_test.go`

**Step 1: Define status types and health checker**

```go
// pkg/ui/standalone/status.go
package standalone

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

type DaemonState string

const (
    StateUnknown   DaemonState = "unknown"
    StateStopped   DaemonState = "stopped"
    StateStarting  DaemonState = "starting"
    StateRunning   DaemonState = "running"
    StateUnhealthy DaemonState = "unhealthy"
)

type DaemonStatus struct {
    State   DaemonState `json:"state"`
    Port    int         `json:"port,omitempty"`
    Version string      `json:"version,omitempty"`
    Error   string      `json:"error,omitempty"`
}

type HealthResponse struct {
    Status    string `json:"status"`
    Version   string `json:"version"`
    Uptime    string `json:"uptime"`
    Scenarios int    `json:"scenarios"`
}

type HealthChecker struct {
    client  *http.Client
    statuses map[string]DaemonStatus
    mu      sync.RWMutex
}

func NewHealthChecker() *HealthChecker {
    return &HealthChecker{
        client: &http.Client{Timeout: 5 * time.Second},
        statuses: make(map[string]DaemonStatus),
    }
}

func (h *HealthChecker) Check(ctx context.Context, project Project) DaemonStatus {
    var url string
    var port int

    if project.RemoteURL != "" {
        url = project.RemoteURL + "/health"
    } else if project.Path != "" {
        // Try common ports for local projects
        port = h.findRunningPort(ctx, project)
        if port == 0 {
            return DaemonStatus{State: StateStopped}
        }
        url = fmt.Sprintf("http://localhost:%d/health", port)
    } else {
        return DaemonStatus{State: StateUnknown}
    }

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return DaemonStatus{State: StateStopped, Error: err.Error()}
    }

    resp, err := h.client.Do(req)
    if err != nil {
        return DaemonStatus{State: StateStopped}
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return DaemonStatus{State: StateUnhealthy, Port: port}
    }

    var health HealthResponse
    if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
        return DaemonStatus{State: StateUnhealthy, Port: port, Error: "invalid response"}
    }

    return DaemonStatus{
        State:   StateRunning,
        Port:    port,
        Version: health.Version,
    }
}

func (h *HealthChecker) findRunningPort(ctx context.Context, project Project) int {
    // Try common Chronicle ports
    ports := []int{8080, 3000, 8081, 8082}

    for _, port := range ports {
        url := fmt.Sprintf("http://localhost:%d/health", port)
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        resp, err := h.client.Do(req)
        if err == nil {
            resp.Body.Close()
            if resp.StatusCode == http.StatusOK {
                return port
            }
        }
    }
    return 0
}

func (h *HealthChecker) GetStatus(projectID string) DaemonStatus {
    h.mu.RLock()
    defer h.mu.RUnlock()
    if s, ok := h.statuses[projectID]; ok {
        return s
    }
    return DaemonStatus{State: StateUnknown}
}

func (h *HealthChecker) SetStatus(projectID string, status DaemonStatus) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.statuses[projectID] = status
}
```

**Step 2: Write tests**

```go
// pkg/ui/standalone/status_test.go
package standalone

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthChecker_Check_Running(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"healthy","version":"0.1.0"}`))
    }))
    defer srv.Close()

    checker := NewHealthChecker()
    project := Project{
        ID:        "test",
        RemoteURL: srv.URL,
    }

    status := checker.Check(context.Background(), project)

    if status.State != StateRunning {
        t.Errorf("State = %v, want %v", status.State, StateRunning)
    }
    if status.Version != "0.1.0" {
        t.Errorf("Version = %q, want %q", status.Version, "0.1.0")
    }
}

func TestHealthChecker_Check_Stopped(t *testing.T) {
    checker := NewHealthChecker()
    project := Project{
        ID:        "test",
        RemoteURL: "http://localhost:59999", // unlikely to be running
    }

    status := checker.Check(context.Background(), project)

    if status.State != StateStopped {
        t.Errorf("State = %v, want %v", status.State, StateStopped)
    }
}
```

**Step 3: Run tests and commit**

```bash
go test ./pkg/ui/standalone/... -v
git add pkg/ui/standalone/
git commit -m "feat(ui): add daemon health checker for standalone mode"
```

---

### Task 3: Process Launcher

**Files:**
- Create: `pkg/ui/standalone/launcher.go`
- Create: `pkg/ui/standalone/launcher_test.go`

**Step 1: Implement launcher**

```go
// pkg/ui/standalone/launcher.go
package standalone

import (
    "context"
    "fmt"
    "net"
    "os"
    "os/exec"
    "sync"
    "time"
)

type Launcher struct {
    processes map[string]*exec.Cmd
    ports     map[string]int
    mu        sync.RWMutex
    checker   *HealthChecker
}

func NewLauncher(checker *HealthChecker) *Launcher {
    return &Launcher{
        processes: make(map[string]*exec.Cmd),
        ports:     make(map[string]int),
        checker:   checker,
    }
}

func (l *Launcher) Launch(ctx context.Context, project Project) (int, error) {
    l.mu.Lock()

    if _, running := l.processes[project.ID]; running {
        l.mu.Unlock()
        return l.ports[project.ID], nil
    }

    port, err := l.findAvailablePort()
    if err != nil {
        l.mu.Unlock()
        return 0, fmt.Errorf("no available port: %w", err)
    }

    cmd := exec.CommandContext(ctx, "chronicle", "daemon", "--port", fmt.Sprintf("%d", port))
    cmd.Dir = project.Path
    cmd.Stdout = os.Stdout // TODO: capture to buffer
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        l.mu.Unlock()
        return 0, fmt.Errorf("failed to start daemon: %w", err)
    }

    l.processes[project.ID] = cmd
    l.ports[project.ID] = port
    l.mu.Unlock()

    // Wait for daemon to be healthy
    if err := l.waitForHealth(ctx, project, port, 30*time.Second); err != nil {
        l.Stop(ctx, project.ID)
        return 0, err
    }

    return port, nil
}

func (l *Launcher) Stop(ctx context.Context, projectID string) error {
    l.mu.Lock()
    defer l.mu.Unlock()

    cmd, ok := l.processes[projectID]
    if !ok {
        return nil
    }

    // Graceful shutdown
    if cmd.Process != nil {
        cmd.Process.Signal(os.Interrupt)

        done := make(chan error, 1)
        go func() { done <- cmd.Wait() }()

        select {
        case <-time.After(10 * time.Second):
            cmd.Process.Kill()
        case <-done:
        }
    }

    delete(l.processes, projectID)
    delete(l.ports, projectID)
    return nil
}

func (l *Launcher) IsRunning(projectID string) bool {
    l.mu.RLock()
    defer l.mu.RUnlock()
    _, ok := l.processes[projectID]
    return ok
}

func (l *Launcher) GetPort(projectID string) int {
    l.mu.RLock()
    defer l.mu.RUnlock()
    return l.ports[projectID]
}

func (l *Launcher) findAvailablePort() (int, error) {
    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        return 0, err
    }
    defer listener.Close()
    return listener.Addr().(*net.TCPAddr).Port, nil
}

func (l *Launcher) waitForHealth(ctx context.Context, project Project, port int, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    tempProject := project
    tempProject.RemoteURL = fmt.Sprintf("http://localhost:%d", port)

    for time.Now().Before(deadline) {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            status := l.checker.Check(ctx, tempProject)
            if status.State == StateRunning {
                return nil
            }
        }
    }

    return fmt.Errorf("daemon did not become healthy within %v", timeout)
}
```

**Step 2: Write tests and commit**

```bash
go test ./pkg/ui/standalone/... -v
git add pkg/ui/standalone/
git commit -m "feat(ui): add process launcher for standalone mode"
```

---

### Task 4: Standalone Server

**Files:**
- Create: `pkg/ui/standalone/server.go`
- Modify: `pkg/ui/server.go` (add standalone mode detection)

**Step 1: Create standalone server with API endpoints**

```go
// pkg/ui/standalone/server.go
package standalone

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io/fs"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/joshua-temple/chronicle/web"
)

type Server struct {
    port     int
    registry *Registry
    checker  *HealthChecker
    launcher *Launcher
    mux      *http.ServeMux
    server   *http.Server
    webFS    fs.FS
}

type ServerOption func(*Server)

func WithPort(port int) ServerOption {
    return func(s *Server) { s.port = port }
}

func NewServer(opts ...ServerOption) *Server {
    home, _ := os.UserHomeDir()
    registryPath := filepath.Join(home, ".chronicle", "projects.json")

    checker := NewHealthChecker()

    s := &Server{
        port:     3000,
        registry: NewRegistry(registryPath),
        checker:  checker,
        launcher: NewLauncher(checker),
        mux:      http.NewServeMux(),
    }

    for _, opt := range opts {
        opt(s)
    }

    s.initWebFS()
    s.setupRoutes()
    return s
}

func (s *Server) initWebFS() {
    subFS, err := fs.Sub(web.WebFS, "dist")
    if err != nil {
        return
    }
    s.webFS = subFS
}

func (s *Server) setupRoutes() {
    // Standalone API
    s.mux.HandleFunc("GET /api/standalone/mode", s.handleMode)
    s.mux.HandleFunc("GET /api/standalone/projects", s.handleListProjects)
    s.mux.HandleFunc("POST /api/standalone/projects", s.handleAddProject)
    s.mux.HandleFunc("DELETE /api/standalone/projects/{id}", s.handleRemoveProject)
    s.mux.HandleFunc("PUT /api/standalone/projects/{id}", s.handleUpdateProject)
    s.mux.HandleFunc("POST /api/standalone/projects/{id}/launch", s.handleLaunch)
    s.mux.HandleFunc("POST /api/standalone/projects/{id}/stop", s.handleStop)
    s.mux.HandleFunc("GET /api/standalone/projects/{id}/health", s.handleHealth)
    s.mux.HandleFunc("POST /api/standalone/discover", s.handleDiscover)

    // SPA
    if s.webFS != nil {
        s.mux.Handle("GET /", s.spaHandler())
    }
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]string{"mode": "standalone"})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
    projects := s.registry.List()
    ctx := r.Context()

    // Enrich with status
    type ProjectWithStatus struct {
        Project
        Status DaemonStatus `json:"status"`
    }

    result := make([]ProjectWithStatus, len(projects))
    for i, p := range projects {
        result[i] = ProjectWithStatus{
            Project: p,
            Status:  s.checker.Check(ctx, p),
        }
    }

    writeJSON(w, http.StatusOK, map[string]any{"projects": result})
}

func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
    var p Project
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }

    if p.Path == "" && p.RemoteURL == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path or remoteUrl required"})
        return
    }

    if err := s.registry.Add(p); err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }

    writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleRemoveProject(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if err := s.registry.Remove(id); err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var p Project
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }
    p.ID = id
    if err := s.registry.Update(p); err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleLaunch(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    project, ok := s.registry.Get(id)
    if !ok {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
        return
    }

    if project.RemoteURL != "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot launch remote project"})
        return
    }

    port, err := s.launcher.Launch(r.Context(), project)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }

    writeJSON(w, http.StatusOK, map[string]any{"success": true, "port": port})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if err := s.launcher.Stop(r.Context(), id); err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    project, ok := s.registry.Get(id)
    if !ok {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
        return
    }

    status := s.checker.Check(r.Context(), project)
    writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
    discovered := s.discoverProjects()
    writeJSON(w, http.StatusOK, map[string]any{"discovered": discovered})
}

func (s *Server) discoverProjects() []Project {
    var discovered []Project
    home, _ := os.UserHomeDir()

    // Check common directories
    searchPaths := []string{
        filepath.Join(home, "code"),
        filepath.Join(home, "projects"),
        filepath.Join(home, "src"),
        filepath.Join(home, "go", "src"),
    }

    existing := make(map[string]bool)
    for _, p := range s.registry.List() {
        existing[p.Path] = true
    }

    for _, basePath := range searchPaths {
        filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
            if err != nil || info.IsDir() {
                return nil
            }
            if info.Name() == "chronicle.yaml" || info.Name() == "chronicle.yml" {
                dir := filepath.Dir(path)
                if !existing[dir] {
                    discovered = append(discovered, Project{
                        Name:           filepath.Base(dir),
                        Path:           dir,
                        AutoDiscovered: true,
                    })
                    existing[dir] = true
                }
            }
            return nil
        })
    }

    return discovered
}

func (s *Server) spaHandler() http.Handler {
    fileServer := http.FileServer(http.FS(s.webFS))

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/")
        if path == "" {
            path = "index.html"
        }

        _, err := fs.Stat(s.webFS, path)
        if err != nil {
            r.URL.Path = "/"
        }
        fileServer.ServeHTTP(w, r)
    })
}

func (s *Server) Start(ctx context.Context) error {
    s.server = &http.Server{
        Addr:    fmt.Sprintf(":%d", s.port),
        Handler: s.mux,
    }

    errCh := make(chan error, 1)
    go func() {
        if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            errCh <- err
        }
        close(errCh)
    }()

    select {
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        return s.server.Shutdown(shutdownCtx)
    case err := <-errCh:
        return err
    }
}

func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}
```

**Step 2: Run tests and commit**

```bash
go test ./pkg/ui/standalone/... -v
git add pkg/ui/standalone/
git commit -m "feat(ui): add standalone server with project management API"
```

---

### Task 5: CLI Integration

**Files:**
- Modify: `pkg/cli/ui.go` (add --standalone flag)

**Step 1: Add standalone flag to ui command**

Add `--standalone` flag that creates `standalone.Server` instead of regular `ui.Server`.

**Step 2: Commit**

```bash
git add pkg/cli/ui.go
git commit -m "feat(cli): add --standalone flag to ui command"
```

---

## Phase 2: React UI

### Task 6: Mode Detection Hook

**Files:**
- Create: `web/src/hooks/useMode.ts`

**Step 1: Implement mode detection**

```typescript
// web/src/hooks/useMode.ts
import { useState, useEffect } from 'react';

export type AppMode = 'single' | 'standalone' | 'loading';

export function useMode(): AppMode {
  const [mode, setMode] = useState<AppMode>('loading');

  useEffect(() => {
    fetch('/api/standalone/mode')
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (data?.mode === 'standalone') {
          setMode('standalone');
        } else {
          setMode('single');
        }
      })
      .catch(() => setMode('single'));
  }, []);

  return mode;
}
```

**Step 2: Commit**

```bash
git add web/src/hooks/useMode.ts
git commit -m "feat(web): add mode detection hook for standalone UI"
```

---

### Task 7: Projects Store

**Files:**
- Create: `web/src/stores/projects.ts`

**Step 1: Implement Zustand store**

```typescript
// web/src/stores/projects.ts
import { create } from 'zustand';

export interface Project {
  id: string;
  name: string;
  path?: string;
  remoteUrl?: string;
  addedAt: string;
  lastOpened?: string;
  lastScenarios?: string[];
  preferences?: Record<string, string>;
  autoDiscovered?: boolean;
  status: {
    state: 'unknown' | 'stopped' | 'starting' | 'running' | 'unhealthy';
    port?: number;
    version?: string;
    error?: string;
  };
}

interface ProjectsState {
  projects: Project[];
  discovered: Project[];
  loading: boolean;
  error: string | null;
  activeProjectId: string | null;

  fetchProjects: () => Promise<void>;
  addProject: (project: Partial<Project>) => Promise<void>;
  removeProject: (id: string) => Promise<void>;
  launchProject: (id: string) => Promise<void>;
  stopProject: (id: string) => Promise<void>;
  setActiveProject: (id: string | null) => void;
  discover: () => Promise<void>;
}

export const useProjectsStore = create<ProjectsState>((set, get) => ({
  projects: [],
  discovered: [],
  loading: false,
  error: null,
  activeProjectId: null,

  fetchProjects: async () => {
    set({ loading: true, error: null });
    try {
      const res = await fetch('/api/standalone/projects');
      const data = await res.json();
      set({ projects: data.projects, loading: false });
    } catch (err) {
      set({ error: 'Failed to fetch projects', loading: false });
    }
  },

  addProject: async (project) => {
    try {
      const res = await fetch('/api/standalone/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(project),
      });
      if (res.ok) {
        get().fetchProjects();
      }
    } catch (err) {
      set({ error: 'Failed to add project' });
    }
  },

  removeProject: async (id) => {
    try {
      await fetch(`/api/standalone/projects/${id}`, { method: 'DELETE' });
      get().fetchProjects();
    } catch (err) {
      set({ error: 'Failed to remove project' });
    }
  },

  launchProject: async (id) => {
    try {
      const res = await fetch(`/api/standalone/projects/${id}/launch`, {
        method: 'POST',
      });
      if (res.ok) {
        get().fetchProjects();
      }
    } catch (err) {
      set({ error: 'Failed to launch project' });
    }
  },

  stopProject: async (id) => {
    try {
      await fetch(`/api/standalone/projects/${id}/stop`, { method: 'POST' });
      get().fetchProjects();
    } catch (err) {
      set({ error: 'Failed to stop project' });
    }
  },

  setActiveProject: (id) => set({ activeProjectId: id }),

  discover: async () => {
    try {
      const res = await fetch('/api/standalone/discover', { method: 'POST' });
      const data = await res.json();
      set({ discovered: data.discovered });
    } catch (err) {
      set({ error: 'Failed to discover projects' });
    }
  },
}));
```

**Step 2: Commit**

```bash
git add web/src/stores/projects.ts
git commit -m "feat(web): add projects store for standalone mode"
```

---

### Task 8: ProjectSelector Component

**Files:**
- Create: `web/src/components/standalone/ProjectSelector.tsx`
- Create: `web/src/components/standalone/ProjectCard.tsx`
- Create: `web/src/components/standalone/AddProjectModal.tsx`

**Step 1: Implement components**

(Full component implementations with proper styling, status indicators, launch/stop buttons)

**Step 2: Commit**

```bash
git add web/src/components/standalone/
git commit -m "feat(web): add ProjectSelector components for standalone mode"
```

---

### Task 9: App Integration

**Files:**
- Modify: `web/src/App.tsx`

**Step 1: Add mode-based routing**

```typescript
// In App.tsx, detect mode and render ProjectSelector or existing UI
const mode = useMode();

if (mode === 'loading') return <Loading />;
if (mode === 'standalone' && !activeProjectId) return <ProjectSelector />;
return <ExistingApp />;
```

**Step 2: Commit**

```bash
git add web/src/App.tsx
git commit -m "feat(web): integrate standalone mode with app routing"
```

---

## Phase 3: Polish

### Task 10: Status Polling

**Files:**
- Modify: `web/src/stores/projects.ts`

Add periodic polling for project status updates.

### Task 11: Error Handling

Add proper error states and recovery for:
- Network failures
- Daemon launch failures
- Invalid project paths

### Task 12: Final Testing

- Run all Go tests
- Run all TypeScript tests
- Manual testing of full flow
- Lint and format

---

## Commit Summary

After completing all tasks:

```bash
git log --oneline -15
```

Expected commits:
1. feat(ui): add project registry for standalone mode
2. feat(ui): add daemon health checker for standalone mode
3. feat(ui): add process launcher for standalone mode
4. feat(ui): add standalone server with project management API
5. feat(cli): add --standalone flag to ui command
6. feat(web): add mode detection hook for standalone UI
7. feat(web): add projects store for standalone mode
8. feat(web): add ProjectSelector components for standalone mode
9. feat(web): integrate standalone mode with app routing
10. feat(web): add status polling for projects
11. fix(ui): improve error handling in standalone mode
12. test(ui): add comprehensive tests for standalone mode
