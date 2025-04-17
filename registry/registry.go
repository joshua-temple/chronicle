package registry

import (
	"fmt"
	"sync"

	"github.com/joshua-temple/chronicle/core"
)

// InfrastructureConfiguration defines a complete infrastructure setup
type InfrastructureConfiguration struct {
	Name     string
	Services []core.ServiceRequirement
	Networks []NetworkRequirement
}

// NetworkRequirement defines requirements for a network
type NetworkRequirement struct {
	Name   string
	Driver string
}

// BundleRegistry provides a central registry for templates and bundles
type BundleRegistry struct {
	infrastructureTemplates map[string]InfrastructureConfiguration
	middleware              map[string]core.MiddlewareFunc
	flagBundles             map[string][]string
	optionBundles           map[string][]core.TestOption
	mutex                   sync.RWMutex
}

// NewBundleRegistry creates a new bundle registry
func NewBundleRegistry() *BundleRegistry {
	return &BundleRegistry{
		infrastructureTemplates: make(map[string]InfrastructureConfiguration),
		middleware:              make(map[string]core.MiddlewareFunc),
		flagBundles:             make(map[string][]string),
		optionBundles:           make(map[string][]core.TestOption),
	}
}

// RegisterInfrastructure registers an infrastructure configuration
func (r *BundleRegistry) RegisterInfrastructure(config InfrastructureConfiguration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.infrastructureTemplates[config.Name] = config
}

// GetInfrastructure retrieves an infrastructure configuration by name
func (r *BundleRegistry) GetInfrastructure(name string) (InfrastructureConfiguration, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	config, exists := r.infrastructureTemplates[name]
	if !exists {
		return InfrastructureConfiguration{}, fmt.Errorf("infrastructure template %s not found", name)
	}

	return config, nil
}

// RegisterMiddleware registers a middleware function
func (r *BundleRegistry) RegisterMiddleware(name string, middleware core.MiddlewareFunc) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.middleware[name] = middleware
}

// GetMiddleware retrieves a middleware function by name
func (r *BundleRegistry) GetMiddleware(name string) (core.MiddlewareFunc, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	middleware, exists := r.middleware[name]
	if !exists {
		return nil, fmt.Errorf("middleware %s not found", name)
	}

	return middleware, nil
}

// RegisterFlagBundle registers a bundle of feature flags
func (r *BundleRegistry) RegisterFlagBundle(name string, flags []string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.flagBundles[name] = flags
}

// GetFlagBundle retrieves a bundle of feature flags by name
func (r *BundleRegistry) GetFlagBundle(name string) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	flags, exists := r.flagBundles[name]
	if !exists {
		return nil, fmt.Errorf("flag bundle %s not found", name)
	}

	return flags, nil
}

// RegisterOptionBundle registers a bundle of test options
func (r *BundleRegistry) RegisterOptionBundle(name string, options []core.TestOption) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.optionBundles[name] = options
}

// GetOptionBundle retrieves a bundle of test options by name
func (r *BundleRegistry) GetOptionBundle(name string) ([]core.TestOption, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	options, exists := r.optionBundles[name]
	if !exists {
		return nil, fmt.Errorf("option bundle %s not found", name)
	}

	return options, nil
}

// CreateInfrastructureProvider creates an infrastructure provider from an infrastructure configuration
func (r *BundleRegistry) CreateInfrastructureProvider(config InfrastructureConfiguration, providerType string) (core.InfrastructureProvider, error) {
	switch providerType {
	case "testcontainers":
		provider := NewTestContainersProvider()
		for _, service := range config.Services {
			if err := provider.AddService(service); err != nil {
				return nil, fmt.Errorf("failed to add service %s: %w", service.Name, err)
			}
		}
		return provider, nil
	case "docker-compose":
		// For docker-compose, we'd typically use a docker-compose.yml file
		// This is simplified for demonstration
		return nil, fmt.Errorf("docker-compose provider not yet implemented in registry")
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// NewTestContainersProvider creates a new TestContainers provider
func NewTestContainersProvider() core.InfrastructureProvider {
	// This is a simplified implementation for the registry
	// In a real implementation, we'd import from the infrastructure package
	// But we want to avoid circular dependencies
	return nil
}

// Global registry instance
var Registry = NewBundleRegistry()
