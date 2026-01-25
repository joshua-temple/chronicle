package infrastructure

import "testing"

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
