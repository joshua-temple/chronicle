package scenario

import (
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
)

func TestNewResolver(t *testing.T) {
	cfg := &config.Config{}
	reg := &discovery.Registry{}

	resolver := NewResolver(cfg, reg)

	if resolver == nil {
		t.Fatal("NewResolver should not return nil")
	}

	if resolver.config != cfg {
		t.Error("Resolver config not set correctly")
	}

	if resolver.registry != reg {
		t.Error("Resolver registry not set correctly")
	}
}

func TestResolver_Resolve(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:        "test-scenario",
				Description: "Test description",
				Timeout:     5 * time.Minute,
				Tags:        []string{"unit"},
				Flow: []config.FlowItemConfig{
					{Task: "TestTask"},
				},
			},
		},
	}

	reg := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{},
	}

	resolver := NewResolver(cfg, reg)

	scenario, err := resolver.Resolve("test-scenario")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	if scenario.Name != "test-scenario" {
		t.Errorf("Name = %q, expected 'test-scenario'", scenario.Name)
	}

	if scenario.Description != "Test description" {
		t.Errorf("Description = %q, expected 'Test description'", scenario.Description)
	}

	if scenario.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, expected 5m", scenario.Timeout)
	}
}

func TestResolver_ResolveNotFound(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{},
	}

	resolver := NewResolver(cfg, nil)

	_, err := resolver.Resolve("non-existent")
	if err == nil {
		t.Error("Resolve() should return error for non-existent scenario")
	}
}

func TestResolver_ResolveAll(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name: "scenario-1",
				Flow: []config.FlowItemConfig{{Task: "Task1"}},
			},
			{
				Name: "scenario-2",
				Flow: []config.FlowItemConfig{{Task: "Task2"}},
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	scenarios, err := resolver.ResolveAll()
	if err != nil {
		t.Fatalf("ResolveAll() unexpected error: %v", err)
	}

	if len(scenarios) != 2 {
		t.Errorf("Expected 2 scenarios, got %d", len(scenarios))
	}
}

func TestResolver_ResolveAllSkipsAbstract(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:     "abstract-base",
				Abstract: true,
				Flow:     []config.FlowItemConfig{{Setup: "BaseSetup"}},
			},
			{
				Name:    "concrete",
				Extends: "abstract-base",
				Flow:    []config.FlowItemConfig{{Task: "Task1"}},
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	scenarios, err := resolver.ResolveAll()
	if err != nil {
		t.Fatalf("ResolveAll() unexpected error: %v", err)
	}

	// Should only include concrete scenario
	if len(scenarios) != 1 {
		t.Errorf("Expected 1 scenario (abstract excluded), got %d", len(scenarios))
	}

	if scenarios[0].Name != "concrete" {
		t.Errorf("Expected 'concrete', got %q", scenarios[0].Name)
	}
}

func TestResolver_InheritanceFlow(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:     "base",
				Abstract: true,
				Flow: []config.FlowItemConfig{
					{Setup: "BaseSetup"},
				},
			},
			{
				Name:    "child",
				Extends: "base",
				Flow: []config.FlowItemConfig{
					{Task: "ChildTask"},
				},
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	scenario, err := resolver.Resolve("child")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	// Flow should have parent items first, then child items
	if len(scenario.Flow) != 2 {
		t.Errorf("Expected 2 flow items (inherited + own), got %d", len(scenario.Flow))
	}

	if scenario.Flow[0].Name != "BaseSetup" {
		t.Errorf("First flow item = %q, expected 'BaseSetup'", scenario.Flow[0].Name)
	}

	if scenario.Flow[1].Name != "ChildTask" {
		t.Errorf("Second flow item = %q, expected 'ChildTask'", scenario.Flow[1].Name)
	}
}

func TestResolver_InheritanceTags(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:     "base",
				Abstract: true,
				Tags:     []string{"parent-tag"},
			},
			{
				Name:    "child",
				Extends: "base",
				Tags:    []string{"child-tag"},
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	scenario, err := resolver.Resolve("child")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	// Should have both parent and child tags
	if len(scenario.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(scenario.Tags))
	}

	// Check both tags exist (order may vary due to map iteration)
	tagMap := make(map[string]bool)
	for _, tag := range scenario.Tags {
		tagMap[tag] = true
	}

	if !tagMap["parent-tag"] {
		t.Error("Missing 'parent-tag' from inherited tags")
	}
	if !tagMap["child-tag"] {
		t.Error("Missing 'child-tag' from own tags")
	}
}

func TestResolver_InheritanceFlags(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:     "base",
				Abstract: true,
				Flags: map[string]any{
					"base-flag": "base-value",
					"override":  "base",
				},
			},
			{
				Name:    "child",
				Extends: "base",
				Flags: map[string]any{
					"child-flag": "child-value",
					"override":   "child",
				},
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	scenario, err := resolver.Resolve("child")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	// Child flag should override parent
	if scenario.Flags["override"] != "child" {
		t.Errorf("override flag = %v, expected 'child'", scenario.Flags["override"])
	}

	// Both base and child flags should exist
	if scenario.Flags["base-flag"] != "base-value" {
		t.Errorf("base-flag = %v, expected 'base-value'", scenario.Flags["base-flag"])
	}

	if scenario.Flags["child-flag"] != "child-value" {
		t.Errorf("child-flag = %v, expected 'child-value'", scenario.Flags["child-flag"])
	}
}

func TestResolver_InheritanceTimeout(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:     "base",
				Abstract: true,
				Timeout:  10 * time.Minute,
			},
			{
				Name:    "child-no-timeout",
				Extends: "base",
			},
			{
				Name:    "child-with-timeout",
				Extends: "base",
				Timeout: 5 * time.Minute,
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	// Child without timeout inherits parent
	scenario1, err := resolver.Resolve("child-no-timeout")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if scenario1.Timeout != 10*time.Minute {
		t.Errorf("Inherited timeout = %v, expected 10m", scenario1.Timeout)
	}

	// Child with timeout overrides parent
	scenario2, err := resolver.Resolve("child-with-timeout")
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if scenario2.Timeout != 5*time.Minute {
		t.Errorf("Overridden timeout = %v, expected 5m", scenario2.Timeout)
	}
}

func TestResolver_NonAbstractExtends(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:     "not-abstract",
				Abstract: false,
			},
			{
				Name:    "child",
				Extends: "not-abstract",
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	_, err := resolver.Resolve("child")
	if err == nil {
		t.Error("Should error when extending non-abstract scenario")
	}
}

func TestResolver_ParentNotFound(t *testing.T) {
	cfg := &config.Config{
		Scenarios: []config.ScenarioConfig{
			{
				Name:    "child",
				Extends: "non-existent-parent",
			},
		},
	}

	resolver := NewResolver(cfg, nil)

	_, err := resolver.Resolve("child")
	if err == nil {
		t.Error("Should error when parent scenario not found")
	}
}

func TestResolver_ResolveFlowItem(t *testing.T) {
	reg := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"TestTask": {
				Name: "TestTask",
				Type: core.ComponentTask,
			},
		},
	}

	resolver := NewResolver(&config.Config{}, reg)

	tests := []struct {
		name         string
		flowCfg      *config.FlowItemConfig
		expectedType core.ComponentType
		expectedName string
		expectError  bool
	}{
		{
			name:         "setup component",
			flowCfg:      &config.FlowItemConfig{Setup: "SetupDB"},
			expectedType: core.ComponentSetup,
			expectedName: "SetupDB",
		},
		{
			name:         "task component",
			flowCfg:      &config.FlowItemConfig{Task: "TestTask"},
			expectedType: core.ComponentTask,
			expectedName: "TestTask",
		},
		{
			name:         "validation component",
			flowCfg:      &config.FlowItemConfig{Validation: "Validate"},
			expectedType: core.ComponentValidation,
			expectedName: "Validate",
		},
		{
			name:         "step component",
			flowCfg:      &config.FlowItemConfig{Step: "StepOne"},
			expectedType: core.ComponentStep,
			expectedName: "StepOne",
		},
		{
			name:         "rollup component",
			flowCfg:      &config.FlowItemConfig{Rollup: "Aggregate"},
			expectedType: core.ComponentRollup,
			expectedName: "Aggregate",
		},
		{
			name:         "teardown component",
			flowCfg:      &config.FlowItemConfig{Teardown: "Cleanup"},
			expectedType: core.ComponentTeardown,
			expectedName: "Cleanup",
		},
		{
			name:        "no component type",
			flowCfg:     &config.FlowItemConfig{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := resolver.resolveFlowItem(tt.flowCfg)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if item.Type != tt.expectedType {
				t.Errorf("Type = %q, expected %q", item.Type, tt.expectedType)
			}

			if item.Name != tt.expectedName {
				t.Errorf("Name = %q, expected %q", item.Name, tt.expectedName)
			}
		})
	}
}

func TestResolver_ConvertCondition(t *testing.T) {
	resolver := NewResolver(&config.Config{}, nil)

	tests := []struct {
		name               string
		condCfg            *config.ConditionConfig
		expectedExpression string
		expectedReason     string
	}{
		{
			name: "expression condition",
			condCfg: &config.ConditionConfig{
				Expression: "env.CI == true",
				Reason:     "Skip in CI",
			},
			expectedExpression: "env.CI == true",
			expectedReason:     "Skip in CI",
		},
		{
			name: "env condition",
			condCfg: &config.ConditionConfig{
				Env:    "CI",
				Reason: "CI environment",
			},
			expectedExpression: "env.CI is set",
			expectedReason:     "CI environment",
		},
		{
			name: "flag condition",
			condCfg: &config.ConditionConfig{
				Flag:   "integration",
				Reason: "Integration mode",
			},
			expectedExpression: "flags.integration == true",
			expectedReason:     "Integration mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := resolver.convertCondition(tt.condCfg)

			if cond.Expression != tt.expectedExpression {
				t.Errorf("Expression = %q, expected %q", cond.Expression, tt.expectedExpression)
			}

			if cond.Reason != tt.expectedReason {
				t.Errorf("Reason = %q, expected %q", cond.Reason, tt.expectedReason)
			}
		})
	}
}

func TestResolver_GenerateCombinations(t *testing.T) {
	resolver := NewResolver(&config.Config{}, nil)

	tests := []struct {
		name           string
		matrix         map[string][]any
		keys           []string
		expectedCount  int
	}{
		{
			name:          "empty matrix",
			matrix:        map[string][]any{},
			keys:          []string{},
			expectedCount: 1, // One empty combination
		},
		{
			name: "single key",
			matrix: map[string][]any{
				"version": {"1.0", "2.0", "3.0"},
			},
			keys:          []string{"version"},
			expectedCount: 3,
		},
		{
			name: "two keys",
			matrix: map[string][]any{
				"version": {"1.0", "2.0"},
				"os":      {"linux", "darwin"},
			},
			keys:          []string{"version", "os"},
			expectedCount: 4, // 2 * 2
		},
		{
			name: "three keys",
			matrix: map[string][]any{
				"version": {"1", "2"},
				"os":      {"linux", "darwin"},
				"arch":    {"amd64", "arm64"},
			},
			keys:          []string{"version", "os", "arch"},
			expectedCount: 8, // 2 * 2 * 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combinations := resolver.generateCombinations(tt.matrix, tt.keys)

			if len(combinations) != tt.expectedCount {
				t.Errorf("Expected %d combinations, got %d", tt.expectedCount, len(combinations))
			}
		})
	}
}

func TestResolver_SubstituteParams(t *testing.T) {
	resolver := NewResolver(&config.Config{}, nil)

	matrixValues := map[string]any{
		"version": "2.0",
		"env":     "production",
	}

	tests := []struct {
		name     string
		params   map[string]any
		expected map[string]any
	}{
		{
			name:     "nil params",
			params:   nil,
			expected: nil,
		},
		{
			name: "no substitution needed",
			params: map[string]any{
				"key": "value",
			},
			expected: map[string]any{
				"key": "value",
			},
		},
		{
			name: "simple substitution",
			params: map[string]any{
				"ver": "${{ matrix.version }}",
			},
			expected: map[string]any{
				"ver": "2.0",
			},
		},
		{
			name: "substitution with spaces",
			params: map[string]any{
				"ver": "${{  matrix.version  }}",
			},
			expected: map[string]any{
				"ver": "2.0",
			},
		},
		{
			name: "multiple substitutions",
			params: map[string]any{
				"ver": "${{ matrix.version }}",
				"env": "${{ matrix.env }}",
			},
			expected: map[string]any{
				"ver": "2.0",
				"env": "production",
			},
		},
		{
			name: "unknown matrix key preserved",
			params: map[string]any{
				"key": "${{ matrix.unknown }}",
			},
			expected: map[string]any{
				"key": "${{ matrix.unknown }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.substituteParams(tt.params, matrixValues)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			for k, expected := range tt.expected {
				if result[k] != expected {
					t.Errorf("params[%q] = %v, expected %v", k, result[k], expected)
				}
			}
		})
	}
}

func TestResolver_ExpandMatrix(t *testing.T) {
	cfg := &config.Config{}
	resolver := NewResolver(cfg, nil)

	scenario := NewScenario("test")
	scenario.Matrix["version"] = []any{"1.0", "2.0"}
	scenario.Matrix["os"] = []any{"linux", "darwin"}
	scenario.Flow = []FlowItem{
		{Type: core.ComponentTask, Name: "Test"},
	}

	expanded, err := resolver.expandMatrix(scenario)
	if err != nil {
		t.Fatalf("expandMatrix() unexpected error: %v", err)
	}

	// Should have 4 scenarios (2 * 2)
	if len(expanded) != 4 {
		t.Errorf("Expected 4 expanded scenarios, got %d", len(expanded))
	}

	// Each should have a unique name
	names := make(map[string]bool)
	for _, s := range expanded {
		if names[s.Name] {
			t.Errorf("Duplicate scenario name: %s", s.Name)
		}
		names[s.Name] = true
	}
}

func TestResolver_ExpandMatrixEmpty(t *testing.T) {
	resolver := NewResolver(&config.Config{}, nil)

	scenario := NewScenario("test")
	// No matrix parameters

	expanded, err := resolver.expandMatrix(scenario)
	if err != nil {
		t.Fatalf("expandMatrix() unexpected error: %v", err)
	}

	// Should return original scenario
	if len(expanded) != 1 {
		t.Errorf("Expected 1 scenario (no expansion), got %d", len(expanded))
	}

	if expanded[0] != scenario {
		t.Error("Should return original scenario when no matrix")
	}
}
