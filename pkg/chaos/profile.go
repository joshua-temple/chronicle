// Package chaos provides fault injection and chaos engineering capabilities for Chronicle.
package chaos

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Profile defines a chaos engineering profile that can be applied to scenarios.
type Profile struct {
	Name        string
	Description string
	Faults      []Fault
	Enabled     bool
	Selector    Selector
}

// ProfileOption configures a Profile.
type ProfileOption func(*Profile)

// NewProfile creates a new chaos profile.
func NewProfile(name string, opts ...ProfileOption) *Profile {
	p := &Profile{
		Name:    name,
		Enabled: true,
		Faults:  make([]Fault, 0),
		Selector: AllSelector{}, // Apply to all by default
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithDescription sets the profile description.
func WithDescription(desc string) ProfileOption {
	return func(p *Profile) {
		p.Description = desc
	}
}

// WithFaults adds faults to the profile.
func WithFaults(faults ...Fault) ProfileOption {
	return func(p *Profile) {
		p.Faults = append(p.Faults, faults...)
	}
}

// WithSelector sets the target selector for the profile.
func WithSelector(selector Selector) ProfileOption {
	return func(p *Profile) {
		p.Selector = selector
	}
}

// Disabled creates a disabled profile.
func Disabled() ProfileOption {
	return func(p *Profile) {
		p.Enabled = false
	}
}

// AddFault adds a fault to the profile.
func (p *Profile) AddFault(fault Fault) *Profile {
	p.Faults = append(p.Faults, fault)
	return p
}

// Enable enables the profile.
func (p *Profile) Enable() *Profile {
	p.Enabled = true
	return p
}

// Disable disables the profile.
func (p *Profile) Disable() *Profile {
	p.Enabled = false
	return p
}

// Apply applies the chaos profile to a target.
func (p *Profile) Apply(ctx context.Context, target Target) error {
	if !p.Enabled {
		return nil
	}

	if !p.Selector.Matches(target) {
		return nil
	}

	for _, fault := range p.Faults {
		if err := fault.Inject(ctx, target); err != nil {
			return fmt.Errorf("inject fault %s: %w", fault.Name(), err)
		}
	}

	return nil
}

// Selector determines which targets a profile applies to.
type Selector interface {
	Matches(target Target) bool
}

// AllSelector matches all targets.
type AllSelector struct{}

// Matches returns true for all targets.
func (s AllSelector) Matches(_ Target) bool {
	return true
}

// NameSelector matches targets by name pattern.
type NameSelector struct {
	Pattern string
	Prefix  bool
	Suffix  bool
}

// Matches returns true if the target name matches the pattern.
func (s NameSelector) Matches(target Target) bool {
	name := target.Name()

	if s.Prefix && s.Suffix {
		return contains(name, s.Pattern)
	}
	if s.Prefix {
		return len(name) >= len(s.Pattern) && name[:len(s.Pattern)] == s.Pattern
	}
	if s.Suffix {
		return len(name) >= len(s.Pattern) && name[len(name)-len(s.Pattern):] == s.Pattern
	}
	return name == s.Pattern
}

// TypeSelector matches targets by type.
type TypeSelector struct {
	Types []string
}

// Matches returns true if the target type is in the list.
func (s TypeSelector) Matches(target Target) bool {
	targetType := target.Type()
	for _, t := range s.Types {
		if t == targetType {
			return true
		}
	}
	return false
}

// TagSelector matches targets by tags.
type TagSelector struct {
	Tags    []string
	MatchAll bool
}

// Matches returns true if the target has the required tags.
func (s TagSelector) Matches(target Target) bool {
	targetTags := target.Tags()

	if s.MatchAll {
		for _, tag := range s.Tags {
			found := false
			for _, t := range targetTags {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	// Match any
	for _, tag := range s.Tags {
		for _, t := range targetTags {
			if t == tag {
				return true
			}
		}
	}
	return false
}

// ProbabilitySelector randomly matches based on probability.
type ProbabilitySelector struct {
	Probability float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// NewProbabilitySelector creates a new probability selector.
func NewProbabilitySelector(probability float64) *ProbabilitySelector {
	return &ProbabilitySelector{
		Probability: probability,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Matches returns true based on the configured probability.
func (s *ProbabilitySelector) Matches(_ Target) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rng.Float64() < s.Probability
}

// CompositeSelector combines multiple selectors.
type CompositeSelector struct {
	Selectors []Selector
	Mode      CompositeMode
}

// CompositeMode defines how composite selectors combine results.
type CompositeMode int

const (
	// ModeAnd requires all selectors to match.
	ModeAnd CompositeMode = iota
	// ModeOr requires at least one selector to match.
	ModeOr
)

// Matches returns true based on the composite mode.
func (s CompositeSelector) Matches(target Target) bool {
	if len(s.Selectors) == 0 {
		return true
	}

	switch s.Mode {
	case ModeAnd:
		for _, sel := range s.Selectors {
			if !sel.Matches(target) {
				return false
			}
		}
		return true
	case ModeOr:
		for _, sel := range s.Selectors {
			if sel.Matches(target) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// Target represents something that chaos can be applied to.
type Target interface {
	Name() string
	Type() string
	Tags() []string
}

// SimpleTarget is a basic target implementation.
type SimpleTarget struct {
	name    string
	typ     string
	tags    []string
}

// NewSimpleTarget creates a new simple target.
func NewSimpleTarget(name, typ string, tags ...string) *SimpleTarget {
	return &SimpleTarget{
		name: name,
		typ:  typ,
		tags: tags,
	}
}

// Name returns the target name.
func (t *SimpleTarget) Name() string {
	return t.name
}

// Type returns the target type.
func (t *SimpleTarget) Type() string {
	return t.typ
}

// Tags returns the target tags.
func (t *SimpleTarget) Tags() []string {
	return t.tags
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Common errors.
var (
	ErrChaosInjected = errors.New("chaos: injected error")
	ErrLatencyExceeded = errors.New("chaos: latency exceeded timeout")
)
