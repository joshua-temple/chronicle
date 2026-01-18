package suite

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/joshua-temple/chronicle/chronicle"
	"github.com/joshua-temple/chronicle/core"
	"github.com/joshua-temple/chronicle/metrics"
	"github.com/joshua-temple/chronicle/mocks"
	"github.com/joshua-temple/chronicle/scenarios"
)

// StateAccessLevel defines how a test can interact with suite-level state
type StateAccessLevel int

const (
	// ReadOnly allows test to read but not modify suite state
	ReadOnly StateAccessLevel = iota

	// ReadWrite allows test to read and modify suite state
	ReadWrite

	// Isolated gives test a copy of suite state, but changes aren't propagated back
	Isolated
)

// RetryPolicy defines how and when a test should be retried
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int

	// BackoffStrategy defines how to delay between retries
	BackoffStrategy BackoffStrategy

	// RetryCondition is a function that determines if an error should trigger a retry
	RetryCondition func(error) bool
}

// BackoffStrategy defines how to delay between retries
type BackoffStrategy interface {
	// NextDelay returns the next delay duration based on the attempt number
	NextDelay(attempt int) time.Duration
}

// FixedBackoff implements a fixed delay backoff strategy
type FixedBackoff struct {
	Delay time.Duration
}

// NextDelay returns the fixed delay regardless of attempt number
func (b *FixedBackoff) NextDelay(attempt int) time.Duration {
	return b.Delay
}

// ExponentialBackoff implements an exponential backoff strategy
type ExponentialBackoff struct {
	InitialDelay time.Duration
	Factor       float64
	MaxDelay     time.Duration
}

// NextDelay returns an exponentially increasing delay
func (b *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	// Calculate exponential delay
	delay := float64(b.InitialDelay) * pow(b.Factor, float64(attempt-1))

	// Cap at max delay
	if delay > float64(b.MaxDelay) {
		return b.MaxDelay
	}

	return time.Duration(delay)
}

// Helper function for exponential calculation
func pow(x, y float64) float64 {
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	return result
}

// SuiteHook represents lifecycle hooks for the suite
type SuiteHook interface {
	// BeforeSuite is called before the suite starts
	BeforeSuite(ctx context.Context, s *Suite) error

	// AfterSuite is called after the suite completes
	AfterSuite(ctx context.Context, s *Suite) error

	// BeforeTest is called before each test starts
	BeforeTest(ctx context.Context, s *Suite, t *Test) error

	// AfterTest is called after each test completes
	AfterTest(ctx context.Context, s *Suite, t *Test, result *TestResult) error

	// OnTestFailure is called when a test fails
	OnTestFailure(ctx context.Context, s *Suite, t *Test, err error) error
}

// ConfigRule allows for dynamic configuration of tests based on conditions
type ConfigRule interface {
	// EvaluateTest determines if this rule applies to a given test
	EvaluateTest(test *Test) bool

	// ModifyTest modifies a test's configuration
	ModifyTest(test *Test)
}

// TestResult contains the result of a test execution
type TestResult struct {
	// Test is the test that was executed
	Test *Test

	// Success indicates whether the test succeeded
	Success bool

	// Error is the error that occurred, if any
	Error error

	// StartTime is when the test started
	StartTime time.Time

	// Duration is how long the test took to execute
	Duration time.Duration

	// Attempt is the attempt number (for retries)
	Attempt int

	// State captures test-specific state
	State map[string]interface{}
}

// Suite represents a collection of related tests with shared infrastructure
type Suite struct {
	// ID is the unique identifier for this suite
	ID core.SuiteID

	// Description provides details about the suite
	Description string

	// InfraProvider manages infrastructure for the suite
	InfraProvider core.InfrastructureProvider

	// GlobalSetup contains components run before any tests
	GlobalSetup []scenarios.Component

	// GlobalTeardown contains components run after all tests
	GlobalTeardown []scenarios.Component

	// SharedState stores data shared across tests
	SharedState *core.DataStore

	// Tests contains all tests in this suite
	Tests []*Test

	// Hooks contains lifecycle hooks for the suite
	Hooks []SuiteHook

	// ConfigurationRules contains rules for dynamic test configuration
	ConfigurationRules []ConfigRule

	// Parameters contains suite-level parameters
	Parameters map[string]interface{}

	// Timeout is the maximum duration for the entire suite
	Timeout time.Duration

	// MockRegistry manages mock clients and stubs
	MockRegistry *mocks.MockRegistry

	// MetricsCollector receives metrics events
	MetricsCollector metrics.Collector

	// ChronicleEnabled enables chronicle recording
	ChronicleEnabled bool

	// suiteChronicle aggregates test chronicles
	suiteChronicle *chronicle.SuiteChronicle

	// TestRegistry holds all tests by ID
	testRegistry map[core.TestID]*Test

	// ComponentRegistry holds all components
	componentRegistry map[core.ComponentID]scenarios.Component

	// Mutex protects concurrent access
	mutex sync.RWMutex
}

// Test represents an individual test within a suite
type Test struct {
	// ID is the unique identifier for this test
	ID core.TestID

	// Description provides details about the test
	Description string

	// Components contains the test's components to execute
	Components []core.ComponentID

	// Dependencies specifies test IDs that must run before this test
	Dependencies []core.TestID

	// RequiredServices specifies services this test needs
	RequiredServices []core.ServiceID

	// StateAccess defines how this test interacts with suite state
	StateAccess StateAccessLevel

	// Timeout is the maximum duration for this test
	Timeout time.Duration

	// RetryPolicy defines how to handle failures
	RetryPolicy *RetryPolicy

	// Tags are metadata tags for categorization
	Tags []core.TagID

	// Parallel indicates if this test can run in parallel with others
	Parallel bool

	// Skip indicates if this test should be skipped
	Skip bool

	// SkipReason explains why this test is skipped
	SkipReason string

	// Parameters contains test-specific parameters
	Parameters map[string]interface{}

	// parent suite reference
	suite *Suite
}

// NewSuite creates a new test suite
func NewSuite(id core.SuiteID) *Suite {
	return &Suite{
		ID:                 id,
		SharedState:        core.NewDataStore(),
		Tests:              []*Test{},
		Hooks:              []SuiteHook{},
		ConfigurationRules: []ConfigRule{},
		Parameters:         make(map[string]interface{}),
		testRegistry:       make(map[core.TestID]*Test),
		componentRegistry:  make(map[core.ComponentID]scenarios.Component),
	}
}

// WithDescription adds a description to the suite
func (s *Suite) WithDescription(description string) *Suite {
	s.Description = description
	return s
}

// WithInfraProvider sets the infrastructure provider
func (s *Suite) WithInfraProvider(provider core.InfrastructureProvider) *Suite {
	s.InfraProvider = provider
	return s
}

// WithGlobalSetup adds global setup components
func (s *Suite) WithGlobalSetup(components ...scenarios.Component) *Suite {
	s.GlobalSetup = append(s.GlobalSetup, components...)
	return s
}

// WithGlobalTeardown adds global teardown components
func (s *Suite) WithGlobalTeardown(components ...scenarios.Component) *Suite {
	s.GlobalTeardown = append(s.GlobalTeardown, components...)
	return s
}

// WithHooks adds lifecycle hooks
func (s *Suite) WithHooks(hooks ...SuiteHook) *Suite {
	s.Hooks = append(s.Hooks, hooks...)
	return s
}

// WithConfigurationRules adds configuration rules
func (s *Suite) WithConfigurationRules(rules ...ConfigRule) *Suite {
	s.ConfigurationRules = append(s.ConfigurationRules, rules...)
	return s
}

// WithTimeout sets the suite timeout
func (s *Suite) WithTimeout(timeout time.Duration) *Suite {
	s.Timeout = timeout
	return s
}

// WithMockRegistry sets the mock registry
func (s *Suite) WithMockRegistry(registry *mocks.MockRegistry) *Suite {
	s.MockRegistry = registry
	return s
}

// WithMetrics sets the metrics collector
func (s *Suite) WithMetrics(collector metrics.Collector) *Suite {
	s.MetricsCollector = collector
	return s
}

// WithChronicle enables chronicle recording
func (s *Suite) WithChronicle() *Suite {
	s.ChronicleEnabled = true
	return s
}

// SuiteChronicle returns the suite chronicle if enabled
func (s *Suite) SuiteChronicle() *chronicle.SuiteChronicle {
	return s.suiteChronicle
}

// AddTest adds a test to the suite
func (s *Suite) AddTest(test *Test) *Suite {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Set parent suite reference
	test.suite = s

	// Register the test
	s.Tests = append(s.Tests, test)
	s.testRegistry[test.ID] = test

	return s
}

// AddTests adds multiple tests to the suite
func (s *Suite) AddTests(tests ...*Test) *Suite {
	for _, test := range tests {
		s.AddTest(test)
	}
	return s
}

// GetTest retrieves a test by ID
func (s *Suite) GetTest(id core.TestID) (*Test, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	test, exists := s.testRegistry[id]
	return test, exists
}

// RegisterComponent registers a component
func (s *Suite) RegisterComponent(id core.ComponentID, component scenarios.Component) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.componentRegistry[id] = component
}

// GetComponent retrieves a component by ID
func (s *Suite) GetComponent(id core.ComponentID) (scenarios.Component, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	component, exists := s.componentRegistry[id]
	return component, exists
}

// Run executes the entire suite
func (s *Suite) Run(ctx context.Context) error {
	start := time.Now()

	// Initialize suite chronicle if enabled
	if s.ChronicleEnabled {
		s.suiteChronicle = chronicle.NewSuiteChronicle(string(s.ID), s.Description)
	}

	// Create a context with timeout if specified
	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}

	// Run before suite hooks
	for _, hook := range s.Hooks {
		if err := hook.BeforeSuite(ctx, s); err != nil {
			return fmt.Errorf("before suite hook failed: %w", err)
		}
	}

	// Initialize and start infrastructure
	if s.InfraProvider != nil {
		stop := s.InfraProvider.Start(ctx)
		defer stop(ctx)
	}

	// Apply global mocks
	if s.MockRegistry != nil {
		if err := s.MockRegistry.ApplyGlobal(ctx); err != nil {
			return fmt.Errorf("failed to apply global mocks: %w", err)
		}
		defer s.MockRegistry.ClearGlobal(ctx)
	}

	// Run global setup
	for i, setup := range s.GlobalSetup {
		scenarioCtx := scenarios.NewContext()
		if err := setup(scenarioCtx); err != nil {
			return fmt.Errorf("global setup step %d failed: %w", i+1, err)
		}
	}

	// Calculate test execution order (based on dependencies)
	executionOrder, err := s.calculateExecutionOrder()
	if err != nil {
		return err
	}

	// Track results for metrics
	var passed, failed, skipped int

	// Execute tests in order
	for _, test := range executionOrder {
		if test.Skip {
			skipped++
			continue
		}
		if err := s.runTest(ctx, test); err != nil {
			failed++
			fmt.Printf("Test %s failed: %v\n", test.ID, err)
		} else {
			passed++
		}
	}

	// Run global teardown
	for i, teardown := range s.GlobalTeardown {
		scenarioCtx := scenarios.NewContext()
		if err := teardown(scenarioCtx); err != nil {
			fmt.Printf("Warning: global teardown step %d failed: %v\n", i+1, err)
		}
	}

	// Run after suite hooks
	for _, hook := range s.Hooks {
		if err := hook.AfterSuite(ctx, s); err != nil {
			return fmt.Errorf("after suite hook failed: %w", err)
		}
	}

	// Finalize suite chronicle
	if s.suiteChronicle != nil {
		s.suiteChronicle.Finalize()
	}

	// Emit suite metrics
	if s.MetricsCollector != nil {
		s.MetricsCollector.Collect(ctx, &metrics.SuiteCompleted{
			SuiteID:  s.ID,
			Duration: time.Since(start),
			Passed:   passed,
			Failed:   failed,
			Skipped:  skipped,
		})
	}

	return nil
}

// runTest executes a single test with retries
func (s *Suite) runTest(ctx context.Context, test *Test) error {
	// Skip if marked
	if test.Skip {
		fmt.Printf("Skipping test %s: %s\n", test.ID, test.SkipReason)
		return nil
	}

	// Create test context
	testCtx := context.Background()
	if test.Timeout > 0 {
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(testCtx, test.Timeout)
		defer cancel()
	}

	// Run before test hooks
	for _, hook := range s.Hooks {
		if err := hook.BeforeTest(ctx, s, test); err != nil {
			return fmt.Errorf("before test hook failed: %w", err)
		}
	}

	// Initialize test result
	result := &TestResult{
		Test:      test,
		StartTime: time.Now(),
		State:     make(map[string]interface{}),
	}

	// Get retry policy
	retryPolicy := test.RetryPolicy
	if retryPolicy == nil {
		// Default policy: no retries
		retryPolicy = &RetryPolicy{
			MaxAttempts:     1,
			BackoffStrategy: &FixedBackoff{Delay: 0},
			RetryCondition:  func(error) bool { return false },
		}
	}

	// Execute test with retries
	var lastErr error
	var testChronicle *chronicle.Chronicle
	for attempt := 1; attempt <= retryPolicy.MaxAttempts; attempt++ {
		result.Attempt = attempt

		// Create chronicle for this test if enabled
		if s.ChronicleEnabled {
			testChronicle = chronicle.New(string(test.ID), test.Description)
		}

		// Try running the test
		err := s.executeTest(testCtx, test, result, testChronicle)
		if err == nil {
			// Test succeeded
			result.Success = true
			result.Error = nil
			break
		}

		// Test failed
		lastErr = err
		result.Success = false
		result.Error = err

		// Call failure hooks
		for _, hook := range s.Hooks {
			hook.OnTestFailure(ctx, s, test, err)
		}

		// Check if we should retry
		if attempt < retryPolicy.MaxAttempts && retryPolicy.RetryCondition(err) {
			// Wait before retrying
			delay := retryPolicy.BackoffStrategy.NextDelay(attempt)
			time.Sleep(delay)
			continue
		}

		// No more retries or condition not met
		break
	}

	// Finalize result
	result.Duration = time.Since(result.StartTime)

	// Finalize test chronicle and add to suite
	if testChronicle != nil {
		testChronicle.Finalize()
		if s.suiteChronicle != nil {
			s.suiteChronicle.Add(testChronicle.Root())
		}
	}

	// Emit test metrics
	if s.MetricsCollector != nil {
		status := metrics.StatusPassed
		if lastErr != nil {
			status = metrics.StatusFailed
		}
		s.MetricsCollector.Collect(ctx, &metrics.TestCompleted{
			SuiteID:  s.ID,
			TestID:   test.ID,
			Duration: result.Duration,
			Status:   status,
			Retries:  result.Attempt - 1,
			Error:    errString(lastErr),
		})
	}

	// Run after test hooks
	for _, hook := range s.Hooks {
		if err := hook.AfterTest(ctx, s, test, result); err != nil {
			fmt.Printf("Warning: after test hook failed: %v\n", err)
		}
	}

	return lastErr
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// executeTest runs a single test execution attempt
func (s *Suite) executeTest(ctx context.Context, test *Test, result *TestResult, testChronicle *chronicle.Chronicle) error {
	// Create scenario context
	scenarioCtx := scenarios.NewContext()

	// Set up chronicle and metrics on the context
	if testChronicle != nil {
		scenarioCtx.SetChronicle(testChronicle)
	}
	if s.MetricsCollector != nil {
		scenarioCtx.SetMetrics(metrics.NewEmitter(s.MetricsCollector))
	}

	// Set up access to suite state based on test's access level
	switch test.StateAccess {
	case ReadOnly:
		// Read-only access to suite state
		s.mutex.RLock()
		// Copy suite state values to scenario context
		for key, value := range s.SharedState.AsMap() {
			scenarioCtx.Set(key, value)
		}
		s.mutex.RUnlock()

	case ReadWrite:
		// Read-write access to suite state
		// We'll update suite state after test completion
		s.mutex.RLock()
		// Copy suite state values to scenario context
		for key, value := range s.SharedState.AsMap() {
			scenarioCtx.Set(key, value)
		}
		s.mutex.RUnlock()

	case Isolated:
		// Isolated state, no read or write to suite state
		// Start with empty state
	}

	// Set test parameters
	for key, value := range test.Parameters {
		scenarioCtx.SetParameter(key, value)
	}

	// Execute each component in sequence
	for _, componentID := range test.Components {
		component, exists := s.GetComponent(componentID)
		if !exists {
			return fmt.Errorf("component %s not found", componentID)
		}

		// Record component in chronicle
		done := scenarioCtx.Chronicle().Component(string(componentID), "")
		err := component(scenarioCtx)
		done(err)

		if err != nil {
			return fmt.Errorf("component %s failed: %w", componentID, err)
		}
	}

	// If test has ReadWrite access, update suite state
	if test.StateAccess == ReadWrite {
		s.mutex.Lock()
		// Copy scenario context state to suite state
		for key, value := range scenarioCtx.Store().AsMap() {
			s.SharedState.Set(key, value)
		}
		s.mutex.Unlock()
	}

	// Store test-specific state in result
	result.State = scenarioCtx.Store().AsMap()

	return nil
}

// calculateExecutionOrder determines test execution order based on dependencies
func (s *Suite) calculateExecutionOrder() ([]*Test, error) {
	// We'll implement a topological sort to handle dependencies

	// Track visited and temp marks for cycle detection
	visited := make(map[core.TestID]bool)
	tempMarks := make(map[core.TestID]bool)

	// Result order
	var order []*Test

	// Recursive helper function for depth-first search
	var visit func(core.TestID) error
	visit = func(id core.TestID) error {
		// Check for cycles
		if tempMarks[id] {
			return fmt.Errorf("cycle detected in test dependencies involving test %s", id)
		}

		// Skip if already visited
		if visited[id] {
			return nil
		}

		// Mark as temporary
		tempMarks[id] = true

		// Visit dependencies
		test, exists := s.testRegistry[id]
		if !exists {
			return fmt.Errorf("test %s not found", id)
		}

		for _, depID := range test.Dependencies {
			if err := visit(depID); err != nil {
				return err
			}
		}

		// Mark as permanently visited
		visited[id] = true
		tempMarks[id] = false

		// Add to order
		order = append(order, test)

		return nil
	}

	// Visit all tests
	for _, test := range s.Tests {
		if !visited[test.ID] {
			if err := visit(test.ID); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

// NewTest creates a new test
func NewTest(id core.TestID) *Test {
	return &Test{
		ID:          id,
		StateAccess: ReadOnly, // Default to read-only
		Parameters:  make(map[string]interface{}),
	}
}

// WithDescription adds a description to the test
func (t *Test) WithDescription(description string) *Test {
	t.Description = description
	return t
}

// WithComponents adds components to the test
func (t *Test) WithComponents(components ...core.ComponentID) *Test {
	t.Components = append(t.Components, components...)
	return t
}

// WithDependencies adds dependencies to the test
func (t *Test) WithDependencies(dependencies ...core.TestID) *Test {
	t.Dependencies = append(t.Dependencies, dependencies...)
	return t
}

// WithRequiredServices adds required services to the test
func (t *Test) WithRequiredServices(services ...core.ServiceID) *Test {
	t.RequiredServices = append(t.RequiredServices, services...)
	return t
}

// WithStateAccess sets the state access level
func (t *Test) WithStateAccess(access StateAccessLevel) *Test {
	t.StateAccess = access
	return t
}

// WithTimeout sets the test timeout
func (t *Test) WithTimeout(timeout time.Duration) *Test {
	t.Timeout = timeout
	return t
}

// WithRetryPolicy sets the retry policy
func (t *Test) WithRetryPolicy(policy *RetryPolicy) *Test {
	t.RetryPolicy = policy
	return t
}

// WithTags adds tags to the test
func (t *Test) WithTags(tags ...core.TagID) *Test {
	t.Tags = append(t.Tags, tags...)
	return t
}

// WithParallel sets whether this test can run in parallel
func (t *Test) WithParallel(parallel bool) *Test {
	t.Parallel = parallel
	return t
}

// WithSkip marks this test to be skipped
func (t *Test) WithSkip(reason string) *Test {
	t.Skip = true
	t.SkipReason = reason
	return t
}

// WithParameter adds a parameter to the test
func (t *Test) WithParameter(key string, value interface{}) *Test {
	t.Parameters[key] = value
	return t
}
