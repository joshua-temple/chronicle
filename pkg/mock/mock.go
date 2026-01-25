// Package mock provides mocking capabilities for Chronicle tests.
package mock

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Registry manages mock definitions and their behavior.
type Registry struct {
	mocks map[string]*Mock
	mu    sync.RWMutex
}

// NewRegistry creates a new mock registry.
func NewRegistry() *Registry {
	return &Registry{
		mocks: make(map[string]*Mock),
	}
}

// Register adds a mock to the registry.
func (r *Registry) Register(mock *Mock) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mocks[mock.Name] = mock
	return r
}

// Get retrieves a mock by name.
func (r *Registry) Get(name string) (*Mock, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mock, ok := r.mocks[name]
	return mock, ok
}

// Remove deletes a mock from the registry.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.mocks, name)
}

// Clear removes all mocks from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mocks = make(map[string]*Mock)
}

// Names returns all registered mock names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.mocks))
	for name := range r.mocks {
		names = append(names, name)
	}
	return names
}

// Mock represents a mock object with configurable behavior.
type Mock struct {
	Name     string
	handlers map[string]*MethodHandler
	calls    []Call
	mu       sync.RWMutex
}

// NewMock creates a new mock.
func NewMock(name string) *Mock {
	return &Mock{
		Name:     name,
		handlers: make(map[string]*MethodHandler),
		calls:    make([]Call, 0),
	}
}

// On configures a method handler.
func (m *Mock) On(method string) *MethodHandler {
	m.mu.Lock()
	defer m.mu.Unlock()

	handler := &MethodHandler{
		mock:   m,
		method: method,
	}
	m.handlers[method] = handler
	return handler
}

// Call invokes a mocked method.
func (m *Mock) Call(ctx context.Context, method string, args ...any) (any, error) {
	m.mu.Lock()
	handler, ok := m.handlers[method]
	call := Call{
		Method: method,
		Args:   args,
	}
	m.calls = append(m.calls, call)
	m.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("mock %s: no handler for method %s", m.Name, method)
	}

	return handler.Execute(ctx, args...)
}

// Calls returns all recorded calls.
func (m *Mock) Calls() []Call {
	m.mu.RLock()
	defer m.mu.RUnlock()

	calls := make([]Call, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// CallsFor returns calls for a specific method.
func (m *Mock) CallsFor(method string) []Call {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var calls []Call
	for _, c := range m.calls {
		if c.Method == method {
			calls = append(calls, c)
		}
	}
	return calls
}

// Reset clears all recorded calls.
func (m *Mock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]Call, 0)

	// Reset handler state
	for _, h := range m.handlers {
		h.Reset()
	}
}

// AssertCalled verifies a method was called.
func (m *Mock) AssertCalled(method string) bool {
	calls := m.CallsFor(method)
	return len(calls) > 0
}

// AssertCalledTimes verifies a method was called a specific number of times.
func (m *Mock) AssertCalledTimes(method string, times int) bool {
	calls := m.CallsFor(method)
	return len(calls) == times
}

// AssertCalledWith verifies a method was called with specific arguments.
func (m *Mock) AssertCalledWith(method string, args ...any) bool {
	calls := m.CallsFor(method)
	for _, c := range calls {
		if matchArgs(c.Args, args) {
			return true
		}
	}
	return false
}

// Call represents a recorded method call.
type Call struct {
	Method string
	Args   []any
}

// MethodHandler configures the behavior of a mocked method.
type MethodHandler struct {
	mock       *Mock
	method     string
	returnVals []any
	returnErr  error
	callback   func(ctx context.Context, args ...any) (any, error)
	times      int // Number of times to return configured values
	called     int // Number of times called
	mu         sync.Mutex
}

// Return configures the return values.
func (h *MethodHandler) Return(vals ...any) *MethodHandler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.returnVals = vals
	return h
}

// ReturnError configures an error to return.
func (h *MethodHandler) ReturnError(err error) *MethodHandler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.returnErr = err
	return h
}

// Times limits how many times the configured response is used.
func (h *MethodHandler) Times(n int) *MethodHandler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.times = n
	return h
}

// Once is shorthand for Times(1).
func (h *MethodHandler) Once() *MethodHandler {
	return h.Times(1)
}

// Twice is shorthand for Times(2).
func (h *MethodHandler) Twice() *MethodHandler {
	return h.Times(2)
}

// Callback sets a custom callback function.
func (h *MethodHandler) Callback(fn func(ctx context.Context, args ...any) (any, error)) *MethodHandler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callback = fn
	return h
}

// Execute runs the handler and returns the configured response.
func (h *MethodHandler) Execute(ctx context.Context, args ...any) (any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.called++

	// Check if we've exceeded the configured times
	if h.times > 0 && h.called > h.times {
		return nil, fmt.Errorf("mock %s.%s: exceeded configured call count (%d)",
			h.mock.Name, h.method, h.times)
	}

	// If callback is set, use it
	if h.callback != nil {
		return h.callback(ctx, args...)
	}

	// Return configured error
	if h.returnErr != nil {
		return nil, h.returnErr
	}

	// Return configured values
	if len(h.returnVals) == 0 {
		return nil, nil
	}
	if len(h.returnVals) == 1 {
		return h.returnVals[0], nil
	}
	return h.returnVals, nil
}

// Reset resets the handler state.
func (h *MethodHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.called = 0
}

// matchArgs checks if two argument slices match.
func matchArgs(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !matchArg(a[i], b[i]) {
			return false
		}
	}
	return true
}

// matchArg checks if two arguments match.
func matchArg(a, b any) bool {
	// Check for Any matcher
	if _, ok := b.(AnyMatcher); ok {
		return true
	}

	// Check for custom matcher
	if matcher, ok := b.(Matcher); ok {
		return matcher.Match(a)
	}

	// Deep equality
	return reflect.DeepEqual(a, b)
}

// Matcher is an interface for custom argument matchers.
type Matcher interface {
	Match(actual any) bool
}

// AnyMatcher matches any value.
type AnyMatcher struct{}

// Match always returns true.
func (m AnyMatcher) Match(_ any) bool {
	return true
}

// Any returns a matcher that matches any value.
func Any() AnyMatcher {
	return AnyMatcher{}
}

// TypeMatcher matches values of a specific type.
type TypeMatcher struct {
	typ reflect.Type
}

// OfType returns a matcher that matches values of the given type.
func OfType[T any]() TypeMatcher {
	var t T
	return TypeMatcher{typ: reflect.TypeOf(t)}
}

// Match returns true if the actual value is of the expected type.
func (m TypeMatcher) Match(actual any) bool {
	if actual == nil {
		return false
	}
	return reflect.TypeOf(actual) == m.typ
}

// FuncMatcher uses a custom function to match.
type FuncMatcher struct {
	fn func(any) bool
}

// MatchFunc returns a matcher that uses a custom function.
func MatchFunc(fn func(any) bool) FuncMatcher {
	return FuncMatcher{fn: fn}
}

// Match returns the result of the custom function.
func (m FuncMatcher) Match(actual any) bool {
	return m.fn(actual)
}

// Common errors for mocking.
var (
	ErrMockNotFound   = errors.New("mock not found")
	ErrMethodNotMocked = errors.New("method not mocked")
)
