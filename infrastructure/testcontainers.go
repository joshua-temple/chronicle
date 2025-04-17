package infrastructure

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/docker/go-connections/nat"
	"github.com/joshua-temple/chronicle/core"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestContainersProvider is an infrastructure provider that uses the TestContainers library
type TestContainersProvider struct {
	services     map[string]core.Service
	requirements map[string]core.ServiceRequirement
	networks     map[string]*testcontainers.DockerNetwork
	containers   map[string]testcontainers.Container
	mutex        sync.Mutex
}

// NewTestContainersProvider creates a new TestContainers provider
func NewTestContainersProvider() *TestContainersProvider {
	return &TestContainersProvider{
		services:     make(map[string]core.Service),
		requirements: make(map[string]core.ServiceRequirement),
		networks:     make(map[string]*testcontainers.DockerNetwork),
		containers:   make(map[string]testcontainers.Container),
	}
}

// Initialize prepares the infrastructure
func (p *TestContainersProvider) Initialize(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Create default network if none exists
	if len(p.networks) == 0 {
		networkName := "chronicle-network"
		network, err := createNetwork(ctx, networkName)
		if err != nil {
			return fmt.Errorf("failed to create default network: %w", err)
		}
		p.networks[networkName] = network
	}

	return nil
}

// createNetwork creates a Docker network with the given name
func createNetwork(ctx context.Context, name string) (*testcontainers.DockerNetwork, error) {
	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:   name,
			Driver: "bridge",
		},
	})
	if err != nil {
		return nil, err
	}
	// Type assertion to DockerNetwork
	dockerNetwork, ok := network.(*testcontainers.DockerNetwork)
	if !ok {
		return nil, fmt.Errorf("failed to convert network to DockerNetwork")
	}
	return dockerNetwork, nil
}

// Start launches all required services
func (p *TestContainersProvider) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Start services in dependency order
	for name, req := range p.requirements {
		if _, exists := p.services[name]; exists {
			continue // Service already started
		}

		if err := p.startService(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

// startService starts a single service and its dependencies
func (p *TestContainersProvider) startService(ctx context.Context, req core.ServiceRequirement) error {
	// First, start dependencies
	for _, depName := range req.Dependencies {
		depReq, exists := p.requirements[depName]
		if !exists {
			return fmt.Errorf("dependency %s not found for service %s", depName, req.Name)
		}

		if _, exists := p.services[depName]; !exists {
			if err := p.startService(ctx, depReq); err != nil {
				return err
			}
		}
	}

	// Get the first network (or default)
	var networkName string
	for name := range p.networks {
		networkName = name
		break
	}

	// Create container configuration
	containerReq := testcontainers.ContainerRequest{
		Image:        req.Image,
		Name:         req.Name,
		ExposedPorts: []string{},
		Env:          req.Environment,
		Labels:       req.Labels,
		Networks:     []string{networkName},
		WaitingFor:   p.createWaitStrategy(req),
	}

	// Add exposed ports
	for _, port := range req.Ports {
		containerReq.ExposedPorts = append(
			containerReq.ExposedPorts,
			fmt.Sprintf("%d/%s", port.Internal, "tcp"),
		)
	}

	// Create and start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: containerReq,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start container for service %s: %w", req.Name, err)
	}

	// Store container reference
	p.containers[req.Name] = container

	// Create service interface
	svc := &testContainerService{
		name:         req.Name,
		image:        req.Image,
		ports:        req.Ports,
		env:          req.Environment,
		labels:       req.Labels,
		dependencies: req.Dependencies,
		waitStrategy: req.ReadinessProbe,
		container:    container,
		running:      true,
	}

	// Update mapped ports
	if err := p.updatePortMappings(ctx, svc); err != nil {
		return fmt.Errorf("failed to update port mappings for service %s: %w", req.Name, err)
	}

	p.services[req.Name] = svc
	return nil
}

// updatePortMappings updates the external port mappings for a service
func (p *TestContainersProvider) updatePortMappings(ctx context.Context, svc *testContainerService) error {
	for i, port := range svc.ports {
		// Get mapped port
		natPort, err := nat.NewPort("tcp", strconv.Itoa(port.Internal))
		if err != nil {
			return fmt.Errorf("invalid port specification: %w", err)
		}

		mappedPort, err := svc.container.MappedPort(ctx, natPort)
		if err != nil {
			return fmt.Errorf("failed to get mapped port: %w", err)
		}

		// Update port mapping
		portInt, err := strconv.Atoi(mappedPort.Port())
		if err != nil {
			return fmt.Errorf("invalid mapped port: %w", err)
		}

		svc.ports[i].External = portInt
	}
	return nil
}

// createWaitStrategy creates a wait strategy for the container
func (p *TestContainersProvider) createWaitStrategy(req core.ServiceRequirement) wait.Strategy {
	// If a custom wait strategy is provided, use it
	if req.ReadinessProbe != nil {
		return &customWaitStrategy{
			strategy: req.ReadinessProbe,
		}
	}

	// Default to waiting for the first exposed port
	for _, port := range req.Ports {
		return wait.ForListeningPort(nat.Port(fmt.Sprintf("%d/tcp", port.Internal)))
	}

	// If no ports, wait for log output indicating readiness
	return wait.ForLog("ready")
}

// customWaitStrategy adapts a Chronicle WaitStrategy to a TestContainers wait.Strategy
type customWaitStrategy struct {
	strategy core.WaitStrategy
}

// WaitUntilReady implements the wait.Strategy interface
func (s *customWaitStrategy) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	// Adapt the TestContainers target to our own context
	// In a real implementation, this would call the Chronicle WaitStrategy with appropriate context
	return s.strategy.Wait(ctx)
}

// Stop terminates all services
func (p *TestContainersProvider) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var lastErr error

	// Stop services in reverse dependency order
	for name, svc := range p.services {
		if err := svc.Stop(ctx); err != nil {
			lastErr = fmt.Errorf("failed to stop service %s: %w", name, err)
		}
	}

	// Clean up containers
	for name, container := range p.containers {
		if err := container.Terminate(ctx); err != nil {
			lastErr = fmt.Errorf("failed to terminate container %s: %w", name, err)
		}
	}

	// Remove networks
	for name, network := range p.networks {
		if err := network.Remove(ctx); err != nil {
			lastErr = fmt.Errorf("failed to remove network %s: %w", name, err)
		}
	}

	// Clear the maps
	p.services = make(map[string]core.Service)
	p.containers = make(map[string]testcontainers.Container)
	p.networks = make(map[string]*testcontainers.DockerNetwork)

	return lastErr
}

// AddService registers a service to be managed
func (p *TestContainersProvider) AddService(req core.ServiceRequirement) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.requirements[req.Name] = req
	return nil
}

// GetService retrieves a running service by name
func (p *TestContainersProvider) GetService(name string) (core.Service, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	svc, exists := p.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return svc, nil
}

// ServiceEndpoint returns connection info for a service
func (p *TestContainersProvider) ServiceEndpoint(serviceName string, portName string) (string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	svc, exists := p.services[serviceName]
	if !exists {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	// Find the port by name or number
	for _, port := range svc.Ports() {
		if port.Name == portName || strconv.Itoa(port.Internal) == portName {
			// Get host from container
			container, exists := p.containers[serviceName]
			if !exists {
				return "", fmt.Errorf("container for service %s not found", serviceName)
			}

			host, err := container.Host(context.Background())
			if err != nil {
				return "", fmt.Errorf("failed to get host for service %s: %w", serviceName, err)
			}

			return fmt.Sprintf("%s:%d", host, port.External), nil
		}
	}

	return "", fmt.Errorf("port %s not found for service %s", portName, serviceName)
}

// Flush resets service state but keeps services running
func (p *TestContainersProvider) Flush(ctx context.Context) error {
	// Implementation depends on what "flushing" means for each service
	// For example, clearing a database or queue
	// This is a minimal implementation
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for name, svc := range p.services {
		// For database services, we might run a command to clear data
		// For message queues, we might purge the queues
		// This is service-specific logic

		if svc.IsRunning() {
			// Example: For databases, execute a reset command
			if _, ok := svc.Labels()["type"]; ok && svc.Labels()["type"] == "database" {
				container, exists := p.containers[name]
				if exists {
					// This is a simplified example
					_, _, err := container.Exec(ctx, []string{"sh", "-c", "echo 'FLUSH ALL' | redis-cli"})
					if err != nil {
						return fmt.Errorf("failed to flush service %s: %w", name, err)
					}
				}
			}
		}
	}

	return nil
}

// testContainerService is an implementation of the Service interface backed by TestContainers
type testContainerService struct {
	name         string
	image        string
	ports        []core.Port
	env          map[string]string
	labels       map[string]string
	dependencies []string
	waitStrategy core.WaitStrategy
	container    testcontainers.Container
	running      bool
}

// Name returns the name of the service
func (s *testContainerService) Name() string {
	return s.name
}

// Image returns the Docker image name
func (s *testContainerService) Image() string {
	return s.image
}

// Ports returns the port mappings
func (s *testContainerService) Ports() []core.Port {
	return s.ports
}

// Env returns the environment variables
func (s *testContainerService) Env() map[string]string {
	return s.env
}

// Labels returns the Docker labels
func (s *testContainerService) Labels() map[string]string {
	return s.labels
}

// Dependencies returns the names of services this service depends on
func (s *testContainerService) Dependencies() []string {
	return s.dependencies
}

// WaitStrategy returns the strategy for waiting for the service to be ready
func (s *testContainerService) WaitStrategy() core.WaitStrategy {
	return s.waitStrategy
}

// Start starts the service
func (s *testContainerService) Start(ctx context.Context) error {
	// The container is already started by the provider, so this is a no-op
	s.running = true
	return nil
}

// Stop stops the service
func (s *testContainerService) Stop(ctx context.Context) error {
	s.running = false
	return nil
}

// IsRunning returns whether the service is running
func (s *testContainerService) IsRunning() bool {
	return s.running
}

// TestContainersWaitStrategy adapts the TestContainers wait.Strategy to Chronicle's WaitStrategy
type TestContainersWaitStrategy struct {
	testStrategy wait.Strategy
}

// Wait implements the WaitStrategy interface
func (s *TestContainersWaitStrategy) Wait(ctx context.Context) error {
	// This is a placeholder - the actual waiting is done by TestContainers
	return nil
}

// createTestContainersWaitStrategy creates a wait.Strategy from Chronicle's WaitStrategy
func createTestContainersWaitStrategy(waitStrategy core.WaitStrategy) wait.Strategy {
	if waitStrategy == nil {
		return wait.ForLog("ready")
	}

	// Handle specific wait strategy types
	switch s := waitStrategy.(type) {
	case *PortWaitStrategy:
		return wait.ForListeningPort(nat.Port(fmt.Sprintf("%d/tcp", s.Port)))
	case *HttpWaitStrategy:
		return wait.ForHTTP(s.Path).WithStatusCodeMatcher(func(status int) bool {
			return status == s.ExpectedCode
		})
	default:
		// Default to port or log strategy
		return wait.ForLog("ready")
	}
}
