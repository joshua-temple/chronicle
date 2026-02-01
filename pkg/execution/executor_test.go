package execution

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	chronicleCtx "github.com/joshua-temple/chronicle/pkg/context"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

func TestExecutorNew(t *testing.T) {
	exec := NewExecutor()

	if exec.defaultTimeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", exec.defaultTimeout)
	}
	if exec.parallelism != 1 {
		t.Errorf("expected parallelism 1, got %d", exec.parallelism)
	}
	if exec.failFast {
		t.Error("expected failFast to be false by default")
	}
}

func TestExecutorWithOptions(t *testing.T) {
	exec := NewExecutor(
		WithDefaultTimeout(1*time.Minute),
		WithParallelism(4),
		WithFailFast(true),
	)

	if exec.defaultTimeout != 1*time.Minute {
		t.Errorf("expected timeout 1m, got %v", exec.defaultTimeout)
	}
	if exec.parallelism != 4 {
		t.Errorf("expected parallelism 4, got %d", exec.parallelism)
	}
	if !exec.failFast {
		t.Error("expected failFast to be true")
	}
}

func TestExecutorWithParallelismMinimum(t *testing.T) {
	exec := NewExecutor(WithParallelism(0))
	if exec.parallelism != 1 {
		t.Errorf("expected parallelism to be at least 1, got %d", exec.parallelism)
	}

	exec = NewExecutor(WithParallelism(-5))
	if exec.parallelism != 1 {
		t.Errorf("expected parallelism to be at least 1, got %d", exec.parallelism)
	}
}

func TestExecutorRegisterComponent(t *testing.T) {
	exec := NewExecutor()

	comp := core.NewComponent("test-component", core.ComponentSetup)
	exec.RegisterComponent(comp)

	retrieved, ok := exec.GetComponent("test-component")
	if !ok {
		t.Fatal("expected to find registered component")
	}
	if retrieved.Name != "test-component" {
		t.Errorf("expected name 'test-component', got %s", retrieved.Name)
	}

	_, ok = exec.GetComponent("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent component")
	}
}

func TestExecutorRegisterComponents(t *testing.T) {
	exec := NewExecutor()

	components := []*core.Component{
		core.NewComponent("comp1", core.ComponentSetup),
		core.NewComponent("comp2", core.ComponentTask),
		core.NewComponent("comp3", core.ComponentTeardown),
	}

	exec.RegisterComponents(components)

	for _, c := range components {
		if _, ok := exec.GetComponent(c.Name); !ok {
			t.Errorf("expected to find component %s", c.Name)
		}
	}
}

func TestExecuteSimpleScenario(t *testing.T) {
	exec := NewExecutor()

	// Register a simple component
	executed := false
	comp := core.NewComponent("setup-user", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			executed = true
			ctx.Set("user", "test-user")
			return nil
		})
	exec.RegisterComponent(comp)

	// Create a scenario
	s := scenario.NewScenario("test-scenario")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "setup-user"))

	// Execute
	ctx := context.Background()
	result := exec.Execute(ctx, s)

	if !executed {
		t.Error("expected component to be executed")
	}
	if result.State != StateCompleted {
		t.Errorf("expected state Completed, got %s", result.State)
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
	if len(result.FlowResults) != 1 {
		t.Errorf("expected 1 flow result, got %d", len(result.FlowResults))
	}
	if result.FlowResults[0].State != StateCompleted {
		t.Errorf("expected flow item state Completed, got %s", result.FlowResults[0].State)
	}
}

func TestExecuteWithMultipleSteps(t *testing.T) {
	exec := NewExecutor()

	var order []string

	comp1 := core.NewComponent("step1", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			order = append(order, "step1")
			return nil
		})
	comp2 := core.NewComponent("step2", core.ComponentTask).
		WithFunc(func(ctx chronicleCtx.Context) error {
			order = append(order, "step2")
			return nil
		})
	comp3 := core.NewComponent("step3", core.ComponentValidation).
		WithFunc(func(ctx chronicleCtx.Context) error {
			order = append(order, "step3")
			return nil
		})

	exec.RegisterComponents([]*core.Component{comp1, comp2, comp3})

	s := scenario.NewScenario("multi-step")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step1"))
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "step2"))
	s.AddFlow(scenario.NewFlowItem(core.ComponentValidation, "step3"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateCompleted {
		t.Errorf("expected Completed, got %s: %v", result.State, result.Error)
	}
	if len(order) != 3 {
		t.Errorf("expected 3 executions, got %d", len(order))
	}
	if order[0] != "step1" || order[1] != "step2" || order[2] != "step3" {
		t.Errorf("expected sequential order, got %v", order)
	}
}

func TestExecuteWithFailure(t *testing.T) {
	exec := NewExecutor()

	expectedErr := errors.New("component failed")
	comp := core.NewComponent("failing-step", core.ComponentTask).
		WithFunc(func(ctx chronicleCtx.Context) error {
			return expectedErr
		})
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("failing")
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "failing-step"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateFailed {
		t.Errorf("expected Failed, got %s", result.State)
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestExecuteWithTeardown(t *testing.T) {
	exec := NewExecutor()

	var order []string

	setup := core.NewComponent("setup", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			order = append(order, "setup")
			return nil
		})
	teardown := core.NewComponent("teardown", core.ComponentTeardown).
		WithFunc(func(ctx chronicleCtx.Context) error {
			order = append(order, "teardown")
			return nil
		})

	exec.RegisterComponents([]*core.Component{setup, teardown})

	s := scenario.NewScenario("with-teardown")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "setup"))
	s.AddTeardown(scenario.NewFlowItem(core.ComponentTeardown, "teardown"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateCompleted {
		t.Errorf("expected Completed, got %s: %v", result.State, result.Error)
	}
	if len(order) != 2 || order[0] != "setup" || order[1] != "teardown" {
		t.Errorf("expected [setup, teardown], got %v", order)
	}
	if len(result.TeardownResults) != 1 {
		t.Errorf("expected 1 teardown result, got %d", len(result.TeardownResults))
	}
}

func TestExecuteTeardownRunsOnFailure(t *testing.T) {
	exec := NewExecutor()

	teardownExecuted := false

	setup := core.NewComponent("setup", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			return errors.New("setup failed")
		})
	teardown := core.NewComponent("teardown", core.ComponentTeardown).
		WithFunc(func(ctx chronicleCtx.Context) error {
			teardownExecuted = true
			return nil
		})

	exec.RegisterComponents([]*core.Component{setup, teardown})

	s := scenario.NewScenario("failing-with-teardown")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "setup"))
	s.AddTeardown(scenario.NewFlowItem(core.ComponentTeardown, "teardown"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateFailed {
		t.Errorf("expected Failed, got %s", result.State)
	}
	if !teardownExecuted {
		t.Error("expected teardown to execute even on failure")
	}
}

func TestExecuteFailFast(t *testing.T) {
	exec := NewExecutor(WithFailFast(true))

	var executedSteps []string

	step1 := core.NewComponent("step1", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			executedSteps = append(executedSteps, "step1")
			return nil
		})
	step2 := core.NewComponent("step2", core.ComponentTask).
		WithFunc(func(ctx chronicleCtx.Context) error {
			executedSteps = append(executedSteps, "step2")
			return errors.New("step2 failed")
		})
	step3 := core.NewComponent("step3", core.ComponentValidation).
		WithFunc(func(ctx chronicleCtx.Context) error {
			executedSteps = append(executedSteps, "step3")
			return nil
		})

	exec.RegisterComponents([]*core.Component{step1, step2, step3})

	s := scenario.NewScenario("fail-fast")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step1"))
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "step2"))
	s.AddFlow(scenario.NewFlowItem(core.ComponentValidation, "step3"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateFailed {
		t.Errorf("expected Failed, got %s", result.State)
	}
	// step3 should not execute because fail-fast is enabled
	if len(executedSteps) != 2 {
		t.Errorf("expected 2 steps executed (step3 skipped due to fail-fast), got %v", executedSteps)
	}
}

func TestExecuteParallel(t *testing.T) {
	exec := NewExecutor(WithParallelism(3))

	var count int32

	createComponent := func(name string) *core.Component {
		return core.NewComponent(name, core.ComponentTask).
			WithFunc(func(ctx chronicleCtx.Context) error {
				atomic.AddInt32(&count, 1)
				time.Sleep(10 * time.Millisecond) // Small delay to ensure parallel execution
				return nil
			})
	}

	exec.RegisterComponents([]*core.Component{
		createComponent("p1"),
		createComponent("p2"),
		createComponent("p3"),
	})

	s := scenario.NewScenario("parallel")
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "p1").AsParallel())
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "p2").AsParallel())
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "p3").AsParallel())

	start := time.Now()
	result := exec.Execute(context.Background(), s)
	duration := time.Since(start)

	if result.State != StateCompleted {
		t.Errorf("expected Completed, got %s: %v", result.State, result.Error)
	}
	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("expected 3 executions, got %d", count)
	}
	// If truly parallel, should complete in roughly 10ms, not 30ms
	if duration > 50*time.Millisecond {
		t.Errorf("parallel execution took too long (%v), may not be parallel", duration)
	}
}

func TestExecuteAbstractScenario(t *testing.T) {
	exec := NewExecutor()

	s := scenario.NewScenario("abstract")
	s.Abstract = true

	result := exec.Execute(context.Background(), s)

	if result.State != StateSkipped {
		t.Errorf("expected Skipped, got %s", result.State)
	}
	if result.SkipReason == "" {
		t.Error("expected skip reason to be set")
	}
}

func TestExecuteEmptyScenario(t *testing.T) {
	exec := NewExecutor()

	s := scenario.NewScenario("empty")

	result := exec.Execute(context.Background(), s)

	if result.State != StateSkipped {
		t.Errorf("expected Skipped for empty scenario, got %s", result.State)
	}
}

func TestExecuteComponentNotFound(t *testing.T) {
	exec := NewExecutor()

	s := scenario.NewScenario("missing-component")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "nonexistent"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateFailed {
		t.Errorf("expected Failed, got %s", result.State)
	}
	if result.FlowResults[0].Error == nil {
		t.Error("expected error for missing component")
	}
}

func TestExecuteComponentNoFunc(t *testing.T) {
	exec := NewExecutor()

	comp := core.NewComponent("no-func", core.ComponentSetup)
	// Don't set Func
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("no-func")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "no-func"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateFailed {
		t.Errorf("expected Failed, got %s", result.State)
	}
}

func TestExecuteWithTimeout(t *testing.T) {
	exec := NewExecutor(WithDefaultTimeout(50 * time.Millisecond))

	comp := core.NewComponent("slow", core.ComponentTask).
		WithFunc(func(ctx chronicleCtx.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("timeout-test")
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "slow"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateFailed {
		t.Errorf("expected Failed due to timeout, got %s", result.State)
	}

	var timeoutErr *TimeoutError
	if result.FlowResults[0].Error != nil {
		if errors.As(result.FlowResults[0].Error, &timeoutErr) {
			if timeoutErr.Component != "slow" {
				t.Errorf("expected component 'slow', got %s", timeoutErr.Component)
			}
		} else {
			t.Errorf("expected TimeoutError, got %T", result.FlowResults[0].Error)
		}
	}
}

func TestExecuteWithCancellation(t *testing.T) {
	exec := NewExecutor()

	comp := core.NewComponent("blocking", core.ComponentTask).
		WithFunc(func(ctx chronicleCtx.Context) error {
			// Wait for context cancellation
			<-ctx.Done()
			return ctx.Err()
		})
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("cancel-test")
	s.AddFlow(scenario.NewFlowItem(core.ComponentTask, "blocking"))

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result := exec.Execute(ctx, s)

	if result.State != StateCancelled && result.State != StateFailed {
		t.Errorf("expected Cancelled or Failed, got %s", result.State)
	}
}

func TestExecuteMultiple(t *testing.T) {
	exec := NewExecutor()

	comp := core.NewComponent("step", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			return nil
		})
	exec.RegisterComponent(comp)

	s1 := scenario.NewScenario("s1")
	s1.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))
	s2 := scenario.NewScenario("s2")
	s2.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))
	s3 := scenario.NewScenario("s3")
	s3.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

	scenarios := []*scenario.Scenario{s1, s2, s3}

	results := exec.ExecuteMultiple(context.Background(), scenarios)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if r.State != StateCompleted {
			t.Errorf("scenario %d: expected Completed, got %s", i, r.State)
		}
	}
}

func TestExecuteMultipleFailFast(t *testing.T) {
	exec := NewExecutor(WithFailFast(true))

	passComp := core.NewComponent("pass", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error { return nil })
	failComp := core.NewComponent("fail", core.ComponentTask).
		WithFunc(func(ctx chronicleCtx.Context) error { return errors.New("fail") })

	exec.RegisterComponents([]*core.Component{passComp, failComp})

	s1 := scenario.NewScenario("s1")
	s1.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "pass"))
	s2 := scenario.NewScenario("s2")
	s2.AddFlow(scenario.NewFlowItem(core.ComponentTask, "fail"))
	s3 := scenario.NewScenario("s3")
	s3.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "pass"))

	scenarios := []*scenario.Scenario{s1, s2, s3}

	results := exec.ExecuteMultiple(context.Background(), scenarios)

	if results[0].State != StateCompleted {
		t.Errorf("s1: expected Completed, got %s", results[0].State)
	}
	if results[1].State != StateFailed {
		t.Errorf("s2: expected Failed, got %s", results[1].State)
	}
	if results[2].State != StateSkipped {
		t.Errorf("s3: expected Skipped due to fail-fast, got %s", results[2].State)
	}
}

func TestExecuteHooks(t *testing.T) {
	var hookOrder []string

	exec := NewExecutor(
		WithBeforeScenario(func(ctx context.Context, s *scenario.Scenario) error {
			hookOrder = append(hookOrder, "before-scenario")
			return nil
		}),
		WithAfterScenario(func(ctx context.Context, s *scenario.Scenario, result *ScenarioResult) {
			hookOrder = append(hookOrder, "after-scenario")
		}),
		WithBeforeItem(func(ctx context.Context, item *scenario.FlowItem) {
			hookOrder = append(hookOrder, "before-item")
		}),
		WithAfterItem(func(ctx context.Context, item *scenario.FlowItem, result *FlowItemResult) {
			hookOrder = append(hookOrder, "after-item")
		}),
	)

	comp := core.NewComponent("step", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			hookOrder = append(hookOrder, "step")
			return nil
		})
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("hooks-test")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

	exec.Execute(context.Background(), s)

	expected := []string{"before-scenario", "before-item", "step", "after-item", "after-scenario"}
	if len(hookOrder) != len(expected) {
		t.Errorf("expected %d hooks, got %d: %v", len(expected), len(hookOrder), hookOrder)
	}
	for i, v := range expected {
		if i < len(hookOrder) && hookOrder[i] != v {
			t.Errorf("position %d: expected %s, got %s", i, v, hookOrder[i])
		}
	}
}

func TestExecuteBeforeScenarioHookFails(t *testing.T) {
	exec := NewExecutor(
		WithBeforeScenario(func(ctx context.Context, s *scenario.Scenario) error {
			return errors.New("before hook failed")
		}),
	)

	comp := core.NewComponent("step", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error { return nil })
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("hook-fail")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "step"))

	result := exec.Execute(context.Background(), s)

	if result.State != StateFailed {
		t.Errorf("expected Failed, got %s", result.State)
	}
	if len(result.FlowResults) != 0 {
		t.Error("expected no flow results when before hook fails")
	}
}

func TestGroupFlowItems(t *testing.T) {
	exec := NewExecutor()

	items := []scenario.FlowItem{
		{Name: "s1", Parallel: false},
		{Name: "p1", Parallel: true},
		{Name: "p2", Parallel: true},
		{Name: "s2", Parallel: false},
		{Name: "p3", Parallel: true},
	}

	groups := exec.groupFlowItems(items)

	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(groups))
	}

	// Group 1: s1 (sequential)
	if len(groups[0]) != 1 || groups[0][0].Name != "s1" {
		t.Error("group 0 should be [s1]")
	}

	// Group 2: p1, p2 (parallel)
	if len(groups[1]) != 2 || groups[1][0].Name != "p1" || groups[1][1].Name != "p2" {
		t.Error("group 1 should be [p1, p2]")
	}

	// Group 3: s2 (sequential)
	if len(groups[2]) != 1 || groups[2][0].Name != "s2" {
		t.Error("group 2 should be [s2]")
	}

	// Group 4: p3 (parallel, single item)
	if len(groups[3]) != 1 || groups[3][0].Name != "p3" {
		t.Error("group 3 should be [p3]")
	}
}

func TestExecutionStateString(t *testing.T) {
	tests := []struct {
		state    ExecutionState
		expected string
	}{
		{StateNotStarted, "not_started"},
		{StateRunning, "running"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StateSkipped, "skipped"},
		{StateCancelled, "cancelled"},
		{ExecutionState(99), "unknown"},
	}

	for _, tc := range tests {
		if tc.state.String() != tc.expected {
			t.Errorf("state %d: expected %s, got %s", tc.state, tc.expected, tc.state.String())
		}
	}
}

func TestScenarioResultIsSuccess(t *testing.T) {
	result := &ScenarioResult{State: StateCompleted}
	if !result.IsSuccess() {
		t.Error("expected IsSuccess() to return true for Completed state")
	}

	result.State = StateFailed
	if result.IsSuccess() {
		t.Error("expected IsSuccess() to return false for Failed state")
	}
}

func TestTimeoutErrorUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &TimeoutError{
		Component: "test",
		Timeout:   5 * time.Second,
		Wrapped:   inner,
	}

	if err.Unwrap() != inner {
		t.Error("expected Unwrap to return inner error")
	}

	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to find inner error")
	}
}

func TestContextParameterPassing(t *testing.T) {
	exec := NewExecutor()

	var receivedValue string

	comp := core.NewComponent("read-param", core.ComponentSetup).
		WithFunc(func(ctx chronicleCtx.Context) error {
			v, ok := ctx.Get("testParam")
			if ok {
				receivedValue = v.(string)
			}
			return nil
		})
	exec.RegisterComponent(comp)

	s := scenario.NewScenario("param-test")
	s.AddFlow(scenario.NewFlowItem(core.ComponentSetup, "read-param").
		WithParams(map[string]any{"testParam": "test-value"}))

	exec.Execute(context.Background(), s)

	if receivedValue != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", receivedValue)
	}
}
