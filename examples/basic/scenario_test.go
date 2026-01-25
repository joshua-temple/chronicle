package basic_test

import (
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

func TestScenarioConfiguration(t *testing.T) {
	cfg, err := config.Load("./chronicle.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	t.Run("loads project metadata", func(t *testing.T) {
		if cfg.Name != "basic-example" {
			t.Errorf("expected name 'basic-example', got %s", cfg.Name)
		}
		if cfg.Version != "1.0" {
			t.Errorf("expected version '1.0', got %s", cfg.Version)
		}
	})

	t.Run("loads discovery paths", func(t *testing.T) {
		if len(cfg.Discovery.Paths) != 1 {
			t.Errorf("expected 1 discovery path, got %d", len(cfg.Discovery.Paths))
		}
		if cfg.Discovery.Paths[0] != "." {
			t.Errorf("expected path '.', got %s", cfg.Discovery.Paths[0])
		}
	})

	t.Run("loads scenarios", func(t *testing.T) {
		// Should have 5 scenarios in config
		if len(cfg.Scenarios) != 5 {
			t.Errorf("expected 5 scenarios, got %d", len(cfg.Scenarios))
		}
	})

	t.Run("loads abstract scenario", func(t *testing.T) {
		s, ok := cfg.GetScenario("user-operations-base")
		if !ok {
			t.Fatal("user-operations-base not found")
		}
		if !s.Abstract {
			t.Error("user-operations-base should be abstract")
		}
	})

	t.Run("loads scenario with inheritance", func(t *testing.T) {
		s, ok := cfg.GetScenario("user-creates-order")
		if !ok {
			t.Fatal("user-creates-order not found")
		}
		if s.Extends != "user-operations-base" {
			t.Errorf("expected extends 'user-operations-base', got %s", s.Extends)
		}
	})

	t.Run("loads scenario with conditions", func(t *testing.T) {
		s, ok := cfg.GetScenario("ci-only-checkout")
		if !ok {
			t.Fatal("ci-only-checkout not found")
		}
		if len(s.SkipUnless) != 1 {
			t.Errorf("expected 1 skip_unless condition, got %d", len(s.SkipUnless))
		}
	})

	t.Run("loads scenario with matrix", func(t *testing.T) {
		s, ok := cfg.GetScenario("multi-quantity-order")
		if !ok {
			t.Fatal("multi-quantity-order not found")
		}
		if len(s.Matrix) != 1 {
			t.Errorf("expected 1 matrix parameter, got %d", len(s.Matrix))
		}
		if len(s.Matrix["quantity"]) != 3 {
			t.Errorf("expected 3 quantity values, got %d", len(s.Matrix["quantity"]))
		}
	})

	t.Run("loads execution config", func(t *testing.T) {
		if cfg.Execution.Parallelism != 2 {
			t.Errorf("expected parallelism 2, got %d", cfg.Execution.Parallelism)
		}
		if cfg.Execution.DefaultTimeout != 30*time.Second {
			t.Errorf("expected default timeout 30s, got %v", cfg.Execution.DefaultTimeout)
		}
		if cfg.Execution.TeardownMode != "always" {
			t.Errorf("expected teardown mode 'always', got %s", cfg.Execution.TeardownMode)
		}
	})

	t.Run("validates successfully", func(t *testing.T) {
		err := cfg.Validate()
		if err != nil {
			t.Errorf("validation failed: %v", err)
		}
	})
}

func TestScenarioResolution(t *testing.T) {
	cfg, err := config.Load("./chronicle.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	parser := discovery.NewParser("./")
	registry, err := parser.Discover()
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	resolver := scenario.NewResolver(cfg, registry)

	t.Run("resolves all non-abstract scenarios", func(t *testing.T) {
		scenarios, err := resolver.ResolveAll()
		if err != nil {
			t.Fatalf("ResolveAll failed: %v", err)
		}

		// Should have: basic-checkout, user-creates-order, ci-only-checkout,
		// plus 3 matrix-expanded scenarios from multi-quantity-order
		// = 3 + 3 = 6 scenarios (excluding abstract base)
		expectedCount := 6
		if len(scenarios) != expectedCount {
			t.Errorf("expected %d scenarios, got %d", expectedCount, len(scenarios))
			for _, s := range scenarios {
				t.Logf("  - %s", s.Name)
			}
		}
	})

	t.Run("resolves inherited scenario", func(t *testing.T) {
		s, err := resolver.Resolve("user-creates-order")
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		// Should have inherited flow from parent + own flow
		// Parent: CreateUser
		// Own: CreateOrder, OrderValid
		if len(s.Flow) != 3 {
			t.Errorf("expected 3 flow items (inherited + own), got %d", len(s.Flow))
		}

		// First item should be from parent
		if s.Flow[0].Name != "CreateUser" {
			t.Errorf("expected first flow item 'CreateUser', got %s", s.Flow[0].Name)
		}
		if s.Flow[0].Type != core.ComponentSetup {
			t.Errorf("expected first flow item to be setup, got %s", s.Flow[0].Type)
		}
	})

	t.Run("resolves matrix scenarios", func(t *testing.T) {
		scenarios, err := resolver.ResolveAll()
		if err != nil {
			t.Fatalf("ResolveAll failed: %v", err)
		}

		// Find matrix-expanded scenarios
		var matrixScenarios []*scenario.Scenario
		for _, s := range scenarios {
			if s.Name == "multi-quantity-order[1]" ||
				s.Name == "multi-quantity-order[5]" ||
				s.Name == "multi-quantity-order[10]" {
				matrixScenarios = append(matrixScenarios, s)
			}
		}

		if len(matrixScenarios) != 3 {
			t.Errorf("expected 3 matrix scenarios, got %d", len(matrixScenarios))
		}

		// Each should have different matrix index
		quantities := make(map[any]bool)
		for _, s := range matrixScenarios {
			q := s.MatrixIndex["quantity"]
			if quantities[q] {
				t.Errorf("duplicate quantity in matrix: %v", q)
			}
			quantities[q] = true
		}
	})

	t.Run("resolves component references", func(t *testing.T) {
		s, err := resolver.Resolve("basic-checkout")
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		// Check that components are resolved
		for _, item := range s.Flow {
			if item.Component == nil {
				t.Errorf("flow item %s has nil component reference", item.Name)
			}
		}
	})
}

func TestScenarioBuilder(t *testing.T) {
	t.Run("builds scenario programmatically", func(t *testing.T) {
		s := scenario.NewBuilder("programmatic-checkout").
			Description("Checkout flow built programmatically").
			Timeout(60 * time.Second).
			Tags("smoke", "programmatic").
			Setup("CreateUser").
			Task("CreateOrder").
			Validation("OrderValid").
			Flag("debug", true).
			Build()

		if s.Name != "programmatic-checkout" {
			t.Error("name not set")
		}
		if s.Timeout != 60*time.Second {
			t.Error("timeout not set")
		}
		if len(s.Flow) != 3 {
			t.Errorf("expected 3 flow items, got %d", len(s.Flow))
		}
		if !s.IsRunnable() {
			t.Error("scenario should be runnable")
		}
	})

	t.Run("builds abstract scenario", func(t *testing.T) {
		s := scenario.NewBuilder("abstract-base").
			Abstract().
			Setup("CommonSetup").
			Build()

		if !s.Abstract {
			t.Error("scenario should be abstract")
		}
		if s.IsRunnable() {
			t.Error("abstract scenario should not be runnable")
		}
	})
}

func TestConditionalExecution(t *testing.T) {
	t.Run("evaluates skip_unless with environment", func(t *testing.T) {
		s := scenario.NewBuilder("env-test").
			Setup("Setup").
			SkipUnless("env.CI is set", "Only runs in CI").
			Build()

		// Without CI env, should skip
		skip, reason := scenario.EvaluateSkipConditions(s, nil)
		if !skip {
			t.Error("should skip when CI is not set")
		}
		if reason != "Only runs in CI" {
			t.Errorf("wrong reason: %s", reason)
		}
	})

	t.Run("evaluates skip_if with flags", func(t *testing.T) {
		s := scenario.NewBuilder("flag-test").
			Setup("Setup").
			SkipIf("flags.skip == true", "Skip flag is set").
			Build()

		// Without flag, should not skip
		skip, _ := scenario.EvaluateSkipConditions(s, nil)
		if skip {
			t.Error("should not skip when flag is not set")
		}

		// With flag, should skip
		skip, reason := scenario.EvaluateSkipConditions(s, map[string]any{"skip": true})
		if !skip {
			t.Error("should skip when flag is set")
		}
		if reason != "Skip flag is set" {
			t.Errorf("wrong reason: %s", reason)
		}
	})
}
