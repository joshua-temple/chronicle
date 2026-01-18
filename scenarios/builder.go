package scenarios

import (
	"time"

	"github.com/joshua-temple/chronicle/chronicle"
	"github.com/joshua-temple/chronicle/metrics"
)

// ScenarioBuilder provides a fluent interface for building scenarios
type ScenarioBuilder struct {
	scenario          *Scenario
	chronicleEnabled  bool
	metricsCollector  metrics.Collector
}

// NewBuilder creates a new scenario builder with the given name
func NewBuilder(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		scenario: &Scenario{
			Name:       name,
			Setup:      []Component{},
			Components: []namedComponent{},
			Teardown:   []Component{},
			Parameters: []Parameter{},
		},
	}
}

// WithDescription adds a description to the scenario
func (b *ScenarioBuilder) WithDescription(description string) *ScenarioBuilder {
	b.scenario.Description = description
	return b
}

// WithTimeout sets a timeout for the scenario
func (b *ScenarioBuilder) WithTimeout(timeout time.Duration) *ScenarioBuilder {
	b.scenario.Timeout = timeout
	return b
}

// SetupWith adds a setup component to the scenario
func (b *ScenarioBuilder) SetupWith(setup Component) *ScenarioBuilder {
	b.scenario.Setup = append(b.scenario.Setup, setup)
	return b
}

// AddComponent adds a component to the scenario
func (b *ScenarioBuilder) AddComponent(name string, component Component) *ScenarioBuilder {
	b.scenario.Components = append(b.scenario.Components, namedComponent{
		Name:      name,
		Component: component,
	})
	return b
}

// WithConditionalComponent adds a component with a condition to the scenario
func (b *ScenarioBuilder) WithConditionalComponent(
	name string,
	condition func(*Context) bool,
	component Component,
) *ScenarioBuilder {
	b.scenario.Components = append(b.scenario.Components, namedComponent{
		Name:      name,
		Component: component,
		Condition: condition,
	})
	return b
}

// TeardownWith adds a teardown component to the scenario
func (b *ScenarioBuilder) TeardownWith(teardown Component) *ScenarioBuilder {
	b.scenario.Teardown = append(b.scenario.Teardown, teardown)
	return b
}

// WithParameter adds a parameter to the scenario
func (b *ScenarioBuilder) WithParameter(param Parameter) *ScenarioBuilder {
	b.scenario.Parameters = append(b.scenario.Parameters, param)
	return b
}

// WithStringParameter adds a string parameter with a generator to the scenario
func (b *ScenarioBuilder) WithStringParameter(
	name string,
	description string,
	minLength int,
	maxLength int,
	charset string,
) *ScenarioBuilder {
	return b.WithParameter(Parameter{
		Name:        name,
		Description: description,
		Generator: &StringGenerator{
			MinLength: minLength,
			MaxLength: maxLength,
			Charset:   charset,
		},
	})
}

// WithIntParameter adds an integer parameter with a generator to the scenario
func (b *ScenarioBuilder) WithIntParameter(
	name string,
	description string,
	min int,
	max int,
) *ScenarioBuilder {
	return b.WithParameter(Parameter{
		Name:        name,
		Description: description,
		Generator: &IntGenerator{
			Min: min,
			Max: max,
		},
	})
}

// WithEnumParameter adds an enum parameter with a generator to the scenario
func (b *ScenarioBuilder) WithEnumParameter(
	name string,
	description string,
	options []interface{},
) *ScenarioBuilder {
	return b.WithParameter(Parameter{
		Name:        name,
		Description: description,
		Generator: &EnumGenerator{
			Options: options,
		},
	})
}

// WithChronicle enables chronicle recording for the scenario
func (b *ScenarioBuilder) WithChronicle() *ScenarioBuilder {
	b.chronicleEnabled = true
	return b
}

// WithMetrics sets a metrics collector for the scenario
func (b *ScenarioBuilder) WithMetrics(collector metrics.Collector) *ScenarioBuilder {
	b.metricsCollector = collector
	return b
}

// Build creates the scenario
func (b *ScenarioBuilder) Build() *Scenario {
	return b.scenario
}

// Execute builds and runs the scenario, returning the chronicle root if enabled
func (b *ScenarioBuilder) Execute() (*chronicle.Entry, error) {
	ctx := NewContext()

	// Set up chronicle if enabled
	if b.chronicleEnabled {
		chron := chronicle.New(b.scenario.Name, b.scenario.Description)
		ctx.SetChronicle(chron)
	}

	// Set up metrics if collector provided
	if b.metricsCollector != nil {
		ctx.SetMetrics(metrics.NewEmitter(b.metricsCollector))
	}

	err := b.scenario.Execute(ctx)

	// Return chronicle root if enabled
	if b.chronicleEnabled {
		return ctx.Chronicle().Root(), err
	}
	return nil, err
}

// New creates a scenario with the given name and returns the builder
func New(name string) *ScenarioBuilder {
	return NewBuilder(name)
}

// Run executes a scenario with a new context
func Run(scenario *Scenario) error {
	ctx := NewContext()
	return scenario.Execute(ctx)
}

// RunWithChronicle executes a scenario with chronicle recording enabled
func RunWithChronicle(scenario *Scenario) (*chronicle.Entry, error) {
	ctx := NewContext()
	chron := chronicle.New(scenario.Name, scenario.Description)
	ctx.SetChronicle(chron)
	err := scenario.Execute(ctx)
	return chron.Root(), err
}

// RunWithMetrics executes a scenario with metrics collection
func RunWithMetrics(scenario *Scenario, collector metrics.Collector) error {
	ctx := NewContext()
	ctx.SetMetrics(metrics.NewEmitter(collector))
	return scenario.Execute(ctx)
}
