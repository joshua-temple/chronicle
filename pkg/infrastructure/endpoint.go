package infrastructure

import (
	"fmt"
	"sync"
)

// Endpoint represents a reachable service endpoint.
type Endpoint struct {
	Host         string            // External hostname (e.g., "localhost")
	Port         int               // External mapped port
	InternalHost string            // Container network hostname (e.g., "postgres")
	InternalPort int               // Container internal port
	Protocol     string            // "tcp", "http", "grpc"
	Metadata     map[string]string // Additional info (user, database, etc.)
}

// Address returns host:port for external access from the host machine.
func (e Endpoint) Address() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

// InternalAddress returns the container-network address for container-to-container communication.
func (e Endpoint) InternalAddress() string {
	return fmt.Sprintf("%s:%d", e.InternalHost, e.InternalPort)
}

// EndpointRegistry tracks all available service endpoints.
type EndpointRegistry struct {
	mu        sync.RWMutex
	endpoints map[string]Endpoint
}

// NewEndpointRegistry creates a new endpoint registry.
func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[string]Endpoint),
	}
}

// Register adds or updates an endpoint in the registry.
func (r *EndpointRegistry) Register(name string, ep Endpoint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endpoints[name] = ep
}

// Get retrieves an endpoint by name.
func (r *EndpointRegistry) Get(name string) (Endpoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ep, ok := r.endpoints[name]
	return ep, ok
}

// All returns a copy of all registered endpoints.
func (r *EndpointRegistry) All() map[string]Endpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]Endpoint, len(r.endpoints))
	for k, v := range r.endpoints {
		result[k] = v
	}
	return result
}

// Names returns all registered endpoint names.
func (r *EndpointRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.endpoints))
	for name := range r.endpoints {
		names = append(names, name)
	}
	return names
}
