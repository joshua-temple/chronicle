package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	chronicleCtx "github.com/joshua-temple/chronicle/pkg/context"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/middleware"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

func TestRunnerNew(t *testing.T) {
	runner := NewRunner(t)

	if runner.t != t {
		t.Error("expected testing.T to be set")
	}
	if !runner.subtests {
		t.Error("expected subtests to be true by default")
	}
	if runner.stopOnFail {
		t.Error("expected stopOnFail to be false by default")
	}
	if runner.parallel {
		t.Error("expected parallel to be false by default")
	}
	if runner.executor == nil {
		t.Error("expected executor to be created")
	}
}

func TestRunnerWithOptions(t *testing.T) {
	exec := NewExecutor()
	runner := NewRunner(t,
		WithRunnerExecutor(exec),
		WithSubtests(false),
		WithStopOnFail(true),
		WithRunnerParallel(true),
		WithSetupTimeout(1*time.Minute),
	)

	if runner.executor != exec {
		t.Error("expected custom executor to be set")
	}
	if runner.subtests {
		t.Error("expected subtests to be false")
	}
	if !runner.stopOnFail {
		t.Error("expected stopOnFail to be true")
	}
	if !runner.parallel {
		t.Error("expected parallel to be true")
	}
	if runner.setupTimeout != 1*time.Minute {
		t.Errorf("expected setupTimeout 1m, got %v", runner.setupTimeout)
	}
}

func TestRunnerRegisterComponent(t *testing.T) {
	runner := NewRunner(t)
	comp := core.NewComponent("test", core.ComponentSetup)

	result := runner.RegisterComponent(comp)

	if result != runner {
		t.Error("expected fluent interface to return runner")
	}
	if _, ok := runner.executor.GetComponent("test"); !ok {
		t.Error("expected component to be registered")
	}
}

func TestRunnerAddScenario(t *testing.T) {
	runner := NewRunner(t)
	s := scenario.NewScenario("test")

	result := runner.AddScenario(s)

	if result != runner {
		t.Error("expected fluent interface to return runner")
	}
	if len(runner.scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(runner.scenarios))
	}
}

func TestRunnerRunSubtests(t *testing.T) {
	// Use a sub-test to isolate
	t.Run("subtests", func(t *testing.T) {
		runner := NewRunner(t)

		executed := 0
		comp := core.NewComponent("step", core.ComponentSetup).
			WithFunc(func(ctx chronicleCtx.Context) error {
				executed++
				return nil
			})
		runner.RegisterComponent(comp)

		s1 := scenario.NewScenario("scenario1")
		s1.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

		s2 := scenario.NewScenario("scenario2")
		s2.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

		runner.AddScenarios([]*scenario.Scenario{s1, s2})
		runner.Run(context.Background())

		if executed != 2 {
			t.Errorf("expected 2 executions, got %d", executed)
		}
	})
}

func TestRunnerRunSequentially(t *testing.T) {
	t.Run("sequential", func(t *testing.T) {
		runner := NewRunner(t, WithSubtests(false))

		executed := 0
		comp := core.NewComponent("step", core.ComponentSetup).
			WithFunc(func(ctx chronicleCtx.Context) error {
				executed++
				return nil
			})
		runner.RegisterComponent(comp)

		s := scenario.NewScenario("scenario")
		s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

		runner.AddScenario(s)
		runner.Run(context.Background())

		if executed != 1 {
			t.Errorf("expected 1 execution, got %d", executed)
		}
	})
}

func TestTHelper(t *testing.T) {
	comp := core.NewComponent("step", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			return nil
		})

	s := scenario.NewScenario("test")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

	result := T(t, s, []*core.Component{comp})

	if result.State != StateCompleted {
		t.Errorf("expected Completed, got %s", result.State)
	}
}

func TestTableTests(t *testing.T) {
	passComp := core.NewComponent("pass", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error { return nil })
	failComp := core.NewComponent("fail", core.ComponentTask).
		WithFunc(func(ctx chronicleCtx.Context) error { return errors.New("fail") })

	passScenario := scenario.NewScenario("pass")
	passScenario.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "pass"))

	failScenario := scenario.NewScenario("fail")
	failScenario.AddFlow(scenario.NewFlowItem(core.ComponentTask, "fail"))

	tests := []TableTest{
		{
			Name:       "passing scenario",
			Scenario:   passScenario,
			Components: []*core.Component{passComp},
			WantState:  StateCompleted,
		},
		{
			Name:       "failing scenario",
			Scenario:   failScenario,
			Components: []*core.Component{failComp},
			WantState:  StateFailed,
			WantError:  true,
		},
	}

	RunTableTests(t, tests)
}

func TestAssertResult(t *testing.T) {
	t.Run("succeeded", func(t *testing.T) {
		result := &ScenarioResult{State: StateCompleted}
		assert := NewAssertResult(t, result)
		assert.Succeeded()
	})

	t.Run("failed", func(t *testing.T) {
		result := &ScenarioResult{State: StateFailed}
		assert := NewAssertResult(t, result)
		assert.Failed()
	})

	t.Run("skipped", func(t *testing.T) {
		result := &ScenarioResult{State: StateSkipped}
		assert := NewAssertResult(t, result)
		assert.Skipped()
	})

	t.Run("with error", func(t *testing.T) {
		result := &ScenarioResult{
			State: StateFailed,
			Error: errors.New("something went wrong"),
		}
		assert := NewAssertResult(t, result)
		assert.WithError("wrong")
	})

	t.Run("duration", func(t *testing.T) {
		result := &ScenarioResult{Duration: 10 * time.Millisecond}
		assert := NewAssertResult(t, result)
		assert.DurationLessThan(1 * time.Second)
	})

	t.Run("flow count", func(t *testing.T) {
		result := &ScenarioResult{
			FlowResults: []FlowItemResult{{}, {}},
		}
		assert := NewAssertResult(t, result)
		assert.FlowItemCount(2)
	})
}

func TestTestScenarioHelper(t *testing.T) {
	s := TestScenario("my-test",
		WithStep(core.ComponentSetup, "setup"),
		WithStep(core.ComponentTask, "action"),
		WithTeardownStep("cleanup"),
	)

	if s.Name != "my-test" {
		t.Errorf("expected name 'my-test', got %s", s.Name)
	}
	if len(s.Flow) != 2 {
		t.Errorf("expected 2 flow items, got %d", len(s.Flow))
	}
	if len(s.TeardownFlow) != 1 {
		t.Errorf("expected 1 teardown item, got %d", len(s.TeardownFlow))
	}
}

func TestTestComponent(t *testing.T) {
	executed := false
	comp := TestComponent("test", core.ComponentSetup, func() error {
		executed = true
		return nil
	})

	if comp.Name != "test" {
		t.Errorf("expected name 'test', got %s", comp.Name)
	}
	if comp.Type != core.ComponentSetup {
		t.Errorf("expected type setup, got %s", comp.Type)
	}

	// Execute the function
	fn := comp.Func.(func(core.Context) error)
	ctx := chronicleCtx.New(context.Background())
	err := fn(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !executed {
		t.Error("expected function to be executed")
	}
}

func TestSummary(t *testing.T) {
	results := []*ScenarioResult{
		{State: StateCompleted, Duration: 10 * time.Millisecond},
		{State: StateCompleted, Duration: 20 * time.Millisecond},
		{State: StateFailed, Duration: 5 * time.Millisecond},
		{State: StateSkipped},
	}

	summary := Summary(results)

	if summary == "" {
		t.Error("expected non-empty summary")
	}
	// Should contain counts
	if !containsString(summary, "2 passed") {
		t.Errorf("expected '2 passed' in summary: %s", summary)
	}
	if !containsString(summary, "1 failed") {
		t.Errorf("expected '1 failed' in summary: %s", summary)
	}
	if !containsString(summary, "1 skipped") {
		t.Errorf("expected '1 skipped' in summary: %s", summary)
	}
}

func TestQuickTest(t *testing.T) {
	executed := false
	QuickTest(t, "quick", func(ctx context.Context) error {
		executed = true
		return nil
	})

	if !executed {
		t.Error("expected quick test to execute")
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "xyz", false},
		{"hello", "", true},
		{"", "", true},
		{"", "a", false},
	}

	for _, tc := range tests {
		got := containsString(tc.s, tc.substr)
		if got != tc.want {
			t.Errorf("containsString(%q, %q) = %v, want %v", tc.s, tc.substr, got, tc.want)
		}
	}
}

func TestSuiteRunner(t *testing.T) {
	setupRan := false
	teardownRan := false

	comp := core.NewComponent("step", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error { return nil })

	s := scenario.NewScenario("test")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

	suite := &Suite{
		Name:        "test-suite",
		Description: "A test suite",
		Setup: func(ctx context.Context) error {
			setupRan = true
			return nil
		},
		Teardown: func(ctx context.Context) error {
			teardownRan = true
			return nil
		},
		Scenarios: []*scenario.Scenario{s},
	}

	exec := NewExecutor()
	exec.RegisterComponent(comp)

	RunSuite(t, suite, WithRunnerExecutor(exec))

	if !setupRan {
		t.Error("expected setup to run")
	}
	if !teardownRan {
		t.Error("expected teardown to run")
	}
}

func TestWithMiddlewareChain(t *testing.T) {
	var order []string

	mw1 := func(next middleware.Runner) middleware.Runner {
		return func(ctx chronicleCtx.Context) error {
			order = append(order, "mw1-before")
			err := next(ctx)
			order = append(order, "mw1-after")
			return err
		}
	}

	mw2 := func(next middleware.Runner) middleware.Runner {
		return func(ctx chronicleCtx.Context) error {
			order = append(order, "mw2-before")
			err := next(ctx)
			order = append(order, "mw2-after")
			return err
		}
	}

	exec := NewExecutor(WithMiddlewareChain(mw1, mw2))

	comp := core.NewComponent("step", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			order = append(order, "step")
			return nil
		})
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("test")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

	exec.Execute(context.Background(), s)

	expected := []string{"mw1-before", "mw2-before", "step", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Errorf("expected %d items, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if i < len(order) && order[i] != v {
			t.Errorf("position %d: expected %s, got %s", i, v, order[i])
		}
	}
}
