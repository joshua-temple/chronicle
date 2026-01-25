package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		s := New()

		if s.Port() != DefaultPort {
			t.Errorf("expected port %d, got %d", DefaultPort, s.Port())
		}

		if s.Dir() != DefaultDir {
			t.Errorf("expected dir %q, got %q", DefaultDir, s.Dir())
		}
	})

	t.Run("with custom port", func(t *testing.T) {
		s := New(WithPort(8080))

		if s.Port() != 8080 {
			t.Errorf("expected port 8080, got %d", s.Port())
		}
	})

	t.Run("with custom dir", func(t *testing.T) {
		s := New(WithDir("/custom/path"))

		if s.Dir() != "/custom/path" {
			t.Errorf("expected dir /custom/path, got %q", s.Dir())
		}
	})

	t.Run("with multiple options", func(t *testing.T) {
		s := New(
			WithPort(9000),
			WithDir("/my/project"),
		)

		if s.Port() != 9000 {
			t.Errorf("expected port 9000, got %d", s.Port())
		}

		if s.Dir() != "/my/project" {
			t.Errorf("expected dir /my/project, got %q", s.Dir())
		}
	})
}

func TestServer_Start(t *testing.T) {
	t.Run("starts and shuts down gracefully", func(t *testing.T) {
		// Use a high port to avoid conflicts
		s := New(WithPort(19876))

		ctx, cancel := context.WithCancel(context.Background())

		// Channel to capture server start
		started := make(chan struct{})
		errCh := make(chan error, 1)

		go func() {
			// Signal that we're about to start
			close(started)
			errCh <- s.Start(ctx)
		}()

		// Wait for server to start
		<-started
		time.Sleep(100 * time.Millisecond) // Give server time to bind

		// Verify server is running by making a request
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/local/project", s.Port()))
		if err != nil {
			t.Fatalf("failed to connect to server: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		// Cancel context to trigger shutdown
		cancel()

		// Wait for server to shut down
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("server returned error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("server did not shut down in time")
		}
	})

	t.Run("routes are registered", func(t *testing.T) {
		s := New(WithPort(19877))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- s.Start(ctx)
		}()

		// Wait for server to start
		time.Sleep(100 * time.Millisecond)

		// Test all API routes - expected status varies by endpoint behavior
		routes := []struct {
			method         string
			path           string
			expectedStatus int
		}{
			{"GET", "/api/local/project", http.StatusOK},
			{"GET", "/api/local/config", http.StatusNotFound},  // No config file in default dir
			{"PUT", "/api/local/config", http.StatusBadRequest}, // Empty body is invalid
			{"POST", "/api/local/config/validate", http.StatusOK},
			{"POST", "/api/local/discover", http.StatusOK},
			{"GET", "/api/local/components", http.StatusOK},
		}

		client := &http.Client{Timeout: 5 * time.Second}

		for _, route := range routes {
			t.Run(fmt.Sprintf("%s %s", route.method, route.path), func(t *testing.T) {
				url := fmt.Sprintf("http://localhost:%d%s", s.Port(), route.path)
				req, err := http.NewRequest(route.method, url, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				defer func() { _ = resp.Body.Close() }()

				if resp.StatusCode != route.expectedStatus {
					t.Errorf("expected status %d, got %d", route.expectedStatus, resp.StatusCode)
				}

				contentType := resp.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %q", contentType)
				}
			})
		}

		cancel()
		<-errCh
	})
}

func TestServer_Accessors(t *testing.T) {
	s := New(WithPort(5000), WithDir("/test/dir"))

	if s.Port() != 5000 {
		t.Errorf("Port() = %d, want 5000", s.Port())
	}

	if s.Dir() != "/test/dir" {
		t.Errorf("Dir() = %q, want /test/dir", s.Dir())
	}
}

func TestServer_HandleProject(t *testing.T) {
	// Test with config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "chronicle.yaml")
	if err := os.WriteFile(configPath, []byte("version: \"1\""), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))
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
	if info.ConfigFile != "chronicle.yaml" {
		t.Errorf("expected config file 'chronicle.yaml', got %s", info.ConfigFile)
	}

	// Test without config file
	tmpDir2 := t.TempDir()
	s2 := New(WithDir(tmpDir2))
	req2 := httptest.NewRequest("GET", "/api/local/project", nil)
	w2 := httptest.NewRecorder()
	s2.mux.ServeHTTP(w2, req2)

	var info2 ProjectInfo
	if err := json.NewDecoder(w2.Body).Decode(&info2); err != nil {
		t.Fatal(err)
	}

	if info2.ConfigExists {
		t.Error("expected config to not exist")
	}
}

func TestServer_HandleGetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `version: "1"
scenarios:
  - name: test
    flow:
      - setup: Setup
`
	if err := os.WriteFile(filepath.Join(tmpDir, "chronicle.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))
	req := httptest.NewRequest("GET", "/api/local/config", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_HandleGetConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(WithDir(tmpDir))
	req := httptest.NewRequest("GET", "/api/local/config", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestServer_HandleValidateConfig(t *testing.T) {
	s := New()
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

func TestServer_HandleValidateConfig_Invalid(t *testing.T) {
	s := New()
	body := `{"scenarios": [{"name": ""}]}`
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

	if result.Valid {
		t.Error("expected invalid config")
	}
	if len(result.Errors) == 0 {
		t.Error("expected validation errors")
	}
}

func TestServer_HandleValidateConfig_InvalidJSON(t *testing.T) {
	s := New()
	body := `{invalid json}`
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

	if result.Valid {
		t.Error("expected invalid config")
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for invalid JSON")
	}
}

func TestServer_HandlePutConfig(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(WithDir(tmpDir))

	// Valid config with at least one scenario that has a flow
	body := `{"version": "1", "scenarios": [{"name": "test-scenario", "flow": [{"setup": "TestSetup"}]}]}`
	req := httptest.NewRequest("PUT", "/api/local/config", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was written
	configPath := filepath.Join(tmpDir, "chronicle.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestServer_HandlePutConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(WithDir(tmpDir))

	body := `{invalid json}`
	req := httptest.NewRequest("PUT", "/api/local/config", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestServer_HandlePutConfig_ValidationFailed(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(WithDir(tmpDir))

	// Invalid config: scenario without name
	body := `{"scenarios": [{"name": ""}]}`
	req := httptest.NewRequest("PUT", "/api/local/config", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_HandleDiscover(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(WithDir(tmpDir))

	req := httptest.NewRequest("POST", "/api/local/discover", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result DiscoveryResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	// Empty directory should have no components
	if len(result.Components) != 0 {
		t.Errorf("expected 0 components, got %d", len(result.Components))
	}

	if result.DiscoveredAt.IsZero() {
		t.Error("expected non-zero discovery time")
	}
}

func TestServer_HandleDiscover_WithAnnotatedCode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go file with Chronicle annotations
	goCode := `package testpkg

// @chronicle:setup name="CreateUser" produces="user:User"
// @chronicle:description "Creates a test user"
// @chronicle:tags auth, user
func CreateUser() {}

// @chronicle:task name="ProcessOrder" requires="user:User" produces="order:Order"
func ProcessOrder() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "components.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))
	req := httptest.NewRequest("POST", "/api/local/discover", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result DiscoveryResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	// Should find the annotated components
	if len(result.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(result.Components))
	}

	// Verify components are cached
	s.componentsMu.RLock()
	cachedCount := len(s.components)
	s.componentsMu.RUnlock()

	if cachedCount != 2 {
		t.Errorf("expected 2 cached components, got %d", cachedCount)
	}
}

func TestServer_HandleGetComponents_Empty(t *testing.T) {
	s := New()
	req := httptest.NewRequest("GET", "/api/local/components", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result DiscoveryResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	// No discovery run yet, should return empty
	if len(result.Components) != 0 {
		t.Errorf("expected 0 components, got %d", len(result.Components))
	}

	if !result.DiscoveredAt.IsZero() {
		t.Error("expected zero discovery time for empty cache")
	}
}

func TestServer_HandleGetComponents_AfterDiscover(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go file with a Chronicle annotation
	goCode := `package testpkg

// @chronicle:setup name="TestSetup" produces="data:Data"
func TestSetup() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "setup.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))

	// First, run discovery
	req1 := httptest.NewRequest("POST", "/api/local/discover", nil)
	w1 := httptest.NewRecorder()
	s.mux.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("discover failed: %d: %s", w1.Code, w1.Body.String())
	}

	var discoverResult DiscoveryResult
	if err := json.NewDecoder(w1.Body).Decode(&discoverResult); err != nil {
		t.Fatal(err)
	}

	// Now get components from cache
	req2 := httptest.NewRequest("GET", "/api/local/components", nil)
	w2 := httptest.NewRecorder()
	s.mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}

	var componentsResult DiscoveryResult
	if err := json.NewDecoder(w2.Body).Decode(&componentsResult); err != nil {
		t.Fatal(err)
	}

	// Should return cached components
	if len(componentsResult.Components) != 1 {
		t.Errorf("expected 1 component, got %d", len(componentsResult.Components))
	}

	// Discovery time should match
	if !componentsResult.DiscoveredAt.Equal(discoverResult.DiscoveredAt) {
		t.Error("discovery times should match")
	}
}

func TestDiscoveredComponent_Fields(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a comprehensive Go file with all annotation fields
	goCode := `package testpkg

// @chronicle:task name="FullTask" requires="input:Input" produces="output:Output"
// @chronicle:description "A fully annotated task"
// @chronicle:tags critical, api
func FullTask() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "full.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(WithDir(tmpDir))
	req := httptest.NewRequest("POST", "/api/local/discover", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("discover failed: %d: %s", w.Code, w.Body.String())
	}

	var result DiscoveryResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	if len(result.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(result.Components))
	}

	c := result.Components[0]

	if c.Name != "FullTask" {
		t.Errorf("expected name 'FullTask', got %q", c.Name)
	}

	if c.Type != "task" {
		t.Errorf("expected type 'task', got %q", c.Type)
	}

	if c.Description != "A fully annotated task" {
		t.Errorf("expected description 'A fully annotated task', got %q", c.Description)
	}

	if len(c.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(c.Tags))
	}

	if len(c.Requires) != 1 || c.Requires[0] != "input:Input" {
		t.Errorf("expected requires ['input:Input'], got %v", c.Requires)
	}

	if len(c.Produces) != 1 || c.Produces[0] != "output:Output" {
		t.Errorf("expected produces ['output:Output'], got %v", c.Produces)
	}

	if c.SourceFile == "" {
		t.Error("expected non-empty source file")
	}
}
