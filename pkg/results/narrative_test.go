package results

import (
	"testing"
	"time"
)

func TestNarrativeGeneratorNew(t *testing.T) {
	ng := NewNarrativeGenerator()

	if ng.style != StyleStandard {
		t.Errorf("expected StyleStandard, got %v", ng.style)
	}
	if !ng.showTiming {
		t.Error("expected showTiming to be true by default")
	}
	if ng.showLogs {
		t.Error("expected showLogs to be false by default")
	}
	if ng.maxErrors != 5 {
		t.Errorf("expected maxErrors 5, got %d", ng.maxErrors)
	}
}

func TestNarrativeGeneratorWithOptions(t *testing.T) {
	ng := NewNarrativeGenerator(
		WithStyle(StyleVerbose),
		WithTiming(false),
		WithLogs(true),
		WithMaxErrors(10),
	)

	if ng.style != StyleVerbose {
		t.Errorf("expected StyleVerbose, got %v", ng.style)
	}
	if ng.showTiming {
		t.Error("expected showTiming to be false")
	}
	if !ng.showLogs {
		t.Error("expected showLogs to be true")
	}
	if ng.maxErrors != 10 {
		t.Errorf("expected maxErrors 10, got %d", ng.maxErrors)
	}
}

func TestGenerateBriefNarrative(t *testing.T) {
	result := createTestResult()

	narrative := GenerateBriefNarrative(result)

	if narrative == "" {
		t.Error("expected non-empty narrative")
	}
	if !containsStr(narrative, "PASS") {
		t.Errorf("expected PASS in brief narrative: %s", narrative)
	}
	if !containsStr(narrative, "2/2") {
		t.Errorf("expected 2/2 in brief narrative: %s", narrative)
	}
}

func TestGenerateNarrative(t *testing.T) {
	result := createTestResult()

	narrative := GenerateNarrative(result)

	if narrative == "" {
		t.Error("expected non-empty narrative")
	}
	if !containsStr(narrative, "Chronicle Run") {
		t.Errorf("expected 'Chronicle Run' in narrative: %s", narrative)
	}
	if !containsStr(narrative, "Status:") {
		t.Errorf("expected 'Status:' in narrative: %s", narrative)
	}
}

func TestGenerateMarkdownReport(t *testing.T) {
	result := createTestResult()

	report := GenerateMarkdownReport(result)

	if report == "" {
		t.Error("expected non-empty report")
	}
	if !containsStr(report, "# Chronicle Run") {
		t.Errorf("expected markdown header in report: %s", report)
	}
	if !containsStr(report, "## Summary") {
		t.Errorf("expected Summary section in report: %s", report)
	}
	if !containsStr(report, "| Metric | Value |") {
		t.Errorf("expected markdown table in report: %s", report)
	}
}

func TestNarrativeWithFailedScenarios(t *testing.T) {
	result := &RunResult{
		Name:      "failed-run",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Stats: RunStats{
			Total:  2,
			Passed: 1,
			Failed: 1,
		},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "passing-scenario",
				State:        "completed",
				Duration:     50 * time.Millisecond,
			},
			{
				ScenarioName: "failing-scenario",
				State:        "failed",
				Error:        "something went wrong",
				Duration:     50 * time.Millisecond,
				FlowResults: []FlowItemRunResult{
					{
						Name:     "setup",
						Type:     "setup",
						State:    "completed",
						Duration: 10 * time.Millisecond,
					},
					{
						Name:     "action",
						Type:     "task",
						State:    "failed",
						Error:    "action failed",
						Duration: 40 * time.Millisecond,
					},
				},
			},
		},
	}

	narrative := GenerateNarrative(result)

	if !containsStr(narrative, "FAIL") {
		t.Errorf("expected FAIL in narrative: %s", narrative)
	}
	if !containsStr(narrative, "Failed Scenarios:") {
		t.Errorf("expected 'Failed Scenarios:' in narrative: %s", narrative)
	}
	if !containsStr(narrative, "failing-scenario") {
		t.Errorf("expected 'failing-scenario' in narrative: %s", narrative)
	}
}

func TestNarrativeWithSkippedScenarios(t *testing.T) {
	result := &RunResult{
		Name:     "skip-run",
		Duration: 50 * time.Millisecond,
		Stats: RunStats{
			Total:   2,
			Passed:  1,
			Skipped: 1,
		},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "passing-scenario",
				State:        "completed",
				Duration:     50 * time.Millisecond,
			},
			{
				ScenarioName: "skipped-scenario",
				State:        "skipped",
				SkipReason:   "not supported on this OS",
			},
		},
	}

	narrative := GenerateNarrative(result)

	if !containsStr(narrative, "Skipped Scenarios:") {
		t.Errorf("expected 'Skipped Scenarios:' in narrative: %s", narrative)
	}
	if !containsStr(narrative, "not supported on this OS") {
		t.Errorf("expected skip reason in narrative: %s", narrative)
	}
}

func TestVerboseNarrative(t *testing.T) {
	result := &RunResult{
		ID:        "run-123",
		Name:      "verbose-run",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Config: RunConfig{
			Parallelism: 4,
			FailFast:    true,
			Tags:        []string{"integration", "slow"},
		},
		Environment: EnvironmentInfo{
			Hostname:  "test-host",
			OS:        "darwin",
			Arch:      "arm64",
			GoVersion: "go1.21.0",
			CI:        true,
			CIProvider: "github-actions",
			Branch:    "main",
			Commit:    "abc123",
		},
		Stats: RunStats{
			Total:       1,
			Passed:      1,
			AvgDuration: 100 * time.Millisecond,
		},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "test-scenario",
				State:        "completed",
				Duration:     100 * time.Millisecond,
				FlowResults: []FlowItemRunResult{
					{
						Name:     "step-1",
						Type:     "setup",
						State:    "completed",
						Duration: 50 * time.Millisecond,
					},
					{
						Name:     "step-2",
						Type:     "task",
						State:    "completed",
						Duration: 50 * time.Millisecond,
					},
				},
				TeardownResults: []FlowItemRunResult{
					{
						Name:     "cleanup",
						Type:     "teardown",
						State:    "completed",
						Duration: 10 * time.Millisecond,
					},
				},
			},
		},
	}

	ng := NewNarrativeGenerator(WithStyle(StyleVerbose))
	narrative := ng.Generate(result)

	if !containsStr(narrative, "Chronicle Run Report") {
		t.Errorf("expected 'Chronicle Run Report' in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "run-123") {
		t.Errorf("expected run ID in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "Timing:") {
		t.Errorf("expected 'Timing:' in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "Configuration:") {
		t.Errorf("expected 'Configuration:' in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "Parallelism: 4") {
		t.Errorf("expected parallelism in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "Environment:") {
		t.Errorf("expected 'Environment:' in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "github-actions") {
		t.Errorf("expected CI provider in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "Scenario Details:") {
		t.Errorf("expected 'Scenario Details:' in verbose narrative: %s", narrative)
	}
	if !containsStr(narrative, "Teardown:") {
		t.Errorf("expected 'Teardown:' in verbose narrative: %s", narrative)
	}
}

func TestVerboseNarrativeWithLogs(t *testing.T) {
	result := &RunResult{
		Name:     "log-run",
		Duration: 100 * time.Millisecond,
		Stats:    RunStats{Total: 1, Passed: 1},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "test",
				State:        "completed",
				Duration:     100 * time.Millisecond,
				Logs: []LogEntry{
					{Level: "INFO", Component: "setup", Message: "Starting setup"},
					{Level: "DEBUG", Component: "task", Message: "Processing data"},
				},
			},
		},
	}

	ng := NewNarrativeGenerator(WithStyle(StyleVerbose), WithLogs(true))
	narrative := ng.Generate(result)

	if !containsStr(narrative, "Logs:") {
		t.Errorf("expected 'Logs:' in verbose narrative with logs: %s", narrative)
	}
	if !containsStr(narrative, "Starting setup") {
		t.Errorf("expected log message in narrative: %s", narrative)
	}
}

func TestMarkdownNarrativeWithFailures(t *testing.T) {
	result := &RunResult{
		Name:     "md-fail-run",
		EndTime:  time.Now(),
		Duration: 100 * time.Millisecond,
		Stats: RunStats{
			Total:  2,
			Passed: 1,
			Failed: 1,
		},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "passing",
				State:        "completed",
				Duration:     50 * time.Millisecond,
			},
			{
				ScenarioName: "failing",
				State:        "failed",
				Error:        "validation error",
				Duration:     50 * time.Millisecond,
				FlowResults: []FlowItemRunResult{
					{
						Name:  "validate",
						Type:  "assertion",
						State: "failed",
						Error: "expected X got Y",
					},
				},
			},
		},
	}

	report := GenerateMarkdownReport(result)

	if !containsStr(report, "status-FAIL-critical") {
		t.Errorf("expected FAIL badge in markdown: %s", report)
	}
	if !containsStr(report, "## Failed Scenarios") {
		t.Errorf("expected '## Failed Scenarios' in markdown: %s", report)
	}
	if !containsStr(report, "### ❌ failing") {
		t.Errorf("expected failed scenario heading in markdown: %s", report)
	}
}

func TestMarkdownNarrativeWithSkipped(t *testing.T) {
	result := &RunResult{
		Name:     "md-skip-run",
		EndTime:  time.Now(),
		Duration: 50 * time.Millisecond,
		Stats: RunStats{
			Total:   2,
			Passed:  1,
			Skipped: 1,
		},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "passing",
				State:        "completed",
				Duration:     50 * time.Millisecond,
			},
			{
				ScenarioName: "skipped",
				State:        "skipped",
				SkipReason:   "feature flag disabled",
			},
		},
	}

	report := GenerateMarkdownReport(result)

	if !containsStr(report, "## Skipped Scenarios") {
		t.Errorf("expected '## Skipped Scenarios' in markdown: %s", report)
	}
	if !containsStr(report, "feature flag disabled") {
		t.Errorf("expected skip reason in markdown: %s", report)
	}
}

func TestMaxErrorsLimit(t *testing.T) {
	// Create result with many failures
	scenarios := make([]ScenarioRunResult, 10)
	for i := range scenarios {
		scenarios[i] = ScenarioRunResult{
			ScenarioName: "fail-" + string(rune('a'+i)),
			State:        "failed",
			Error:        "error message",
		}
	}

	result := &RunResult{
		Name:      "many-failures",
		Duration:  100 * time.Millisecond,
		Stats:     RunStats{Total: 10, Failed: 10},
		Scenarios: scenarios,
	}

	ng := NewNarrativeGenerator(WithMaxErrors(3))
	narrative := ng.Generate(result)

	if !containsStr(narrative, "... and 7 more failures") {
		t.Errorf("expected truncation message in narrative: %s", narrative)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a very long string that needs truncation", 20, "this is a very lo..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := truncate(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

func TestStatusEmoji(t *testing.T) {
	if statusEmoji(true) != "✓ PASS" {
		t.Errorf("expected '✓ PASS' for success, got %s", statusEmoji(true))
	}
	if statusEmoji(false) != "✗ FAIL" {
		t.Errorf("expected '✗ FAIL' for failure, got %s", statusEmoji(false))
	}
}

func createTestResult() *RunResult {
	return &RunResult{
		ID:        "test-id",
		Name:      "test-run",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Stats: RunStats{
			Total:       2,
			Passed:      2,
			Failed:      0,
			Skipped:     0,
			AvgDuration: 50 * time.Millisecond,
		},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "scenario-1",
				State:        "completed",
				Duration:     50 * time.Millisecond,
			},
			{
				ScenarioName: "scenario-2",
				State:        "completed",
				Duration:     50 * time.Millisecond,
			},
		},
	}
}
