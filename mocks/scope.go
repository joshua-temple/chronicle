package mocks

import "github.com/joshua-temple/chronicle/core"

// GlobalScope represents suite-level stubs (applied once, cleared at suite end)
type GlobalScope struct{}

// IsGlobal returns true for GlobalScope
func (GlobalScope) IsGlobal() bool { return true }

// TestID returns empty for GlobalScope
func (GlobalScope) TestID() core.TestID { return "" }

// TestScope represents test-level stubs (applied per-test, cleared after test)
type TestScope struct {
	testID core.TestID
}

// NewTestScope creates a test-scoped scope
func NewTestScope(id core.TestID) TestScope {
	return TestScope{testID: id}
}

// IsGlobal returns false for TestScope
func (s TestScope) IsGlobal() bool { return false }

// TestID returns the test ID
func (s TestScope) TestID() core.TestID { return s.testID }
