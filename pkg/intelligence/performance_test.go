package intelligence

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestPerformanceStatus_String(t *testing.T) {
	tests := []struct {
		status   PerformanceStatus
		expected string
	}{
		{PerformanceStatusNormal, "normal"},
		{PerformanceStatusImproved, "improved"},
		{PerformanceStatusDegraded, "degraded"},
		{PerformanceStatusCritical, "critical"},
		{PerformanceStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("PerformanceStatus.String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestNewPerformanceTracker(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	if pt == nil {
		t.Fatal("NewPerformanceTracker() returned nil")
	}

	if pt.baselines == nil {
		t.Error("NewPerformanceTracker() did not initialize baselines map")
	}
}

func TestPerformanceTracker_RecordTiming(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        5,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 3,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Record some timings
	pt.RecordTiming("test-1", 100*time.Millisecond, true, "run-1")
	pt.RecordTiming("test-1", 120*time.Millisecond, true, "run-2")
	pt.RecordTiming("test-1", 110*time.Millisecond, true, "run-3")

	baseline, exists := pt.GetBaseline("test-1")
	if !exists {
		t.Fatal("GetBaseline() did not find recorded test")
	}

	if len(baseline.Samples) != 3 {
		t.Errorf("Expected 3 samples, got %d", len(baseline.Samples))
	}

	if baseline.ScenarioName != "test-1" {
		t.Errorf("ScenarioName = %q, expected 'test-1'", baseline.ScenarioName)
	}
}

func TestPerformanceTracker_RecordTiming_WindowSize(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            5,
		BaselineWindow:        3,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 3,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Record more samples than window size
	for i := 0; i < 10; i++ {
		pt.RecordTiming("test-1", time.Duration(100+i)*time.Millisecond, true, "")
	}

	baseline, _ := pt.GetBaseline("test-1")
	if len(baseline.Samples) != 5 {
		t.Errorf("Expected 5 samples (windowSize), got %d", len(baseline.Samples))
	}
}

func TestPerformanceTracker_AnalyzeBaseline_Normal(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Record consistent timings
	for i := 0; i < 20; i++ {
		pt.RecordTiming("stable-test", 100*time.Millisecond, true, "")
	}

	baseline, _ := pt.GetBaseline("stable-test")

	if baseline.Status != PerformanceStatusNormal {
		t.Errorf("Expected normal status, got %v", baseline.Status)
	}
}

func TestPerformanceTracker_AnalyzeBaseline_Degraded(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Record baseline timings (100ms)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("degraded-test", 100*time.Millisecond, true, "")
	}

	// Record degraded timings (130ms = 30% slower)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("degraded-test", 130*time.Millisecond, true, "")
	}

	baseline, _ := pt.GetBaseline("degraded-test")

	if baseline.Status != PerformanceStatusDegraded {
		t.Errorf("Expected degraded status, got %v", baseline.Status)
	}
}

func TestPerformanceTracker_AnalyzeBaseline_Critical(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Record baseline timings (100ms)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("critical-test", 100*time.Millisecond, true, "")
	}

	// Record critical timings (160ms = 60% slower)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("critical-test", 160*time.Millisecond, true, "")
	}

	baseline, _ := pt.GetBaseline("critical-test")

	if baseline.Status != PerformanceStatusCritical {
		t.Errorf("Expected critical status, got %v", baseline.Status)
	}
}

func TestPerformanceTracker_AnalyzeBaseline_Improved(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Record baseline timings (100ms)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("improved-test", 100*time.Millisecond, true, "")
	}

	// Record improved timings (80ms = 20% faster)
	for i := 0; i < 10; i++ {
		pt.RecordTiming("improved-test", 80*time.Millisecond, true, "")
	}

	baseline, _ := pt.GetBaseline("improved-test")

	if baseline.Status != PerformanceStatusImproved {
		t.Errorf("Expected improved status, got %v", baseline.Status)
	}
}

func TestPerformanceTracker_GetRegressions(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Create normal test
	for i := 0; i < 20; i++ {
		pt.RecordTiming("normal-test", 100*time.Millisecond, true, "")
	}

	// Create degraded test
	for i := 0; i < 10; i++ {
		pt.RecordTiming("degraded-test", 100*time.Millisecond, true, "")
	}
	for i := 0; i < 10; i++ {
		pt.RecordTiming("degraded-test", 130*time.Millisecond, true, "")
	}

	regressions := pt.GetRegressions()

	if len(regressions) != 1 {
		t.Errorf("Expected 1 regression, got %d", len(regressions))
	}

	if len(regressions) > 0 && regressions[0].ScenarioName != "degraded-test" {
		t.Errorf("Expected degraded-test, got %s", regressions[0].ScenarioName)
	}
}

func TestPerformanceTracker_GenerateReport(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		DegradationThreshold:  20.0,
		CriticalThreshold:     50.0,
		ImprovementThreshold:  15.0,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Create various tests
	for i := 0; i < 20; i++ {
		pt.RecordTiming("normal-test", 100*time.Millisecond, true, "")
	}

	// Degraded test
	for i := 0; i < 10; i++ {
		pt.RecordTiming("degraded-test", 100*time.Millisecond, true, "")
	}
	for i := 0; i < 10; i++ {
		pt.RecordTiming("degraded-test", 130*time.Millisecond, true, "")
	}

	// Improved test
	for i := 0; i < 10; i++ {
		pt.RecordTiming("improved-test", 100*time.Millisecond, true, "")
	}
	for i := 0; i < 10; i++ {
		pt.RecordTiming("improved-test", 80*time.Millisecond, true, "")
	}

	report := pt.GenerateReport(context.Background())

	if report == nil {
		t.Fatal("GenerateReport() returned nil")
	}

	if report.TotalScenarios != 3 {
		t.Errorf("TotalScenarios = %d, expected 3", report.TotalScenarios)
	}

	if report.NormalCount != 1 {
		t.Errorf("NormalCount = %d, expected 1", report.NormalCount)
	}

	if report.DegradedCount != 1 {
		t.Errorf("DegradedCount = %d, expected 1", report.DegradedCount)
	}

	if report.ImprovedCount != 1 {
		t.Errorf("ImprovedCount = %d, expected 1", report.ImprovedCount)
	}

	// Should have recommendations for degraded test
	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations in report")
	}
}

func TestPerformanceTracker_ResetBaseline(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		MinSamplesForAnalysis: 5,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Add samples
	for i := 0; i < 10; i++ {
		pt.RecordTiming("test-1", 100*time.Millisecond, true, "")
	}

	// Reset baseline
	pt.ResetBaseline("test-1")

	baseline, exists := pt.GetBaseline("test-1")
	if !exists {
		t.Fatal("Baseline should still exist after reset")
	}

	if len(baseline.Samples) != 0 {
		t.Errorf("Samples should be empty after reset, got %d", len(baseline.Samples))
	}

	if baseline.Status != PerformanceStatusNormal {
		t.Errorf("Status should be normal after reset, got %v", baseline.Status)
	}
}

func TestPerformanceTracker_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        10,
		MinSamplesForAnalysis: 5,
		StoragePath:           filepath.Join(tmpDir, "perf"),
	}

	// Create tracker and add data
	pt1 := NewPerformanceTracker(cfg)
	for i := 0; i < 5; i++ {
		pt1.RecordTiming("test-1", 100*time.Millisecond, true, "")
	}

	// Create new tracker and verify data was loaded
	pt2 := NewPerformanceTracker(cfg)
	baseline, exists := pt2.GetBaseline("test-1")
	if !exists {
		t.Fatal("Baseline was not loaded from persistence")
	}

	if len(baseline.Samples) != 5 {
		t.Errorf("Expected 5 samples from persistence, got %d", len(baseline.Samples))
	}
}

func TestPerformanceTracker_FailedTestsExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := PerformanceTrackerConfig{
		WindowSize:            50,
		BaselineWindow:        5,
		MinSamplesForAnalysis: 3,
		StoragePath:           tmpDir,
	}

	pt := NewPerformanceTracker(cfg)

	// Record mix of passing and failing tests
	pt.RecordTiming("test-1", 100*time.Millisecond, true, "")
	pt.RecordTiming("test-1", 500*time.Millisecond, false, "") // Failed - should be excluded
	pt.RecordTiming("test-1", 110*time.Millisecond, true, "")
	pt.RecordTiming("test-1", 600*time.Millisecond, false, "") // Failed - should be excluded
	pt.RecordTiming("test-1", 105*time.Millisecond, true, "")

	baseline, _ := pt.GetBaseline("test-1")

	// Mean should be around 105ms, not affected by failed tests
	if baseline.Mean > 150 {
		t.Errorf("Mean = %f, expected around 105ms (failed tests should be excluded)", baseline.Mean)
	}
}

// Statistical function tests

func TestMean(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{}, 0},
		{[]float64{5}, 5},
		{[]float64{1, 2, 3, 4, 5}, 3},
		{[]float64{10, 20, 30}, 20},
		{[]float64{100, 200}, 150},
	}

	for _, tt := range tests {
		result := mean(tt.values)
		if result != tt.expected {
			t.Errorf("mean(%v) = %f, expected %f", tt.values, result, tt.expected)
		}
	}
}

func TestStdDev(t *testing.T) {
	// No variation
	result1 := stdDev([]float64{5, 5, 5, 5}, 5)
	if result1 != 0 {
		t.Errorf("stdDev of constant values should be 0, got %f", result1)
	}

	// Some variation
	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	m := mean(values)
	result2 := stdDev(values, m)
	// Expected stdDev is approximately 2.14
	if result2 < 2 || result2 > 2.3 {
		t.Errorf("stdDev = %f, expected approximately 2.14", result2)
	}

	// Single value
	result3 := stdDev([]float64{5}, 5)
	if result3 != 0 {
		t.Errorf("stdDev of single value should be 0, got %f", result3)
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		values     []float64
		percentile float64
		expected   float64
	}{
		{[]float64{}, 50, 0},
		{[]float64{1, 2, 3, 4, 5}, 50, 3},
		{[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 50, 5.5},
		{[]float64{1, 2, 3, 4, 5}, 0, 1},
		{[]float64{1, 2, 3, 4, 5}, 100, 5},
	}

	for _, tt := range tests {
		result := percentile(tt.values, tt.percentile)
		if math.Abs(result-tt.expected) > 0.01 {
			t.Errorf("percentile(%v, %f) = %f, expected %f", tt.values, tt.percentile, result, tt.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{}, 0},
		{[]float64{5}, 5},
		{[]float64{3, 1, 4, 1, 5}, 1},
		{[]float64{-5, -10, -3}, -10},
		{[]float64{100, 50, 75}, 50},
	}

	for _, tt := range tests {
		result := min(tt.values)
		if result != tt.expected {
			t.Errorf("min(%v) = %f, expected %f", tt.values, result, tt.expected)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{}, 0},
		{[]float64{5}, 5},
		{[]float64{3, 1, 4, 1, 5}, 5},
		{[]float64{-5, -10, -3}, -3},
		{[]float64{100, 50, 75}, 100},
	}

	for _, tt := range tests {
		result := max(tt.values)
		if result != tt.expected {
			t.Errorf("max(%v) = %f, expected %f", tt.values, result, tt.expected)
		}
	}
}

func TestLinearTrendSlope(t *testing.T) {
	// Constant values - no trend
	result1 := linearTrendSlope([]float64{5, 5, 5, 5})
	if result1 != 0 {
		t.Errorf("linearTrendSlope of constant values should be 0, got %f", result1)
	}

	// Increasing values - positive trend
	result2 := linearTrendSlope([]float64{1, 2, 3, 4, 5})
	if result2 <= 0 {
		t.Errorf("linearTrendSlope of increasing values should be positive, got %f", result2)
	}

	// Decreasing values - negative trend
	result3 := linearTrendSlope([]float64{5, 4, 3, 2, 1})
	if result3 >= 0 {
		t.Errorf("linearTrendSlope of decreasing values should be negative, got %f", result3)
	}

	// Single value
	result4 := linearTrendSlope([]float64{5})
	if result4 != 0 {
		t.Errorf("linearTrendSlope of single value should be 0, got %f", result4)
	}
}

