package context

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/infrastructure"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarn:
		return "WARN"
	case LogError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// NarrativeLevel represents the verbosity level for narrative entries.
type NarrativeLevel int

const (
	NarrativeSummary NarrativeLevel = iota // High-level actions only
	NarrativeDetail                        // Include important details
	NarrativeVerbose                       // Include all details
)

// String returns the string representation of the narrative level.
func (l NarrativeLevel) String() string {
	switch l {
	case NarrativeSummary:
		return "summary"
	case NarrativeDetail:
		return "detail"
	case NarrativeVerbose:
		return "verbose"
	default:
		return "unknown"
	}
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Component string
	Message   string
	Args      []any
}

// NarrativeEntry represents a single narrative entry.
type NarrativeEntry struct {
	Timestamp time.Time
	Level     NarrativeLevel
	Component string
	SpanID    string
	Action    string
	Details   map[string]any
	Duration  time.Duration
}

// Context is the interface that components receive during execution.
// Context is NOT thread-safe. Each goroutine must use a Child() context.
type Context interface {
	context.Context

	// State management
	Get(key string) (any, bool)
	Set(key string, value any)
	SetLocal(key string, value any)

	// Infrastructure clients
	Client(name string) (any, error)
	RegisterClient(name string, client any)

	// Endpoint returns infrastructure endpoint info by name.
	Endpoint(name string) (infrastructure.Endpoint, bool)

	// Flags and parameters
	Flag(name string) any
	Param(name string) any

	// Tracing
	Trace() *core.TraceContext
	WithSpan(name string) Context

	// Child contexts (for goroutines)
	Child(name string) Context

	// Logging
	Log(level LogLevel, msg string, args ...any)

	// Narrative
	Narrate(level NarrativeLevel, action string, details map[string]any)

	// Teardown context info
	FailureReason() error
	PartialResults() map[string]any

	// Component info
	ComponentName() string
	SetComponentName(name string)

	// Logs and narrative access
	Logs() []LogEntry
	Narrative() []NarrativeEntry
}

// contextImpl is the default implementation of Context.
type contextImpl struct {
	context.Context

	name             string
	parent           *contextImpl
	state            map[string]any
	flags            map[string]any
	params           map[string]any
	clients          map[string]any
	trace            *core.TraceContext
	logs             []LogEntry
	narrative        []NarrativeEntry
	failure          error
	partial          map[string]any
	startTime        time.Time
	sizeLimit        int
	endpointRegistry *infrastructure.EndpointRegistry

	// Client provider function (injected by execution engine)
	clientProvider func(name string) (any, error)
}

// DefaultSizeLimit is the maximum size (in bytes) of stored values.
const DefaultSizeLimit = 10 * 1024 * 1024 // 10MB

// ContextOption configures a context.
type ContextOption func(*contextImpl)

// WithTrace sets the trace context.
func WithTrace(tc *core.TraceContext) ContextOption {
	return func(c *contextImpl) {
		c.trace = tc
	}
}

// WithFlags sets the flags map.
func WithFlags(flags map[string]any) ContextOption {
	return func(c *contextImpl) {
		c.flags = flags
	}
}

// WithParams sets the params map.
func WithParams(params map[string]any) ContextOption {
	return func(c *contextImpl) {
		c.params = params
	}
}

// WithSizeLimit sets the maximum size for stored values.
func WithSizeLimit(limit int) ContextOption {
	return func(c *contextImpl) {
		c.sizeLimit = limit
	}
}

// WithClientProvider sets the function to resolve infrastructure clients.
func WithClientProvider(provider func(name string) (any, error)) ContextOption {
	return func(c *contextImpl) {
		c.clientProvider = provider
	}
}

// WithEndpointRegistry sets the endpoint registry for infrastructure discovery.
func WithEndpointRegistry(registry *infrastructure.EndpointRegistry) ContextOption {
	return func(c *contextImpl) {
		c.endpointRegistry = registry
	}
}

// New creates a new Context with the given options.
func New(ctx context.Context, opts ...ContextOption) Context {
	c := &contextImpl{
		Context:   ctx,
		state:     make(map[string]any),
		flags:     make(map[string]any),
		params:    make(map[string]any),
		clients:   make(map[string]any),
		logs:      make([]LogEntry, 0),
		narrative: make([]NarrativeEntry, 0),
		partial:   make(map[string]any),
		startTime: time.Now(),
		sizeLimit: DefaultSizeLimit,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.trace == nil {
		c.trace = core.NewTraceContext()
	}

	return c
}

// Get retrieves a value from the context state.
func (c *contextImpl) Get(key string) (any, bool) {
	v, ok := c.state[key]
	if !ok && c.parent != nil {
		return c.parent.Get(key)
	}
	return v, ok
}

// Set stores a value in the context state.
// Values are propagated to the parent context so sibling contexts can access them.
func (c *contextImpl) Set(key string, value any) {
	c.state[key] = value
	// Also set on parent so sibling contexts can access the value
	if c.parent != nil {
		c.parent.Set(key, value)
	}
}

// SetLocal stores a value only in this context's local state.
// Unlike Set, this does not propagate to parent contexts.
func (c *contextImpl) SetLocal(key string, value any) {
	c.state[key] = value
}

// Client retrieves an infrastructure client by name.
func (c *contextImpl) Client(name string) (any, error) {
	// Check cached clients first
	if client, ok := c.clients[name]; ok {
		return client, nil
	}

	// Check parent
	if c.parent != nil {
		return c.parent.Client(name)
	}

	// Use provider if available
	if c.clientProvider != nil {
		client, err := c.clientProvider(name)
		if err != nil {
			return nil, err
		}
		c.clients[name] = client
		return client, nil
	}

	return nil, fmt.Errorf("client not found: %s", name)
}

// Endpoint returns infrastructure endpoint info by name.
func (c *contextImpl) Endpoint(name string) (infrastructure.Endpoint, bool) {
	if c.endpointRegistry == nil {
		if c.parent != nil {
			return c.parent.Endpoint(name)
		}
		return infrastructure.Endpoint{}, false
	}
	return c.endpointRegistry.Get(name)
}

// Flag retrieves a flag value by name.
func (c *contextImpl) Flag(name string) any {
	if v, ok := c.flags[name]; ok {
		return v
	}
	if c.parent != nil {
		return c.parent.Flag(name)
	}
	return nil
}

// Param retrieves a parameter value by name.
func (c *contextImpl) Param(name string) any {
	if v, ok := c.params[name]; ok {
		return v
	}
	if c.parent != nil {
		return c.parent.Param(name)
	}
	return nil
}

// Trace returns the current trace context.
func (c *contextImpl) Trace() *core.TraceContext {
	return c.trace
}

// WithSpan creates a child context with a new span.
func (c *contextImpl) WithSpan(name string) Context {
	child := c.createChild(name)
	child.trace = c.trace.NewSpan(name)
	return child
}

// Child creates a child context for use in a goroutine.
// The child has its own state map but can read from parent.
func (c *contextImpl) Child(name string) Context {
	return c.createChild(name)
}

func (c *contextImpl) createChild(name string) *contextImpl {
	child := &contextImpl{
		Context:          c.Context,
		name:             name,
		parent:           c,
		state:            make(map[string]any),
		flags:            c.flags,    // Share flags
		params:           c.params,   // Share params
		clients:          c.clients,  // Share clients cache
		trace:            c.trace,    // Will be overridden by WithSpan
		logs:             c.logs,     // Share logs slice
		narrative:        c.narrative,
		partial:          c.partial,
		startTime:        time.Now(),
		sizeLimit:        c.sizeLimit,
		endpointRegistry: c.endpointRegistry,
		clientProvider:   c.clientProvider,
	}
	return child
}

// Log records a log entry.
func (c *contextImpl) Log(level LogLevel, msg string, args ...any) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: c.name,
		Message:   msg,
		Args:      args,
	}
	c.logs = append(c.logs, entry)
}

// Narrate records a narrative entry.
func (c *contextImpl) Narrate(level NarrativeLevel, action string, details map[string]any) {
	entry := NarrativeEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: c.name,
		SpanID:    string(c.trace.SpanID),
		Action:    action,
		Details:   details,
		Duration:  time.Since(c.startTime),
	}
	c.narrative = append(c.narrative, entry)
}

// FailureReason returns the error that caused the failure (for teardown).
func (c *contextImpl) FailureReason() error {
	return c.failure
}

// PartialResults returns results collected before failure (for teardown).
func (c *contextImpl) PartialResults() map[string]any {
	return c.partial
}

// ComponentName returns the current component name.
func (c *contextImpl) ComponentName() string {
	return c.name
}

// SetComponentName sets the current component name.
func (c *contextImpl) SetComponentName(name string) {
	c.name = name
}

// Logs returns all log entries.
func (c *contextImpl) Logs() []LogEntry {
	return c.logs
}

// Narrative returns all narrative entries.
func (c *contextImpl) Narrative() []NarrativeEntry {
	return c.narrative
}

// SetFailure sets the failure reason (called by execution engine).
func (c *contextImpl) SetFailure(err error) {
	c.failure = err
}

// SetPartialResults sets the partial results (called by execution engine).
func (c *contextImpl) SetPartialResults(results map[string]any) {
	c.partial = results
}

// RegisterClient registers a custom client that can be retrieved via Client().
// Registered clients take precedence over clients from the provider.
func (c *contextImpl) RegisterClient(name string, client any) {
	c.clients[name] = client
}

// Generic accessor functions.

// Get retrieves a typed value from the context.
// Returns the zero value if the key doesn't exist or type doesn't match.
func Get[T any](ctx Context, key string) T {
	v, ok := ctx.Get(key)
	if !ok {
		var zero T
		return zero
	}
	typed, ok := v.(T)
	if !ok {
		var zero T
		return zero
	}
	return typed
}

// GetOK retrieves a typed value from the context with existence check.
func GetOK[T any](ctx Context, key string) (T, bool) {
	v, ok := ctx.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typed, true
}

// Set stores a typed value in the context.
func Set[T any](ctx Context, key string, value T) {
	ctx.Set(key, value)
}

// MustGet retrieves a typed value from the context.
// Panics if the key doesn't exist or type doesn't match.
func MustGet[T any](ctx Context, key string) T {
	v, ok := ctx.Get(key)
	if !ok {
		panic(fmt.Sprintf("context key not found: %s", key))
	}
	typed, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("context key %s: type mismatch, expected %T, got %T", key, *new(T), v))
	}
	return typed
}

// ThreadSafeContext wraps a Context with mutex protection.
// Use this when you need to share state between goroutines.
type ThreadSafeContext struct {
	Context
	mu sync.RWMutex
}

// NewThreadSafe wraps a Context with mutex protection.
func NewThreadSafe(ctx Context) *ThreadSafeContext {
	return &ThreadSafeContext{Context: ctx}
}

// Get retrieves a value from the context state (thread-safe).
func (c *ThreadSafeContext) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Context.Get(key)
}

// Set stores a value in the context state (thread-safe).
func (c *ThreadSafeContext) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Context.Set(key, value)
}

// SetLocal stores a value only in this context's local state (thread-safe).
func (c *ThreadSafeContext) SetLocal(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Context.SetLocal(key, value)
}
