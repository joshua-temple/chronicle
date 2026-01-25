package middleware

import (
	gocontext "context"
	"errors"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/context"
)

func newTestContext(name string) context.Context {
	ctx := context.New(gocontext.Background())
	ctx.SetComponentName(name)
	return ctx
}

func TestChain(t *testing.T) {
	t.Run("chains middlewares in correct order", func(t *testing.T) {
		var order []string

		m1 := func(next Runner) Runner {
			return func(ctx context.Context) error {
				order = append(order, "m1-before")
				err := next(ctx)
				order = append(order, "m1-after")
				return err
			}
		}

		m2 := func(next Runner) Runner {
			return func(ctx context.Context) error {
				order = append(order, "m2-before")
				err := next(ctx)
				order = append(order, "m2-after")
				return err
			}
		}

		chain := Chain(m1, m2)
		runner := chain(func(ctx context.Context) error {
			order = append(order, "handler")
			return nil
		})

		ctx := newTestContext("test")
		err := runner(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
		if len(order) != len(expected) {
			t.Fatalf("expected %d calls, got %d", len(expected), len(order))
		}
		for i, v := range expected {
			if order[i] != v {
				t.Errorf("position %d: expected %s, got %s", i, v, order[i])
			}
		}
	})

	t.Run("empty chain returns runner unchanged", func(t *testing.T) {
		called := false
		runner := func(ctx context.Context) error {
			called = true
			return nil
		}

		chain := Chain()
		wrapped := chain(runner)

		ctx := newTestContext("test")
		_ = wrapped(ctx)

		if !called {
			t.Error("runner should have been called")
		}
	})
}

func TestLogging(t *testing.T) {
	t.Run("logs success", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		runner := Logging()(func(ctx context.Context) error {
			return nil
		})

		err := runner(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logs := ctx.Logs()
		if len(logs) != 2 {
			t.Fatalf("expected 2 log entries, got %d", len(logs))
		}
		if logs[0].Level != context.LogInfo {
			t.Error("first log should be INFO")
		}
		if logs[1].Level != context.LogInfo {
			t.Error("second log should be INFO on success")
		}
	})

	t.Run("logs failure", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		expectedErr := errors.New("test error")
		runner := Logging()(func(ctx context.Context) error {
			return expectedErr
		})

		err := runner(ctx)
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}

		logs := ctx.Logs()
		if len(logs) != 2 {
			t.Fatalf("expected 2 log entries, got %d", len(logs))
		}
		if logs[1].Level != context.LogError {
			t.Error("second log should be ERROR on failure")
		}
	})
}

func TestTracing(t *testing.T) {
	t.Run("creates child span", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		originalTraceID := ctx.Trace().TraceID

		var childTraceID string
		runner := Tracing()(func(ctx context.Context) error {
			childTraceID = ctx.Trace().TraceID.String()
			return nil
		})

		_ = runner(ctx)

		// Trace ID should be the same (span changes, not trace)
		if childTraceID != originalTraceID.String() {
			t.Error("TraceID should remain the same in child span")
		}

		// Should have narrative entries
		narrative := ctx.Narrative()
		if len(narrative) < 2 {
			t.Errorf("expected at least 2 narrative entries, got %d", len(narrative))
		}
	})
}

func TestMetrics(t *testing.T) {
	t.Run("records successful execution", func(t *testing.T) {
		collector := NewInMemoryMetricsCollector()
		ctx := newTestContext("TestComponent")

		runner := Metrics(collector)(func(ctx context.Context) error {
			return nil
		})

		_ = runner(ctx)

		if len(collector.Executions) != 1 {
			t.Fatalf("expected 1 execution record, got %d", len(collector.Executions))
		}
		if !collector.Executions[0].Success {
			t.Error("execution should be marked as success")
		}
		if collector.Executions[0].Component != "TestComponent" {
			t.Errorf("wrong component name: %s", collector.Executions[0].Component)
		}
	})

	t.Run("records failed execution", func(t *testing.T) {
		collector := NewInMemoryMetricsCollector()
		ctx := newTestContext("TestComponent")

		runner := Metrics(collector)(func(ctx context.Context) error {
			return errors.New("error")
		})

		_ = runner(ctx)

		if len(collector.Executions) != 1 {
			t.Fatalf("expected 1 execution record, got %d", len(collector.Executions))
		}
		if collector.Executions[0].Success {
			t.Error("execution should be marked as failure")
		}
	})
}

func TestBackoffStrategies(t *testing.T) {
	t.Run("constant backoff", func(t *testing.T) {
		b := ConstantBackoff{Delay_: 100 * time.Millisecond}
		if b.Delay(0) != 100*time.Millisecond {
			t.Error("first delay incorrect")
		}
		if b.Delay(5) != 100*time.Millisecond {
			t.Error("fifth delay should be same")
		}
	})

	t.Run("exponential backoff", func(t *testing.T) {
		b := ExponentialBackoff{
			Initial:    100 * time.Millisecond,
			Multiplier: 2.0,
			MaxDelay:   1 * time.Second,
			Jitter:     false,
		}

		d0 := b.Delay(0)
		d1 := b.Delay(1)
		d2 := b.Delay(2)

		if d0 != 100*time.Millisecond {
			t.Errorf("delay(0) = %v, want 100ms", d0)
		}
		if d1 != 200*time.Millisecond {
			t.Errorf("delay(1) = %v, want 200ms", d1)
		}
		if d2 != 400*time.Millisecond {
			t.Errorf("delay(2) = %v, want 400ms", d2)
		}

		// Test max delay
		d10 := b.Delay(10)
		if d10 > 1*time.Second {
			t.Errorf("delay should be capped at 1s, got %v", d10)
		}
	})

	t.Run("linear backoff", func(t *testing.T) {
		b := LinearBackoff{
			Initial:   100 * time.Millisecond,
			Increment: 50 * time.Millisecond,
			MaxDelay:  300 * time.Millisecond,
		}

		if b.Delay(0) != 100*time.Millisecond {
			t.Error("first delay incorrect")
		}
		if b.Delay(1) != 150*time.Millisecond {
			t.Error("second delay incorrect")
		}
		if b.Delay(10) != 300*time.Millisecond {
			t.Error("large delay should be capped")
		}
	})
}

func TestRetry(t *testing.T) {
	t.Run("returns immediately on success", func(t *testing.T) {
		attempts := 0
		ctx := newTestContext("TestComponent")

		config := RetryConfig{
			MaxRetries: 3,
			Backoff:    ConstantBackoff{Delay_: 1 * time.Millisecond},
		}

		runner := Retry(config)(func(ctx context.Context) error {
			attempts++
			return nil
		})

		err := runner(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("retries on failure", func(t *testing.T) {
		attempts := 0
		ctx := newTestContext("TestComponent")

		config := RetryConfig{
			MaxRetries: 3,
			Backoff:    ConstantBackoff{Delay_: 1 * time.Millisecond},
		}

		runner := Retry(config)(func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return errors.New("fail")
			}
			return nil
		})

		err := runner(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("gives up after max retries", func(t *testing.T) {
		attempts := 0
		ctx := newTestContext("TestComponent")

		config := RetryConfig{
			MaxRetries: 2,
			Backoff:    ConstantBackoff{Delay_: 1 * time.Millisecond},
		}

		runner := Retry(config)(func(ctx context.Context) error {
			attempts++
			return errors.New("always fail")
		})

		err := runner(ctx)
		if err == nil {
			t.Error("expected error after max retries")
		}
		if attempts != 3 { // initial + 2 retries
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("respects RetryOn predicate", func(t *testing.T) {
		attempts := 0
		ctx := newTestContext("TestComponent")
		permanentErr := errors.New("permanent")

		config := RetryConfig{
			MaxRetries: 3,
			Backoff:    ConstantBackoff{Delay_: 1 * time.Millisecond},
			RetryOn: func(err error) bool {
				return err != permanentErr
			},
		}

		runner := Retry(config)(func(ctx context.Context) error {
			attempts++
			return permanentErr
		})

		err := runner(ctx)
		if err != permanentErr {
			t.Errorf("expected permanent error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt (no retry for permanent error), got %d", attempts)
		}
	})
}

func TestTimeout(t *testing.T) {
	t.Run("completes before timeout", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		runner := Timeout(100 * time.Millisecond)(func(ctx context.Context) error {
			return nil
		})

		err := runner(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("times out slow execution", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		runner := Timeout(10 * time.Millisecond)(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		err := runner(ctx)
		if err == nil {
			t.Fatal("expected timeout error")
		}

		var timeoutErr *TimeoutError
		if !errors.As(err, &timeoutErr) {
			t.Errorf("expected TimeoutError, got %T", err)
		}
		if timeoutErr.Component != "TestComponent" {
			t.Errorf("wrong component: %s", timeoutErr.Component)
		}
	})
}

func TestRecover(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		runner := Recover()(func(ctx context.Context) error {
			panic("test panic")
		})

		err := runner(ctx)
		if err == nil {
			t.Fatal("expected panic error")
		}

		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Errorf("expected PanicError, got %T", err)
		}
		if panicErr.Value != "test panic" {
			t.Errorf("wrong panic value: %v", panicErr.Value)
		}
	})

	t.Run("passes through normal errors", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		expectedErr := errors.New("normal error")
		runner := Recover()(func(ctx context.Context) error {
			return expectedErr
		})

		err := runner(ctx)
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestCondition(t *testing.T) {
	t.Run("runs when condition is true", func(t *testing.T) {
		ran := false
		ctx := newTestContext("TestComponent")

		runner := Condition(func(ctx context.Context) bool {
			return true
		})(func(ctx context.Context) error {
			ran = true
			return nil
		})

		_ = runner(ctx)
		if !ran {
			t.Error("runner should have been called")
		}
	})

	t.Run("skips when condition is false", func(t *testing.T) {
		ran := false
		ctx := newTestContext("TestComponent")

		runner := Condition(func(ctx context.Context) bool {
			return false
		})(func(ctx context.Context) error {
			ran = true
			return nil
		})

		err := runner(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ran {
			t.Error("runner should not have been called")
		}
	})
}

func TestBeforeAfter(t *testing.T) {
	t.Run("Before runs before handler", func(t *testing.T) {
		var order []string
		ctx := newTestContext("TestComponent")

		runner := Before(func(ctx context.Context) error {
			order = append(order, "before")
			return nil
		})(func(ctx context.Context) error {
			order = append(order, "handler")
			return nil
		})

		_ = runner(ctx)

		if len(order) != 2 || order[0] != "before" || order[1] != "handler" {
			t.Errorf("wrong order: %v", order)
		}
	})

	t.Run("Before error stops execution", func(t *testing.T) {
		handlerCalled := false
		ctx := newTestContext("TestComponent")
		beforeErr := errors.New("before error")

		runner := Before(func(ctx context.Context) error {
			return beforeErr
		})(func(ctx context.Context) error {
			handlerCalled = true
			return nil
		})

		err := runner(ctx)
		if err != beforeErr {
			t.Errorf("expected before error, got %v", err)
		}
		if handlerCalled {
			t.Error("handler should not have been called")
		}
	})

	t.Run("After runs after handler", func(t *testing.T) {
		var order []string
		ctx := newTestContext("TestComponent")

		runner := After(func(ctx context.Context, err error) error {
			order = append(order, "after")
			return err
		})(func(ctx context.Context) error {
			order = append(order, "handler")
			return nil
		})

		_ = runner(ctx)

		if len(order) != 2 || order[0] != "handler" || order[1] != "after" {
			t.Errorf("wrong order: %v", order)
		}
	})

	t.Run("After can transform error", func(t *testing.T) {
		ctx := newTestContext("TestComponent")
		handlerErr := errors.New("handler error")
		transformedErr := errors.New("transformed error")

		runner := After(func(ctx context.Context, err error) error {
			if err != nil {
				return transformedErr
			}
			return nil
		})(func(ctx context.Context) error {
			return handlerErr
		})

		err := runner(ctx)
		if err != transformedErr {
			t.Errorf("expected transformed error, got %v", err)
		}
	})
}

func TestNoOpMiddleware(t *testing.T) {
	called := false
	ctx := newTestContext("TestComponent")

	runner := NoOpMiddleware()(func(ctx context.Context) error {
		called = true
		return nil
	})

	_ = runner(ctx)
	if !called {
		t.Error("runner should have been called")
	}
}

func TestTimeoutError(t *testing.T) {
	err := &TimeoutError{
		Component: "TestComponent",
		Timeout:   5 * time.Second,
	}
	expected := "component TestComponent timed out after 5s"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}

func TestPanicError(t *testing.T) {
	err := &PanicError{
		Component: "TestComponent",
		Value:     "test panic",
	}
	expected := "panic in component TestComponent: test panic"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}
