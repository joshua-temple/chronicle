package util

import (
	"context"
	"fmt"
	"time"

	"github.com/joshua-temple/chronicle/core"
)

// RetryWithBackoff executes a function repeatedly until it succeeds or exceeds the maximum retries
// It applies exponential backoff between retries
func RetryWithBackoff(ctx context.Context, tc *core.TestContext, fn func() error, maxRetries int, initialBackoff time.Duration) error {
	var err error
	backoff := initialBackoff

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			tc.Debug("Retry attempt %d/%d with backoff %v", i, maxRetries, backoff)
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				// Continue with retry
			}
			// Increase backoff for next attempt
			backoff = backoff * 2
		}

		// Attempt the operation
		err = fn()
		if err == nil {
			if i > 0 {
				tc.Debug("Operation succeeded after %d retries", i)
			}
			return nil
		}

		tc.Debug("Attempt %d failed: %v", i+1, err)
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

// WaitForCondition waits for a condition to be true, polling at the specified interval
func WaitForCondition(ctx context.Context, tc *core.TestContext, condition func() (bool, error), timeout, pollInterval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		satisfied, err := condition()
		if err != nil {
			return fmt.Errorf("error checking condition: %w", err)
		}

		if satisfied {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for condition after %v", timeout)
		}

		tc.Debug("Condition not yet satisfied, waiting %v before next check", pollInterval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// Continue checking
		}
	}
}

// CreateStepFunc creates a step function from a setup, task and expectation
func CreateStepFunc[T any](setup core.SetupFunc, task core.TaskFunc[T], expect core.ExpectFunc[T]) core.StepFunc[T] {
	return func(ctx context.Context, tc *core.TestContext) (T, error) {
		var empty T

		// Run setup if provided
		if setup != nil {
			if err := setup(ctx, tc); err != nil {
				return empty, fmt.Errorf("setup failed: %w", err)
			}
		}

		// Execute the task
		result, err := task(ctx, tc)
		if err != nil {
			return empty, fmt.Errorf("task failed: %w", err)
		}

		// Validate the result if an expectation is provided
		if expect != nil {
			if err := expect(ctx, tc, result); err != nil {
				return result, fmt.Errorf("expectation failed: %w", err)
			}
		}

		return result, nil
	}
}

// ComposeSteps runs multiple steps in sequence
func ComposeSteps(steps ...func(context.Context, *core.TestContext) error) core.RollupFunc {
	return func(ctx context.Context, tc *core.TestContext) error {
		for i, step := range steps {
			tc.Debug("Starting step %d/%d", i+1, len(steps))
			if err := step(ctx, tc); err != nil {
				return fmt.Errorf("step %d failed: %w", i+1, err)
			}
			tc.Debug("Completed step %d/%d", i+1, len(steps))
		}
		return nil
	}
}

// WithHook wraps a function with before and after hooks
func WithHook(fn func(context.Context, *core.TestContext) error, before, after func(context.Context, *core.TestContext) error) func(context.Context, *core.TestContext) error {
	return func(ctx context.Context, tc *core.TestContext) error {
		if before != nil {
			if err := before(ctx, tc); err != nil {
				return fmt.Errorf("before hook failed: %w", err)
			}
		}

		err := fn(ctx, tc)

		if after != nil {
			afterErr := after(ctx, tc)
			if err == nil {
				err = afterErr
			}
			if afterErr != nil {
				tc.Warning("After hook failed: %v", afterErr)
			}
		}

		return err
	}
}

// MeasureExecutionTime measures the execution time of a function
func MeasureExecutionTime(fn func(context.Context, *core.TestContext) error) func(context.Context, *core.TestContext) error {
	return func(ctx context.Context, tc *core.TestContext) error {
		start := time.Now()
		err := fn(ctx, tc)
		duration := time.Since(start)

		if err != nil {
			tc.Debug("Function failed after %v: %v", duration, err)
		} else {
			tc.Debug("Function completed in %v", duration)
		}

		// Store the execution time in the test context for later analysis
		tc.Store.Set("executionTime", duration)

		return err
	}
}

// WithTimeout runs a function with a specific timeout
func WithTimeout(fn func(context.Context, *core.TestContext) error, timeout time.Duration) func(context.Context, *core.TestContext) error {
	return func(ctx context.Context, tc *core.TestContext) error {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resChan := make(chan error, 1)
		go func() {
			resChan <- fn(timeoutCtx, tc)
		}()

		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("operation timed out after %v", timeout)
			}
			return timeoutCtx.Err()
		case res := <-resChan:
			return res
		}
	}
}

// SkipIfCondition skips a test if the given condition is true
func SkipIfCondition(tc *core.TestContext, condition bool, reason string) {
	if condition {
		tc.T.Skipf("Skipping test: %s", reason)
	}
}
