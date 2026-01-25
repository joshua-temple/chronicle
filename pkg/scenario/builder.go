package scenario

import (
	"time"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// Builder provides a fluent API for constructing scenarios.
type Builder struct {
	scenario *Scenario
}

// NewBuilder creates a new scenario builder.
func NewBuilder(name string) *Builder {
	return &Builder{
		scenario: NewScenario(name),
	}
}

// Description sets the scenario description.
func (b *Builder) Description(desc string) *Builder {
	b.scenario.Description = desc
	return b
}

// Timeout sets the scenario timeout.
func (b *Builder) Timeout(d time.Duration) *Builder {
	b.scenario.Timeout = d
	return b
}

// Tags adds tags to the scenario.
func (b *Builder) Tags(tags ...string) *Builder {
	b.scenario.Tags = append(b.scenario.Tags, tags...)
	return b
}

// Extends sets the parent scenario for inheritance.
func (b *Builder) Extends(parent string) *Builder {
	b.scenario.Extends = parent
	return b
}

// Abstract marks the scenario as abstract (cannot be run directly).
func (b *Builder) Abstract() *Builder {
	b.scenario.Abstract = true
	return b
}

// Setup adds a setup component to the flow.
func (b *Builder) Setup(name string) *Builder {
	b.scenario.AddFlow(NewFlowItem(core.ComponentSetup, name))
	return b
}

// SetupWithTimeout adds a setup component with a specific timeout.
func (b *Builder) SetupWithTimeout(name string, timeout time.Duration) *Builder {
	b.scenario.AddFlow(NewFlowItem(core.ComponentSetup, name).WithTimeout(timeout))
	return b
}

// Task adds a task component to the flow.
func (b *Builder) Task(name string) *Builder {
	b.scenario.AddFlow(NewFlowItem(core.ComponentTask, name))
	return b
}

// TaskWithParams adds a task component with parameters.
func (b *Builder) TaskWithParams(name string, params map[string]any) *Builder {
	b.scenario.AddFlow(NewFlowItem(core.ComponentTask, name).WithParams(params))
	return b
}

// Validation adds a validation component to the flow.
func (b *Builder) Validation(name string) *Builder {
	b.scenario.AddFlow(NewFlowItem(core.ComponentValidation, name))
	return b
}

// Step adds a step component to the flow.
func (b *Builder) Step(name string) *Builder {
	b.scenario.AddFlow(NewFlowItem(core.ComponentStep, name))
	return b
}

// Rollup adds a rollup component to the flow.
func (b *Builder) Rollup(name string) *Builder {
	b.scenario.AddFlow(NewFlowItem(core.ComponentRollup, name))
	return b
}

// Teardown adds a teardown component to the teardown flow.
func (b *Builder) Teardown(name string) *Builder {
	b.scenario.AddTeardown(NewFlowItem(core.ComponentTeardown, name))
	return b
}

// Flow adds a custom flow item.
func (b *Builder) Flow(item FlowItem) *Builder {
	b.scenario.AddFlow(item)
	return b
}

// Parallel adds multiple flow items to be executed in parallel.
func (b *Builder) Parallel(items ...FlowItem) *Builder {
	parallelItem := FlowItem{
		Type:          core.ComponentStep, // Parallel block acts like a step
		Name:          "parallel-block",
		Parallel:      true,
		ParallelItems: items,
	}
	b.scenario.AddFlow(parallelItem)
	return b
}

// Flag sets a flag for the scenario.
func (b *Builder) Flag(name string, value any) *Builder {
	b.scenario.SetFlag(name, value)
	return b
}

// Flags sets multiple flags.
func (b *Builder) Flags(flags map[string]any) *Builder {
	for k, v := range flags {
		b.scenario.SetFlag(k, v)
	}
	return b
}

// Options adds option names to the scenario.
func (b *Builder) Options(options ...string) *Builder {
	b.scenario.Options = append(b.scenario.Options, options...)
	return b
}

// ChaosProfiles adds chaos profile names.
func (b *Builder) ChaosProfiles(profiles ...string) *Builder {
	b.scenario.ChaosProfiles = append(b.scenario.ChaosProfiles, profiles...)
	return b
}

// MockProfiles adds mock profile names.
func (b *Builder) MockProfiles(profiles ...string) *Builder {
	b.scenario.MockProfiles = append(b.scenario.MockProfiles, profiles...)
	return b
}

// SkipIf adds a skip condition.
func (b *Builder) SkipIf(expression, reason string) *Builder {
	b.scenario.SkipIf = append(b.scenario.SkipIf, Condition{
		Expression: expression,
		Reason:     reason,
	})
	return b
}

// SkipUnless adds a skip-unless condition.
func (b *Builder) SkipUnless(expression, reason string) *Builder {
	b.scenario.SkipUnless = append(b.scenario.SkipUnless, Condition{
		Expression: expression,
		Reason:     reason,
	})
	return b
}

// Matrix sets a matrix parameter for parameterized testing.
func (b *Builder) Matrix(key string, values []any) *Builder {
	b.scenario.Matrix[key] = values
	return b
}

// Build finalizes and returns the scenario.
func (b *Builder) Build() *Scenario {
	return b.scenario
}

// MustBuild finalizes and returns the scenario, panicking on validation errors.
func (b *Builder) MustBuild() *Scenario {
	s := b.scenario
	if !s.Abstract && len(s.Flow) == 0 {
		panic("scenario must have at least one flow item or be abstract")
	}
	if s.Name == "" {
		panic("scenario must have a name")
	}
	return s
}
