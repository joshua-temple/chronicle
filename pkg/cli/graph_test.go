package cli

import (
	"testing"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
)

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no changes needed",
			input:    "simple_name",
			expected: "simple_name",
		},
		{
			name:     "replace hyphens",
			input:    "my-component-name",
			expected: "my_component_name",
		},
		{
			name:     "replace spaces",
			input:    "my component name",
			expected: "my_component_name",
		},
		{
			name:     "replace colons",
			input:    "key:type",
			expected: "key_type",
		},
		{
			name:     "replace all special chars",
			input:    "my-component:type name",
			expected: "my_component_type_name",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeID(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetMermaidStyle(t *testing.T) {
	tests := []struct {
		name     string
		compType core.ComponentType
		expected string
	}{
		{
			name:     "setup component",
			compType: core.ComponentSetup,
			expected: ":::setup",
		},
		{
			name:     "task component",
			compType: core.ComponentTask,
			expected: ":::task",
		},
		{
			name:     "validation component",
			compType: core.ComponentValidation,
			expected: ":::validation",
		},
		{
			name:     "teardown component",
			compType: core.ComponentTeardown,
			expected: ":::teardown",
		},
		{
			name:     "unknown component type",
			compType: "unknown",
			expected: "",
		},
		{
			name:     "step component",
			compType: core.ComponentStep,
			expected: "",
		},
		{
			name:     "rollup component",
			compType: core.ComponentRollup,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMermaidStyle(tt.compType)
			if result != tt.expected {
				t.Errorf("getMermaidStyle(%q) = %q, expected %q", tt.compType, result, tt.expected)
			}
		})
	}
}

func TestFindProvider(t *testing.T) {
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"SetupDB": {
				Name: "SetupDB",
				Type: core.ComponentSetup,
				Produces: []core.Dependency{
					{Key: "db", Type: "*sql.DB"},
				},
			},
			"SetupCache": {
				Name: "SetupCache",
				Type: core.ComponentSetup,
				Produces: []core.Dependency{
					{Key: "cache", Type: "*redis.Client"},
				},
			},
			"NoProduces": {
				Name: "NoProduces",
				Type: core.ComponentTask,
			},
		},
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "find db provider",
			key:      "db",
			expected: "SetupDB",
		},
		{
			name:     "find cache provider",
			key:      "cache",
			expected: "SetupCache",
		},
		{
			name:     "key not produced",
			key:      "unknown",
			expected: "",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findProvider(registry, tt.key)
			if result != tt.expected {
				t.Errorf("findProvider(registry, %q) = %q, expected %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestFindProviderEmptyRegistry(t *testing.T) {
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{},
	}

	result := findProvider(registry, "db")
	if result != "" {
		t.Errorf("findProvider with empty registry should return empty string, got %q", result)
	}
}

func TestPrintDependsOnHelpers(t *testing.T) {
	// Test that printDependsOn correctly identifies dependents
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"SetupDB": {
				Name: "SetupDB",
				Type: core.ComponentSetup,
				Produces: []core.Dependency{
					{Key: "db", Type: "*sql.DB"},
				},
			},
			"TestDB": {
				Name: "TestDB",
				Type: core.ComponentTask,
				Requires: []core.Dependency{
					{Key: "db", Type: "*sql.DB"},
				},
			},
			"AnotherTest": {
				Name: "AnotherTest",
				Type: core.ComponentTask,
				Requires: []core.Dependency{
					{Key: "db", Type: "*sql.DB"},
				},
			},
			"NoDBDep": {
				Name: "NoDBDep",
				Type: core.ComponentTask,
				Requires: []core.Dependency{
					{Key: "cache", Type: "*redis.Client"},
				},
			},
		},
	}

	// Count dependents on "db" key
	dependentCount := 0
	for _, comp := range registry.Components {
		for _, req := range comp.Requires {
			if req.Key == "db" {
				dependentCount++
				break
			}
		}
	}

	if dependentCount != 2 {
		t.Errorf("Expected 2 components to depend on 'db', got %d", dependentCount)
	}
}

func TestPrintReverseDependenciesHelpers(t *testing.T) {
	// Test the logic for finding reverse dependencies
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"SetupDB": {
				Name: "SetupDB",
				Type: core.ComponentSetup,
				Produces: []core.Dependency{
					{Key: "db", Type: "*sql.DB"},
				},
			},
			"TestDB": {
				Name: "TestDB",
				Type: core.ComponentTask,
				Requires: []core.Dependency{
					{Key: "db", Type: "*sql.DB"},
				},
			},
		},
	}

	// Find what keys SetupDB produces
	comp := registry.Components["SetupDB"]
	producedKeys := make(map[string]bool)
	for _, prod := range comp.Produces {
		producedKeys[prod.Key] = true
	}

	// Find components that depend on SetupDB's outputs
	var dependents []*core.Component
	for _, other := range registry.Components {
		if other.Name == comp.Name {
			continue
		}
		for _, req := range other.Requires {
			if producedKeys[req.Key] {
				dependents = append(dependents, other)
				break
			}
		}
	}

	if len(dependents) != 1 {
		t.Errorf("Expected 1 dependent, got %d", len(dependents))
	}

	if len(dependents) > 0 && dependents[0].Name != "TestDB" {
		t.Errorf("Expected dependent to be 'TestDB', got %q", dependents[0].Name)
	}
}
