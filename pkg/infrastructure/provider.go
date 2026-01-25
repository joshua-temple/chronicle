package infrastructure

import (
	"context"
	"sync"
	"time"
)

// ProviderStatus represents the current state of an infrastructure provider.
type ProviderStatus int

const (
	// StatusStopped indicates the provider is not running.
	StatusStopped ProviderStatus = iota
	// StatusStarting indicates the provider is initializing.
	StatusStarting
	// StatusRunning indicates the provider is healthy and running.
	StatusRunning
	// StatusUnhealthy indicates the provider is running but unhealthy.
	StatusUnhealthy
	// StatusStopping indicates the provider is shutting down.
	StatusStopping
)

func (s ProviderStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

// ServiceHealth represents the health status of a single service.
type ServiceHealth struct {
	Name    string
	Status  string // "healthy", "unhealthy", "starting", "stopped"
	Latency time.Duration
	Error   error
}

// HealthReport contains health information for all services managed by a provider.
type HealthReport struct {
	Healthy  bool
	Message  string
	Services map[string]ServiceHealth
}

// Provider defines the interface that all infrastructure providers must implement.
// The framework is unopinionated about HOW infrastructure comes up - providers implement this interface.
type Provider interface {
	// Name returns the unique identifier for this provider.
	Name() string

	// Initialize prepares the provider with the given configuration.
	// This should validate config but not start any resources.
	Initialize(ctx context.Context, config map[string]any) error

	// Start brings up the infrastructure resources.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the infrastructure resources.
	Stop(ctx context.Context) error

	// HealthCheck returns the current health status of all managed services.
	HealthCheck(ctx context.Context) HealthReport

	// Status returns the provider's current lifecycle status.
	Status() ProviderStatus

	// Client returns a client connection to the named service.
	// The returned type is provider-specific (e.g., *sql.DB for postgres).
	Client(name string) (any, error)
}

// FlushConfig specifies how to reset provider state between test runs.
type FlushConfig struct {
	Strategy string         // Provider-specific strategy (e.g., "truncate", "flushdb")
	Include  []string       // Items to flush
	Exclude  []string       // Items to preserve
	Options  map[string]any // Additional provider-specific options
}

// FlushableProvider extends Provider with the ability to reset state
// while keeping infrastructure running.
type FlushableProvider interface {
	Provider

	// Flush resets state using the default strategy.
	Flush(ctx context.Context) error

	// FlushWithConfig resets state using the specified configuration.
	FlushWithConfig(ctx context.Context, config FlushConfig) error
}

// ReconfigurableProvider extends Provider with hot-reload capabilities.
type ReconfigurableProvider interface {
	Provider

	// Reconfigure updates the provider with new configuration without restart.
	// Used for secret rotation and dynamic config changes.
	Reconfigure(ctx context.Context, config map[string]any) error
}

// ReuseBehavior defines how infrastructure is managed between test executions.
type ReuseBehavior int

const (
	// AlwaysFresh destroys and recreates infrastructure for each test.
	// Slowest, but maximum isolation.
	AlwaysFresh ReuseBehavior = iota

	// ReuseWithFlush keeps infrastructure alive but flushes state between tests.
	// Fast startup, good isolation (data cleared).
	ReuseWithFlush

	// FullReuse keeps infrastructure alive and state intact.
	// Fastest, useful for debugging or sequential test dependencies.
	FullReuse
)

func (r ReuseBehavior) String() string {
	switch r {
	case AlwaysFresh:
		return "always_fresh"
	case ReuseWithFlush:
		return "flush"
	case FullReuse:
		return "full"
	default:
		return "unknown"
	}
}

// ParseReuseBehavior converts a string to ReuseBehavior.
func ParseReuseBehavior(s string) ReuseBehavior {
	switch s {
	case "always_fresh":
		return AlwaysFresh
	case "flush":
		return ReuseWithFlush
	case "full":
		return FullReuse
	default:
		return ReuseWithFlush // Default to flush
	}
}

// IsolationLevel defines the level of isolation between tests.
type IsolationLevel int

const (
	// NoIsolation - tests share state (use with FullReuse).
	NoIsolation IsolationLevel = iota

	// DataIsolation - flush data between tests.
	DataIsolation

	// SchemaIsolation - separate schemas per test.
	SchemaIsolation

	// InstanceIsolation - separate container instances per test.
	InstanceIsolation
)

func (i IsolationLevel) String() string {
	switch i {
	case NoIsolation:
		return "none"
	case DataIsolation:
		return "data"
	case SchemaIsolation:
		return "schema"
	case InstanceIsolation:
		return "instance"
	default:
		return "unknown"
	}
}

// ParseIsolationLevel converts a string to IsolationLevel.
func ParseIsolationLevel(s string) IsolationLevel {
	switch s {
	case "none":
		return NoIsolation
	case "data":
		return DataIsolation
	case "schema":
		return SchemaIsolation
	case "instance":
		return InstanceIsolation
	default:
		return DataIsolation // Default to data isolation
	}
}

// ProviderConfig holds configuration for a single provider instance.
type ProviderConfig struct {
	Name      string
	Provider  string         // Provider type (e.g., "postgres", "redis")
	Config    map[string]any // Provider-specific configuration
	Reuse     ReuseBehavior
	Isolation IsolationLevel
	Flush     FlushConfig
}

// Registry tracks available provider factories.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ProviderFactory
}

// ProviderFactory creates new provider instances.
type ProviderFactory func() Provider

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ProviderFactory),
	}
}

// Register adds a provider factory to the registry.
func (r *Registry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Create creates a new provider instance by name.
func (r *Registry) Create(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[name]
	if !ok {
		return nil, false
	}
	return factory(), true
}

// Available returns a list of registered provider names.
func (r *Registry) Available() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry is the global provider registry.
var DefaultRegistry = NewRegistry()

// RegisterProvider registers a provider factory with the default registry.
func RegisterProvider(name string, factory ProviderFactory) {
	DefaultRegistry.Register(name, factory)
}
