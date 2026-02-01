package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load loads configuration from one or more YAML files.
// Files are merged in order, with later files overriding earlier ones.
func Load(paths ...string) (*Config, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one config path is required")
	}

	config := &Config{
		Infrastructure: make(map[string]InfraConfig),
		Suites:         make(map[string]SuiteConfig),
		ChaosProfiles:  make(map[string]ChaosProfile),
		MockProfiles:   make(map[string]MockProfile),
		Options:        make(map[string]OptionConfig),
	}

	for _, path := range paths {
		if err := loadFile(path, config); err != nil {
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}
	}

	return config, nil
}

// LoadWithOverlay loads a base config and overlays environment-specific config.
func LoadWithOverlay(base string, overlay string) (*Config, error) {
	if overlay == "" {
		return Load(base)
	}
	return Load(base, overlay)
}

// LoadFromDir loads all YAML files from a directory.
func LoadFromDir(dir string) (*Config, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
			paths = append(paths, filepath.Join(dir, name))
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no YAML files found in %s", dir)
	}

	return Load(paths...)
}

// loadFile loads a single YAML file into the config.
func loadFile(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Decode into a temporary config for merging
	var fileConfig Config
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("parsing YAML: %w", err)
	}

	// Merge the file config into the main config
	mergeConfig(config, &fileConfig)
	return nil
}

// mergeConfig merges src into dst.
func mergeConfig(dst, src *Config) {
	// Simple fields - overwrite if set
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.Version != "" {
		dst.Version = src.Version
	}

	// Discovery
	if len(src.Discovery.Paths) > 0 {
		dst.Discovery.Paths = append(dst.Discovery.Paths, src.Discovery.Paths...)
	}
	if len(src.Discovery.Exclude) > 0 {
		dst.Discovery.Exclude = append(dst.Discovery.Exclude, src.Discovery.Exclude...)
	}

	// Infrastructure - merge maps
	if dst.Infrastructure == nil {
		dst.Infrastructure = make(map[string]InfraConfig)
	}
	for k, v := range src.Infrastructure {
		dst.Infrastructure[k] = v
	}

	// Scenarios - append
	dst.Scenarios = append(dst.Scenarios, src.Scenarios...)

	// Suites - merge maps
	if dst.Suites == nil {
		dst.Suites = make(map[string]SuiteConfig)
	}
	for k, v := range src.Suites {
		dst.Suites[k] = v
	}

	// ChaosProfiles - merge maps
	if dst.ChaosProfiles == nil {
		dst.ChaosProfiles = make(map[string]ChaosProfile)
	}
	for k, v := range src.ChaosProfiles {
		dst.ChaosProfiles[k] = v
	}

	// MockProfiles - merge maps
	if dst.MockProfiles == nil {
		dst.MockProfiles = make(map[string]MockProfile)
	}
	for k, v := range src.MockProfiles {
		dst.MockProfiles[k] = v
	}

	// Flags
	if src.Flags.Definitions != nil {
		if dst.Flags.Definitions == nil {
			dst.Flags.Definitions = make(map[string]FlagDefinition)
		}
		for k, v := range src.Flags.Definitions {
			dst.Flags.Definitions[k] = v
		}
	}
	if src.Flags.Defaults != nil {
		if dst.Flags.Defaults == nil {
			dst.Flags.Defaults = make(map[string]any)
		}
		for k, v := range src.Flags.Defaults {
			dst.Flags.Defaults[k] = v
		}
	}

	// Options - merge maps
	if dst.Options == nil {
		dst.Options = make(map[string]OptionConfig)
	}
	for k, v := range src.Options {
		dst.Options[k] = v
	}

	// Bundles
	if src.Bundles.Infrastructure != nil {
		if dst.Bundles.Infrastructure == nil {
			dst.Bundles.Infrastructure = make(map[string][]string)
		}
		for k, v := range src.Bundles.Infrastructure {
			dst.Bundles.Infrastructure[k] = v
		}
	}
	if src.Bundles.Flags != nil {
		if dst.Bundles.Flags == nil {
			dst.Bundles.Flags = make(map[string][]string)
		}
		for k, v := range src.Bundles.Flags {
			dst.Bundles.Flags[k] = v
		}
	}
	if src.Bundles.Options != nil {
		if dst.Bundles.Options == nil {
			dst.Bundles.Options = make(map[string][]string)
		}
		for k, v := range src.Bundles.Options {
			dst.Bundles.Options[k] = v
		}
	}
	if src.Bundles.Middleware != nil {
		if dst.Bundles.Middleware == nil {
			dst.Bundles.Middleware = make(map[string][]string)
		}
		for k, v := range src.Bundles.Middleware {
			dst.Bundles.Middleware[k] = v
		}
	}

	// Secrets
	if src.Secrets.Provider != "" {
		dst.Secrets = src.Secrets
	}

	// Execution
	if src.Execution.Parallelism > 0 {
		dst.Execution.Parallelism = src.Execution.Parallelism
	}
	if src.Execution.DefaultTimeout > 0 {
		dst.Execution.DefaultTimeout = src.Execution.DefaultTimeout
	}
	if src.Execution.RetryConfig.MaxRetries > 0 {
		dst.Execution.RetryConfig = src.Execution.RetryConfig
	}
	if src.Execution.TeardownMode != "" {
		dst.Execution.TeardownMode = src.Execution.TeardownMode
	}
	if src.Execution.FailFast {
		dst.Execution.FailFast = src.Execution.FailFast
	}
	if src.Execution.ShuffleScenarios {
		dst.Execution.ShuffleScenarios = src.Execution.ShuffleScenarios
	}

	// Results
	if src.Results.Storage.Type != "" {
		dst.Results.Storage = src.Results.Storage
	}
	if len(src.Results.Reports) > 0 {
		dst.Results.Reports = append(dst.Results.Reports, src.Results.Reports...)
	}
	if src.Results.Retention.Days > 0 || src.Results.Retention.Count > 0 {
		dst.Results.Retention = src.Results.Retention
	}

	// Notifications
	if src.Notifications.Slack.Enabled {
		dst.Notifications.Slack = src.Notifications.Slack
	}
	if src.Notifications.Email.Enabled {
		dst.Notifications.Email = src.Notifications.Email
	}
	if src.Notifications.Webhook.Enabled {
		dst.Notifications.Webhook = src.Notifications.Webhook
	}
}

// GetScenario returns a scenario by name.
func (c *Config) GetScenario(name string) (*ScenarioConfig, bool) {
	for i := range c.Scenarios {
		if c.Scenarios[i].Name == name {
			return &c.Scenarios[i], true
		}
	}
	return nil, false
}

// GetScenariosByTag returns all scenarios matching any of the given tags.
func (c *Config) GetScenariosByTag(tags ...string) []ScenarioConfig {
	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}

	var result []ScenarioConfig
	for _, s := range c.Scenarios {
		for _, t := range s.Tags {
			if tagSet[t] {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

// GetNonAbstractScenarios returns all runnable (non-abstract) scenarios.
func (c *Config) GetNonAbstractScenarios() []ScenarioConfig {
	var result []ScenarioConfig
	for _, s := range c.Scenarios {
		if !s.Abstract {
			result = append(result, s)
		}
	}
	return result
}

// GetSuite returns a suite by name.
func (c *Config) GetSuite(name string) (*SuiteConfig, bool) {
	if c.Suites == nil {
		return nil, false
	}
	suite, ok := c.Suites[name]
	if !ok {
		return nil, false
	}
	return &suite, true
}

// GetSuiteScenarios returns all scenario names that belong to a suite.
// It combines explicitly named scenarios with tag-filtered scenarios.
func (c *Config) GetSuiteScenarios(name string) ([]string, bool) {
	suite, ok := c.GetSuite(name)
	if !ok {
		return nil, false
	}

	// Start with explicitly named scenarios
	scenarioSet := make(map[string]bool)
	for _, s := range suite.Scenarios {
		scenarioSet[s] = true
	}

	// Add scenarios matching tags
	if len(suite.Tags) > 0 {
		tagSet := make(map[string]bool)
		for _, t := range suite.Tags {
			tagSet[t] = true
		}
		for _, s := range c.Scenarios {
			if s.Abstract {
				continue
			}
			for _, t := range s.Tags {
				if tagSet[t] {
					scenarioSet[s.Name] = true
					break
				}
			}
		}
	}

	// Remove scenarios matching exclude tags
	if len(suite.ExcludeTags) > 0 {
		excludeSet := make(map[string]bool)
		for _, t := range suite.ExcludeTags {
			excludeSet[t] = true
		}
		for _, s := range c.Scenarios {
			for _, t := range s.Tags {
				if excludeSet[t] {
					delete(scenarioSet, s.Name)
					break
				}
			}
		}
	}

	// Convert to slice
	var result []string
	for name := range scenarioSet {
		result = append(result, name)
	}
	return result, true
}

// ListSuites returns all suite names.
func (c *Config) ListSuites() []string {
	if c.Suites == nil {
		return nil
	}
	var result []string
	for name := range c.Suites {
		result = append(result, name)
	}
	return result
}
