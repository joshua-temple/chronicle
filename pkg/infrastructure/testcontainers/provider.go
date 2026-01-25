package testcontainers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ContainerConfig holds configuration for a container.
type ContainerConfig struct {
	Image        string
	ExposedPorts []string
	Env          map[string]string
	WaitFor      wait.Strategy
	StartupWait  time.Duration
}

// ContainerProvider is a base provider for testcontainers-based infrastructure.
type ContainerProvider struct {
	name      string
	container testcontainers.Container
	config    ContainerConfig
	mu        sync.RWMutex
	status    atomic.Int32
	clients   map[string]any
}

// NewContainerProvider creates a new container provider.
func NewContainerProvider(name string) *ContainerProvider {
	return &ContainerProvider{
		name:    name,
		clients: make(map[string]any),
	}
}

// Name returns the provider name.
func (p *ContainerProvider) Name() string {
	return p.name
}

// Initialize configures the provider from a config map.
func (p *ContainerProvider) Initialize(ctx context.Context, config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = ContainerConfig{
		ExposedPorts: []string{},
		Env:          make(map[string]string),
		StartupWait:  30 * time.Second,
	}

	if image, ok := config["image"].(string); ok {
		p.config.Image = image
	}

	if ports, ok := config["ports"].([]any); ok {
		for _, port := range ports {
			if portStr, ok := port.(string); ok {
				p.config.ExposedPorts = append(p.config.ExposedPorts, portStr)
			}
		}
	}

	if env, ok := config["env"].(map[string]any); ok {
		for k, v := range env {
			if vStr, ok := v.(string); ok {
				p.config.Env[k] = vStr
			}
		}
	}

	if wait, ok := config["startup_wait"].(time.Duration); ok {
		p.config.StartupWait = wait
	}

	return nil
}

// SetConfig sets the container configuration directly.
func (p *ContainerProvider) SetConfig(config ContainerConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
}

// Start starts the container.
func (p *ContainerProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status.Store(int32(infrastructure.StatusStarting))

	req := testcontainers.ContainerRequest{
		Image:        p.config.Image,
		ExposedPorts: p.config.ExposedPorts,
		Env:          p.config.Env,
		WaitingFor:   p.config.WaitFor,
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		p.status.Store(int32(infrastructure.StatusStopped))
		return fmt.Errorf("failed to start container: %w", err)
	}

	p.container = container
	p.status.Store(int32(infrastructure.StatusRunning))

	return nil
}

// Stop stops and removes the container.
func (p *ContainerProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.container == nil {
		p.status.Store(int32(infrastructure.StatusStopped))
		return nil
	}

	p.status.Store(int32(infrastructure.StatusStopping))

	if err := p.container.Terminate(ctx); err != nil {
		return fmt.Errorf("failed to terminate container: %w", err)
	}

	p.container = nil
	p.status.Store(int32(infrastructure.StatusStopped))

	return nil
}

// HealthCheck returns the health status of the container.
func (p *ContainerProvider) HealthCheck(ctx context.Context) infrastructure.HealthReport {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.container == nil {
		return infrastructure.HealthReport{
			Healthy: false,
			Message: "container not started",
			Services: map[string]infrastructure.ServiceHealth{
				p.name: {
					Name:   p.name,
					Status: "stopped",
				},
			},
		}
	}

	// Check if container is running
	state, err := p.container.State(ctx)
	if err != nil {
		return infrastructure.HealthReport{
			Healthy: false,
			Message: fmt.Sprintf("failed to get container state: %v", err),
			Services: map[string]infrastructure.ServiceHealth{
				p.name: {
					Name:   p.name,
					Status: "unhealthy",
					Error:  err,
				},
			},
		}
	}

	if !state.Running {
		return infrastructure.HealthReport{
			Healthy: false,
			Message: "container not running",
			Services: map[string]infrastructure.ServiceHealth{
				p.name: {
					Name:   p.name,
					Status: "stopped",
				},
			},
		}
	}

	return infrastructure.HealthReport{
		Healthy: true,
		Message: "container running",
		Services: map[string]infrastructure.ServiceHealth{
			p.name: {
				Name:   p.name,
				Status: "healthy",
			},
		},
	}
}

// Status returns the provider status.
func (p *ContainerProvider) Status() infrastructure.ProviderStatus {
	return infrastructure.ProviderStatus(p.status.Load())
}

// Client returns a client by name.
func (p *ContainerProvider) Client(name string) (any, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if name == "" {
		name = "default"
	}

	client, ok := p.clients[name]
	if !ok {
		return nil, fmt.Errorf("client %q not found", name)
	}

	return client, nil
}

// SetClient registers a client with the provider.
func (p *ContainerProvider) SetClient(name string, client any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[name] = client
}

// Container returns the underlying testcontainers container.
func (p *ContainerProvider) Container() testcontainers.Container {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.container
}

// Host returns the container host.
func (p *ContainerProvider) Host(ctx context.Context) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.container == nil {
		return "", fmt.Errorf("container not started")
	}

	return p.container.Host(ctx)
}

// MappedPort returns the mapped port for a given container port.
func (p *ContainerProvider) MappedPort(ctx context.Context, port string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.container == nil {
		return "", fmt.Errorf("container not started")
	}

	mappedPort, err := p.container.MappedPort(ctx, nat.Port(port))
	if err != nil {
		return "", fmt.Errorf("failed to get mapped port: %w", err)
	}

	return mappedPort.Port(), nil
}
