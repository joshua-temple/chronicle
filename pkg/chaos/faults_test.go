package chaos

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLatencyFault(t *testing.T) {
	t.Run("injects latency", func(t *testing.T) {
		fault := NewLatencyFault(50*time.Millisecond, 100*time.Millisecond)
		target := NewSimpleTarget("test", "type")

		start := time.Now()
		err := fault.Inject(context.Background(), target)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if elapsed < 50*time.Millisecond {
			t.Errorf("expected at least 50ms latency, got %v", elapsed)
		}
	})

	t.Run("respects probability", func(t *testing.T) {
		fault := NewLatencyFault(100*time.Millisecond, 100*time.Millisecond,
			WithLatencyProbability(0.0))
		target := NewSimpleTarget("test", "type")

		start := time.Now()
		err := fault.Inject(context.Background(), target)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if elapsed >= 100*time.Millisecond {
			t.Errorf("expected no latency with probability 0, got %v", elapsed)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		fault := NewLatencyFault(1*time.Second, 1*time.Second)
		target := NewSimpleTarget("test", "type")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := fault.Inject(ctx, target)
		if err == nil {
			t.Error("expected error due to context cancellation")
		}
	})

	t.Run("name option", func(t *testing.T) {
		fault := NewLatencyFault(0, 0, WithLatencyName("custom-latency"))
		if fault.Name() != "custom-latency" {
			t.Errorf("expected name 'custom-latency', got %s", fault.Name())
		}
	})
}

func TestErrorFault(t *testing.T) {
	t.Run("injects error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		fault := NewErrorFault(expectedErr, 1.0)
		target := NewSimpleTarget("test", "type")

		err := fault.Inject(context.Background(), target)

		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("no error with zero probability", func(t *testing.T) {
		fault := NewErrorFault(ErrChaosInjected, 0.0)
		target := NewSimpleTarget("test", "type")

		err := fault.Inject(context.Background(), target)

		if err != nil {
			t.Errorf("expected no error with probability 0, got %v", err)
		}
	})

	t.Run("name option", func(t *testing.T) {
		fault := NewErrorFault(ErrChaosInjected, 1.0, WithErrorName("custom-error"))
		if fault.Name() != "custom-error" {
			t.Errorf("expected name 'custom-error', got %s", fault.Name())
		}
	})
}

func TestTimeoutFault(t *testing.T) {
	t.Run("causes timeout", func(t *testing.T) {
		fault := NewTimeoutFault(1.0)
		target := NewSimpleTarget("test", "type")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := fault.Inject(ctx, target)

		if err == nil {
			t.Error("expected timeout error")
		}
		if !errors.Is(err, ErrLatencyExceeded) {
			t.Errorf("expected ErrLatencyExceeded, got %v", err)
		}
	})

	t.Run("no timeout with zero probability", func(t *testing.T) {
		fault := NewTimeoutFault(0.0)
		target := NewSimpleTarget("test", "type")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := fault.Inject(ctx, target)

		if err != nil {
			t.Errorf("expected no error with probability 0, got %v", err)
		}
	})

	t.Run("name option", func(t *testing.T) {
		fault := NewTimeoutFault(1.0, WithTimeoutName("custom-timeout"))
		if fault.Name() != "custom-timeout" {
			t.Errorf("expected name 'custom-timeout', got %s", fault.Name())
		}
	})
}

func TestPanicFault(t *testing.T) {
	t.Run("causes panic", func(t *testing.T) {
		fault := NewPanicFault("test panic", 1.0)
		target := NewSimpleTarget("test", "type")

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()

		_ = fault.Inject(context.Background(), target)
	})

	t.Run("no panic with zero probability", func(t *testing.T) {
		fault := NewPanicFault("test panic", 0.0)
		target := NewSimpleTarget("test", "type")

		err := fault.Inject(context.Background(), target)

		if err != nil {
			t.Errorf("expected no error with probability 0, got %v", err)
		}
	})

	t.Run("name option", func(t *testing.T) {
		fault := NewPanicFault("msg", 1.0, WithPanicName("custom-panic"))
		if fault.Name() != "custom-panic" {
			t.Errorf("expected name 'custom-panic', got %s", fault.Name())
		}
	})
}

func TestResourceExhaustionFault(t *testing.T) {
	t.Run("returns error", func(t *testing.T) {
		fault := NewResourceExhaustionFault("memory", 1.0)
		target := NewSimpleTarget("test", "type")

		err := fault.Inject(context.Background(), target)

		if err == nil {
			t.Error("expected error")
		}
		if !containsStr(err.Error(), "memory") {
			t.Errorf("expected error to mention 'memory', got %v", err)
		}
	})

	t.Run("no error with zero probability", func(t *testing.T) {
		fault := NewResourceExhaustionFault("memory", 0.0)
		target := NewSimpleTarget("test", "type")

		err := fault.Inject(context.Background(), target)

		if err != nil {
			t.Errorf("expected no error with probability 0, got %v", err)
		}
	})

	t.Run("name option", func(t *testing.T) {
		fault := NewResourceExhaustionFault("cpu", 1.0, WithResourceName("custom-resource"))
		if fault.Name() != "custom-resource" {
			t.Errorf("expected name 'custom-resource', got %s", fault.Name())
		}
	})
}

func TestNetworkPartitionFault(t *testing.T) {
	t.Run("partitions matching target", func(t *testing.T) {
		fault := NewNetworkPartitionFault([]string{"db", "cache"}, 1.0)

		target := NewSimpleTarget("db", "service")
		err := fault.Inject(context.Background(), target)
		if err == nil {
			t.Error("expected error for partitioned target")
		}

		target = NewSimpleTarget("api", "service")
		err = fault.Inject(context.Background(), target)
		if err != nil {
			t.Errorf("expected no error for non-partitioned target, got %v", err)
		}
	})

	t.Run("wildcard partitions all", func(t *testing.T) {
		fault := NewNetworkPartitionFault([]string{"*"}, 1.0)
		target := NewSimpleTarget("any-target", "type")

		err := fault.Inject(context.Background(), target)

		if err == nil {
			t.Error("expected error for wildcard partition")
		}
	})

	t.Run("name option", func(t *testing.T) {
		fault := NewNetworkPartitionFault(nil, 1.0, WithNetworkPartitionName("custom-partition"))
		if fault.Name() != "custom-partition" {
			t.Errorf("expected name 'custom-partition', got %s", fault.Name())
		}
	})
}

func TestCorruptionFault(t *testing.T) {
	t.Run("returns error", func(t *testing.T) {
		fault := NewCorruptionFault("payload", 1.0)
		target := NewSimpleTarget("test", "type")

		err := fault.Inject(context.Background(), target)

		if err == nil {
			t.Error("expected error")
		}
		if !containsStr(err.Error(), "payload") {
			t.Errorf("expected error to mention 'payload', got %v", err)
		}
	})

	t.Run("name option", func(t *testing.T) {
		fault := NewCorruptionFault("data", 1.0, WithCorruptionName("custom-corruption"))
		if fault.Name() != "custom-corruption" {
			t.Errorf("expected name 'custom-corruption', got %s", fault.Name())
		}
	})
}

func TestCompositeFault(t *testing.T) {
	t.Run("mode and applies all", func(t *testing.T) {
		applied := make([]string, 0)
		fault1 := &trackingFault{name: "fault1", applied: &applied}
		fault2 := &trackingFault{name: "fault2", applied: &applied}

		composite := NewCompositeFault([]Fault{fault1, fault2}, ModeAnd)
		target := NewSimpleTarget("test", "type")

		err := composite.Inject(context.Background(), target)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(applied) != 2 {
			t.Errorf("expected 2 faults applied, got %d", len(applied))
		}
	})

	t.Run("mode and stops on error", func(t *testing.T) {
		expectedErr := errors.New("stop here")
		fault1 := NewErrorFault(expectedErr, 1.0)
		fault2 := &trackingFault{name: "fault2", applied: new([]string)}

		composite := NewCompositeFault([]Fault{fault1, fault2}, ModeAnd)
		target := NewSimpleTarget("test", "type")

		err := composite.Inject(context.Background(), target)

		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("name option", func(t *testing.T) {
		composite := NewCompositeFault(nil, ModeAnd, WithCompositeName("custom-composite"))
		if composite.Name() != "custom-composite" {
			t.Errorf("expected name 'custom-composite', got %s", composite.Name())
		}
	})
}

func TestSequentialFault(t *testing.T) {
	t.Run("applies in sequence", func(t *testing.T) {
		applied := make([]string, 0)
		fault1 := &trackingFault{name: "fault1", applied: &applied}
		fault2 := &trackingFault{name: "fault2", applied: &applied}
		fault3 := &trackingFault{name: "fault3", applied: &applied}

		sequential := NewSequentialFault([]Fault{fault1, fault2, fault3}, 10*time.Millisecond)
		target := NewSimpleTarget("test", "type")

		start := time.Now()
		err := sequential.Inject(context.Background(), target)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(applied) != 3 {
			t.Errorf("expected 3 faults applied, got %d", len(applied))
		}
		if elapsed < 20*time.Millisecond {
			t.Errorf("expected at least 20ms for delays, got %v", elapsed)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		fault1 := &trackingFault{name: "fault1", applied: new([]string)}
		fault2 := &trackingFault{name: "fault2", applied: new([]string)}

		sequential := NewSequentialFault([]Fault{fault1, fault2}, 1*time.Second)
		target := NewSimpleTarget("test", "type")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := sequential.Inject(ctx, target)

		if err == nil {
			t.Error("expected error due to context cancellation")
		}
	})

	t.Run("name option", func(t *testing.T) {
		sequential := NewSequentialFault(nil, 0, WithSequentialName("custom-sequential"))
		if sequential.Name() != "custom-sequential" {
			t.Errorf("expected name 'custom-sequential', got %s", sequential.Name())
		}
	})
}

// trackingFault is a test helper that tracks when it's applied.
type trackingFault struct {
	name    string
	applied *[]string
}

func (f *trackingFault) Name() string {
	return f.name
}

func (f *trackingFault) Inject(_ context.Context, _ Target) error {
	*f.applied = append(*f.applied, f.name)
	return nil
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
