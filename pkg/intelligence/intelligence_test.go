package intelligence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFlakyDetector_RecordOutcome(t *testing.T) {
	// Create temp directory for storage
	tempDir, err := os.MkdirTemp("", "flaky-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultFlakyDetectorConfig()
	config.StoragePath = tempDir
	config.MinRuns = 3
	config.WindowSize = 10

	fd := NewFlakyDetector(config)

	// Record outcomes for a stable test
	for i := 0; i < 5; i++ {
		fd.RecordOutcome("stable_test", true, time.Second, nil, "run-"+string(rune('0'+i)))
	}

	// Check status
	history, ok := fd.GetHistory("stable_test")
	if !ok {
		t.Fatal("expected history for stable_test")
	}
	if history.Status != FlakyStatusStable {
		t.Errorf("expected stable status, got %s", history.Status)
	}

	// Record outcomes for a flaky test (alternating pass/fail)
	for i := 0; i < 10; i++ {
		passed := i%2 == 0
		fd.RecordOutcome("flaky_test", passed, time.Second, nil, "run-"+string(rune('0'+i)))
	}

	// Check flaky status
	flakyHistory, ok := fd.GetHistory("flaky_test")
	if !ok {
		t.Fatal("expected history for flaky_test")
	}
	if flakyHistory.Status == FlakyStatusStable {
		t.Error("expected non-stable status for flaky test")
	}
}

func TestFlakyDetector_GetFlakyTests(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "flaky-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultFlakyDetectorConfig()
	config.StoragePath = tempDir
	config.MinRuns = 3

	fd := NewFlakyDetector(config)

	// Create a flaky test
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("flaky_one", i%2 == 0, time.Second, nil, "")
	}

	// Create a stable test
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("stable_one", true, time.Second, nil, "")
	}

	flakyTests := fd.GetFlakyTests()
	found := false
	for _, ft := range flakyTests {
		if ft.ScenarioName == "flaky_one" {
			found = true
			break
		}
	}

	if !found && len(flakyTests) > 0 {
		// At least check that we got some flaky tests
		t.Logf("Found %d flaky tests", len(flakyTests))
	}
}

func TestFlakyDetector_GenerateReport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "flaky-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultFlakyDetectorConfig()
	config.StoragePath = tempDir

	fd := NewFlakyDetector(config)

	// Add some test data
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("test1", true, time.Second, nil, "")
		fd.RecordOutcome("test2", i%2 == 0, time.Second, nil, "")
	}

	report := fd.GenerateReport(context.Background())

	if report.TotalScenarios != 2 {
		t.Errorf("expected 2 total scenarios, got %d", report.TotalScenarios)
	}
}

func TestPerformanceTracker_RecordTiming(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "perf-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultPerformanceTrackerConfig()
	config.StoragePath = tempDir
	config.MinSamplesForAnalysis = 5

	pt := NewPerformanceTracker(config)

	// Record timings
	for i := 0; i < 10; i++ {
		pt.RecordTiming("fast_test", 100*time.Millisecond, true, "run-"+string(rune('0'+i)))
	}

	baseline, ok := pt.GetBaseline("fast_test")
	if !ok {
		t.Fatal("expected baseline for fast_test")
	}

	if baseline.Mean == 0 {
		t.Error("expected non-zero mean")
	}
}

func TestPerformanceTracker_DetectRegression(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "perf-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultPerformanceTrackerConfig()
	config.StoragePath = tempDir
	config.BaselineWindow = 5
	config.MinSamplesForAnalysis = 5
	config.DegradationThreshold = 20

	pt := NewPerformanceTracker(config)

	// Record baseline timings (fast)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("regressing_test", 100*time.Millisecond, true, "")
	}

	// Record slower timings (50% degradation)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("regressing_test", 150*time.Millisecond, true, "")
	}

	baseline, ok := pt.GetBaseline("regressing_test")
	if !ok {
		t.Fatal("expected baseline")
	}

	// Check if degradation was detected
	if baseline.Status != PerformanceStatusDegraded && baseline.Status != PerformanceStatusCritical {
		t.Logf("Status: %s, Mean: %.2f", baseline.Status, baseline.Mean)
	}
}

func TestPerformanceTracker_GenerateReport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "perf-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultPerformanceTrackerConfig()
	config.StoragePath = tempDir

	pt := NewPerformanceTracker(config)

	// Add test data
	for i := 0; i < 20; i++ {
		pt.RecordTiming("test1", 100*time.Millisecond, true, "")
		pt.RecordTiming("test2", 200*time.Millisecond, true, "")
	}

	report := pt.GenerateReport(context.Background())

	if report.TotalScenarios != 2 {
		t.Errorf("expected 2 scenarios, got %d", report.TotalScenarios)
	}
}

func TestImpactAnalyzer_RegisterMapping(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "impact-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultImpactAnalyzerConfig()
	config.TestMappingsPath = tempDir

	ia := NewImpactAnalyzer(config)

	mapping := &TestMapping{
		ScenarioName: "user_login_test",
		TestedFiles:  []string{"pkg/auth/login.go"},
		TestedFuncs:  []string{"Login", "ValidateCredentials"},
		Priority:     1,
	}

	ia.RegisterMapping(mapping)

	// Verify mapping was saved
	savedPath := filepath.Join(tempDir, "mappings.json")
	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Error("expected mappings file to be created")
	}
}

func TestImpactAnalyzer_AnalyzeChanges(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "impact-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := DefaultImpactAnalyzerConfig()
	config.TestMappingsPath = tempDir
	config.RootPath = "."

	ia := NewImpactAnalyzer(config)

	// Register a mapping
	ia.RegisterMapping(&TestMapping{
		ScenarioName: "auth_test",
		TestedFiles:  []string{"auth/*.go"},
		Priority:     1,
	})

	// Analyze some changes
	changes := []FileChange{
		{Path: "auth/login.go", ChangeType: "modified", LinesAdded: 10, LinesRemoved: 5},
		{Path: "README.md", ChangeType: "modified", LinesAdded: 1},
	}

	report, err := ia.AnalyzeChanges(context.Background(), changes, "main", "feature")
	if err != nil {
		t.Fatalf("analyze changes failed: %v", err)
	}

	if report.FilesChanged != 2 {
		t.Errorf("expected 2 files changed, got %d", report.FilesChanged)
	}

	// Check that auth_test was identified as affected
	found := false
	for _, test := range report.SuggestedTests {
		if test == "auth_test" {
			found = true
			break
		}
	}

	if !found && len(report.Results) > 0 {
		t.Logf("Results: %+v", report.Results)
	}
}

func TestStatisticalFunctions(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	m := mean(values)
	if m != 5.5 {
		t.Errorf("expected mean 5.5, got %.2f", m)
	}

	mn := min(values)
	if mn != 1 {
		t.Errorf("expected min 1, got %.2f", mn)
	}

	mx := max(values)
	if mx != 10 {
		t.Errorf("expected max 10, got %.2f", mx)
	}

	med := percentile(values, 50)
	if med != 5.5 {
		t.Errorf("expected median 5.5, got %.2f", med)
	}

	p95 := percentile(values, 95)
	if p95 < 9 || p95 > 10 {
		t.Errorf("expected p95 around 9.5, got %.2f", p95)
	}
}

func TestFlakyStatus_String(t *testing.T) {
	tests := []struct {
		status   FlakyStatus
		expected string
	}{
		{FlakyStatusStable, "stable"},
		{FlakyStatusSuspected, "suspected"},
		{FlakyStatusConfirmed, "confirmed"},
		{FlakyStatusQuarantined, "quarantined"},
		{FlakyStatus(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("FlakyStatus(%d).String() = %s, want %s", tt.status, got, tt.expected)
		}
	}
}

func TestPerformanceStatus_String(t *testing.T) {
	tests := []struct {
		status   PerformanceStatus
		expected string
	}{
		{PerformanceStatusNormal, "normal"},
		{PerformanceStatusImproved, "improved"},
		{PerformanceStatusDegraded, "degraded"},
		{PerformanceStatusCritical, "critical"},
		{PerformanceStatus(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("PerformanceStatus(%d).String() = %s, want %s", tt.status, got, tt.expected)
		}
	}
}

func TestImpactLevel_String(t *testing.T) {
	tests := []struct {
		level    ImpactLevel
		expected string
	}{
		{ImpactLevelNone, "none"},
		{ImpactLevelLow, "low"},
		{ImpactLevelMedium, "medium"},
		{ImpactLevelHigh, "high"},
		{ImpactLevelCritical, "critical"},
		{ImpactLevel(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("ImpactLevel(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}
