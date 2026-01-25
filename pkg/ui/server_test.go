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
