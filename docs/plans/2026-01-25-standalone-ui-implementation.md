# Standalone UI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `chronicle ui` command that serves a local web interface for editing configuration and building scenarios.

**Architecture:** New `pkg/ui` package provides HTTP server with local file system APIs. Frontend detects mode (standalone vs daemon) and renders appropriate UI. Config editing uses form-based interface with YAML read/write on backend.

**Tech Stack:** Go HTTP server, existing React/TypeScript frontend, TanStack Query, Zustand, gopkg.in/yaml.v3

---

### Task 1: Create UI Server Package

**Files:**
- Create: `pkg/ui/server.go`
- Create: `pkg/ui/server_test.go`

**Step 1: Create server.go with basic structure**

```go
package ui

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/joshua-temple/chronicle/web"
)

// Server serves the Chronicle UI in standalone mode.
type Server struct {
	port    int
	dir     string
	server  *http.Server
	mux     *http.ServeMux
}

// Option configures the server.
type Option func(*Server)

// WithPort sets the server port.
func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

// WithDir sets the project directory.
func WithDir(dir string) Option {
	return func(s *Server) {
		s.dir = dir
	}
}

// New creates a new UI server.
func New(opts ...Option) *Server {
	s := &Server{
		port: 3000,
		dir:  ".",
		mux:  http.NewServeMux(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start starts the server.
func (s *Server) Start(ctx context.Context) error {
	s.setupRoutes()

	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	return s.server.Serve(ln)
}

// Port returns the configured port.
func (s *Server) Port() int {
	return s.port
}

// Dir returns the project directory.
func (s *Server) Dir() string {
	return s.dir
}

func (s *Server) setupRoutes() {
	// Local API endpoints
	s.mux.HandleFunc("GET /api/local/project", s.handleProject)
	s.mux.HandleFunc("GET /api/local/config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /api/local/config", s.handlePutConfig)
	s.mux.HandleFunc("POST /api/local/config/validate", s.handleValidateConfig)
	s.mux.HandleFunc("POST /api/local/discover", s.handleDiscover)
	s.mux.HandleFunc("GET /api/local/components", s.handleGetComponents)

	// Static files
	subFS, err := fs.Sub(web.WebFS, "dist")
	if err == nil {
		s.mux.Handle("GET /", spaHandler(subFS))
	}
}

func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:] // Remove leading slash
		}
		_, err := fs.Stat(fsys, path)
		if err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// Placeholder handlers - implemented in handlers.go
func (s *Server) handleProject(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request)      {}
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request)      {}
func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {}
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request)       {}
func (s *Server) handleGetComponents(w http.ResponseWriter, r *http.Request)  {}
```

**Step 2: Create basic test**

```go
package ui

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New(WithPort(3001), WithDir("/tmp"))
	if s.Port() != 3001 {
		t.Errorf("expected port 3001, got %d", s.Port())
	}
	if s.Dir() != "/tmp" {
		t.Errorf("expected dir /tmp, got %s", s.Dir())
	}
}

func TestServer_Start(t *testing.T) {
	s := New(WithPort(0)) // Use port 0 for random available port
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel to trigger shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("server did not shut down in time")
	}
}
```

**Step 3: Run tests**

Run: `go test ./pkg/ui/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/ui/
git commit -m "feat(ui): add standalone UI server package"
```

---

### Task 2: Implement Project Handler

**Files:**
- Modify: `pkg/ui/server.go`
- Create: `pkg/ui/handlers.go`
- Modify: `pkg/ui/server_test.go`

**Step 1: Create handlers.go with project handler**

```go
package ui

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ProjectInfo contains information about the Chronicle project.
type ProjectInfo struct {
	Directory    string     `json:"directory"`
	ConfigFile   string     `json:"config_file"`
	ConfigExists bool       `json:"config_exists"`
	LastModified *time.Time `json:"last_modified,omitempty"`
}

func (s *Server) handleProject(w http.ResponseWriter, _ *http.Request) {
	absDir, err := filepath.Abs(s.dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve directory", err)
		return
	}

	configPath := filepath.Join(absDir, "chronicle.yaml")
	info := ProjectInfo{
		Directory:    absDir,
		ConfigFile:   "chronicle.yaml",
		ConfigExists: false,
	}

	if stat, err := os.Stat(configPath); err == nil {
		info.ConfigExists = true
		modTime := stat.ModTime()
		info.LastModified = &modTime
	}

	writeJSON(w, http.StatusOK, info)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]string{
		"error":   message,
		"details": "",
	}
	if err != nil {
		resp["details"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}
```

**Step 2: Remove placeholder from server.go**

Remove the placeholder `handleProject` method from server.go (it's now in handlers.go).

**Step 3: Add test for project handler**

```go
func TestServer_HandleProject(t *testing.T) {
	// Create temp dir with config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "chronicle.yaml")
	if err := os.WriteFile(configPath, []byte("version: \"1\""), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))
	s.setupRoutes()

	req := httptest.NewRequest("GET", "/api/local/project", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var info ProjectInfo
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}

	if !info.ConfigExists {
		t.Error("expected config to exist")
	}
	if info.LastModified == nil {
		t.Error("expected last modified time")
	}
}
```

**Step 4: Run tests**

Run: `go test ./pkg/ui/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/ui/
git commit -m "feat(ui): implement project info handler"
```

---

### Task 3: Implement Config Handlers

**Files:**
- Modify: `pkg/ui/handlers.go`
- Modify: `pkg/ui/server_test.go`

**Step 1: Add config read/write handlers**

Add to handlers.go:

```go
import (
	"io"
	"github.com/joshua-temple/chronicle/pkg/config"
	"gopkg.in/yaml.v3"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	configPath := filepath.Join(s.dir, "chronicle.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "config file not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read config", err)
		return
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse config", err)
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body", err)
		return
	}

	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", err)
		return
	}

	// Validate before saving
	if err := cfg.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation failed", err)
		return
	}

	yamlData, err := yaml.Marshal(&cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal YAML", err)
		return
	}

	configPath := filepath.Join(s.dir, "chronicle.yaml")
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write config", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// ValidationResult holds the result of config validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body", err)
		return
	}

	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeJSON(w, http.StatusOK, ValidationResult{
			Valid:  false,
			Errors: []string{"Invalid JSON: " + err.Error()},
		})
		return
	}

	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if err := cfg.Validate(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	writeJSON(w, http.StatusOK, result)
}
```

**Step 2: Add tests for config handlers**

```go
func TestServer_HandleGetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `version: "1"
scenarios:
  - name: test
    flow:
      - component: Setup
`
	if err := os.WriteFile(filepath.Join(tmpDir, "chronicle.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))
	s.setupRoutes()

	req := httptest.NewRequest("GET", "/api/local/config", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_HandlePutConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Create initial config
	if err := os.WriteFile(filepath.Join(tmpDir, "chronicle.yaml"), []byte("version: \"1\""), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))
	s.setupRoutes()

	body := `{"version": "1", "scenarios": []}`
	req := httptest.NewRequest("PUT", "/api/local/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_HandleValidateConfig(t *testing.T) {
	s := New()
	s.setupRoutes()

	body := `{"version": "1"}`
	req := httptest.NewRequest("POST", "/api/local/config/validate", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result ValidationResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	if !result.Valid {
		t.Errorf("expected valid config, got errors: %v", result.Errors)
	}
}
```

**Step 3: Run tests**

Run: `go test ./pkg/ui/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/ui/
git commit -m "feat(ui): implement config read/write/validate handlers"
```

---

### Task 4: Implement Discovery Handlers

**Files:**
- Modify: `pkg/ui/handlers.go`
- Modify: `pkg/ui/server.go` (add cached components)
- Modify: `pkg/ui/server_test.go`

**Step 1: Add discovery handlers**

Add to server.go:

```go
import "sync"

type Server struct {
	// ... existing fields
	componentsMu sync.RWMutex
	components   []DiscoveredComponent
	discoveredAt time.Time
}
```

Add to handlers.go:

```go
import (
	"github.com/joshua-temple/chronicle/pkg/discovery"
)

// DiscoveredComponent represents a discovered component.
type DiscoveredComponent struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Produces    []string `json:"produces"`
	Requires    []string `json:"requires"`
	SourceFile  string   `json:"source_file"`
}

// DiscoveryResult contains discovery results.
type DiscoveryResult struct {
	Components   []DiscoveredComponent `json:"components"`
	DiscoveredAt time.Time             `json:"discovered_at"`
}

func (s *Server) handleDiscover(w http.ResponseWriter, _ *http.Request) {
	discovered, err := discovery.Discover(s.dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "discovery failed", err)
		return
	}

	components := make([]DiscoveredComponent, 0, len(discovered))
	for _, c := range discovered {
		components = append(components, DiscoveredComponent{
			Name:        c.Name,
			Type:        string(c.Type),
			Description: c.Description,
			Tags:        c.Tags,
			Produces:    c.Produces,
			Requires:    c.Requires,
			SourceFile:  c.SourceFile,
		})
	}

	now := time.Now()
	s.componentsMu.Lock()
	s.components = components
	s.discoveredAt = now
	s.componentsMu.Unlock()

	writeJSON(w, http.StatusOK, DiscoveryResult{
		Components:   components,
		DiscoveredAt: now,
	})
}

func (s *Server) handleGetComponents(w http.ResponseWriter, _ *http.Request) {
	s.componentsMu.RLock()
	components := s.components
	discoveredAt := s.discoveredAt
	s.componentsMu.RUnlock()

	if components == nil {
		writeJSON(w, http.StatusOK, DiscoveryResult{
			Components:   []DiscoveredComponent{},
			DiscoveredAt: time.Time{},
		})
		return
	}

	writeJSON(w, http.StatusOK, DiscoveryResult{
		Components:   components,
		DiscoveredAt: discoveredAt,
	})
}
```

**Step 2: Run tests**

Run: `go test ./pkg/ui/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/ui/
git commit -m "feat(ui): implement component discovery handlers"
```

---

### Task 5: Create CLI Command

**Files:**
- Create: `pkg/cli/ui.go`
- Modify: `cmd/chronicle/main.go`

**Step 1: Create ui.go**

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/joshua-temple/chronicle/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUICmd() *cobra.Command {
	var port int
	var dir string
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the Chronicle UI for editing configuration",
		Long: `Launch a local web server that serves the Chronicle UI.

The UI allows you to:
- Edit chronicle.yaml configuration
- Build and modify scenarios
- Browse discovered components

Example:
  chronicle ui
  chronicle ui --port 8080
  chronicle ui --dir ./my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUI(port, dir, noBrowser)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to serve on")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Project directory")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")

	return cmd
}

func runUI(port int, dir string, noBrowser bool) error {
	server := ui.New(
		ui.WithPort(port),
		ui.WithDir(dir),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Chronicle UI available at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	// Open browser
	if !noBrowser {
		go openBrowser(url)
	}

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
```

**Step 2: Register command in main.go**

Add to the root command setup:

```go
rootCmd.AddCommand(cli.NewUICmd())
```

**Step 3: Build and test manually**

Run: `go build ./cmd/chronicle && ./chronicle ui --help`
Expected: Shows help text for ui command

**Step 4: Commit**

```bash
git add pkg/cli/ui.go cmd/chronicle/
git commit -m "feat(cli): add chronicle ui command"
```

---

### Task 6: Add Mode Detection Store

**Files:**
- Create: `web/src/stores/mode.ts`
- Modify: `web/src/App.tsx`

**Step 1: Create mode store**

```typescript
// web/src/stores/mode.ts
import { create } from 'zustand'

export type AppMode = 'standalone' | 'daemon' | 'disconnected' | 'detecting'

interface ModeState {
  mode: AppMode
  setMode: (mode: AppMode) => void
  detectMode: () => Promise<void>
}

export const useModeStore = create<ModeState>((set) => ({
  mode: 'detecting',
  setMode: (mode) => set({ mode }),
  detectMode: async () => {
    // Try standalone API first
    try {
      const res = await fetch('/api/local/project')
      if (res.ok) {
        set({ mode: 'standalone' })
        return
      }
    } catch {
      // Not standalone
    }

    // Try daemon API
    try {
      const res = await fetch('/api/v1/health')
      if (res.ok) {
        set({ mode: 'daemon' })
        return
      }
    } catch {
      // Not daemon
    }

    set({ mode: 'disconnected' })
  },
}))

export function useMode() {
  return useModeStore((state) => state.mode)
}

export function useDetectMode() {
  return useModeStore((state) => state.detectMode)
}
```

**Step 2: Update App.tsx to detect mode on load**

```typescript
// Add to App.tsx
import { useEffect } from 'react'
import { useModeStore, useMode } from '@/stores/mode'

function ModeDetector({ children }: { children: React.ReactNode }) {
  const mode = useMode()
  const detectMode = useModeStore((state) => state.detectMode)

  useEffect(() => {
    detectMode()
  }, [detectMode])

  if (mode === 'detecting') {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-muted-foreground">Connecting...</div>
      </div>
    )
  }

  if (mode === 'disconnected') {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="text-xl font-semibold">Not Connected</h1>
          <p className="text-muted-foreground mt-2">
            Start Chronicle with `chronicle ui` or `chronicle daemon`
          </p>
        </div>
      </div>
    )
  }

  return <>{children}</>
}

// Wrap routes with ModeDetector
```

**Step 3: Build frontend**

Run: `cd web && npm run build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add web/src/stores/mode.ts web/src/App.tsx
git commit -m "feat(web): add mode detection store"
```

---

### Task 7: Create Local API Client

**Files:**
- Create: `web/src/api/local.ts`
- Modify: `web/src/api/index.ts`

**Step 1: Create local.ts**

```typescript
// web/src/api/local.ts
import { apiRequest, ApiError } from './client'

export interface ProjectInfo {
  directory: string
  config_file: string
  config_exists: boolean
  last_modified?: string
}

export interface ChronicleConfig {
  version: string
  scenarios?: ScenarioConfig[]
  infrastructure?: InfrastructureConfig
  chaos?: Record<string, ChaosProfile>
  mocks?: Record<string, MockProfile>
}

export interface ScenarioConfig {
  name: string
  description?: string
  tags?: string[]
  timeout?: string
  parallel?: number
  flow: FlowStep[]
}

export interface FlowStep {
  component: string
  timeout?: string
  condition?: string
}

export interface InfrastructureConfig {
  providers?: ProviderConfig[]
}

export interface ProviderConfig {
  name: string
  type: string
  config?: Record<string, unknown>
}

export interface ChaosProfile {
  name: string
  infrastructure?: ChaosInfraConfig[]
  application?: ChaosAppConfig[]
}

export interface ChaosInfraConfig {
  target: string
  action: string
  duration?: string
}

export interface ChaosAppConfig {
  type: string
  config?: Record<string, unknown>
}

export interface MockProfile {
  name: string
  injectors?: MockInjector[]
}

export interface MockInjector {
  target: string
  responses?: MockResponse[]
}

export interface MockResponse {
  match?: Record<string, unknown>
  return?: unknown
}

export interface ValidationResult {
  valid: boolean
  errors: string[]
  warnings: string[]
}

export interface DiscoveredComponent {
  name: string
  type: 'setup' | 'task' | 'validation' | 'teardown'
  description: string
  tags: string[]
  produces: string[]
  requires: string[]
  source_file: string
}

export interface DiscoveryResult {
  components: DiscoveredComponent[]
  discovered_at: string
}

const LOCAL_BASE = '/api/local'

export async function fetchProject(): Promise<ProjectInfo> {
  return apiRequest<ProjectInfo>(`${LOCAL_BASE}/project`)
}

export async function fetchConfig(): Promise<ChronicleConfig> {
  return apiRequest<ChronicleConfig>(`${LOCAL_BASE}/config`)
}

export async function saveConfig(config: ChronicleConfig): Promise<void> {
  await apiRequest<{ status: string }>(`${LOCAL_BASE}/config`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
}

export async function validateConfig(config: ChronicleConfig): Promise<ValidationResult> {
  return apiRequest<ValidationResult>(`${LOCAL_BASE}/config/validate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
}

export async function runDiscovery(): Promise<DiscoveryResult> {
  return apiRequest<DiscoveryResult>(`${LOCAL_BASE}/discover`, {
    method: 'POST',
  })
}

export async function fetchLocalComponents(): Promise<DiscoveryResult> {
  return apiRequest<DiscoveryResult>(`${LOCAL_BASE}/components`)
}
```

**Step 2: Export from index.ts**

Add to `web/src/api/index.ts`:
```typescript
export * from './local'
```

**Step 3: Build frontend**

Run: `cd web && npm run build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add web/src/api/local.ts web/src/api/index.ts
git commit -m "feat(web): add local API client for standalone mode"
```

---

### Task 8: Create Config Hook

**Files:**
- Create: `web/src/hooks/useLocalConfig.ts`

**Step 1: Create useLocalConfig.ts**

```typescript
// web/src/hooks/useLocalConfig.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchConfig,
  saveConfig,
  validateConfig,
  fetchProject,
  type ChronicleConfig,
  type ValidationResult,
  type ProjectInfo,
} from '@/api/local'

export function useProject() {
  return useQuery<ProjectInfo>({
    queryKey: ['local', 'project'],
    queryFn: fetchProject,
  })
}

export function useConfig() {
  return useQuery<ChronicleConfig>({
    queryKey: ['local', 'config'],
    queryFn: fetchConfig,
  })
}

export function useSaveConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: saveConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['local', 'config'] })
      queryClient.invalidateQueries({ queryKey: ['local', 'project'] })
    },
  })
}

export function useValidateConfig() {
  return useMutation<ValidationResult, Error, ChronicleConfig>({
    mutationFn: validateConfig,
  })
}
```

**Step 2: Build frontend**

Run: `cd web && npm run build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add web/src/hooks/useLocalConfig.ts
git commit -m "feat(web): add useLocalConfig hook for config management"
```

---

### Task 9: Create Config Editor Page

**Files:**
- Create: `web/src/pages/ConfigEditor.tsx`
- Create: `web/src/components/config/GeneralSection.tsx`
- Create: `web/src/components/config/ScenariosSection.tsx`

**Step 1: Create ConfigEditor.tsx**

```typescript
// web/src/pages/ConfigEditor.tsx
import { useState } from 'react'
import { useConfig, useSaveConfig, useValidateConfig } from '@/hooks/useLocalConfig'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, Save, CheckCircle2, AlertCircle } from 'lucide-react'
import type { ChronicleConfig } from '@/api/local'

type TabId = 'general' | 'scenarios' | 'infrastructure' | 'chaos' | 'mocks'

const TABS: { id: TabId; label: string }[] = [
  { id: 'general', label: 'General' },
  { id: 'scenarios', label: 'Scenarios' },
  { id: 'infrastructure', label: 'Infrastructure' },
  { id: 'chaos', label: 'Chaos' },
  { id: 'mocks', label: 'Mocks' },
]

export function ConfigEditor() {
  const { data: config, isLoading, error } = useConfig()
  const saveConfig = useSaveConfig()
  const validateConfig = useValidateConfig()
  const [activeTab, setActiveTab] = useState<TabId>('general')
  const [editedConfig, setEditedConfig] = useState<ChronicleConfig | null>(null)
  const [validationResult, setValidationResult] = useState<{ valid: boolean; errors: string[] } | null>(null)

  // Initialize edited config when loaded
  const currentConfig = editedConfig ?? config

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center text-destructive">
              <AlertCircle className="mx-auto h-8 w-8 mb-2" />
              <p>Failed to load configuration</p>
              <p className="text-sm text-muted-foreground mt-1">{error.message}</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!currentConfig) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6 text-center text-muted-foreground">
            No configuration found. Create a chronicle.yaml file to get started.
          </CardContent>
        </Card>
      </div>
    )
  }

  const handleSave = async () => {
    if (!editedConfig) return

    // Validate first
    const result = await validateConfig.mutateAsync(editedConfig)
    setValidationResult(result)

    if (result.valid) {
      await saveConfig.mutateAsync(editedConfig)
      setEditedConfig(null)
    }
  }

  const hasChanges = editedConfig !== null

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">Edit your Chronicle configuration</p>
        </div>
        <div className="flex items-center gap-2">
          {hasChanges && (
            <Badge variant="secondary">Unsaved changes</Badge>
          )}
          {validationResult && !validationResult.valid && (
            <Badge variant="destructive">
              {validationResult.errors.length} error(s)
            </Badge>
          )}
          <Button
            onClick={handleSave}
            disabled={!hasChanges || saveConfig.isPending}
          >
            {saveConfig.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : saveConfig.isSuccess ? (
              <CheckCircle2 className="mr-2 h-4 w-4" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Save
          </Button>
        </div>
      </div>

      {/* Tab navigation */}
      <div className="flex gap-1 border-b">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
              activeTab === tab.id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <Card>
        <CardHeader>
          <CardTitle>{TABS.find(t => t.id === activeTab)?.label}</CardTitle>
        </CardHeader>
        <CardContent>
          {activeTab === 'general' && (
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">Version</label>
                <p className="text-muted-foreground">{currentConfig.version}</p>
              </div>
            </div>
          )}
          {activeTab === 'scenarios' && (
            <div className="space-y-4">
              {currentConfig.scenarios?.map((scenario, i) => (
                <Card key={i}>
                  <CardHeader className="py-3">
                    <CardTitle className="text-base">{scenario.name}</CardTitle>
                  </CardHeader>
                  <CardContent className="py-3">
                    <p className="text-sm text-muted-foreground">
                      {scenario.flow?.length || 0} steps
                    </p>
                  </CardContent>
                </Card>
              )) || (
                <p className="text-muted-foreground">No scenarios defined</p>
              )}
            </div>
          )}
          {activeTab === 'infrastructure' && (
            <p className="text-muted-foreground">Infrastructure configuration coming soon</p>
          )}
          {activeTab === 'chaos' && (
            <p className="text-muted-foreground">Chaos profiles coming soon</p>
          )}
          {activeTab === 'mocks' && (
            <p className="text-muted-foreground">Mock profiles coming soon</p>
          )}
        </CardContent>
      </Card>

      {/* Validation errors */}
      {validationResult && !validationResult.valid && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">Validation Errors</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="list-disc list-inside space-y-1">
              {validationResult.errors.map((err, i) => (
                <li key={i} className="text-sm text-destructive">{err}</li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
```

**Step 2: Build frontend**

Run: `cd web && npm run build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add web/src/pages/ConfigEditor.tsx
git commit -m "feat(web): add config editor page"
```

---

### Task 10: Add Standalone Routes

**Files:**
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/layout/Sidebar.tsx`

**Step 1: Update App.tsx with mode-based routing**

```typescript
// Update App.tsx to conditionally render routes based on mode
import { ConfigEditor } from '@/pages/ConfigEditor'

// In the routes, add:
// For standalone mode: /config route
// Modify Layout to pass mode prop
```

**Step 2: Update Sidebar to show different nav based on mode**

```typescript
// Add mode-aware navigation
// Standalone: Config, Scenarios, Components
// Daemon: Dashboard, Scenarios, Runs, Results, Components
```

**Step 3: Build and test**

Run: `cd web && npm run build && cd .. && make build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add web/src/
git commit -m "feat(web): add mode-based routing for standalone UI"
```

---

### Task 11: Final Integration Testing

**Files:**
- Test the full flow manually

**Step 1: Build everything**

Run: `make build`

**Step 2: Test standalone mode**

Run: `./bin/chronicle ui --dir ./examples/full-stack`
Expected: Browser opens, shows config editor, can view scenarios

**Step 3: Verify mode detection**

- With `chronicle ui`: Should show Config, Scenarios, Components nav
- With `chronicle daemon`: Should show Dashboard, Scenarios, Runs, Results, Components nav

**Step 4: Run all tests**

Run: `go test ./... && cd web && npm run build`
Expected: All pass

**Step 5: Update documentation**

Update gap-analysis.md and PROGRESS.md

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: complete standalone UI implementation"
```

---

## Summary

This plan implements:
1. New `pkg/ui` package with local HTTP server
2. Local API endpoints for config and discovery
3. `chronicle ui` CLI command
4. Frontend mode detection
5. Config editor page with tabbed interface
6. Mode-aware navigation

Total: 11 tasks
