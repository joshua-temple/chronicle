package infrastructure

import (
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
