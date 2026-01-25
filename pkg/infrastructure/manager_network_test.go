package infrastructure

import (
	"context"
	"testing"
)

func TestManager_NetworkName(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)

	// Before start, network name should be empty
	if mgr.NetworkName() != "" {
		t.Errorf("NetworkName() before start = %q, want empty", mgr.NetworkName())
	}
}

func TestManager_WithNetworkEnabled(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry, WithNetworkEnabled(true))

	if !mgr.IsNetworkEnabled() {
		t.Error("WithNetworkEnabled(true) did not enable networking")
	}
}

func TestManager_NetworkDisabledByDefault(t *testing.T) {
	registry := NewRegistry()
	mgr := NewManager(registry)

	if mgr.IsNetworkEnabled() {
		t.Error("Network should be disabled by default")
	}
}

func TestManager_NetworkLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	registry := NewRegistry()
	mgr := NewManager(registry, WithNetworkEnabled(true))

	// Before start, network name should be empty
	if mgr.NetworkName() != "" {
		t.Error("NetworkName() should be empty before Start()")
	}

	// Start manager
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// After start, network name should be set (testcontainers uses UUID)
	netName := mgr.NetworkName()
	if netName == "" {
		t.Error("NetworkName() should be set after Start()")
	}
	// Verify it's a valid UUID format (8-4-4-4-12 hex characters)
	if len(netName) != 36 || netName[8] != '-' || netName[13] != '-' || netName[18] != '-' || netName[23] != '-' {
		t.Errorf("NetworkName() = %q, expected UUID format", netName)
	}

	// Stop manager
	if err := mgr.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// After stop, network name should be cleared
	if mgr.NetworkName() != "" {
		t.Error("NetworkName() should be empty after Stop()")
	}
}

func TestManager_NetworkNotCreatedWhenDisabled(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	mgr := NewManager(registry) // Default: network disabled

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := mgr.Stop(ctx); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	}()

	if mgr.NetworkName() != "" {
		t.Error("NetworkName() should be empty when network is disabled")
	}
}
