package scenario

import (
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
)

func TestNewScenario(t *testing.T) {
	s := NewScenario("test-scenario")

	if s.Name != "test-scenario" {
		t.Errorf("expected name 'test-scenario', got %s", s.Name)
	}
	if s.ID == "" {
		t.Error("expected ID to be set")
	}
	if s.Flags == nil {
		t.Error("expected Flags map to be initialized")
	}
}

func TestScenario_AddFlow(t *testing.T) {
	s := NewScenario("test")

	s.AddFlow(NewFlowItem(core.ComponentSetup, "Setup1"))
	s.AddFlow(NewFlowItem(core.ComponentTask, "Task1"))

	if len(s.Flow) != 2 {
		t.Errorf("expected 2 flow items, got %d", len(s.Flow))
	}
	if s.Flow[0].Name != "Setup1" {
		t.Errorf("expected first item to be 'Setup1', got %s", s.Flow[0].Name)
	}
}

func TestScenario_SetFlag(t *testing.T) {
	s := NewScenario("test")

	s.SetFlag("debug", true)
	s.SetFlag("count", 5)

	val, ok := s.GetFlag("debug")
	if !ok {
		t.Error("expected to find 'debug' flag")
	}
	if val != true {
		t.Errorf("expected debug=true, got %v", val)
	}

	val, ok = s.GetFlag("count")
	if !ok {
		t.Error("expected to find 'count' flag")
	}
	if val != 5 {
		t.Errorf("expected count=5, got %v", val)
	}

	_, ok = s.GetFlag("nonexistent")
	if ok {
		t.Error("expected not to find 'nonexistent' flag")
	}
}

func TestScenario_HasTag(t *testing.T) {
	s := NewScenario("test")
	s.Tags = []string{"smoke", "api"}

	if !s.HasTag("smoke") {
		t.Error("expected to have 'smoke' tag")
	}
	if !s.HasTag("api") {
		t.Error("expected to have 'api' tag")
	}
	if s.HasTag("integration") {
		t.Error("expected not to have 'integration' tag")
	}
}

func TestScenario_IsRunnable(t *testing.T) {
	t.Run("abstract scenario is not runnable", func(t *testing.T) {
		s := NewScenario("abstract")
		s.Abstract = true
		s.AddFlow(NewFlowItem(core.ComponentSetup, "Setup1"))

		if s.IsRunnable() {
			t.Error("abstract scenario should not be runnable")
		}
	})

	t.Run("scenario without flow is not runnable", func(t *testing.T) {
		s := NewScenario("empty")

		if s.IsRunnable() {
			t.Error("scenario without flow should not be runnable")
		}
	})

	t.Run("concrete scenario with flow is runnable", func(t *testing.T) {
		s := NewScenario("runnable")
		s.AddFlow(NewFlowItem(core.ComponentSetup, "Setup1"))

		if !s.IsRunnable() {
			t.Error("concrete scenario with flow should be runnable")
		}
	})
}

func TestScenario_EffectiveTimeout(t *testing.T) {
	defaultTimeout := 30 * time.Second

	t.Run("uses scenario timeout if set", func(t *testing.T) {
		s := NewScenario("test")
		s.Timeout = 60 * time.Second

		if s.EffectiveTimeout(defaultTimeout) != 60*time.Second {
			t.Error("should use scenario timeout")
		}
	})

	t.Run("uses default timeout if not set", func(t *testing.T) {
		s := NewScenario("test")

		if s.EffectiveTimeout(defaultTimeout) != defaultTimeout {
			t.Error("should use default timeout")
		}
	})
}

func TestScenario_Clone(t *testing.T) {
	original := NewScenario("original")
	original.Description = "Original description"
	original.Tags = []string{"tag1", "tag2"}
	original.AddFlow(NewFlowItem(core.ComponentSetup, "Setup1"))
	original.SetFlag("debug", true)

	clone := original.Clone()

	// Check values are copied
	if clone.Name != original.Name {
		t.Error("name should be copied")
	}
	if clone.Description != original.Description {
		t.Error("description should be copied")
	}
	if len(clone.Tags) != len(original.Tags) {
		t.Error("tags should be copied")
	}
	if len(clone.Flow) != len(original.Flow) {
		t.Error("flow should be copied")
	}

	// Check independence
	clone.Tags = append(clone.Tags, "tag3")
	if len(original.Tags) != 2 {
		t.Error("modifying clone should not affect original")
	}

	clone.SetFlag("newFlag", "value")
	if _, ok := original.GetFlag("newFlag"); ok {
		t.Error("modifying clone flags should not affect original")
	}

	// IDs should be different
	if clone.ID == original.ID {
		t.Error("clone should have different ID")
	}
}

func TestBuilder(t *testing.T) {
	t.Run("builds complete scenario", func(t *testing.T) {
		s := NewBuilder("checkout-flow").
			Description("Complete checkout flow").
			Timeout(30 * time.Second).
			Tags("smoke", "checkout").
			Setup("CreateUser").
			Setup("SeedInventory").
			Task("ProcessPayment").
			Validation("VerifyOrder").
			Teardown("Cleanup").
			Flag("debug", true).
			Options("fast-mode").
			ChaosProfiles("network-chaos").
			Build()

		if s.Name != "checkout-flow" {
			t.Errorf("expected name 'checkout-flow', got %s", s.Name)
		}
		if s.Description != "Complete checkout flow" {
			t.Error("description not set")
		}
		if s.Timeout != 30*time.Second {
			t.Error("timeout not set")
		}
		if len(s.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(s.Tags))
		}
		if len(s.Flow) != 4 {
			t.Errorf("expected 4 flow items, got %d", len(s.Flow))
		}
		if len(s.TeardownFlow) != 1 {
			t.Errorf("expected 1 teardown item, got %d", len(s.TeardownFlow))
		}
	})

	t.Run("builds abstract scenario", func(t *testing.T) {
		s := NewBuilder("base").
			Abstract().
			Setup("CommonSetup").
			Build()

		if !s.Abstract {
			t.Error("scenario should be abstract")
		}
	})

	t.Run("builds inherited scenario", func(t *testing.T) {
		s := NewBuilder("child").
			Extends("parent").
			Task("ChildTask").
			Build()

		if s.Extends != "parent" {
			t.Errorf("expected extends 'parent', got %s", s.Extends)
		}
	})

	t.Run("builds scenario with parallel flow", func(t *testing.T) {
		s := NewBuilder("parallel-test").
			Setup("Setup").
			Parallel(
				NewFlowItem(core.ComponentTask, "Task1"),
				NewFlowItem(core.ComponentTask, "Task2"),
			).
			Validation("Validate").
			Build()

		if len(s.Flow) != 3 {
			t.Errorf("expected 3 flow items, got %d", len(s.Flow))
		}
		if !s.Flow[1].Parallel {
			t.Error("second item should be parallel block")
		}
	})

	t.Run("builds scenario with conditions", func(t *testing.T) {
		s := NewBuilder("conditional").
			Setup("Setup").
			SkipIf("env.SKIP_TEST is set", "Skip in certain environments").
			SkipUnless("env.CI == true", "Only run in CI").
			Build()

		if len(s.SkipIf) != 1 {
			t.Error("expected 1 skip_if condition")
		}
		if len(s.SkipUnless) != 1 {
			t.Error("expected 1 skip_unless condition")
		}
	})

	t.Run("builds scenario with matrix", func(t *testing.T) {
		s := NewBuilder("matrix-test").
			Setup("Setup").
			Matrix("currency", []any{"USD", "EUR", "GBP"}).
			Matrix("quantity", []any{1, 10, 100}).
			Build()

		if len(s.Matrix) != 2 {
			t.Errorf("expected 2 matrix parameters, got %d", len(s.Matrix))
		}
		if len(s.Matrix["currency"]) != 3 {
			t.Error("expected 3 currency values")
		}
	})
}

func TestFlowItem(t *testing.T) {
	t.Run("creates flow item with fluent API", func(t *testing.T) {
		item := NewFlowItem(core.ComponentTask, "MyTask").
			WithTimeout(10 * time.Second).
			WithDependsOn("Setup1", "Setup2").
			WithParam("key1", "value1").
			AsParallel()

		if item.Name != "MyTask" {
			t.Error("name not set")
		}
		if item.Timeout != 10*time.Second {
			t.Error("timeout not set")
		}
		if len(item.DependsOn) != 2 {
			t.Error("depends_on not set")
		}
		if item.Params["key1"] != "value1" {
			t.Error("param not set")
		}
		if !item.Parallel {
			t.Error("parallel not set")
		}
	})
}

func TestResolver(t *testing.T) {
	t.Run("resolves simple scenario", func(t *testing.T) {
		cfg := &config.Config{
			Scenarios: []config.ScenarioConfig{
				{
					Name:        "simple-test",
					Description: "A simple test",
					Timeout:     30 * time.Second,
					Tags:        []string{"smoke"},
					Flow: []config.FlowItemConfig{
						{Setup: "CreateUser"},
						{Task: "DoWork"},
						{Validation: "Verify"},
					},
				},
			},
		}

		resolver := NewResolver(cfg, nil)
		scenarios, err := resolver.ResolveAll()
		if err != nil {
			t.Fatalf("ResolveAll failed: %v", err)
		}

		if len(scenarios) != 1 {
			t.Errorf("expected 1 scenario, got %d", len(scenarios))
		}

		s := scenarios[0]
		if s.Name != "simple-test" {
			t.Errorf("expected name 'simple-test', got %s", s.Name)
		}
		if len(s.Flow) != 3 {
			t.Errorf("expected 3 flow items, got %d", len(s.Flow))
		}
	})

	t.Run("resolves inherited scenario", func(t *testing.T) {
		cfg := &config.Config{
			Scenarios: []config.ScenarioConfig{
				{
					Name:     "base",
					Abstract: true,
					Tags:     []string{"base-tag"},
					Flow: []config.FlowItemConfig{
						{Setup: "CommonSetup"},
					},
				},
				{
					Name:    "child",
					Extends: "base",
					Tags:    []string{"child-tag"},
					Flow: []config.FlowItemConfig{
						{Task: "ChildTask"},
					},
				},
			},
		}

		resolver := NewResolver(cfg, nil)
		scenarios, err := resolver.ResolveAll()
		if err != nil {
			t.Fatalf("ResolveAll failed: %v", err)
		}

		// Should only have 1 scenario (child), base is abstract
		if len(scenarios) != 1 {
			t.Errorf("expected 1 scenario, got %d", len(scenarios))
		}

		s := scenarios[0]
		if s.Name != "child" {
			t.Errorf("expected name 'child', got %s", s.Name)
		}

		// Should have inherited flow: CommonSetup, ChildTask
		if len(s.Flow) != 2 {
			t.Errorf("expected 2 flow items (inherited + own), got %d", len(s.Flow))
		}
		if s.Flow[0].Name != "CommonSetup" {
			t.Errorf("expected first flow item 'CommonSetup', got %s", s.Flow[0].Name)
		}
		if s.Flow[1].Name != "ChildTask" {
			t.Errorf("expected second flow item 'ChildTask', got %s", s.Flow[1].Name)
		}

		// Should have merged tags
		if len(s.Tags) != 2 {
			t.Errorf("expected 2 tags (merged), got %d", len(s.Tags))
		}
	})

	t.Run("expands matrix scenarios", func(t *testing.T) {
		cfg := &config.Config{
			Scenarios: []config.ScenarioConfig{
				{
					Name: "matrix-test",
					Matrix: map[string][]any{
						"currency": {"USD", "EUR"},
						"quantity": {1, 10},
					},
					Flow: []config.FlowItemConfig{
						{Task: "Checkout"},
					},
				},
			},
		}

		resolver := NewResolver(cfg, nil)
		scenarios, err := resolver.ResolveAll()
		if err != nil {
			t.Fatalf("ResolveAll failed: %v", err)
		}

		// Should have 4 scenarios (2 currencies × 2 quantities)
		if len(scenarios) != 4 {
			t.Errorf("expected 4 scenarios, got %d", len(scenarios))
		}

		// Each should have unique matrix index
		seen := make(map[string]bool)
		for _, s := range scenarios {
			key := s.Name
			if seen[key] {
				t.Errorf("duplicate scenario name: %s", key)
			}
			seen[key] = true

			if len(s.MatrixIndex) != 2 {
				t.Errorf("expected 2 matrix index values, got %d", len(s.MatrixIndex))
			}
		}
	})

	t.Run("substitutes matrix values in params", func(t *testing.T) {
		cfg := &config.Config{
			Scenarios: []config.ScenarioConfig{
				{
					Name: "param-test",
					Matrix: map[string][]any{
						"amount": {100, 200},
					},
					Flow: []config.FlowItemConfig{
						{
							Task: "Process",
							Params: map[string]any{
								"amount": "${{ matrix.amount }}",
							},
						},
					},
				},
			},
		}

		resolver := NewResolver(cfg, nil)
		scenarios, err := resolver.ResolveAll()
		if err != nil {
			t.Fatalf("ResolveAll failed: %v", err)
		}

		if len(scenarios) != 2 {
			t.Errorf("expected 2 scenarios, got %d", len(scenarios))
		}

		// Check that params were substituted
		for _, s := range scenarios {
			amount := s.MatrixIndex["amount"]
			paramAmount := s.Flow[0].Params["amount"]

			expectedParam := amount // After substitution, should be the actual value
			if paramAmount != expectedParam {
				// With string substitution, the result will be a string
				expectedStr := s.Flow[0].Params["amount"].(string)
				if expectedStr != "100" && expectedStr != "200" {
					t.Errorf("param should be substituted, got %v", paramAmount)
				}
			}
		}
	})
}
