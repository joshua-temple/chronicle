package metrics

import (
	"context"
	"sync"
)

// MemoryCollector stores events in memory (for testing/assertions)
type MemoryCollector struct {
	events []Event
	mu     sync.Mutex
}

// NewMemoryCollector creates a memory collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{events: make([]Event, 0)}
}

// Collect stores the event in memory
func (m *MemoryCollector) Collect(ctx context.Context, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

// Flush is a no-op for memory collector
func (m *MemoryCollector) Flush(ctx context.Context) error {
	return nil
}

// Close is a no-op for memory collector
func (m *MemoryCollector) Close() error {
	return nil
}

// Events returns all collected events
func (m *MemoryCollector) Events() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event{}, m.events...)
}

// Clear removes all collected events
func (m *MemoryCollector) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]Event, 0)
}

// ScenarioCompletedEvents returns all ScenarioCompleted events
func (m *MemoryCollector) ScenarioCompletedEvents() []*ScenarioCompleted {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*ScenarioCompleted
	for _, e := range m.events {
		if typed, ok := e.(*ScenarioCompleted); ok {
			result = append(result, typed)
		}
	}
	return result
}

// ComponentCompletedEvents returns all ComponentCompleted events
func (m *MemoryCollector) ComponentCompletedEvents() []*ComponentCompleted {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*ComponentCompleted
	for _, e := range m.events {
		if typed, ok := e.(*ComponentCompleted); ok {
			result = append(result, typed)
		}
	}
	return result
}

// CustomEvents returns all Custom events
func (m *MemoryCollector) CustomEvents() []*Custom {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Custom
	for _, e := range m.events {
		if typed, ok := e.(*Custom); ok {
			result = append(result, typed)
		}
	}
	return result
}
