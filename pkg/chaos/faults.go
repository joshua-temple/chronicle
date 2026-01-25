package chaos

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Fault represents a chaos fault that can be injected.
type Fault interface {
	Name() string
	Inject(ctx context.Context, target Target) error
}

// LatencyFault injects latency into operations.
type LatencyFault struct {
	name        string
	min         time.Duration
	max         time.Duration
	probability float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// LatencyOption configures a LatencyFault.
type LatencyOption func(*LatencyFault)

// NewLatencyFault creates a new latency fault.
func NewLatencyFault(min, max time.Duration, opts ...LatencyOption) *LatencyFault {
	f := &LatencyFault{
		name:        "latency",
		min:         min,
		max:         max,
		probability: 1.0,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithLatencyName sets the fault name.
func WithLatencyName(name string) LatencyOption {
	return func(f *LatencyFault) {
		f.name = name
	}
}

// WithLatencyProbability sets the injection probability.
func WithLatencyProbability(prob float64) LatencyOption {
	return func(f *LatencyFault) {
		f.probability = prob
	}
}

// Name returns the fault name.
func (f *LatencyFault) Name() string {
	return f.name
}

// Inject adds latency to the operation.
func (f *LatencyFault) Inject(ctx context.Context, _ Target) error {
	f.mu.Lock()
	shouldInject := f.rng.Float64() < f.probability
	var delay time.Duration
	if shouldInject {
		delay = f.min + time.Duration(f.rng.Int63n(int64(f.max-f.min+1)))
	}
	f.mu.Unlock()

	if !shouldInject {
		return nil
	}

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: context cancelled during latency injection", ctx.Err())
	}
}

// ErrorFault injects errors into operations.
type ErrorFault struct {
	name        string
	err         error
	probability float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// ErrorOption configures an ErrorFault.
type ErrorOption func(*ErrorFault)

// NewErrorFault creates a new error fault.
func NewErrorFault(err error, probability float64, opts ...ErrorOption) *ErrorFault {
	f := &ErrorFault{
		name:        "error",
		err:         err,
		probability: probability,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithErrorName sets the fault name.
func WithErrorName(name string) ErrorOption {
	return func(f *ErrorFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *ErrorFault) Name() string {
	return f.name
}

// Inject potentially returns an error.
func (f *ErrorFault) Inject(_ context.Context, _ Target) error {
	f.mu.Lock()
	shouldInject := f.rng.Float64() < f.probability
	f.mu.Unlock()

	if shouldInject {
		return f.err
	}
	return nil
}

// TimeoutFault causes operations to timeout.
type TimeoutFault struct {
	name        string
	probability float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// TimeoutOption configures a TimeoutFault.
type TimeoutOption func(*TimeoutFault)

// NewTimeoutFault creates a new timeout fault.
func NewTimeoutFault(probability float64, opts ...TimeoutOption) *TimeoutFault {
	f := &TimeoutFault{
		name:        "timeout",
		probability: probability,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithTimeoutName sets the fault name.
func WithTimeoutName(name string) TimeoutOption {
	return func(f *TimeoutFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *TimeoutFault) Name() string {
	return f.name
}

// Inject blocks until context is cancelled.
func (f *TimeoutFault) Inject(ctx context.Context, _ Target) error {
	f.mu.Lock()
	shouldInject := f.rng.Float64() < f.probability
	f.mu.Unlock()

	if !shouldInject {
		return nil
	}

	<-ctx.Done()
	return fmt.Errorf("%w: %v", ErrLatencyExceeded, ctx.Err())
}

// PanicFault causes a panic.
type PanicFault struct {
	name        string
	message     string
	probability float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// PanicOption configures a PanicFault.
type PanicOption func(*PanicFault)

// NewPanicFault creates a new panic fault.
func NewPanicFault(message string, probability float64, opts ...PanicOption) *PanicFault {
	f := &PanicFault{
		name:        "panic",
		message:     message,
		probability: probability,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithPanicName sets the fault name.
func WithPanicName(name string) PanicOption {
	return func(f *PanicFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *PanicFault) Name() string {
	return f.name
}

// Inject potentially panics.
func (f *PanicFault) Inject(_ context.Context, _ Target) error {
	f.mu.Lock()
	shouldInject := f.rng.Float64() < f.probability
	f.mu.Unlock()

	if shouldInject {
		panic(f.message)
	}
	return nil
}

// ResourceExhaustionFault simulates resource exhaustion.
type ResourceExhaustionFault struct {
	name         string
	resourceType string
	probability  float64
	rng          *rand.Rand
	mu           sync.Mutex
}

// ResourceOption configures a ResourceExhaustionFault.
type ResourceOption func(*ResourceExhaustionFault)

// NewResourceExhaustionFault creates a new resource exhaustion fault.
func NewResourceExhaustionFault(resourceType string, probability float64, opts ...ResourceOption) *ResourceExhaustionFault {
	f := &ResourceExhaustionFault{
		name:         "resource_exhaustion",
		resourceType: resourceType,
		probability:  probability,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithResourceName sets the fault name.
func WithResourceName(name string) ResourceOption {
	return func(f *ResourceExhaustionFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *ResourceExhaustionFault) Name() string {
	return f.name
}

// Inject returns a resource exhaustion error.
func (f *ResourceExhaustionFault) Inject(_ context.Context, _ Target) error {
	f.mu.Lock()
	shouldInject := f.rng.Float64() < f.probability
	f.mu.Unlock()

	if shouldInject {
		return fmt.Errorf("resource exhaustion: %s unavailable", f.resourceType)
	}
	return nil
}

// NetworkPartitionFault simulates network partitions.
type NetworkPartitionFault struct {
	name        string
	targets     []string // Target names or patterns to partition
	probability float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// NetworkPartitionOption configures a NetworkPartitionFault.
type NetworkPartitionOption func(*NetworkPartitionFault)

// NewNetworkPartitionFault creates a new network partition fault.
func NewNetworkPartitionFault(targets []string, probability float64, opts ...NetworkPartitionOption) *NetworkPartitionFault {
	f := &NetworkPartitionFault{
		name:        "network_partition",
		targets:     targets,
		probability: probability,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithNetworkPartitionName sets the fault name.
func WithNetworkPartitionName(name string) NetworkPartitionOption {
	return func(f *NetworkPartitionFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *NetworkPartitionFault) Name() string {
	return f.name
}

// Inject simulates a network partition.
func (f *NetworkPartitionFault) Inject(_ context.Context, target Target) error {
	f.mu.Lock()
	shouldInject := f.rng.Float64() < f.probability
	f.mu.Unlock()

	if !shouldInject {
		return nil
	}

	// Check if target is in the partition list
	for _, t := range f.targets {
		if t == target.Name() || t == "*" {
			return errors.New("network partition: connection refused")
		}
	}

	return nil
}

// CorruptionFault simulates data corruption.
type CorruptionFault struct {
	name        string
	dataType    string
	probability float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// CorruptionOption configures a CorruptionFault.
type CorruptionOption func(*CorruptionFault)

// NewCorruptionFault creates a new data corruption fault.
func NewCorruptionFault(dataType string, probability float64, opts ...CorruptionOption) *CorruptionFault {
	f := &CorruptionFault{
		name:        "corruption",
		dataType:    dataType,
		probability: probability,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithCorruptionName sets the fault name.
func WithCorruptionName(name string) CorruptionOption {
	return func(f *CorruptionFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *CorruptionFault) Name() string {
	return f.name
}

// Inject returns a data corruption error.
func (f *CorruptionFault) Inject(_ context.Context, _ Target) error {
	f.mu.Lock()
	shouldInject := f.rng.Float64() < f.probability
	f.mu.Unlock()

	if shouldInject {
		return fmt.Errorf("data corruption: %s checksum mismatch", f.dataType)
	}
	return nil
}

// CompositeFault combines multiple faults.
type CompositeFault struct {
	name   string
	faults []Fault
	mode   CompositeMode
}

// CompositeFaultOption configures a CompositeFault.
type CompositeFaultOption func(*CompositeFault)

// NewCompositeFault creates a new composite fault.
func NewCompositeFault(faults []Fault, mode CompositeMode, opts ...CompositeFaultOption) *CompositeFault {
	f := &CompositeFault{
		name:   "composite",
		faults: faults,
		mode:   mode,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithCompositeName sets the fault name.
func WithCompositeName(name string) CompositeFaultOption {
	return func(f *CompositeFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *CompositeFault) Name() string {
	return f.name
}

// Inject applies all faults based on the mode.
func (f *CompositeFault) Inject(ctx context.Context, target Target) error {
	switch f.mode {
	case ModeAnd:
		// All faults are applied
		for _, fault := range f.faults {
			if err := fault.Inject(ctx, target); err != nil {
				return err
			}
		}
		return nil
	case ModeOr:
		// Only the first fault to produce an error is returned
		for _, fault := range f.faults {
			if err := fault.Inject(ctx, target); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

// SequentialFault applies faults in sequence with delays.
type SequentialFault struct {
	name       string
	faults     []Fault
	delayBetween time.Duration
}

// SequentialOption configures a SequentialFault.
type SequentialOption func(*SequentialFault)

// NewSequentialFault creates a new sequential fault.
func NewSequentialFault(faults []Fault, delay time.Duration, opts ...SequentialOption) *SequentialFault {
	f := &SequentialFault{
		name:         "sequential",
		faults:       faults,
		delayBetween: delay,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithSequentialName sets the fault name.
func WithSequentialName(name string) SequentialOption {
	return func(f *SequentialFault) {
		f.name = name
	}
}

// Name returns the fault name.
func (f *SequentialFault) Name() string {
	return f.name
}

// Inject applies faults sequentially with delays.
func (f *SequentialFault) Inject(ctx context.Context, target Target) error {
	for i, fault := range f.faults {
		if err := fault.Inject(ctx, target); err != nil {
			return err
		}

		if i < len(f.faults)-1 && f.delayBetween > 0 {
			select {
			case <-time.After(f.delayBetween):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
