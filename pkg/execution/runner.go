package execution

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/joshua-temple/chronicle/pkg/middleware"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

// Runner integrates Chronicle with Go's testing framework.
type Runner struct {
	executor     *Executor
	infraManager *infrastructure.Manager
	config       *config.Config
	scenarios    []*scenario.Scenario

	// Test integration
	t            *testing.T
	subtests     bool
	stopOnFail   bool
	parallel     bool
	setupTimeout time.Duration
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// NewRunner creates a new Runner with the given testing.T and options.
func NewRunner(t *testing.T, opts ...RunnerOption) *Runner {
	r := &Runner{
		t:            t,
		subtests:     true,
		stopOnFail:   false,
		parallel:     false,
		setupTimeout: 5 * time.Minute,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Create default executor if not provided
	if r.executor == nil {
		r.executor = NewExecutor()
	}

	return r
}

// WithExecutor sets a custom executor.
func WithRunnerExecutor(exec *Executor) RunnerOption {
	return func(r *Runner) {
		r.executor = exec
	}
}

// WithConfig sets the configuration.
func WithConfig(cfg *config.Config) RunnerOption {
	return func(r *Runner) {
		r.config = cfg
	}
}

// WithInfrastructureManager sets the infrastructure manager.
func WithInfrastructureManager(mgr *infrastructure.Manager) RunnerOption {
	return func(r *Runner) {
		r.infraManager = mgr
	}
}

// WithSubtests enables/disables subtests for each scenario.
func WithSubtests(enabled bool) RunnerOption {
	return func(r *Runner) {
		r.subtests = enabled
	}
}

// WithStopOnFail enables/disables stopping on first failure.
func WithStopOnFail(enabled bool) RunnerOption {
	return func(r *Runner) {
		r.stopOnFail = enabled
	}
}

// WithRunnerParallel enables/disables parallel scenario execution.
func WithRunnerParallel(enabled bool) RunnerOption {
	return func(r *Runner) {
		r.parallel = enabled
	}
}

// WithSetupTimeout sets the timeout for infrastructure setup.
func WithSetupTimeout(timeout time.Duration) RunnerOption {
	return func(r *Runner) {
		r.setupTimeout = timeout
	}
}

// RegisterComponent registers a component with the runner's executor.
func (r *Runner) RegisterComponent(c *core.Component) *Runner {
	r.executor.RegisterComponent(c)
	return r
}

// RegisterComponents registers multiple components.
func (r *Runner) RegisterComponents(components []*core.Component) *Runner {
	r.executor.RegisterComponents(components)
	return r
}

// AddScenario adds a scenario to run.
func (r *Runner) AddScenario(s *scenario.Scenario) *Runner {
	r.scenarios = append(r.scenarios, s)
	return r
}

// AddScenarios adds multiple scenarios to run.
func (r *Runner) AddScenarios(scenarios []*scenario.Scenario) *Runner {
	r.scenarios = append(r.scenarios, scenarios...)
	return r
}

// Run executes all registered scenarios.
func (r *Runner) Run(ctx context.Context) {
	r.t.Helper()

	// Set up infrastructure if manager is provided
	if r.infraManager != nil {
		r.setupInfrastructure(ctx)
		defer r.teardownInfrastructure()

		// Set infrastructure on executor
		r.executor.infraManager = r.infraManager
	}

	// Run scenarios
	if r.subtests {
		r.runAsSubtests(ctx)
	} else {
		r.runSequentially(ctx)
	}
}

// setupInfrastructure starts all configured infrastructure.
func (r *Runner) setupInfrastructure(ctx context.Context) {
	r.t.Helper()

	setupCtx, cancel := context.WithTimeout(ctx, r.setupTimeout)
	defer cancel()

	if err := r.infraManager.Start(setupCtx); err != nil {
		r.t.Fatalf("Failed to start infrastructure: %v", err)
	}

	// Wait for health checks
	if !r.infraManager.AllHealthy(setupCtx) {
		r.t.Fatal("Infrastructure health check failed")
	}
}

// teardownInfrastructure stops all infrastructure.
func (r *Runner) teardownInfrastructure() {
	r.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := r.infraManager.Stop(ctx); err != nil {
		r.t.Logf("Warning: failed to stop infrastructure: %v", err)
	}
}

// runAsSubtests runs each scenario as a subtest.
func (r *Runner) runAsSubtests(ctx context.Context) {
	r.t.Helper()

	for _, s := range r.scenarios {
		s := s // capture for closure
		r.t.Run(s.Name, func(t *testing.T) {
			if r.parallel {
				t.Parallel()
			}

			result := r.executor.Execute(ctx, s)
			r.reportResult(t, result)
		})

		if r.stopOnFail && r.t.Failed() {
			return
		}
	}
}

// runSequentially runs scenarios without subtests.
func (r *Runner) runSequentially(ctx context.Context) {
	r.t.Helper()

	for _, s := range r.scenarios {
		result := r.executor.Execute(ctx, s)
		r.reportResultToT(r.t, s.Name, result)

		if r.stopOnFail && r.t.Failed() {
			return
		}
	}
}

// reportResult reports a scenario result to a testing.T.
func (r *Runner) reportResult(t *testing.T, result *ScenarioResult) {
	t.Helper()

	switch result.State {
	case StateCompleted:
		// Success - nothing to report
		t.Logf("Scenario completed in %v", result.Duration)

	case StateFailed:
		t.Errorf("Scenario failed: %v", result.Error)
		r.reportFlowResults(t, result)

	case StateSkipped:
		t.Skipf("Scenario skipped: %s", result.SkipReason)

	case StateCancelled:
		t.Errorf("Scenario cancelled: %v", result.Error)

	default:
		t.Errorf("Unexpected scenario state: %s", result.State)
	}
}

// reportResultToT reports a scenario result without using subtests.
func (r *Runner) reportResultToT(t *testing.T, name string, result *ScenarioResult) {
	t.Helper()

	switch result.State {
	case StateCompleted:
		t.Logf("[PASS] %s completed in %v", name, result.Duration)

	case StateFailed:
		t.Errorf("[FAIL] %s failed: %v", name, result.Error)
		r.reportFlowResults(t, result)

	case StateSkipped:
		t.Logf("[SKIP] %s: %s", name, result.SkipReason)

	case StateCancelled:
		t.Errorf("[CANCEL] %s cancelled: %v", name, result.Error)

	default:
		t.Errorf("[ERROR] %s unexpected state: %s", name, result.State)
	}
}

// reportFlowResults reports individual flow item results on failure.
func (r *Runner) reportFlowResults(t *testing.T, result *ScenarioResult) {
	t.Helper()

	t.Logf("Flow item results:")
	for _, fr := range result.FlowResults {
		switch fr.State {
		case StateCompleted:
			t.Logf("  ✓ %s (%v)", fr.Name, fr.Duration)
		case StateFailed:
			t.Logf("  ✗ %s (%v): %v", fr.Name, fr.Duration, fr.Error)
		case StateSkipped:
			t.Logf("  ○ %s (skipped)", fr.Name)
		default:
			t.Logf("  ? %s: %s", fr.Name, fr.State)
		}
	}

	if len(result.TeardownResults) > 0 {
		t.Logf("Teardown results:")
		for _, tr := range result.TeardownResults {
			switch tr.State {
			case StateCompleted:
				t.Logf("  ✓ %s (%v)", tr.Name, tr.Duration)
			case StateFailed:
				t.Logf("  ✗ %s (%v): %v", tr.Name, tr.Duration, tr.Error)
			default:
				t.Logf("  ? %s: %s", tr.Name, tr.State)
			}
		}
	}
}

// Suite helps organize scenarios into a test suite.
type Suite struct {
	Name        string
	Description string
	Setup       func(ctx context.Context) error
	Teardown    func(ctx context.Context) error
	Scenarios   []*scenario.Scenario
}

// RunSuite runs a test suite.
func RunSuite(t *testing.T, suite *Suite, opts ...RunnerOption) {
	t.Helper()

	t.Run(suite.Name, func(t *testing.T) {
		ctx := context.Background()

		// Run suite setup
		if suite.Setup != nil {
			if err := suite.Setup(ctx); err != nil {
				t.Fatalf("Suite setup failed: %v", err)
			}
		}

		// Ensure teardown runs
		if suite.Teardown != nil {
			defer func() {
				if err := suite.Teardown(ctx); err != nil {
					t.Logf("Suite teardown error: %v", err)
				}
			}()
		}

		// Create runner and add scenarios
		runner := NewRunner(t, opts...)
		runner.AddScenarios(suite.Scenarios)

		// Run all scenarios
		runner.Run(ctx)
	})
}

// T provides a simplified interface for running a single scenario.
func T(t *testing.T, s *scenario.Scenario, components []*core.Component, opts ...ExecutorOption) *ScenarioResult {
	t.Helper()

	exec := NewExecutor(opts...)
	exec.RegisterComponents(components)

	ctx := context.Background()
	result := exec.Execute(ctx, s)

	switch result.State {
	case StateCompleted:
		// Success
	case StateFailed:
		t.Errorf("Scenario %s failed: %v", s.Name, result.Error)
	case StateSkipped:
		t.Skipf("Scenario %s skipped: %s", s.Name, result.SkipReason)
	case StateCancelled:
		t.Errorf("Scenario %s cancelled", s.Name)
	}

	return result
}

// Benchmark runs a scenario for benchmarking.
func Benchmark(b *testing.B, s *scenario.Scenario, components []*core.Component, opts ...ExecutorOption) {
	exec := NewExecutor(opts...)
	exec.RegisterComponents(components)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := exec.Execute(ctx, s)
		if result.State == StateFailed {
			b.Fatalf("Scenario failed: %v", result.Error)
		}
	}
}

// WithMiddlewareChain creates an executor option that applies a middleware chain.
func WithMiddlewareChain(middlewares ...middleware.Middleware) ExecutorOption {
	return func(e *Executor) {
		if len(middlewares) > 0 {
			e.middlewareChain = middleware.Chain(middlewares...)
		}
	}
}

// QuickTest is a helper for simple, single-component tests.
func QuickTest(t *testing.T, name string, fn func(ctx context.Context) error) {
	t.Helper()

	ctx := context.Background()
	if err := fn(ctx); err != nil {
		t.Errorf("%s failed: %v", name, err)
	}
}

// TableTest runs table-driven tests using Chronicle scenarios.
type TableTest struct {
	Name       string
	Scenario   *scenario.Scenario
	Components []*core.Component
	WantError  bool
	WantState  ExecutionState
}

// RunTableTests runs a slice of table tests.
func RunTableTests(t *testing.T, tests []TableTest, opts ...ExecutorOption) {
	t.Helper()

	for _, tc := range tests {
		tc := tc // capture
		t.Run(tc.Name, func(t *testing.T) {
			exec := NewExecutor(opts...)
			exec.RegisterComponents(tc.Components)

			result := exec.Execute(context.Background(), tc.Scenario)

			if tc.WantState != 0 && result.State != tc.WantState {
				t.Errorf("expected state %s, got %s", tc.WantState, result.State)
			}

			if tc.WantError && result.Error == nil {
				t.Error("expected error but got none")
			}
			if !tc.WantError && result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
			}
		})
	}
}

// AssertResult is a helper for making assertions on scenario results.
type AssertResult struct {
	t      *testing.T
	result *ScenarioResult
}

// NewAssertResult creates a new result asserter.
func NewAssertResult(t *testing.T, result *ScenarioResult) *AssertResult {
	return &AssertResult{t: t, result: result}
}

// Succeeded asserts the scenario succeeded.
func (a *AssertResult) Succeeded() *AssertResult {
	a.t.Helper()
	if !a.result.IsSuccess() {
		a.t.Errorf("expected success, got %s: %v", a.result.State, a.result.Error)
	}
	return a
}

// Failed asserts the scenario failed.
func (a *AssertResult) Failed() *AssertResult {
	a.t.Helper()
	if a.result.State != StateFailed {
		a.t.Errorf("expected failure, got %s", a.result.State)
	}
	return a
}

// Skipped asserts the scenario was skipped.
func (a *AssertResult) Skipped() *AssertResult {
	a.t.Helper()
	if a.result.State != StateSkipped {
		a.t.Errorf("expected skipped, got %s", a.result.State)
	}
	return a
}

// WithError asserts the error message contains the given substring.
func (a *AssertResult) WithError(contains string) *AssertResult {
	a.t.Helper()
	if a.result.Error == nil {
		a.t.Error("expected error, got nil")
		return a
	}
	if !containsString(a.result.Error.Error(), contains) {
		a.t.Errorf("error %q does not contain %q", a.result.Error.Error(), contains)
	}
	return a
}

// DurationLessThan asserts the duration is less than the given value.
func (a *AssertResult) DurationLessThan(d time.Duration) *AssertResult {
	a.t.Helper()
	if a.result.Duration >= d {
		a.t.Errorf("duration %v >= %v", a.result.Duration, d)
	}
	return a
}

// FlowItemCount asserts the number of flow items.
func (a *AssertResult) FlowItemCount(count int) *AssertResult {
	a.t.Helper()
	if len(a.result.FlowResults) != count {
		a.t.Errorf("expected %d flow items, got %d", count, len(a.result.FlowResults))
	}
	return a
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper function for creating test scenarios quickly.
func TestScenario(name string, steps ...func(s *scenario.Scenario)) *scenario.Scenario {
	s := scenario.NewScenario(name)
	for _, step := range steps {
		step(s)
	}
	return s
}

// WithStep adds a flow item to a scenario.
func WithStep(componentType core.ComponentType, name string) func(s *scenario.Scenario) {
	return func(s *scenario.Scenario) {
		s.AddFlow(scenario.NewFlowItem(componentType, name))
	}
}

// WithTeardownStep adds a teardown item to a scenario.
func WithTeardownStep(name string) func(s *scenario.Scenario) {
	return func(s *scenario.Scenario) {
		s.AddTeardown(scenario.NewFlowItem(core.ComponentTeardown, name))
	}
}

// TestComponent creates a component with a simple test function.
func TestComponent(name string, componentType core.ComponentType, fn func() error) *core.Component {
	return core.NewComponent(name, componentType).
		WithFunc(func(ctx core.Context) error {
			return fn()
		})
}

// Summary generates a summary of execution results.
func Summary(results []*ScenarioResult) string {
	var passed, failed, skipped int
	var totalDuration time.Duration

	for _, r := range results {
		totalDuration += r.Duration
		switch r.State {
		case StateCompleted:
			passed++
		case StateFailed:
			failed++
		case StateSkipped:
			skipped++
		}
	}

	return fmt.Sprintf(
		"Results: %d passed, %d failed, %d skipped (total: %v)",
		passed, failed, skipped, totalDuration,
	)
}
