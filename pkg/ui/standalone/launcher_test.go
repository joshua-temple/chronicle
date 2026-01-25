package standalone

import (
	"context"
	"testing"
)

func TestLauncher_IsRunning(t *testing.T) {
	checker := NewHealthChecker()
	launcher := NewLauncher(checker)

	t.Run("returns false when no daemon running", func(t *testing.T) {
		if launcher.IsRunning("test-project") {
			t.Error("expected IsRunning to return false for non-existent project")
		}
	})

	t.Run("returns false for different project ID", func(t *testing.T) {
		if launcher.IsRunning("different-project") {
			t.Error("expected IsRunning to return false for different project")
		}
	})
}

func TestLauncher_GetPort(t *testing.T) {
	checker := NewHealthChecker()
	launcher := NewLauncher(checker)

	t.Run("returns 0 when no daemon running", func(t *testing.T) {
		port := launcher.GetPort("test-project")
		if port != 0 {
			t.Errorf("expected GetPort to return 0, got %d", port)
		}
	})

	t.Run("returns 0 for different project ID", func(t *testing.T) {
		port := launcher.GetPort("different-project")
		if port != 0 {
			t.Errorf("expected GetPort to return 0, got %d", port)
		}
	})
}

func TestLauncher_FindAvailablePort(t *testing.T) {
	checker := NewHealthChecker()
	launcher := NewLauncher(checker)

	t.Run("finds available port", func(t *testing.T) {
		port, err := launcher.findAvailablePort()
		if err != nil {
			t.Fatalf("failed to find available port: %v", err)
		}
		if port <= 0 {
			t.Errorf("expected positive port number, got %d", port)
		}
	})

	t.Run("finds different ports on subsequent calls", func(t *testing.T) {
		port1, err := launcher.findAvailablePort()
		if err != nil {
			t.Fatalf("failed to find first port: %v", err)
		}

		port2, err := launcher.findAvailablePort()
		if err != nil {
			t.Fatalf("failed to find second port: %v", err)
		}

		// Ports should be different (though not guaranteed, very likely)
		if port1 == port2 {
			t.Logf("Warning: got same port twice (%d), may be expected but unlikely", port1)
		}
	})
}

func TestLauncher_Stop(t *testing.T) {
	checker := NewHealthChecker()
	launcher := NewLauncher(checker)
	ctx := context.Background()

	t.Run("returns error when no daemon running", func(t *testing.T) {
		err := launcher.Stop(ctx, "nonexistent-project")
		if err == nil {
			t.Error("expected error when stopping non-existent daemon")
		}
	})
}

func TestLauncher_SetBinaryPath(t *testing.T) {
	checker := NewHealthChecker()
	launcher := NewLauncher(checker)

	t.Run("sets binary path", func(t *testing.T) {
		customPath := "/custom/path/to/chronicle"
		launcher.SetBinaryPath(customPath)

		if launcher.binaryPath != customPath {
			t.Errorf("expected binary path %s, got %s", customPath, launcher.binaryPath)
		}
	})

	t.Run("allows changing binary path", func(t *testing.T) {
		firstPath := "/first/path"
		secondPath := "/second/path"

		launcher.SetBinaryPath(firstPath)
		if launcher.binaryPath != firstPath {
			t.Errorf("expected binary path %s, got %s", firstPath, launcher.binaryPath)
		}

		launcher.SetBinaryPath(secondPath)
		if launcher.binaryPath != secondPath {
			t.Errorf("expected binary path %s, got %s", secondPath, launcher.binaryPath)
		}
	})
}

func TestNewLauncher(t *testing.T) {
	t.Run("creates launcher with health checker", func(t *testing.T) {
		checker := NewHealthChecker()
		launcher := NewLauncher(checker)

		if launcher == nil {
			t.Fatal("expected launcher to be non-nil")
		}

		if launcher.checker != checker {
			t.Error("launcher health checker doesn't match provided checker")
		}

		if launcher.processes == nil {
			t.Error("expected processes map to be initialized")
		}

		if launcher.ports == nil {
			t.Error("expected ports map to be initialized")
		}

		if launcher.binaryPath != "chronicle" {
			t.Errorf("expected default binary path 'chronicle', got %s", launcher.binaryPath)
		}
	})

	t.Run("creates independent launchers", func(t *testing.T) {
		checker := NewHealthChecker()
		launcher1 := NewLauncher(checker)
		launcher2 := NewLauncher(checker)

		if launcher1 == launcher2 {
			t.Error("expected different launcher instances")
		}

		// Verify maps are initialized and independent by adding to one
		launcher1.ports["test"] = 8080
		if _, exists := launcher2.ports["test"]; exists {
			t.Error("expected ports maps to be independent")
		}
	})
}

func TestLauncher_ThreadSafety(t *testing.T) {
	checker := NewHealthChecker()
	launcher := NewLauncher(checker)

	t.Run("concurrent IsRunning calls", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				for j := 0; j < 100; j++ {
					launcher.IsRunning("test-project")
				}
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent GetPort calls", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				for j := 0; j < 100; j++ {
					launcher.GetPort("test-project")
				}
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent SetBinaryPath calls", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				for j := 0; j < 100; j++ {
					launcher.SetBinaryPath("/path/to/binary")
				}
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
