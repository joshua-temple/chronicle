package ui

import (
	"context"
	"fmt"
	"net/http"
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

		// Test all API routes
		routes := []struct {
			method string
			path   string
		}{
			{"GET", "/api/local/project"},
			{"GET", "/api/local/config"},
			{"PUT", "/api/local/config"},
			{"POST", "/api/local/config/validate"},
			{"POST", "/api/local/discover"},
			{"GET", "/api/local/components"},
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

				if resp.StatusCode != http.StatusOK {
					t.Errorf("expected status 200, got %d", resp.StatusCode)
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
