package chaos

import (
	"context"
	"sync"
	"time"

	chronicleCtx "github.com/joshua-temple/chronicle/pkg/context"
	"github.com/joshua-temple/chronicle/pkg/middleware"
)

// Injector manages chaos injection for Chronicle scenarios.
type Injector struct {
	profiles []*Profile
	enabled  bool
	mu       sync.RWMutex
}

// NewInjector creates a new chaos injector.
func NewInjector() *Injector {
	return &Injector{
		profiles: make([]*Profile, 0),
		enabled:  true,
	}
}

// AddProfile adds a chaos profile to the injector.
func (i *Injector) AddProfile(profile *Profile) *Injector {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.profiles = append(i.profiles, profile)
	return i
}

// RemoveProfile removes a profile by name.
func (i *Injector) RemoveProfile(name string) *Injector {
	i.mu.Lock()
	defer i.mu.Unlock()

	profiles := make([]*Profile, 0, len(i.profiles))
	for _, p := range i.profiles {
		if p.Name != name {
			profiles = append(profiles, p)
		}
	}
	i.profiles = profiles
	return i
}

// Enable enables the injector.
func (i *Injector) Enable() *Injector {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.enabled = true
	return i
}

// Disable disables the injector.
func (i *Injector) Disable() *Injector {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.enabled = false
	return i
}

// IsEnabled returns whether the injector is enabled.
func (i *Injector) IsEnabled() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.enabled
}

// Profiles returns a copy of the registered profiles.
func (i *Injector) Profiles() []*Profile {
	i.mu.RLock()
	defer i.mu.RUnlock()

	profiles := make([]*Profile, len(i.profiles))
	copy(profiles, i.profiles)
	return profiles
}

// Inject applies all enabled profiles to the target.
func (i *Injector) Inject(ctx context.Context, target Target) error {
	i.mu.RLock()
	enabled := i.enabled
	profiles := i.profiles
	i.mu.RUnlock()

	if !enabled {
		return nil
	}

	for _, profile := range profiles {
		if err := profile.Apply(ctx, target); err != nil {
			return err
		}
	}

	return nil
}

// Middleware returns a Chronicle middleware that injects chaos.
func (i *Injector) Middleware() middleware.Middleware {
	return func(next middleware.Runner) middleware.Runner {
		return func(ctx chronicleCtx.Context) error {
			// Create a target from the context
			target := targetFromContext(ctx)

			// Inject chaos before execution
			if err := i.Inject(ctx, target); err != nil {
				return err
			}

			// Continue with execution
			return next(ctx)
		}
	}
}

// BeforeMiddleware returns middleware that runs chaos before component execution.
func (i *Injector) BeforeMiddleware() middleware.Middleware {
	return i.Middleware()
}

// AfterMiddleware returns middleware that runs chaos after component execution.
func (i *Injector) AfterMiddleware() middleware.Middleware {
	return func(next middleware.Runner) middleware.Runner {
		return func(ctx chronicleCtx.Context) error {
			// Execute the component first
			err := next(ctx)
			if err != nil {
				return err
			}

			// Inject chaos after successful execution
			target := targetFromContext(ctx)
			return i.Inject(ctx, target)
		}
	}
}

// contextTarget adapts a Chronicle context to a chaos Target.
type contextTarget struct {
	ctx chronicleCtx.Context
}

func targetFromContext(ctx chronicleCtx.Context) *contextTarget {
	return &contextTarget{ctx: ctx}
}

// Name returns the component name from context.
func (t *contextTarget) Name() string {
	return t.ctx.ComponentName()
}

// Type returns the component type as a string.
func (t *contextTarget) Type() string {
	// Try to get type from context metadata if available
	return "component"
}

// Tags returns empty tags for now.
func (t *contextTarget) Tags() []string {
	return nil
}

// ChaosConfig configures chaos injection from configuration.
type ChaosConfig struct {
	Enabled  bool            `yaml:"enabled"`
	Profiles []ProfileConfig `yaml:"profiles"`
}

// ProfileConfig configures a chaos profile from YAML.
type ProfileConfig struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description,omitempty"`
	Enabled     bool          `yaml:"enabled"`
	Selector    SelectorConfig `yaml:"selector,omitempty"`
	Faults      []FaultConfig  `yaml:"faults"`
}

// SelectorConfig configures a selector from YAML.
type SelectorConfig struct {
	Type        string   `yaml:"type"` // all, name, type, tag, probability
	Pattern     string   `yaml:"pattern,omitempty"`
	Types       []string `yaml:"types,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	MatchAll    bool     `yaml:"match_all,omitempty"`
	Probability float64  `yaml:"probability,omitempty"`
}

// FaultConfig configures a fault from YAML.
type FaultConfig struct {
	Type        string  `yaml:"type"` // latency, error, timeout, panic, resource, partition, corruption
	Probability float64 `yaml:"probability"`

	// Latency-specific
	MinLatency string `yaml:"min_latency,omitempty"`
	MaxLatency string `yaml:"max_latency,omitempty"`

	// Error-specific
	ErrorMessage string `yaml:"error_message,omitempty"`

	// Panic-specific
	PanicMessage string `yaml:"panic_message,omitempty"`

	// Resource-specific
	ResourceType string `yaml:"resource_type,omitempty"`

	// Partition-specific
	Targets []string `yaml:"targets,omitempty"`

	// Corruption-specific
	DataType string `yaml:"data_type,omitempty"`
}

// NewInjectorFromConfig creates an injector from configuration.
func NewInjectorFromConfig(cfg ChaosConfig) (*Injector, error) {
	injector := NewInjector()

	if !cfg.Enabled {
		injector.Disable()
	}

	for _, pcfg := range cfg.Profiles {
		profile, err := newProfileFromConfig(pcfg)
		if err != nil {
			return nil, err
		}
		injector.AddProfile(profile)
	}

	return injector, nil
}

func newProfileFromConfig(cfg ProfileConfig) (*Profile, error) {
	profile := NewProfile(cfg.Name,
		WithDescription(cfg.Description),
	)

	if !cfg.Enabled {
		profile.Disable()
	}

	// Configure selector
	selector, err := newSelectorFromConfig(cfg.Selector)
	if err != nil {
		return nil, err
	}
	profile.Selector = selector

	// Configure faults
	for _, fcfg := range cfg.Faults {
		fault, err := newFaultFromConfig(fcfg)
		if err != nil {
			return nil, err
		}
		profile.AddFault(fault)
	}

	return profile, nil
}

func newSelectorFromConfig(cfg SelectorConfig) (Selector, error) {
	switch cfg.Type {
	case "", "all":
		return AllSelector{}, nil
	case "name":
		return NameSelector{
			Pattern: cfg.Pattern,
			Prefix:  len(cfg.Pattern) > 0 && cfg.Pattern[len(cfg.Pattern)-1] == '*',
			Suffix:  len(cfg.Pattern) > 0 && cfg.Pattern[0] == '*',
		}, nil
	case "type":
		return TypeSelector{Types: cfg.Types}, nil
	case "tag":
		return TagSelector{Tags: cfg.Tags, MatchAll: cfg.MatchAll}, nil
	case "probability":
		return NewProbabilitySelector(cfg.Probability), nil
	default:
		return AllSelector{}, nil
	}
}

func newFaultFromConfig(cfg FaultConfig) (Fault, error) {
	switch cfg.Type {
	case "latency":
		minDur, _ := parseDuration(cfg.MinLatency)
		maxDur, _ := parseDuration(cfg.MaxLatency)
		return NewLatencyFault(minDur, maxDur, WithLatencyProbability(cfg.Probability)), nil
	case "error":
		return NewErrorFault(
			&InjectedError{Message: cfg.ErrorMessage},
			cfg.Probability,
		), nil
	case "timeout":
		return NewTimeoutFault(cfg.Probability), nil
	case "panic":
		return NewPanicFault(cfg.PanicMessage, cfg.Probability), nil
	case "resource":
		return NewResourceExhaustionFault(cfg.ResourceType, cfg.Probability), nil
	case "partition":
		return NewNetworkPartitionFault(cfg.Targets, cfg.Probability), nil
	case "corruption":
		return NewCorruptionFault(cfg.DataType, cfg.Probability), nil
	default:
		return NewErrorFault(ErrChaosInjected, cfg.Probability), nil
	}
}

// parseDuration parses a duration string with a default of 0.
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}

// InjectedError is an error that was injected by chaos.
type InjectedError struct {
	Message string
}

// Error returns the error message.
func (e *InjectedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "chaos: injected error"
}

// Is checks if the error is a chaos injected error.
func (e *InjectedError) Is(target error) bool {
	_, ok := target.(*InjectedError)
	return ok
}
