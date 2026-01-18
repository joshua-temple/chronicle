package metrics

import (
	"time"

	"github.com/joshua-temple/chronicle/core"
)

// Event is the base interface for all metric events
type Event interface {
	Timestamp() time.Time
	Tags() map[string]string
}

// baseEvent provides common fields
type baseEvent struct {
	ts   time.Time
	tags map[string]string
}

func (e baseEvent) Timestamp() time.Time    { return e.ts }
func (e baseEvent) Tags() map[string]string { return e.tags }

func newBaseEvent() baseEvent {
	return baseEvent{
		ts:   time.Now(),
		tags: make(map[string]string),
	}
}

// Status for metric events
type Status int

const (
	StatusPassed Status = iota
	StatusFailed
	StatusSkipped
)

func (s Status) String() string {
	switch s {
	case StatusPassed:
		return "Passed"
	case StatusFailed:
		return "Failed"
	case StatusSkipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// --- Execution Events ---

// ScenarioStarted emitted when a scenario begins
type ScenarioStarted struct {
	baseEvent
	ScenarioID   string
	ScenarioName string
}

// ScenarioCompleted emitted when a scenario finishes
type ScenarioCompleted struct {
	baseEvent
	ScenarioID   string
	ScenarioName string
	Duration     time.Duration
	Status       Status
	Error        string
}

// ComponentCompleted emitted when a component finishes
type ComponentCompleted struct {
	baseEvent
	ScenarioID    string
	ComponentName string
	Duration      time.Duration
	Status        Status
	Error         string
}

// TestCompleted emitted when a suite test finishes
type TestCompleted struct {
	baseEvent
	SuiteID  core.SuiteID
	TestID   core.TestID
	Duration time.Duration
	Status   Status
	Retries  int
	Error    string
}

// SuiteCompleted emitted when a suite finishes
type SuiteCompleted struct {
	baseEvent
	SuiteID  core.SuiteID
	Duration time.Duration
	Passed   int
	Failed   int
	Skipped  int
}

// --- Infrastructure Events ---

// ServiceStarted emitted when an infrastructure service starts
type ServiceStarted struct {
	baseEvent
	ServiceID   core.ServiceID
	ServiceName string
	Duration    time.Duration
}

// ServiceStopped emitted when a service stops
type ServiceStopped struct {
	baseEvent
	ServiceID   core.ServiceID
	ServiceName string
	Uptime      time.Duration
}

// ServiceHealthCheck emitted on health check results
type ServiceHealthCheck struct {
	baseEvent
	ServiceID   core.ServiceID
	ServiceName string
	Healthy     bool
	Latency     time.Duration
}

// --- Custom Events ---

// Custom allows users to emit their own metrics
type Custom struct {
	baseEvent
	Name  string
	Value float64
	Unit  string
}
