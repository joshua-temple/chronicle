package intelligence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFlakyStatus_String(t *testing.T) {
	tests := []struct {
		status   FlakyStatus
		expected string
	}{
		{FlakyStatusStable, "stable"},
		{FlakyStatusSuspected, "suspected"},
		{FlakyStatusConfirmed, "confirmed"},
		{FlakyStatusQuarantined, "quarantined"},
		{FlakyStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("FlakyStatus.String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestNewFlakyDetector(t *testing.T) {
	// Create temp directory for storage
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              10,
		FlipThreshold:           2,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        5,
		QuarantineAfterFailures: 3,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	if fd == nil {
		t.Fatal("NewFlakyDetector() returned nil")
	}

	if fd.history == nil {
		t.Error("NewFlakyDetector() did not initialize history map")
	}
}

func TestFlakyDetector_RecordOutcome(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              10,
		FlipThreshold:           2,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        5,
		QuarantineAfterFailures: 3,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Record some outcomes
	fd.RecordOutcome("test-1", true, 100*time.Millisecond, nil, "run-1")
	fd.RecordOutcome("test-1", false, 50*time.Millisecond, nil, "run-2")
	fd.RecordOutcome("test-1", true, 75*time.Millisecond, nil, "run-3")

	history, exists := fd.GetHistory("test-1")
	if !exists {
		t.Fatal("GetHistory() did not find recorded test")
	}

	if len(history.Outcomes) != 3 {
		t.Errorf("Expected 3 outcomes, got %d", len(history.Outcomes))
	}

	if history.ScenarioName != "test-1" {
		t.Errorf("ScenarioName = %q, expected 'test-1'", history.ScenarioName)
	}
}

func TestFlakyDetector_RecordOutcome_WindowSize(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 3,
		WindowSize:              5,
		FlipThreshold:           2,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        5,
		QuarantineAfterFailures: 10,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Record more outcomes than window size
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("test-1", i%2 == 0, 100*time.Millisecond, nil, "")
	}

	history, _ := fd.GetHistory("test-1")
	if len(history.Outcomes) != 5 {
		t.Errorf("Expected 5 outcomes (windowSize), got %d", len(history.Outcomes))
	}
}

func TestFlakyDetector_AnalyzeTest_Stable(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 5,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Record all passing tests - should be stable
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("stable-test", true, 100*time.Millisecond, nil, "")
	}

	history, _ := fd.GetHistory("stable-test")
	if history.Status != FlakyStatusStable {
		t.Errorf("Expected stable status, got %v", history.Status)
	}

	if history.FlakyScore > 0.1 {
		t.Errorf("Expected low flaky score for stable test, got %f", history.FlakyScore)
	}
}

func TestFlakyDetector_AnalyzeTest_Flaky(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 100, // High to avoid quarantine
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Record alternating pass/fail - highly flaky
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("flaky-test", i%2 == 0, 100*time.Millisecond, nil, "")
	}

	history, _ := fd.GetHistory("flaky-test")
	if history.Status != FlakyStatusConfirmed && history.Status != FlakyStatusSuspected {
		t.Errorf("Expected suspected/confirmed flaky status, got %v", history.Status)
	}

	// Flaky score should be high for alternating tests
	if history.FlakyScore < 0.3 {
		t.Errorf("Expected high flaky score, got %f", history.FlakyScore)
	}
}

func TestFlakyDetector_AnalyzeTest_Quarantine(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 3, // Low threshold for quarantine
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Record consecutive failures
	for i := 0; i < 5; i++ {
		fd.RecordOutcome("failing-test", false, 100*time.Millisecond, nil, "")
	}

	history, _ := fd.GetHistory("failing-test")
	if history.Status != FlakyStatusQuarantined {
		t.Errorf("Expected quarantined status, got %v", history.Status)
	}
}

func TestFlakyDetector_GetFlakyTests(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 100,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Create a stable test
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("stable-test", true, 100*time.Millisecond, nil, "")
	}

	// Create a flaky test
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("flaky-test", i%2 == 0, 100*time.Millisecond, nil, "")
	}

	flakyTests := fd.GetFlakyTests()

	// Should only contain the flaky test
	if len(flakyTests) != 1 {
		t.Errorf("Expected 1 flaky test, got %d", len(flakyTests))
	}

	if len(flakyTests) > 0 && flakyTests[0].ScenarioName != "flaky-test" {
		t.Errorf("Expected flaky-test, got %s", flakyTests[0].ScenarioName)
	}
}

func TestFlakyDetector_GetQuarantinedTests(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 3,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Create a test that gets quarantined
	for i := 0; i < 5; i++ {
		fd.RecordOutcome("quarantined-test", false, 100*time.Millisecond, nil, "")
	}

	// Create a stable test
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("stable-test", true, 100*time.Millisecond, nil, "")
	}

	quarantined := fd.GetQuarantinedTests()

	if len(quarantined) != 1 {
		t.Errorf("Expected 1 quarantined test, got %d", len(quarantined))
	}

	if len(quarantined) > 0 && quarantined[0].ScenarioName != "quarantined-test" {
		t.Errorf("Expected quarantined-test, got %s", quarantined[0].ScenarioName)
	}
}

func TestFlakyDetector_IsQuarantined(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 3,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 3,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Quarantine a test
	for i := 0; i < 5; i++ {
		fd.RecordOutcome("quarantined-test", false, 100*time.Millisecond, nil, "")
	}

	if !fd.IsQuarantined("quarantined-test") {
		t.Error("IsQuarantined() should return true for quarantined test")
	}

	if fd.IsQuarantined("non-existent") {
		t.Error("IsQuarantined() should return false for non-existent test")
	}
}

func TestFlakyDetector_Unquarantine(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 3,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 3,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Quarantine a test
	for i := 0; i < 5; i++ {
		fd.RecordOutcome("test-1", false, 100*time.Millisecond, nil, "")
	}

	if !fd.IsQuarantined("test-1") {
		t.Fatal("Test should be quarantined")
	}

	// Unquarantine
	fd.Unquarantine("test-1")

	if fd.IsQuarantined("test-1") {
		t.Error("Test should not be quarantined after Unquarantine()")
	}

	// History should be cleared
	history, _ := fd.GetHistory("test-1")
	if len(history.Outcomes) != 0 {
		t.Error("Unquarantine() should clear history")
	}

	if history.Status != FlakyStatusStable {
		t.Error("Unquarantine() should reset status to stable")
	}
}

func TestFlakyDetector_GenerateReport(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 3,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Create stable test
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("stable-test", true, 100*time.Millisecond, nil, "")
	}

	// Create flaky test
	for i := 0; i < 10; i++ {
		fd.RecordOutcome("flaky-test", i%2 == 0, 100*time.Millisecond, nil, "")
	}

	// Create quarantined test
	for i := 0; i < 5; i++ {
		fd.RecordOutcome("quarantined-test", false, 100*time.Millisecond, nil, "")
	}

	report := fd.GenerateReport(context.Background())

	if report == nil {
		t.Fatal("GenerateReport() returned nil")
	}

	if report.TotalScenarios != 3 {
		t.Errorf("TotalScenarios = %d, expected 3", report.TotalScenarios)
	}

	if report.StableScenarios != 1 {
		t.Errorf("StableScenarios = %d, expected 1", report.StableScenarios)
	}

	if report.QuarantinedTests != 1 {
		t.Errorf("QuarantinedTests = %d, expected 1", report.QuarantinedTests)
	}

	// Should have recommendations
	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations in report")
	}

	// FlakyTests should be sorted by score descending
	if len(report.FlakyTests) > 1 {
		for i := 1; i < len(report.FlakyTests); i++ {
			if report.FlakyTests[i].FlakyScore > report.FlakyTests[i-1].FlakyScore {
				t.Error("FlakyTests not sorted by score descending")
			}
		}
	}
}

func TestFlakyDetector_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 3,
		WindowSize:              10,
		FlipThreshold:           2,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        5,
		QuarantineAfterFailures: 10,
		StoragePath:             tmpDir,
	}

	// Create detector and add data
	fd1 := NewFlakyDetector(cfg)
	for i := 0; i < 5; i++ {
		fd1.RecordOutcome("test-1", i%2 == 0, 100*time.Millisecond, nil, "")
	}

	// Check file was created
	historyPath := filepath.Join(tmpDir, "history.json")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Fatal("History file was not created")
	}

	// Create new detector and verify data was loaded
	fd2 := NewFlakyDetector(cfg)
	history, exists := fd2.GetHistory("test-1")
	if !exists {
		t.Fatal("History was not loaded from persistence")
	}

	if len(history.Outcomes) != 5 {
		t.Errorf("Expected 5 outcomes from persistence, got %d", len(history.Outcomes))
	}
}

func TestFlakyDetector_MinRunsRequired(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 100,
		StoragePath:             tmpDir,
	}

	fd := NewFlakyDetector(cfg)

	// Record fewer runs than MinRuns
	for i := 0; i < 3; i++ {
		fd.RecordOutcome("test-1", i%2 == 0, 100*time.Millisecond, nil, "")
	}

	history, _ := fd.GetHistory("test-1")

	// Should be stable with 0 flaky score because not enough runs
	if history.Status != FlakyStatusStable {
		t.Errorf("Expected stable status with fewer than MinRuns, got %v", history.Status)
	}

	if history.FlakyScore != 0 {
		t.Errorf("Expected 0 flaky score with fewer than MinRuns, got %f", history.FlakyScore)
	}
}

