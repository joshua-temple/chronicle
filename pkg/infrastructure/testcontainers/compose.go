package testcontainers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

// ComposeConfig holds configuration for a compose-based provider.
type ComposeConfig struct {
	File     string   // Path to docker-compose.yml
	Services []string // Optional: specific services to start
}

// ComposeProvider manages infrastructure defined in a docker-compose file.
type ComposeProvider struct {
	name        string
	config      ComposeConfig
	compose     *compose.DockerCompose
	networkName string
	mu          sync.RWMutex
	status      atomic.Int32
	endpoints   *infrastructure.EndpointRegistry
}

// NewComposeProvider creates a new compose-based provider.
func NewComposeProvider(name string) *ComposeProvider {
	return &ComposeProvider{
		name: name,
	}
}

// Name returns the provider name.
func (p *ComposeProvider) Name() string {
	return p.name
}

// SetNetwork sets the Docker network for containers to join.
func (p *ComposeProvider) SetNetwork(networkName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.networkName = networkName
}

// Network returns the current network name.
func (p *ComposeProvider) Network() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.networkName
}

// SetEndpointRegistry sets the registry for endpoint registration.
func (p *ComposeProvider) SetEndpointRegistry(registry *infrastructure.EndpointRegistry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.endpoints = registry
}

// Initialize configures the provider from a config map.
func (p *ComposeProvider) Initialize(ctx context.Context, config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if file, ok := config["compose_file"].(string); ok {
		p.config.File = file
	}

	if services, ok := config["services"].([]any); ok {
		for _, s := range services {
			if svc, ok := s.(string); ok {
				p.config.Services = append(p.config.Services, svc)
			}
		}
	}

	if p.config.File == "" {
		return errors.New("compose_file is required")
	}

	return nil
}

// Start brings up the compose stack.
func (p *ComposeProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status.Store(int32(infrastructure.StatusStarting))

	// Create compose instance
	composeInstance, err := compose.NewDockerCompose(p.config.File)
	if err != nil {
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to create compose instance: %w", err)
	}

	// Start with wait
	if err := composeInstance.Up(ctx, compose.Wait(true)); err != nil {
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to start compose stack: %w", err)
	}

	p.compose = composeInstance
	p.status.Store(int32(infrastructure.StatusRunning))

	// TODO: Register endpoints for each service

	return nil
}

// Stop shuts down the compose stack.
func (p *ComposeProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.compose == nil {
		p.status.Store(int32(infrastructure.StatusStopped))
		return nil
	}

	p.status.Store(int32(infrastructure.StatusStopping))

	if err := p.compose.Down(ctx); err != nil {
		p.status.Store(int32(infrastructure.StatusUnhealthy))
		return fmt.Errorf("failed to stop compose stack: %w", err)
	}

	p.compose = nil
	p.status.Store(int32(infrastructure.StatusStopped))

	return nil
}

// HealthCheck returns health status.
// TODO: Query actual service health via compose API for more accurate reporting.
func (p *ComposeProvider) HealthCheck(ctx context.Context) infrastructure.HealthReport {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.compose == nil {
		return infrastructure.HealthReport{
			Healthy:  false,
			Message:  "compose stack not started",
			Services: map[string]infrastructure.ServiceHealth{},
		}
	}

	return infrastructure.HealthReport{
		Healthy:  true,
		Message:  "compose stack running",
		Services: map[string]infrastructure.ServiceHealth{},
	}
}

// Status returns the provider status.
func (p *ComposeProvider) Status() infrastructure.ProviderStatus {
	return infrastructure.ProviderStatus(p.status.Load())
}

// Client returns a client by name.
// ComposeProvider does not support direct clients; use endpoints instead.
func (p *ComposeProvider) Client(name string) (any, error) {
	return nil, fmt.Errorf("ComposeProvider does not support direct clients; use endpoints to connect to services")
}
