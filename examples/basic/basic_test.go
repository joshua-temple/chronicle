package basic_test

import (
	gocontext "context"
	"testing"

	"github.com/joshua-temple/chronicle/examples/basic"
	"github.com/joshua-temple/chronicle/pkg/context"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
)

func TestDiscovery(t *testing.T) {
	parser := discovery.NewParser("./")
	registry, err := parser.Discover()
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	// Check types were discovered
	t.Run("discovers types", func(t *testing.T) {
		if _, ok := registry.Types["User"]; !ok {
			t.Error("User type not found")
		}
		if _, ok := registry.Types["Order"]; !ok {
			t.Error("Order type not found")
		}
	})

	// Check components were discovered
	t.Run("discovers setup component", func(t *testing.T) {
		c, ok := registry.GetComponentByName("CreateUser")
		if !ok {
			t.Fatal("CreateUser component not found")
		}
		if c.Type != core.ComponentSetup {
			t.Errorf("expected setup type, got %s", c.Type)
		}
		if !c.ProducesKey("user") {
			t.Error("should produce 'user'")
		}
		if c.Teardown != "DeleteUser" {
			t.Error("should have teardown 'DeleteUser'")
		}
		if c.Description != "Creates a test user for the scenario" {
			t.Errorf("wrong description: %s", c.Description)
		}
	})

	t.Run("discovers teardown component", func(t *testing.T) {
		c, ok := registry.GetComponentByName("DeleteUser")
		if !ok {
			t.Fatal("DeleteUser component not found")
		}
		if c.Type != core.ComponentTeardown {
			t.Errorf("expected teardown type, got %s", c.Type)
		}
		if !c.RequiresKey("user") {
			t.Error("should require 'user'")
		}
	})

	t.Run("discovers task component", func(t *testing.T) {
		c, ok := registry.GetComponentByName("CreateOrder")
		if !ok {
			t.Fatal("CreateOrder component not found")
		}
		if c.Type != core.ComponentTask {
			t.Errorf("expected task type, got %s", c.Type)
		}
		if !c.RequiresKey("user") {
			t.Error("should require 'user'")
		}
		if !c.ProducesKey("order") {
			t.Error("should produce 'order'")
		}
	})

	t.Run("discovers validation component", func(t *testing.T) {
		c, ok := registry.GetComponentByName("OrderValid")
		if !ok {
			t.Fatal("OrderValid component not found")
		}
		if c.Type != core.ComponentValidation {
			t.Errorf("expected validation type, got %s", c.Type)
		}
		if !c.RequiresKey("order") {
			t.Error("should require 'order'")
		}
	})
}

func TestComponentExecution(t *testing.T) {
	ctx := context.New(gocontext.Background())

	t.Run("CreateUser sets user in context", func(t *testing.T) {
		err := basic.CreateUser(ctx)
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		user := context.Get[*basic.User](ctx, "user")
		if user == nil {
			t.Fatal("user should be in context")
		}
		if user.ID != "usr_123" {
			t.Errorf("expected user ID 'usr_123', got %s", user.ID)
		}
		if user.Email != "test@example.com" {
			t.Errorf("expected email 'test@example.com', got %s", user.Email)
		}
	})

	t.Run("CreateOrder creates order with user", func(t *testing.T) {
		order, err := basic.CreateOrder(ctx)
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}

		if order.ID != "ord_456" {
			t.Errorf("expected order ID 'ord_456', got %s", order.ID)
		}
		if order.UserID != "usr_123" {
			t.Errorf("expected UserID 'usr_123', got %s", order.UserID)
		}
		if order.Total != 99.99 {
			t.Errorf("expected total 99.99, got %f", order.Total)
		}
	})

	t.Run("OrderValid validates correct order", func(t *testing.T) {
		order := context.Get[*basic.Order](ctx, "order")
		err := basic.OrderValid(ctx, order)
		if err != nil {
			t.Errorf("OrderValid failed: %v", err)
		}
	})

	t.Run("OrderValid rejects invalid order", func(t *testing.T) {
		invalidOrder := &basic.Order{ID: "", Total: 0}
		err := basic.OrderValid(ctx, invalidOrder)
		if err == nil {
			t.Error("OrderValid should reject order with empty ID")
		}
	})

	t.Run("DeleteUser cleans up", func(t *testing.T) {
		err := basic.DeleteUser(ctx)
		if err != nil {
			t.Errorf("DeleteUser failed: %v", err)
		}
	})
}

func TestDependencyGraph(t *testing.T) {
	parser := discovery.NewParser("./")
	registry, err := parser.Discover()
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	graph := registry.DependencyGraph()

	t.Run("graph has correct nodes", func(t *testing.T) {
		if len(graph.Nodes) != 4 {
			t.Errorf("expected 4 nodes, got %d", len(graph.Nodes))
		}
	})

	t.Run("graph has correct edges", func(t *testing.T) {
		// CreateUser -> CreateOrder (via user)
		// CreateOrder -> OrderValid (via order)
		// CreateUser -> DeleteUser (via user)
		if len(graph.Edges) < 2 {
			t.Errorf("expected at least 2 edges, got %d", len(graph.Edges))
		}
	})

	t.Run("topological sort works", func(t *testing.T) {
		order, err := graph.TopologicalSort()
		if err != nil {
			t.Fatalf("topological sort failed: %v", err)
		}

		// CreateUser should come before CreateOrder
		// CreateOrder should come before OrderValid
		createUserIdx := -1
		createOrderIdx := -1
		orderValidIdx := -1

		for i, name := range order {
			switch name {
			case "CreateUser":
				createUserIdx = i
			case "CreateOrder":
				createOrderIdx = i
			case "OrderValid":
				orderValidIdx = i
			}
		}

		if createUserIdx > createOrderIdx {
			t.Error("CreateUser should come before CreateOrder")
		}
		if createOrderIdx > orderValidIdx {
			t.Error("CreateOrder should come before OrderValid")
		}
	})
}

func TestRegistryValidation(t *testing.T) {
	parser := discovery.NewParser("./")
	registry, err := parser.Discover()
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	t.Run("registry validates successfully", func(t *testing.T) {
		err := registry.Validate()
		if err != nil {
			t.Errorf("registry validation failed: %v", err)
		}
	})

	t.Run("no cycles detected", func(t *testing.T) {
		cycles := registry.DetectCycles()
		if len(cycles) != 0 {
			t.Errorf("unexpected cycles: %v", cycles)
		}
	})
}
