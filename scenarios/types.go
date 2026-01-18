package scenarios

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/joshua-temple/chronicle/chronicle"
	"github.com/joshua-temple/chronicle/core"
	"github.com/joshua-temple/chronicle/metrics"
)

// Component represents a reusable test component that can be composed into scenarios.
// Components are designed to be independent of testing.T to allow for reuse in
// different contexts, such as automated testing or continuous chaos testing.
type Component func(ctx *Context) error

// Context provides shared state and utilities for scenario components.
// It wraps the core.TestContext but can be used without an actual testing.T
// for daemon-mode operations.
type Context struct {
	ctx        context.Context
	cancel     context.CancelFunc
	store      *core.DataStore
	parameters map[string]interface{}
	logger     Logger
	parent     *core.TestContext // Optional, may be nil in daemon mode
	chron      *chronicle.Chronicle
	metricsEm  *metrics.Emitter
	mutex      sync.RWMutex
}

// Debug logs a debug message
func (c *Context) Debug(format string, args ...interface{}) {
	c.logger.Debug(format, args...)
}

// Info logs an info message
func (c *Context) Info(format string, args ...interface{}) {
	c.logger.Info(format, args...)
}

// Success logs a success message
func (c *Context) Success(format string, args ...interface{}) {
	c.logger.Success(format, args...)
}

// Warning logs a warning message
func (c *Context) Warning(format string, args ...interface{}) {
	c.logger.Warning(format, args...)
}

// Error logs an error message
func (c *Context) Error(format string, args ...interface{}) {
	c.logger.Error(format, args...)
}

// Fatal logs a fatal message
func (c *Context) Fatal(format string, args ...interface{}) {
	c.logger.Fatal(format, args...)
}

// Logger defines the logging interface for scenarios
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Success(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

// defaultLogger provides a basic implementation of the Logger interface
type defaultLogger struct{}

func (l *defaultLogger) Debug(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}
func (l *defaultLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}
func (l *defaultLogger) Success(format string, args ...interface{}) {
	fmt.Printf("[SUCCESS] "+format+"\n", args...)
}
func (l *defaultLogger) Warning(format string, args ...interface{}) {
	fmt.Printf("[WARNING] "+format+"\n", args...)
}
func (l *defaultLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
func (l *defaultLogger) Fatal(format string, args ...interface{}) {
	fmt.Printf("[FATAL] "+format+"\n", args...)
}

// NewContext creates a new scenario context
func NewContext() *Context {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		ctx:        ctx,
		cancel:     cancel,
		store:      core.NewDataStore(),
		parameters: make(map[string]interface{}),
		logger:     &defaultLogger{},
	}
}

// WithCoreContext creates a scenario context from a core.TestContext
func WithCoreContext(parent *core.TestContext) *Context {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		ctx:        ctx,
		cancel:     cancel,
		store:      parent.Store,
		parameters: make(map[string]interface{}),
		logger:     &testContextLogger{parent},
		parent:     parent,
	}
}

// testContextLogger adapts a core.TestContext to the Logger interface
type testContextLogger struct {
	tc *core.TestContext
}

func (l *testContextLogger) Debug(format string, args ...interface{}) { l.tc.Debug(format, args...) }
func (l *testContextLogger) Info(format string, args ...interface{})  { l.tc.Info(format, args...) }
func (l *testContextLogger) Success(format string, args ...interface{}) {
	l.tc.Success(format, args...)
}
func (l *testContextLogger) Warning(format string, args ...interface{}) {
	l.tc.Warning(format, args...)
}
func (l *testContextLogger) Error(format string, args ...interface{}) { l.tc.Error(format, args...) }
func (l *testContextLogger) Fatal(format string, args ...interface{}) { l.tc.Fatal(format, args...) }

// Get retrieves a value from the context store
func (c *Context) Get(key string) (interface{}, bool) {
	return c.store.Get(key)
}

// Set stores a value in the context store
func (c *Context) Set(key string, value interface{}) {
	c.store.Set(key, value)
}

// GetString retrieves a string value from the context store
func (c *Context) GetString(key string) (string, bool) {
	return c.store.GetString(key)
}

// GetInt retrieves an int value from the context store
func (c *Context) GetInt(key string) (int, bool) {
	return c.store.GetInt(key)
}

// GetParameter retrieves a parameter value
func (c *Context) GetParameter(name string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	value, ok := c.parameters[name]
	return value, ok
}

// SetParameter sets a parameter value
func (c *Context) SetParameter(name string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.parameters[name] = value
}

// ParentContext returns the parent core.TestContext, if any
func (c *Context) ParentContext() *core.TestContext {
	return c.parent
}

// Context returns the underlying context.Context
func (c *Context) Context() context.Context {
	return c.ctx
}

// Cancel cancels the context
func (c *Context) Cancel() {
	c.cancel()
}

// Store returns the context's data store
func (c *Context) Store() *core.DataStore {
	return c.store
}

// Chronicle returns the chronicle recorder, or a no-op if not configured
func (c *Context) Chronicle() *chronicle.Chronicle {
	if c.chron == nil {
		return chronicle.Noop()
	}
	return c.chron
}

// Metrics returns the metrics emitter, or a no-op if not configured
func (c *Context) Metrics() *metrics.Emitter {
	if c.metricsEm == nil {
		return metrics.NoopEmitter()
	}
	return c.metricsEm
}

// SetChronicle sets the chronicle recorder
func (c *Context) SetChronicle(chron *chronicle.Chronicle) {
	c.chron = chron
}

// SetMetrics sets the metrics emitter
func (c *Context) SetMetrics(em *metrics.Emitter) {
	c.metricsEm = em
}

// Parameter defines a parameter with optional generator for a scenario
type Parameter struct {
	Name        string
	Description string
	Generator   Generator
	Default     interface{}
}

// Generator defines an interface for generating random but constrained values
type Generator interface {
	Generate() interface{}
}

// StringGenerator generates random strings within length constraints
type StringGenerator struct {
	MinLength int
	MaxLength int
	Charset   string
}

// Generate implements the Generator interface
func (g *StringGenerator) Generate() interface{} {
	if g.Charset == "" {
		g.Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	length := g.MinLength
	if g.MaxLength > g.MinLength {
		length = g.MinLength + rand.Intn(g.MaxLength-g.MinLength+1)
	}

	result := make([]byte, length)
	for i := range result {
		result[i] = g.Charset[rand.Intn(len(g.Charset))]
	}
	return string(result)
}

// IntGenerator generates random integers within constraints
type IntGenerator struct {
	Min int
	Max int
}

// Generate implements the Generator interface
func (g *IntGenerator) Generate() interface{} {
	return g.Min + rand.Intn(g.Max-g.Min+1)
}

// EnumGenerator generates random values from a set of options
type EnumGenerator struct {
	Options []interface{}
}

// Generate implements the Generator interface
func (g *EnumGenerator) Generate() interface{} {
	if len(g.Options) == 0 {
		return nil
	}
	return g.Options[rand.Intn(len(g.Options))]
}

// namedComponent associates a name with a component and optional condition
type namedComponent struct {
	Name      string
	Component Component
	Condition func(*Context) bool
}

// Scenario represents a test scenario composed of multiple components
type Scenario struct {
	Name        string
	Description string
	Setup       []Component
	Components  []namedComponent
	Teardown    []Component
	Parameters  []Parameter
	Timeout     time.Duration
}

// Execute runs the scenario with the given context
func (s *Scenario) Execute(ctx *Context) error {
	start := time.Now()

	// Emit metrics for scenario start
	ctx.Metrics().ScenarioStart(ctx.ctx, s.Name, s.Name)

	var execErr error
	defer func() {
		status := metrics.StatusPassed
		if execErr != nil {
			status = metrics.StatusFailed
		}
		ctx.Metrics().ScenarioEnd(ctx.ctx, s.Name, s.Name, time.Since(start), status, execErr)
		ctx.Chronicle().Finalize()
	}()

	// Set timeout
	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx.ctx, cancel = context.WithTimeout(ctx.ctx, s.Timeout)
		defer cancel()
	}

	// Generate parameter values
	for _, param := range s.Parameters {
		if param.Generator != nil {
			ctx.SetParameter(param.Name, param.Generator.Generate())
		} else if param.Default != nil {
			ctx.SetParameter(param.Name, param.Default)
		}
	}

	// Execute setup
	ctx.logger.Info("Starting scenario: %s", s.Name)
	for i, setup := range s.Setup {
		setupName := fmt.Sprintf("Setup %d", i+1)
		done := ctx.Chronicle().Setup(setupName, "")
		ctx.logger.Debug("Running setup step %d", i+1)
		err := setup(ctx)
		done(err)
		if err != nil {
			execErr = fmt.Errorf("setup failed: %w", err)
			return execErr
		}
	}

	// Execute components
	for _, component := range s.Components {
		// Check if component should be executed
		if component.Condition != nil && !component.Condition(ctx) {
			ctx.logger.Debug("Skipping component %s (condition not met)", component.Name)
			ctx.Chronicle().Skip(component.Name, "condition not met")
			continue
		}

		compStart := time.Now()
		done := ctx.Chronicle().Component(component.Name, "")
		ctx.logger.Debug("Running component: %s", component.Name)
		err := component.Component(ctx)
		done(err)

		// Emit component metrics
		compStatus := metrics.StatusPassed
		if err != nil {
			compStatus = metrics.StatusFailed
		}
		ctx.Metrics().ComponentEnd(ctx.ctx, s.Name, component.Name, time.Since(compStart), compStatus, err)

		if err != nil {
			execErr = fmt.Errorf("component %s failed: %w", component.Name, err)
			return execErr
		}
	}

	// Execute teardown (always attempt teardown, even if components failed)
	for i, teardown := range s.Teardown {
		teardownName := fmt.Sprintf("Teardown %d", i+1)
		done := ctx.Chronicle().Teardown(teardownName, "")
		ctx.logger.Debug("Running teardown step %d", i+1)
		err := teardown(ctx)
		done(err)
		if err != nil {
			ctx.logger.Warning("Teardown step %d failed: %v", i+1, err)
		}
	}

	ctx.logger.Success("Scenario completed: %s", s.Name)
	return nil
}

// Init initializes the random number generator
func init() {
	rand.Seed(time.Now().UnixNano())
}
