package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshua-temple/chronicle/pkg/core"
)

func TestParseDependencies(t *testing.T) {
	tests := []struct {
		input    string
		expected []core.Dependency
	}{
		{
			input: "user:User",
			expected: []core.Dependency{
				{Key: "user", Type: "User"},
			},
		},
		{
			input: "user:User,cart:Cart",
			expected: []core.Dependency{
				{Key: "user", Type: "User"},
				{Key: "cart", Type: "Cart"},
			},
		},
		{
			input: "user:*User, cart:*Cart",
			expected: []core.Dependency{
				{Key: "user", Type: "*User"},
				{Key: "cart", Type: "*Cart"},
			},
		},
		{
			input: "",
			expected: nil,
		},
		{
			input: "keyonly",
			expected: []core.Dependency{
				{Key: "keyonly"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDependencies(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d dependencies, got %d", len(tt.expected), len(result))
				return
			}
			for i, dep := range result {
				if dep.Key != tt.expected[i].Key || dep.Type != tt.expected[i].Type {
					t.Errorf("dependency %d: expected %v, got %v", i, tt.expected[i], dep)
				}
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "setup,user,critical",
			expected: []string{"setup", "user", "critical"},
		},
		{
			input:    "single",
			expected: []string{"single"},
		},
		{
			input:    " spaced , tags ",
			expected: []string{"spaced", "tags"},
		},
		{
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseTags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d", len(tt.expected), len(result))
				return
			}
			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("tag %d: expected %s, got %s", i, tt.expected[i], tag)
				}
			}
		})
	}
}

func TestParserAnnotationParsing(t *testing.T) {
	p := NewParser()

	tests := []struct {
		text     string
		expected *Annotation
	}{
		{
			text: `@chronicle:setup name="CreateUser" produces="user:User"`,
			expected: &Annotation{
				Type: "setup",
				Attributes: map[string]string{
					"name":     "CreateUser",
					"produces": "user:User",
				},
			},
		},
		{
			text: `@chronicle:task name="ProcessOrder" requires="user:User,cart:Cart" produces="order:Order"`,
			expected: &Annotation{
				Type: "task",
				Attributes: map[string]string{
					"name":     "ProcessOrder",
					"requires": "user:User,cart:Cart",
					"produces": "order:Order",
				},
			},
		},
		{
			text: `@chronicle:description "This is a description"`,
			expected: &Annotation{
				Type:       "description",
				Value:      "This is a description",
				Attributes: map[string]string{},
			},
		},
		{
			text: `@chronicle:tags setup,user,critical`,
			expected: &Annotation{
				Type:       "tags",
				Value:      "setup,user,critical",
				Attributes: map[string]string{},
			},
		},
		{
			text: `@chronicle:type`,
			expected: &Annotation{
				Type:       "type",
				Attributes: map[string]string{},
			},
		},
		{
			text: `@chronicle:deprecated "Use NewMethod instead" sunset="2025-06-01"`,
			expected: &Annotation{
				Type:  "deprecated",
				Value: "Use NewMethod instead",
				Attributes: map[string]string{
					"sunset": "2025-06-01",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := p.parseAnnotation(tt.text, "test.go", 1)
			if result == nil {
				t.Fatal("expected annotation, got nil")
			}
			if result.Type != tt.expected.Type {
				t.Errorf("type: expected %s, got %s", tt.expected.Type, result.Type)
			}
			if tt.expected.Value != "" && result.Value != tt.expected.Value {
				t.Errorf("value: expected %s, got %s", tt.expected.Value, result.Value)
			}
			for key, val := range tt.expected.Attributes {
				if result.Attributes[key] != val {
					t.Errorf("attribute %s: expected %s, got %s", key, val, result.Attributes[key])
				}
			}
		})
	}
}

func TestParserDiscovery(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "chronicle-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test file with annotations
	testCode := `package testpkg

// @chronicle:type
type User struct {
	ID    string
	Email string
}

// @chronicle:type
type Order struct {
	ID     string
	UserID string
	Total  float64
}

// @chronicle:setup name="CreateUser" produces="user:User" teardown="DeleteUser"
// @chronicle:description "Creates a test user for the scenario"
// @chronicle:tags setup,user
// @chronicle:owner test-team
func CreateUser() error {
	return nil
}

// @chronicle:teardown name="DeleteUser" requires="user:User"
func DeleteUser() error {
	return nil
}

// @chronicle:task name="CreateOrder" requires="user:User" produces="order:Order"
// @chronicle:description "Creates an order for the user"
// @chronicle:version "2"
func CreateOrder() error {
	return nil
}

// @chronicle:validation name="OrderValid" requires="order:Order"
// @chronicle:description "Validates the order was created correctly"
func OrderValid() error {
	return nil
}

// @chronicle:middleware name="LoggingMiddleware"
func LoggingMiddleware() {}
`
	testFile := filepath.Join(tmpDir, "components.go")
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse the directory
	parser := NewParser(tmpDir)
	registry, err := parser.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Verify types
	t.Run("discovers types", func(t *testing.T) {
		if len(registry.Types) != 2 {
			t.Errorf("expected 2 types, got %d", len(registry.Types))
		}
		if _, ok := registry.Types["User"]; !ok {
			t.Error("User type not found")
		}
		if _, ok := registry.Types["Order"]; !ok {
			t.Error("Order type not found")
		}
	})

	// Verify components
	t.Run("discovers components", func(t *testing.T) {
		if len(registry.Components) != 4 {
			t.Errorf("expected 4 components, got %d", len(registry.Components))
		}
	})

	t.Run("discovers setup component", func(t *testing.T) {
		c, ok := registry.GetComponentByName("CreateUser")
		if !ok {
			t.Fatal("CreateUser component not found")
		}
		if c.Type != core.ComponentSetup {
			t.Errorf("expected setup type, got %s", c.Type)
		}
		if len(c.Produces) != 1 || c.Produces[0].Key != "user" {
			t.Error("produces not parsed correctly")
		}
		if c.Teardown != "DeleteUser" {
			t.Errorf("teardown not parsed: expected DeleteUser, got %s", c.Teardown)
		}
		if c.Description != "Creates a test user for the scenario" {
			t.Errorf("description not parsed: %s", c.Description)
		}
		if !c.HasTag("setup") || !c.HasTag("user") {
			t.Error("tags not parsed correctly")
		}
		if c.Owner != "test-team" {
			t.Errorf("owner not parsed: %s", c.Owner)
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
		if len(c.Requires) != 1 || c.Requires[0].Key != "user" {
			t.Error("requires not parsed correctly")
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
		if len(c.Requires) != 1 || c.Requires[0].Key != "user" {
			t.Error("requires not parsed correctly")
		}
		if len(c.Produces) != 1 || c.Produces[0].Key != "order" {
			t.Error("produces not parsed correctly")
		}
		if c.Version != "2" {
			t.Errorf("version not parsed: %s", c.Version)
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
	})

	t.Run("discovers middleware", func(t *testing.T) {
		if len(registry.Middleware) != 1 {
			t.Errorf("expected 1 middleware, got %d", len(registry.Middleware))
		}
		if _, ok := registry.Middleware["LoggingMiddleware"]; !ok {
			t.Error("LoggingMiddleware not found")
		}
	})
}

func TestParserSkipsTestFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chronicle-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Regular file
	mainCode := `package testpkg

// @chronicle:setup name="MainSetup"
func MainSetup() error { return nil }
`
	// Test file (should be skipped)
	testCode := `package testpkg

// @chronicle:setup name="TestSetup"
func TestSetup() error { return nil }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser(tmpDir)
	registry, err := parser.Discover()
	if err != nil {
		t.Fatal(err)
	}

	if len(registry.Components) != 1 {
		t.Errorf("expected 1 component (from main.go only), got %d", len(registry.Components))
	}
	if _, ok := registry.GetComponentByName("TestSetup"); ok {
		t.Error("TestSetup should not be discovered from test file")
	}
}

func TestParserSkipsVendor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chronicle-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor", "somepackage")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainCode := `package testpkg
// @chronicle:setup name="MainSetup"
func MainSetup() error { return nil }
`
	vendorCode := `package somepackage
// @chronicle:setup name="VendorSetup"
func VendorSetup() error { return nil }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "vendor.go"), []byte(vendorCode), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser(tmpDir)
	registry, err := parser.Discover()
	if err != nil {
		t.Fatal(err)
	}

	if len(registry.Components) != 1 {
		t.Errorf("expected 1 component (from main.go only), got %d", len(registry.Components))
	}
	if _, ok := registry.GetComponentByName("VendorSetup"); ok {
		t.Error("VendorSetup should not be discovered from vendor")
	}
}
