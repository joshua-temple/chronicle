package chronicle

import (
	"sync"
	"time"
)

// SuiteChronicle aggregates multiple scenario chronicles
type SuiteChronicle struct {
	Name        string
	Description string
	Scenarios   []*Entry
	StartTime   time.Time
	Duration    time.Duration
	mu          sync.Mutex
}

// NewSuiteChronicle creates a suite-level aggregator
func NewSuiteChronicle(name, description string) *SuiteChronicle {
	return &SuiteChronicle{
		Name:        name,
		Description: description,
		Scenarios:   make([]*Entry, 0),
		StartTime:   time.Now(),
	}
}

// Add appends a completed scenario chronicle
func (sc *SuiteChronicle) Add(entry *Entry) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.Scenarios = append(sc.Scenarios, entry)
}

// Finalize sets the total duration
func (sc *SuiteChronicle) Finalize() {
	sc.Duration = time.Since(sc.StartTime)
}

// Summary returns pass/fail/skip counts
func (sc *SuiteChronicle) Summary() (passed, failed, skipped int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for _, s := range sc.Scenarios {
		switch s.Status {
		case Passed:
			passed++
		case Failed:
			failed++
		case Skipped:
			skipped++
		}
	}
	return
}
