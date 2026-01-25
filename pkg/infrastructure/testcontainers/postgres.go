package testcontainers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresConfig holds configuration for a PostgreSQL container.
type PostgresConfig struct {
	Image    string
	Database string
	Username string
	Password string
}

// DefaultPostgresConfig returns the default PostgreSQL configuration.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Image:    "postgres:15-alpine",
		Database: "testdb",
		Username: "postgres",
		Password: "postgres",
	}
}

// PostgresProvider provides PostgreSQL infrastructure via testcontainers.
type PostgresProvider struct {
	*ContainerProvider
	pgContainer *postgres.PostgresContainer
	config      PostgresConfig
	db          *sql.DB
}

// NewPostgresProvider creates a new PostgreSQL provider.
func NewPostgresProvider() *PostgresProvider {
	return &PostgresProvider{
		ContainerProvider: NewContainerProvider("postgres"),
		config:            DefaultPostgresConfig(),
	}
}

// Initialize configures the provider.
func (p *PostgresProvider) Initialize(ctx context.Context, config map[string]any) error {
	if image, ok := config["image"].(string); ok {
		p.config.Image = image
	}
	if database, ok := config["database"].(string); ok {
		p.config.Database = database
	}
	if username, ok := config["username"].(string); ok {
		p.config.Username = username
	}
	if password, ok := config["password"].(string); ok {
		p.config.Password = password
	}

	return nil
}

// Start starts the PostgreSQL container.
func (p *PostgresProvider) Start(ctx context.Context) error {
	p.status.Store(int32(infrastructure.StatusStarting))

	container, err := postgres.Run(ctx,
		p.config.Image,
		postgres.WithDatabase(p.config.Database),
		postgres.WithUsername(p.config.Username),
		postgres.WithPassword(p.config.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	p.pgContainer = container

	// Get connection string and create a connection pool
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		_ = container.Terminate(ctx)
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		_ = container.Terminate(ctx)
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	p.SetClient("default", db)
	p.SetClient("db", db)
	p.status.Store(int32(infrastructure.StatusRunning))

	return nil
}

// Stop stops the PostgreSQL container.
func (p *PostgresProvider) Stop(ctx context.Context) error {
	p.status.Store(int32(infrastructure.StatusStopping))

	if p.db != nil {
		if err := p.db.Close(); err != nil {
			// Log but don't fail
			_ = err
		}
		p.db = nil
	}

	if p.pgContainer != nil {
		if err := p.pgContainer.Terminate(ctx); err != nil {
			p.status.Store(int32(infrastructure.StatusUnhealthy))
			return fmt.Errorf("failed to terminate postgres container: %w", err)
		}
		p.pgContainer = nil
	}

	p.status.Store(int32(infrastructure.StatusStopped))
	return nil
}

// HealthCheck returns the health status.
func (p *PostgresProvider) HealthCheck(ctx context.Context) infrastructure.HealthReport {
	if p.db == nil {
		return infrastructure.HealthReport{
			Healthy: false,
			Message: "database connection not available",
			Services: map[string]infrastructure.ServiceHealth{
				"postgres": {
					Name:   "postgres",
					Status: "stopped",
				},
			},
		}
	}

	start := time.Now()
	err := p.db.PingContext(ctx)
	latency := time.Since(start)

	if err != nil {
		return infrastructure.HealthReport{
			Healthy: false,
			Message: fmt.Sprintf("ping failed: %v", err),
			Services: map[string]infrastructure.ServiceHealth{
				"postgres": {
					Name:    "postgres",
					Status:  "unhealthy",
					Latency: latency,
					Error:   err,
				},
			},
		}
	}

	return infrastructure.HealthReport{
		Healthy: true,
		Message: "database healthy",
		Services: map[string]infrastructure.ServiceHealth{
			"postgres": {
				Name:    "postgres",
				Status:  "healthy",
				Latency: latency,
			},
		},
	}
}

// DB returns the database connection.
func (p *PostgresProvider) DB() *sql.DB {
	return p.db
}

// ConnectionString returns the connection string for the database.
func (p *PostgresProvider) ConnectionString(ctx context.Context) (string, error) {
	if p.pgContainer == nil {
		return "", fmt.Errorf("container not started")
	}
	return p.pgContainer.ConnectionString(ctx, "sslmode=disable")
}

// Flush implements FlushableProvider by truncating all user tables.
func (p *PostgresProvider) Flush(ctx context.Context) error {
	return p.FlushWithConfig(ctx, infrastructure.FlushConfig{
		Strategy: "truncate",
	})
}

// FlushWithConfig flushes data based on configuration.
func (p *PostgresProvider) FlushWithConfig(ctx context.Context, config infrastructure.FlushConfig) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	switch config.Strategy {
	case "truncate":
		return p.flushTruncate(ctx, config)
	case "drop_recreate":
		return p.flushDropRecreate(ctx, config)
	default:
		return p.flushTruncate(ctx, config)
	}
}

func (p *PostgresProvider) flushTruncate(ctx context.Context, config infrastructure.FlushConfig) error {
	// Get list of tables to truncate
	var tables []string

	if len(config.Include) > 0 {
		tables = config.Include
	} else {
		// Get all user tables
		rows, err := p.db.QueryContext(ctx, `
			SELECT tablename FROM pg_tables
			WHERE schemaname = 'public'
		`)
		if err != nil {
			return fmt.Errorf("failed to get tables: %w", err)
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return fmt.Errorf("failed to scan table name: %w", err)
			}
			tables = append(tables, table)
		}
	}

	// Filter out excluded tables
	excludeSet := make(map[string]bool)
	for _, e := range config.Exclude {
		excludeSet[e] = true
	}

	for _, table := range tables {
		if excludeSet[table] {
			continue
		}

		_, err := p.db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %q CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

func (p *PostgresProvider) flushDropRecreate(ctx context.Context, config infrastructure.FlushConfig) error {
	// Drop and recreate the public schema
	_, err := p.db.ExecContext(ctx, `
		DROP SCHEMA IF EXISTS public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO postgres;
		GRANT ALL ON SCHEMA public TO public;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop/recreate schema: %w", err)
	}

	return nil
}

func init() {
	infrastructure.RegisterProvider("postgres", func() infrastructure.Provider {
		return NewPostgresProvider()
	})
}
