package daemon

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/joshua-temple/chronicle/core"
	"github.com/joshua-temple/chronicle/scenarios"
)

// ExecutionStrategy defines how tests are selected for execution
type ExecutionStrategy string

const (
	// RandomExecution selects tests randomly
	RandomExecution ExecutionStrategy = "random"

	// SequentialExecution selects tests in sequence
	SequentialExecution ExecutionStrategy = "sequential"

	// WeightedExecution selects tests based on weights
	WeightedExecution ExecutionStrategy = "weighted"

	// PriorityExecution selects tests based on priority
	PriorityExecution ExecutionStrategy = "priority"
)

// TimeWindow represents a scheduled execution window
type TimeWindow struct {
	// DaysOfWeek specifies which days this window is active
	DaysOfWeek []time.Weekday

	// StartTime is the time of day the window begins ("HH:MM" format)
	StartTime string

	// EndTime is the time of day the window ends ("HH:MM" format)
	EndTime string

	// TimeZone is the IANA time zone name
	TimeZone string
}

// IsActive checks if the window is currently active
func (w *TimeWindow) IsActive(t time.Time) bool {
	// Parse the location
	loc, err := time.LoadLocation(w.TimeZone)
	if err != nil {
		// Default to local time if zone invalid
		loc = time.Local
	}

	// Convert to correct timezone
	localTime := t.In(loc)

	// Check day of week
	dayMatch := false
	for _, day := range w.DaysOfWeek {
		if localTime.Weekday() == day {
			dayMatch = true
			break
		}
	}

	if !dayMatch {
		return false
	}

	// Parse start time
	startHour, startMinute := 0, 0
	fmt.Sscanf(w.StartTime, "%d:%d", &startHour, &startMinute)

	// Parse end time
	endHour, endMinute := 23, 59
	fmt.Sscanf(w.EndTime, "%d:%d", &endHour, &endMinute)

	// Create time points for comparison
	startOfDay := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), 0, 0, 0, 0, loc)
	windowStart := startOfDay.Add(time.Duration(startHour)*time.Hour + time.Duration(startMinute)*time.Minute)
	windowEnd := startOfDay.Add(time.Duration(endHour)*time.Hour + time.Duration(endMinute)*time.Minute)

	// Check if current time is within window
	return localTime.After(windowStart) && localTime.Before(windowEnd)
}

// ResourceUsage defines resource consumption limits
type ResourceUsage struct {
	CPU    int // CPU percentage (0-100)
	Memory int // Memory in MB
	Disk   int // Disk space in MB
}

// TestExecutionConfig controls how a test is executed by the daemon
type TestExecutionConfig struct {
	// Basic scheduling
	Frequency       time.Duration // How often to run
	MaxParallelSelf int           // How many instances can run in parallel
	Jitter          time.Duration // Random delay added to scheduling

	// Advanced scheduling
	TimeWindows    []TimeWindow  // Only run during specific time windows
	CooldownPeriod time.Duration // Minimum time between runs
	ExecutionQuota int           // Max executions per time period
	QuotaPeriod    time.Duration // Time period for quota reset

	// Resource constraints
	ResourceWeight   int           // Resource consumption weight
	MaxResourceUsage ResourceUsage // Maximum resources this test can use

	// Failure handling
	MaxConsecutiveFailures int           // Max failures before disabling
	QuarantineDuration     time.Duration // Time to disable after excessive failures
	AlertThreshold         int           // Failures before triggering alert

	// Input variation
	VariationStrategy VariationStrategy // How to vary inputs between runs
	InputConstraints  []InputConstraint // Constraints on generated inputs

	// Internal tracking fields
	lastExecution        time.Time
	executionsThisPeriod int
	consecutiveFailures  int
	quarantinedUntil     time.Time
}

// InputConstraint defines a constraint on generated input
type InputConstraint struct {
	ParameterName string                 // Name of the parameter
	Validator     func(interface{}) bool // Validation function
}

// VariationStrategy defines how inputs are varied between runs
type VariationStrategy string

const (
	// RandomVariation generates completely random inputs each time
	RandomVariation VariationStrategy = "random"

	// IncrementalVariation systematically varies inputs
	IncrementalVariation VariationStrategy = "incremental"

	// BoundaryVariation focuses on boundary values
	BoundaryVariation VariationStrategy = "boundary"

	// PreviousFailureVariation focuses on inputs that caused failures
	PreviousFailureVariation VariationStrategy = "previous_failure"
)

// EnhancedDaemonConfig controls the behavior of the testing daemon with advanced features
type EnhancedDaemonConfig struct {
	// Test selection
	ExecutionStrategy ExecutionStrategy

	// Execution control
	MaxConcurrency      int           // Max tests running simultaneously
	GlobalThrottleRate  int           // Max test starts per minute
	ShutdownGracePeriod time.Duration // How long to wait during shutdown

	// Resource management
	MaxResourceAllocation ResourceUsage

	// Observation
	MetricsExportInterval time.Duration
	ResultsExportPath     string

	// Chaos features
	ChaosLevel    float64 // 0.0-1.0 intensity of chaos
	ChaosFeatures []ChaosFeature

	// Test configs - map of test ID to config
	TestConfigs map[core.TestID]*TestExecutionConfig
}

// ChaosFeature represents a type of chaos that can be introduced
type ChaosFeature string

const (
	// NetworkLatencyChaos introduces random network latency
	NetworkLatencyChaos ChaosFeature = "network_latency"

	// ServiceRestartChaos randomly restarts services
	ServiceRestartChaos ChaosFeature = "service_restart"

	// ResourceContentionChaos introduces resource contention
	ResourceContentionChaos ChaosFeature = "resource_contention"

	// ClockDriftChaos manipulates system time
	ClockDriftChaos ChaosFeature = "clock_drift"
)

// NewEnhancedDaemonConfig creates a new daemon configuration with defaults
func NewEnhancedDaemonConfig() *EnhancedDaemonConfig {
	return &EnhancedDaemonConfig{
		ExecutionStrategy:   RandomExecution,
		MaxConcurrency:      5,
		GlobalThrottleRate:  30,
		ShutdownGracePeriod: 30 * time.Second,
		MaxResourceAllocation: ResourceUsage{
			CPU:    80,    // 80% CPU
			Memory: 2048,  // 2GB RAM
			Disk:   10240, // 10GB disk
		},
		MetricsExportInterval: 1 * time.Minute,
		ChaosLevel:            0.0, // No chaos by default
		TestConfigs:           make(map[core.TestID]*TestExecutionConfig),
	}
}

// DefaultTestConfig creates a default test execution configuration
func DefaultTestConfig() *TestExecutionConfig {
	return &TestExecutionConfig{
		Frequency:              1 * time.Hour,
		MaxParallelSelf:        1,
		Jitter:                 10 * time.Second,
		CooldownPeriod:         1 * time.Minute,
		ExecutionQuota:         10,
		QuotaPeriod:            24 * time.Hour,
		ResourceWeight:         1,
		MaxResourceUsage:       ResourceUsage{CPU: 10, Memory: 256, Disk: 100},
		MaxConsecutiveFailures: 3,
		QuarantineDuration:     1 * time.Hour,
		AlertThreshold:         1,
		VariationStrategy:      RandomVariation,
	}
}

// TestScenarioStore stores scenarios by test ID
type TestScenarioStore struct {
	// Map of test ID to registered scenario
	scenarios map[core.TestID]*scenarios.Scenario

	// Registry of component by ID
	components map[core.ComponentID]scenarios.Component

	// Mutex for concurrent access
	mutex sync.RWMutex
}

// NewTestScenarioStore creates a new test scenario store
func NewTestScenarioStore() *TestScenarioStore {
	return &TestScenarioStore{
		scenarios:  make(map[core.TestID]*scenarios.Scenario),
		components: make(map[core.ComponentID]scenarios.Component),
	}
}

// RegisterScenario adds a scenario for a test ID
func (s *TestScenarioStore) RegisterScenario(id core.TestID, scenario *scenarios.Scenario) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scenarios[id] = scenario
}

// RegisterComponent adds a component
func (s *TestScenarioStore) RegisterComponent(id core.ComponentID, component scenarios.Component) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.components[id] = component
}

// GetScenario retrieves a scenario by test ID
func (s *TestScenarioStore) GetScenario(id core.TestID) (*scenarios.Scenario, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	scenario, exists := s.scenarios[id]
	return scenario, exists
}

// GetComponent retrieves a component by ID
func (s *TestScenarioStore) GetComponent(id core.ComponentID) (scenarios.Component, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	component, exists := s.components[id]
	return component, exists
}

// GetAllTestIDs returns all registered test IDs
func (s *TestScenarioStore) GetAllTestIDs() []core.TestID {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ids := make([]core.TestID, 0, len(s.scenarios))
	for id := range s.scenarios {
		ids = append(ids, id)
	}

	return ids
}

// ExecutionScheduler schedules test executions
type ExecutionScheduler struct {
	config  *EnhancedDaemonConfig
	store   *TestScenarioStore
	queue   chan core.TestID
	running map[core.TestID]int
	mutex   sync.Mutex
	stopCh  chan struct{}
	logger  Logger
}

// NewExecutionScheduler creates a new scheduler
func NewExecutionScheduler(config *EnhancedDaemonConfig, store *TestScenarioStore, logger Logger) *ExecutionScheduler {
	return &ExecutionScheduler{
		config:  config,
		store:   store,
		queue:   make(chan core.TestID, 100),
		running: make(map[core.TestID]int),
		stopCh:  make(chan struct{}),
		logger:  logger,
	}
}

// Start begins the scheduling process
func (s *ExecutionScheduler) Start() {
	go s.scheduleLoop()
}

// Stop halts the scheduling process
func (s *ExecutionScheduler) Stop() {
	close(s.stopCh)
}

// scheduleLoop continuously schedules tests
func (s *ExecutionScheduler) scheduleLoop() {
	// Create tickers for each test
	tickers := make(map[core.TestID]*time.Ticker)

	// Initial scheduling for all tests
	allTestIDs := s.store.GetAllTestIDs()
	for _, testID := range allTestIDs {
		config, exists := s.config.TestConfigs[testID]
		if !exists {
			config = DefaultTestConfig()
			s.config.TestConfigs[testID] = config
		}

		// Create a ticker with the test's frequency
		ticker := time.NewTicker(config.Frequency)
		tickers[testID] = ticker

		// Schedule initial execution with jitter
		if config.Jitter > 0 {
			jitter := time.Duration(rand.Int63n(int64(config.Jitter)))
			time.AfterFunc(jitter, func() {
				s.scheduleTest(testID)
			})
		} else {
			s.scheduleTest(testID)
		}
	}

	// Main loop
	for {
		select {
		case <-s.stopCh:
			// Stop all tickers
			for _, ticker := range tickers {
				ticker.Stop()
			}
			return

		default:
			// Check if any tickers have fired
			for testID, ticker := range tickers {
				select {
				case <-ticker.C:
					// Schedule test execution
					s.scheduleTest(testID)

				default:
					// No tick yet
				}
			}

			// Sleep a bit to avoid busy loop
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// scheduleTest adds a test to the execution queue if it's eligible
func (s *ExecutionScheduler) scheduleTest(testID core.TestID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	config, exists := s.config.TestConfigs[testID]
	if !exists {
		config = DefaultTestConfig()
		s.config.TestConfigs[testID] = config
	}

	// Check if test is quarantined
	if !config.quarantinedUntil.IsZero() && time.Now().Before(config.quarantinedUntil) {
		s.logger.Debug("Test %s is quarantined until %s", testID, config.quarantinedUntil)
		return
	}

	// Check cooldown period
	if !config.lastExecution.IsZero() &&
		time.Since(config.lastExecution) < config.CooldownPeriod {
		s.logger.Debug("Test %s is cooling down", testID)
		return
	}

	// Check quota
	if config.executionsThisPeriod >= config.ExecutionQuota && config.ExecutionQuota > 0 {
		s.logger.Debug("Test %s has exceeded its quota", testID)
		return
	}

	// Check time windows
	if len(config.TimeWindows) > 0 {
		inWindow := false
		now := time.Now()
		for _, window := range config.TimeWindows {
			if window.IsActive(now) {
				inWindow = true
				break
			}
		}

		if !inWindow {
			s.logger.Debug("Test %s is outside its execution window", testID)
			return
		}
	}

	// Check if test is already running at max capacity
	if s.running[testID] >= config.MaxParallelSelf {
		s.logger.Debug("Test %s is already running at max capacity", testID)
		return
	}

	// Update tracking info
	config.lastExecution = time.Now()
	config.executionsThisPeriod++
	s.running[testID]++

	// Reset quota if period has elapsed
	if config.QuotaPeriod > 0 && time.Since(config.lastExecution) > config.QuotaPeriod {
		config.executionsThisPeriod = 1 // Count this execution
	}

	// Add to execution queue
	select {
	case s.queue <- testID:
		s.logger.Debug("Scheduled test %s for execution", testID)
	default:
		s.logger.Warning("Queue full, couldn't schedule test %s", testID)
		// Rollback tracking info
		s.running[testID]--
	}
}

// GetNextTest gets the next test to execute (used by workers)
func (s *ExecutionScheduler) GetNextTest() (core.TestID, bool) {
	select {
	case testID := <-s.queue:
		return testID, true
	case <-time.After(100 * time.Millisecond):
		return "", false
	}
}

// MarkTestComplete updates tracking when a test completes
func (s *ExecutionScheduler) MarkTestComplete(testID core.TestID, success bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	config, exists := s.config.TestConfigs[testID]
	if !exists {
		return
	}

	// Update running count
	if s.running[testID] > 0 {
		s.running[testID]--
	}

	// Update failure tracking
	if success {
		config.consecutiveFailures = 0
	} else {
		config.consecutiveFailures++

		// Check if test should be quarantined
		if config.MaxConsecutiveFailures > 0 && config.consecutiveFailures >= config.MaxConsecutiveFailures {
			config.quarantinedUntil = time.Now().Add(config.QuarantineDuration)
			s.logger.Warning("Test %s quarantined until %s after %d consecutive failures",
				testID, config.quarantinedUntil, config.consecutiveFailures)
		}
	}
}
