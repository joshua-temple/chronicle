package daemon

import (
	"sync"
	"time"
)

// EventType represents the type of event.
type EventType string

const (
	EventRunStarted    EventType = "run.started"
	EventRunCompleted  EventType = "run.completed"
	EventRunFailed     EventType = "run.failed"
	EventRunCanceled   EventType = "run.canceled"
	EventStepStarted   EventType = "step.started"
	EventStepCompleted EventType = "step.completed"
	EventStepFailed    EventType = "step.failed"
	EventConfigReload  EventType = "config.reload"
	EventServerStart   EventType = "server.start"
	EventServerStop    EventType = "server.stop"
)

// Event represents an event in the system.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      map[string]any
}

// EventHandler is a function that handles events.
type EventHandler func(Event)

// EventBus provides event publishing and subscription.
type EventBus interface {
	// Publish sends an event to all subscribers.
	Publish(event Event)

	// Subscribe registers a handler for events.
	Subscribe(handler EventHandler) func()

	// SubscribeType registers a handler for specific event types.
	SubscribeType(eventType EventType, handler EventHandler) func()
}

// EmbeddedEventBus is an in-memory event bus implementation.
type EmbeddedEventBus struct {
	mu           sync.RWMutex
	handlers     []EventHandler
	typeHandlers map[EventType][]EventHandler
	nextID       int
	handlerIDs   map[int]struct{}
}

// NewEmbeddedEventBus creates a new embedded event bus.
func NewEmbeddedEventBus() *EmbeddedEventBus {
	return &EmbeddedEventBus{
		typeHandlers: make(map[EventType][]EventHandler),
		handlerIDs:   make(map[int]struct{}),
	}
}

// Publish sends an event to all subscribers.
func (b *EmbeddedEventBus) Publish(event Event) {
	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers))
	copy(handlers, b.handlers)

	typeHandlers := make([]EventHandler, len(b.typeHandlers[event.Type]))
	copy(typeHandlers, b.typeHandlers[event.Type])
	b.mu.RUnlock()

	// Call general handlers
	for _, h := range handlers {
		go h(event)
	}

	// Call type-specific handlers
	for _, h := range typeHandlers {
		go h(event)
	}
}

// Subscribe registers a handler for all events.
func (b *EmbeddedEventBus) Subscribe(handler EventHandler) func() {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.handlers = append(b.handlers, handler)
	idx := len(b.handlers) - 1
	b.handlerIDs[id] = struct{}{}
	b.mu.Unlock()

	// Return unsubscribe function
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.handlerIDs[id]; ok {
			delete(b.handlerIDs, id)
			// Remove handler by index (if still valid)
			if idx < len(b.handlers) {
				b.handlers = append(b.handlers[:idx], b.handlers[idx+1:]...)
			}
		}
	}
}

// SubscribeType registers a handler for specific event types.
func (b *EmbeddedEventBus) SubscribeType(eventType EventType, handler EventHandler) func() {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.typeHandlers[eventType] = append(b.typeHandlers[eventType], handler)
	idx := len(b.typeHandlers[eventType]) - 1
	b.handlerIDs[id] = struct{}{}
	b.mu.Unlock()

	// Return unsubscribe function
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.handlerIDs[id]; ok {
			delete(b.handlerIDs, id)
			handlers := b.typeHandlers[eventType]
			if idx < len(handlers) {
				b.typeHandlers[eventType] = append(handlers[:idx], handlers[idx+1:]...)
			}
		}
	}
}

// BufferedEventBus buffers events for replay.
type BufferedEventBus struct {
	*EmbeddedEventBus
	mu      sync.RWMutex
	buffer  []Event
	maxSize int
}

// NewBufferedEventBus creates a buffered event bus.
func NewBufferedEventBus(maxSize int) *BufferedEventBus {
	return &BufferedEventBus{
		EmbeddedEventBus: NewEmbeddedEventBus(),
		maxSize:          maxSize,
		buffer:           make([]Event, 0, maxSize),
	}
}

// Publish sends an event and stores it in the buffer.
func (b *BufferedEventBus) Publish(event Event) {
	// Store in buffer
	b.mu.Lock()
	if len(b.buffer) >= b.maxSize {
		// Remove oldest event
		b.buffer = b.buffer[1:]
	}
	b.buffer = append(b.buffer, event)
	b.mu.Unlock()

	// Forward to embedded bus
	b.EmbeddedEventBus.Publish(event)
}

// GetHistory returns buffered events.
func (b *BufferedEventBus) GetHistory(since time.Time) []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []Event
	for _, e := range b.buffer {
		if e.Timestamp.After(since) {
			result = append(result, e)
		}
	}
	return result
}

// GetHistoryByType returns buffered events of a specific type.
func (b *BufferedEventBus) GetHistoryByType(eventType EventType, since time.Time) []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []Event
	for _, e := range b.buffer {
		if e.Type == eventType && e.Timestamp.After(since) {
			result = append(result, e)
		}
	}
	return result
}
