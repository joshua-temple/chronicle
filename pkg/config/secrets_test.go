package config

import (
	"context"
	"errors"
	"os"
	"testing"
)

// setTestEnv sets an environment variable for testing, returning a cleanup function.
func setTestEnv(t *testing.T, key, value string) func() {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}
	return func() {
		_ = os.Unsetenv(key)
	}
}

func TestEnvSecretProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves existing env var", func(t *testing.T) {
		cleanup := setTestEnv(t, "TEST_SECRET_KEY", "secret-value")
		defer cleanup()

		provider := NewEnvSecretProvider()
		val, err := provider.Resolve(ctx, "TEST_SECRET_KEY")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "secret-value" {
			t.Errorf("expected 'secret-value', got %s", val)
		}
	})

	t.Run("returns error for missing env var", func(t *testing.T) {
		provider := NewEnvSecretProvider()
		_, err := provider.Resolve(ctx, "NONEXISTENT_SECRET_KEY_12345")
		if err == nil {
			t.Fatal("expected error for missing key")
		}
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("expected ErrSecretNotFound, got %v", err)
		}
	})

	t.Run("with prefix", func(t *testing.T) {
		cleanup := setTestEnv(t, "APP_SECRET_DB_PASSWORD", "db-pass")
		defer cleanup()

		provider := NewEnvSecretProvider().WithPrefix("APP_SECRET_")
		val, err := provider.Resolve(ctx, "DB_PASSWORD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "db-pass" {
			t.Errorf("expected 'db-pass', got %s", val)
		}
	})

	t.Run("watch not supported", func(t *testing.T) {
		provider := NewEnvSecretProvider()
		err := provider.Watch(ctx, "key", func(string) {})
		if !errors.Is(err, ErrNotSupported) {
			t.Errorf("expected ErrNotSupported, got %v", err)
		}
	})
}

func TestStaticSecretProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves static secret", func(t *testing.T) {
		provider := NewStaticSecretProvider(map[string]string{
			"api-key":     "key-123",
			"db-password": "pass-456",
		})

		val, err := provider.Resolve(ctx, "api-key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "key-123" {
			t.Errorf("expected 'key-123', got %s", val)
		}
	})

	t.Run("returns error for missing key", func(t *testing.T) {
		provider := NewStaticSecretProvider(nil)
		_, err := provider.Resolve(ctx, "nonexistent")
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("expected ErrSecretNotFound, got %v", err)
		}
	})

	t.Run("set and resolve", func(t *testing.T) {
		provider := NewStaticSecretProvider(nil)
		provider.Set("dynamic-key", "dynamic-value")

		val, err := provider.Resolve(ctx, "dynamic-key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "dynamic-value" {
			t.Errorf("expected 'dynamic-value', got %s", val)
		}
	})
}

func TestChainedSecretProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves from first provider", func(t *testing.T) {
		p1 := NewStaticSecretProvider(map[string]string{"key": "value1"})
		p2 := NewStaticSecretProvider(map[string]string{"key": "value2"})

		chain := NewChainedSecretProvider(p1, p2)
		val, err := chain.Resolve(ctx, "key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "value1" {
			t.Errorf("expected 'value1', got %s", val)
		}
	})

	t.Run("falls back to second provider", func(t *testing.T) {
		p1 := NewStaticSecretProvider(map[string]string{"other": "value"})
		p2 := NewStaticSecretProvider(map[string]string{"key": "value2"})

		chain := NewChainedSecretProvider(p1, p2)
		val, err := chain.Resolve(ctx, "key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "value2" {
			t.Errorf("expected 'value2', got %s", val)
		}
	})

	t.Run("returns error if all fail", func(t *testing.T) {
		p1 := NewStaticSecretProvider(nil)
		p2 := NewStaticSecretProvider(nil)

		chain := NewChainedSecretProvider(p1, p2)
		_, err := chain.Resolve(ctx, "missing")
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("expected ErrSecretNotFound, got %v", err)
		}
	})
}

func TestVariableResolver(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves env vars", func(t *testing.T) {
		cleanup := setTestEnv(t, "TEST_VAR", "test-value")
		defer cleanup()

		resolver := NewVariableResolver(nil)
		result, err := resolver.Resolve(ctx, "prefix-${TEST_VAR}-suffix")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "prefix-test-value-suffix" {
			t.Errorf("expected 'prefix-test-value-suffix', got %s", result)
		}
	})

	t.Run("resolves explicit env vars", func(t *testing.T) {
		cleanup := setTestEnv(t, "TEST_ENV_VAR", "env-value")
		defer cleanup()

		resolver := NewVariableResolver(nil)
		result, err := resolver.Resolve(ctx, "value: ${env.TEST_ENV_VAR}")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "value: env-value" {
			t.Errorf("expected 'value: env-value', got %s", result)
		}
	})

	t.Run("resolves secrets", func(t *testing.T) {
		provider := NewStaticSecretProvider(map[string]string{
			"db-password": "secret123",
		})

		resolver := NewVariableResolver(provider)
		result, err := resolver.Resolve(ctx, "password: ${secrets.db-password}")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "password: secret123" {
			t.Errorf("expected 'password: secret123', got %s", result)
		}
	})

	t.Run("falls back to env for secrets", func(t *testing.T) {
		cleanup := setTestEnv(t, "api-key", "env-api-key")
		defer cleanup()

		// Empty secret provider - will fall back to env
		provider := NewStaticSecretProvider(nil)

		resolver := NewVariableResolver(provider).WithFallbackToEnv(true)
		result, err := resolver.Resolve(ctx, "key: ${secrets.api-key}")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "key: env-api-key" {
			t.Errorf("expected 'key: env-api-key', got %s", result)
		}
	})

	t.Run("error on undefined with default policy", func(t *testing.T) {
		resolver := NewVariableResolver(nil)
		_, err := resolver.Resolve(ctx, "${UNDEFINED_VAR_12345}")
		if err == nil {
			t.Fatal("expected error for undefined variable")
		}
	})

	t.Run("keeps undefined with KeepUndefined policy", func(t *testing.T) {
		resolver := NewVariableResolver(nil).WithUndefinedPolicy(KeepUndefined)
		result, err := resolver.Resolve(ctx, "${UNDEFINED_VAR}")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "${UNDEFINED_VAR}" {
			t.Errorf("expected '${UNDEFINED_VAR}', got %s", result)
		}
	})

	t.Run("empties undefined with EmptyUndefined policy", func(t *testing.T) {
		resolver := NewVariableResolver(nil).WithUndefinedPolicy(EmptyUndefined)
		result, err := resolver.Resolve(ctx, "prefix-${UNDEFINED_VAR}-suffix")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "prefix--suffix" {
			t.Errorf("expected 'prefix--suffix', got %s", result)
		}
	})

	t.Run("resolves multiple variables", func(t *testing.T) {
		cleanup1 := setTestEnv(t, "VAR1", "one")
		cleanup2 := setTestEnv(t, "VAR2", "two")
		defer cleanup1()
		defer cleanup2()

		resolver := NewVariableResolver(nil)
		result, err := resolver.Resolve(ctx, "${VAR1}-${VAR2}")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "one-two" {
			t.Errorf("expected 'one-two', got %s", result)
		}
	})

	t.Run("no change for strings without variables", func(t *testing.T) {
		resolver := NewVariableResolver(nil)
		result, err := resolver.Resolve(ctx, "plain string")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "plain string" {
			t.Errorf("expected 'plain string', got %s", result)
		}
	})
}

func TestHasVariables(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"plain string", false},
		{"${VAR}", true},
		{"${env.VAR}", true},
		{"${secrets.key}", true},
		{"prefix-${VAR}-suffix", true},
		{"$VAR", false}, // Not our syntax
		{"{VAR}", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := HasVariables(tt.input)
			if result != tt.expected {
				t.Errorf("HasVariables(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"plain string", nil},
		{"${VAR}", []string{"VAR"}},
		{"${env.VAR}", []string{"env.VAR"}},
		{"${secrets.key}", []string{"secrets.key"}},
		{"${VAR1}-${VAR2}", []string{"VAR1", "VAR2"}},
		{"${VAR}-${VAR}", []string{"VAR"}}, // Deduplication
		{"${VAR}-${env.VAR}-${secrets.key}", []string{"VAR", "env.VAR", "secrets.key"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ExtractVariables(tt.input)

			if len(result) != len(tt.expected) {
				t.Fatalf("ExtractVariables(%q) returned %d vars, want %d", tt.input, len(result), len(tt.expected))
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("ExtractVariables(%q)[%d] = %s, want %s", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestResolveMapValues(t *testing.T) {
	ctx := context.Background()

	cleanup := setTestEnv(t, "MAP_VAR", "resolved")
	defer cleanup()

	t.Run("resolves string values", func(t *testing.T) {
		m := map[string]any{
			"key":    "${MAP_VAR}",
			"plain":  "plain-value",
			"number": 42,
		}

		resolver := NewVariableResolver(nil)
		err := ResolveMapValues(ctx, m, resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if m["key"] != "resolved" {
			t.Errorf("expected 'resolved', got %v", m["key"])
		}
		if m["plain"] != "plain-value" {
			t.Errorf("expected 'plain-value', got %v", m["plain"])
		}
		if m["number"] != 42 {
			t.Errorf("expected 42, got %v", m["number"])
		}
	})

	t.Run("resolves nested maps", func(t *testing.T) {
		m := map[string]any{
			"outer": map[string]any{
				"inner": "${MAP_VAR}",
			},
		}

		resolver := NewVariableResolver(nil)
		err := ResolveMapValues(ctx, m, resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		outer := m["outer"].(map[string]any)
		if outer["inner"] != "resolved" {
			t.Errorf("expected 'resolved', got %v", outer["inner"])
		}
	})

	t.Run("resolves array values", func(t *testing.T) {
		m := map[string]any{
			"list": []any{"${MAP_VAR}", "plain"},
		}

		resolver := NewVariableResolver(nil)
		err := ResolveMapValues(ctx, m, resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		list := m["list"].([]any)
		if list[0] != "resolved" {
			t.Errorf("expected 'resolved', got %v", list[0])
		}
		if list[1] != "plain" {
			t.Errorf("expected 'plain', got %v", list[1])
		}
	})
}

func TestCreateSecretProvider(t *testing.T) {
	t.Run("creates env provider", func(t *testing.T) {
		cfg := SecretsConfig{Provider: "env"}
		provider := CreateSecretProvider(cfg)
		if provider == nil {
			t.Fatal("expected provider")
		}
	})

	t.Run("creates default provider", func(t *testing.T) {
		cfg := SecretsConfig{Provider: ""}
		provider := CreateSecretProvider(cfg)
		if provider == nil {
			t.Fatal("expected provider")
		}
	})

	t.Run("creates chained provider with fallback", func(t *testing.T) {
		cfg := SecretsConfig{
			Provider:      "vault",
			FallbackToEnv: true,
		}
		provider := CreateSecretProvider(cfg)
		if provider == nil {
			t.Fatal("expected provider")
		}
		// Should be a chained provider (vault + env)
		_, ok := provider.(*ChainedSecretProvider)
		if !ok {
			t.Error("expected ChainedSecretProvider")
		}
	})
}

func TestResolveInConfig(t *testing.T) {
	ctx := context.Background()

	cleanup1 := setTestEnv(t, "DB_IMAGE", "postgres:15")
	cleanup2 := setTestEnv(t, "DB_PASSWORD", "secret123")
	cleanup3 := setTestEnv(t, "HEALTH_ENDPOINT", "http://localhost:5432")
	defer cleanup1()
	defer cleanup2()
	defer cleanup3()

	t.Run("resolves infrastructure image", func(t *testing.T) {
		cfg := &Config{
			Infrastructure: map[string]InfraConfig{
				"db": {
					Image: "${DB_IMAGE}",
				},
			},
		}

		resolver := NewVariableResolver(nil)
		err := ResolveInConfig(ctx, cfg, resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Infrastructure["db"].Image != "postgres:15" {
			t.Errorf("expected 'postgres:15', got %s", cfg.Infrastructure["db"].Image)
		}
	})

	t.Run("resolves infrastructure env values", func(t *testing.T) {
		cfg := &Config{
			Infrastructure: map[string]InfraConfig{
				"db": {
					Env: map[string]string{
						"POSTGRES_PASSWORD": "${DB_PASSWORD}",
						"PLAIN_VALUE":       "plain",
					},
				},
			},
		}

		resolver := NewVariableResolver(nil)
		err := ResolveInConfig(ctx, cfg, resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Infrastructure["db"].Env["POSTGRES_PASSWORD"] != "secret123" {
			t.Errorf("expected 'secret123', got %s", cfg.Infrastructure["db"].Env["POSTGRES_PASSWORD"])
		}
		if cfg.Infrastructure["db"].Env["PLAIN_VALUE"] != "plain" {
			t.Errorf("expected 'plain', got %s", cfg.Infrastructure["db"].Env["PLAIN_VALUE"])
		}
	})

	t.Run("resolves health check endpoint", func(t *testing.T) {
		cfg := &Config{
			Infrastructure: map[string]InfraConfig{
				"db": {
					HealthCheck: HealthCheckConfig{
						Endpoint: "${HEALTH_ENDPOINT}",
					},
				},
			},
		}

		resolver := NewVariableResolver(nil)
		err := ResolveInConfig(ctx, cfg, resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Infrastructure["db"].HealthCheck.Endpoint != "http://localhost:5432" {
			t.Errorf("expected 'http://localhost:5432', got %s", cfg.Infrastructure["db"].HealthCheck.Endpoint)
		}
	})

	t.Run("resolves vault config", func(t *testing.T) {
		cleanup := setTestEnv(t, "VAULT_ADDR", "https://vault.example.com")
		defer cleanup()

		cfg := &Config{
			Secrets: SecretsConfig{
				Vault: &VaultConfig{
					Address: "${VAULT_ADDR}",
				},
			},
		}

		resolver := NewVariableResolver(nil)
		err := ResolveInConfig(ctx, cfg, resolver)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Secrets.Vault.Address != "https://vault.example.com" {
			t.Errorf("expected 'https://vault.example.com', got %s", cfg.Secrets.Vault.Address)
		}
	})

	t.Run("nil resolver returns nil", func(t *testing.T) {
		cfg := &Config{}
		err := ResolveInConfig(ctx, cfg, nil)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
