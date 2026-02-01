// Package main demonstrates Chronicle's infrastructure management capabilities.
//
// This example shows:
// - Provider registration and configuration
// - Infrastructure manager usage
// - Secret management with variable resolution
// - Reuse behavior for efficient test execution
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/infrastructure"

	// Import testcontainers providers to auto-register them
	_ "github.com/joshua-temple/chronicle/pkg/infrastructure/testcontainers"
)

func main() {
	ctx := context.Background()

	// Demonstrate secret management
	fmt.Println("=== Secret Management Demo ===")
	demoSecrets(ctx)

	// Demonstrate infrastructure manager
	fmt.Println("\n=== Infrastructure Manager Demo ===")
	demoManager(ctx)

	// Demonstrate reuse behavior
	fmt.Println("\n=== Reuse Manager Demo ===")
	demoReuse(ctx)
}

func demoSecrets(ctx context.Context) {
	// Set up some environment variables for demonstration
	_ = os.Setenv("DB_HOST", "localhost")
	_ = os.Setenv("DB_PASSWORD", "secret123")
	defer func() {
		_ = os.Unsetenv("DB_HOST")
		_ = os.Unsetenv("DB_PASSWORD")
	}()

	// Create a secret provider chain: static secrets -> env vars
	staticSecrets := config.NewStaticSecretProvider(map[string]string{
		"api-key":    "static-api-key-value",
		"jwt-secret": "my-jwt-secret",
	})
	envSecrets := config.NewEnvSecretProvider()

	// Chained provider tries static first, then falls back to env
	chainedProvider := config.NewChainedSecretProvider(staticSecrets, envSecrets)

	// Create a variable resolver
	resolver := config.NewVariableResolver(chainedProvider).
		WithFallbackToEnv(true).
		WithUndefinedPolicy(config.KeepUndefined)

	// Demonstrate variable resolution
	testCases := []string{
		"Database host: ${DB_HOST}",
		"Database password: ${secrets.DB_PASSWORD}",
		"API Key: ${secrets.api-key}",
		"Combined: ${DB_HOST}:5432 with ${secrets.api-key}",
		"Undefined: ${UNDEFINED_VAR}", // Will be kept as-is
	}

	for _, tc := range testCases {
		resolved, err := resolver.Resolve(ctx, tc)
		if err != nil {
			log.Printf("Error resolving %q: %v", tc, err)
			continue
		}
		fmt.Printf("  Input:    %s\n", tc)
		fmt.Printf("  Resolved: %s\n\n", resolved)
	}

	// Demonstrate config resolution
	cfg := &config.Config{
		Infrastructure: map[string]config.InfraConfig{
			"database": {
				Image: "postgres:15",
				Env: map[string]string{
					"POSTGRES_HOST":     "${DB_HOST}",
					"POSTGRES_PASSWORD": "${secrets.DB_PASSWORD}",
				},
			},
		},
		Secrets: config.SecretsConfig{
			Vault: &config.VaultConfig{
				Address: "https://${DB_HOST}:8200",
			},
		},
	}

	if err := config.ResolveInConfig(ctx, cfg, resolver); err != nil {
		log.Printf("Error resolving config: %v", err)
	} else {
		fmt.Println("Config resolved successfully:")
		fmt.Printf("  Database env POSTGRES_HOST: %s\n", cfg.Infrastructure["database"].Env["POSTGRES_HOST"])
		fmt.Printf("  Vault address: %s\n", cfg.Secrets.Vault.Address)
	}
}

func demoManager(ctx context.Context) {
	// Get available providers
	fmt.Println("Available providers:", infrastructure.DefaultRegistry.Available())

	// Create a manager
	manager := infrastructure.NewManager(nil)
	manager.SetDefaultReuse(infrastructure.ReuseWithFlush)
	manager.SetDefaultIsolation(infrastructure.DataIsolation)

	// Configure a PostgreSQL provider (without actually starting it)
	err := manager.Configure(infrastructure.ProviderConfig{
		Name:     "test-db",
		Provider: "postgres",
		Config: map[string]any{
			"image":    "postgres:15-alpine",
			"database": "testdb",
			"username": "postgres",
			"password": "postgres",
		},
		Reuse:     infrastructure.ReuseWithFlush,
		Isolation: infrastructure.DataIsolation,
		Flush: infrastructure.FlushConfig{
			Strategy: "truncate",
			Exclude:  []string{"schema_migrations"},
		},
	})
	if err != nil {
		log.Printf("Error configuring postgres: %v", err)
	} else {
		fmt.Println("PostgreSQL provider configured successfully")
	}

	// Configure a Redis provider (without actually starting it)
	err = manager.Configure(infrastructure.ProviderConfig{
		Name:     "cache",
		Provider: "redis",
		Config: map[string]any{
			"image": "redis:7-alpine",
		},
		Reuse:     infrastructure.FullReuse,
		Isolation: infrastructure.NoIsolation,
		Flush: infrastructure.FlushConfig{
			Strategy: "flushdb",
		},
	})
	if err != nil {
		log.Printf("Error configuring redis: %v", err)
	} else {
		fmt.Println("Redis provider configured successfully")
	}

	// Show configured providers
	fmt.Println("Configured providers:", manager.ProviderNames())

	// Demonstrate status before start
	fmt.Println("Provider statuses:")
	for name, status := range manager.Status() {
		fmt.Printf("  %s: %s\n", name, status)
	}

	// Note: We don't actually start the providers in this demo
	// as that would require Docker to be running. In a real test,
	// you would call manager.Start(ctx) here.
	fmt.Println("\n(Skipping actual container start - requires Docker)")
}

func demoReuse(ctx context.Context) {
	// Create a reuse manager
	rm := infrastructure.NewReuseManager()
	rm.SetStorePath("") // Disable persistence for demo

	// Simulate first test run - creates new entry
	reuseConfig := infrastructure.ReuseConfig{
		Enabled: true,
		TTL:     1 * time.Hour,
		Config: map[string]any{
			"image":    "postgres:15",
			"database": "testdb",
		},
	}

	entry1, existed, err := rm.GetOrCreate(ctx, "postgres", reuseConfig)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("First call: entry existed=%v, key=%s\n", existed, entry1.Key)

	// Update with endpoints (simulating container start)
	err = rm.Update(entry1.Key, map[string]string{
		"default": "localhost:5432",
		"admin":   "localhost:5433",
	})
	if err != nil {
		log.Printf("Error updating: %v", err)
	}

	// Simulate second test run - reuses existing entry
	entry2, existed, err := rm.GetOrCreate(ctx, "postgres", reuseConfig)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("Second call: entry existed=%v, key=%s\n", existed, entry2.Key)
	fmt.Printf("Reused endpoints: %v\n", entry2.Endpoints)

	// Demonstrate key computation consistency
	key1 := infrastructure.ComputeKey("postgres", map[string]any{"image": "postgres:15"})
	key2 := infrastructure.ComputeKey("postgres", map[string]any{"image": "postgres:15"})
	key3 := infrastructure.ComputeKey("postgres", map[string]any{"image": "postgres:16"})
	fmt.Printf("\nKey consistency:\n")
	fmt.Printf("  postgres:15 key: %s\n", key1)
	fmt.Printf("  postgres:15 key: %s (same)\n", key2)
	fmt.Printf("  postgres:16 key: %s (different)\n", key3)

	// Show entry details
	fmt.Printf("\nEntry details:\n")
	fmt.Printf("  Provider: %s\n", entry1.Provider)
	fmt.Printf("  Created: %v\n", entry1.CreatedAt)
	fmt.Printf("  Expires: %v\n", entry1.ExpiresAt)
	fmt.Printf("  Time remaining: %v\n", entry1.TimeRemaining())

	// Clean up
	rm.Clear()
	fmt.Printf("\nEntries after clear: %d\n", len(rm.Entries()))
}
