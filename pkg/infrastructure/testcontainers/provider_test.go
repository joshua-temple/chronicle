//go:build integration
// +build integration

package testcontainers

import (
	"context"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/infrastructure"
)

func TestContainerProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Run("lifecycle management", func(t *testing.T) {
		provider := NewContainerProvider("test")

		if provider.Name() != "test" {
			t.Errorf("expected name 'test', got %s", provider.Name())
		}

		if provider.Status() != infrastructure.StatusStopped {
			t.Error("expected status stopped")
		}
	})

	t.Run("client management", func(t *testing.T) {
		provider := NewContainerProvider("test")

		// Register a client
		provider.SetClient("default", "test-client")
		provider.SetClient("named", "named-client")

		// Get default client
		client, err := provider.Client("")
		if err != nil {
			t.Fatalf("failed to get default client: %v", err)
		}
		if client != "test-client" {
			t.Errorf("expected 'test-client', got %v", client)
		}

		// Get named client
		client, err = provider.Client("named")
		if err != nil {
			t.Fatalf("failed to get named client: %v", err)
		}
		if client != "named-client" {
			t.Errorf("expected 'named-client', got %v", client)
		}

		// Get non-existent client
		_, err = provider.Client("nonexistent")
		if err == nil {
			t.Error("expected error for non-existent client")
		}
	})
}

func TestPostgresProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	provider := NewPostgresProvider()

	t.Run("initialize", func(t *testing.T) {
		err := provider.Initialize(ctx, map[string]any{
			"image":    "postgres:15-alpine",
			"database": "testdb",
			"username": "postgres",
			"password": "postgres",
		})
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}
	})

	t.Run("start container", func(t *testing.T) {
		err := provider.Start(ctx)
		if err != nil {
			t.Fatalf("start failed: %v", err)
		}

		if provider.Status() != infrastructure.StatusRunning {
			t.Errorf("expected status running, got %v", provider.Status())
		}
	})

	t.Run("health check", func(t *testing.T) {
		report := provider.HealthCheck(ctx)
		if !report.Healthy {
			t.Errorf("expected healthy, got message: %s", report.Message)
		}
	})

	t.Run("get client", func(t *testing.T) {
		client, err := provider.Client("default")
		if err != nil {
			t.Fatalf("failed to get client: %v", err)
		}

		db := provider.DB()
		if db == nil {
			t.Fatal("expected db not nil")
		}

		if client != db {
			t.Error("expected client to be the same as DB()")
		}

		// Test connection
		var result int
		err = db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if result != 1 {
			t.Errorf("expected 1, got %d", result)
		}
	})

	t.Run("connection string", func(t *testing.T) {
		connStr, err := provider.ConnectionString(ctx)
		if err != nil {
			t.Fatalf("failed to get connection string: %v", err)
		}
		if connStr == "" {
			t.Error("expected non-empty connection string")
		}
	})

	t.Run("flush truncate", func(t *testing.T) {
		db := provider.DB()

		// Create a test table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS test_table (
				id SERIAL PRIMARY KEY,
				name TEXT
			)
		`)
		if err != nil {
			t.Fatalf("failed to create table: %v", err)
		}

		// Insert some data
		_, err = db.ExecContext(ctx, `INSERT INTO test_table (name) VALUES ('test')`)
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}

		// Verify data exists
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_table").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 row, got %d", count)
		}

		// Flush
		err = provider.Flush(ctx)
		if err != nil {
			t.Fatalf("flush failed: %v", err)
		}

		// Verify data is gone
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_table").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count after flush: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 rows after flush, got %d", count)
		}
	})

	t.Run("stop container", func(t *testing.T) {
		err := provider.Stop(ctx)
		if err != nil {
			t.Fatalf("stop failed: %v", err)
		}

		if provider.Status() != infrastructure.StatusStopped {
			t.Errorf("expected status stopped, got %v", provider.Status())
		}
	})
}

func TestRedisProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	provider := NewRedisProvider()

	t.Run("initialize", func(t *testing.T) {
		err := provider.Initialize(ctx, map[string]any{
			"image": "redis:7-alpine",
		})
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}
	})

	t.Run("start container", func(t *testing.T) {
		err := provider.Start(ctx)
		if err != nil {
			t.Fatalf("start failed: %v", err)
		}

		if provider.Status() != infrastructure.StatusRunning {
			t.Errorf("expected status running, got %v", provider.Status())
		}
	})

	t.Run("health check", func(t *testing.T) {
		report := provider.HealthCheck(ctx)
		if !report.Healthy {
			t.Errorf("expected healthy, got message: %s", report.Message)
		}
	})

	t.Run("get client", func(t *testing.T) {
		client, err := provider.Client("default")
		if err != nil {
			t.Fatalf("failed to get client: %v", err)
		}

		redisClient := provider.RedisClient()
		if redisClient == nil {
			t.Fatal("expected redis client not nil")
		}

		if client != redisClient {
			t.Error("expected client to be the same as RedisClient()")
		}

		// Test connection
		err = redisClient.Set(ctx, "test-key", "test-value", 0).Err()
		if err != nil {
			t.Fatalf("failed to set: %v", err)
		}

		val, err := redisClient.Get(ctx, "test-key").Result()
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if val != "test-value" {
			t.Errorf("expected 'test-value', got %s", val)
		}
	})

	t.Run("flush flushdb", func(t *testing.T) {
		redisClient := provider.RedisClient()

		// Set some data
		err := redisClient.Set(ctx, "key1", "value1", 0).Err()
		if err != nil {
			t.Fatalf("failed to set: %v", err)
		}

		// Verify data exists
		_, err = redisClient.Get(ctx, "key1").Result()
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		// Flush
		err = provider.Flush(ctx)
		if err != nil {
			t.Fatalf("flush failed: %v", err)
		}

		// Verify data is gone
		_, err = redisClient.Get(ctx, "key1").Result()
		if err == nil {
			t.Error("expected key to be deleted after flush")
		}
	})

	t.Run("flush pattern", func(t *testing.T) {
		redisClient := provider.RedisClient()

		// Set some data with different prefixes
		_ = redisClient.Set(ctx, "test:key1", "value1", 0).Err()
		_ = redisClient.Set(ctx, "test:key2", "value2", 0).Err()
		_ = redisClient.Set(ctx, "other:key1", "value1", 0).Err()

		// Flush only test:* pattern
		err := provider.FlushWithConfig(ctx, infrastructure.FlushConfig{
			Strategy: "pattern",
			Options:  map[string]any{"pattern": "test:*"},
		})
		if err != nil {
			t.Fatalf("flush pattern failed: %v", err)
		}

		// test:* keys should be gone
		_, err = redisClient.Get(ctx, "test:key1").Result()
		if err == nil {
			t.Error("expected test:key1 to be deleted")
		}

		// other:* keys should remain
		val, err := redisClient.Get(ctx, "other:key1").Result()
		if err != nil {
			t.Error("expected other:key1 to still exist")
		}
		if val != "value1" {
			t.Errorf("expected 'value1', got %s", val)
		}
	})

	t.Run("stop container", func(t *testing.T) {
		err := provider.Stop(ctx)
		if err != nil {
			t.Fatalf("stop failed: %v", err)
		}

		if provider.Status() != infrastructure.StatusStopped {
			t.Errorf("expected status stopped, got %v", provider.Status())
		}
	})
}

func TestProviderRegistration(t *testing.T) {
	t.Run("postgres registered", func(t *testing.T) {
		provider, ok := infrastructure.DefaultRegistry.Create("postgres")
		if !ok {
			t.Fatal("postgres provider not registered")
		}
		if provider.Name() != "postgres" {
			t.Errorf("expected name 'postgres', got %s", provider.Name())
		}
	})

	t.Run("redis registered", func(t *testing.T) {
		provider, ok := infrastructure.DefaultRegistry.Create("redis")
		if !ok {
			t.Fatal("redis provider not registered")
		}
		if provider.Name() != "redis" {
			t.Errorf("expected name 'redis', got %s", provider.Name())
		}
	})
}
