package mocks

import (
	"context"

	"github.com/joshua-temple/chronicle/core"
)

// MockClient is the interface to a mock service
// Users implement this for their chosen mock technology
type MockClient interface {
	// Apply adds stubs to the mock service
	Apply(ctx context.Context, stubs ...Stub) error

	// Remove removes specific stubs by ID
	Remove(ctx context.Context, stubIDs ...string) error

	// Clear removes all stubs for a given scope
	Clear(ctx context.Context, scope Scope) error
}

// Stub represents a mock definition
// Users implement this per their technology
type Stub interface {
	// ID returns a unique identifier for this stub
	ID() string

	// Scope returns when this stub should be active
	Scope() Scope
}

// Scope identifies the lifecycle of a stub
type Scope interface {
	// IsGlobal returns true if this is a suite-level stub
	IsGlobal() bool

	// TestID returns the test ID if test-scoped, empty if global
	TestID() core.TestID
}
