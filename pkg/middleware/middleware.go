package middleware

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/joshua-temple/chronicle/pkg/context"
)

// Runner executes a component and returns an error.
type Runner func(ctx context.Context) error

// Middleware wraps a Runner to add cross-cutting behavior.
type Middleware func(next Runner) Runner

// Chain combines multiple middlewares into a single middleware.
// The first middleware in the list is the outermost (executes first).
func Chain(middlewares ...Middleware) Middleware {
	return func(next Runner) Runner {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// Logging creates a middleware that logs component execution.
func Logging() Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			name := ctx.ComponentName()
			ctx.Log(context.LogInfo, "Starting %s", name)
			start := time.Now()

			err := next(ctx)

			duration := time.Since(start)
			if err != nil {
				ctx.Log(context.LogError, "Failed %s after %v: %v", name, duration, err)
			} else {
				ctx.Log(context.LogInfo, "Completed %s in %v", name, duration)
			}
			return err
		}
	}
}

// Tracing creates a middleware that adds distributed tracing spans.
func Tracing() Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			name := ctx.ComponentName()
			spanCtx := ctx.WithSpan(name)

			ctx.Narrate(context.NarrativeDetail, fmt.Sprintf("Starting %s", name), map[string]any{
				"trace_id": spanCtx.Trace().TraceID.String(),
				"span_id":  spanCtx.Trace().SpanID.String(),
			})

			err := next(spanCtx)

			ctx.Narrate(context.NarrativeDetail, fmt.Sprintf("Completed %s", name), map[string]any{
				"duration": spanCtx.Trace().Duration().String(),
				"error":    err,
			})

			return err
		}
	}
}

// MetricsCollector is an interface for collecting execution metrics.
type MetricsCollector interface {
	RecordExecution(component string, duration time.Duration, success bool)
	RecordRetry(component string, attempt int)
	RecordTimeout(component string, duration time.Duration)
}

// Metrics creates a middleware that records execution metrics.
func Metrics(collector MetricsCollector) Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			name := ctx.ComponentName()
			start := time.Now()

			err := next(ctx)

			duration := time.Since(start)
			collector.RecordExecution(name, duration, err == nil)

			return err
		}
	}
}

// BackoffStrategy defines how to calculate retry delays.
type BackoffStrategy interface {
	Delay(attempt int) time.Duration
}

// ConstantBackoff returns the same delay for each retry.
type ConstantBackoff struct {
	Delay_ time.Duration
}

// Delay returns the constant delay.
func (b ConstantBackoff) Delay(attempt int) time.Duration {
	return b.Delay_
}

// ExponentialBackoff returns exponentially increasing delays.
type ExponentialBackoff struct {
	Initial    time.Duration
	Multiplier float64
	MaxDelay   time.Duration
	Jitter     bool
}

// Delay returns the exponentially calculated delay.
func (b ExponentialBackoff) Delay(attempt int) time.Duration {
	delay := float64(b.Initial) * math.Pow(b.Multiplier, float64(attempt))
	if b.MaxDelay > 0 && time.Duration(delay) > b.MaxDelay {
		delay = float64(b.MaxDelay)
	}
	if b.Jitter {
		// Add up to 25% jitter
		jitter := delay * 0.25 * rand.Float64()
		delay += jitter
	}
	return time.Duration(delay)
}

// LinearBackoff returns linearly increasing delays.
type LinearBackoff struct {
	Initial   time.Duration
	Increment time.Duration
	MaxDelay  time.Duration
}

// Delay returns the linearly calculated delay.
func (b LinearBackoff) Delay(attempt int) time.Duration {
	delay := b.Initial + time.Duration(attempt)*b.Increment
	if b.MaxDelay > 0 && delay > b.MaxDelay {
		return b.MaxDelay
	}
	return delay
}

// RetryConfig configures the retry middleware.
type RetryConfig struct {
	MaxRetries int
	Backoff    BackoffStrategy
	// RetryOn is a function that determines if an error should be retried.
	// If nil, all errors are retried.
	RetryOn func(error) bool
}

// DefaultRetryConfig returns a reasonable default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		Backoff: ExponentialBackoff{
			Initial:    100 * time.Millisecond,
			Multiplier: 2.0,
			MaxDelay:   5 * time.Second,
			Jitter:     true,
		},
	}
}

// Retry creates a middleware that retries failed executions.
func Retry(config RetryConfig) Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			name := ctx.ComponentName()
			var lastErr error

			for attempt := 0; attempt <= config.MaxRetries; attempt++ {
				if attempt > 0 {
					delay := config.Backoff.Delay(attempt - 1)
					ctx.Log(context.LogDebug, "Retry %d/%d for %s after %v", attempt, config.MaxRetries, name, delay)
					time.Sleep(delay)
				}

				lastErr = next(ctx)
				if lastErr == nil {
					return nil
				}

				// Check if we should retry this error
				if config.RetryOn != nil && !config.RetryOn(lastErr) {
					return lastErr
				}
			}

			return fmt.Errorf("failed after %d retries: %w", config.MaxRetries, lastErr)
		}
	}
}

// Timeout creates a middleware that enforces execution timeouts.
func Timeout(timeout time.Duration) Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			done := make(chan error, 1)

			go func() {
				done <- next(ctx)
			}()

			select {
			case err := <-done:
				return err
			case <-time.After(timeout):
				return &TimeoutError{
					Component: ctx.ComponentName(),
					Timeout:   timeout,
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// TimeoutError indicates a component execution timed out.
type TimeoutError struct {
	Component string
	Timeout   time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("component %s timed out after %v", e.Component, e.Timeout)
}

// Recover creates a middleware that recovers from panics.
func Recover() Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					name := ctx.ComponentName()
					ctx.Log(context.LogError, "Panic in %s: %v", name, r)
					err = &PanicError{
						Component: name,
						Value:     r,
					}
				}
			}()
			return next(ctx)
		}
	}
}

// PanicError wraps a recovered panic value.
type PanicError struct {
	Component string
	Value     any
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic in component %s: %v", e.Component, e.Value)
}

// Condition creates a middleware that conditionally runs the component.
func Condition(predicate func(ctx context.Context) bool) Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			if !predicate(ctx) {
				ctx.Log(context.LogInfo, "Skipping %s: condition not met", ctx.ComponentName())
				return nil
			}
			return next(ctx)
		}
	}
}

// NoOpMiddleware is a middleware that does nothing.
func NoOpMiddleware() Middleware {
	return func(next Runner) Runner {
		return next
	}
}

// Before creates a middleware that runs a function before the component.
func Before(fn func(ctx context.Context) error) Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			if err := fn(ctx); err != nil {
				return err
			}
			return next(ctx)
		}
	}
}

// After creates a middleware that runs a function after the component.
func After(fn func(ctx context.Context, err error) error) Middleware {
	return func(next Runner) Runner {
		return func(ctx context.Context) error {
			err := next(ctx)
			return fn(ctx, err)
		}
	}
}

// InMemoryMetricsCollector is a simple metrics collector for testing.
type InMemoryMetricsCollector struct {
	Executions []ExecutionRecord
	Retries    []RetryRecord
	Timeouts   []TimeoutRecord
}

// ExecutionRecord records a single execution.
type ExecutionRecord struct {
	Component string
	Duration  time.Duration
	Success   bool
}

// RetryRecord records a retry event.
type RetryRecord struct {
	Component string
	Attempt   int
}

// TimeoutRecord records a timeout event.
type TimeoutRecord struct {
	Component string
	Duration  time.Duration
}

// NewInMemoryMetricsCollector creates a new InMemoryMetricsCollector.
func NewInMemoryMetricsCollector() *InMemoryMetricsCollector {
	return &InMemoryMetricsCollector{
		Executions: make([]ExecutionRecord, 0),
		Retries:    make([]RetryRecord, 0),
		Timeouts:   make([]TimeoutRecord, 0),
	}
}

// RecordExecution records an execution.
func (c *InMemoryMetricsCollector) RecordExecution(component string, duration time.Duration, success bool) {
	c.Executions = append(c.Executions, ExecutionRecord{
		Component: component,
		Duration:  duration,
		Success:   success,
	})
}

// RecordRetry records a retry.
func (c *InMemoryMetricsCollector) RecordRetry(component string, attempt int) {
	c.Retries = append(c.Retries, RetryRecord{
		Component: component,
		Attempt:   attempt,
	})
}

// RecordTimeout records a timeout.
func (c *InMemoryMetricsCollector) RecordTimeout(component string, duration time.Duration) {
	c.Timeouts = append(c.Timeouts, TimeoutRecord{
		Component: component,
		Duration:  duration,
	})
}
