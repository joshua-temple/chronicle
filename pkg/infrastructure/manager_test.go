package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestManager_Configure(t *testing.T) {
	registry := NewRegistry()
	registry.Register("mock", func() Provider {
		return newMockProvider("mock")
	})

	t.Run("configure valid provider", func(t *testing.T) {
		mgr := NewManager(registry)
		err := mgr.Configure(ProviderConfig{
			Name:     "test-postgres",
			Provider: "mock",
			Config:   map[string]any{"host": "localhost"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		names := mgr.ProviderNames()
		if len(names) != 1 || names[0] != "test-postgres" {
			t.Errorf("expected one provider 'test-postgres', got %v", names)
		}
	})

	t.Run("configure unknown provider type", func(t *testing.T) {
		mgr := NewManager(registry)
		err := mgr.Configure(ProviderConfig{
			Name:     "test",
			Provider: "nonexistent",
		})
		if err == nil {
			t.Error("expected error for unknown provider type")
		}
	})

	t.Run("configure without name", func(t *testing.T) {
		mgr := NewManager(registry)
		err := mgr.Configure(ProviderConfig{
			Provider: "mock",
		})
		if err == nil {
			t.Error("expected error for missing name")
		}
	})
}

func TestManager_AddProvider(t *testing.T) {
	t.Run("add provider directly", func(t *testing.T) {
		mgr := NewManager(nil)
		provider := newMockProvider("direct")

		err := mgr.AddProvider("direct", provider, ProviderConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p, ok := mgr.Provider("direct")
		if !ok {
			t.Fatal("expected to find provider")
		}
		if p.Name() != "direct" {
			t.Errorf("expected name 'direct', got %s", p.Name())
		}
	})

	t.Run("add without name", func(t *testing.T) {
		mgr := NewManager(nil)
		err := mgr.AddProvider("", newMockProvider("test"), ProviderConfig{})
		if err == nil {
			t.Error("expected error for empty name")
		}
	})

	t.Run("add nil provider", func(t *testing.T) {
		mgr := NewManager(nil)
		err := mgr.AddProvider("test", nil, ProviderConfig{})
		if err == nil {
			t.Error("expected error for nil provider")
		}
	})
}

func TestManager_StartStop(t *testing.T) {
	ctx := context.Background()

	t.Run("start and stop providers", func(t *testing.T) {
		mgr := NewManager(nil)
		p1 := newMockProvider("p1")
		p2 := newMockProvider("p2")
		p1.clients["default"] = "client1"
		p2.clients["default"] = "client2"

		_ = mgr.AddProvider("p1", p1, ProviderConfig{Config: map[string]any{}})
		_ = mgr.AddProvider("p2", p2, ProviderConfig{Config: map[string]any{}})

		// Start
		err := mgr.Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}
		if !mgr.IsStarted() {
			t.Error("expected manager to be started")
		}

		// Check status
		statuses := mgr.Status()
		if statuses["p1"] != StatusRunning {
			t.Errorf("expected p1 running, got %v", statuses["p1"])
		}
		if statuses["p2"] != StatusRunning {
			t.Errorf("expected p2 running, got %v", statuses["p2"])
		}

		// Stop
		err = mgr.Stop(ctx)
		if err != nil {
			t.Fatalf("Stop failed: %v", err)
		}
		if mgr.IsStarted() {
			t.Error("expected manager to be stopped")
		}

		// Check status
		statuses = mgr.Status()
		if statuses["p1"] != StatusStopped {
			t.Errorf("expected p1 stopped, got %v", statuses["p1"])
		}
	})

	t.Run("start idempotent", func(t *testing.T) {
		mgr := NewManager(nil)
		p := newMockProvider("p")
		_ = mgr.AddProvider("p", p, ProviderConfig{Config: map[string]any{}})

		_ = mgr.Start(ctx)
		err := mgr.Start(ctx) // Second call should be no-op
		if err != nil {
			t.Fatalf("second Start should succeed: %v", err)
		}
	})

	t.Run("stop idempotent", func(t *testing.T) {
		mgr := NewManager(nil)
		err := mgr.Stop(ctx) // Should succeed even if not started
		if err != nil {
			t.Fatalf("Stop on unstarted manager should succeed: %v", err)
		}
	})

	t.Run("start failure rolls back", func(t *testing.T) {
		mgr := NewManager(nil)
		p1 := newMockProvider("p1")
		p2 := newMockProvider("p2")
		p2.startErr = errors.New("start failed")

		_ = mgr.AddProvider("p1", p1, ProviderConfig{Config: map[string]any{}})
		_ = mgr.AddProvider("p2", p2, ProviderConfig{Config: map[string]any{}})

		err := mgr.Start(ctx)
		if err == nil {
			t.Fatal("expected start to fail")
		}

		// p1 should have been stopped during rollback
		if p1.Status() != StatusStopped {
			t.Error("expected p1 to be stopped after rollback")
		}
	})
}

func TestManager_Client(t *testing.T) {
	ctx := context.Background()

	t.Run("get client from running provider", func(t *testing.T) {
		mgr := NewManager(nil)
		p := newMockProvider("db")
		p.clients["default"] = "db-client"
		p.clients["reader"] = "reader-client"

		_ = mgr.AddProvider("db", p, ProviderConfig{Config: map[string]any{}})
		_ = mgr.Start(ctx)

		// Get default client
		client, err := mgr.ClientByService("db")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client != "db-client" {
			t.Errorf("expected 'db-client', got %v", client)
		}

		// Get named client
		client, err = mgr.Client("db", "reader")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client != "reader-client" {
			t.Errorf("expected 'reader-client', got %v", client)
		}
	})

	t.Run("get client from unknown service", func(t *testing.T) {
		mgr := NewManager(nil)
		_, err := mgr.Client("nonexistent", "")
		if err == nil {
			t.Error("expected error for unknown service")
		}
	})

	t.Run("get client from stopped provider", func(t *testing.T) {
		mgr := NewManager(nil)
		p := newMockProvider("db")
		p.clients["default"] = "client"
		_ = mgr.AddProvider("db", p, ProviderConfig{Config: map[string]any{}})

		// Don't start - status is stopped
		_, err := mgr.Client("db", "")
		if err == nil {
			t.Error("expected error for stopped provider")
		}
	})
}

func TestManager_HealthCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("all healthy", func(t *testing.T) {
		mgr := NewManager(nil)
		p1 := newMockProvider("p1")
		p2 := newMockProvider("p2")

		_ = mgr.AddProvider("p1", p1, ProviderConfig{Config: map[string]any{}})
		_ = mgr.AddProvider("p2", p2, ProviderConfig{Config: map[string]any{}})
		_ = mgr.Start(ctx)

		if !mgr.AllHealthy(ctx) {
			t.Error("expected all providers to be healthy")
		}

		reports := mgr.HealthCheck(ctx)
		if len(reports) != 2 {
			t.Errorf("expected 2 reports, got %d", len(reports))
		}
	})

	t.Run("one unhealthy", func(t *testing.T) {
		mgr := NewManager(nil)
		p1 := newMockProvider("p1")
		p2 := newMockProvider("p2")
		p2.healthReport = HealthReport{Healthy: false}

		_ = mgr.AddProvider("p1", p1, ProviderConfig{Config: map[string]any{}})
		_ = mgr.AddProvider("p2", p2, ProviderConfig{Config: map[string]any{}})
		_ = mgr.Start(ctx)

		if mgr.AllHealthy(ctx) {
			t.Error("expected not all healthy")
		}
	})
}

func TestManager_Reset(t *testing.T) {
	ctx := context.Background()

	t.Run("reset with flush", func(t *testing.T) {
		mgr := NewManager(nil)
		mgr.SetDefaultReuse(ReuseWithFlush)

		p := newMockProvider("db")
		p.clients["default"] = "client"

		_ = mgr.AddProvider("db", p, ProviderConfig{
			Config: map[string]any{},
			Reuse:  ReuseWithFlush,
		})
		_ = mgr.Start(ctx)

		err := mgr.Reset(ctx)
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}
		// Provider should still be running after flush (not restarted)
		if p.Status() != StatusRunning {
			t.Errorf("expected running after flush, got %v", p.Status())
		}
	})

	t.Run("reset with full reuse does nothing", func(t *testing.T) {
		mgr := NewManager(nil)
		p := newMockProvider("db")
		p.clients["default"] = "client"
		_ = mgr.AddProvider("db", p, ProviderConfig{
			Config: map[string]any{},
			Reuse:  FullReuse,
		})
		_ = mgr.Start(ctx)

		initialStatus := p.Status()
		err := mgr.Reset(ctx)
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}
		if p.Status() != initialStatus {
			t.Error("expected status unchanged for full reuse")
		}
	})

	t.Run("reset with always fresh restarts", func(t *testing.T) {
		mgr := NewManager(nil)
		p := newMockProvider("db")
		p.clients["default"] = "client"
		_ = mgr.AddProvider("db", p, ProviderConfig{
			Config: map[string]any{},
			Reuse:  AlwaysFresh,
		})
		_ = mgr.Start(ctx)

		err := mgr.Reset(ctx)
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}
		// Provider should still be running after restart
		if p.Status() != StatusRunning {
			t.Errorf("expected running after restart, got %v", p.Status())
		}
	})
}

func TestManager_Defaults(t *testing.T) {
	t.Run("set default reuse", func(t *testing.T) {
		mgr := NewManager(nil)
		mgr.SetDefaultReuse(FullReuse)
		// No assertion - just verifying it doesn't panic
	})

	t.Run("set default isolation", func(t *testing.T) {
		mgr := NewManager(nil)
		mgr.SetDefaultIsolation(SchemaIsolation)
		// No assertion - just verifying it doesn't panic
	})
}

func TestManager_WithNilRegistry(t *testing.T) {
	// Should use DefaultRegistry when nil is passed
	mgr := NewManager(nil)
	if mgr == nil {
		t.Fatal("expected manager to be created")
	}
}

func TestManager_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mgr := NewManager(nil)
	p := newMockProvider("slow")
	p.clients["default"] = "client"

	_ = mgr.AddProvider("slow", p, ProviderConfig{Config: map[string]any{}})

	// Should work with timeout context
	err := mgr.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = mgr.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
