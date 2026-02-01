package cli

import (
	"testing"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
)

func TestFilterComponents(t *testing.T) {
	// Create a mock registry with components
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"SetupDB": {
				Name: "SetupDB",
				Type: core.ComponentSetup,
				Tags: []string{"database", "setup"},
			},
			"RunTest": {
				Name: "RunTest",
				Type: core.ComponentTask,
				Tags: []string{"test", "unit"},
			},
			"ValidateResult": {
				Name: "ValidateResult",
				Type: core.ComponentValidation,
				Tags: []string{"validation", "unit"},
			},
			"TeardownDB": {
				Name: "TeardownDB",
				Type: core.ComponentTeardown,
				Tags: []string{"database", "teardown"},
			},
		},
	}

	tests := []struct {
		name          string
		typeFilter    string
		tagsFilter    []string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "no filters returns all sorted",
			typeFilter:    "",
			tagsFilter:    nil,
			expectedCount: 4,
			expectedNames: []string{"RunTest", "SetupDB", "TeardownDB", "ValidateResult"},
		},
		{
			name:          "filter by type setup",
			typeFilter:    "setup",
			tagsFilter:    nil,
			expectedCount: 1,
			expectedNames: []string{"SetupDB"},
		},
		{
			name:          "filter by type task",
			typeFilter:    "task",
			tagsFilter:    nil,
			expectedCount: 1,
			expectedNames: []string{"RunTest"},
		},
		{
			name:          "filter by tag database",
			typeFilter:    "",
			tagsFilter:    []string{"database"},
			expectedCount: 2,
			expectedNames: []string{"SetupDB", "TeardownDB"},
		},
		{
			name:          "filter by tag unit",
			typeFilter:    "",
			tagsFilter:    []string{"unit"},
			expectedCount: 2,
			expectedNames: []string{"RunTest", "ValidateResult"},
		},
		{
			name:          "filter by type and tag",
			typeFilter:    "setup",
			tagsFilter:    []string{"database"},
			expectedCount: 1,
			expectedNames: []string{"SetupDB"},
		},
		{
			name:          "no matching type",
			typeFilter:    "rollup",
			tagsFilter:    nil,
			expectedCount: 0,
		},
		{
			name:          "no matching tag",
			typeFilter:    "",
			tagsFilter:    []string{"nonexistent"},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterComponents(registry, tt.typeFilter, tt.tagsFilter)

			if len(result) != tt.expectedCount {
				t.Errorf("filterComponents() returned %d components, expected %d", len(result), tt.expectedCount)
			}

			for i, name := range tt.expectedNames {
				if i >= len(result) || result[i].Name != name {
					actualName := ""
					if i < len(result) {
						actualName = result[i].Name
					}
					t.Errorf("filterComponents()[%d].Name = %q, expected %q", i, actualName, name)
				}
			}
		})
	}
}

func TestHasAnyTag(t *testing.T) {
	tests := []struct {
		name          string
		componentTags []string
		filterTags    []string
		expected      bool
	}{
		{
			name:          "empty component tags",
			componentTags: []string{},
			filterTags:    []string{"tag1"},
			expected:      false,
		},
		{
			name:          "empty filter tags",
			componentTags: []string{"tag1"},
			filterTags:    []string{},
			expected:      false,
		},
		{
			name:          "both empty",
			componentTags: []string{},
			filterTags:    []string{},
			expected:      false,
		},
		{
			name:          "matching tag",
			componentTags: []string{"tag1", "tag2"},
			filterTags:    []string{"tag2"},
			expected:      true,
		},
		{
			name:          "multiple matching tags",
			componentTags: []string{"tag1", "tag2", "tag3"},
			filterTags:    []string{"tag2", "tag3"},
			expected:      true,
		},
		{
			name:          "no matching tags",
			componentTags: []string{"tag1", "tag2"},
			filterTags:    []string{"tag3", "tag4"},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAnyTag(tt.componentTags, tt.filterTags)
			if result != tt.expected {
				t.Errorf("hasAnyTag(%v, %v) = %v, expected %v", tt.componentTags, tt.filterTags, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "string equal to max",
			input:    "exactly10!",
			maxLen:   10,
			expected: "exactly10!",
		},
		{
			name:     "string longer than max",
			input:    "this is a very long string",
			maxLen:   10,
			expected: "this is...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "max length of 3",
			input:    "hello",
			maxLen:   3,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestFormatDependencies(t *testing.T) {
	tests := []struct {
		name     string
		deps     []core.Dependency
		expected string
	}{
		{
			name:     "empty dependencies",
			deps:     []core.Dependency{},
			expected: "-",
		},
		{
			name:     "nil dependencies",
			deps:     nil,
			expected: "-",
		},
		{
			name: "single dependency",
			deps: []core.Dependency{
				{Key: "db", Type: "*sql.DB"},
			},
			expected: "db:*sql.DB",
		},
		{
			name: "multiple dependencies",
			deps: []core.Dependency{
				{Key: "db", Type: "*sql.DB"},
				{Key: "cache", Type: "*redis.Client"},
			},
			expected: "db:*sql.DB, cache:*redis.Client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDependencies(tt.deps)
			if result != tt.expected {
				t.Errorf("formatDependencies() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFormatDependenciesJSON(t *testing.T) {
	tests := []struct {
		name     string
		deps     []core.Dependency
		expected string
	}{
		{
			name:     "empty dependencies",
			deps:     []core.Dependency{},
			expected: "[]",
		},
		{
			name:     "nil dependencies",
			deps:     nil,
			expected: "[]",
		},
		{
			name: "single dependency",
			deps: []core.Dependency{
				{Key: "db", Type: "*sql.DB"},
			},
			expected: `[{"key": "db", "type": "*sql.DB"}]`,
		},
		{
			name: "multiple dependencies",
			deps: []core.Dependency{
				{Key: "db", Type: "*sql.DB"},
				{Key: "cache", Type: "*redis.Client"},
			},
			expected: `[{"key": "db", "type": "*sql.DB"}, {"key": "cache", "type": "*redis.Client"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDependenciesJSON(tt.deps)
			if result != tt.expected {
				t.Errorf("formatDependenciesJSON() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFormatTagsJSON(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "empty tags",
			tags:     []string{},
			expected: "[]",
		},
		{
			name:     "nil tags",
			tags:     nil,
			expected: "[]",
		},
		{
			name:     "single tag",
			tags:     []string{"unit"},
			expected: `["unit"]`,
		},
		{
			name:     "multiple tags",
			tags:     []string{"unit", "fast"},
			expected: `["unit", "fast"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTagsJSON(tt.tags)
			if result != tt.expected {
				t.Errorf("formatTagsJSON() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
