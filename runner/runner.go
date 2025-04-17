package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/core"
)

// RunTest runs a single test with the given options
func RunTest(t *testing.T, testFunc core.TestRunner, opts ...core.TestOption) {
	// Create test context
	tc := core.NewTestContext(t, opts...)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), tc.Timeout)
	defer cancel()

	// Apply middleware
	runner := applyMiddleware(testFunc, tc.Middleware)

	// Initialize infrastructure if provided
	if tc.InfraProvider != nil {
		t.Log("Initializing infrastructure...")
		if err := tc.InfraProvider.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize infrastructure: %v", err)
		}

		// Ensure cleanup based on reuse behavior
		defer func() {
			if tc.ReuseBehavior == core.AlwaysFresh {
				t.Log("Cleaning up infrastructure...")
				if err := tc.InfraProvider.Stop(ctx); err != nil {
					t.Logf("Warning: Failed to stop infrastructure: %v", err)
				}
			} else if tc.ReuseBehavior == core.ReuseWithFlush {
				t.Log("Flushing infrastructure state...")
				if err := tc.InfraProvider.Flush(ctx); err != nil {
					t.Logf("Warning: Failed to flush infrastructure: %v", err)
				}
			}
		}()

		// Start infrastructure
		t.Log("Starting infrastructure...")
		if err := tc.InfraProvider.Start(ctx); err != nil {
			t.Fatalf("Failed to start infrastructure: %v", err)
		}
	}

	// Run the test
	t.Log("Running test...")
	if err := runner(ctx, tc); err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	t.Log("Test completed successfully")
}

// RunSuite runs a suite of tests
func RunSuite(t *testing.T, tests []core.TestRunner, opts ...core.TestOption) {
	for i, test := range tests {
		t.Run(fmt.Sprintf("Test_%d", i+1), func(t *testing.T) {
			RunTest(t, test, opts...)
		})
	}
}

// applyMiddleware applies middleware to a test runner
func applyMiddleware(runner core.TestRunner, middleware []core.MiddlewareFunc) core.TestRunner {
	// Apply middleware in reverse order so that the first middleware
	// in the list is the outermost wrapper
	for i := len(middleware) - 1; i >= 0; i-- {
		runner = middleware[i](runner)
	}
	return runner
}

// WithInfrastructureProvider sets the infrastructure provider
func WithInfrastructureProvider(provider core.InfrastructureProvider) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.InfraProvider = provider
		return tc
	}
}

// WithTimeout sets a test timeout
func WithTimeout(timeout time.Duration) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.Timeout = timeout
		return tc
	}
}

// WithLogLevel sets the log level
func WithLogLevel(level core.LogLevel) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.LogLevel = level
		return tc
	}
}

// WithReuseBehavior sets the infrastructure reuse behavior
func WithReuseBehavior(behavior core.ReuseBehavior) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.ReuseBehavior = behavior
		return tc
	}
}

// WithIsolationLevel sets the test isolation level
func WithIsolationLevel(level core.IsolationLevel) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.IsolationLevel = level
		return tc
	}
}

// WithFlags adds test feature flags
func WithFlags(flags ...string) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.Flags = append(tc.Flags, flags...)
		return tc
	}
}

// WithoutFlags excludes specific feature flags
func WithoutFlags(flags ...string) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.ExcludeFlags = append(tc.ExcludeFlags, flags...)
		return tc
	}
}

// WithMiddleware adds middleware to the test
func WithMiddleware(middleware ...core.MiddlewareFunc) core.TestOption {
	return func(tc *core.TestContext) *core.TestContext {
		tc.Middleware = append(tc.Middleware, middleware...)
		return tc
	}
}

// LoggingMiddleware creates a middleware that logs test execution
func LoggingMiddleware() core.MiddlewareFunc {
	return func(next core.TestRunner) core.TestRunner {
		return func(ctx context.Context, tc *core.TestContext) error {
			tc.Debug("Starting test execution")
			start := time.Now()
			err := next(ctx, tc)
			duration := time.Since(start)
			if err != nil {
				tc.Error("Test failed after %v: %v", duration, err)
			} else {
				tc.Debug("Completed test execution in %v", duration)
			}
			return err
		}
	}
}

// RetryMiddleware creates a middleware that retries a test on failure
func RetryMiddleware(maxRetries int, delay time.Duration) core.MiddlewareFunc {
	return func(next core.TestRunner) core.TestRunner {
		return func(ctx context.Context, tc *core.TestContext) error {
			var lastErr error
			for i := 0; i <= maxRetries; i++ {
				if i > 0 {
					tc.Debug("Retry attempt %d/%d after %v", i, maxRetries, delay)
					time.Sleep(delay)
				}

				lastErr = next(ctx, tc)
				if lastErr == nil {
					if i > 0 {
						tc.Debug("Successful after %d retries", i)
					}
					return nil
				}

				tc.Debug("Attempt %d failed: %v", i+1, lastErr)
			}
			return fmt.Errorf("test failed after %d retries: %w", maxRetries, lastErr)
		}
	}
}
