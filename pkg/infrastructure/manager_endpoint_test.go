package infrastructure

import (
	"os"
	"testing"
)

func TestManager_EndpointRegistry(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)

	// Register an endpoint
	mgr.Endpoints().Register("postgres", Endpoint{
		Host:         "localhost",
		Port:         54321,
		InternalHost: "postgres",
		InternalPort: 5432,
	})

	// Retrieve via Manager
	ep, ok := mgr.Endpoint("postgres")
	if !ok {
		t.Fatal("Endpoint() returned false, want true")
	}

	if ep.Port != 54321 {
		t.Errorf("Endpoint().Port = %d, want 54321", ep.Port)
	}
}

func TestManager_Endpoint_NotFound(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)

	_, ok := mgr.Endpoint("nonexistent")
	if ok {
		t.Error("Endpoint() returned true for nonexistent, want false")
	}
}

func TestManager_EnvVars(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)

	mgr.Endpoints().Register("postgres", Endpoint{
		Host:         "localhost",
		Port:         54321,
		InternalHost: "postgres",
		InternalPort: 5432,
		Metadata: map[string]string{
			"user":     "testuser",
			"database": "testdb",
		},
	})

	env := mgr.EnvVars()

	tests := map[string]string{
		"POSTGRES_HOST":          "localhost",
		"POSTGRES_PORT":          "54321",
		"POSTGRES_INTERNAL_HOST": "postgres",
		"POSTGRES_INTERNAL_PORT": "5432",
		"POSTGRES_ADDRESS":       "localhost:54321",
		"POSTGRES_USER":          "testuser",
		"POSTGRES_DATABASE":      "testdb",
	}

	for key, want := range tests {
		got, ok := env[key]
		if !ok {
			t.Errorf("EnvVars() missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("EnvVars()[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestManager_EnvVars_MultipleServices(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)

	mgr.Endpoints().Register("postgres", Endpoint{Host: "localhost", Port: 5432})
	mgr.Endpoints().Register("redis", Endpoint{Host: "localhost", Port: 6379})

	env := mgr.EnvVars()

	if _, ok := env["POSTGRES_HOST"]; !ok {
		t.Error("EnvVars() missing POSTGRES_HOST")
	}
	if _, ok := env["REDIS_HOST"]; !ok {
		t.Error("EnvVars() missing REDIS_HOST")
	}
}

func TestManager_EnvVars_HyphenatedName(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)

	mgr.Endpoints().Register("my-service", Endpoint{Host: "localhost", Port: 8080})

	env := mgr.EnvVars()

	if _, ok := env["MY_SERVICE_HOST"]; !ok {
		t.Error("EnvVars() should convert hyphens to underscores: expected MY_SERVICE_HOST")
	}
}

func TestManager_SetEnvVars(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)
	mgr.Endpoints().Register("test-svc", Endpoint{
		Host: "localhost",
		Port: 9999,
	})

	err := mgr.SetEnvVars()
	if err != nil {
		t.Fatalf("SetEnvVars() error = %v", err)
	}

	// Verify variables were set
	if os.Getenv("TEST_SVC_HOST") != "localhost" {
		t.Error("TEST_SVC_HOST not set correctly")
	}
	if os.Getenv("TEST_SVC_PORT") != "9999" {
		t.Error("TEST_SVC_PORT not set correctly")
	}

	// Cleanup
	_ = os.Unsetenv("TEST_SVC_HOST")
	_ = os.Unsetenv("TEST_SVC_PORT")
	_ = os.Unsetenv("TEST_SVC_ADDRESS")
}
