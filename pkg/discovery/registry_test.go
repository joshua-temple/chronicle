package discovery

import (
	"testing"

	"github.com/joshua-temple/chronicle/pkg/core"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r.Components == nil {
		t.Error("Components should not be nil")
	}
	if r.Types == nil {
		t.Error("Types should not be nil")
	}
	if r.Middleware == nil {
		t.Error("Middleware should not be nil")
	}
}

func TestRegistryGetComponent(t *testing.T) {
	r := NewRegistry()
	c := core.NewComponent("CreateUser", core.ComponentSetup)
	r.Components[c.ID] = c

	t.Run("GetComponent finds existing", func(t *testing.T) {
		found, ok := r.GetComponent(c.ID)
		if !ok {
			t.Error("component should be found")
		}
		if found.Name != "CreateUser" {
			t.Error("wrong component returned")
		}
	})

	t.Run("GetComponent returns false for missing", func(t *testing.T) {
		_, ok := r.GetComponent(core.ComponentID("missing"))
		if ok {
			t.Error("missing component should not be found")
		}
	})

	t.Run("GetComponentByName finds existing", func(t *testing.T) {
		found, ok := r.GetComponentByName("CreateUser")
		if !ok {
			t.Error("component should be found by name")
		}
		if found.Name != "CreateUser" {
			t.Error("wrong component returned")
		}
	})
}

func TestRegistryComponentsByType(t *testing.T) {
	r := NewRegistry()
	r.Components[core.ComponentID("Setup1")] = core.NewComponent("Setup1", core.ComponentSetup)
	r.Components[core.ComponentID("Setup2")] = core.NewComponent("Setup2", core.ComponentSetup)
	r.Components[core.ComponentID("Task1")] = core.NewComponent("Task1", core.ComponentTask)

	setups := r.ComponentsByType(core.ComponentSetup)
	if len(setups) != 2 {
		t.Errorf("expected 2 setup components, got %d", len(setups))
	}

	tasks := r.ComponentsByType(core.ComponentTask)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task component, got %d", len(tasks))
	}
}

func TestRegistryComponentsByTag(t *testing.T) {
	r := NewRegistry()
	c1 := core.NewComponent("C1", core.ComponentSetup).WithTags("critical", "user")
	c2 := core.NewComponent("C2", core.ComponentTask).WithTags("critical")
	c3 := core.NewComponent("C3", core.ComponentValidation).WithTags("user")
	r.Components[c1.ID] = c1
	r.Components[c2.ID] = c2
	r.Components[c3.ID] = c3

	critical := r.ComponentsByTag("critical")
	if len(critical) != 2 {
		t.Errorf("expected 2 critical components, got %d", len(critical))
	}

	user := r.ComponentsByTag("user")
	if len(user) != 2 {
		t.Errorf("expected 2 user components, got %d", len(user))
	}
}

func TestRegistryValidate(t *testing.T) {
	t.Run("valid registry passes", func(t *testing.T) {
		r := NewRegistry()
		c1 := core.NewComponent("CreateUser", core.ComponentSetup).
			WithProduces("user", "User")
		c2 := core.NewComponent("CreateOrder", core.ComponentTask).
			WithRequires("user", "User").
			WithProduces("order", "Order")
		r.Components[c1.ID] = c1
		r.Components[c2.ID] = c2

		if err := r.Validate(); err != nil {
			t.Errorf("valid registry should pass: %v", err)
		}
	})

	t.Run("unsatisfied dependency fails", func(t *testing.T) {
		r := NewRegistry()
		c := core.NewComponent("CreateOrder", core.ComponentTask).
			WithRequires("user", "User") // No component produces user
		r.Components[c.ID] = c

		err := r.Validate()
		if err == nil {
			t.Error("should fail for unsatisfied dependency")
		}
	})

	t.Run("missing teardown fails", func(t *testing.T) {
		r := NewRegistry()
		c := core.NewComponent("CreateUser", core.ComponentSetup).
			WithTeardown("DeleteUser") // DeleteUser doesn't exist
		r.Components[c.ID] = c

		err := r.Validate()
		if err == nil {
			t.Error("should fail for missing teardown")
		}
	})

	t.Run("teardown with produces fails", func(t *testing.T) {
		r := NewRegistry()
		c := core.NewComponent("DeleteUser", core.ComponentTeardown).
			WithProduces("result", "bool")
		r.Components[c.ID] = c

		err := r.Validate()
		if err == nil {
			t.Error("teardown with produces should fail validation")
		}
	})
}

func TestRegistryDetectCycles(t *testing.T) {
	t.Run("no cycles in acyclic graph", func(t *testing.T) {
		r := NewRegistry()
		c1 := core.NewComponent("A", core.ComponentSetup).WithProduces("a", "A")
		c2 := core.NewComponent("B", core.ComponentTask).WithRequires("a", "A").WithProduces("b", "B")
		c3 := core.NewComponent("C", core.ComponentValidation).WithRequires("b", "B")
		r.Components[c1.ID] = c1
		r.Components[c2.ID] = c2
		r.Components[c3.ID] = c3

		cycles := r.DetectCycles()
		if len(cycles) != 0 {
			t.Errorf("expected no cycles, got %d", len(cycles))
		}
	})

	t.Run("detects simple cycle", func(t *testing.T) {
		r := NewRegistry()
		// A requires B, B requires A
		c1 := core.NewComponent("A", core.ComponentSetup).
			WithRequires("b", "B").
			WithProduces("a", "A")
		c2 := core.NewComponent("B", core.ComponentTask).
			WithRequires("a", "A").
			WithProduces("b", "B")
		r.Components[c1.ID] = c1
		r.Components[c2.ID] = c2

		cycles := r.DetectCycles()
		if len(cycles) == 0 {
			t.Error("expected to detect cycle")
		}
	})
}

func TestRegistryDependencyGraph(t *testing.T) {
	r := NewRegistry()
	c1 := core.NewComponent("CreateUser", core.ComponentSetup).
		WithProduces("user", "User")
	c2 := core.NewComponent("CreateOrder", core.ComponentTask).
		WithRequires("user", "User").
		WithProduces("order", "Order")
	c3 := core.NewComponent("ValidateOrder", core.ComponentValidation).
		WithRequires("order", "Order")
	r.Components[c1.ID] = c1
	r.Components[c2.ID] = c2
	r.Components[c3.ID] = c3

	g := r.DependencyGraph()

	t.Run("contains all nodes", func(t *testing.T) {
		if len(g.Nodes) != 3 {
			t.Errorf("expected 3 nodes, got %d", len(g.Nodes))
		}
	})

	t.Run("contains correct edges", func(t *testing.T) {
		// Should have edges: CreateUser -> CreateOrder, CreateOrder -> ValidateOrder
		if len(g.Edges) != 2 {
			t.Errorf("expected 2 edges, got %d", len(g.Edges))
		}
	})
}

func TestGraphTopologicalSort(t *testing.T) {
	t.Run("sorts acyclic graph", func(t *testing.T) {
		r := NewRegistry()
		c1 := core.NewComponent("A", core.ComponentSetup).WithProduces("a", "A")
		c2 := core.NewComponent("B", core.ComponentTask).WithRequires("a", "A").WithProduces("b", "B")
		c3 := core.NewComponent("C", core.ComponentValidation).WithRequires("b", "B")
		r.Components[c1.ID] = c1
		r.Components[c2.ID] = c2
		r.Components[c3.ID] = c3

		g := r.DependencyGraph()
		order, err := g.TopologicalSort()
		if err != nil {
			t.Fatalf("topological sort failed: %v", err)
		}

		// A must come before B, B must come before C
		aIdx, bIdx, cIdx := -1, -1, -1
		for i, name := range order {
			switch name {
			case "A":
				aIdx = i
			case "B":
				bIdx = i
			case "C":
				cIdx = i
			}
		}

		if aIdx > bIdx || bIdx > cIdx {
			t.Errorf("invalid order: A=%d, B=%d, C=%d", aIdx, bIdx, cIdx)
		}
	})

	t.Run("fails on cycle", func(t *testing.T) {
		r := NewRegistry()
		c1 := core.NewComponent("A", core.ComponentSetup).
			WithRequires("b", "B").WithProduces("a", "A")
		c2 := core.NewComponent("B", core.ComponentTask).
			WithRequires("a", "A").WithProduces("b", "B")
		r.Components[c1.ID] = c1
		r.Components[c2.ID] = c2

		g := r.DependencyGraph()
		_, err := g.TopologicalSort()
		if err == nil {
			t.Error("should fail on cycle")
		}
	})
}

func TestRegistryStats(t *testing.T) {
	r := NewRegistry()
	r.Components[core.ComponentID("S1")] = core.NewComponent("S1", core.ComponentSetup)
	r.Components[core.ComponentID("S2")] = core.NewComponent("S2", core.ComponentSetup)
	r.Components[core.ComponentID("T1")] = core.NewComponent("T1", core.ComponentTask)
	r.Types["User"] = &core.TypeInfo{Name: "User"}
	r.Middleware["Logger"] = &MiddlewareInfo{Name: "Logger"}

	stats := r.Stats()

	if stats["components"] != 3 {
		t.Errorf("expected 3 components, got %d", stats["components"])
	}
	if stats["types"] != 1 {
		t.Errorf("expected 1 type, got %d", stats["types"])
	}
	if stats["middleware"] != 1 {
		t.Errorf("expected 1 middleware, got %d", stats["middleware"])
	}
	if stats["setup_count"] != 2 {
		t.Errorf("expected 2 setup components, got %d", stats["setup_count"])
	}
	if stats["task_count"] != 1 {
		t.Errorf("expected 1 task component, got %d", stats["task_count"])
	}
}

func TestRegistryMerge(t *testing.T) {
	t.Run("merges disjoint registries", func(t *testing.T) {
		r1 := NewRegistry()
		r1.Components[core.ComponentID("A")] = core.NewComponent("A", core.ComponentSetup)

		r2 := NewRegistry()
		r2.Components[core.ComponentID("B")] = core.NewComponent("B", core.ComponentTask)

		if err := r1.Merge(r2); err != nil {
			t.Errorf("merge failed: %v", err)
		}

		if len(r1.Components) != 2 {
			t.Errorf("expected 2 components after merge, got %d", len(r1.Components))
		}
	})

	t.Run("fails on duplicate component", func(t *testing.T) {
		r1 := NewRegistry()
		r1.Components[core.ComponentID("A")] = core.NewComponent("A", core.ComponentSetup)

		r2 := NewRegistry()
		r2.Components[core.ComponentID("A")] = core.NewComponent("A", core.ComponentTask)

		if err := r1.Merge(r2); err == nil {
			t.Error("should fail on duplicate component")
		}
	})

	t.Run("fails on duplicate type", func(t *testing.T) {
		r1 := NewRegistry()
		r1.Types["User"] = &core.TypeInfo{Name: "User", SourceFile: "a.go"}

		r2 := NewRegistry()
		r2.Types["User"] = &core.TypeInfo{Name: "User", SourceFile: "b.go"}

		if err := r1.Merge(r2); err == nil {
			t.Error("should fail on duplicate type")
		}
	})
}

func TestValidationErrors(t *testing.T) {
	t.Run("Error formats correctly", func(t *testing.T) {
		err := ValidationError{
			Component: "CreateUser",
			Field:     "requires",
			Message:   "dependency not found",
		}
		expected := "CreateUser.requires: dependency not found"
		if err.Error() != expected {
			t.Errorf("expected %s, got %s", expected, err.Error())
		}
	})

	t.Run("ValidationErrors formats multiple errors", func(t *testing.T) {
		errs := ValidationErrors{
			{Component: "A", Field: "f1", Message: "m1"},
			{Component: "B", Field: "f2", Message: "m2"},
		}
		result := errs.Error()
		if result != "A.f1: m1; B.f2: m2" {
			t.Errorf("unexpected format: %s", result)
		}
	})

	t.Run("Empty ValidationErrors", func(t *testing.T) {
		errs := ValidationErrors{}
		if errs.Error() != "no errors" {
			t.Errorf("expected 'no errors', got %s", errs.Error())
		}
	})
}

func TestCycle(t *testing.T) {
	c := Cycle{Path: []string{"A", "B", "C"}}
	expected := "A -> B -> C -> A"
	if c.String() != expected {
		t.Errorf("expected %s, got %s", expected, c.String())
	}
}
