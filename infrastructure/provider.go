package infrastructure

import (
	"context"
	"time"

	"github.com/joshua-temple/chronicle/core"
)

// PortWaitStrategy waits for a specific port to be available
type PortWaitStrategy struct {
	Port    int
	Timeout time.Duration
}

// Wait waits for the port to be available
func (s *PortWaitStrategy) Wait(ctx context.Context) error {
	// Implementation to wait for port to be available
	return nil
}

// HttpWaitStrategy waits for an HTTP endpoint to return a specific status code
type HttpWaitStrategy struct {
	Path         string
	ExpectedCode int
	Timeout      time.Duration
}

// Wait waits for the HTTP endpoint to return the expected status code
func (s *HttpWaitStrategy) Wait(ctx context.Context) error {
	// Implementation to wait for HTTP endpoint
	return nil
}

// NewPortWaitStrategy creates a new port wait strategy
func NewPortWaitStrategy(port int, timeout time.Duration) core.WaitStrategy {
	return &PortWaitStrategy{
		Port:    port,
		Timeout: timeout,
	}
}

// NewHttpWaitStrategy creates a new HTTP wait strategy
func NewHttpWaitStrategy(path string, expectedCode int, timeout time.Duration) core.WaitStrategy {
	return &HttpWaitStrategy{
		Path:         path,
		ExpectedCode: expectedCode,
		Timeout:      timeout,
	}
}

// BuildPortWaitStrategy builds a core.ServiceRequirement with a port wait strategy
func BuildPortWaitStrategy(serviceReq *core.ServiceRequirement, port int, timeout time.Duration) {
	serviceReq.ReadinessProbe = NewPortWaitStrategy(port, timeout)
}

// BuildHttpWaitStrategy builds a core.ServiceRequirement with an HTTP wait strategy
func BuildHttpWaitStrategy(serviceReq *core.ServiceRequirement, path string, expectedCode int, timeout time.Duration) {
	serviceReq.ReadinessProbe = NewHttpWaitStrategy(path, expectedCode, timeout)
}

// NewServiceRequirement creates a new service requirement
func NewServiceRequirement(name, image string) core.ServiceRequirement {
	return core.ServiceRequirement{
		Name:            name,
		Image:           image,
		Environment:     make(map[string]string),
		Labels:          make(map[string]string),
		Dependencies:    []string{},
		StartupTimeout:  30 * time.Second,
		CleanupBehavior: core.Remove,
	}
}

// AddPort adds a port to a service requirement
func AddPort(serviceReq *core.ServiceRequirement, name string, internal int) {
	serviceReq.Ports = append(serviceReq.Ports, core.Port{
		Name:     name,
		Internal: internal,
	})
}

// AddEnv adds an environment variable to a service requirement
func AddEnv(serviceReq *core.ServiceRequirement, key, value string) {
	serviceReq.Environment[key] = value
}

// AddLabel adds a label to a service requirement
func AddLabel(serviceReq *core.ServiceRequirement, key, value string) {
	serviceReq.Labels[key] = value
}

// AddDependency adds a dependency to a service requirement
func AddDependency(serviceReq *core.ServiceRequirement, dependency string) {
	serviceReq.Dependencies = append(serviceReq.Dependencies, dependency)
}

// SetResourceLimits sets resource limits for a service requirement
func SetResourceLimits(serviceReq *core.ServiceRequirement, cpuPercent, memoryMB int) {
	serviceReq.ResourceLimits = &core.ResourceLimits{
		CPUPercent: cpuPercent,
		MemoryMB:   memoryMB,
	}
}

// SetCleanupBehavior sets the cleanup behavior for a service requirement
func SetCleanupBehavior(serviceReq *core.ServiceRequirement, behavior core.CleanupBehavior) {
	serviceReq.CleanupBehavior = behavior
}
