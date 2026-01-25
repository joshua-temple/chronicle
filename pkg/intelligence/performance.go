package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// PerformanceStatus represents the performance status of a test.
type PerformanceStatus int

const (
	PerformanceStatusNormal PerformanceStatus = iota
	PerformanceStatusImproved
	PerformanceStatusDegraded
	PerformanceStatusCritical
)

// String returns the string representation of the performance status.
func (s PerformanceStatus) String() string {
	switch s {
	case PerformanceStatusNormal:
		return "normal"
	case PerformanceStatusImproved:
		return "improved"
	case PerformanceStatusDegraded:
		return "degraded"
	case PerformanceStatusCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// TimingSample represents a single timing measurement.
type TimingSample struct {
	Timestamp time.Time `json:"timestamp"`
	Duration  float64   `json:"duration_ms"`
	RunID     string    `json:"run_id,omitempty"`
	Passed    bool      `json:"passed"`
}

// PerformanceBaseline represents performance statistics for a test.
type PerformanceBaseline struct {
	ScenarioName   string            `json:"scenario_name"`
	Samples        []TimingSample    `json:"samples"`
	Status         PerformanceStatus `json:"status"`
	Mean           float64           `json:"mean_ms"`
	StdDev         float64           `json:"std_dev_ms"`
	Median         float64           `json:"median_ms"`
	P95            float64           `json:"p95_ms"`
	P99            float64           `json:"p99_ms"`
	Min            float64           `json:"min_ms"`
	Max            float64           `json:"max_ms"`
	TrendSlope     float64           `json:"trend_slope"`
	LastUpdated    time.Time         `json:"last_updated"`
	BaselineWindow int               `json:"baseline_window"`
}

// PerformanceReport summarizes performance analysis results.
type PerformanceReport struct {
	GeneratedAt       time.Time               `json:"generated_at"`
	TotalScenarios    int                     `json:"total_scenarios"`
	NormalCount       int                     `json:"normal_count"`
	ImprovedCount     int                     `json:"improved_count"`
	DegradedCount     int                     `json:"degraded_count"`
	CriticalCount     int                     `json:"critical_count"`
	Regressions       []PerformanceRegression `json:"regressions"`
	Improvements      []PerformanceRegression `json:"improvements"`
	Recommendations   []string                `json:"recommendations"`
}

// PerformanceRegression describes a performance change.
type PerformanceRegression struct {
	ScenarioName   string  `json:"scenario_name"`
	BaselineMean   float64 `json:"baseline_mean_ms"`
	CurrentMean    float64 `json:"current_mean_ms"`
	ChangePercent  float64 `json:"change_percent"`
	Significance   float64 `json:"significance"`
	TrendDirection string  `json:"trend_direction"`
}

// PerformanceTrackerConfig configures the performance tracker.
type PerformanceTrackerConfig struct {
	// WindowSize is the number of samples to keep.
	WindowSize int `json:"window_size"`

	// BaselineWindow is the number of initial samples used for baseline.
	BaselineWindow int `json:"baseline_window"`

	// DegradationThreshold is the percentage increase that triggers degradation alert.
	DegradationThreshold float64 `json:"degradation_threshold_percent"`

	// CriticalThreshold is the percentage increase that triggers critical alert.
	CriticalThreshold float64 `json:"critical_threshold_percent"`

	// ImprovementThreshold is the percentage decrease that triggers improvement notice.
	ImprovementThreshold float64 `json:"improvement_threshold_percent"`

	// MinSamplesForAnalysis is minimum samples needed for statistical analysis.
	MinSamplesForAnalysis int `json:"min_samples_for_analysis"`

	// StoragePath is where to store performance data.
	StoragePath string `json:"storage_path"`
}

// DefaultPerformanceTrackerConfig returns the default configuration.
func DefaultPerformanceTrackerConfig() PerformanceTrackerConfig {
	return PerformanceTrackerConfig{
		WindowSize:            100,
		BaselineWindow:        20,
		DegradationThreshold:  20.0,  // 20% slower
		CriticalThreshold:     50.0,  // 50% slower
		ImprovementThreshold:  15.0,  // 15% faster
		MinSamplesForAnalysis: 10,
		StoragePath:           ".chronicle/performance",
	}
}

// PerformanceTracker tracks test performance over time.
type PerformanceTracker struct {
	config    PerformanceTrackerConfig
	baselines map[string]*PerformanceBaseline
	mu        sync.RWMutex
}

// NewPerformanceTracker creates a new performance tracker.
func NewPerformanceTracker(config PerformanceTrackerConfig) *PerformanceTracker {
	pt := &PerformanceTracker{
		config:    config,
		baselines: make(map[string]*PerformanceBaseline),
	}

	// Load existing baselines
	_ = pt.loadBaselines()

	return pt
}

// RecordTiming records a timing sample for a test.
func (pt *PerformanceTracker) RecordTiming(scenarioName string, duration time.Duration, passed bool, runID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Get or create baseline
	baseline, exists := pt.baselines[scenarioName]
	if !exists {
		baseline = &PerformanceBaseline{
			ScenarioName:   scenarioName,
			Samples:        make([]TimingSample, 0),
			Status:         PerformanceStatusNormal,
			BaselineWindow: pt.config.BaselineWindow,
		}
		pt.baselines[scenarioName] = baseline
	}

	// Add sample
	sample := TimingSample{
		Timestamp: time.Now(),
		Duration:  float64(duration.Milliseconds()),
		RunID:     runID,
		Passed:    passed,
	}

	baseline.Samples = append(baseline.Samples, sample)
	baseline.LastUpdated = time.Now()

	// Trim to window size
	if len(baseline.Samples) > pt.config.WindowSize {
		baseline.Samples = baseline.Samples[len(baseline.Samples)-pt.config.WindowSize:]
	}

	// Only analyze passing tests for performance
	if passed {
		pt.analyzeBaseline(baseline)
	}

	// Save baselines
	_ = pt.saveBaselines()
}

// analyzeBaseline analyzes performance statistics.
func (pt *PerformanceTracker) analyzeBaseline(baseline *PerformanceBaseline) {
	// Filter to passing tests only
	var durations []float64
	for _, s := range baseline.Samples {
		if s.Passed {
			durations = append(durations, s.Duration)
		}
	}

	if len(durations) < pt.config.MinSamplesForAnalysis {
		baseline.Status = PerformanceStatusNormal
		return
	}

	// Calculate statistics
	baseline.Mean = mean(durations)
	baseline.StdDev = stdDev(durations, baseline.Mean)
	baseline.Median = percentile(durations, 50)
	baseline.P95 = percentile(durations, 95)
	baseline.P99 = percentile(durations, 99)
	baseline.Min = min(durations)
	baseline.Max = max(durations)

	// Calculate trend
	baseline.TrendSlope = linearTrendSlope(durations)

	// Compare recent performance to baseline
	if len(durations) >= pt.config.BaselineWindow {
		baselineSamples := durations[:pt.config.BaselineWindow]
		recentSamples := durations[len(durations)-pt.config.MinSamplesForAnalysis:]

		baselineMean := mean(baselineSamples)
		recentMean := mean(recentSamples)

		if baselineMean > 0 {
			changePercent := ((recentMean - baselineMean) / baselineMean) * 100

			if changePercent >= pt.config.CriticalThreshold {
				baseline.Status = PerformanceStatusCritical
			} else if changePercent >= pt.config.DegradationThreshold {
				baseline.Status = PerformanceStatusDegraded
			} else if changePercent <= -pt.config.ImprovementThreshold {
				baseline.Status = PerformanceStatusImproved
			} else {
				baseline.Status = PerformanceStatusNormal
			}
		}
	}
}

// GetBaseline returns the performance baseline for a scenario.
func (pt *PerformanceTracker) GetBaseline(scenarioName string) (*PerformanceBaseline, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	b, ok := pt.baselines[scenarioName]
	return b, ok
}

// GetRegressions returns all scenarios with performance degradation.
func (pt *PerformanceTracker) GetRegressions() []*PerformanceBaseline {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var regressions []*PerformanceBaseline
	for _, b := range pt.baselines {
		if b.Status == PerformanceStatusDegraded || b.Status == PerformanceStatusCritical {
			regressions = append(regressions, b)
		}
	}

	return regressions
}

// GenerateReport generates a performance analysis report.
func (pt *PerformanceTracker) GenerateReport(ctx context.Context) *PerformanceReport {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	report := &PerformanceReport{
		GeneratedAt:     time.Now(),
		TotalScenarios:  len(pt.baselines),
		Regressions:     make([]PerformanceRegression, 0),
		Improvements:    make([]PerformanceRegression, 0),
		Recommendations: make([]string, 0),
	}

	for _, b := range pt.baselines {
		switch b.Status {
		case PerformanceStatusNormal:
			report.NormalCount++
		case PerformanceStatusImproved:
			report.ImprovedCount++
		case PerformanceStatusDegraded:
			report.DegradedCount++
		case PerformanceStatusCritical:
			report.CriticalCount++
		}

		// Calculate change for non-normal scenarios
		if b.Status != PerformanceStatusNormal && len(b.Samples) >= pt.config.BaselineWindow {
			var durations []float64
			for _, s := range b.Samples {
				if s.Passed {
					durations = append(durations, s.Duration)
				}
			}

			if len(durations) >= pt.config.MinSamplesForAnalysis {
				baselineMean := mean(durations[:pt.config.BaselineWindow])
				currentMean := mean(durations[len(durations)-pt.config.MinSamplesForAnalysis:])
				changePercent := ((currentMean - baselineMean) / baselineMean) * 100

				regression := PerformanceRegression{
					ScenarioName:  b.ScenarioName,
					BaselineMean:  baselineMean,
					CurrentMean:   currentMean,
					ChangePercent: changePercent,
					Significance:  math.Abs(changePercent) / 10, // Simple significance estimate
				}

				if changePercent > 0 {
					regression.TrendDirection = "slower"
				} else {
					regression.TrendDirection = "faster"
				}

				switch b.Status {
				case PerformanceStatusDegraded, PerformanceStatusCritical:
					report.Regressions = append(report.Regressions, regression)
				case PerformanceStatusImproved:
					report.Improvements = append(report.Improvements, regression)
				default:
					// Normal status - no action needed
				}
			}
		}
	}

	// Sort by significance
	sort.Slice(report.Regressions, func(i, j int) bool {
		return math.Abs(report.Regressions[i].ChangePercent) > math.Abs(report.Regressions[j].ChangePercent)
	})
	sort.Slice(report.Improvements, func(i, j int) bool {
		return math.Abs(report.Improvements[i].ChangePercent) > math.Abs(report.Improvements[j].ChangePercent)
	})

	// Generate recommendations
	if report.CriticalCount > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("URGENT: %d tests have critical performance degradation (>%.0f%% slower)",
				report.CriticalCount, pt.config.CriticalThreshold))
	}
	if report.DegradedCount > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Investigate %d tests with performance degradation (>%.0f%% slower)",
				report.DegradedCount, pt.config.DegradationThreshold))
	}
	if report.ImprovedCount > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Verify %d tests showing improvement - ensure functionality is preserved",
				report.ImprovedCount))
	}

	return report
}

// ResetBaseline resets the baseline for a scenario.
func (pt *PerformanceTracker) ResetBaseline(scenarioName string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if b, ok := pt.baselines[scenarioName]; ok {
		b.Samples = b.Samples[:0]
		b.Status = PerformanceStatusNormal
		_ = pt.saveBaselines()
	}
}

// loadBaselines loads baselines from storage.
func (pt *PerformanceTracker) loadBaselines() error {
	path := filepath.Join(pt.config.StoragePath, "baselines.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &pt.baselines)
}

// saveBaselines saves baselines to storage.
func (pt *PerformanceTracker) saveBaselines() error {
	if err := os.MkdirAll(pt.config.StoragePath, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(pt.baselines, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(pt.config.StoragePath, "baselines.json")
	return os.WriteFile(path, data, 0644)
}

// Statistical helper functions

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	index := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	fraction := index - float64(lower)
	return sorted[lower]*(1-fraction) + sorted[upper]*fraction
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func linearTrendSlope(values []float64) float64 {
	n := float64(len(values))
	if n < 2 {
		return 0
	}

	// Simple linear regression: y = mx + b
	// We're looking for m (slope)
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// m = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}

	return (n*sumXY - sumX*sumY) / denominator
}
