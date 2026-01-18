package chronicle

import "time"

// EntryType categorizes the operation
type EntryType int

const (
	ScenarioEntry  EntryType = iota // Top-level scenario
	ComponentEntry                  // Named component within scenario
	SetupEntry                      // Setup operation
	TaskEntry                       // Task/action operation
	ExpectEntry                     // Expectation check
	TeardownEntry                   // Teardown operation
)

func (t EntryType) String() string {
	switch t {
	case ScenarioEntry:
		return "Scenario"
	case ComponentEntry:
		return "Component"
	case SetupEntry:
		return "Setup"
	case TaskEntry:
		return "Task"
	case ExpectEntry:
		return "Expect"
	case TeardownEntry:
		return "Teardown"
	default:
		return "Unknown"
	}
}

// Status indicates the outcome (terminal states only)
type Status int

const (
	Passed  Status = iota // Completed successfully
	Failed                // Completed with error
	Skipped               // Did not execute (condition not met)
)

func (s Status) String() string {
	switch s {
	case Passed:
		return "Passed"
	case Failed:
		return "Failed"
	case Skipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// Entry represents a single node in the chronicle tree
type Entry struct {
	Name        string        `json:"name" yaml:"name"`
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
	Type        EntryType     `json:"type" yaml:"type"`
	Children    []*Entry      `json:"children,omitempty" yaml:"children,omitempty"`
	StartTime   time.Time     `json:"start_time" yaml:"start_time"`
	Duration    time.Duration `json:"duration" yaml:"duration"`
	Status      Status        `json:"status" yaml:"status"`
	Error       string        `json:"error,omitempty" yaml:"error,omitempty"`
}
