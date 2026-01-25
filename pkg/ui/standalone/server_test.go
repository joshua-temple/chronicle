package standalone

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// TestModeEndpoint tests the GET /api/standalone/mode endpoint.
func TestModeEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/standalone/mode", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["mode"] != "standalone" {
		t.Errorf("expected mode 'standalone', got %q", resp["mode"])
	}
}

// TestListProjects tests the GET /api/standalone/projects endpoint.
func TestListProjects(t *testing.T) {
	srv := setupTestServer(t)

	// Add some test projects
	p1 := Project{Name: "project1", Path: "/tmp/project1"}
	p2 := Project{Name: "project2", Path: "/tmp/project2"}

	id1, err := srv.registry.Add(p1)
	if err != nil {
		t.Fatalf("failed to add project1: %v", err)
	}
	id2, err := srv.registry.Add(p2)
	if err != nil {
		t.Fatalf("failed to add project2: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/standalone/projects", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Projects []struct {
			ID     string        `json:"id"`
			Name   string        `json:"name"`
			Path   string        `json:"path"`
			Status *DaemonStatus `json:"status"`
		} `json:"projects"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(resp.Projects))
	}

	// Verify IDs match
	foundIDs := map[string]bool{resp.Projects[0].ID: true, resp.Projects[1].ID: true}
	if !foundIDs[id1] || !foundIDs[id2] {
		t.Errorf("expected project IDs %s and %s, got %v", id1, id2, foundIDs)
	}
}

// TestAddProject tests the POST /api/standalone/projects endpoint.
func TestAddProject(t *testing.T) {
	srv := setupTestServer(t)

	// Create test project directory
	tmpDir := t.TempDir()

	body := strings.NewReader(`{"name":"testproject","path":"` + tmpDir + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/standalone/projects", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}

	// Verify project was added to registry
	project := srv.registry.Get(resp.ID)
	if project == nil {
		t.Fatal("project not found in registry")
	}
	if project.Name != "testproject" {
		t.Errorf("expected name 'testproject', got %q", project.Name)
	}
	if project.Path != tmpDir {
		t.Errorf("expected path %q, got %q", tmpDir, project.Path)
	}
}

// TestAddProjectDuplicate tests adding a project with duplicate path.
func TestAddProjectDuplicate(t *testing.T) {
	srv := setupTestServer(t)

	tmpDir := t.TempDir()

	// Add first project
	p := Project{Name: "project1", Path: tmpDir}
	_, err := srv.registry.Add(p)
	if err != nil {
		t.Fatalf("failed to add first project: %v", err)
	}

	// Try to add duplicate
	body := strings.NewReader(`{"name":"project2","path":"` + tmpDir + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/standalone/projects", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

// TestRemoveProject tests the DELETE /api/standalone/projects/{id} endpoint.
func TestRemoveProject(t *testing.T) {
	srv := setupTestServer(t)

	// Add a project
	p := Project{Name: "project1", Path: "/tmp/project1"}
	id, err := srv.registry.Add(p)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/standalone/projects/"+id, nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify project was removed
	if srv.registry.Get(id) != nil {
		t.Error("project should have been removed from registry")
	}
}

// TestRemoveProjectNotFound tests removing a non-existent project.
func TestRemoveProjectNotFound(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/standalone/projects/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestUpdateProject tests the PUT /api/standalone/projects/{id} endpoint.
func TestUpdateProject(t *testing.T) {
	srv := setupTestServer(t)

	// Add a project
	p := Project{Name: "project1", Path: "/tmp/project1", Preferences: map[string]interface{}{}}
	id, err := srv.registry.Add(p)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	// Update preferences
	body := strings.NewReader(`{"preferences":{"theme":"dark","autoRefresh":true}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/standalone/projects/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify preferences were updated
	updated := srv.registry.Get(id)
	if updated == nil {
		t.Fatal("project not found after update")
	}
	if updated.Preferences["theme"] != "dark" {
		t.Errorf("expected theme 'dark', got %v", updated.Preferences["theme"])
	}
	if updated.Preferences["autoRefresh"] != true {
		t.Errorf("expected autoRefresh true, got %v", updated.Preferences["autoRefresh"])
	}
}

// TestUpdateProjectNotFound tests updating a non-existent project.
func TestUpdateProjectNotFound(t *testing.T) {
	srv := setupTestServer(t)

	body := strings.NewReader(`{"preferences":{"theme":"dark"}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/standalone/projects/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestLaunchDaemon tests the POST /api/standalone/projects/{id}/launch endpoint.
func TestLaunchDaemon(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping launcher test in short mode")
	}

	srv := setupTestServer(t)

	// Add a local project
	tmpDir := t.TempDir()
	p := Project{Name: "project1", Path: tmpDir}
	id, err := srv.registry.Add(p)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/standalone/projects/"+id+"/launch", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	// Should fail because we don't have a real chronicle binary
	// This is expected to return 500 since the server can't launch the daemon
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	// Verify error response contains useful information
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestLaunchDaemonRemoteProject tests launching a daemon for a remote project.
func TestLaunchDaemonRemoteProject(t *testing.T) {
	srv := setupTestServer(t)

	// Add a remote project
	p := Project{Name: "remote", Path: "/tmp/remote", RemoteURL: "https://example.com"}
	id, err := srv.registry.Add(p)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/standalone/projects/"+id+"/launch", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp["error"], "remote") {
		t.Errorf("expected error about remote project, got %q", resp["error"])
	}
}

// TestStopDaemon tests the POST /api/standalone/projects/{id}/stop endpoint.
func TestStopDaemon(t *testing.T) {
	srv := setupTestServer(t)

	// Add a local project
	p := Project{Name: "project1", Path: "/tmp/project1"}
	id, err := srv.registry.Add(p)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/standalone/projects/"+id+"/stop", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	// Should return error because no daemon is running
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHealthCheck tests the GET /api/standalone/projects/{id}/health endpoint.
func TestHealthCheck(t *testing.T) {
	srv := setupTestServer(t)

	// Add a project with a path that definitely won't have a daemon
	p := Project{Name: "project1", Path: "/nonexistent/project1"}
	id, err := srv.registry.Add(p)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/standalone/projects/"+id+"/health", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var status DaemonStatus
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should be stopped or unhealthy since no daemon is running
	if status.State != StateStopped && status.State != StateUnhealthy {
		t.Errorf("expected state 'stopped' or 'unhealthy', got %q", status.State)
	}
}

// TestHealthCheckNotFound tests health check for non-existent project.
func TestHealthCheckNotFound(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/standalone/projects/nonexistent/health", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestDiscover tests the POST /api/standalone/discover endpoint.
func TestDiscover(t *testing.T) {
	srv := setupTestServer(t)

	// Create test directories with chronicle.yaml
	tmpDir := t.TempDir()
	project1 := filepath.Join(tmpDir, "project1")
	project2 := filepath.Join(tmpDir, "project2")

	if err := os.MkdirAll(project1, 0755); err != nil {
		t.Fatalf("failed to create project1: %v", err)
	}
	if err := os.MkdirAll(project2, 0755); err != nil {
		t.Fatalf("failed to create project2: %v", err)
	}

	// Create chronicle.yaml files
	if err := os.WriteFile(filepath.Join(project1, "chronicle.yaml"), []byte("version: 1.0\n"), 0644); err != nil {
		t.Fatalf("failed to create chronicle.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project2, "chronicle.yaml"), []byte("version: 1.0\n"), 0644); err != nil {
		t.Fatalf("failed to create chronicle.yaml: %v", err)
	}

	body := strings.NewReader(`{"path":"` + tmpDir + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/standalone/discover", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Found []string `json:"found"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Found) != 2 {
		t.Errorf("expected 2 projects, got %d: %v", len(resp.Found), resp.Found)
	}
}

// TestSPAHandler tests the SPA routing (serving index.html for non-API routes).
func TestSPAHandler(t *testing.T) {
	// Create a test filesystem
	testFS := fstest.MapFS{
		"dist/index.html": {
			Data: []byte("<html><body>Test App</body></html>"),
		},
		"dist/assets/main.js": {
			Data: []byte("console.log('test');"),
		},
	}

	srv := setupTestServerWithFS(t, testFS)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "root serves index.html",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "Test App",
		},
		{
			name:       "client route serves index.html",
			path:       "/scenarios",
			wantStatus: http.StatusOK,
			wantBody:   "Test App",
		},
		{
			name:       "asset file serves directly",
			path:       "/assets/main.js",
			wantStatus: http.StatusOK,
			wantBody:   "console.log('test')",
		},
		{
			name:       "api route not handled by SPA",
			path:       "/api/standalone/mode",
			wantStatus: http.StatusOK,
			wantBody:   "standalone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBody, w.Body.String())
			}
		})
	}
}

// TestCORSHeaders tests that CORS headers are set for API requests.
func TestCORSHeaders(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/standalone/mode", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected Access-Control-Allow-Origin header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
}

// TestServerStartStop tests starting and stopping the server.
func TestServerStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "projects.json")

	registry, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	srv, err := NewServer(
		WithPort(0), // use random port
		WithRegistry(registry),
	)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	if err := srv.Stop(stopCtx); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}

	// Wait for start to finish
	select {
	case err := <-errChan:
		// Server should exit without error when stopped gracefully
		if err != nil && err != http.ErrServerClosed && !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("unexpected error from Start: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("server did not stop in time")
	}
}

// TestWithOptions tests server options.
func TestWithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "projects.json")

	registry, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	srv, err := NewServer(
		WithPort(9999),
		WithRegistry(registry),
	)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if srv.port != 9999 {
		t.Errorf("expected port 9999, got %d", srv.port)
	}

	if srv.registry == nil {
		t.Error("expected registry to be set")
	}
}

// setupTestServer creates a test server with in-memory registry.
func setupTestServer(t *testing.T) *Server {
	t.Helper()

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "projects.json")

	registry, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Create a simple test filesystem
	testFS := fstest.MapFS{
		"dist/index.html": {
			Data: []byte("<html><body>Test</body></html>"),
		},
	}

	srv, err := NewServer(
		WithPort(0),
		WithRegistry(registry),
		WithWebFS(testFS),
	)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv
}

// setupTestServerWithFS creates a test server with a custom filesystem.
func setupTestServerWithFS(t *testing.T, webFS fs.FS) *Server {
	t.Helper()

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "projects.json")

	registry, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	srv, err := NewServer(
		WithPort(0),
		WithRegistry(registry),
		WithWebFS(webFS),
	)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv
}
