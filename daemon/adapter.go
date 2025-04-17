package daemon

import (
	"time"

	"github.com/joshua-temple/chronicle/core"
	"github.com/joshua-temple/chronicle/scenarios"
)

// LegacyTestSetAdapter converts between the old test set model and the new typed ID model
type LegacyTestSetAdapter struct {
	store      *TestScenarioStore
	testSets   map[string]TestSet
	components map[string]scenarios.Component
}

// NewLegacyTestSetAdapter creates a new adapter
func NewLegacyTestSetAdapter() *LegacyTestSetAdapter {
	return &LegacyTestSetAdapter{
		store:      NewTestScenarioStore(),
		testSets:   make(map[string]TestSet),
		components: make(map[string]scenarios.Component),
	}
}

// AddTestSet adds a test set to the adapter
func (a *LegacyTestSetAdapter) AddTestSet(testSet TestSet) {
	a.testSets[testSet.Name] = testSet

	// Create a typed ID for the test set
	testID := core.TestID(testSet.Name)

	// Register components
	for _, compName := range testSet.Components {
		if comp, ok := a.components[compName]; ok {
			a.store.RegisterComponent(core.ComponentID(compName), comp)
		}
	}

	// Create a scenario
	builder := scenarios.NewBuilder(testSet.Name)
	for _, param := range testSet.RequiredParameters {
		if gen, ok := testSet.ParameterGenerators[param]; ok {
			builder.WithParameter(scenarios.Parameter{
				Name:      param,
				Generator: gen,
			})
		}
	}

	// Register the scenario
	a.store.RegisterScenario(testID, builder.Build())
}

// RegisterComponent adds a component to the adapter
func (a *LegacyTestSetAdapter) RegisterComponent(name string, component scenarios.Component) {
	a.components[name] = component
	a.store.RegisterComponent(core.ComponentID(name), component)
}

// GetStore returns the test scenario store
func (a *LegacyTestSetAdapter) GetStore() *TestScenarioStore {
	return a.store
}

// ConfigAdapter converts between the old DaemonConfig and the new EnhancedDaemonConfig
type ConfigAdapter struct {
	legacyConfig   *DaemonConfig
	enhancedConfig *EnhancedDaemonConfig
	adapter        *LegacyTestSetAdapter
}

// NewConfigAdapter creates a new config adapter
func NewConfigAdapter(legacyConfig *DaemonConfig) *ConfigAdapter {
	adapter := NewLegacyTestSetAdapter()

	// Create enhanced config
	enhancedConfig := &EnhancedDaemonConfig{
		ExecutionStrategy:     ExecutionStrategy("random"), // Default
		MaxConcurrency:        legacyConfig.MaxConcurrency,
		ShutdownGracePeriod:   30 * time.Second, // Default
		MetricsExportInterval: legacyConfig.ReportInterval,
		TestConfigs:           make(map[core.TestID]*TestExecutionConfig),
	}

	// Add test sets
	for _, testSet := range legacyConfig.TestSets {
		adapter.AddTestSet(testSet)

		// Create a default config for this test set
		testConfig := &TestExecutionConfig{
			Frequency:       legacyConfig.ExecutionInterval * time.Duration(testSet.ExecutionWeight),
			MaxParallelSelf: 1,                                  // Default
			Jitter:          legacyConfig.ExecutionInterval / 2, // Default jitter
		}

		enhancedConfig.TestConfigs[core.TestID(testSet.Name)] = testConfig
	}

	return &ConfigAdapter{
		legacyConfig:   legacyConfig,
		enhancedConfig: enhancedConfig,
		adapter:        adapter,
	}
}

// GetEnhancedConfig returns the enhanced config
func (a *ConfigAdapter) GetEnhancedConfig() *EnhancedDaemonConfig {
	return a.enhancedConfig
}

// GetTestStore returns the test scenario store
func (a *ConfigAdapter) GetTestStore() *TestScenarioStore {
	return a.adapter.GetStore()
}

// RegisterComponent adds a component to the adapter
func (a *ConfigAdapter) RegisterComponent(name string, component scenarios.Component) {
	a.adapter.RegisterComponent(name, component)
}

// BuildRunner creates an execution scheduler for the enhanced config
func (a *ConfigAdapter) BuildRunner(logger Logger) *ExecutionScheduler {
	return NewExecutionScheduler(a.enhancedConfig, a.adapter.GetStore(), logger)
}

// UpdateFromLegacy updates the enhanced config from legacy config
func (a *ConfigAdapter) UpdateFromLegacy() {
	// Update test sets
	for _, testSet := range a.legacyConfig.TestSets {
		a.adapter.AddTestSet(testSet)

		// Update config for this test set
		testConfig, exists := a.enhancedConfig.TestConfigs[core.TestID(testSet.Name)]
		if !exists {
			testConfig = &TestExecutionConfig{}
			a.enhancedConfig.TestConfigs[core.TestID(testSet.Name)] = testConfig
		}

		testConfig.Frequency = a.legacyConfig.ExecutionInterval * time.Duration(testSet.ExecutionWeight)
	}

	// Update global settings
	a.enhancedConfig.MaxConcurrency = a.legacyConfig.MaxConcurrency
	a.enhancedConfig.MetricsExportInterval = a.legacyConfig.ReportInterval
}

// GetLegacyConfig returns the legacy config
func (a *ConfigAdapter) GetLegacyConfig() *DaemonConfig {
	return a.legacyConfig
}
