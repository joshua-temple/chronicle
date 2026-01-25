package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// SecretProvider defines the interface for retrieving secrets.
type SecretProvider interface {
	// Resolve retrieves a secret value by key.
	Resolve(ctx context.Context, key string) (string, error)

	// Watch monitors for secret rotation (optional - may return ErrNotSupported).
	Watch(ctx context.Context, key string, callback func(newValue string)) error
}

// ErrNotSupported indicates that an operation is not supported by the provider.
var ErrNotSupported = errors.New("operation not supported")

// ErrSecretNotFound indicates that a secret was not found.
var ErrSecretNotFound = errors.New("secret not found")

// EnvSecretProvider resolves secrets from environment variables.
type EnvSecretProvider struct {
	prefix string // Optional prefix for env vars
}

// NewEnvSecretProvider creates a new environment-based secret provider.
func NewEnvSecretProvider() *EnvSecretProvider {
	return &EnvSecretProvider{}
}

// WithPrefix sets an optional prefix for environment variable names.
func (p *EnvSecretProvider) WithPrefix(prefix string) *EnvSecretProvider {
	p.prefix = prefix
	return p
}

// Resolve retrieves a value from environment variables.
func (p *EnvSecretProvider) Resolve(ctx context.Context, key string) (string, error) {
	envKey := key
	if p.prefix != "" {
		envKey = p.prefix + key
	}

	val, ok := os.LookupEnv(envKey)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	return val, nil
}

// Watch is not supported for environment variables (they don't change dynamically).
func (p *EnvSecretProvider) Watch(ctx context.Context, key string, callback func(string)) error {
	return ErrNotSupported
}

// StaticSecretProvider provides secrets from an in-memory map (useful for testing).
type StaticSecretProvider struct {
	secrets map[string]string
	mu      sync.RWMutex
}

// NewStaticSecretProvider creates a new static secret provider.
func NewStaticSecretProvider(secrets map[string]string) *StaticSecretProvider {
	if secrets == nil {
		secrets = make(map[string]string)
	}
	return &StaticSecretProvider{
		secrets: secrets,
	}
}

// Set adds or updates a secret.
func (p *StaticSecretProvider) Set(key, value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.secrets[key] = value
}

// Resolve retrieves a secret from the static map.
func (p *StaticSecretProvider) Resolve(ctx context.Context, key string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	val, ok := p.secrets[key]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	return val, nil
}

// Watch is not supported for static secrets.
func (p *StaticSecretProvider) Watch(ctx context.Context, key string, callback func(string)) error {
	return ErrNotSupported
}

// ChainedSecretProvider tries multiple providers in order.
type ChainedSecretProvider struct {
	providers []SecretProvider
}

// NewChainedSecretProvider creates a provider that chains multiple providers.
func NewChainedSecretProvider(providers ...SecretProvider) *ChainedSecretProvider {
	return &ChainedSecretProvider{
		providers: providers,
	}
}

// Resolve tries each provider until one succeeds.
func (p *ChainedSecretProvider) Resolve(ctx context.Context, key string) (string, error) {
	for _, provider := range p.providers {
		val, err := provider.Resolve(ctx, key)
		if err == nil {
			return val, nil
		}
		if !errors.Is(err, ErrSecretNotFound) {
			// Non-not-found errors are returned immediately
			return "", err
		}
	}
	return "", fmt.Errorf("%w: %s", ErrSecretNotFound, key)
}

// Watch tries to watch on the first provider that supports it.
func (p *ChainedSecretProvider) Watch(ctx context.Context, key string, callback func(string)) error {
	for _, provider := range p.providers {
		err := provider.Watch(ctx, key, callback)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrNotSupported) {
			return err
		}
	}
	return ErrNotSupported
}

// Variable interpolation patterns
var (
	// ${VAR} - environment variable
	envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

	// ${secrets.KEY} - secret provider
	secretPattern = regexp.MustCompile(`\$\{secrets\.([A-Za-z_][A-Za-z0-9_.-]*)\}`)

	// ${env.VAR} - explicit environment variable
	explicitEnvPattern = regexp.MustCompile(`\$\{env\.([A-Za-z_][A-Za-z0-9_]*)\}`)
)

// VariableResolver resolves variables in configuration values.
type VariableResolver struct {
	secretProvider  SecretProvider
	fallbackToEnv   bool
	undefinedPolicy UndefinedPolicy
}

// UndefinedPolicy determines how to handle undefined variables.
type UndefinedPolicy int

const (
	// ErrorOnUndefined returns an error for undefined variables.
	ErrorOnUndefined UndefinedPolicy = iota
	// KeepUndefined leaves undefined variables as-is.
	KeepUndefined
	// EmptyUndefined replaces undefined variables with empty string.
	EmptyUndefined
)

// NewVariableResolver creates a new variable resolver.
func NewVariableResolver(provider SecretProvider) *VariableResolver {
	return &VariableResolver{
		secretProvider:  provider,
		fallbackToEnv:   true,
		undefinedPolicy: ErrorOnUndefined,
	}
}

// WithFallbackToEnv controls whether to fall back to environment variables.
func (r *VariableResolver) WithFallbackToEnv(fallback bool) *VariableResolver {
	r.fallbackToEnv = fallback
	return r
}

// WithUndefinedPolicy sets the policy for undefined variables.
func (r *VariableResolver) WithUndefinedPolicy(policy UndefinedPolicy) *VariableResolver {
	r.undefinedPolicy = policy
	return r
}

// Resolve resolves all variables in a string.
func (r *VariableResolver) Resolve(ctx context.Context, value string) (string, error) {
	var errs []error

	// Replace ${secrets.KEY} first
	result := secretPattern.ReplaceAllStringFunc(value, func(match string) string {
		key := secretPattern.FindStringSubmatch(match)[1]
		if r.secretProvider != nil {
			val, err := r.secretProvider.Resolve(ctx, key)
			if err == nil {
				return val
			}
			if !errors.Is(err, ErrSecretNotFound) {
				errs = append(errs, err)
				return match
			}
		}

		// Fallback to env if enabled
		if r.fallbackToEnv {
			if val, ok := os.LookupEnv(key); ok {
				return val
			}
		}

		return r.handleUndefined(match, key, &errs)
	})

	// Replace ${env.VAR} explicit env references
	result = explicitEnvPattern.ReplaceAllStringFunc(result, func(match string) string {
		key := explicitEnvPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return r.handleUndefined(match, key, &errs)
	})

	// Replace ${VAR} (plain env var references)
	result = envVarPattern.ReplaceAllStringFunc(result, func(match string) string {
		key := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return r.handleUndefined(match, key, &errs)
	})

	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}

	return result, nil
}

func (r *VariableResolver) handleUndefined(match, key string, errs *[]error) string {
	switch r.undefinedPolicy {
	case ErrorOnUndefined:
		*errs = append(*errs, fmt.Errorf("undefined variable: %s", key))
		return match
	case KeepUndefined:
		return match
	case EmptyUndefined:
		return ""
	default:
		return match
	}
}

// ResolveInConfig resolves all variables in a Config struct.
func ResolveInConfig(ctx context.Context, cfg *Config, resolver *VariableResolver) error {
	if resolver == nil {
		return nil
	}

	var errs []error

	// Resolve infrastructure config values
	for name, infra := range cfg.Infrastructure {
		// Resolve Image
		if infra.Image != "" {
			resolved, err := resolver.Resolve(ctx, infra.Image)
			if err != nil {
				errs = append(errs, fmt.Errorf("infrastructure.%s.image: %w", name, err))
			}
			infra.Image = resolved
		}

		// Resolve Env values
		for k, v := range infra.Env {
			resolved, err := resolver.Resolve(ctx, v)
			if err != nil {
				errs = append(errs, fmt.Errorf("infrastructure.%s.env.%s: %w", name, k, err))
			}
			infra.Env[k] = resolved
		}

		// Resolve HealthCheck endpoint
		if infra.HealthCheck.Endpoint != "" {
			resolved, err := resolver.Resolve(ctx, infra.HealthCheck.Endpoint)
			if err != nil {
				errs = append(errs, fmt.Errorf("infrastructure.%s.health_check.endpoint: %w", name, err))
			}
			infra.HealthCheck.Endpoint = resolved
		}

		// Resolve HealthCheck command
		for i, cmd := range infra.HealthCheck.Command {
			resolved, err := resolver.Resolve(ctx, cmd)
			if err != nil {
				errs = append(errs, fmt.Errorf("infrastructure.%s.health_check.command[%d]: %w", name, i, err))
			}
			infra.HealthCheck.Command[i] = resolved
		}

		cfg.Infrastructure[name] = infra
	}

	// Resolve secrets config
	if cfg.Secrets.Vault != nil {
		if cfg.Secrets.Vault.Address != "" {
			addr, err := resolver.Resolve(ctx, cfg.Secrets.Vault.Address)
			if err != nil {
				errs = append(errs, fmt.Errorf("secrets.vault.address: %w", err))
			}
			cfg.Secrets.Vault.Address = addr
		}
		if cfg.Secrets.Vault.Token != "" {
			token, err := resolver.Resolve(ctx, cfg.Secrets.Vault.Token)
			if err != nil {
				errs = append(errs, fmt.Errorf("secrets.vault.token: %w", err))
			}
			cfg.Secrets.Vault.Token = token
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ResolveMapValues recursively resolves string values in a map.
func ResolveMapValues(ctx context.Context, m map[string]any, resolver *VariableResolver) error {
	var errs []error

	for k, v := range m {
		switch val := v.(type) {
		case string:
			resolved, err := resolver.Resolve(ctx, val)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", k, err))
			}
			m[k] = resolved
		case map[string]any:
			if err := ResolveMapValues(ctx, val, resolver); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", k, err))
			}
		case []any:
			for i, item := range val {
				if str, ok := item.(string); ok {
					resolved, err := resolver.Resolve(ctx, str)
					if err != nil {
						errs = append(errs, fmt.Errorf("%s[%d]: %w", k, i, err))
					}
					val[i] = resolved
				} else if nestedMap, ok := item.(map[string]any); ok {
					if err := ResolveMapValues(ctx, nestedMap, resolver); err != nil {
						errs = append(errs, fmt.Errorf("%s[%d]: %w", k, i, err))
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// HasVariables checks if a string contains unresolved variables.
func HasVariables(s string) bool {
	return envVarPattern.MatchString(s) ||
		secretPattern.MatchString(s) ||
		explicitEnvPattern.MatchString(s)
}

// ExtractVariables returns all variable names found in a string.
func ExtractVariables(s string) []string {
	var vars []string
	seen := make(map[string]bool)

	// Extract ${VAR}
	for _, match := range envVarPattern.FindAllStringSubmatch(s, -1) {
		if !seen[match[1]] {
			vars = append(vars, match[1])
			seen[match[1]] = true
		}
	}

	// Extract ${env.VAR}
	for _, match := range explicitEnvPattern.FindAllStringSubmatch(s, -1) {
		key := "env." + match[1]
		if !seen[key] {
			vars = append(vars, key)
			seen[key] = true
		}
	}

	// Extract ${secrets.KEY}
	for _, match := range secretPattern.FindAllStringSubmatch(s, -1) {
		key := "secrets." + match[1]
		if !seen[key] {
			vars = append(vars, key)
			seen[key] = true
		}
	}

	return vars
}

// CreateSecretProvider creates a SecretProvider based on SecretsConfig.
func CreateSecretProvider(cfg SecretsConfig) SecretProvider {
	var providers []SecretProvider

	switch strings.ToLower(cfg.Provider) {
	case "env":
		providers = append(providers, NewEnvSecretProvider())
	case "vault":
		// Vault provider would be implemented separately
		// For now, fall back to env
		providers = append(providers, NewEnvSecretProvider())
	case "aws-secrets-manager", "gcp-secret-manager":
		// Cloud providers would be implemented separately
		providers = append(providers, NewEnvSecretProvider())
	default:
		// Default to env
		providers = append(providers, NewEnvSecretProvider())
	}

	// Add env fallback if configured
	if cfg.FallbackToEnv && cfg.Provider != "env" {
		providers = append(providers, NewEnvSecretProvider())
	}

	if len(providers) == 1 {
		return providers[0]
	}

	return NewChainedSecretProvider(providers...)
}
