//go:build integration

package infrastructure_test

import (
	"context"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/joshua-temple/chronicle/pkg/infrastructure/testcontainers"
)

func TestManager_SharedNetwork_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	registry := infrastructure.NewRegistry()
	registry.Register("postgres", func() infrastructure.Provider {
		return testcontainers.NewPostgresProvider()
	})

	mgr := infrastructure.NewManager(registry, infrastructure.WithNetworkEnabled(true))

	err := mgr.Configure(infrastructure.ProviderConfig{
		Name:     "postgres",
		Provider: "postgres",
		Config: map[string]any{
			"image":    "postgres:15",
			"user":     "test",
			"password": "test",
			"database": "testdb",
		},
	})
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer mgr.Stop(ctx)

	// Verify network was created
	if mgr.NetworkName() == "" {
		t.Error("NetworkName() is empty after Start()")
	}

	// Verify endpoint was registered
	ep, ok := mgr.Endpoint("postgres")
	if !ok {
		t.Fatal("Endpoint('postgres') not found")
	}

	if ep.Host == "" || ep.Port == 0 {
		t.Errorf("Endpoint has empty host/port: %+v", ep)
	}

	if ep.InternalHost != "postgres" {
		t.Errorf("Endpoint.InternalHost = %q, want %q", ep.InternalHost, "postgres")
	}

	// Verify env vars
	env := mgr.EnvVars()
	if env["POSTGRES_HOST"] == "" {
		t.Error("EnvVars() missing POSTGRES_HOST")
	}

	t.Logf("Network: %s", mgr.NetworkName())
	t.Logf("Endpoint: %+v", ep)
	t.Logf("EnvVars: %v", env)
}
