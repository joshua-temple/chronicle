package infrastructure

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// mockProvider is a test implementation of Provider.
type mockProvider struct {
	name         string
	status       atomic.Int32
	config       map[string]any
	initErr      error
	startErr     error
	stopErr      error
	flushErr     error
	clients      map[string]any
	healthReport HealthReport
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:    name,
		clients: make(map[string]any),
		healthReport: HealthReport{
			Healthy: true,
			Services: map[string]ServiceHealth{
				name: {Name: name, Status: "healthy"},
			},
		},
	}
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Initialize(ctx context.Context, config map[string]any) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.config = config
	return nil
}

func (m *mockProvider) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.status.Store(int32(StatusRunning))
	return nil
}

func (m *mockProvider) Stop(ctx context.Context) error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.status.Store(int32(StatusStopped))
	return nil
}

func (m *mockProvider) HealthCheck(ctx context.Context) HealthReport {
	return m.healthReport
}

func (m *mockProvider) Status() ProviderStatus {
	return ProviderStatus(m.status.Load())
}

func (m *mockProvider) Client(name string) (any, error) {
	if name == "" {
		name = "default"
	}
	client, ok := m.clients[name]
	if !ok {
		return nil, errors.New("client not found")
	}
	return client, nil
}

// Implement FlushableProvider
func (m *mockProvider) Flush(ctx context.Context) error {
	return m.flushErr
}

func (m *mockProvider) FlushWithConfig(ctx context.Context, config FlushConfig) error {
	return m.flushErr
}

func TestProviderStatus_String(t *testing.T) {
	tests := []struct {
		status   ProviderStatus
		expected string
	}{
		{StatusStopped, "stopped"},
		{StatusStarting, "starting"},
		{StatusRunning, "running"},
		{StatusUnhealthy, "unhealthy"},
		{StatusStopping, "stopping"},
		{ProviderStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestReuseBehavior_String(t *testing.T) {
	tests := []struct {
		behavior ReuseBehavior
		expected string
	}{
		{AlwaysFresh, "always_fresh"},
		{ReuseWithFlush, "flush"},
		{FullReuse, "full"},
		{ReuseBehavior(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.behavior.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestParseReuseBehavior(t *testing.T) {
	tests := []struct {
		input    string
		expected ReuseBehavior
	}{
		{"always_fresh", AlwaysFresh},
		{"flush", ReuseWithFlush},
		{"full", FullReuse},
		{"invalid", ReuseWithFlush}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseReuseBehavior(tt.input); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestIsolationLevel_String(t *testing.T) {
	tests := []struct {
		level    IsolationLevel
		expected string
	}{
		{NoIsolation, "none"},
		{DataIsolation, "data"},
		{SchemaIsolation, "schema"},
		{InstanceIsolation, "instance"},
		{IsolationLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestParseIsolationLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected IsolationLevel
	}{
		{"none", NoIsolation},
		{"data", DataIsolation},
		{"schema", SchemaIsolation},
		{"instance", InstanceIsolation},
		{"invalid", DataIsolation}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseIsolationLevel(tt.input); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	t.Run("register and create provider", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register("mock", func() Provider {
			return newMockProvider("mock")
		})

		provider, ok := registry.Create("mock")
		if !ok {
			t.Fatal("expected provider to be created")
		}
		if provider.Name() != "mock" {
			t.Errorf("expected name 'mock', got %s", provider.Name())
		}
	})

	t.Run("create unknown provider", func(t *testing.T) {
		registry := NewRegistry()
		_, ok := registry.Create("nonexistent")
		if ok {
			t.Error("expected create to fail for unknown provider")
		}
	})

	t.Run("available providers", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register("a", func() Provider { return newMockProvider("a") })
		registry.Register("b", func() Provider { return newMockProvider("b") })

		available := registry.Available()
		if len(available) != 2 {
			t.Errorf("expected 2 providers, got %d", len(available))
		}
	})
}

func TestHealthReport(t *testing.T) {
	report := HealthReport{
		Healthy: true,
		Message: "all services running",
		Services: map[string]ServiceHealth{
			"postgres": {
				Name:    "postgres",
				Status:  "healthy",
				Latency: 5 * time.Millisecond,
			},
			"redis": {
				Name:    "redis",
				Status:  "healthy",
				Latency: 1 * time.Millisecond,
			},
		},
	}

	if !report.Healthy {
		t.Error("expected report to be healthy")
	}
	if len(report.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(report.Services))
	}
}

func TestServiceHealth(t *testing.T) {
	health := ServiceHealth{
		Name:    "postgres",
		Status:  "healthy",
		Latency: 10 * time.Millisecond,
		Error:   nil,
	}

	if health.Name != "postgres" {
		t.Errorf("expected name 'postgres', got %s", health.Name)
	}
	if health.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", health.Status)
	}
	if health.Latency != 10*time.Millisecond {
		t.Errorf("expected latency 10ms, got %v", health.Latency)
	}
}

func TestFlushConfig(t *testing.T) {
	config := FlushConfig{
		Strategy: "truncate",
		Include:  []string{"users", "orders"},
		Exclude:  []string{"migrations"},
		Options: map[string]any{
			"cascade": true,
		},
	}

	if config.Strategy != "truncate" {
		t.Errorf("expected strategy 'truncate', got %s", config.Strategy)
	}
	if len(config.Include) != 2 {
		t.Errorf("expected 2 includes, got %d", len(config.Include))
	}
	if len(config.Exclude) != 1 {
		t.Errorf("expected 1 exclude, got %d", len(config.Exclude))
	}
}

func TestProviderConfig(t *testing.T) {
	config := ProviderConfig{
		Name:      "postgres",
		Provider:  "postgres",
		Config:    map[string]any{"host": "localhost", "port": 5432},
		Reuse:     ReuseWithFlush,
		Isolation: DataIsolation,
		Flush: FlushConfig{
			Strategy: "truncate",
		},
	}

	if config.Name != "postgres" {
		t.Errorf("expected name 'postgres', got %s", config.Name)
	}
	if config.Reuse != ReuseWithFlush {
		t.Errorf("expected ReuseWithFlush, got %v", config.Reuse)
	}
}
