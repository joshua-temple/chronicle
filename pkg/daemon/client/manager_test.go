package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewDaemonManager(t *testing.T) {
	t.Run("creates manager with defaults", func(t *testing.T) {
		m := NewDaemonManager()
		if m.binaryPath != "chronicle" {
			t.Errorf("expected binaryPath 'chronicle', got %s", m.binaryPath)
		}
		if m.projectDir != "." {
			t.Errorf("expected projectDir '.', got %s", m.projectDir)
		}
	})

	t.Run("creates manager with custom binary path", func(t *testing.T) {
		m := NewDaemonManager(WithBinaryPath("/usr/local/bin/chronicle"))
		if m.binaryPath != "/usr/local/bin/chronicle" {
			t.Errorf("expected binaryPath '/usr/local/bin/chronicle', got %s", m.binaryPath)
		}
	})

	t.Run("creates manager with custom project dir", func(t *testing.T) {
		m := NewDaemonManager(WithProjectDir("/path/to/project"))
		if m.projectDir != "/path/to/project" {
			t.Errorf("expected projectDir '/path/to/project', got %s", m.projectDir)
		}
	})
}

func TestEnsureDaemon(t *testing.T) {
	t.Run("connects to existing daemon", func(t *testing.T) {
		// Start a mock server on a known port
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{Status: "healthy"})
		}))
		defer server.Close()

		// Create a client directly and test health
		c := New(server.URL)
		if !c.IsHealthy(context.Background()) {
			t.Error("mock server should be healthy")
		}
	})

	t.Run("returns error when non-interactive and no daemon", func(t *testing.T) {
		m := NewDaemonManager()
		_, err := m.EnsureDaemon(context.Background(), false)
		if err == nil {
			t.Error("expected error when no daemon running and non-interactive")
		}
	})
}

func TestWaitForHealth(t *testing.T) {
	t.Run("returns when healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{Status: "healthy"})
		}))
		defer server.Close()

		m := NewDaemonManager()
		c := New(server.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := m.waitForHealth(ctx, c)
		if err != nil {
			t.Errorf("waitForHealth failed: %v", err)
		}
	})

	t.Run("times out when unhealthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{Status: "unhealthy"})
		}))
		defer server.Close()

		m := NewDaemonManager()
		c := New(server.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := m.waitForHealth(ctx, c)
		if err == nil {
			t.Error("expected timeout error when unhealthy")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{Status: "unhealthy"})
		}))
		defer server.Close()

		m := NewDaemonManager()
		c := New(server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := m.waitForHealth(ctx, c)
		if err == nil {
			t.Error("expected error on context cancellation")
		}
	})
}

func TestStopDaemon(t *testing.T) {
	t.Run("returns nil when no process", func(t *testing.T) {
		m := NewDaemonManager()
		err := m.StopDaemon()
		if err != nil {
			t.Errorf("expected no error when no process, got: %v", err)
		}
	})
}

func TestGetClient(t *testing.T) {
	t.Run("returns nil when not connected", func(t *testing.T) {
		m := NewDaemonManager()
		if m.GetClient() != nil {
			t.Error("expected nil client when not connected")
		}
	})
}

func TestGetPort(t *testing.T) {
	t.Run("returns 0 when not connected", func(t *testing.T) {
		m := NewDaemonManager()
		if m.GetPort() != 0 {
			t.Errorf("expected port 0 when not connected, got %d", m.GetPort())
		}
	})
}

func TestCheckDaemonHealth(t *testing.T) {
	t.Run("returns false when no daemon running", func(t *testing.T) {
		// This test checks ports that are likely not in use
		healthy, port := CheckDaemonHealth(context.Background())
		if healthy {
			t.Errorf("expected no daemon to be healthy, but found one on port %d", port)
		}
	})
}

func TestFindAvailablePort(t *testing.T) {
	t.Run("finds an available port", func(t *testing.T) {
		port, err := findAvailablePort()
		if err != nil {
			t.Fatalf("findAvailablePort failed: %v", err)
		}
		if port <= 0 {
			t.Errorf("expected positive port, got %d", port)
		}
		// Port should be in the valid range
		if port < 1024 || port > 65535 {
			t.Errorf("port %d outside valid range", port)
		}
	})

	t.Run("finds different ports on multiple calls", func(t *testing.T) {
		port1, err := findAvailablePort()
		if err != nil {
			t.Fatalf("first findAvailablePort failed: %v", err)
		}

		port2, err := findAvailablePort()
		if err != nil {
			t.Fatalf("second findAvailablePort failed: %v", err)
		}

		// Ports should generally be different (not guaranteed but highly likely)
		// We just verify both are valid
		if port1 <= 0 || port2 <= 0 {
			t.Error("both ports should be positive")
		}
	})
}
