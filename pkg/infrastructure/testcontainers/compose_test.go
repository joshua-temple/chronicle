package testcontainers

import (
	"testing"

	"github.com/joshua-temple/chronicle/pkg/infrastructure"
)

func TestComposeProvider_Name(t *testing.T) {
	p := NewComposeProvider("my-stack")

	if p.Name() != "my-stack" {
		t.Errorf("Name() = %q, want %q", p.Name(), "my-stack")
	}
}

func TestComposeProvider_ImplementsProvider(t *testing.T) {
	p := NewComposeProvider("my-stack")

	// Verify it implements Provider interface
	var _ infrastructure.Provider = p
}

func TestComposeProvider_ImplementsNetworkAwareProvider(t *testing.T) {
	p := NewComposeProvider("my-stack")

	// Verify it implements NetworkAwareProvider
	var _ infrastructure.NetworkAwareProvider = p
}
