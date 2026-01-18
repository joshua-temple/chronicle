package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/joshua-temple/chronicle/core"
)

// MockRegistry manages mock clients and stub definitions
type MockRegistry struct {
	clients     map[string]MockClient
	globalStubs map[string][]Stub
	mu          sync.RWMutex
}

// NewMockRegistry creates a new registry
func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		clients:     make(map[string]MockClient),
		globalStubs: make(map[string][]Stub),
	}
}

// RegisterClient adds a mock client
func (r *MockRegistry) RegisterClient(name string, client MockClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[name] = client
}

// RegisterGlobalStubs registers stubs to be applied at suite start
func (r *MockRegistry) RegisterGlobalStubs(clientName string, stubs ...Stub) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.globalStubs[clientName] = append(r.globalStubs[clientName], stubs...)
}

// ApplyGlobal applies all global stubs
func (r *MockRegistry) ApplyGlobal(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, stubs := range r.globalStubs {
		client, ok := r.clients[name]
		if !ok {
			return fmt.Errorf("mock client %s not registered", name)
		}
		if err := client.Apply(ctx, stubs...); err != nil {
			return fmt.Errorf("applying global stubs for %s: %w", name, err)
		}
	}
	return nil
}

// ClearGlobal clears all global stubs
func (r *MockRegistry) ClearGlobal(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, client := range r.clients {
		if err := client.Clear(ctx, GlobalScope{}); err != nil {
			return fmt.Errorf("clearing global stubs for %s: %w", name, err)
		}
	}
	return nil
}

// ApplyForTest applies test-scoped stubs
func (r *MockRegistry) ApplyForTest(ctx context.Context, testID core.TestID, stubs ...Stub) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Group stubs by client (based on which client they belong to)
	// For now, apply to all registered clients
	for name, client := range r.clients {
		if err := client.Apply(ctx, stubs...); err != nil {
			return fmt.Errorf("applying test stubs for %s: %w", name, err)
		}
	}
	return nil
}

// ClearForTest clears test-scoped stubs
func (r *MockRegistry) ClearForTest(ctx context.Context, testID core.TestID) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	scope := NewTestScope(testID)
	for name, client := range r.clients {
		if err := client.Clear(ctx, scope); err != nil {
			return fmt.Errorf("clearing test stubs for %s: %w", name, err)
		}
	}
	return nil
}

// Client returns a registered mock client by name
func (r *MockRegistry) Client(name string) (MockClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	client, ok := r.clients[name]
	return client, ok
}
