package mock

import (
	"context"
	"errors"
	"testing"
)

func TestRegistry(t *testing.T) {
	t.Run("register and get", func(t *testing.T) {
		registry := NewRegistry()
		mock := NewMock("test")

		registry.Register(mock)

		got, ok := registry.Get("test")
		if !ok {
			t.Fatal("expected to find mock")
		}
		if got != mock {
			t.Error("expected same mock instance")
		}
	})

	t.Run("remove", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register(NewMock("test"))
		registry.Remove("test")

		_, ok := registry.Get("test")
		if ok {
			t.Error("expected mock to be removed")
		}
	})

	t.Run("clear", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register(NewMock("test1"))
		registry.Register(NewMock("test2"))
		registry.Clear()

		names := registry.Names()
		if len(names) != 0 {
			t.Errorf("expected no mocks, got %d", len(names))
		}
	})

	t.Run("names", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register(NewMock("mock1"))
		registry.Register(NewMock("mock2"))

		names := registry.Names()
		if len(names) != 2 {
			t.Errorf("expected 2 names, got %d", len(names))
		}
	})
}

func TestMock(t *testing.T) {
	t.Run("basic call", func(t *testing.T) {
		m := NewMock("test")
		m.On("GetUser").Return("user-123")

		result, err := m.Call(context.Background(), "GetUser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "user-123" {
			t.Errorf("expected 'user-123', got %v", result)
		}
	})

	t.Run("return error", func(t *testing.T) {
		m := NewMock("test")
		expectedErr := errors.New("not found")
		m.On("GetUser").ReturnError(expectedErr)

		_, err := m.Call(context.Background(), "GetUser")
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("unhandled method", func(t *testing.T) {
		m := NewMock("test")

		_, err := m.Call(context.Background(), "UnknownMethod")
		if err == nil {
			t.Error("expected error for unhandled method")
		}
	})

	t.Run("callback", func(t *testing.T) {
		m := NewMock("test")
		m.On("Add").Callback(func(_ context.Context, args ...any) (any, error) {
			a := args[0].(int)
			b := args[1].(int)
			return a + b, nil
		})

		result, err := m.Call(context.Background(), "Add", 2, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 5 {
			t.Errorf("expected 5, got %v", result)
		}
	})
}

func TestMockCalls(t *testing.T) {
	t.Run("records calls", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return(nil)

		_, _ = m.Call(context.Background(), "Method", "arg1", "arg2")
		_, _ = m.Call(context.Background(), "Method", "arg3")

		calls := m.Calls()
		if len(calls) != 2 {
			t.Errorf("expected 2 calls, got %d", len(calls))
		}
	})

	t.Run("calls for method", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method1").Return(nil)
		m.On("Method2").Return(nil)

		_, _ = m.Call(context.Background(), "Method1")
		_, _ = m.Call(context.Background(), "Method2")
		_, _ = m.Call(context.Background(), "Method1")

		calls := m.CallsFor("Method1")
		if len(calls) != 2 {
			t.Errorf("expected 2 calls to Method1, got %d", len(calls))
		}
	})

	t.Run("reset", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return(nil)

		_, _ = m.Call(context.Background(), "Method")
		m.Reset()

		calls := m.Calls()
		if len(calls) != 0 {
			t.Errorf("expected 0 calls after reset, got %d", len(calls))
		}
	})
}

func TestMockAssertions(t *testing.T) {
	t.Run("assert called", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return(nil)

		if m.AssertCalled("Method") {
			t.Error("should not be called yet")
		}

		_, _ = m.Call(context.Background(), "Method")

		if !m.AssertCalled("Method") {
			t.Error("should be called")
		}
	})

	t.Run("assert called times", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return(nil)

		_, _ = m.Call(context.Background(), "Method")
		_, _ = m.Call(context.Background(), "Method")

		if !m.AssertCalledTimes("Method", 2) {
			t.Error("should be called twice")
		}
		if m.AssertCalledTimes("Method", 3) {
			t.Error("should not be called three times")
		}
	})

	t.Run("assert called with", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return(nil)

		_, _ = m.Call(context.Background(), "Method", "arg1", 42)

		if !m.AssertCalledWith("Method", "arg1", 42) {
			t.Error("should match exact args")
		}
		if m.AssertCalledWith("Method", "arg1", 99) {
			t.Error("should not match different args")
		}
	})

	t.Run("assert called with any matcher", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return(nil)

		_, _ = m.Call(context.Background(), "Method", "arg1", 42)

		if !m.AssertCalledWith("Method", "arg1", Any()) {
			t.Error("Any matcher should match any value")
		}
	})
}

func TestMethodHandlerTimes(t *testing.T) {
	t.Run("once", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return("result").Once()

		// First call should work
		_, err := m.Call(context.Background(), "Method")
		if err != nil {
			t.Fatalf("first call failed: %v", err)
		}

		// Second call should fail
		_, err = m.Call(context.Background(), "Method")
		if err == nil {
			t.Error("expected error on second call")
		}
	})

	t.Run("twice", func(t *testing.T) {
		m := NewMock("test")
		m.On("Method").Return("result").Twice()

		for i := 0; i < 2; i++ {
			_, err := m.Call(context.Background(), "Method")
			if err != nil {
				t.Fatalf("call %d failed: %v", i+1, err)
			}
		}

		// Third call should fail
		_, err := m.Call(context.Background(), "Method")
		if err == nil {
			t.Error("expected error on third call")
		}
	})
}

func TestMatchers(t *testing.T) {
	t.Run("any matcher", func(t *testing.T) {
		matcher := Any()
		if !matcher.Match("anything") {
			t.Error("should match string")
		}
		if !matcher.Match(123) {
			t.Error("should match int")
		}
		if !matcher.Match(nil) {
			t.Error("should match nil")
		}
	})

	t.Run("type matcher", func(t *testing.T) {
		matcher := OfType[string]()
		if !matcher.Match("hello") {
			t.Error("should match string")
		}
		if matcher.Match(123) {
			t.Error("should not match int")
		}
		if matcher.Match(nil) {
			t.Error("should not match nil")
		}
	})

	t.Run("func matcher", func(t *testing.T) {
		matcher := MatchFunc(func(v any) bool {
			n, ok := v.(int)
			return ok && n > 0
		})

		if !matcher.Match(5) {
			t.Error("should match positive int")
		}
		if matcher.Match(-5) {
			t.Error("should not match negative int")
		}
		if matcher.Match("string") {
			t.Error("should not match string")
		}
	})
}

func TestMatchArgs(t *testing.T) {
	tests := []struct {
		name string
		a    []any
		b    []any
		want bool
	}{
		{"empty", []any{}, []any{}, true},
		{"equal", []any{1, "a"}, []any{1, "a"}, true},
		{"different", []any{1, "a"}, []any{1, "b"}, false},
		{"different length", []any{1}, []any{1, 2}, false},
		{"with any matcher", []any{1, "a"}, []any{1, Any()}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchArgs(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("matchArgs(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
