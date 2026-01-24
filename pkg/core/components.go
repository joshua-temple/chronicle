package core

import (
	"fmt"
	"reflect"
	"time"
)

// ComponentType represents the type of a component.
type ComponentType string

const (
	// ComponentSetup prepares state before action (Given).
	ComponentSetup ComponentType = "setup"
	// ComponentTask executes an action (When).
	ComponentTask ComponentType = "task"
	// ComponentValidation asserts outcomes (Then).
	ComponentValidation ComponentType = "validation"
	// ComponentStep is a reusable bundle of 2+ Setup/Task/Validation.
	ComponentStep ComponentType = "step"
	// ComponentRollup is a higher-order composition of Steps/Rollups.
	ComponentRollup ComponentType = "rollup"
	// ComponentTeardown cleans up state (runs even on failure).
	ComponentTeardown ComponentType = "teardown"
)

// IsValid returns true if the ComponentType is a known type.
func (ct ComponentType) IsValid() bool {
	switch ct {
	case ComponentSetup, ComponentTask, ComponentValidation, ComponentStep, ComponentRollup, ComponentTeardown:
		return true
	}
	return false
}

// String returns the string representation of the ComponentType.
func (ct ComponentType) String() string {
	return string(ct)
}

// Dependency represents a data dependency for a component.
type Dependency struct {
	Key         string // Context key name (e.g., "user")
	Type        string // Type name (e.g., "User", "*User")
	Description string // Human-readable description
}

// String returns a string representation of the dependency.
func (d Dependency) String() string {
	if d.Description != "" {
		return fmt.Sprintf("%s:%s (%s)", d.Key, d.Type, d.Description)
	}
	return fmt.Sprintf("%s:%s", d.Key, d.Type)
}

// Component represents a discoverable test component.
type Component struct {
	// Core identification
	ID   ComponentID   // Unique identifier
	Name string        // Human-readable name
	Type ComponentType // setup, task, validation, step, rollup, teardown

	// Dependencies
	Produces []Dependency // What this component produces
	Requires []Dependency // What this component requires

	// Paired teardown
	Teardown string // Name of paired teardown component (for setup)

	// Metadata
	Description string   // Human-readable description
	Tags        []string // Tags for filtering/grouping
	Owner       string   // Team or person responsible
	Version     string   // Component version

	// Deprecation
	Deprecated string    // Deprecation message (empty if not deprecated)
	Sunset     time.Time // Date when component will be removed

	// Source location (populated by discovery)
	SourceFile string // File path where component is defined
	SourceLine int    // Line number in source file

	// Runtime binding (populated during execution setup)
	Func any // The actual function to execute
}

// NewComponent creates a new Component with the given name and type.
func NewComponent(name string, componentType ComponentType) *Component {
	return &Component{
		ID:       NewComponentID(name),
		Name:     name,
		Type:     componentType,
		Produces: make([]Dependency, 0),
		Requires: make([]Dependency, 0),
		Tags:     make([]string, 0),
	}
}

// WithProduces adds a produces dependency.
func (c *Component) WithProduces(key, typeName string) *Component {
	c.Produces = append(c.Produces, Dependency{Key: key, Type: typeName})
	return c
}

// WithProducesDesc adds a produces dependency with a description.
func (c *Component) WithProducesDesc(key, typeName, description string) *Component {
	c.Produces = append(c.Produces, Dependency{Key: key, Type: typeName, Description: description})
	return c
}

// WithRequires adds a requires dependency.
func (c *Component) WithRequires(key, typeName string) *Component {
	c.Requires = append(c.Requires, Dependency{Key: key, Type: typeName})
	return c
}

// WithRequiresDesc adds a requires dependency with a description.
func (c *Component) WithRequiresDesc(key, typeName, description string) *Component {
	c.Requires = append(c.Requires, Dependency{Key: key, Type: typeName, Description: description})
	return c
}

// WithTeardown sets the paired teardown component name.
func (c *Component) WithTeardown(name string) *Component {
	c.Teardown = name
	return c
}

// WithDescription sets the component description.
func (c *Component) WithDescription(desc string) *Component {
	c.Description = desc
	return c
}

// WithTags sets the component tags.
func (c *Component) WithTags(tags ...string) *Component {
	c.Tags = tags
	return c
}

// WithOwner sets the component owner.
func (c *Component) WithOwner(owner string) *Component {
	c.Owner = owner
	return c
}

// WithVersion sets the component version.
func (c *Component) WithVersion(version string) *Component {
	c.Version = version
	return c
}

// WithDeprecated marks the component as deprecated.
func (c *Component) WithDeprecated(message string, sunset time.Time) *Component {
	c.Deprecated = message
	c.Sunset = sunset
	return c
}

// WithSource sets the source location.
func (c *Component) WithSource(file string, line int) *Component {
	c.SourceFile = file
	c.SourceLine = line
	return c
}

// WithFunc binds the function to execute.
func (c *Component) WithFunc(fn any) *Component {
	c.Func = fn
	return c
}

// IsDeprecated returns true if the component is deprecated.
func (c *Component) IsDeprecated() bool {
	return c.Deprecated != ""
}

// IsSunset returns true if the sunset date has passed.
func (c *Component) IsSunset() bool {
	return !c.Sunset.IsZero() && time.Now().After(c.Sunset)
}

// HasTag returns true if the component has the given tag.
func (c *Component) HasTag(tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ProducesKey returns true if the component produces the given key.
func (c *Component) ProducesKey(key string) bool {
	for _, d := range c.Produces {
		if d.Key == key {
			return true
		}
	}
	return false
}

// RequiresKey returns true if the component requires the given key.
func (c *Component) RequiresKey(key string) bool {
	for _, d := range c.Requires {
		if d.Key == key {
			return true
		}
	}
	return false
}

// Validate checks if the component is properly configured.
func (c *Component) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("component name is required")
	}
	if !c.Type.IsValid() {
		return fmt.Errorf("invalid component type: %s", c.Type)
	}
	if c.Type == ComponentTeardown && len(c.Produces) > 0 {
		return fmt.Errorf("teardown components cannot produce values")
	}
	return nil
}

// Clone creates a deep copy of the component.
func (c *Component) Clone() *Component {
	clone := &Component{
		ID:          c.ID,
		Name:        c.Name,
		Type:        c.Type,
		Teardown:    c.Teardown,
		Description: c.Description,
		Owner:       c.Owner,
		Version:     c.Version,
		Deprecated:  c.Deprecated,
		Sunset:      c.Sunset,
		SourceFile:  c.SourceFile,
		SourceLine:  c.SourceLine,
		Func:        c.Func,
	}

	clone.Produces = make([]Dependency, len(c.Produces))
	copy(clone.Produces, c.Produces)

	clone.Requires = make([]Dependency, len(c.Requires))
	copy(clone.Requires, c.Requires)

	clone.Tags = make([]string, len(c.Tags))
	copy(clone.Tags, c.Tags)

	return clone
}

// Component function signatures.
// These define the expected function types for each component type.

// SetupFunc is the signature for setup components.
// Setup components prepare state and return only an error.
type SetupFunc func(ctx Context) error

// TaskFunc is the signature for task components.
// Task components execute actions and return a result and error.
type TaskFunc[T any] func(ctx Context) (T, error)

// ValidationFunc is the signature for validation components.
// Validation components receive the result from upstream and return an error.
type ValidationFunc func(ctx Context, result any) error

// StepFunc is the signature for step components.
// Step components are composites that return only an error.
type StepFunc func(ctx Context) error

// RollupFunc is the signature for rollup components.
// Rollup components are higher-order composites that return only an error.
type RollupFunc func(ctx Context) error

// TeardownFunc is the signature for teardown components.
// Teardown components clean up state and return only an error.
type TeardownFunc func(ctx Context) error

// Context is the interface that all component functions receive.
// This is a forward declaration - the actual implementation is in pkg/context.
type Context interface {
	// Get retrieves a value from the context state.
	Get(key string) (any, bool)
	// Set stores a value in the context state.
	Set(key string, value any)
}

// TypeInfo represents information about a discovered type.
type TypeInfo struct {
	Name        string       // Type name
	PackagePath string       // Full package path
	ReflectType reflect.Type // Reflection type
	IsAlias     bool         // True if this is a type alias
	AliasOf     string       // If alias, the original type name
	SourceFile  string       // File where type is defined
	SourceLine  int          // Line number in source file
}

// NewTypeInfo creates a new TypeInfo from a name and reflect.Type.
func NewTypeInfo(name string, rt reflect.Type) *TypeInfo {
	return &TypeInfo{
		Name:        name,
		ReflectType: rt,
	}
}

// String returns a string representation of the type info.
func (ti *TypeInfo) String() string {
	if ti.IsAlias {
		return fmt.Sprintf("%s (alias of %s)", ti.Name, ti.AliasOf)
	}
	return ti.Name
}
