package scenario

import (
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/core"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("test-scenario")

	if builder == nil {
		t.Fatal("NewBuilder should not return nil")
	}

	if builder.scenario == nil {
		t.Fatal("Builder.scenario should not be nil")
	}

	if builder.scenario.Name != "test-scenario" {
		t.Errorf("Builder.scenario.Name = %q, expected 'test-scenario'", builder.scenario.Name)
	}
}

func TestBuilder_Description(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Description("Test description")

	if result != builder {
		t.Error("Description() should return the builder for chaining")
	}

	if builder.scenario.Description != "Test description" {
		t.Errorf("Description = %q, expected 'Test description'", builder.scenario.Description)
	}
}

func TestBuilder_Timeout(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Timeout(5 * time.Minute)

	if result != builder {
		t.Error("Timeout() should return the builder for chaining")
	}

	if builder.scenario.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, expected 5m", builder.scenario.Timeout)
	}
}

func TestBuilder_Tags(t *testing.T) {
	builder := NewBuilder("test")

	result := builder.Tags("tag1", "tag2")
	if result != builder {
		t.Error("Tags() should return the builder for chaining")
	}

	builder.Tags("tag3")

	if len(builder.scenario.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(builder.scenario.Tags))
	}

	expectedTags := []string{"tag1", "tag2", "tag3"}
	for i, expected := range expectedTags {
		if builder.scenario.Tags[i] != expected {
			t.Errorf("Tags[%d] = %q, expected %q", i, builder.scenario.Tags[i], expected)
		}
	}
}

func TestBuilder_Extends(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Extends("base-scenario")

	if result != builder {
		t.Error("Extends() should return the builder for chaining")
	}

	if builder.scenario.Extends != "base-scenario" {
		t.Errorf("Extends = %q, expected 'base-scenario'", builder.scenario.Extends)
	}
}

func TestBuilder_Abstract(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Abstract()

	if result != builder {
		t.Error("Abstract() should return the builder for chaining")
	}

	if !builder.scenario.Abstract {
		t.Error("Abstract = false, expected true")
	}
}

func TestBuilder_Setup(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Setup("SetupDB")

	if result != builder {
		t.Error("Setup() should return the builder for chaining")
	}

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}

	item := builder.scenario.Flow[0]
	if item.Type != core.ComponentSetup {
		t.Errorf("Flow item type = %q, expected 'setup'", item.Type)
	}
	if item.Name != "SetupDB" {
		t.Errorf("Flow item name = %q, expected 'SetupDB'", item.Name)
	}
}

func TestBuilder_SetupWithTimeout(t *testing.T) {
	builder := NewBuilder("test")
	builder.SetupWithTimeout("SetupDB", 30*time.Second)

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}

	item := builder.scenario.Flow[0]
	if item.Timeout != 30*time.Second {
		t.Errorf("Flow item timeout = %v, expected 30s", item.Timeout)
	}
}

func TestBuilder_Task(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Task("RunTest")

	if result != builder {
		t.Error("Task() should return the builder for chaining")
	}

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}

	item := builder.scenario.Flow[0]
	if item.Type != core.ComponentTask {
		t.Errorf("Flow item type = %q, expected 'task'", item.Type)
	}
	if item.Name != "RunTest" {
		t.Errorf("Flow item name = %q, expected 'RunTest'", item.Name)
	}
}

func TestBuilder_TaskWithParams(t *testing.T) {
	builder := NewBuilder("test")
	params := map[string]any{"key": "value", "count": 10}
	builder.TaskWithParams("RunTest", params)

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}

	item := builder.scenario.Flow[0]
	if item.Params["key"] != "value" {
		t.Errorf("Params['key'] = %v, expected 'value'", item.Params["key"])
	}
	if item.Params["count"] != 10 {
		t.Errorf("Params['count'] = %v, expected 10", item.Params["count"])
	}
}

func TestBuilder_Validation(t *testing.T) {
	builder := NewBuilder("test")
	builder.Validation("ValidateResult")

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}

	item := builder.scenario.Flow[0]
	if item.Type != core.ComponentValidation {
		t.Errorf("Flow item type = %q, expected 'validation'", item.Type)
	}
}

func TestBuilder_Step(t *testing.T) {
	builder := NewBuilder("test")
	builder.Step("StepOne")

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}

	item := builder.scenario.Flow[0]
	if item.Type != core.ComponentStep {
		t.Errorf("Flow item type = %q, expected 'step'", item.Type)
	}
}

func TestBuilder_Rollup(t *testing.T) {
	builder := NewBuilder("test")
	builder.Rollup("AggregateResults")

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}

	item := builder.scenario.Flow[0]
	if item.Type != core.ComponentRollup {
		t.Errorf("Flow item type = %q, expected 'rollup'", item.Type)
	}
}

func TestBuilder_Teardown(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Teardown("CleanupDB")

	if result != builder {
		t.Error("Teardown() should return the builder for chaining")
	}

	if len(builder.scenario.TeardownFlow) != 1 {
		t.Fatalf("Expected 1 teardown item, got %d", len(builder.scenario.TeardownFlow))
	}

	item := builder.scenario.TeardownFlow[0]
	if item.Type != core.ComponentTeardown {
		t.Errorf("Teardown item type = %q, expected 'teardown'", item.Type)
	}
	if item.Name != "CleanupDB" {
		t.Errorf("Teardown item name = %q, expected 'CleanupDB'", item.Name)
	}
}

func TestBuilder_Flow(t *testing.T) {
	builder := NewBuilder("test")
	item := NewFlowItem(core.ComponentTask, "CustomTask")
	result := builder.Flow(item)

	if result != builder {
		t.Error("Flow() should return the builder for chaining")
	}

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 flow item, got %d", len(builder.scenario.Flow))
	}
}

func TestBuilder_Parallel(t *testing.T) {
	builder := NewBuilder("test")
	item1 := NewFlowItem(core.ComponentTask, "Task1")
	item2 := NewFlowItem(core.ComponentTask, "Task2")

	result := builder.Parallel(item1, item2)

	if result != builder {
		t.Error("Parallel() should return the builder for chaining")
	}

	if len(builder.scenario.Flow) != 1 {
		t.Fatalf("Expected 1 parallel block, got %d", len(builder.scenario.Flow))
	}

	block := builder.scenario.Flow[0]
	if !block.Parallel {
		t.Error("Parallel block should have Parallel = true")
	}
	if len(block.ParallelItems) != 2 {
		t.Errorf("Expected 2 parallel items, got %d", len(block.ParallelItems))
	}
}

func TestBuilder_Flag(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Flag("debug", true)

	if result != builder {
		t.Error("Flag() should return the builder for chaining")
	}

	if builder.scenario.Flags["debug"] != true {
		t.Errorf("Flags['debug'] = %v, expected true", builder.scenario.Flags["debug"])
	}
}

func TestBuilder_Flags(t *testing.T) {
	builder := NewBuilder("test")
	flags := map[string]any{
		"debug":   true,
		"env":     "test",
		"timeout": 30,
	}
	result := builder.Flags(flags)

	if result != builder {
		t.Error("Flags() should return the builder for chaining")
	}

	if len(builder.scenario.Flags) != 3 {
		t.Errorf("Expected 3 flags, got %d", len(builder.scenario.Flags))
	}

	if builder.scenario.Flags["env"] != "test" {
		t.Errorf("Flags['env'] = %v, expected 'test'", builder.scenario.Flags["env"])
	}
}

func TestBuilder_Options(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Options("opt1", "opt2")

	if result != builder {
		t.Error("Options() should return the builder for chaining")
	}

	if len(builder.scenario.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(builder.scenario.Options))
	}
}

func TestBuilder_ChaosProfiles(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.ChaosProfiles("network-latency", "cpu-stress")

	if result != builder {
		t.Error("ChaosProfiles() should return the builder for chaining")
	}

	if len(builder.scenario.ChaosProfiles) != 2 {
		t.Errorf("Expected 2 chaos profiles, got %d", len(builder.scenario.ChaosProfiles))
	}
}

func TestBuilder_MockProfiles(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.MockProfiles("http-mock", "db-mock")

	if result != builder {
		t.Error("MockProfiles() should return the builder for chaining")
	}

	if len(builder.scenario.MockProfiles) != 2 {
		t.Errorf("Expected 2 mock profiles, got %d", len(builder.scenario.MockProfiles))
	}
}

func TestBuilder_SkipIf(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.SkipIf("env.CI == true", "Skip in CI")

	if result != builder {
		t.Error("SkipIf() should return the builder for chaining")
	}

	if len(builder.scenario.SkipIf) != 1 {
		t.Fatalf("Expected 1 skip condition, got %d", len(builder.scenario.SkipIf))
	}

	cond := builder.scenario.SkipIf[0]
	if cond.Expression != "env.CI == true" {
		t.Errorf("SkipIf expression = %q, expected 'env.CI == true'", cond.Expression)
	}
	if cond.Reason != "Skip in CI" {
		t.Errorf("SkipIf reason = %q, expected 'Skip in CI'", cond.Reason)
	}
}

func TestBuilder_SkipUnless(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.SkipUnless("flags.integration", "Requires integration flag")

	if result != builder {
		t.Error("SkipUnless() should return the builder for chaining")
	}

	if len(builder.scenario.SkipUnless) != 1 {
		t.Fatalf("Expected 1 skip-unless condition, got %d", len(builder.scenario.SkipUnless))
	}
}

func TestBuilder_Matrix(t *testing.T) {
	builder := NewBuilder("test")
	result := builder.Matrix("version", []any{"1.0", "2.0", "3.0"})

	if result != builder {
		t.Error("Matrix() should return the builder for chaining")
	}

	if len(builder.scenario.Matrix["version"]) != 3 {
		t.Errorf("Expected 3 matrix values, got %d", len(builder.scenario.Matrix["version"]))
	}
}

func TestBuilder_Build(t *testing.T) {
	builder := NewBuilder("test")
	builder.Description("Test scenario").
		Timeout(5 * time.Minute).
		Tags("unit", "fast").
		Setup("SetupDB").
		Task("RunTest").
		Validation("Validate").
		Teardown("Cleanup")

	scenario := builder.Build()

	if scenario == nil {
		t.Fatal("Build() should not return nil")
	}

	if scenario.Name != "test" {
		t.Errorf("Name = %q, expected 'test'", scenario.Name)
	}

	if scenario.Description != "Test scenario" {
		t.Errorf("Description = %q, expected 'Test scenario'", scenario.Description)
	}

	if len(scenario.Flow) != 3 {
		t.Errorf("Expected 3 flow items, got %d", len(scenario.Flow))
	}

	if len(scenario.TeardownFlow) != 1 {
		t.Errorf("Expected 1 teardown item, got %d", len(scenario.TeardownFlow))
	}
}

func TestBuilder_MustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() should not panic with valid scenario: %v", r)
		}
	}()

	builder := NewBuilder("test")
	builder.Setup("Setup")

	scenario := builder.MustBuild()
	if scenario == nil {
		t.Fatal("MustBuild() should return a scenario")
	}
}

func TestBuilder_MustBuild_EmptyFlow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should panic with empty flow")
		}
	}()

	builder := NewBuilder("test")
	builder.MustBuild()
}

func TestBuilder_MustBuild_NoName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should panic with empty name")
		}
	}()

	builder := &Builder{
		scenario: &Scenario{
			Name:  "",
			Flow:  []FlowItem{{Name: "test"}},
			Flags: make(map[string]any),
		},
	}
	builder.MustBuild()
}

func TestBuilder_MustBuild_AbstractAllowed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() should not panic with abstract scenario: %v", r)
		}
	}()

	builder := NewBuilder("base")
	builder.Abstract()

	scenario := builder.MustBuild()
	if !scenario.Abstract {
		t.Error("Scenario should be abstract")
	}
}

func TestBuilder_Chaining(t *testing.T) {
	// Test that all methods can be chained
	scenario := NewBuilder("full-test").
		Description("A comprehensive test").
		Timeout(10 * time.Minute).
		Tags("integration", "slow").
		Setup("SetupDB").
		SetupWithTimeout("SetupCache", 30*time.Second).
		Task("TestFeature").
		TaskWithParams("TestWithParams", map[string]any{"key": "value"}).
		Validation("ValidateResults").
		Step("ProcessData").
		Rollup("AggregateMetrics").
		Teardown("CleanupDB").
		Teardown("CleanupCache").
		Flag("debug", true).
		Flags(map[string]any{"env": "test"}).
		Options("retry", "verbose").
		ChaosProfiles("network").
		MockProfiles("http").
		SkipIf("ci", "CI environment").
		SkipUnless("integration", "Needs integration").
		Matrix("version", []any{"1", "2"}).
		Build()

	if scenario == nil {
		t.Fatal("Chained building should return a scenario")
	}

	if len(scenario.Flow) != 7 {
		t.Errorf("Expected 7 flow items, got %d", len(scenario.Flow))
	}

	if len(scenario.TeardownFlow) != 2 {
		t.Errorf("Expected 2 teardown items, got %d", len(scenario.TeardownFlow))
	}
}
