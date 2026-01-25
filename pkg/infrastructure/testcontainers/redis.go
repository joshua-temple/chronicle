package testcontainers

import (
	"context"
	"fmt"
	"time"

	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisConfig holds configuration for a Redis container.
type RedisConfig struct {
	Image    string
	Password string
}

// DefaultRedisConfig returns the default Redis configuration.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Image:    "redis:7-alpine",
		Password: "",
	}
}

// RedisProvider provides Redis infrastructure via testcontainers.
type RedisProvider struct {
	*ContainerProvider
	redisContainer *tcredis.RedisContainer
	config         RedisConfig
	client         *redis.Client
}

// NewRedisProvider creates a new Redis provider.
func NewRedisProvider() *RedisProvider {
	return &RedisProvider{
		ContainerProvider: NewContainerProvider("redis"),
		config:            DefaultRedisConfig(),
	}
}

// Initialize configures the provider.
func (p *RedisProvider) Initialize(ctx context.Context, config map[string]any) error {
	if image, ok := config["image"].(string); ok {
		p.config.Image = image
	}
	if password, ok := config["password"].(string); ok {
		p.config.Password = password
	}

	return nil
}

// Start starts the Redis container.
func (p *RedisProvider) Start(ctx context.Context) error {
	p.status.Store(int32(infrastructure.StatusStarting))

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(60 * time.Second),
		),
	}

	container, err := tcredis.Run(ctx, p.config.Image, opts...)
	if err != nil {
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to start redis container: %w", err)
	}

	p.redisContainer = container

	// Get connection string
	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	// Parse options from connection string
	opt, err := redis.ParseURL(connStr)
	if err != nil {
		_ = container.Terminate(ctx)
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Create client
	client := redis.NewClient(opt)

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		_ = container.Terminate(ctx)
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	p.client = client
	p.SetClient("default", client)
	p.SetClient("redis", client)
	p.status.Store(int32(infrastructure.StatusRunning))

	return nil
}

// Stop stops the Redis container.
func (p *RedisProvider) Stop(ctx context.Context) error {
	p.status.Store(int32(infrastructure.StatusStopping))

	if p.client != nil {
		if err := p.client.Close(); err != nil {
			// Log but don't fail
			_ = err
		}
		p.client = nil
	}

	if p.redisContainer != nil {
		if err := p.redisContainer.Terminate(ctx); err != nil {
			p.status.Store(int32(infrastructure.StatusUnhealthy))
			return fmt.Errorf("failed to terminate redis container: %w", err)
		}
		p.redisContainer = nil
	}

	p.status.Store(int32(infrastructure.StatusStopped))
	return nil
}

// HealthCheck returns the health status.
func (p *RedisProvider) HealthCheck(ctx context.Context) infrastructure.HealthReport {
	if p.client == nil {
		return infrastructure.HealthReport{
			Healthy: false,
			Message: "redis connection not available",
			Services: map[string]infrastructure.ServiceHealth{
				"redis": {
					Name:   "redis",
					Status: "stopped",
				},
			},
		}
	}

	start := time.Now()
	err := p.client.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		return infrastructure.HealthReport{
			Healthy: false,
			Message: fmt.Sprintf("ping failed: %v", err),
			Services: map[string]infrastructure.ServiceHealth{
				"redis": {
					Name:    "redis",
					Status:  "unhealthy",
					Latency: latency,
					Error:   err,
				},
			},
		}
	}

	return infrastructure.HealthReport{
		Healthy: true,
		Message: "redis healthy",
		Services: map[string]infrastructure.ServiceHealth{
			"redis": {
				Name:    "redis",
				Status:  "healthy",
				Latency: latency,
			},
		},
	}
}

// Client returns the Redis client.
func (p *RedisProvider) RedisClient() *redis.Client {
	return p.client
}

// Flush implements FlushableProvider by flushing the database.
func (p *RedisProvider) Flush(ctx context.Context) error {
	return p.FlushWithConfig(ctx, infrastructure.FlushConfig{
		Strategy: "flushdb",
	})
}

// FlushWithConfig flushes data based on configuration.
func (p *RedisProvider) FlushWithConfig(ctx context.Context, config infrastructure.FlushConfig) error {
	if p.client == nil {
		return fmt.Errorf("redis not connected")
	}

	switch config.Strategy {
	case "flushdb":
		return p.client.FlushDB(ctx).Err()
	case "flushall":
		return p.client.FlushAll(ctx).Err()
	case "pattern":
		return p.flushPattern(ctx, config)
	default:
		return p.client.FlushDB(ctx).Err()
	}
}

func (p *RedisProvider) flushPattern(ctx context.Context, config infrastructure.FlushConfig) error {
	pattern := "*"
	if len(config.Include) > 0 {
		pattern = config.Include[0]
	}
	if opt, ok := config.Options["pattern"].(string); ok {
		pattern = opt
	}

	// Use SCAN to find keys matching pattern
	var cursor uint64
	for {
		keys, nextCursor, err := p.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			// Delete keys in batches
			if err := p.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func init() {
	infrastructure.RegisterProvider("redis", func() infrastructure.Provider {
		return NewRedisProvider()
	})
}
