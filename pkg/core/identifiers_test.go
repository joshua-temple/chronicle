package core

import (
	"strings"
	"sync"
	"testing"
)

func TestIDGeneration(t *testing.T) {
	t.Run("TestID generation is unique", func(t *testing.T) {
		ids := make(map[TestID]bool)
		for i := 0; i < 100; i++ {
			id := NewTestID()
			if ids[id] {
				t.Errorf("duplicate TestID generated: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("TraceID generation is unique", func(t *testing.T) {
		ids := make(map[TraceID]bool)
		for i := 0; i < 100; i++ {
			id := NewTraceID()
			if ids[id] {
				t.Errorf("duplicate TraceID generated: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("SpanID generation is unique", func(t *testing.T) {
		ids := make(map[SpanID]bool)
		for i := 0; i < 100; i++ {
			id := NewSpanID()
			if ids[id] {
				t.Errorf("duplicate SpanID generated: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("IDs have correct prefixes", func(t *testing.T) {
		testID := NewTestID()
		if !strings.HasPrefix(string(testID), "test_") {
			t.Errorf("TestID should have 'test_' prefix, got: %s", testID)
		}

		traceID := NewTraceID()
		if !strings.HasPrefix(string(traceID), "trace_") {
			t.Errorf("TraceID should have 'trace_' prefix, got: %s", traceID)
		}

		runID := NewRunID()
		if !strings.HasPrefix(string(runID), "run_") {
			t.Errorf("RunID should have 'run_' prefix, got: %s", runID)
		}

		spanID := NewSpanID()
		if !strings.HasPrefix(string(spanID), "span_") {
			t.Errorf("SpanID should have 'span_' prefix, got: %s", spanID)
		}

		scenarioID := NewScenarioID()
		if !strings.HasPrefix(string(scenarioID), "scn_") {
			t.Errorf("ScenarioID should have 'scn_' prefix, got: %s", scenarioID)
		}
	})
}

func TestIDValidation(t *testing.T) {
	t.Run("empty IDs are invalid", func(t *testing.T) {
		if TestID("").IsValid() {
			t.Error("empty TestID should be invalid")
		}
		if TraceID("").IsValid() {
			t.Error("empty TraceID should be invalid")
		}
		if RunID("").IsValid() {
			t.Error("empty RunID should be invalid")
		}
	})

	t.Run("non-empty IDs are valid", func(t *testing.T) {
		if !TestID("test_123").IsValid() {
			t.Error("non-empty TestID should be valid")
		}
		if !TraceID("trace_abc").IsValid() {
			t.Error("non-empty TraceID should be valid")
		}
	})
}

func TestIDString(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		toString func() string
	}{
		{"TestID", "test_123", func() string { return TestID("test_123").String() }},
		{"ScenarioID", "scn_456", func() string { return ScenarioID("scn_456").String() }},
		{"ComponentID", "comp_789", func() string { return ComponentID("comp_789").String() }},
		{"ServiceID", "svc_abc", func() string { return ServiceID("svc_abc").String() }},
		{"TraceID", "trace_def", func() string { return TraceID("trace_def").String() }},
		{"RunID", "run_ghi", func() string { return RunID("run_ghi").String() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.toString(); got != tt.id {
				t.Errorf("String() = %s, want %s", got, tt.id)
			}
		})
	}
}

func TestTraceContext(t *testing.T) {
	t.Run("NewTraceContext creates valid context", func(t *testing.T) {
		tc := NewTraceContext()
		if !tc.TraceID.IsValid() {
			t.Error("TraceContext should have valid TraceID")
		}
		if !tc.SpanID.IsValid() {
			t.Error("TraceContext should have valid SpanID")
		}
		if tc.Baggage == nil {
			t.Error("TraceContext should have initialized Baggage map")
		}
		if tc.StartTime.IsZero() {
			t.Error("TraceContext should have non-zero StartTime")
		}
	})

	t.Run("NewSpan creates child with same TraceID", func(t *testing.T) {
		parent := NewTraceContext()
		child := parent.NewSpan("child-operation")

		if child.TraceID != parent.TraceID {
			t.Error("child span should have same TraceID as parent")
		}
		if child.SpanID == parent.SpanID {
			t.Error("child span should have different SpanID than parent")
		}
		if child.ParentSpan != parent.SpanID {
			t.Error("child ParentSpan should be parent's SpanID")
		}
	})

	t.Run("baggage is copied to child spans", func(t *testing.T) {
		parent := NewTraceContext()
		parent.SetBaggage("key1", "value1")

		child := parent.NewSpan("child")
		if v, ok := child.GetBaggage("key1"); !ok || v != "value1" {
			t.Error("child should inherit parent's baggage")
		}

		// Modifying child's baggage shouldn't affect parent
		child.SetBaggage("key2", "value2")
		if _, ok := parent.GetBaggage("key2"); ok {
			t.Error("parent's baggage should not be affected by child's modifications")
		}
	})

	t.Run("Duration returns positive value", func(t *testing.T) {
		tc := NewTraceContext()
		if tc.Duration() < 0 {
			t.Error("Duration should not be negative")
		}
	})
}

func TestIDRegistry(t *testing.T) {
	t.Run("RegisterTest succeeds for new ID", func(t *testing.T) {
		r := NewIDRegistry()
		err := r.RegisterTest(TestID("test_1"))
		if err != nil {
			t.Errorf("RegisterTest failed: %v", err)
		}
		if !r.HasTest(TestID("test_1")) {
			t.Error("HasTest should return true for registered ID")
		}
	})

	t.Run("RegisterTest fails for duplicate ID", func(t *testing.T) {
		r := NewIDRegistry()
		_ = r.RegisterTest(TestID("test_1"))
		err := r.RegisterTest(TestID("test_1"))
		if err == nil {
			t.Error("RegisterTest should fail for duplicate ID")
		}
	})

	t.Run("RegisterTest fails for empty ID", func(t *testing.T) {
		r := NewIDRegistry()
		err := r.RegisterTest(TestID(""))
		if err == nil {
			t.Error("RegisterTest should fail for empty ID")
		}
	})

	t.Run("UnregisterTest removes ID", func(t *testing.T) {
		r := NewIDRegistry()
		_ = r.RegisterTest(TestID("test_1"))
		r.UnregisterTest(TestID("test_1"))
		if r.HasTest(TestID("test_1")) {
			t.Error("HasTest should return false after UnregisterTest")
		}
	})

	t.Run("RegisterScenario succeeds for new ID", func(t *testing.T) {
		r := NewIDRegistry()
		err := r.RegisterScenario(ScenarioID("scn_1"))
		if err != nil {
			t.Errorf("RegisterScenario failed: %v", err)
		}
	})

	t.Run("RegisterScenario fails for duplicate ID", func(t *testing.T) {
		r := NewIDRegistry()
		_ = r.RegisterScenario(ScenarioID("scn_1"))
		err := r.RegisterScenario(ScenarioID("scn_1"))
		if err == nil {
			t.Error("RegisterScenario should fail for duplicate ID")
		}
	})

	t.Run("RegisterComponent succeeds for new ID", func(t *testing.T) {
		r := NewIDRegistry()
		err := r.RegisterComponent(ComponentID("CreateUser"))
		if err != nil {
			t.Errorf("RegisterComponent failed: %v", err)
		}
	})

	t.Run("RegisterComponent fails for duplicate ID", func(t *testing.T) {
		r := NewIDRegistry()
		_ = r.RegisterComponent(ComponentID("CreateUser"))
		err := r.RegisterComponent(ComponentID("CreateUser"))
		if err == nil {
			t.Error("RegisterComponent should fail for duplicate ID")
		}
	})

	t.Run("RegisterService succeeds for new ID", func(t *testing.T) {
		r := NewIDRegistry()
		err := r.RegisterService(ServiceID("postgres"))
		if err != nil {
			t.Errorf("RegisterService failed: %v", err)
		}
	})

	t.Run("RegisterTrace succeeds for new trace", func(t *testing.T) {
		r := NewIDRegistry()
		tc := NewTraceContext()
		err := r.RegisterTrace(tc)
		if err != nil {
			t.Errorf("RegisterTrace failed: %v", err)
		}

		got, ok := r.GetTrace(tc.TraceID)
		if !ok {
			t.Error("GetTrace should return true for registered trace")
		}
		if got != tc {
			t.Error("GetTrace should return the same TraceContext")
		}
	})

	t.Run("RegisterTrace fails for duplicate trace", func(t *testing.T) {
		r := NewIDRegistry()
		tc := NewTraceContext()
		_ = r.RegisterTrace(tc)
		err := r.RegisterTrace(tc)
		if err == nil {
			t.Error("RegisterTrace should fail for duplicate TraceID")
		}
	})

	t.Run("RegisterTrace fails for nil", func(t *testing.T) {
		r := NewIDRegistry()
		err := r.RegisterTrace(nil)
		if err == nil {
			t.Error("RegisterTrace should fail for nil TraceContext")
		}
	})

	t.Run("Clear removes all IDs", func(t *testing.T) {
		r := NewIDRegistry()
		_ = r.RegisterTest(TestID("test_1"))
		_ = r.RegisterScenario(ScenarioID("scn_1"))
		_ = r.RegisterComponent(ComponentID("comp_1"))
		_ = r.RegisterService(ServiceID("svc_1"))
		_ = r.RegisterTrace(NewTraceContext())

		r.Clear()

		stats := r.Stats()
		for k, v := range stats {
			if v != 0 {
				t.Errorf("Clear should remove all %s, got count: %d", k, v)
			}
		}
	})

	t.Run("Stats returns correct counts", func(t *testing.T) {
		r := NewIDRegistry()
		_ = r.RegisterTest(TestID("test_1"))
		_ = r.RegisterTest(TestID("test_2"))
		_ = r.RegisterScenario(ScenarioID("scn_1"))
		_ = r.RegisterComponent(ComponentID("comp_1"))
		_ = r.RegisterComponent(ComponentID("comp_2"))
		_ = r.RegisterComponent(ComponentID("comp_3"))

		stats := r.Stats()
		if stats["tests"] != 2 {
			t.Errorf("expected 2 tests, got %d", stats["tests"])
		}
		if stats["scenarios"] != 1 {
			t.Errorf("expected 1 scenario, got %d", stats["scenarios"])
		}
		if stats["components"] != 3 {
			t.Errorf("expected 3 components, got %d", stats["components"])
		}
	})
}

func TestIDRegistryConcurrency(t *testing.T) {
	r := NewIDRegistry()
	var wg sync.WaitGroup

	// Concurrent registrations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := NewTestID()
			_ = r.RegisterTest(id)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Stats()
		}()
	}

	wg.Wait()

	// Should have registered some tests (may have fewer due to race conditions)
	stats := r.Stats()
	if stats["tests"] == 0 {
		t.Error("expected some tests to be registered")
	}
}
