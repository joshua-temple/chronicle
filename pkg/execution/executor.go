package execution

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	chronicleCtx "github.com/joshua-temple/chronicle/pkg/context"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/infrastructure"
	"github.com/joshua-temple/chronicle/pkg/middleware"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

// ExecutionState represents the state of a scenario execution.
type ExecutionState int

const (
	// StateNotStarted indicates execution hasn't begun.
	StateNotStarted ExecutionState = iota
	// StateRunning indicates execution is in progress.
	StateRunning
	// StateCompleted indicates execution completed successfully.
	StateCompleted
	// StateFailed indicates execution failed.
	StateFailed
	// StateSkipped indicates execution was skipped.
	StateSkipped
	// StateCancelled indicates execution was cancelled.
	StateCancelled
)

func (s ExecutionState) String() string {
	switch s {
	case StateNotStarted:
		return "not_started"
	case StateRunning:
		return "running"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateSkipped:
		return "skipped"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// FlowItemResult represents the result of executing a single flow item.
type FlowItemResult struct {
	Name      string
	Type      core.ComponentType
	State     ExecutionState
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
	Output    any
}

// ScenarioResult represents the result of executing a scenario.
type ScenarioResult struct {
	ScenarioID      core.ScenarioID
	ScenarioName    string
	State           ExecutionState
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	FlowResults     []FlowItemResult
	TeardownResults []FlowItemResult
	Error           error
	SkipReason      string
}

// IsSuccess returns true if the scenario completed successfully.
func (r *ScenarioResult) IsSuccess() bool {
	return r.State == StateCompleted
}

// Executor executes scenarios and their flow items.
type Executor struct {
	mu sync.RWMutex

	// Component registry for looking up component functions
	components map[string]*core.Component

	// Infrastructure manager for providing clients
	infraManager *infrastructure.Manager

	// Middleware chain to apply to component execution
	middlewareChain middleware.Middleware

	// Configuration
	defaultTimeout time.Duration
	parallelism    int
	failFast       bool

	// Hooks
	beforeScenario func(ctx context.Context, s *scenario.Scenario) error
	afterScenario  func(ctx context.Context, s *scenario.Scenario, result *ScenarioResult)
	beforeItem     func(ctx context.Context, item *scenario.FlowItem)
	afterItem      func(ctx context.Context, item *scenario.FlowItem, result *FlowItemResult)
}

// ExecutorOption configures an Executor.
type ExecutorOption func(*Executor)

// NewExecutor creates a new Executor with the given options.
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		components:     make(map[string]*core.Component),
		defaultTimeout: 30 * time.Second,
		parallelism:    1,
		failFast:       false,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// WithDefaultTimeout sets the default timeout for flow items.
func WithDefaultTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.defaultTimeout = timeout
	}
}

// WithParallelism sets the maximum number of parallel executions.
func WithParallelism(n int) ExecutorOption {
	return func(e *Executor) {
		if n < 1 {
			n = 1
		}
		e.parallelism = n
	}
}

// WithFailFast enables fail-fast mode (stop on first failure).
func WithFailFast(enabled bool) ExecutorOption {
	return func(e *Executor) {
		e.failFast = enabled
	}
}

// WithInfrastructure sets the infrastructure manager.
func WithInfrastructure(mgr *infrastructure.Manager) ExecutorOption {
	return func(e *Executor) {
		e.infraManager = mgr
	}
}

// WithMiddleware sets the middleware chain.
func WithMiddleware(mw middleware.Middleware) ExecutorOption {
	return func(e *Executor) {
		e.middlewareChain = mw
	}
}

// WithBeforeScenario sets a hook to run before each scenario.
func WithBeforeScenario(fn func(ctx context.Context, s *scenario.Scenario) error) ExecutorOption {
	return func(e *Executor) {
		e.beforeScenario = fn
	}
}

// WithAfterScenario sets a hook to run after each scenario.
func WithAfterScenario(fn func(ctx context.Context, s *scenario.Scenario, result *ScenarioResult)) ExecutorOption {
	return func(e *Executor) {
		e.afterScenario = fn
	}
}

// WithBeforeItem sets a hook to run before each flow item.
func WithBeforeItem(fn func(ctx context.Context, item *scenario.FlowItem)) ExecutorOption {
	return func(e *Executor) {
		e.beforeItem = fn
	}
}

// WithAfterItem sets a hook to run after each flow item.
func WithAfterItem(fn func(ctx context.Context, item *scenario.FlowItem, result *FlowItemResult)) ExecutorOption {
	return func(e *Executor) {
		e.afterItem = fn
	}
}

// RegisterComponent registers a component for execution.
func (e *Executor) RegisterComponent(c *core.Component) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.components[c.Name] = c
}

// RegisterComponents registers multiple components.
func (e *Executor) RegisterComponents(components []*core.Component) {
	for _, c := range components {
		e.RegisterComponent(c)
	}
}

// GetComponent returns a component by name.
func (e *Executor) GetComponent(name string) (*core.Component, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	c, ok := e.components[name]
	return c, ok
}

// Execute runs a scenario and returns the result.
func (e *Executor) Execute(ctx context.Context, s *scenario.Scenario) *ScenarioResult {
	result := &ScenarioResult{
		ScenarioID:   s.ID,
		ScenarioName: s.Name,
		State:        StateNotStarted,
		StartTime:    time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// Check if scenario is runnable
	if !s.IsRunnable() {
		result.State = StateSkipped
		result.SkipReason = "scenario is abstract or has no flow items"
		return result
	}

	// Apply scenario timeout
	timeout := s.EffectiveTimeout(e.defaultTimeout)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Run before scenario hook
	if e.beforeScenario != nil {
		if err := e.beforeScenario(ctx, s); err != nil {
			result.State = StateFailed
			result.Error = fmt.Errorf("before scenario hook failed: %w", err)
			return result
		}
	}

	// Create Chronicle context for the scenario with client provider
	var clientProvider func(string) (any, error)
	if e.infraManager != nil {
		clientProvider = func(name string) (any, error) {
			return e.infraManager.ClientByService(name)
		}
	}

	chronicleContext := chronicleCtx.New(ctx, chronicleCtx.WithClientProvider(clientProvider))
	chronicleContext.SetComponentName(s.Name)

	// Execute the main flow
	result.State = StateRunning
	flowErr := e.executeFlow(ctx, chronicleContext, s.Flow, &result.FlowResults)

	// Always run teardown, even if flow failed
	if len(s.TeardownFlow) > 0 {
		teardownErr := e.executeFlow(ctx, chronicleContext, s.TeardownFlow, &result.TeardownResults)
		if teardownErr != nil && flowErr == nil {
			flowErr = fmt.Errorf("teardown failed: %w", teardownErr)
		}
	}

	// Set final state
	if ctx.Err() == context.Canceled {
		result.State = StateCancelled
		result.Error = ctx.Err()
	} else if ctx.Err() == context.DeadlineExceeded {
		result.State = StateFailed
		result.Error = fmt.Errorf("scenario timeout exceeded: %w", ctx.Err())
	} else if flowErr != nil {
		result.State = StateFailed
		result.Error = flowErr
	} else {
		result.State = StateCompleted
	}

	// Run after scenario hook
	if e.afterScenario != nil {
		e.afterScenario(ctx, s, result)
	}

	return result
}

// executeFlow executes a list of flow items.
func (e *Executor) executeFlow(ctx context.Context, chronicleContext chronicleCtx.Context, items []scenario.FlowItem, results *[]FlowItemResult) error {
	var errs []error

	// Group consecutive parallel items
	groups := e.groupFlowItems(items)

	for _, group := range groups {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if len(group) == 1 && !group[0].Parallel {
			// Execute single item
			result := e.executeItem(ctx, chronicleContext, &group[0])
			*results = append(*results, result)

			if result.Error != nil {
				errs = append(errs, result.Error)
				if e.failFast {
					return errors.Join(errs...)
				}
			}
		} else {
			// Execute parallel group
			groupResults := e.executeParallel(ctx, chronicleContext, group)
			*results = append(*results, groupResults...)

			for _, r := range groupResults {
				if r.Error != nil {
					errs = append(errs, r.Error)
					if e.failFast {
						return errors.Join(errs...)
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// groupFlowItems groups consecutive parallel items together.
func (e *Executor) groupFlowItems(items []scenario.FlowItem) [][]scenario.FlowItem {
	var groups [][]scenario.FlowItem
	var currentGroup []scenario.FlowItem
	inParallel := false

	for _, item := range items {
		if item.Parallel {
			if !inParallel {
				// Start new parallel group
				if len(currentGroup) > 0 {
					groups = append(groups, currentGroup)
					currentGroup = nil
				}
				inParallel = true
			}
			currentGroup = append(currentGroup, item)
		} else {
			if inParallel {
				// End parallel group
				if len(currentGroup) > 0 {
					groups = append(groups, currentGroup)
					currentGroup = nil
				}
				inParallel = false
			}
			// Single sequential item
			groups = append(groups, []scenario.FlowItem{item})
		}
	}

	// Add final group if any
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// executeParallel executes a group of items in parallel.
func (e *Executor) executeParallel(ctx context.Context, chronicleContext chronicleCtx.Context, items []scenario.FlowItem) []FlowItemResult {
	results := make([]FlowItemResult, len(items))
	var wg sync.WaitGroup

	// Limit parallelism
	sem := make(chan struct{}, e.parallelism)

	for i, item := range items {
		wg.Add(1)
		go func(idx int, flowItem scenario.FlowItem) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create child context for parallel item
			childCtx := chronicleContext.WithSpan(flowItem.Name)
			results[idx] = e.executeItem(ctx, childCtx, &flowItem)
		}(i, item)
	}

	wg.Wait()
	return results
}

// executeItem executes a single flow item.
func (e *Executor) executeItem(ctx context.Context, chronicleContext chronicleCtx.Context, item *scenario.FlowItem) FlowItemResult {
	result := FlowItemResult{
		Name:      item.Name,
		Type:      item.Type,
		State:     StateNotStarted,
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// Run before item hook
	if e.beforeItem != nil {
		e.beforeItem(ctx, item)
	}

	// Defer after item hook
	defer func() {
		if e.afterItem != nil {
			e.afterItem(ctx, item, &result)
		}
	}()

	// Apply item timeout
	timeout := item.Timeout
	if timeout == 0 {
		timeout = e.defaultTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Look up the component
	component, ok := e.GetComponent(item.Name)
	if !ok {
		result.State = StateFailed
		result.Error = fmt.Errorf("component not found: %s", item.Name)
		return result
	}

	// Check if component has a function bound
	if component.Func == nil {
		result.State = StateFailed
		result.Error = fmt.Errorf("component %s has no function bound", item.Name)
		return result
	}

	// Create execution context with component info
	execCtx := chronicleContext.Child(item.Name)
	execCtx.SetComponentName(item.Name)

	// Set parameters in context
	for k, v := range item.Params {
		execCtx.Set(k, v)
	}

	result.State = StateRunning

	// Execute the component function
	err := e.invokeComponent(ctx, execCtx, component, &result)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.State = StateFailed
			result.Error = &TimeoutError{
				Component: item.Name,
				Timeout:   timeout,
				Wrapped:   err,
			}
		} else if ctx.Err() == context.Canceled {
			result.State = StateCancelled
			result.Error = ctx.Err()
		} else {
			result.State = StateFailed
			result.Error = err
		}
	} else {
		result.State = StateCompleted
	}

	return result
}

// invokeComponent invokes a component's function with the given context.
func (e *Executor) invokeComponent(ctx context.Context, execCtx chronicleCtx.Context, component *core.Component, result *FlowItemResult) error {
	// Create the runner function
	runner := func(runCtx chronicleCtx.Context) error {
		// Try to cast to known function signatures
		switch fn := component.Func.(type) {
		case func(chronicleCtx.Context) error:
			return fn(runCtx)

		case func(core.Context) error:
			return fn(runCtx)

		case core.SetupFunc:
			return fn(runCtx)

		case core.StepFunc:
			return fn(runCtx)

		case core.TeardownFunc:
			return fn(runCtx)

		case core.RollupFunc:
			return fn(runCtx)

		default:
			return fmt.Errorf("unsupported function type for component %s: %T", component.Name, component.Func)
		}
	}

	// Apply middleware if configured
	if e.middlewareChain != nil {
		wrappedRunner := e.middlewareChain(func(mwCtx chronicleCtx.Context) error {
			return runner(mwCtx)
		})
		return wrappedRunner(execCtx)
	}

	return runner(execCtx)
}

// TimeoutError represents a timeout during component execution.
type TimeoutError struct {
	Component string
	Timeout   time.Duration
	Wrapped   error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("component %s timed out after %v: %v", e.Component, e.Timeout, e.Wrapped)
}

func (e *TimeoutError) Unwrap() error {
	return e.Wrapped
}

// ExecuteMultiple executes multiple scenarios and returns all results.
func (e *Executor) ExecuteMultiple(ctx context.Context, scenarios []*scenario.Scenario) []*ScenarioResult {
	results := make([]*ScenarioResult, len(scenarios))

	for i, s := range scenarios {
		select {
		case <-ctx.Done():
			// Mark remaining as cancelled
			for j := i; j < len(scenarios); j++ {
				results[j] = &ScenarioResult{
					ScenarioID:   scenarios[j].ID,
					ScenarioName: scenarios[j].Name,
					State:        StateCancelled,
					Error:        ctx.Err(),
				}
			}
			return results
		default:
			results[i] = e.Execute(ctx, s)

			// Check fail-fast
			if e.failFast && results[i].State == StateFailed {
				// Mark remaining as skipped
				for j := i + 1; j < len(scenarios); j++ {
					results[j] = &ScenarioResult{
						ScenarioID:   scenarios[j].ID,
						ScenarioName: scenarios[j].Name,
						State:        StateSkipped,
						SkipReason:   "skipped due to fail-fast",
					}
				}
				return results
			}
		}
	}

	return results
}
