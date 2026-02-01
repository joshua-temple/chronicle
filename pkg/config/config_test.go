package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("loads single file", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "chronicle.yaml")
		err := os.WriteFile(configPath, []byte(`
name: test-project
version: "1.0"

scenarios:
  - name: basic-test
    description: "A basic test scenario"
    timeout: 30s
    tags: [smoke, basic]
    flow:
      - setup: CreateUser
      - task: DoSomething
      - validation: CheckResult
`), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		config, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if config.Name != "test-project" {
			t.Errorf("expected name 'test-project', got %s", config.Name)
		}
		if config.Version != "1.0" {
			t.Errorf("expected version '1.0', got %s", config.Version)
		}
		if len(config.Scenarios) != 1 {
			t.Errorf("expected 1 scenario, got %d", len(config.Scenarios))
		}

		s := config.Scenarios[0]
		if s.Name != "basic-test" {
			t.Errorf("expected scenario name 'basic-test', got %s", s.Name)
		}
		if s.Timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", s.Timeout)
		}
		if len(s.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(s.Tags))
		}
		if len(s.Flow) != 3 {
			t.Errorf("expected 3 flow items, got %d", len(s.Flow))
		}
	})

	t.Run("merges multiple files", func(t *testing.T) {
		dir := t.TempDir()

		// Base config
		basePath := filepath.Join(dir, "base.yaml")
		err := os.WriteFile(basePath, []byte(`
name: base-project
scenarios:
  - name: scenario-1
    flow:
      - setup: Setup1

infrastructure:
  postgres:
    provider: testcontainers
    image: postgres:15
`), 0644)
		if err != nil {
			t.Fatalf("failed to write base config: %v", err)
		}

		// Overlay config
		overlayPath := filepath.Join(dir, "overlay.yaml")
		err = os.WriteFile(overlayPath, []byte(`
name: overlay-project
scenarios:
  - name: scenario-2
    flow:
      - setup: Setup2

infrastructure:
  redis:
    provider: testcontainers
    image: redis:7
`), 0644)
		if err != nil {
			t.Fatalf("failed to write overlay config: %v", err)
		}

		config, err := Load(basePath, overlayPath)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// Name should be from overlay
		if config.Name != "overlay-project" {
			t.Errorf("expected name 'overlay-project', got %s", config.Name)
		}

		// Should have both scenarios
		if len(config.Scenarios) != 2 {
			t.Errorf("expected 2 scenarios, got %d", len(config.Scenarios))
		}

		// Should have both infrastructure items
		if len(config.Infrastructure) != 2 {
			t.Errorf("expected 2 infrastructure items, got %d", len(config.Infrastructure))
		}
		if _, ok := config.Infrastructure["postgres"]; !ok {
			t.Error("expected postgres infrastructure")
		}
		if _, ok := config.Infrastructure["redis"]; !ok {
			t.Error("expected redis infrastructure")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := Load("/non/existent/path.yaml")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "invalid.yaml")
		err := os.WriteFile(configPath, []byte(`
name: test
scenarios:
  - name: test
    flow
      invalid yaml here
`), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		_, err = Load(configPath)
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

func TestLoadWithOverlay(t *testing.T) {
	t.Run("loads base only when overlay is empty", func(t *testing.T) {
		dir := t.TempDir()
		basePath := filepath.Join(dir, "base.yaml")
		err := os.WriteFile(basePath, []byte(`
name: base-project
`), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		config, err := LoadWithOverlay(basePath, "")
		if err != nil {
			t.Fatalf("LoadWithOverlay failed: %v", err)
		}

		if config.Name != "base-project" {
			t.Errorf("expected name 'base-project', got %s", config.Name)
		}
	})
}

func TestLoadFromDir(t *testing.T) {
	t.Run("loads all YAML files in directory", func(t *testing.T) {
		dir := t.TempDir()

		err := os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(`
name: project
`), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		err = os.WriteFile(filepath.Join(dir, "scenarios.yml"), []byte(`
scenarios:
  - name: test-1
    flow:
      - setup: Setup1
`), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Non-YAML file should be ignored
		err = os.WriteFile(filepath.Join(dir, "readme.md"), []byte(`# README`), 0644)
		if err != nil {
			t.Fatalf("failed to write readme: %v", err)
		}

		config, err := LoadFromDir(dir)
		if err != nil {
			t.Fatalf("LoadFromDir failed: %v", err)
		}

		if config.Name != "project" {
			t.Errorf("expected name 'project', got %s", config.Name)
		}
		if len(config.Scenarios) != 1 {
			t.Errorf("expected 1 scenario, got %d", len(config.Scenarios))
		}
	})

	t.Run("returns error for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		_, err := LoadFromDir(dir)
		if err == nil {
			t.Error("expected error for empty directory")
		}
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("validates valid config", func(t *testing.T) {
		config := &Config{
			Name:    "test",
			Version: "1.0",
			Scenarios: []ScenarioConfig{
				{
					Name: "test-scenario",
					Flow: []FlowItemConfig{
						{Setup: "CreateUser"},
					},
				},
			},
			Infrastructure: make(map[string]InfraConfig),
			ChaosProfiles:  make(map[string]ChaosProfile),
			MockProfiles:   make(map[string]MockProfile),
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("detects duplicate scenario names", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "test", Flow: []FlowItemConfig{{Setup: "S1"}}},
				{Name: "test", Flow: []FlowItemConfig{{Setup: "S2"}}},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for duplicate scenario names")
		}
	})

	t.Run("detects non-abstract scenario without flow", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "test-scenario"},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for scenario without flow")
		}
	})

	t.Run("allows abstract scenario without flow", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "abstract-base", Abstract: true},
				{Name: "concrete", Extends: "abstract-base", Flow: []FlowItemConfig{{Setup: "S1"}}},
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("expected no error for abstract scenario, got: %v", err)
		}
	})

	t.Run("detects extending non-existent scenario", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "test", Extends: "non-existent", Flow: []FlowItemConfig{{Setup: "S1"}}},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for extending non-existent scenario")
		}
	})

	t.Run("detects extending non-abstract scenario", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "base", Flow: []FlowItemConfig{{Setup: "S1"}}},
				{Name: "child", Extends: "base", Flow: []FlowItemConfig{{Setup: "S2"}}},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for extending non-abstract scenario")
		}
	})

	t.Run("detects flow item with no component type", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "test", Flow: []FlowItemConfig{{}}},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for flow item with no component")
		}
	})

	t.Run("detects flow item with multiple component types", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "test", Flow: []FlowItemConfig{{Setup: "S1", Task: "T1"}}},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for flow item with multiple components")
		}
	})

	t.Run("validates chaos profile", func(t *testing.T) {
		config := &Config{
			ChaosProfiles: map[string]ChaosProfile{
				"invalid": {
					Network: NetworkChaosConfig{
						PacketLoss: PacketLossConfig{
							Enabled:    true,
							Percentage: 150, // Invalid
						},
					},
				},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for invalid packet loss percentage")
		}
	})

	t.Run("validates execution config", func(t *testing.T) {
		config := &Config{
			Execution: ExecutionConfig{
				TeardownMode: "invalid_mode",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for invalid teardown mode")
		}
	})
}

func TestConfigGetScenario(t *testing.T) {
	config := &Config{
		Scenarios: []ScenarioConfig{
			{Name: "scenario-1"},
			{Name: "scenario-2"},
		},
	}

	t.Run("finds existing scenario", func(t *testing.T) {
		s, ok := config.GetScenario("scenario-1")
		if !ok {
			t.Error("expected to find scenario-1")
		}
		if s.Name != "scenario-1" {
			t.Errorf("expected scenario-1, got %s", s.Name)
		}
	})

	t.Run("returns false for non-existent scenario", func(t *testing.T) {
		_, ok := config.GetScenario("non-existent")
		if ok {
			t.Error("expected not to find non-existent scenario")
		}
	})
}

func TestConfigGetScenariosByTag(t *testing.T) {
	config := &Config{
		Scenarios: []ScenarioConfig{
			{Name: "s1", Tags: []string{"smoke", "api"}},
			{Name: "s2", Tags: []string{"smoke", "ui"}},
			{Name: "s3", Tags: []string{"integration"}},
		},
	}

	t.Run("finds scenarios by single tag", func(t *testing.T) {
		scenarios := config.GetScenariosByTag("smoke")
		if len(scenarios) != 2 {
			t.Errorf("expected 2 scenarios with smoke tag, got %d", len(scenarios))
		}
	})

	t.Run("finds scenarios by multiple tags", func(t *testing.T) {
		scenarios := config.GetScenariosByTag("api", "integration")
		if len(scenarios) != 2 {
			t.Errorf("expected 2 scenarios, got %d", len(scenarios))
		}
	})

	t.Run("returns empty for non-matching tag", func(t *testing.T) {
		scenarios := config.GetScenariosByTag("nonexistent")
		if len(scenarios) != 0 {
			t.Errorf("expected 0 scenarios, got %d", len(scenarios))
		}
	})
}

func TestConfigGetNonAbstractScenarios(t *testing.T) {
	config := &Config{
		Scenarios: []ScenarioConfig{
			{Name: "abstract-base", Abstract: true},
			{Name: "concrete-1"},
			{Name: "concrete-2"},
		},
	}

	scenarios := config.GetNonAbstractScenarios()
	if len(scenarios) != 2 {
		t.Errorf("expected 2 non-abstract scenarios, got %d", len(scenarios))
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Name != "chronicle" {
		t.Errorf("expected default name 'chronicle', got %s", config.Name)
	}
	if config.Execution.TeardownMode != "always" {
		t.Errorf("expected default teardown mode 'always', got %s", config.Execution.TeardownMode)
	}
	if config.Results.Storage.Type != "file" {
		t.Errorf("expected default storage type 'file', got %s", config.Results.Storage.Type)
	}
}

func TestSuiteConfiguration(t *testing.T) {
	t.Run("loads suites from config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "chronicle.yaml")
		err := os.WriteFile(configPath, []byte(`
name: test-project
version: "1.0"

scenarios:
  - name: smoke-1
    tags: [smoke]
    flow:
      - setup: CreateUser
  - name: smoke-2
    tags: [smoke]
    flow:
      - task: DoSomething
  - name: integration-1
    tags: [integration]
    flow:
      - validation: CheckResult

suites:
  smoke-suite:
    description: "Run smoke tests"
    tags: [smoke]
    parallel: 2
    fail_fast: true
  full-suite:
    description: "Run all tests"
    scenarios: [smoke-1, smoke-2, integration-1]
  mixed-suite:
    description: "Mixed selection"
    scenarios: [smoke-1]
    tags: [integration]
    exclude_tags: []
`), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		config, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if len(config.Suites) != 3 {
			t.Errorf("expected 3 suites, got %d", len(config.Suites))
		}

		// Check smoke-suite
		smokeSuite, ok := config.GetSuite("smoke-suite")
		if !ok {
			t.Fatal("expected to find smoke-suite")
		}
		if smokeSuite.Description != "Run smoke tests" {
			t.Errorf("expected description 'Run smoke tests', got %s", smokeSuite.Description)
		}
		if smokeSuite.Parallel != 2 {
			t.Errorf("expected parallel 2, got %d", smokeSuite.Parallel)
		}
		if !smokeSuite.FailFast {
			t.Error("expected FailFast to be true")
		}

		// Check full-suite
		fullSuite, ok := config.GetSuite("full-suite")
		if !ok {
			t.Fatal("expected to find full-suite")
		}
		if len(fullSuite.Scenarios) != 3 {
			t.Errorf("expected 3 scenarios in full-suite, got %d", len(fullSuite.Scenarios))
		}
	})

	t.Run("GetSuiteScenarios resolves by scenarios", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "s1", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s2", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s3", Tags: []string{"integration"}, Flow: []FlowItemConfig{{Setup: "S"}}},
			},
			Suites: map[string]SuiteConfig{
				"explicit": {Scenarios: []string{"s1", "s3"}},
			},
		}

		scenarios, ok := config.GetSuiteScenarios("explicit")
		if !ok {
			t.Fatal("expected to find suite")
		}
		if len(scenarios) != 2 {
			t.Errorf("expected 2 scenarios, got %d", len(scenarios))
		}
	})

	t.Run("GetSuiteScenarios resolves by tags", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "s1", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s2", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s3", Tags: []string{"integration"}, Flow: []FlowItemConfig{{Setup: "S"}}},
			},
			Suites: map[string]SuiteConfig{
				"by-tag": {Tags: []string{"smoke"}},
			},
		}

		scenarios, ok := config.GetSuiteScenarios("by-tag")
		if !ok {
			t.Fatal("expected to find suite")
		}
		if len(scenarios) != 2 {
			t.Errorf("expected 2 scenarios with smoke tag, got %d", len(scenarios))
		}
	})

	t.Run("GetSuiteScenarios respects exclude_tags", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "s1", Tags: []string{"smoke", "slow"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s2", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s3", Tags: []string{"integration"}, Flow: []FlowItemConfig{{Setup: "S"}}},
			},
			Suites: map[string]SuiteConfig{
				"fast-smoke": {Tags: []string{"smoke"}, ExcludeTags: []string{"slow"}},
			},
		}

		scenarios, ok := config.GetSuiteScenarios("fast-smoke")
		if !ok {
			t.Fatal("expected to find suite")
		}
		if len(scenarios) != 1 {
			t.Errorf("expected 1 scenario (excluding slow), got %d", len(scenarios))
		}
		if scenarios[0] != "s2" {
			t.Errorf("expected scenario s2, got %s", scenarios[0])
		}
	})

	t.Run("GetSuiteScenarios combines scenarios and tags", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "s1", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s2", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
				{Name: "s3", Tags: []string{"integration"}, Flow: []FlowItemConfig{{Setup: "S"}}},
			},
			Suites: map[string]SuiteConfig{
				"combined": {Scenarios: []string{"s3"}, Tags: []string{"smoke"}},
			},
		}

		scenarios, ok := config.GetSuiteScenarios("combined")
		if !ok {
			t.Fatal("expected to find suite")
		}
		if len(scenarios) != 3 {
			t.Errorf("expected 3 scenarios (s3 + smoke tagged), got %d", len(scenarios))
		}
	})

	t.Run("ListSuites returns all suite names", func(t *testing.T) {
		config := &Config{
			Suites: map[string]SuiteConfig{
				"suite-a": {},
				"suite-b": {},
				"suite-c": {},
			},
		}

		names := config.ListSuites()
		if len(names) != 3 {
			t.Errorf("expected 3 suite names, got %d", len(names))
		}
	})

	t.Run("GetSuiteScenarios returns false for non-existent suite", func(t *testing.T) {
		config := &Config{Suites: make(map[string]SuiteConfig)}
		_, ok := config.GetSuiteScenarios("non-existent")
		if ok {
			t.Error("expected false for non-existent suite")
		}
	})
}

func TestSuiteValidation(t *testing.T) {
	t.Run("validates suite with non-existent scenario", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "s1", Flow: []FlowItemConfig{{Setup: "S"}}},
			},
			Suites: map[string]SuiteConfig{
				"bad-suite": {Scenarios: []string{"s1", "non-existent"}},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for suite referencing non-existent scenario")
		}
	})

	t.Run("validates suite with invalid parallel value", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "s1", Flow: []FlowItemConfig{{Setup: "S"}}},
			},
			Suites: map[string]SuiteConfig{
				"bad-suite": {Scenarios: []string{"s1"}, Parallel: -1},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for negative parallel value")
		}
	})

	t.Run("valid suite passes validation", func(t *testing.T) {
		config := &Config{
			Scenarios: []ScenarioConfig{
				{Name: "s1", Tags: []string{"smoke"}, Flow: []FlowItemConfig{{Setup: "S"}}},
			},
			Suites: map[string]SuiteConfig{
				"good-suite": {Scenarios: []string{"s1"}, Tags: []string{"smoke"}, Parallel: 2},
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("expected no validation error, got: %v", err)
		}
	})
}
