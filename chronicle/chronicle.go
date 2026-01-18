package chronicle

import (
	"sync"
	"time"
)

// Chronicle records the execution narrative
type Chronicle struct {
	root     *Entry
	current  *Entry
	stack    []*Entry
	disabled bool
	mu       sync.Mutex
}

// New creates a new chronicle recorder
func New(scenarioName, description string) *Chronicle {
	root := &Entry{
		Name:        scenarioName,
		Description: description,
		Type:        ScenarioEntry,
		StartTime:   time.Now(),
		Status:      Passed,
		Children:    make([]*Entry, 0),
	}
	return &Chronicle{
		root:    root,
		current: root,
		stack:   []*Entry{root},
	}
}

// noop is a shared disabled chronicle
var noop = &Chronicle{disabled: true}

// Noop returns a disabled chronicle (safe to call methods on)
func Noop() *Chronicle {
	return noop
}

// Begin starts a new entry as a child of the current entry
// Returns a function to close the entry (for defer pattern)
func (c *Chronicle) Begin(name, description string, entryType EntryType) func(error) {
	if c.disabled {
		return func(error) {}
	}

	c.mu.Lock()
	entry := &Entry{
		Name:        name,
		Description: description,
		Type:        entryType,
		StartTime:   time.Now(),
		Status:      Passed,
		Children:    make([]*Entry, 0),
	}
	c.current.Children = append(c.current.Children, entry)
	c.stack = append(c.stack, entry)
	c.current = entry
	c.mu.Unlock()

	return func(err error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		entry.Duration = time.Since(entry.StartTime)
		if err != nil {
			entry.Status = Failed
			entry.Error = err.Error()
			c.propagateFailure()
		}
		c.stack = c.stack[:len(c.stack)-1]
		if len(c.stack) > 0 {
			c.current = c.stack[len(c.stack)-1]
		}
	}
}

// propagateFailure marks all ancestors as failed
func (c *Chronicle) propagateFailure() {
	for _, entry := range c.stack {
		entry.Status = Failed
	}
}

// Root returns the complete chronicle tree
func (c *Chronicle) Root() *Entry {
	return c.root
}

// Finalize marks the chronicle as complete
func (c *Chronicle) Finalize() {
	if c.disabled || c.root == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.root.Duration = time.Since(c.root.StartTime)
}

// Component opens a component entry
func (c *Chronicle) Component(name, description string) func(error) {
	return c.Begin(name, description, ComponentEntry)
}

// Setup opens a setup entry
func (c *Chronicle) Setup(name, description string) func(error) {
	return c.Begin(name, description, SetupEntry)
}

// Task opens a task entry
func (c *Chronicle) Task(name, description string) func(error) {
	return c.Begin(name, description, TaskEntry)
}

// Teardown opens a teardown entry
func (c *Chronicle) Teardown(name, description string) func(error) {
	return c.Begin(name, description, TeardownEntry)
}

// Expect records an expectation check (leaf node)
func (c *Chronicle) Expect(description string, passed bool) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	status := Passed
	if !passed {
		status = Failed
		c.propagateFailure()
	}

	c.current.Children = append(c.current.Children, &Entry{
		Name:      description,
		Type:      ExpectEntry,
		StartTime: time.Now(),
		Status:    status,
	})
}

// Skip marks that an operation was skipped
func (c *Chronicle) Skip(name, reason string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.current.Children = append(c.current.Children, &Entry{
		Name:        name,
		Description: reason,
		Status:      Skipped,
		Type:        ComponentEntry,
	})
}
