package examples

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/core"
	"github.com/joshua-temple/chronicle/infrastructure"
	"github.com/joshua-temple/chronicle/runner"
)

// ExampleTest demonstrates a simple test using the Chronicle framework
func TestExample(t *testing.T) {
	// Skip if not running as part of the example suite
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	// Define infrastructure requirements
	redisService := infrastructure.NewServiceRequirement("redis", "redis:latest")
	infrastructure.AddPort(&redisService, "default", 6379)
	infrastructure.BuildPortWaitStrategy(&redisService, 6379, 10*time.Second)

	// Create infrastructure provider (TestContainers)
	provider := infrastructure.NewTestContainersProvider()
	provider.AddService(redisService)

	// Run test with infrastructure
	runner.RunTest(t, func(ctx context.Context, tc *core.TestContext) error {
		// Setup: Prepare test data
		setupFn := func(ctx context.Context, tc *core.TestContext) error {
			tc.Info("Setting up test data...")
			// In a real test, this would seed data into Redis
			return nil
		}

		// Task: Perform an operation and return result
		taskFn := func(ctx context.Context, tc *core.TestContext) (string, error) {
			tc.Info("Executing task...")
			// Get Redis endpoint
			endpoint, err := tc.InfraProvider.ServiceEndpoint("redis", "default")
			if err != nil {
				return "", fmt.Errorf("failed to get Redis endpoint: %w", err)
			}

			// In a real test, this would interact with Redis
			return fmt.Sprintf("Connected to Redis at %s", endpoint), nil
		}

		// Expectation: Validate the result
		expectFn := func(ctx context.Context, tc *core.TestContext, result string) error {
			tc.Info("Validating result: %s", result)
			// In a real test, this would validate the result against expectations
			if result == "" {
				return fmt.Errorf("expected non-empty result")
			}
			return nil
		}

		// Setup
		if err := setupFn(ctx, tc); err != nil {
			return err
		}

		// Execute task and validate result
		result, err := core.Atomic[string]()(ctx, tc, taskFn, expectFn)
		if err != nil {
			return err
		}

		tc.Success("Test completed successfully with result: %s", result)
		return nil
	},
		// Options
		runner.WithInfrastructureProvider(provider),
		runner.WithTimeout(30*time.Second),
		runner.WithLogLevel(core.Info),
		runner.WithReuseBehavior(core.AlwaysFresh),
		runner.WithMiddleware(runner.LoggingMiddleware()))
}

// DockerComposeExample demonstrates using Docker Compose infrastructure
func TestDockerComposeExample(t *testing.T) {
	// Skip if not running as part of the example suite
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	// Create Docker Compose provider
	// This assumes a docker-compose.yml file exists in the examples directory
	provider := infrastructure.NewDockerComposeProvider("./docker-compose.yml", "chronicle-example")

	// Run test with infrastructure
	runner.RunTest(t, func(ctx context.Context, tc *core.TestContext) error {
		tc.Info("Running test with Docker Compose infrastructure")

		// Get service endpoint
		endpoint, err := tc.InfraProvider.ServiceEndpoint("redis", "6379")
		if err != nil {
			return fmt.Errorf("failed to get Redis endpoint: %w", err)
		}

		tc.Success("Connected to Redis at %s", endpoint)
		return nil
	},
		runner.WithInfrastructureProvider(provider),
		runner.WithReuseBehavior(core.ReuseWithFlush))
}
