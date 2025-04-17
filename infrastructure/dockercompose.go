package infrastructure

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joshua-temple/chronicle/core"
)

// DockerComposeProvider is an infrastructure provider that uses Docker Compose
type DockerComposeProvider struct {
	composeFilePath  string
	projectName      string
	services         map[string]core.Service
	runningServices  map[string]bool
	containerIDCache map[string]string
	mutex            sync.Mutex
}

// NewDockerComposeProvider creates a new Docker Compose provider
func NewDockerComposeProvider(composeFilePath string, projectName string) *DockerComposeProvider {
	return &DockerComposeProvider{
		composeFilePath:  composeFilePath,
		projectName:      projectName,
		services:         make(map[string]core.Service),
		runningServices:  make(map[string]bool),
		containerIDCache: make(map[string]string),
	}
}

// Initialize prepares the infrastructure
func (p *DockerComposeProvider) Initialize(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if compose file exists
	if _, err := os.Stat(p.composeFilePath); os.IsNotExist(err) {
		return fmt.Errorf("compose file %s does not exist", p.composeFilePath)
	}

	// Pull images
	cmd := exec.CommandContext(ctx, "docker-compose", "-f", p.composeFilePath, "-p", p.projectName, "pull")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull images: %w\nOutput: %s", err, output)
	}

	return nil
}

// Start launches all required services
func (p *DockerComposeProvider) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Start services
	cmd := exec.CommandContext(ctx, "docker-compose", "-f", p.composeFilePath, "-p", p.projectName, "up", "-d")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start services: %w\nOutput: %s", err, output)
	}

	// Get running services
	cmd = exec.CommandContext(ctx, "docker-compose", "-f", p.composeFilePath, "-p", p.projectName, "ps", "-q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list running services: %w\nOutput: %s", err, output)
	}

	// Parse container IDs
	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, containerID := range containerIDs {
		if containerID == "" {
			continue
		}

		// Get container name
		cmd = exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Name}}", containerID)
		nameOutput, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get container name: %w\nOutput: %s", err, nameOutput)
		}

		// Remove leading slash from container name
		containerName := strings.TrimPrefix(strings.TrimSpace(string(nameOutput)), "/")
		serviceName := strings.TrimPrefix(containerName, p.projectName+"_")
		serviceName = strings.Split(serviceName, "_")[0]

		p.containerIDCache[serviceName] = containerID
		p.runningServices[serviceName] = true

		// Create service object
		p.services[serviceName] = &dockerComposeService{
			name:        serviceName,
			containerID: containerID,
			provider:    p,
		}
	}

	// Wait for services to be ready
	// In a real implementation, we'd implement proper wait strategies
	time.Sleep(5 * time.Second)

	return nil
}

// Stop terminates all services
func (p *DockerComposeProvider) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Stop services
	cmd := exec.CommandContext(ctx, "docker-compose", "-f", p.composeFilePath, "-p", p.projectName, "down", "-v")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop services: %w\nOutput: %s", err, output)
	}

	// Clear maps
	p.services = make(map[string]core.Service)
	p.runningServices = make(map[string]bool)
	p.containerIDCache = make(map[string]string)

	return nil
}

// AddService registers a service to be managed
func (p *DockerComposeProvider) AddService(req core.ServiceRequirement) error {
	// For Docker Compose, services are defined in the compose file
	// This method doesn't need to do anything
	return nil
}

// GetService retrieves a running service by name
func (p *DockerComposeProvider) GetService(name string) (core.Service, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	service, exists := p.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// ServiceEndpoint returns connection info for a service
func (p *DockerComposeProvider) ServiceEndpoint(serviceName string, portName string) (string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	containerID, exists := p.containerIDCache[serviceName]
	if !exists {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	// Get container port mappings
	cmd := exec.Command("docker", "port", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get port mapping: %w\nOutput: %s", err, output)
	}

	// Find the requested port
	portMappings := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, mapping := range portMappings {
		parts := strings.Split(mapping, "->")
		if len(parts) != 2 {
			continue
		}

		hostPart := strings.TrimSpace(parts[0])
		containerPart := strings.TrimSpace(parts[1])

		// Extract container port and protocol
		containerPortParts := strings.Split(containerPart, "/")
		if len(containerPortParts) != 2 {
			continue
		}

		containerPort := containerPortParts[0]

		// Check if this is the port we're looking for
		if containerPort == portName {
			return hostPart, nil
		}
	}

	return "", fmt.Errorf("port %s not found for service %s", portName, serviceName)
}

// Flush resets service state but keeps services running
func (p *DockerComposeProvider) Flush(ctx context.Context) error {
	// Implementation would depend on what "flushing" means for each service
	// For example, clearing a database or queue
	return nil
}

// dockerComposeService represents a service managed by Docker Compose
type dockerComposeService struct {
	name        string
	containerID string
	provider    *DockerComposeProvider
	ports       []core.Port
}

// Name returns the name of the service
func (s *dockerComposeService) Name() string {
	return s.name
}

// Image returns the Docker image name
func (s *dockerComposeService) Image() string {
	// Get image name from container
	cmd := exec.Command("docker", "inspect", "--format", "{{.Config.Image}}", s.containerID)
	if output, err := cmd.CombinedOutput(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

// Ports returns the port mappings
func (s *dockerComposeService) Ports() []core.Port {
	if s.ports != nil {
		return s.ports
	}

	s.ports = []core.Port{}

	// Get port mappings from container
	cmd := exec.Command("docker", "port", s.containerID)
	if output, err := cmd.CombinedOutput(); err == nil {
		// Parse port mappings
		portMappings := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, mapping := range portMappings {
			parts := strings.Split(mapping, "->")
			if len(parts) == 2 {
				hostParts := strings.Split(strings.TrimSpace(parts[0]), ":")
				containerParts := strings.Split(strings.TrimSpace(parts[1]), "/")
				if len(hostParts) == 2 && len(containerParts) == 2 {
					hostPort := hostParts[1]
					containerPort := containerParts[0]

					// Convert ports to integers
					internalPort, _ := strconv.Atoi(containerPort)
					externalPort, _ := strconv.Atoi(hostPort)

					s.ports = append(s.ports, core.Port{
						Name:     containerPort,
						Internal: internalPort,
						External: externalPort,
					})
				}
			}
		}
	}

	return s.ports
}

// Env returns the environment variables
func (s *dockerComposeService) Env() map[string]string {
	env := make(map[string]string)

	// Get environment variables from container
	cmd := exec.Command("docker", "inspect", "--format", "{{range .Config.Env}}{{println .}}{{end}}", s.containerID)
	if output, err := cmd.CombinedOutput(); err == nil {
		envVars := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, envVar := range envVars {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}

	return env
}

// Labels returns the Docker labels
func (s *dockerComposeService) Labels() map[string]string {
	labels := make(map[string]string)

	// Get labels from container
	cmd := exec.Command("docker", "inspect", "--format", "{{range $k, $v := .Config.Labels}}{{$k}}={{$v}}\n{{end}}", s.containerID)
	if output, err := cmd.CombinedOutput(); err == nil {
		labelsList := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, label := range labelsList {
			if label == "" {
				continue
			}
			parts := strings.SplitN(label, "=", 2)
			if len(parts) == 2 {
				labels[parts[0]] = parts[1]
			}
		}
	}

	return labels
}

// Dependencies returns the names of services this service depends on
func (s *dockerComposeService) Dependencies() []string {
	// Dependencies are typically defined in the compose file
	// This would require parsing the compose file, which we'll skip for now
	return nil
}

// WaitStrategy returns the strategy for waiting for the service to be ready
func (s *dockerComposeService) WaitStrategy() core.WaitStrategy {
	// No specific wait strategy for Docker Compose services in this simplified implementation
	return nil
}

// Start starts the service
func (s *dockerComposeService) Start(ctx context.Context) error {
	// Start the service if it's not already running
	if !s.IsRunning() {
		cmd := exec.CommandContext(ctx, "docker-compose", "-f", s.provider.composeFilePath, "-p", s.provider.projectName, "start", s.name)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start service %s: %w\nOutput: %s", s.name, err, output)
		}

		s.provider.runningServices[s.name] = true
	}

	return nil
}

// Stop stops the service
func (s *dockerComposeService) Stop(ctx context.Context) error {
	// Stop the service if it's running
	if s.IsRunning() {
		cmd := exec.CommandContext(ctx, "docker-compose", "-f", s.provider.composeFilePath, "-p", s.provider.projectName, "stop", s.name)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to stop service %s: %w\nOutput: %s", s.name, err, output)
		}

		s.provider.runningServices[s.name] = false
	}

	return nil
}

// IsRunning returns whether the service is running
func (s *dockerComposeService) IsRunning() bool {
	// Check if the service is running
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", s.containerID)
	if output, err := cmd.CombinedOutput(); err == nil {
		return strings.TrimSpace(string(output)) == "true"
	}
	return false
}
