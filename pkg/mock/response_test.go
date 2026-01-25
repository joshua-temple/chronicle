package mock

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResponseSequence(t *testing.T) {
	t.Run("returns in order", func(t *testing.T) {
		seq := NewResponseSequence(
			NewResponse("first"),
			NewResponse("second"),
			NewResponse("third"),
		)

		if seq.Next().Value != "first" {
			t.Error("expected first")
		}
		if seq.Next().Value != "second" {
			t.Error("expected second")
		}
		if seq.Next().Value != "third" {
			t.Error("expected third")
		}
	})

	t.Run("repeats last", func(t *testing.T) {
		seq := NewResponseSequence(
			NewResponse("first"),
			NewResponse("last"),
		)

		_ = seq.Next() // first
		_ = seq.Next() // last
		if seq.Next().Value != "last" {
			t.Error("should repeat last response")
		}
	})

	t.Run("empty sequence", func(t *testing.T) {
		seq := NewResponseSequence()
		resp := seq.Next()
		if resp.Value != nil || resp.Error != nil {
			t.Error("expected empty response")
		}
	})

	t.Run("reset", func(t *testing.T) {
		seq := NewResponseSequence(
			NewResponse("first"),
			NewResponse("second"),
		)

		_ = seq.Next() // first
		seq.Reset()
		if seq.Next().Value != "first" {
			t.Error("should reset to first")
		}
	})
}

func TestResponse(t *testing.T) {
	t.Run("value response", func(t *testing.T) {
		resp := NewResponse("value")
		result, err := resp.Apply(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "value" {
			t.Errorf("expected 'value', got %v", result)
		}
	})

	t.Run("error response", func(t *testing.T) {
		expectedErr := errors.New("test error")
		resp := NewErrorResponse(expectedErr)
		_, err := resp.Apply(context.Background())
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("delayed response", func(t *testing.T) {
		resp := NewDelayedResponse("value", 50*time.Millisecond)
		start := time.Now()
		result, err := resp.Apply(context.Background())
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "value" {
			t.Errorf("expected 'value', got %v", result)
		}
		if elapsed < 50*time.Millisecond {
			t.Errorf("expected at least 50ms delay, got %v", elapsed)
		}
	})

	t.Run("delayed response with cancellation", func(t *testing.T) {
		resp := NewDelayedResponse("value", 1*time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := resp.Apply(ctx)
		if err == nil {
			t.Error("expected error due to context cancellation")
		}
	})
}

func TestRecorder(t *testing.T) {
	t.Run("record and retrieve", func(t *testing.T) {
		recorder := NewRecorder()

		recorder.Record(RecordEntry{
			MockName: "mock1",
			Method:   "Method1",
			Args:     []any{"arg1"},
		})
		recorder.Record(RecordEntry{
			MockName: "mock1",
			Method:   "Method2",
			Args:     []any{"arg2"},
		})

		entries := recorder.Entries()
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("count", func(t *testing.T) {
		recorder := NewRecorder()
		recorder.Record(RecordEntry{})
		recorder.Record(RecordEntry{})

		if recorder.Count() != 2 {
			t.Errorf("expected count 2, got %d", recorder.Count())
		}
	})

	t.Run("clear", func(t *testing.T) {
		recorder := NewRecorder()
		recorder.Record(RecordEntry{})
		recorder.Clear()

		if recorder.Count() != 0 {
			t.Error("expected empty after clear")
		}
	})

	t.Run("find by method", func(t *testing.T) {
		recorder := NewRecorder()
		recorder.Record(RecordEntry{Method: "Method1"})
		recorder.Record(RecordEntry{Method: "Method2"})
		recorder.Record(RecordEntry{Method: "Method1"})

		entries := recorder.FindByMethod("Method1")
		if len(entries) != 2 {
			t.Errorf("expected 2 entries for Method1, got %d", len(entries))
		}
	})

	t.Run("find by mock", func(t *testing.T) {
		recorder := NewRecorder()
		recorder.Record(RecordEntry{MockName: "mock1"})
		recorder.Record(RecordEntry{MockName: "mock2"})
		recorder.Record(RecordEntry{MockName: "mock1"})

		entries := recorder.FindByMock("mock1")
		if len(entries) != 2 {
			t.Errorf("expected 2 entries for mock1, got %d", len(entries))
		}
	})
}

func TestRecordEntry(t *testing.T) {
	entry := RecordEntry{
		MockName:  "test-mock",
		Method:    "TestMethod",
		Args:      []any{"arg1", 42},
		Timestamp: time.Now(),
	}

	data, err := entry.JSON()
	if err != nil {
		t.Fatalf("JSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestExpectation(t *testing.T) {
	t.Run("basic expectation", func(t *testing.T) {
		exp := NewExpectation("mock", "Method").
			WithArgs("arg1", 42).
			Returns("result")

		if exp.Mock != "mock" {
			t.Errorf("expected mock 'mock', got %s", exp.Mock)
		}
		if exp.Method != "Method" {
			t.Errorf("expected method 'Method', got %s", exp.Method)
		}
	})

	t.Run("satisfied", func(t *testing.T) {
		exp := NewExpectation("mock", "Method")

		if exp.Satisfied() {
			t.Error("should not be satisfied before being called")
		}

		exp.MarkCalled()

		if !exp.Satisfied() {
			t.Error("should be satisfied after being called")
		}
	})

	t.Run("satisfied with times", func(t *testing.T) {
		exp := NewExpectation("mock", "Method").Times(2)

		exp.MarkCalled()
		if exp.Satisfied() {
			t.Error("should not be satisfied after 1 call")
		}

		exp.MarkCalled()
		if !exp.Satisfied() {
			t.Error("should be satisfied after 2 calls")
		}
	})
}

func TestExpectationSet(t *testing.T) {
	t.Run("add and find", func(t *testing.T) {
		set := NewExpectationSet()
		exp := NewExpectation("mock", "Method")
		set.Add(exp)

		found := set.Find("mock", "Method")
		if found != exp {
			t.Error("should find the expectation")
		}

		notFound := set.Find("other", "Other")
		if notFound != nil {
			t.Error("should not find non-existent expectation")
		}
	})

	t.Run("verify success", func(t *testing.T) {
		set := NewExpectationSet()
		exp := NewExpectation("mock", "Method")
		set.Add(exp)

		exp.MarkCalled()

		err := set.Verify()
		if err != nil {
			t.Errorf("verify should succeed: %v", err)
		}
	})

	t.Run("verify failure", func(t *testing.T) {
		set := NewExpectationSet()
		set.Add(NewExpectation("mock", "Method"))

		err := set.Verify()
		if err == nil {
			t.Error("verify should fail")
		}
	})
}
