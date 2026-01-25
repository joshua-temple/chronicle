package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Manager coordinates multiple infrastructure providers.
type Manager struct {
	mu        sync.RWMutex
	providers map[string]Provider
	configs   map[string]ProviderConfig
	registry  *Registry
	endpoints *EndpointRegistry
	reuse     ReuseBehavior
	isolation IsolationLevel
	started   bool
}

// NewManager creates a new infrastructure manager.
func NewManager(registry *Registry) *Manager {
	if registry == nil {
		registry = DefaultRegistry
	}
	return &Manager{
		providers: make(map[string]Provider),
		configs:   make(map[string]ProviderConfig),
		registry:  registry,
		endpoints: NewEndpointRegistry(),
		reuse:     ReuseWithFlush,
		isolation: DataIsolation,
	}
}

// SetDefaultReuse sets the default reuse behavior for all providers.
func (m *Manager) SetDefaultReuse(behavior ReuseBehavior) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reuse = behavior
}

// SetDefaultIsolation sets the default isolation level for all providers.
func (m *Manager) SetDefaultIsolation(level IsolationLevel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isolation = level
}

// Configure adds a provider configuration to the manager.
func (m *Manager) Configure(config ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.Name == "" {
		return errors.New("provider name is required")
	}

	// Create provider from registry
	provider, ok := m.registry.Create(config.Provider)
	if !ok {
		return fmt.Errorf("unknown provider type: %s", config.Provider)
	}

	m.providers[config.Name] = provider
	m.configs[config.Name] = config

	return nil
}

// AddProvider adds an already-instantiated provider to the manager.
func (m *Manager) AddProvider(name string, provider Provider, config ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		return errors.New("provider name is required")
	}
	if provider == nil {
		return errors.New("provider cannot be nil")
	}

	m.providers[name] = provider
	m.configs[name] = config

	return nil
}

// Start initializes and starts all configured providers.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil // Already started
	}

	var errs []error

	// Initialize all providers first
	for name, provider := range m.providers {
		config := m.configs[name]
		if err := provider.Initialize(ctx, config.Config); err != nil {
			errs = append(errs, fmt.Errorf("failed to initialize %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Start all providers
	for name, provider := range m.providers {
		if err := provider.Start(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to start %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		// Stop any started providers on failure
		for _, provider := range m.providers {
			if provider.Status() == StatusRunning {
				_ = provider.Stop(ctx)
			}
		}
		return errors.Join(errs...)
	}

	m.started = true
	return nil
}

// Stop gracefully shuts down all providers.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil // Already stopped
	}

	var errs []error

	for name, provider := range m.providers {
		if err := provider.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop %s: %w", name, err))
		}
	}

	m.started = false

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Reset resets provider state between scenarios based on reuse behavior.
func (m *Manager) Reset(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errs []error

	for name, provider := range m.providers {
		config := m.configs[name]
		reuse := config.Reuse
		if reuse == 0 {
			reuse = m.reuse // Use manager default
		}

		switch reuse {
		case AlwaysFresh:
			// Stop and restart the provider
			if err := provider.Stop(ctx); err != nil {
				errs = append(errs, fmt.Errorf("failed to stop %s: %w", name, err))
				continue
			}
			if err := provider.Initialize(ctx, config.Config); err != nil {
				errs = append(errs, fmt.Errorf("failed to reinitialize %s: %w", name, err))
				continue
			}
			if err := provider.Start(ctx); err != nil {
				errs = append(errs, fmt.Errorf("failed to restart %s: %w", name, err))
			}

		case ReuseWithFlush:
			// Flush data if provider supports it
			if flushable, ok := provider.(FlushableProvider); ok {
				if err := flushable.FlushWithConfig(ctx, config.Flush); err != nil {
					errs = append(errs, fmt.Errorf("failed to flush %s: %w", name, err))
				}
			}

		case FullReuse:
			// Do nothing - keep state intact
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Client returns a client connection to the specified service.
func (m *Manager) Client(serviceName, clientName string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[serviceName]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s", serviceName)
	}

	if provider.Status() != StatusRunning {
		return nil, fmt.Errorf("service %s is not running", serviceName)
	}

	return provider.Client(clientName)
}

// ClientByService returns the default client for a service.
func (m *Manager) ClientByService(serviceName string) (any, error) {
	return m.Client(serviceName, "")
}

// HealthCheck returns health reports for all providers.
func (m *Manager) HealthCheck(ctx context.Context) map[string]HealthReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	reports := make(map[string]HealthReport)
	for name, provider := range m.providers {
		reports[name] = provider.HealthCheck(ctx)
	}
	return reports
}

// AllHealthy returns true if all providers are healthy.
func (m *Manager) AllHealthy(ctx context.Context) bool {
	reports := m.HealthCheck(ctx)
	for _, report := range reports {
		if !report.Healthy {
			return false
		}
	}
	return true
}

// Provider returns a specific provider by name.
func (m *Manager) Provider(name string) (Provider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	provider, ok := m.providers[name]
	return provider, ok
}

// ProviderNames returns all configured provider names.
func (m *Manager) ProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// Status returns status for all providers.
func (m *Manager) Status() map[string]ProviderStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	statuses := make(map[string]ProviderStatus)
	for name, provider := range m.providers {
		statuses[name] = provider.Status()
	}
	return statuses
}

// IsStarted returns whether the manager has been started.
func (m *Manager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// Endpoints returns the endpoint registry.
func (m *Manager) Endpoints() *EndpointRegistry {
	return m.endpoints
}

// Endpoint retrieves an endpoint by name.
func (m *Manager) Endpoint(name string) (Endpoint, bool) {
	return m.endpoints.Get(name)
}
