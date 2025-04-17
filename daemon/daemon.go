package daemon

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/joshua-temple/chronicle/scenarios"
)

// DaemonConfig holds configuration for the testing daemon
type DaemonConfig struct {
	// TestSets defines collections of tests to be executed
	TestSets []TestSet

	// MaxConcurrency is the maximum number of scenarios to run in parallel
	MaxConcurrency int

	// ExecutionInterval is the delay between scenario executions
	ExecutionInterval time.Duration

	// TotalRunTime is the total duration for which the daemon should run
	// If 0, runs indefinitely
	TotalRunTime time.Duration

	// ReportInterval defines how often to generate reports
	ReportInterval time.Duration

	// FailureBehavior controls what happens when a scenario fails
	FailureBehavior FailureBehavior

	// OutputHandlers receives test results
	OutputHandlers []OutputHandler
}

// FailureBehavior defines how the daemon responds to failures
type FailureBehavior int

const (
	// Continue runs the next scenario even after a failure
	Continue FailureBehavior = iota

	// PauseTestSet stops running scenarios from the same test set after a failure
	PauseTestSet

	// PauseAll stops all scenario execution after any failure
	PauseAll
)

// TestSet represents a group of related test components to be composed into scenarios
type TestSet struct {
	// Name of the test set
	Name string

	// Components is a list of component names to be included
	Components []string

	// SelectionStrategy determines how components are selected for each run
	SelectionStrategy SelectionStrategy

	// ExecutionWeight determines relative frequency of selection (higher = more frequent)
	ExecutionWeight int

	// RequiredParameters lists parameter names that must be provided
	RequiredParameters []string

	// ParameterGenerators provides generators for parameters
	ParameterGenerators map[string]scenarios.Generator
}

// SelectionStrategy defines how components are selected from a test set
type SelectionStrategy string

const (
	// RandomSelection selects components randomly
	RandomSelection SelectionStrategy = "random"

	// SequentialSelection selects components in sequence
	SequentialSelection SelectionStrategy = "sequential"

	// AllSelection selects all components for each run
	AllSelection SelectionStrategy = "all"
)

// OutputHandler processes test results
type OutputHandler interface {
	// HandleResult processes a test result
	HandleResult(result *ScenarioResult)

	// Flush ensures all results are processed
	Flush()
}

// ScenarioResult contains the results of a scenario execution
type ScenarioResult struct {
	// Scenario is the scenario that was executed
	Scenario *scenarios.Scenario

	// StartTime is when the scenario started
	StartTime time.Time

	// Duration is how long the scenario took to execute
	Duration time.Duration

	// TestSet is the test set the scenario belongs to
	TestSet string

	// Success indicates whether the scenario succeeded
	Success bool

	// Error is the error that occurred, if any
	Error error

	// Parameters holds the parameter values used
	Parameters map[string]interface{}
}

// Daemon is a service that continuously runs test scenarios
type Daemon struct {
	config          DaemonConfig
	registry        *ComponentRegistry
	running         bool
	stopCh          chan struct{}
	resultCh        chan *ScenarioResult
	wg              sync.WaitGroup
	testSetsEnabled map[string]bool
	stateLock       sync.RWMutex
}

// ComponentRegistry holds registered test components
type ComponentRegistry struct {
	components map[string]scenarios.Component
	mutex      sync.RWMutex
}

// NewComponentRegistry creates a new component registry
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[string]scenarios.Component),
	}
}

// RegisterComponent adds a component to the registry
func (r *ComponentRegistry) RegisterComponent(name string, component scenarios.Component) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.components[name] = component
}

// GetComponent retrieves a component from the registry
func (r *ComponentRegistry) GetComponent(name string) (scenarios.Component, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	component, ok := r.components[name]
	return component, ok
}

// ListComponents returns a list of all registered component names
func (r *ComponentRegistry) ListComponents() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}
	return names
}

// NewDaemon creates a new testing daemon
func NewDaemon(config DaemonConfig) *Daemon {
	// Set defaults if not specified
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 1
	}
	if config.ExecutionInterval <= 0 {
		config.ExecutionInterval = 100 * time.Millisecond
	}
	if config.ReportInterval <= 0 {
		config.ReportInterval = 1 * time.Minute
	}

	// Enable all test sets by default
	testSetsEnabled := make(map[string]bool)
	for _, ts := range config.TestSets {
		testSetsEnabled[ts.Name] = true
	}

	return &Daemon{
		config:          config,
		registry:        NewComponentRegistry(),
		stopCh:          make(chan struct{}),
		resultCh:        make(chan *ScenarioResult, 100),
		testSetsEnabled: testSetsEnabled,
	}
}

// Start begins the daemon's execution
func (d *Daemon) Start() error {
	d.stateLock.Lock()
	if d.running {
		d.stateLock.Unlock()
		return fmt.Errorf("daemon already running")
	}
	d.running = true
	d.stateLock.Unlock()

	// Reset stop channel
	d.stopCh = make(chan struct{})

	// Start result handler
	d.wg.Add(1)
	go d.processResults()

	// Start workers
	for i := 0; i < d.config.MaxConcurrency; i++ {
		d.wg.Add(1)
		go d.runWorker(i)
	}

	// If a total run time is specified, schedule shutdown
	if d.config.TotalRunTime > 0 {
		go func() {
			time.Sleep(d.config.TotalRunTime)
			d.Stop()
		}()
	}

	return nil
}

// Stop halts all daemon execution
func (d *Daemon) Stop() {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	if !d.running {
		return
	}

	close(d.stopCh)
	d.wg.Wait()
	d.running = false

	// Ensure all outputs are flushed
	for _, handler := range d.config.OutputHandlers {
		handler.Flush()
	}
}

// IsRunning returns whether the daemon is currently running
func (d *Daemon) IsRunning() bool {
	d.stateLock.RLock()
	defer d.stateLock.RUnlock()
	return d.running
}

// EnableTestSet enables a test set by name
func (d *Daemon) EnableTestSet(name string) {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()
	d.testSetsEnabled[name] = true
}

// DisableTestSet disables a test set by name
func (d *Daemon) DisableTestSet(name string) {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()
	d.testSetsEnabled[name] = false
}

// runWorker is the main worker loop for executing scenarios
func (d *Daemon) runWorker(id int) {
	defer d.wg.Done()

	for {
		select {
		case <-d.stopCh:
			return
		default:
			// Select a test set and create a scenario
			testSet, scenario, err := d.createScenario()
			if err != nil {
				// If we can't create a scenario, wait before trying again
				time.Sleep(d.config.ExecutionInterval)
				continue
			}

			// Execute the scenario
			result := &ScenarioResult{
				Scenario:   scenario,
				StartTime:  time.Now(),
				TestSet:    testSet.Name,
				Parameters: make(map[string]interface{}),
			}

			// Create context for execution
			ctx := scenarios.NewContext()

			// Apply parameter values to the result for reporting
			for _, param := range scenario.Parameters {
				if param.Generator != nil {
					val := param.Generator.Generate()
					ctx.SetParameter(param.Name, val)
					result.Parameters[param.Name] = val
				}
			}

			// Execute the scenario
			err = scenario.Execute(ctx)
			result.Duration = time.Since(result.StartTime)

			if err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.Success = true
			}

			// Send the result for processing
			d.resultCh <- result

			// Wait before next execution
			time.Sleep(d.config.ExecutionInterval)
		}
	}
}

// processResults handles test results
func (d *Daemon) processResults() {
	defer d.wg.Done()

	for {
		select {
		case <-d.stopCh:
			return
		case result := <-d.resultCh:
			// Process the result
			for _, handler := range d.config.OutputHandlers {
				handler.HandleResult(result)
			}

			// Handle failures according to configured behavior
			if !result.Success {
				switch d.config.FailureBehavior {
				case PauseTestSet:
					d.DisableTestSet(result.TestSet)
				case PauseAll:
					d.Stop()
					return
				case Continue:
					// Continue execution
				}
			}
		}
	}
}

// createScenario selects a test set and creates a scenario from it
func (d *Daemon) createScenario() (*TestSet, *scenarios.Scenario, error) {
	// Select a test set
	testSet := d.selectTestSet()
	if testSet == nil {
		return nil, nil, fmt.Errorf("no test sets available")
	}

	// Select components based on the strategy
	selectedComponents := d.selectComponents(testSet)
	if len(selectedComponents) == 0 {
		return nil, nil, fmt.Errorf("no components selected from test set %s", testSet.Name)
	}

	// Build the scenario
	builder := scenarios.NewBuilder(fmt.Sprintf("Auto-generated: %s", testSet.Name))

	// Add components
	for _, compName := range selectedComponents {
		component, ok := d.registry.GetComponent(compName)
		if !ok {
			return nil, nil, fmt.Errorf("component %s not found in registry", compName)
		}

		builder.AddComponent(compName, component)
	}

	// Add parameters
	for name, generator := range testSet.ParameterGenerators {
		builder.WithParameter(scenarios.Parameter{
			Name:      name,
			Generator: generator,
		})
	}

	return testSet, builder.Build(), nil
}

// selectTestSet randomly selects a test set based on weights
func (d *Daemon) selectTestSet() *TestSet {
	d.stateLock.RLock()
	defer d.stateLock.RUnlock()

	var enabledSets []TestSet
	var totalWeight int

	// Find enabled test sets and calculate total weight
	for _, ts := range d.config.TestSets {
		if enabled, ok := d.testSetsEnabled[ts.Name]; ok && enabled {
			enabledSets = append(enabledSets, ts)
			totalWeight += ts.ExecutionWeight
		}
	}

	if len(enabledSets) == 0 || totalWeight == 0 {
		return nil
	}

	// Select a random weight
	targetWeight := rand.Intn(totalWeight)
	currentWeight := 0

	// Find the test set corresponding to the selected weight
	for _, ts := range enabledSets {
		currentWeight += ts.ExecutionWeight
		if targetWeight < currentWeight {
			return &ts
		}
	}

	// Should never reach here if weights are positive
	return &enabledSets[0]
}

// selectComponents selects components from a test set based on its strategy
func (d *Daemon) selectComponents(testSet *TestSet) []string {
	if len(testSet.Components) == 0 {
		return nil
	}

	switch testSet.SelectionStrategy {
	case AllSelection:
		// Return all components
		return testSet.Components

	case SequentialSelection:
		// This would require maintaining state between calls
		// Simplified for this implementation
		return testSet.Components

	case RandomSelection:
		// Select a random subset of components
		numComponents := rand.Intn(len(testSet.Components)) + 1 // At least 1
		selected := make([]string, 0, numComponents)

		// Select unique components
		availableIndices := make([]int, len(testSet.Components))
		for i := range availableIndices {
			availableIndices[i] = i
		}

		for i := 0; i < numComponents; i++ {
			if len(availableIndices) == 0 {
				break
			}
			idx := rand.Intn(len(availableIndices))
			selected = append(selected, testSet.Components[availableIndices[idx]])

			// Remove the selected index
			availableIndices[idx] = availableIndices[len(availableIndices)-1]
			availableIndices = availableIndices[:len(availableIndices)-1]
		}

		return selected

	default:
		// Default to returning a single random component
		idx := rand.Intn(len(testSet.Components))
		return []string{testSet.Components[idx]}
	}
}

// RegisterComponent adds a component to the registry
func (d *Daemon) RegisterComponent(name string, component scenarios.Component) {
	d.registry.RegisterComponent(name, component)
}

// Init initializes the random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
