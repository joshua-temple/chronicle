package results

import (
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/execution"
)

func TestNewRunResult(t *testing.T) {
	execResults := []*execution.ScenarioResult{
		{
			ScenarioID:   core.NewScenarioID(),
			ScenarioName: "scenario-1",
			State:        execution.StateCompleted,
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(100 * time.Millisecond),
			Duration:     100 * time.Millisecond,
		},
		{
			ScenarioID:   core.NewScenarioID(),
			ScenarioName: "scenario-2",
			State:        execution.StateFailed,
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(50 * time.Millisecond),
			Duration:     50 * time.Millisecond,
			Error:        errTest,
		},
	}

	result := NewRunResult("test-run", execResults)

	if result.Name != "test-run" {
		t.Errorf("expected name 'test-run', got %s", result.Name)
	}
	if result.ID == "" {
		t.Error("expected ID to be set")
	}
	if len(result.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(result.Scenarios))
	}
	if result.Stats.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Stats.Total)
	}
	if result.Stats.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Stats.Passed)
	}
	if result.Stats.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Stats.Failed)
	}
}

var errTest = &testError{msg: "test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestRunResultIsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		failed   int
		expected bool
	}{
		{"no failures", 0, true},
		{"with failures", 1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := &RunResult{
				Stats: RunStats{Failed: tc.failed},
			}
			if result.IsSuccess() != tc.expected {
				t.Errorf("expected IsSuccess=%v, got %v", tc.expected, result.IsSuccess())
			}
		})
	}
}

func TestRunResultPassRate(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		passed   int
		expected float64
	}{
		{"all passed", 10, 10, 100.0},
		{"half passed", 10, 5, 50.0},
		{"none passed", 10, 0, 0.0},
		{"empty", 0, 0, 0.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := &RunResult{
				Stats: RunStats{Total: tc.total, Passed: tc.passed},
			}
			rate := result.PassRate()
			if rate != tc.expected {
				t.Errorf("expected pass rate %.1f, got %.1f", tc.expected, rate)
			}
		})
	}
}

func TestRunResultJSON(t *testing.T) {
	result := &RunResult{
		ID:   "test-id",
		Name: "test-run",
		Stats: RunStats{
			Total:  2,
			Passed: 1,
			Failed: 1,
		},
	}

	jsonStr, err := result.JSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON")
	}
	if !containsStr(jsonStr, "test-id") {
		t.Error("expected JSON to contain 'test-id'")
	}
	if !containsStr(jsonStr, "test-run") {
		t.Error("expected JSON to contain 'test-run'")
	}
}

func TestRunResultCompactJSON(t *testing.T) {
	result := &RunResult{
		ID:   "test-id",
		Name: "test-run",
	}

	jsonStr, err := result.CompactJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compact JSON should not have newlines in the middle
	if containsStr(jsonStr, "\n  ") {
		t.Error("compact JSON should not have formatted newlines")
	}
}

func TestRunResultFailedScenarios(t *testing.T) {
	result := &RunResult{
		Scenarios: []ScenarioRunResult{
			{ScenarioName: "pass1", State: "completed"},
			{ScenarioName: "fail1", State: "failed"},
			{ScenarioName: "pass2", State: "completed"},
			{ScenarioName: "fail2", State: "failed"},
			{ScenarioName: "skip1", State: "skipped"},
		},
	}

	failed := result.FailedScenarios()

	if len(failed) != 2 {
		t.Errorf("expected 2 failed scenarios, got %d", len(failed))
	}
	if failed[0].ScenarioName != "fail1" {
		t.Errorf("expected 'fail1', got %s", failed[0].ScenarioName)
	}
	if failed[1].ScenarioName != "fail2" {
		t.Errorf("expected 'fail2', got %s", failed[1].ScenarioName)
	}
}

func TestCollector(t *testing.T) {
	collector := NewCollector("test-suite")

	if collector.Count() != 0 {
		t.Errorf("expected count 0, got %d", collector.Count())
	}

	// Add single result
	collector.Add(&execution.ScenarioResult{
		ScenarioID:   core.NewScenarioID(),
		ScenarioName: "scenario-1",
		State:        execution.StateCompleted,
		Duration:     100 * time.Millisecond,
	})

	if collector.Count() != 1 {
		t.Errorf("expected count 1, got %d", collector.Count())
	}

	// Add multiple results
	collector.AddAll([]*execution.ScenarioResult{
		{
			ScenarioID:   core.NewScenarioID(),
			ScenarioName: "scenario-2",
			State:        execution.StateFailed,
			Duration:     50 * time.Millisecond,
			Error:        errTest,
		},
		{
			ScenarioID:   core.NewScenarioID(),
			ScenarioName: "scenario-3",
			State:        execution.StateSkipped,
			SkipReason:   "test skip",
		},
	})

	if collector.Count() != 3 {
		t.Errorf("expected count 3, got %d", collector.Count())
	}
}

func TestCollectorBuild(t *testing.T) {
	collector := NewCollector("my-suite")
	collector.SetConfig(RunConfig{
		Parallelism: 4,
		FailFast:    true,
		Tags:        []string{"integration"},
	})
	collector.SetEnvironment(EnvironmentInfo{
		OS:       "darwin",
		Arch:     "arm64",
		CI:       true,
		CIProvider: "github-actions",
	})

	collector.Add(&execution.ScenarioResult{
		ScenarioID:   core.NewScenarioID(),
		ScenarioName: "test",
		State:        execution.StateCompleted,
		Duration:     100 * time.Millisecond,
	})

	result := collector.Build()

	if result.Name != "my-suite" {
		t.Errorf("expected name 'my-suite', got %s", result.Name)
	}
	if result.Config.Parallelism != 4 {
		t.Errorf("expected parallelism 4, got %d", result.Config.Parallelism)
	}
	if !result.Config.FailFast {
		t.Error("expected FailFast to be true")
	}
	if result.Environment.CI != true {
		t.Error("expected CI to be true")
	}
	if result.Environment.CIProvider != "github-actions" {
		t.Errorf("expected 'github-actions', got %s", result.Environment.CIProvider)
	}
}

func TestConvertScenarioResult(t *testing.T) {
	scenarioID := core.NewScenarioID()
	startTime := time.Now()
	endTime := startTime.Add(100 * time.Millisecond)

	execResult := &execution.ScenarioResult{
		ScenarioID:   scenarioID,
		ScenarioName: "test-scenario",
		State:        execution.StateCompleted,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     100 * time.Millisecond,
		FlowResults: []execution.FlowItemResult{
			{
				Name:     "step-1",
				Type:     core.ComponentSetup,
				State:    execution.StateCompleted,
				Duration: 50 * time.Millisecond,
			},
			{
				Name:     "step-2",
				Type:     core.ComponentTask,
				State:    execution.StateCompleted,
				Duration: 50 * time.Millisecond,
			},
		},
		TeardownResults: []execution.FlowItemResult{
			{
				Name:     "cleanup",
				Type:     core.ComponentTeardown,
				State:    execution.StateCompleted,
				Duration: 10 * time.Millisecond,
			},
		},
	}

	sr := convertScenarioResult(execResult)

	if sr.ScenarioID != scenarioID.String() {
		t.Errorf("expected scenario ID %s, got %s", scenarioID.String(), sr.ScenarioID)
	}
	if sr.ScenarioName != "test-scenario" {
		t.Errorf("expected 'test-scenario', got %s", sr.ScenarioName)
	}
	if sr.State != "completed" {
		t.Errorf("expected 'completed', got %s", sr.State)
	}
	if len(sr.FlowResults) != 2 {
		t.Errorf("expected 2 flow results, got %d", len(sr.FlowResults))
	}
	if len(sr.TeardownResults) != 1 {
		t.Errorf("expected 1 teardown result, got %d", len(sr.TeardownResults))
	}
}

func TestConvertScenarioResultWithError(t *testing.T) {
	execResult := &execution.ScenarioResult{
		ScenarioID:   core.NewScenarioID(),
		ScenarioName: "failed-scenario",
		State:        execution.StateFailed,
		Error:        errTest,
		FlowResults: []execution.FlowItemResult{
			{
				Name:  "failing-step",
				Type:  core.ComponentTask,
				State: execution.StateFailed,
				Error: errTest,
			},
		},
	}

	sr := convertScenarioResult(execResult)

	if sr.Error != "test error" {
		t.Errorf("expected error 'test error', got %s", sr.Error)
	}
	if sr.FlowResults[0].Error != "test error" {
		t.Errorf("expected flow error 'test error', got %s", sr.FlowResults[0].Error)
	}
}

func TestParseState(t *testing.T) {
	tests := []struct {
		input    string
		expected execution.ExecutionState
	}{
		{"not_started", execution.StateNotStarted},
		{"running", execution.StateRunning},
		{"completed", execution.StateCompleted},
		{"failed", execution.StateFailed},
		{"skipped", execution.StateSkipped},
		{"cancelled", execution.StateCancelled},
		{"unknown", execution.StateNotStarted},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseState(tc.input)
			if result != tc.expected {
				t.Errorf("parseState(%s) = %s, want %s", tc.input, result, tc.expected)
			}
		})
	}
}

func TestRunResultSummary(t *testing.T) {
	collector := NewCollector("test")
	collector.Add(&execution.ScenarioResult{
		ScenarioID:   core.NewScenarioID(),
		ScenarioName: "pass",
		State:        execution.StateCompleted,
	})
	collector.Add(&execution.ScenarioResult{
		ScenarioID:   core.NewScenarioID(),
		ScenarioName: "fail",
		State:        execution.StateFailed,
	})

	result := collector.Build()
	summary := result.Summary()

	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if !containsStr(summary, "1 passed") {
		t.Errorf("expected '1 passed' in summary: %s", summary)
	}
	if !containsStr(summary, "1 failed") {
		t.Errorf("expected '1 failed' in summary: %s", summary)
	}
}

func TestRunStatsTimingCalculations(t *testing.T) {
	execResults := []*execution.ScenarioResult{
		{
			ScenarioID:   core.NewScenarioID(),
			ScenarioName: "fast",
			State:        execution.StateCompleted,
			Duration:     10 * time.Millisecond,
		},
		{
			ScenarioID:   core.NewScenarioID(),
			ScenarioName: "medium",
			State:        execution.StateCompleted,
			Duration:     50 * time.Millisecond,
		},
		{
			ScenarioID:   core.NewScenarioID(),
			ScenarioName: "slow",
			State:        execution.StateCompleted,
			Duration:     100 * time.Millisecond,
		},
	}

	result := NewRunResult("timing-test", execResults)

	if result.Stats.MinDuration != 10*time.Millisecond {
		t.Errorf("expected min 10ms, got %v", result.Stats.MinDuration)
	}
	if result.Stats.MaxDuration != 100*time.Millisecond {
		t.Errorf("expected max 100ms, got %v", result.Stats.MaxDuration)
	}
	// Average of 10, 50, 100 = 53.33ms (rounded to 53ms due to integer division)
	expectedAvg := 160 * time.Millisecond / 3
	if result.Stats.AvgDuration != expectedAvg {
		t.Errorf("expected avg %v, got %v", expectedAvg, result.Stats.AvgDuration)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && findStr(s, substr) >= 0
}

func findStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
