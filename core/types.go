package core

import (
	"context"
	"testing"
	"time"
)

// SetupFunc can represent the action of seeding data into the system
// This being the "Given I have this" part of the test
type SetupFunc func(context.Context, *TestContext) error

// TaskFunc can represent the action of interacting with the API to initiate a process
// This being the "When I do this" part of the test
// This can be seen as either a read or write operation
type TaskFunc[T any] func(context.Context, *TestContext) (T, error)

// ExpectFunc can represent the action of validating the results of either the TaskFunc
// This being the "Then I should see this" part of the test
type ExpectFunc[T any] func(context.Context, *TestContext, T) error

// AtomicFunc is the combination of TaskFunc and ExpectFunc
// This being the "When I do this, then I should see this" part of the test
// This allows caller to fold the TaskFunc and ExpectFunc into a single function
// The output of TaskFunc is inherently passed to ExpectFunc
// AtomicFn inherently suggests that the two functions are related
type AtomicFunc[T any] func(context.Context, *TestContext, TaskFunc[T], ExpectFunc[T]) (T, error)

// StepFunc is a function that represents a single step in a test
// This being the "Given I have this and when I do this, then I should see this" part of the test
type StepFunc[T any] func(context.Context, *TestContext) (T, error)

// RollupFunc combines multiple steps into a higher-level test scenario
type RollupFunc func(context.Context, *TestContext) error

// TestRunner is the function type that executes a test
type TestRunner func(context.Context, *TestContext) error

// TestOption is a function that can be used to configure a test
type TestOption func(*TestContext) *TestContext

// MiddlewareFunc is a function that wraps a TestRunner
type MiddlewareFunc func(TestRunner) TestRunner

// LogLevel represents the severity of log messages
type LogLevel int

const (
	// Debug is the most verbose log level
	Debug LogLevel = iota
	// Info is for general operational information
	Info
	// Success is for successful test operations
	Success
	// Warning is for non-critical issues
	Warning
	// Error is for failures that don't terminate the test
	Error
	// Fatal is for failures that terminate the test
	Fatal
)

// ReuseBehavior defines how infrastructure should be reused between tests
type ReuseBehavior int

const (
	// AlwaysFresh always creates fresh infrastructure for each test
	AlwaysFresh ReuseBehavior = iota
	// ReuseIfAvailable reuses existing infrastructure if available and compatible
	ReuseIfAvailable
	// ReuseWithFlush reuses but flushes/resets state before test
	ReuseWithFlush
	// ReuseWithCleanData reuses but with clean data sources
	ReuseWithCleanData
)

// IsolationLevel defines how isolated a test's resources should be
type IsolationLevel int

const (
	// ShareAll allows sharing all infrastructure with other tests
	ShareAll IsolationLevel = iota
	// ShareInfrastructureOnly allows sharing service infrastructure but needs isolated data
	ShareInfrastructureOnly
	// Isolated requires completely isolated infrastructure
	Isolated
)

// DataStore is a generic container for test data
type DataStore struct {
	// Map to store any type of data
	data map[string]interface{}
}

// NewDataStore creates a new DataStore
func NewDataStore() *DataStore {
	return &DataStore{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the DataStore
func (ds *DataStore) Set(key string, value interface{}) {
	ds.data[key] = value
}

// Get retrieves a value from the DataStore
func (ds *DataStore) Get(key string) (interface{}, bool) {
	value, ok := ds.data[key]
	return value, ok
}

// AsMap returns a copy of the internal data as a map
func (ds *DataStore) AsMap() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range ds.data {
		result[key] = value
	}
	return result
}

// GetString retrieves a string value from the DataStore
func (ds *DataStore) GetString(key string) (string, bool) {
	value, ok := ds.data[key]
	if !ok {
		return "", false
	}
	strValue, ok := value.(string)
	return strValue, ok
}

// GetInt retrieves an int value from the DataStore
func (ds *DataStore) GetInt(key string) (int, bool) {
	value, ok := ds.data[key]
	if !ok {
		return 0, false
	}
	intValue, ok := value.(int)
	return intValue, ok
}

// TestContext holds the context for a test
type TestContext struct {
	// The testing.T instance
	T *testing.T

	// Infrastructure provider
	InfraProvider InfrastructureProvider

	// Test runner function
	Runner TestRunner

	// Test options
	Options []TestOption

	// Middleware
	Middleware []MiddlewareFunc

	// Data store for test state
	Store *DataStore

	// Configuration
	LogLevel       LogLevel
	Timeout        time.Duration
	ReuseBehavior  ReuseBehavior
	IsolationLevel IsolationLevel
	Flags          []string
	ExcludeFlags   []string

	// Additional request interceptors
	RequestInterceptors []func(req interface{})
}

// InfrastructureProvider is an interface for managing test infrastructure
type InfrastructureProvider interface {
	// Initialize prepares the infrastructure
	Initialize(context.Context) error

	// Start launches all required services
	Start(context.Context) error

	// Stop terminates all services
	Stop(context.Context) error

	// AddService registers a service to be managed
	AddService(ServiceRequirement) error

	// GetService retrieves a running service by name
	GetService(name string) (Service, error)

	// ServiceEndpoint returns connection info for a service
	ServiceEndpoint(serviceName string, portName string) (string, error)

	// Flush resets service state but keeps services running
	Flush(context.Context) error
}

// Service represents a containerized service in the test environment
type Service interface {
	// Name returns the name of the service
	Name() string

	// Image returns the Docker image name
	Image() string

	// Ports returns the port mappings
	Ports() []Port

	// Env returns the environment variables
	Env() map[string]string

	// Labels returns the Docker labels
	Labels() map[string]string

	// Dependencies returns the names of services this service depends on
	Dependencies() []string

	// WaitStrategy returns the strategy for waiting for the service to be ready
	WaitStrategy() WaitStrategy

	// Start starts the service
	Start(context.Context) error

	// Stop stops the service
	Stop(context.Context) error

	// IsRunning returns whether the service is running
	IsRunning() bool
}

// Port represents a port mapping for a service
type Port struct {
	Name     string
	Internal int
	External int
}

// WaitStrategy defines how to wait for a service to be ready
type WaitStrategy interface {
	// Wait waits for a service to be ready
	Wait(context.Context) error
}

// ServiceRequirement defines the requirements for a service
type ServiceRequirement struct {
	Name            string
	Image           string
	Version         string
	Ports           []Port
	Environment     map[string]string
	Labels          map[string]string
	Dependencies    []string
	ReadinessProbe  WaitStrategy
	StartupTimeout  time.Duration
	ResourceLimits  *ResourceLimits
	CleanupBehavior CleanupBehavior
}

// ResourceLimits defines resource constraints for a service
type ResourceLimits struct {
	CPUPercent int
	MemoryMB   int
}

// CleanupBehavior defines what happens to a service when it's no longer needed
type CleanupBehavior int

const (
	// KeepRunning keeps the service running
	KeepRunning CleanupBehavior = iota

	// StopOnly stops the service but doesn't remove it
	StopOnly

	// Remove stops and removes the service
	Remove
)

// NewTestContext creates a new TestContext
func NewTestContext(t *testing.T, opts ...TestOption) *TestContext {
	tc := &TestContext{
		T:                   t,
		Store:               NewDataStore(),
		LogLevel:            Info,
		Timeout:             2 * time.Minute,
		ReuseBehavior:       ReuseIfAvailable,
		IsolationLevel:      ShareInfrastructureOnly,
		Flags:               []string{},
		ExcludeFlags:        []string{},
		RequestInterceptors: []func(req interface{}){},
	}

	// Apply options
	for _, opt := range opts {
		tc = opt(tc)
	}

	return tc
}

// Debug logs a debug message
func (tc *TestContext) Debug(format string, args ...interface{}) {
	if tc.LogLevel <= Debug {
		tc.log(Debug, format, args...)
	}
}

// Info logs an info message
func (tc *TestContext) Info(format string, args ...interface{}) {
	if tc.LogLevel <= Info {
		tc.log(Info, format, args...)
	}
}

// Success logs a success message
func (tc *TestContext) Success(format string, args ...interface{}) {
	if tc.LogLevel <= Success {
		tc.log(Success, format, args...)
	}
}

// Warning logs a warning message
func (tc *TestContext) Warning(format string, args ...interface{}) {
	if tc.LogLevel <= Warning {
		tc.log(Warning, format, args...)
	}
}

// Error logs an error message
func (tc *TestContext) Error(format string, args ...interface{}) {
	if tc.LogLevel <= Error {
		tc.log(Error, format, args...)
	}
}

// Fatal logs a fatal message and fails the test
func (tc *TestContext) Fatal(format string, args ...interface{}) {
	if tc.LogLevel <= Fatal {
		tc.log(Fatal, format, args...)
		tc.T.FailNow()
	}
}

// log logs a message with the given level
func (tc *TestContext) log(level LogLevel, format string, args ...interface{}) {
	// Implementation depends on the logging framework used
	tc.T.Logf(format, args...)
}

// AddRequestInterceptor adds a function that will be called for each request
func (tc *TestContext) AddRequestInterceptor(interceptor func(req interface{})) {
	tc.RequestInterceptors = append(tc.RequestInterceptors, interceptor)
}

// ApplyInterceptors applies all request interceptors to a request
func (tc *TestContext) ApplyInterceptors(req interface{}) {
	for _, interceptor := range tc.RequestInterceptors {
		interceptor(req)
	}
}

// Context returns a context.Context for this test context
func (tc *TestContext) Context() context.Context {
	return context.Background()
}

// Atomic combines a TaskFunc and an ExpectFunc
func Atomic[T any]() AtomicFunc[T] {
	return func(ctx context.Context, tc *TestContext, tf TaskFunc[T], ef ExpectFunc[T]) (T, error) {
		result, err := tf(ctx, tc)
		if err != nil {
			return result, err
		}
		return result, ef(ctx, tc, result)
	}
}

// WithExpectations allows daisy chaining multiple expectations
func WithExpectations[T any](expectations ...ExpectFunc[T]) ExpectFunc[T] {
	return func(ctx context.Context, tc *TestContext, result T) error {
		for _, expectation := range expectations {
			if err := expectation(ctx, tc, result); err != nil {
				tc.Warning("Expectation failed: %v", err)
				return err
			}
		}
		return nil
	}
}
