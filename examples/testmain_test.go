package examples

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/infrastructure"
)

// Shared infrastructure for all tests in this file.
// In a real project, this would typically be defined at the package level.
var testInfra *infrastructure.TestContainersProvider

// TestMain demonstrates using shared infrastructure across all tests in a package.
// Infrastructure is started once before any tests run and stopped after all tests complete.
//
// Usage:
//
//	func TestMain(m *testing.M) {
//	    provider := infrastructure.NewTestContainersProvider().WithTestMain(m)
//	    provider.AddService(myService)
//	    ctx := context.Background()
//	    stop := provider.Start(ctx)
//	    os.Exit(stop(ctx))
//	}
func TestMain(m *testing.M) {
	// Check for short mode by inspecting os.Args directly
	// (testing.Short() requires flag.Parse() which m.Run() calls)
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "--test.short" {
			os.Exit(m.Run())
		}
	}

	// Define infrastructure requirements
	redis := infrastructure.NewServiceRequirement("redis", "redis:7")
	infrastructure.AddPort(&redis, "default", 6379)
	infrastructure.BuildPortWaitStrategy(&redis, 6379, 30*time.Second)

	postgres := infrastructure.NewServiceRequirement("postgres", "postgres:15")
	infrastructure.AddPort(&postgres, "default", 5432)
	postgres.Environment = map[string]string{
		"POSTGRES_USER":     "test",
		"POSTGRES_PASSWORD": "test",
		"POSTGRES_DB":       "testdb",
	}
	infrastructure.BuildPortWaitStrategy(&postgres, 5432, 30*time.Second)

	// Create provider with TestMain lifecycle
	testInfra = infrastructure.NewTestContainersProvider().WithTestMain(m)
	if err := testInfra.AddService(redis); err != nil {
		log.Fatalf("Failed to add redis service: %v", err)
	}
	if err := testInfra.AddService(postgres); err != nil {
		log.Fatalf("Failed to add postgres service: %v", err)
	}

	// Start infrastructure, run tests, stop infrastructure
	ctx := context.Background()
	stop := testInfra.Start(ctx)
	os.Exit(stop(ctx))
}

// TestWithSharedRedis demonstrates accessing shared Redis infrastructure.
func TestWithSharedRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Access the shared infrastructure
	endpoint, err := testInfra.ServiceEndpoint("redis", "default")
	if err != nil {
		t.Fatalf("Failed to get Redis endpoint: %v", err)
	}

	t.Logf("Redis available at: %s", endpoint)

	// In a real test, you would connect to Redis and perform operations
	// client := redis.NewClient(&redis.Options{Addr: endpoint})
	// ...
}

// TestWithSharedPostgres demonstrates accessing shared Postgres infrastructure.
func TestWithSharedPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Access the shared infrastructure
	endpoint, err := testInfra.ServiceEndpoint("postgres", "default")
	if err != nil {
		t.Fatalf("Failed to get Postgres endpoint: %v", err)
	}

	t.Logf("Postgres available at: %s", endpoint)

	// In a real test, you would connect to Postgres and perform operations
	// db, err := sql.Open("postgres", fmt.Sprintf("postgres://test:test@%s/testdb?sslmode=disable", endpoint))
	// ...
}

// TestMultipleServicesInteraction demonstrates tests that use multiple services.
func TestMultipleServicesInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Both services are available from the shared infrastructure
	redisEndpoint, err := testInfra.ServiceEndpoint("redis", "default")
	if err != nil {
		t.Fatalf("Failed to get Redis endpoint: %v", err)
	}

	pgEndpoint, err := testInfra.ServiceEndpoint("postgres", "default")
	if err != nil {
		t.Fatalf("Failed to get Postgres endpoint: %v", err)
	}

	t.Logf("Services available - Redis: %s, Postgres: %s", redisEndpoint, pgEndpoint)

	// In a real test, you would test interactions between services
	// For example, caching database queries in Redis
}
