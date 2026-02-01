package mock

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// ResponseSequence returns different responses on subsequent calls.
type ResponseSequence struct {
	responses []Response
	index     int
	mu        sync.Mutex
}

// NewResponseSequence creates a new response sequence.
func NewResponseSequence(responses ...Response) *ResponseSequence {
	return &ResponseSequence{
		responses: responses,
	}
}

// Next returns the next response in the sequence.
func (s *ResponseSequence) Next() Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.responses) == 0 {
		return Response{}
	}

	if s.index >= len(s.responses) {
		// Return the last response for subsequent calls
		return s.responses[len(s.responses)-1]
	}

	resp := s.responses[s.index]
	s.index++
	return resp
}

// Reset resets the sequence to the beginning.
func (s *ResponseSequence) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index = 0
}

// Response represents a mock response.
type Response struct {
	Value any
	Error error
	Delay time.Duration
}

// NewResponse creates a new response with a value.
func NewResponse(value any) Response {
	return Response{Value: value}
}

// NewErrorResponse creates a new response with an error.
func NewErrorResponse(err error) Response {
	return Response{Error: err}
}

// NewDelayedResponse creates a new response with a delay.
func NewDelayedResponse(value any, delay time.Duration) Response {
	return Response{Value: value, Delay: delay}
}

// Apply returns the response value after any configured delay.
func (r Response) Apply(ctx context.Context) (any, error) {
	if r.Delay > 0 {
		select {
		case <-time.After(r.Delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if r.Error != nil {
		return nil, r.Error
	}
	return r.Value, nil
}

// Recorder records all calls and responses for verification.
type Recorder struct {
	entries []RecordEntry
	mu      sync.RWMutex
}

// NewRecorder creates a new recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		entries: make([]RecordEntry, 0),
	}
}

// Record adds an entry to the recorder.
func (r *Recorder) Record(entry RecordEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry)
}

// Entries returns all recorded entries.
func (r *Recorder) Entries() []RecordEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]RecordEntry, len(r.entries))
	copy(entries, r.entries)
	return entries
}

// Clear removes all recorded entries.
func (r *Recorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make([]RecordEntry, 0)
}

// Count returns the number of recorded entries.
func (r *Recorder) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// FindByMethod returns all entries for a specific method.
func (r *Recorder) FindByMethod(method string) []RecordEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var entries []RecordEntry
	for _, e := range r.entries {
		if e.Method == method {
			entries = append(entries, e)
		}
	}
	return entries
}

// FindByMock returns all entries for a specific mock.
func (r *Recorder) FindByMock(mockName string) []RecordEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var entries []RecordEntry
	for _, e := range r.entries {
		if e.MockName == mockName {
			entries = append(entries, e)
		}
	}
	return entries
}

// RecordEntry represents a single recorded call.
type RecordEntry struct {
	MockName  string    `json:"mock_name"`
	Method    string    `json:"method"`
	Args      []any     `json:"args"`
	Result    any       `json:"result,omitempty"`
	Error     error     `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// JSON returns the entry as JSON.
func (e RecordEntry) JSON() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// Expectation defines expected mock behavior.
type Expectation struct {
	Mock       string
	Method     string
	Args       []any
	Response   Response
	CallCount  int
	called     int
	mu         sync.Mutex
}

// NewExpectation creates a new expectation.
func NewExpectation(mock, method string) *Expectation {
	return &Expectation{
		Mock:   mock,
		Method: method,
	}
}

// WithArgs sets the expected arguments.
func (e *Expectation) WithArgs(args ...any) *Expectation {
	e.Args = args
	return e
}

// Returns sets the expected response.
func (e *Expectation) Returns(value any) *Expectation {
	e.Response = NewResponse(value)
	return e
}

// ReturnsError sets the expected error.
func (e *Expectation) ReturnsError(err error) *Expectation {
	e.Response = NewErrorResponse(err)
	return e
}

// Times sets the expected call count.
func (e *Expectation) Times(n int) *Expectation {
	e.CallCount = n
	return e
}

// Satisfied returns true if the expectation has been satisfied.
func (e *Expectation) Satisfied() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.CallCount == 0 {
		return e.called > 0
	}
	return e.called == e.CallCount
}

// MarkCalled marks the expectation as called.
func (e *Expectation) MarkCalled() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.called++
}

// ExpectationSet manages a set of expectations.
type ExpectationSet struct {
	expectations []*Expectation
	mu           sync.RWMutex
}

// NewExpectationSet creates a new expectation set.
func NewExpectationSet() *ExpectationSet {
	return &ExpectationSet{
		expectations: make([]*Expectation, 0),
	}
}

// Add adds an expectation to the set.
func (s *ExpectationSet) Add(e *Expectation) *ExpectationSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expectations = append(s.expectations, e)
	return s
}

// Verify checks if all expectations have been satisfied.
func (s *ExpectationSet) Verify() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.expectations {
		if !e.Satisfied() {
			return &ExpectationError{
				Mock:     e.Mock,
				Method:   e.Method,
				Expected: e.CallCount,
				Actual:   e.called,
			}
		}
	}
	return nil
}

// Find finds an expectation matching the mock and method.
func (s *ExpectationSet) Find(mock, method string) *Expectation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.expectations {
		if e.Mock == mock && e.Method == method {
			return e
		}
	}
	return nil
}

// ExpectationError is returned when an expectation is not satisfied.
type ExpectationError struct {
	Mock     string
	Method   string
	Expected int
	Actual   int
}

// Error returns the error message.
func (e *ExpectationError) Error() string {
	if e.Expected == 0 {
		return "expectation not satisfied: " + e.Mock + "." + e.Method + " was not called"
	}
	return errors.New("expectation not satisfied: " + e.Mock + "." + e.Method +
		" called " + string(rune('0'+e.Actual)) + " times, expected " +
		string(rune('0'+e.Expected))).Error()
}
