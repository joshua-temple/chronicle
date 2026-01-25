package context

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/infrastructure"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogDebug, "DEBUG"},
		{LogInfo, "INFO"},
		{LogWarn, "WARN"},
		{LogError, "ERROR"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("LogLevel(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}

func TestNarrativeLevel(t *testing.T) {
	tests := []struct {
		level    NarrativeLevel
		expected string
	}{
		{NarrativeSummary, "summary"},
		{NarrativeDetail, "detail"},
		{NarrativeVerbose, "verbose"},
		{NarrativeLevel(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("NarrativeLevel(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}

func TestNewContext(t *testing.T) {
	t.Run("creates context with defaults", func(t *testing.T) {
		ctx := New(context.Background())
		if ctx == nil {
			t.Fatal("New should not return nil")
		}
		if ctx.Trace() == nil {
			t.Error("context should have a trace")
		}
	})

	t.Run("creates context with custom trace", func(t *testing.T) {
		tc := core.NewTraceContext()
		ctx := New(context.Background(), WithTrace(tc))
		if ctx.Trace() != tc {
			t.Error("context should use provided trace")
		}
	})

	t.Run("creates context with flags", func(t *testing.T) {
		flags := map[string]any{"debug": true}
		ctx := New(context.Background(), WithFlags(flags))
		if ctx.Flag("debug") != true {
			t.Error("context should have flag 'debug'")
		}
	})

	t.Run("creates context with params", func(t *testing.T) {
		params := map[string]any{"timeout": 30}
		ctx := New(context.Background(), WithParams(params))
		if ctx.Param("timeout") != 30 {
			t.Error("context should have param 'timeout'")
		}
	})
}

func TestContextGetSet(t *testing.T) {
	t.Run("Get returns false for missing key", func(t *testing.T) {
		ctx := New(context.Background())
		_, ok := ctx.Get("missing")
		if ok {
			t.Error("Get should return false for missing key")
		}
	})

	t.Run("Set and Get work correctly", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("key", "value")
		v, ok := ctx.Get("key")
		if !ok {
			t.Error("Get should return true for existing key")
		}
		if v != "value" {
			t.Errorf("Get returned %v, want 'value'", v)
		}
	})

	t.Run("Set overwrites existing value", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("key", "value1")
		ctx.Set("key", "value2")
		v, _ := ctx.Get("key")
		if v != "value2" {
			t.Errorf("Get returned %v, want 'value2'", v)
		}
	})
}

func TestGenericAccessors(t *testing.T) {
	t.Run("Get returns typed value", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("count", 42)
		v := Get[int](ctx, "count")
		if v != 42 {
			t.Errorf("Get[int] returned %d, want 42", v)
		}
	})

	t.Run("Get returns zero for missing key", func(t *testing.T) {
		ctx := New(context.Background())
		v := Get[int](ctx, "missing")
		if v != 0 {
			t.Errorf("Get[int] returned %d for missing key, want 0", v)
		}
	})

	t.Run("Get returns zero for type mismatch", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("key", "string")
		v := Get[int](ctx, "key")
		if v != 0 {
			t.Errorf("Get[int] returned %d for string value, want 0", v)
		}
	})

	t.Run("GetOK returns false for missing key", func(t *testing.T) {
		ctx := New(context.Background())
		_, ok := GetOK[int](ctx, "missing")
		if ok {
			t.Error("GetOK should return false for missing key")
		}
	})

	t.Run("GetOK returns false for type mismatch", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("key", "string")
		_, ok := GetOK[int](ctx, "key")
		if ok {
			t.Error("GetOK should return false for type mismatch")
		}
	})

	t.Run("GetOK returns true for valid key", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("count", 42)
		v, ok := GetOK[int](ctx, "count")
		if !ok {
			t.Error("GetOK should return true for valid key")
		}
		if v != 42 {
			t.Errorf("GetOK returned %d, want 42", v)
		}
	})

	t.Run("Set stores typed value", func(t *testing.T) {
		ctx := New(context.Background())
		Set(ctx, "count", 42)
		v, ok := ctx.Get("count")
		if !ok || v != 42 {
			t.Error("Set should store typed value")
		}
	})

	t.Run("MustGet returns value", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("count", 42)
		v := MustGet[int](ctx, "count")
		if v != 42 {
			t.Errorf("MustGet returned %d, want 42", v)
		}
	})

	t.Run("MustGet panics for missing key", func(t *testing.T) {
		ctx := New(context.Background())
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGet should panic for missing key")
			}
		}()
		MustGet[int](ctx, "missing")
	})

	t.Run("MustGet panics for type mismatch", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("key", "string")
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGet should panic for type mismatch")
			}
		}()
		MustGet[int](ctx, "key")
	})
}

func TestContextClient(t *testing.T) {
	t.Run("Client returns error for missing client", func(t *testing.T) {
		ctx := New(context.Background())
		_, err := ctx.Client("missing")
		if err == nil {
			t.Error("Client should return error for missing client")
		}
	})

	t.Run("Client uses provider function", func(t *testing.T) {
		mockClient := &struct{ name string }{name: "mock"}
		provider := func(name string) (any, error) {
			if name == "db" {
				return mockClient, nil
			}
			return nil, errors.New("not found")
		}

		ctx := New(context.Background(), WithClientProvider(provider))
		client, err := ctx.Client("db")
		if err != nil {
			t.Errorf("Client returned error: %v", err)
		}
		if client != mockClient {
			t.Error("Client should return mock client")
		}

		// Second call should use cached client
		client2, _ := ctx.Client("db")
		if client2 != mockClient {
			t.Error("Client should cache the client")
		}
	})
}

func TestContextFlags(t *testing.T) {
	t.Run("Flag returns nil for missing flag", func(t *testing.T) {
		ctx := New(context.Background())
		if ctx.Flag("missing") != nil {
			t.Error("Flag should return nil for missing flag")
		}
	})

	t.Run("Flag returns value for existing flag", func(t *testing.T) {
		flags := map[string]any{"debug": true}
		ctx := New(context.Background(), WithFlags(flags))
		if ctx.Flag("debug") != true {
			t.Error("Flag should return value for existing flag")
		}
	})
}

func TestContextParams(t *testing.T) {
	t.Run("Param returns nil for missing param", func(t *testing.T) {
		ctx := New(context.Background())
		if ctx.Param("missing") != nil {
			t.Error("Param should return nil for missing param")
		}
	})

	t.Run("Param returns value for existing param", func(t *testing.T) {
		params := map[string]any{"timeout": 30}
		ctx := New(context.Background(), WithParams(params))
		if ctx.Param("timeout") != 30 {
			t.Error("Param should return value for existing param")
		}
	})
}

func TestContextTracing(t *testing.T) {
	t.Run("Trace returns trace context", func(t *testing.T) {
		ctx := New(context.Background())
		tc := ctx.Trace()
		if tc == nil {
			t.Fatal("Trace should not return nil")
		}
		if !tc.TraceID.IsValid() {
			t.Error("Trace should have valid TraceID")
		}
	})

	t.Run("WithSpan creates child span", func(t *testing.T) {
		ctx := New(context.Background())
		originalTrace := ctx.Trace()
		childCtx := ctx.WithSpan("operation")
		childTrace := childCtx.Trace()

		if childTrace.TraceID != originalTrace.TraceID {
			t.Error("child span should have same TraceID")
		}
		if childTrace.SpanID == originalTrace.SpanID {
			t.Error("child span should have different SpanID")
		}
		if childTrace.ParentSpan != originalTrace.SpanID {
			t.Error("child span ParentSpan should be parent's SpanID")
		}
	})
}

func TestContextChild(t *testing.T) {
	t.Run("Child Set propagates to parent for sibling access", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.Set("parent_key", "parent_value")

		child := ctx.Child("worker")

		// Child can read parent's values
		v, ok := child.Get("parent_key")
		if !ok || v != "parent_value" {
			t.Error("child should be able to read parent's values")
		}

		// Child's Set values propagate to parent (so siblings can access)
		child.Set("child_key", "child_value")
		v, ok = ctx.Get("child_key")
		if !ok || v != "child_value" {
			t.Error("parent should see child's Set values (for sibling access)")
		}
	})

	t.Run("Child SetLocal is isolated", func(t *testing.T) {
		ctx := New(context.Background())
		child := ctx.Child("worker")

		// Child's SetLocal values don't affect parent
		child.SetLocal("local_key", "local_value")
		_, ok := ctx.Get("local_key")
		if ok {
			t.Error("parent should not see child's SetLocal values")
		}

		// But child can still access its own local value
		v, ok := child.Get("local_key")
		if !ok || v != "local_value" {
			t.Error("child should be able to read its own local values")
		}
	})

	t.Run("Child shares flags and params", func(t *testing.T) {
		flags := map[string]any{"debug": true}
		params := map[string]any{"timeout": 30}
		ctx := New(context.Background(), WithFlags(flags), WithParams(params))
		child := ctx.Child("worker")

		if child.Flag("debug") != true {
			t.Error("child should have access to parent's flags")
		}
		if child.Param("timeout") != 30 {
			t.Error("child should have access to parent's params")
		}
	})
}

func TestContextLogging(t *testing.T) {
	t.Run("Log records entries", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.SetComponentName("TestComponent")
		ctx.Log(LogInfo, "test message %d", 42)

		logs := ctx.Logs()
		if len(logs) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(logs))
		}
		if logs[0].Level != LogInfo {
			t.Error("log level should be INFO")
		}
		if logs[0].Message != "test message %d" {
			t.Error("log message mismatch")
		}
		if logs[0].Component != "TestComponent" {
			t.Error("log component mismatch")
		}
	})
}

func TestContextNarrative(t *testing.T) {
	t.Run("Narrate records entries", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.SetComponentName("TestComponent")
		ctx.Narrate(NarrativeDetail, "test action", map[string]any{"key": "value"})

		narrative := ctx.Narrative()
		if len(narrative) != 1 {
			t.Fatalf("expected 1 narrative entry, got %d", len(narrative))
		}
		if narrative[0].Level != NarrativeDetail {
			t.Error("narrative level should be detail")
		}
		if narrative[0].Action != "test action" {
			t.Error("narrative action mismatch")
		}
		if narrative[0].Details["key"] != "value" {
			t.Error("narrative details mismatch")
		}
	})
}

func TestContextTeardown(t *testing.T) {
	t.Run("FailureReason returns nil by default", func(t *testing.T) {
		ctx := New(context.Background())
		if ctx.FailureReason() != nil {
			t.Error("FailureReason should return nil by default")
		}
	})

	t.Run("PartialResults returns empty map by default", func(t *testing.T) {
		ctx := New(context.Background())
		if len(ctx.PartialResults()) != 0 {
			t.Error("PartialResults should return empty map by default")
		}
	})
}

func TestContextComponentName(t *testing.T) {
	t.Run("ComponentName and SetComponentName work", func(t *testing.T) {
		ctx := New(context.Background())
		ctx.SetComponentName("MyComponent")
		if ctx.ComponentName() != "MyComponent" {
			t.Error("ComponentName should return set value")
		}
	})
}

func TestThreadSafeContext(t *testing.T) {
	t.Run("concurrent access is safe", func(t *testing.T) {
		ctx := NewThreadSafe(New(context.Background()))
		var wg sync.WaitGroup

		// Concurrent writes
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				ctx.Set("key", i)
			}(i)
		}

		// Concurrent reads
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.Get("key")
			}()
		}

		wg.Wait()
		// If we get here without a race condition, the test passes
	})
}

func TestContextWithPointers(t *testing.T) {
	type User struct {
		ID    string
		Email string
	}

	t.Run("Get and Set with pointer types", func(t *testing.T) {
		ctx := New(context.Background())
		user := &User{ID: "123", Email: "test@example.com"}
		Set(ctx, "user", user)

		retrieved := Get[*User](ctx, "user")
		if retrieved == nil {
			t.Fatal("retrieved user should not be nil")
		}
		if retrieved.ID != "123" || retrieved.Email != "test@example.com" {
			t.Error("retrieved user should match original")
		}
	})

	t.Run("Get and Set with value types", func(t *testing.T) {
		ctx := New(context.Background())
		user := User{ID: "456", Email: "other@example.com"}
		Set(ctx, "user", user)

		retrieved := Get[User](ctx, "user")
		if retrieved.ID != "456" || retrieved.Email != "other@example.com" {
			t.Error("retrieved user should match original")
		}
	})
}

func TestContext_Endpoint(t *testing.T) {
	t.Run("returns endpoint from registry", func(t *testing.T) {
		registry := infrastructure.NewEndpointRegistry()
		registry.Register("postgres", infrastructure.Endpoint{
			Host: "localhost",
			Port: 5432,
		})

		ctx := New(context.Background(), WithEndpointRegistry(registry))

		ep, ok := ctx.Endpoint("postgres")
		if !ok {
			t.Fatal("Endpoint() returned false, want true")
		}

		if ep.Port != 5432 {
			t.Errorf("Endpoint().Port = %d, want 5432", ep.Port)
		}
	})

	t.Run("returns false for nonexistent endpoint", func(t *testing.T) {
		ctx := New(context.Background())

		_, ok := ctx.Endpoint("nonexistent")
		if ok {
			t.Error("Endpoint() returned true for nonexistent, want false")
		}
	})

	t.Run("returns false when no registry configured", func(t *testing.T) {
		ctx := New(context.Background())

		_, ok := ctx.Endpoint("postgres")
		if ok {
			t.Error("Endpoint() returned true when no registry configured, want false")
		}
	})

	t.Run("checks parent context when no local registry", func(t *testing.T) {
		registry := infrastructure.NewEndpointRegistry()
		registry.Register("redis", infrastructure.Endpoint{
			Host: "localhost",
			Port: 6379,
		})

		parent := New(context.Background(), WithEndpointRegistry(registry))
		child := parent.Child("worker")

		ep, ok := child.Endpoint("redis")
		if !ok {
			t.Fatal("child should inherit endpoint registry from parent")
		}

		if ep.Port != 6379 {
			t.Errorf("Endpoint().Port = %d, want 6379", ep.Port)
		}
	})
}
