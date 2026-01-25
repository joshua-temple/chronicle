package results

import (
	"encoding/json"
	"time"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/execution"
)

// RunResult represents the complete result of a Chronicle run.
type RunResult struct {
	// Identification
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// Configuration
	Config RunConfig `json:"config"`

	// Results
	Scenarios []ScenarioRunResult `json:"scenarios"`

	// Aggregates
	Stats RunStats `json:"stats"`

	// Environment
	Environment EnvironmentInfo `json:"environment"`

	// Narrative (human-readable summary)
	Narrative string `json:"narrative,omitempty"`
}

// RunConfig captures the configuration used for this run.
type RunConfig struct {
	Tags        []string       `json:"tags,omitempty"`
	Flags       map[string]any `json:"flags,omitempty"`
	Parallelism int            `json:"parallelism"`
	FailFast    bool           `json:"fail_fast"`
	Timeout     time.Duration  `json:"timeout,omitempty"`
}

// ScenarioRunResult wraps execution.ScenarioResult with additional metadata.
type ScenarioRunResult struct {
	// From execution result
	ScenarioID   string              `json:"scenario_id"`
	ScenarioName string              `json:"scenario_name"`
	State        string              `json:"state"`
	StartTime    time.Time           `json:"start_time"`
	EndTime      time.Time           `json:"end_time"`
	Duration     time.Duration       `json:"duration"`
	Error        string              `json:"error,omitempty"`
	SkipReason   string              `json:"skip_reason,omitempty"`

	// Flow results
	FlowResults     []FlowItemRunResult `json:"flow_results"`
	TeardownResults []FlowItemRunResult `json:"teardown_results,omitempty"`

	// Additional metadata
	Tags    []string               `json:"tags,omitempty"`
	Matrix  map[string]any         `json:"matrix,omitempty"`
	Logs    []LogEntry             `json:"logs,omitempty"`
	Metrics map[string]MetricValue `json:"metrics,omitempty"`
}

// FlowItemRunResult captures the result of a single flow item.
type FlowItemRunResult struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	State     string        `json:"state"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
	Output    any           `json:"output,omitempty"`

	// Logs for this specific item
	Logs []LogEntry `json:"logs,omitempty"`
}

// LogEntry represents a log message.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Component string    `json:"component"`
	Message   string    `json:"message"`
}

// MetricValue represents a recorded metric.
type MetricValue struct {
	Value     float64           `json:"value"`
	Unit      string            `json:"unit,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// RunStats aggregates statistics across all scenarios.
type RunStats struct {
	Total     int           `json:"total"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Skipped   int           `json:"skipped"`
	Duration  time.Duration `json:"duration"`

	// Per-state counts
	ByState map[string]int `json:"by_state"`

	// Timing statistics
	MinDuration time.Duration `json:"min_duration"`
	MaxDuration time.Duration `json:"max_duration"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// EnvironmentInfo captures details about the test environment.
type EnvironmentInfo struct {
	OS           string            `json:"os,omitempty"`
	Arch         string            `json:"arch,omitempty"`
	GoVersion    string            `json:"go_version,omitempty"`
	Hostname     string            `json:"hostname,omitempty"`
	CI           bool              `json:"ci"`
	CIProvider   string            `json:"ci_provider,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Commit       string            `json:"commit,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// NewRunResult creates a new RunResult from execution results.
func NewRunResult(name string, execResults []*execution.ScenarioResult) *RunResult {
	result := &RunResult{
		ID:        core.NewRunID().String(),
		Name:      name,
		StartTime: time.Now(),
		Scenarios: make([]ScenarioRunResult, 0, len(execResults)),
		Stats: RunStats{
			ByState: make(map[string]int),
		},
	}

	var minDuration, maxDuration, totalDuration time.Duration
	first := true

	for _, er := range execResults {
		sr := convertScenarioResult(er)
		result.Scenarios = append(result.Scenarios, sr)

		// Update stats
		result.Stats.Total++
		result.Stats.ByState[sr.State]++

		switch er.State {
		case execution.StateCompleted:
			result.Stats.Passed++
		case execution.StateFailed:
			result.Stats.Failed++
		case execution.StateSkipped:
			result.Stats.Skipped++
		}

		// Track timing
		totalDuration += er.Duration
		if first || er.Duration < minDuration {
			minDuration = er.Duration
		}
		if first || er.Duration > maxDuration {
			maxDuration = er.Duration
		}
		first = false
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Stats.Duration = totalDuration
	result.Stats.MinDuration = minDuration
	result.Stats.MaxDuration = maxDuration
	if result.Stats.Total > 0 {
		result.Stats.AvgDuration = totalDuration / time.Duration(result.Stats.Total)
	}

	return result
}

// convertScenarioResult converts an execution.ScenarioResult to ScenarioRunResult.
func convertScenarioResult(er *execution.ScenarioResult) ScenarioRunResult {
	sr := ScenarioRunResult{
		ScenarioID:   er.ScenarioID.String(),
		ScenarioName: er.ScenarioName,
		State:        er.State.String(),
		StartTime:    er.StartTime,
		EndTime:      er.EndTime,
		Duration:     er.Duration,
		SkipReason:   er.SkipReason,
	}

	if er.Error != nil {
		sr.Error = er.Error.Error()
	}

	// Convert flow results
	sr.FlowResults = make([]FlowItemRunResult, 0, len(er.FlowResults))
	for _, fr := range er.FlowResults {
		sr.FlowResults = append(sr.FlowResults, convertFlowItemResult(fr))
	}

	// Convert teardown results
	if len(er.TeardownResults) > 0 {
		sr.TeardownResults = make([]FlowItemRunResult, 0, len(er.TeardownResults))
		for _, tr := range er.TeardownResults {
			sr.TeardownResults = append(sr.TeardownResults, convertFlowItemResult(tr))
		}
	}

	return sr
}

// convertFlowItemResult converts an execution.FlowItemResult to FlowItemRunResult.
func convertFlowItemResult(fr execution.FlowItemResult) FlowItemRunResult {
	fir := FlowItemRunResult{
		Name:      fr.Name,
		Type:      string(fr.Type),
		State:     fr.State.String(),
		StartTime: fr.StartTime,
		EndTime:   fr.EndTime,
		Duration:  fr.Duration,
		Output:    fr.Output,
	}

	if fr.Error != nil {
		fir.Error = fr.Error.Error()
	}

	return fir
}

// IsSuccess returns true if the run was successful (all scenarios passed or skipped).
func (r *RunResult) IsSuccess() bool {
	return r.Stats.Failed == 0
}

// PassRate returns the percentage of passed scenarios.
func (r *RunResult) PassRate() float64 {
	if r.Stats.Total == 0 {
		return 0
	}
	return float64(r.Stats.Passed) / float64(r.Stats.Total) * 100
}

// JSON returns the result as a JSON string.
func (r *RunResult) JSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CompactJSON returns the result as a compact JSON string.
func (r *RunResult) CompactJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FailedScenarios returns only the scenarios that failed.
func (r *RunResult) FailedScenarios() []ScenarioRunResult {
	var failed []ScenarioRunResult
	for _, s := range r.Scenarios {
		if s.State == "failed" {
			failed = append(failed, s)
		}
	}
	return failed
}

// Summary returns a brief text summary of the results.
func (r *RunResult) Summary() string {
	return execution.Summary(r.toExecutionResults())
}

// toExecutionResults converts back to execution results for summary generation.
func (r *RunResult) toExecutionResults() []*execution.ScenarioResult {
	results := make([]*execution.ScenarioResult, len(r.Scenarios))
	for i, s := range r.Scenarios {
		results[i] = &execution.ScenarioResult{
			ScenarioID:   core.ScenarioID(s.ScenarioID),
			ScenarioName: s.ScenarioName,
			State:        parseState(s.State),
			Duration:     s.Duration,
			SkipReason:   s.SkipReason,
		}
	}
	return results
}

// parseState converts a state string back to ExecutionState.
func parseState(s string) execution.ExecutionState {
	switch s {
	case "not_started":
		return execution.StateNotStarted
	case "running":
		return execution.StateRunning
	case "completed":
		return execution.StateCompleted
	case "failed":
		return execution.StateFailed
	case "skipped":
		return execution.StateSkipped
	case "cancelled":
		return execution.StateCancelled
	default:
		return execution.StateNotStarted
	}
}

// Collector aggregates results from multiple scenario executions.
type Collector struct {
	name    string
	results []*execution.ScenarioResult
	config  RunConfig
	env     EnvironmentInfo
}

// NewCollector creates a new result collector.
func NewCollector(name string) *Collector {
	return &Collector{
		name:    name,
		results: make([]*execution.ScenarioResult, 0),
	}
}

// Add adds a scenario result to the collector.
func (c *Collector) Add(result *execution.ScenarioResult) {
	c.results = append(c.results, result)
}

// AddAll adds multiple scenario results to the collector.
func (c *Collector) AddAll(results []*execution.ScenarioResult) {
	c.results = append(c.results, results...)
}

// SetConfig sets the run configuration.
func (c *Collector) SetConfig(config RunConfig) {
	c.config = config
}

// SetEnvironment sets the environment information.
func (c *Collector) SetEnvironment(env EnvironmentInfo) {
	c.env = env
}

// Build creates the final RunResult.
func (c *Collector) Build() *RunResult {
	result := NewRunResult(c.name, c.results)
	result.Config = c.config
	result.Environment = c.env
	return result
}

// Count returns the current number of collected results.
func (c *Collector) Count() int {
	return len(c.results)
}
