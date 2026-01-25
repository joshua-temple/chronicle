package scenario

import (
	"fmt"
	"time"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// Scenario represents a test scenario with its flow and configuration.
type Scenario struct {
	ID          core.ScenarioID
	Name        string
	Description string
	Timeout     time.Duration
	Tags        []string

	// Flow definition
	Flow         []FlowItem
	TeardownFlow []FlowItem

	// Execution modifiers
	Flags         map[string]any
	Options       []string
	ChaosProfiles []string
	MockProfiles  []string

	// Conditions
	SkipIf     []Condition
	SkipUnless []Condition

	// Matrix parameters (for parameterized scenarios)
	Matrix      map[string][]any
	MatrixIndex map[string]any // Current matrix values for this instance

	// Inheritance
	Extends  string
	Abstract bool

	// Metadata
	SourceFile string
	SourceLine int
}

// FlowItem represents a single item in the scenario flow.
type FlowItem struct {
	Type      core.ComponentType
	Name      string
	Component *core.Component // Resolved component reference
	Timeout   time.Duration
	DependsOn []string
	Params    map[string]any
	Parallel  bool

	// For parallel blocks containing multiple items
	ParallelItems []FlowItem
}

// Condition represents a skip condition.
type Condition struct {
	Expression string
	Reason     string
}

// NewScenario creates a new scenario with the given name.
func NewScenario(name string) *Scenario {
	return &Scenario{
		ID:            core.NewScenarioID(),
		Name:          name,
		Flags:         make(map[string]any),
		Matrix:        make(map[string][]any),
		MatrixIndex:   make(map[string]any),
	}
}

// AddFlow adds a flow item to the scenario.
func (s *Scenario) AddFlow(item FlowItem) {
	s.Flow = append(s.Flow, item)
}

// AddTeardown adds a teardown flow item.
func (s *Scenario) AddTeardown(item FlowItem) {
	s.TeardownFlow = append(s.TeardownFlow, item)
}

// SetFlag sets a flag value.
func (s *Scenario) SetFlag(name string, value any) {
	s.Flags[name] = value
}

// GetFlag gets a flag value.
func (s *Scenario) GetFlag(name string) (any, bool) {
	v, ok := s.Flags[name]
	return v, ok
}

// HasTag checks if the scenario has a specific tag.
func (s *Scenario) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// IsRunnable returns true if the scenario can be executed.
func (s *Scenario) IsRunnable() bool {
	return !s.Abstract && len(s.Flow) > 0
}

// EffectiveTimeout returns the timeout to use for this scenario.
func (s *Scenario) EffectiveTimeout(defaultTimeout time.Duration) time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return defaultTimeout
}

// Clone creates a deep copy of the scenario.
func (s *Scenario) Clone() *Scenario {
	clone := &Scenario{
		ID:            core.NewScenarioID(),
		Name:          s.Name,
		Description:   s.Description,
		Timeout:       s.Timeout,
		Tags:          append([]string{}, s.Tags...),
		Flow:          append([]FlowItem{}, s.Flow...),
		TeardownFlow:  append([]FlowItem{}, s.TeardownFlow...),
		Flags:         make(map[string]any),
		Options:       append([]string{}, s.Options...),
		ChaosProfiles: append([]string{}, s.ChaosProfiles...),
		MockProfiles:  append([]string{}, s.MockProfiles...),
		SkipIf:        append([]Condition{}, s.SkipIf...),
		SkipUnless:    append([]Condition{}, s.SkipUnless...),
		Matrix:        make(map[string][]any),
		MatrixIndex:   make(map[string]any),
		Extends:       s.Extends,
		Abstract:      s.Abstract,
		SourceFile:    s.SourceFile,
		SourceLine:    s.SourceLine,
	}

	for k, v := range s.Flags {
		clone.Flags[k] = v
	}
	for k, v := range s.Matrix {
		clone.Matrix[k] = append([]any{}, v...)
	}
	for k, v := range s.MatrixIndex {
		clone.MatrixIndex[k] = v
	}

	return clone
}

// String returns a string representation of the scenario.
func (s *Scenario) String() string {
	return fmt.Sprintf("Scenario{ID: %s, Name: %s, Flow: %d items}", s.ID, s.Name, len(s.Flow))
}

// NewFlowItem creates a new flow item.
func NewFlowItem(componentType core.ComponentType, name string) FlowItem {
	return FlowItem{
		Type:   componentType,
		Name:   name,
		Params: make(map[string]any),
	}
}

// WithTimeout sets the timeout for the flow item.
func (f FlowItem) WithTimeout(timeout time.Duration) FlowItem {
	f.Timeout = timeout
	return f
}

// WithDependsOn sets dependencies for the flow item.
func (f FlowItem) WithDependsOn(deps ...string) FlowItem {
	f.DependsOn = deps
	return f
}

// WithParams sets parameters for the flow item.
func (f FlowItem) WithParams(params map[string]any) FlowItem {
	f.Params = params
	return f
}

// WithParam sets a single parameter.
func (f FlowItem) WithParam(key string, value any) FlowItem {
	if f.Params == nil {
		f.Params = make(map[string]any)
	}
	f.Params[key] = value
	return f
}

// AsParallel marks this flow item as part of a parallel execution block.
func (f FlowItem) AsParallel() FlowItem {
	f.Parallel = true
	return f
}
