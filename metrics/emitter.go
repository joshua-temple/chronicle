package metrics

import (
	"context"
	"sync"
	"time"
)

// Emitter sends metric events to collectors
type Emitter struct {
	collector Collector
	baseTags  map[string]string
	mu        sync.RWMutex
}

// NewEmitter creates an emitter with a collector
func NewEmitter(collector Collector) *Emitter {
	return &Emitter{
		collector: collector,
		baseTags:  make(map[string]string),
	}
}

// noopEmitter is a shared no-op emitter
var noopEmitter = &Emitter{}

// NoopEmitter returns an emitter that does nothing
func NoopEmitter() *Emitter {
	return noopEmitter
}

// WithTags adds base tags applied to all events
func (e *Emitter) WithTags(tags map[string]string) *Emitter {
	e.mu.Lock()
	defer e.mu.Unlock()
	for k, v := range tags {
		e.baseTags[k] = v
	}
	return e
}

// Emit sends an event to the collector
func (e *Emitter) Emit(ctx context.Context, event Event) error {
	if e.collector == nil {
		return nil
	}
	return e.collector.Collect(ctx, e.mergeTags(event))
}

func (e *Emitter) mergeTags(event Event) Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	eventTags := event.Tags()
	if eventTags == nil {
		return event
	}
	for k, v := range e.baseTags {
		if _, exists := eventTags[k]; !exists {
			eventTags[k] = v
		}
	}
	return event
}

// ScenarioStart records a scenario starting
func (e *Emitter) ScenarioStart(ctx context.Context, id, name string) {
	e.Emit(ctx, &ScenarioStarted{
		baseEvent:    newBaseEvent(),
		ScenarioID:   id,
		ScenarioName: name,
	})
}

// ScenarioEnd records a scenario completing
func (e *Emitter) ScenarioEnd(ctx context.Context, id, name string, duration time.Duration, status Status, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	e.Emit(ctx, &ScenarioCompleted{
		baseEvent:    newBaseEvent(),
		ScenarioID:   id,
		ScenarioName: name,
		Duration:     duration,
		Status:       status,
		Error:        errMsg,
	})
}

// ComponentEnd records a component completing
func (e *Emitter) ComponentEnd(ctx context.Context, scenarioID, componentName string, duration time.Duration, status Status, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	e.Emit(ctx, &ComponentCompleted{
		baseEvent:     newBaseEvent(),
		ScenarioID:    scenarioID,
		ComponentName: componentName,
		Duration:      duration,
		Status:        status,
		Error:         errMsg,
	})
}

// Custom emits a user-defined metric
func (e *Emitter) Custom(ctx context.Context, name string, value float64, unit string, tags map[string]string) {
	event := &Custom{
		baseEvent: newBaseEvent(),
		Name:      name,
		Value:     value,
		Unit:      unit,
	}
	for k, v := range tags {
		event.tags[k] = v
	}
	e.Emit(ctx, event)
}

// Timer returns a function that records duration when called
func (e *Emitter) Timer(ctx context.Context, name string, tags map[string]string) func() {
	start := time.Now()
	return func() {
		e.Custom(ctx, name, float64(time.Since(start).Milliseconds()), "ms", tags)
	}
}

// Counter increments a counter metric
func (e *Emitter) Counter(ctx context.Context, name string, delta float64, tags map[string]string) {
	e.Custom(ctx, name, delta, "count", tags)
}

// Gauge sets a gauge metric
func (e *Emitter) Gauge(ctx context.Context, name string, value float64, unit string, tags map[string]string) {
	e.Custom(ctx, name, value, unit, tags)
}
