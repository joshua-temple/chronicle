package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Typed identifiers prevent mixing up IDs at compile time.

// TestID uniquely identifies a test.
type TestID string

// ScenarioID uniquely identifies a scenario.
type ScenarioID string

// ComponentID uniquely identifies a component.
type ComponentID string

// ServiceID uniquely identifies an infrastructure service.
type ServiceID string

// PortID identifies a service port.
type PortID string

// TagID identifies a tag for filtering/grouping.
type TagID string

// SuiteID uniquely identifies a test suite.
type SuiteID string

// EnvID identifies an environment.
type EnvID string

// TraceID uniquely identifies a distributed trace.
type TraceID string

// RunID uniquely identifies an execution run.
type RunID string

// SpanID identifies a span within a trace.
type SpanID string

// String implementations for all ID types.
func (id TestID) String() string      { return string(id) }
func (id ScenarioID) String() string  { return string(id) }
func (id ComponentID) String() string { return string(id) }
func (id ServiceID) String() string   { return string(id) }
func (id PortID) String() string      { return string(id) }
func (id TagID) String() string       { return string(id) }
func (id SuiteID) String() string     { return string(id) }
func (id EnvID) String() string       { return string(id) }
func (id TraceID) String() string     { return string(id) }
func (id RunID) String() string       { return string(id) }
func (id SpanID) String() string      { return string(id) }

// Validation methods.

// IsValid returns true if the TestID is non-empty.
func (id TestID) IsValid() bool { return id != "" }

// IsValid returns true if the ScenarioID is non-empty.
func (id ScenarioID) IsValid() bool { return id != "" }

// IsValid returns true if the ComponentID is non-empty.
func (id ComponentID) IsValid() bool { return id != "" }

// IsValid returns true if the ServiceID is non-empty.
func (id ServiceID) IsValid() bool { return id != "" }

// IsValid returns true if the TraceID is non-empty.
func (id TraceID) IsValid() bool { return id != "" }

// IsValid returns true if the RunID is non-empty.
func (id RunID) IsValid() bool { return id != "" }

// IsValid returns true if the SpanID is non-empty.
func (id SpanID) IsValid() bool { return id != "" }

// ID generation functions.

// NewTestID generates a new unique TestID.
func NewTestID() TestID {
	return TestID(generateID("test"))
}

// NewScenarioID generates a new unique ScenarioID.
func NewScenarioID() ScenarioID {
	return ScenarioID(generateID("scn"))
}

// NewComponentID creates a ComponentID from a name.
func NewComponentID(name string) ComponentID {
	return ComponentID(name)
}

// NewServiceID creates a ServiceID from a name.
func NewServiceID(name string) ServiceID {
	return ServiceID(name)
}

// NewTraceID generates a new unique TraceID.
func NewTraceID() TraceID {
	return TraceID(generateID("trace"))
}

// NewRunID generates a new unique RunID.
func NewRunID() RunID {
	return RunID(generateID("run"))
}

// NewSpanID generates a new unique SpanID.
func NewSpanID() SpanID {
	return SpanID(generateID("span"))
}

// generateID creates a unique ID with a prefix.
func generateID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

// TraceContext carries trace information through execution.
type TraceContext struct {
	TraceID    TraceID           // Unique trace for this execution
	SpanID     SpanID            // Current span within trace
	ParentSpan SpanID            // Parent span (for nested components)
	Baggage    map[string]string // Key-value pairs to propagate
	StartTime  time.Time
}

// NewTraceContext creates a new TraceContext with a fresh TraceID.
func NewTraceContext() *TraceContext {
	return &TraceContext{
		TraceID:   NewTraceID(),
		SpanID:    NewSpanID(),
		Baggage:   make(map[string]string),
		StartTime: time.Now(),
	}
}

// NewSpan creates a child span within this trace.
func (tc *TraceContext) NewSpan(name string) *TraceContext {
	return &TraceContext{
		TraceID:    tc.TraceID,
		SpanID:     NewSpanID(),
		ParentSpan: tc.SpanID,
		Baggage:    copyBaggage(tc.Baggage),
		StartTime:  time.Now(),
	}
}

// SetBaggage adds a baggage item to propagate.
func (tc *TraceContext) SetBaggage(key, value string) {
	if tc.Baggage == nil {
		tc.Baggage = make(map[string]string)
	}
	tc.Baggage[key] = value
}

// GetBaggage retrieves a baggage item.
func (tc *TraceContext) GetBaggage(key string) (string, bool) {
	if tc.Baggage == nil {
		return "", false
	}
	v, ok := tc.Baggage[key]
	return v, ok
}

// Duration returns the time elapsed since the trace started.
func (tc *TraceContext) Duration() time.Duration {
	return time.Since(tc.StartTime)
}

func copyBaggage(src map[string]string) map[string]string {
	if src == nil {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// IDRegistry maintains a registry of all active identifiers.
type IDRegistry struct {
	mu         sync.RWMutex
	tests      map[TestID]bool
	scenarios  map[ScenarioID]bool
	components map[ComponentID]bool
	services   map[ServiceID]bool
	traces     map[TraceID]*TraceContext
}

// NewIDRegistry creates a new IDRegistry.
func NewIDRegistry() *IDRegistry {
	return &IDRegistry{
		tests:      make(map[TestID]bool),
		scenarios:  make(map[ScenarioID]bool),
		components: make(map[ComponentID]bool),
		services:   make(map[ServiceID]bool),
		traces:     make(map[TraceID]*TraceContext),
	}
}

// RegisterTest registers a TestID, returning an error if it already exists.
func (r *IDRegistry) RegisterTest(id TestID) error {
	if !id.IsValid() {
		return fmt.Errorf("invalid test ID: empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tests[id] {
		return fmt.Errorf("duplicate test ID: %s", id)
	}
	r.tests[id] = true
	return nil
}

// UnregisterTest removes a TestID from the registry.
func (r *IDRegistry) UnregisterTest(id TestID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tests, id)
}

// HasTest checks if a TestID is registered.
func (r *IDRegistry) HasTest(id TestID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tests[id]
}

// RegisterScenario registers a ScenarioID, returning an error if it already exists.
func (r *IDRegistry) RegisterScenario(id ScenarioID) error {
	if !id.IsValid() {
		return fmt.Errorf("invalid scenario ID: empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.scenarios[id] {
		return fmt.Errorf("duplicate scenario ID: %s", id)
	}
	r.scenarios[id] = true
	return nil
}

// UnregisterScenario removes a ScenarioID from the registry.
func (r *IDRegistry) UnregisterScenario(id ScenarioID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.scenarios, id)
}

// HasScenario checks if a ScenarioID is registered.
func (r *IDRegistry) HasScenario(id ScenarioID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scenarios[id]
}

// RegisterComponent registers a ComponentID, returning an error if it already exists.
func (r *IDRegistry) RegisterComponent(id ComponentID) error {
	if !id.IsValid() {
		return fmt.Errorf("invalid component ID: empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.components[id] {
		return fmt.Errorf("duplicate component ID: %s", id)
	}
	r.components[id] = true
	return nil
}

// UnregisterComponent removes a ComponentID from the registry.
func (r *IDRegistry) UnregisterComponent(id ComponentID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.components, id)
}

// HasComponent checks if a ComponentID is registered.
func (r *IDRegistry) HasComponent(id ComponentID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.components[id]
}

// RegisterService registers a ServiceID, returning an error if it already exists.
func (r *IDRegistry) RegisterService(id ServiceID) error {
	if !id.IsValid() {
		return fmt.Errorf("invalid service ID: empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.services[id] {
		return fmt.Errorf("duplicate service ID: %s", id)
	}
	r.services[id] = true
	return nil
}

// UnregisterService removes a ServiceID from the registry.
func (r *IDRegistry) UnregisterService(id ServiceID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, id)
}

// HasService checks if a ServiceID is registered.
func (r *IDRegistry) HasService(id ServiceID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[id]
}

// RegisterTrace registers a TraceContext, returning an error if the TraceID already exists.
func (r *IDRegistry) RegisterTrace(tc *TraceContext) error {
	if tc == nil || !tc.TraceID.IsValid() {
		return fmt.Errorf("invalid trace context: nil or empty TraceID")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.traces[tc.TraceID]; exists {
		return fmt.Errorf("duplicate trace ID: %s", tc.TraceID)
	}
	r.traces[tc.TraceID] = tc
	return nil
}

// UnregisterTrace removes a TraceContext from the registry.
func (r *IDRegistry) UnregisterTrace(id TraceID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.traces, id)
}

// GetTrace retrieves a TraceContext by its TraceID.
func (r *IDRegistry) GetTrace(id TraceID) (*TraceContext, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tc, ok := r.traces[id]
	return tc, ok
}

// Clear removes all registered IDs from the registry.
func (r *IDRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tests = make(map[TestID]bool)
	r.scenarios = make(map[ScenarioID]bool)
	r.components = make(map[ComponentID]bool)
	r.services = make(map[ServiceID]bool)
	r.traces = make(map[TraceID]*TraceContext)
}

// Stats returns counts of registered IDs.
func (r *IDRegistry) Stats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return map[string]int{
		"tests":      len(r.tests),
		"scenarios":  len(r.scenarios),
		"components": len(r.components),
		"services":   len(r.services),
		"traces":     len(r.traces),
	}
}
