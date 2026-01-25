//go:build integration
// +build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/joshua-temple/chronicle/pkg/infrastructure/testcontainers"
)

// TestInfrastructureIntegration demonstrates using Chronicle's infrastructure
// management in an actual integration test scenario.
func TestInfrastructureIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Create and start infrastructure manager
	manager := infrastructure.NewManager(nil)
	manager.SetDefaultReuse(infrastructure.ReuseWithFlush)

	// Configure PostgreSQL
	err := manager.Configure(infrastructure.ProviderConfig{
		Name:     "db",
		Provider: "postgres",
		Config: map[string]any{
			"image":    "postgres:15-alpine",
			"database": "testdb",
			"username": "postgres",
			"password": "postgres",
		},
		Reuse: infrastructure.ReuseWithFlush,
		Flush: infrastructure.FlushConfig{
			Strategy: "truncate",
		},
	})
	if err != nil {
		t.Fatalf("Failed to configure postgres: %v", err)
	}

	// Configure Redis
	err = manager.Configure(infrastructure.ProviderConfig{
		Name:     "cache",
		Provider: "redis",
		Config: map[string]any{
			"image": "redis:7-alpine",
		},
		Reuse: infrastructure.ReuseWithFlush,
		Flush: infrastructure.FlushConfig{
			Strategy: "flushdb",
		},
	})
	if err != nil {
		t.Fatalf("Failed to configure redis: %v", err)
	}

	// Start all infrastructure
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Failed to start infrastructure: %v", err)
	}
	defer func() {
		if err := manager.Stop(context.Background()); err != nil {
			t.Logf("Warning: failed to stop infrastructure: %v", err)
		}
	}()

	// Wait for healthy
	if !manager.AllHealthy(ctx) {
		t.Fatal("Infrastructure not healthy")
	}

	// Run test scenarios
	t.Run("Scenario1_DatabaseOperations", func(t *testing.T) {
		// Get database client
		dbProvider, ok := manager.Provider("db")
		if !ok {
			t.Fatal("Database provider not found")
		}

		pgProvider := dbProvider.(*testcontainers.PostgresProvider)
		db := pgProvider.DB()

		// Create table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				email TEXT UNIQUE NOT NULL
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Insert test data
		_, err = db.ExecContext(ctx, `
			INSERT INTO users (name, email) VALUES ('Test User', 'test@example.com')
		`)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		// Query data
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 user, got %d", count)
		}
	})

	t.Run("Scenario2_CacheOperations", func(t *testing.T) {
		// Get Redis client
		cacheProvider, ok := manager.Provider("cache")
		if !ok {
			t.Fatal("Cache provider not found")
		}

		redisProvider := cacheProvider.(*testcontainers.RedisProvider)
		redis := redisProvider.RedisClient()

		// Set value
		err := redis.Set(ctx, "test-key", "test-value", 0).Err()
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		// Get value
		val, err := redis.Get(ctx, "test-key").Result()
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if val != "test-value" {
			t.Errorf("Expected 'test-value', got %s", val)
		}
	})

	// Reset for next scenario (demonstrates reuse behavior)
	t.Run("ResetBetweenScenarios", func(t *testing.T) {
		// Reset flushes data but keeps infrastructure running
		if err := manager.Reset(ctx); err != nil {
			t.Fatalf("Failed to reset: %v", err)
		}

		// Verify database was flushed
		dbProvider, _ := manager.Provider("db")
		pgProvider := dbProvider.(*testcontainers.PostgresProvider)
		db := pgProvider.DB()

		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query after reset: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 users after reset, got %d", count)
		}

		// Verify Redis was flushed
		cacheProvider, _ := manager.Provider("cache")
		redisProvider := cacheProvider.(*testcontainers.RedisProvider)
		redis := redisProvider.RedisClient()

		_, err = redis.Get(ctx, "test-key").Result()
		if err == nil {
			t.Error("Expected key to be deleted after reset")
		}
	})
}

// TestSecretResolutionInConfig demonstrates resolving secrets in infrastructure config.
func TestSecretResolutionInConfig(t *testing.T) {
	ctx := context.Background()

	// Set up test environment
	t.Setenv("POSTGRES_PASSWORD", "test-password-123")
	t.Setenv("REDIS_PORT", "6379")

	// Create secret provider
	provider := config.NewChainedSecretProvider(
		config.NewStaticSecretProvider(map[string]string{
			"db-username": "chronicle_user",
		}),
		config.NewEnvSecretProvider(),
	)

	resolver := config.NewVariableResolver(provider).WithFallbackToEnv(true)

	// Test config with variables
	cfg := &config.Config{
		Infrastructure: map[string]config.InfraConfig{
			"db": {
				Image: "postgres:15",
				Env: map[string]string{
					"POSTGRES_USER":     "${secrets.db-username}",
					"POSTGRES_PASSWORD": "${POSTGRES_PASSWORD}",
				},
			},
		},
	}

	// Resolve variables
	err := config.ResolveInConfig(ctx, cfg, resolver)
	if err != nil {
		t.Fatalf("Failed to resolve config: %v", err)
	}

	// Verify resolution
	dbConfig := cfg.Infrastructure["db"]
	if dbConfig.Env["POSTGRES_USER"] != "chronicle_user" {
		t.Errorf("Expected 'chronicle_user', got %s", dbConfig.Env["POSTGRES_USER"])
	}
	if dbConfig.Env["POSTGRES_PASSWORD"] != "test-password-123" {
		t.Errorf("Expected 'test-password-123', got %s", dbConfig.Env["POSTGRES_PASSWORD"])
	}
}

// TestReuseManagerPersistence demonstrates the reuse manager's persistence capabilities.
func TestReuseManagerPersistence(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create first manager and entry
	rm1 := infrastructure.NewReuseManager()
	rm1.SetStorePath(tmpDir)

	reuseConfig := infrastructure.ReuseConfig{
		Enabled: true,
		TTL:     1 * time.Hour,
		Key:     "test-persistence",
		Config: map[string]any{
			"image": "postgres:15",
		},
	}

	_, existed, err := rm1.GetOrCreate(ctx, "postgres", reuseConfig)
	if err != nil {
		t.Fatalf("Failed to create entry: %v", err)
	}
	if existed {
		t.Error("Expected new entry")
	}

	// Update with endpoints
	err = rm1.Update("test-persistence", map[string]string{
		"default": "localhost:5432",
	})
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Save state
	if err := rm1.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Create second manager and load
	rm2 := infrastructure.NewReuseManager()
	rm2.SetStorePath(tmpDir)
	if err := rm2.Load(); err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify entry was loaded
	entry, ok := rm2.Get("test-persistence")
	if !ok {
		t.Fatal("Entry not found after load")
	}
	if entry.Provider != "postgres" {
		t.Errorf("Expected provider 'postgres', got %s", entry.Provider)
	}
	if entry.Endpoints["default"] != "localhost:5432" {
		t.Errorf("Expected endpoint 'localhost:5432', got %s", entry.Endpoints["default"])
	}
}
