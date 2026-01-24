package core

import (
	"reflect"
	"testing"
	"time"
)

func TestComponentType(t *testing.T) {
	t.Run("valid types", func(t *testing.T) {
		validTypes := []ComponentType{
			ComponentSetup,
			ComponentTask,
			ComponentValidation,
			ComponentStep,
			ComponentRollup,
			ComponentTeardown,
		}
		for _, ct := range validTypes {
			if !ct.IsValid() {
				t.Errorf("%s should be valid", ct)
			}
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		invalid := ComponentType("unknown")
		if invalid.IsValid() {
			t.Error("unknown type should be invalid")
		}
	})

	t.Run("String returns correct value", func(t *testing.T) {
		if ComponentSetup.String() != "setup" {
			t.Errorf("expected 'setup', got %s", ComponentSetup.String())
		}
		if ComponentTask.String() != "task" {
			t.Errorf("expected 'task', got %s", ComponentTask.String())
		}
	})
}

func TestDependency(t *testing.T) {
	t.Run("String without description", func(t *testing.T) {
		d := Dependency{Key: "user", Type: "User"}
		expected := "user:User"
		if d.String() != expected {
			t.Errorf("expected %s, got %s", expected, d.String())
		}
	})

	t.Run("String with description", func(t *testing.T) {
		d := Dependency{Key: "user", Type: "User", Description: "The authenticated user"}
		expected := "user:User (The authenticated user)"
		if d.String() != expected {
			t.Errorf("expected %s, got %s", expected, d.String())
		}
	})
}

func TestNewComponent(t *testing.T) {
	t.Run("creates component with correct defaults", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup)

		if c.Name != "CreateUser" {
			t.Errorf("expected name 'CreateUser', got %s", c.Name)
		}
		if c.Type != ComponentSetup {
			t.Errorf("expected type 'setup', got %s", c.Type)
		}
		if c.ID != ComponentID("CreateUser") {
			t.Errorf("expected ID 'CreateUser', got %s", c.ID)
		}
		if c.Produces == nil {
			t.Error("Produces should not be nil")
		}
		if c.Requires == nil {
			t.Error("Requires should not be nil")
		}
		if c.Tags == nil {
			t.Error("Tags should not be nil")
		}
	})
}

func TestComponentBuilder(t *testing.T) {
	t.Run("fluent builder pattern", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup).
			WithProduces("user", "User").
			WithRequires("config", "Config").
			WithTeardown("DeleteUser").
			WithDescription("Creates a test user").
			WithTags("setup", "user").
			WithOwner("test-team").
			WithVersion("1.0.0").
			WithSource("components.go", 42)

		if len(c.Produces) != 1 || c.Produces[0].Key != "user" {
			t.Error("WithProduces failed")
		}
		if len(c.Requires) != 1 || c.Requires[0].Key != "config" {
			t.Error("WithRequires failed")
		}
		if c.Teardown != "DeleteUser" {
			t.Error("WithTeardown failed")
		}
		if c.Description != "Creates a test user" {
			t.Error("WithDescription failed")
		}
		if len(c.Tags) != 2 || c.Tags[0] != "setup" {
			t.Error("WithTags failed")
		}
		if c.Owner != "test-team" {
			t.Error("WithOwner failed")
		}
		if c.Version != "1.0.0" {
			t.Error("WithVersion failed")
		}
		if c.SourceFile != "components.go" || c.SourceLine != 42 {
			t.Error("WithSource failed")
		}
	})

	t.Run("WithProducesDesc adds description", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup).
			WithProducesDesc("user", "User", "The created user")

		if c.Produces[0].Description != "The created user" {
			t.Error("WithProducesDesc should set description")
		}
	})

	t.Run("WithRequiresDesc adds description", func(t *testing.T) {
		c := NewComponent("CreateOrder", ComponentTask).
			WithRequiresDesc("user", "User", "The authenticated user")

		if c.Requires[0].Description != "The authenticated user" {
			t.Error("WithRequiresDesc should set description")
		}
	})
}

func TestComponentDeprecation(t *testing.T) {
	t.Run("not deprecated by default", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup)
		if c.IsDeprecated() {
			t.Error("component should not be deprecated by default")
		}
		if c.IsSunset() {
			t.Error("component should not be sunset by default")
		}
	})

	t.Run("WithDeprecated marks as deprecated", func(t *testing.T) {
		sunset := time.Now().AddDate(0, 1, 0) // 1 month from now
		c := NewComponent("OldMethod", ComponentTask).
			WithDeprecated("Use NewMethod instead", sunset)

		if !c.IsDeprecated() {
			t.Error("component should be deprecated")
		}
		if c.Deprecated != "Use NewMethod instead" {
			t.Error("deprecation message incorrect")
		}
		if c.IsSunset() {
			t.Error("component should not be sunset yet")
		}
	})

	t.Run("IsSunset returns true for past dates", func(t *testing.T) {
		sunset := time.Now().AddDate(0, 0, -1) // yesterday
		c := NewComponent("OldMethod", ComponentTask).
			WithDeprecated("Use NewMethod instead", sunset)

		if !c.IsSunset() {
			t.Error("component should be sunset")
		}
	})
}

func TestComponentTags(t *testing.T) {
	t.Run("HasTag finds existing tag", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup).
			WithTags("setup", "user", "critical")

		if !c.HasTag("user") {
			t.Error("HasTag should find 'user' tag")
		}
		if !c.HasTag("critical") {
			t.Error("HasTag should find 'critical' tag")
		}
	})

	t.Run("HasTag returns false for missing tag", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup).
			WithTags("setup", "user")

		if c.HasTag("missing") {
			t.Error("HasTag should return false for missing tag")
		}
	})
}

func TestComponentDependencies(t *testing.T) {
	t.Run("ProducesKey finds existing key", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup).
			WithProduces("user", "User").
			WithProduces("token", "Token")

		if !c.ProducesKey("user") {
			t.Error("ProducesKey should find 'user'")
		}
		if !c.ProducesKey("token") {
			t.Error("ProducesKey should find 'token'")
		}
		if c.ProducesKey("missing") {
			t.Error("ProducesKey should return false for missing key")
		}
	})

	t.Run("RequiresKey finds existing key", func(t *testing.T) {
		c := NewComponent("CreateOrder", ComponentTask).
			WithRequires("user", "User").
			WithRequires("cart", "Cart")

		if !c.RequiresKey("user") {
			t.Error("RequiresKey should find 'user'")
		}
		if !c.RequiresKey("cart") {
			t.Error("RequiresKey should find 'cart'")
		}
		if c.RequiresKey("missing") {
			t.Error("RequiresKey should return false for missing key")
		}
	})
}

func TestComponentValidation(t *testing.T) {
	t.Run("valid component passes validation", func(t *testing.T) {
		c := NewComponent("CreateUser", ComponentSetup)
		if err := c.Validate(); err != nil {
			t.Errorf("valid component should pass validation: %v", err)
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		c := &Component{Type: ComponentSetup}
		if err := c.Validate(); err == nil {
			t.Error("empty name should fail validation")
		}
	})

	t.Run("invalid type fails validation", func(t *testing.T) {
		c := &Component{Name: "Test", Type: ComponentType("invalid")}
		if err := c.Validate(); err == nil {
			t.Error("invalid type should fail validation")
		}
	})

	t.Run("teardown with produces fails validation", func(t *testing.T) {
		c := NewComponent("DeleteUser", ComponentTeardown).
			WithProduces("result", "bool")
		if err := c.Validate(); err == nil {
			t.Error("teardown with produces should fail validation")
		}
	})
}

func TestComponentClone(t *testing.T) {
	t.Run("clone creates independent copy", func(t *testing.T) {
		original := NewComponent("CreateUser", ComponentSetup).
			WithProduces("user", "User").
			WithRequires("config", "Config").
			WithTags("setup", "user").
			WithDescription("Original description")

		clone := original.Clone()

		// Verify values are copied
		if clone.Name != original.Name {
			t.Error("clone should have same name")
		}
		if clone.Type != original.Type {
			t.Error("clone should have same type")
		}
		if len(clone.Produces) != len(original.Produces) {
			t.Error("clone should have same produces")
		}
		if len(clone.Tags) != len(original.Tags) {
			t.Error("clone should have same tags")
		}

		// Verify independence
		clone.Name = "ModifiedName"
		clone.Produces = append(clone.Produces, Dependency{Key: "extra", Type: "Extra"})
		clone.Tags = append(clone.Tags, "extra")

		if original.Name == "ModifiedName" {
			t.Error("modifying clone should not affect original name")
		}
		if len(original.Produces) != 1 {
			t.Error("modifying clone should not affect original produces")
		}
		if len(original.Tags) != 2 {
			t.Error("modifying clone should not affect original tags")
		}
	})
}

func TestTypeInfo(t *testing.T) {
	t.Run("NewTypeInfo creates type info", func(t *testing.T) {
		ti := NewTypeInfo("User", reflect.TypeOf(struct{}{}))
		if ti.Name != "User" {
			t.Errorf("expected name 'User', got %s", ti.Name)
		}
	})

	t.Run("String without alias", func(t *testing.T) {
		ti := &TypeInfo{Name: "User"}
		if ti.String() != "User" {
			t.Errorf("expected 'User', got %s", ti.String())
		}
	})

	t.Run("String with alias", func(t *testing.T) {
		ti := &TypeInfo{Name: "Customer", IsAlias: true, AliasOf: "models.CustomerRecord"}
		expected := "Customer (alias of models.CustomerRecord)"
		if ti.String() != expected {
			t.Errorf("expected %s, got %s", expected, ti.String())
		}
	})
}
