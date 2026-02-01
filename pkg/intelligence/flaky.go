package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FlakyStatus represents the flakiness status of a test.
type FlakyStatus int

const (
	FlakyStatusStable FlakyStatus = iota
	FlakyStatusSuspected
	FlakyStatusConfirmed
	FlakyStatusQuarantined
)

// String returns the string representation of the flaky status.
func (s FlakyStatus) String() string {
	switch s {
	case FlakyStatusStable:
		return "stable"
	case FlakyStatusSuspected:
		return "suspected"
	case FlakyStatusConfirmed:
		return "confirmed"
	case FlakyStatusQuarantined:
		return "quarantined"
	default:
		return "unknown"
	}
}

// TestOutcome represents a single test execution outcome.
type TestOutcome struct {
	Timestamp time.Time `json:"timestamp"`
	Passed    bool      `json:"passed"`
	Duration  float64   `json:"duration_ms"`
	Error     string    `json:"error,omitempty"`
	RunID     string    `json:"run_id,omitempty"`
}

// TestHistory tracks the execution history of a test.
type TestHistory struct {
	ScenarioName string        `json:"scenario_name"`
	ComponentName string       `json:"component_name,omitempty"`
	Outcomes     []TestOutcome `json:"outcomes"`
	Status       FlakyStatus   `json:"status"`
	FlakyScore   float64       `json:"flaky_score"`
	LastUpdated  time.Time     `json:"last_updated"`
}

// FlakyReport summarizes flaky test detection results.
type FlakyReport struct {
	GeneratedAt       time.Time             `json:"generated_at"`
	TotalScenarios    int                   `json:"total_scenarios"`
	StableScenarios   int                   `json:"stable_scenarios"`
	SuspectedFlaky    int                   `json:"suspected_flaky"`
	ConfirmedFlaky    int                   `json:"confirmed_flaky"`
	QuarantinedTests  int                   `json:"quarantined_tests"`
	FlakyTests        []FlakyTestSummary    `json:"flaky_tests"`
	Recommendations   []string              `json:"recommendations"`
}

// FlakyTestSummary summarizes a single flaky test.
type FlakyTestSummary struct {
	Name        string      `json:"name"`
	Status      FlakyStatus `json:"status"`
	FlakyScore  float64     `json:"flaky_score"`
	PassRate    float64     `json:"pass_rate"`
	TotalRuns   int         `json:"total_runs"`
	Flips       int         `json:"flips"`
	LastOutcome bool        `json:"last_outcome"`
}

// FlakyDetectorConfig configures the flaky test detector.
type FlakyDetectorConfig struct {
	// MinRuns is the minimum number of runs before detecting flakiness.
	MinRuns int `json:"min_runs"`

	// WindowSize is the number of recent runs to analyze.
	WindowSize int `json:"window_size"`

	// FlipThreshold is the minimum number of pass/fail flips to suspect flakiness.
	FlipThreshold int `json:"flip_threshold"`

	// FlakyScoreThreshold is the score above which a test is considered flaky.
	FlakyScoreThreshold float64 `json:"flaky_score_threshold"`

	// ConfirmationRuns is how many runs to confirm flakiness after suspicion.
	ConfirmationRuns int `json:"confirmation_runs"`

	// QuarantineAfterFailures quarantines after this many consecutive failures.
	QuarantineAfterFailures int `json:"quarantine_after_failures"`

	// StoragePath is where to store flaky detection data.
	StoragePath string `json:"storage_path"`
}

// DefaultFlakyDetectorConfig returns the default configuration.
func DefaultFlakyDetectorConfig() FlakyDetectorConfig {
	return FlakyDetectorConfig{
		MinRuns:                 5,
		WindowSize:              20,
		FlipThreshold:           3,
		FlakyScoreThreshold:     0.3,
		ConfirmationRuns:        10,
		QuarantineAfterFailures: 5,
		StoragePath:             ".chronicle/flaky",
	}
}

// FlakyDetector detects and tracks flaky tests.
type FlakyDetector struct {
	config   FlakyDetectorConfig
	history  map[string]*TestHistory
	mu       sync.RWMutex
}

// NewFlakyDetector creates a new flaky test detector.
func NewFlakyDetector(config FlakyDetectorConfig) *FlakyDetector {
	fd := &FlakyDetector{
		config:  config,
		history: make(map[string]*TestHistory),
	}

	// Load existing history
	_ = fd.loadHistory()

	return fd
}

// RecordOutcome records a test execution outcome.
func (fd *FlakyDetector) RecordOutcome(scenarioName string, passed bool, duration time.Duration, err error, runID string) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	// Get or create history
	history, exists := fd.history[scenarioName]
	if !exists {
		history = &TestHistory{
			ScenarioName: scenarioName,
			Outcomes:     make([]TestOutcome, 0),
			Status:       FlakyStatusStable,
		}
		fd.history[scenarioName] = history
	}

	// Add outcome
	outcome := TestOutcome{
		Timestamp: time.Now(),
		Passed:    passed,
		Duration:  float64(duration.Milliseconds()),
		RunID:     runID,
	}
	if err != nil {
		outcome.Error = err.Error()
	}

	history.Outcomes = append(history.Outcomes, outcome)
	history.LastUpdated = time.Now()

	// Trim to window size
	if len(history.Outcomes) > fd.config.WindowSize {
		history.Outcomes = history.Outcomes[len(history.Outcomes)-fd.config.WindowSize:]
	}

	// Analyze for flakiness
	fd.analyzeTest(history)

	// Save history
	_ = fd.saveHistory()
}

// analyzeTest analyzes a test for flakiness.
func (fd *FlakyDetector) analyzeTest(history *TestHistory) {
	if len(history.Outcomes) < fd.config.MinRuns {
		history.Status = FlakyStatusStable
		history.FlakyScore = 0
		return
	}

	// Count flips (changes between pass/fail)
	flips := 0
	passes := 0
	consecutiveFailures := 0
	maxConsecutiveFailures := 0

	for i, outcome := range history.Outcomes {
		if outcome.Passed {
			passes++
			consecutiveFailures = 0
		} else {
			consecutiveFailures++
			if consecutiveFailures > maxConsecutiveFailures {
				maxConsecutiveFailures = consecutiveFailures
			}
		}

		if i > 0 && outcome.Passed != history.Outcomes[i-1].Passed {
			flips++
		}
	}

	// Calculate flaky score
	// Score is based on:
	// 1. Number of flips relative to runs
	// 2. Pass rate not being close to 0 or 1
	// 3. Pattern irregularity
	n := float64(len(history.Outcomes))
	passRate := float64(passes) / n
	flipRate := float64(flips) / (n - 1)

	// Tests that are truly flaky have pass rates around 0.5 and high flip rates
	// Score formula: high flip rate + pass rate close to 0.5
	passRateScore := 1 - 2*abs(passRate-0.5) // Max at 0.5, min at 0 or 1
	history.FlakyScore = (flipRate + passRateScore) / 2

	// Check for quarantine (too many consecutive failures)
	if maxConsecutiveFailures >= fd.config.QuarantineAfterFailures {
		history.Status = FlakyStatusQuarantined
		return
	}

	// Determine status based on score and flips
	if history.FlakyScore >= fd.config.FlakyScoreThreshold && flips >= fd.config.FlipThreshold {
		if len(history.Outcomes) >= fd.config.ConfirmationRuns {
			history.Status = FlakyStatusConfirmed
		} else {
			history.Status = FlakyStatusSuspected
		}
	} else {
		history.Status = FlakyStatusStable
	}
}

// GetHistory returns the test history for a scenario.
func (fd *FlakyDetector) GetHistory(scenarioName string) (*TestHistory, bool) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	h, ok := fd.history[scenarioName]
	return h, ok
}

// GetFlakyTests returns all tests with suspected or confirmed flakiness.
func (fd *FlakyDetector) GetFlakyTests() []*TestHistory {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	var flaky []*TestHistory
	for _, h := range fd.history {
		if h.Status == FlakyStatusSuspected || h.Status == FlakyStatusConfirmed {
			flaky = append(flaky, h)
		}
	}

	// Sort by flaky score descending
	sort.Slice(flaky, func(i, j int) bool {
		return flaky[i].FlakyScore > flaky[j].FlakyScore
	})

	return flaky
}

// GetQuarantinedTests returns all quarantined tests.
func (fd *FlakyDetector) GetQuarantinedTests() []*TestHistory {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	var quarantined []*TestHistory
	for _, h := range fd.history {
		if h.Status == FlakyStatusQuarantined {
			quarantined = append(quarantined, h)
		}
	}

	return quarantined
}

// IsQuarantined checks if a test is quarantined.
func (fd *FlakyDetector) IsQuarantined(scenarioName string) bool {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	h, ok := fd.history[scenarioName]
	return ok && h.Status == FlakyStatusQuarantined
}

// Unquarantine removes a test from quarantine.
func (fd *FlakyDetector) Unquarantine(scenarioName string) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	if h, ok := fd.history[scenarioName]; ok {
		h.Status = FlakyStatusStable
		h.Outcomes = h.Outcomes[:0] // Reset history
		_ = fd.saveHistory()
	}
}

// GenerateReport generates a flaky test report.
func (fd *FlakyDetector) GenerateReport(ctx context.Context) *FlakyReport {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	report := &FlakyReport{
		GeneratedAt:     time.Now(),
		TotalScenarios:  len(fd.history),
		FlakyTests:      make([]FlakyTestSummary, 0),
		Recommendations: make([]string, 0),
	}

	for _, h := range fd.history {
		switch h.Status {
		case FlakyStatusStable:
			report.StableScenarios++
		case FlakyStatusSuspected:
			report.SuspectedFlaky++
		case FlakyStatusConfirmed:
			report.ConfirmedFlaky++
		case FlakyStatusQuarantined:
			report.QuarantinedTests++
		}

		// Include non-stable tests in report
		if h.Status != FlakyStatusStable {
			passes := 0
			flips := 0
			for i, o := range h.Outcomes {
				if o.Passed {
					passes++
				}
				if i > 0 && o.Passed != h.Outcomes[i-1].Passed {
					flips++
				}
			}

			lastOutcome := false
			if len(h.Outcomes) > 0 {
				lastOutcome = h.Outcomes[len(h.Outcomes)-1].Passed
			}

			report.FlakyTests = append(report.FlakyTests, FlakyTestSummary{
				Name:        h.ScenarioName,
				Status:      h.Status,
				FlakyScore:  h.FlakyScore,
				PassRate:    float64(passes) / float64(len(h.Outcomes)),
				TotalRuns:   len(h.Outcomes),
				Flips:       flips,
				LastOutcome: lastOutcome,
			})
		}
	}

	// Sort flaky tests by score
	sort.Slice(report.FlakyTests, func(i, j int) bool {
		return report.FlakyTests[i].FlakyScore > report.FlakyTests[j].FlakyScore
	})

	// Generate recommendations
	if report.ConfirmedFlaky > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Investigate %d confirmed flaky tests - they are causing unreliable results", report.ConfirmedFlaky))
	}
	if report.QuarantinedTests > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Review %d quarantined tests - they may need fixing or removal", report.QuarantinedTests))
	}
	if report.SuspectedFlaky > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Monitor %d suspected flaky tests - run them more to confirm or clear suspicion", report.SuspectedFlaky))
	}

	return report
}

// loadHistory loads history from storage.
func (fd *FlakyDetector) loadHistory() error {
	path := filepath.Join(fd.config.StoragePath, "history.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &fd.history)
}

// saveHistory saves history to storage.
func (fd *FlakyDetector) saveHistory() error {
	if err := os.MkdirAll(fd.config.StoragePath, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(fd.history, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(fd.config.StoragePath, "history.json")
	return os.WriteFile(path, data, 0644)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
