package cli

import (
	"testing"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]any
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]any{},
		},
		{
			name:     "single key=value",
			input:    []string{"key=value"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "multiple key=value pairs",
			input:    []string{"key1=value1", "key2=value2"},
			expected: map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "key without value (boolean flag)",
			input:    []string{"debug"},
			expected: map[string]any{"debug": true},
		},
		{
			name:     "mixed flags",
			input:    []string{"key=value", "debug", "env=production"},
			expected: map[string]any{"key": "value", "debug": true, "env": "production"},
		},
		{
			name:     "value with equals sign",
			input:    []string{"url=http://example.com?a=b"},
			expected: map[string]any{"url": "http://example.com?a=b"},
		},
		{
			name:     "empty value",
			input:    []string{"key="},
			expected: map[string]any{"key": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFlags(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parseFlags() returned %d items, expected %d", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("parseFlags()[%s] = %v, expected %v", k, result[k], v)
				}
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "item",
			expected: false,
		},
		{
			name:     "item present",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "item not present",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "case sensitive",
			slice:    []string{"A", "B", "C"},
			item:     "a",
			expected: false,
		},
		{
			name:     "first item",
			slice:    []string{"first", "second", "third"},
			item:     "first",
			expected: true,
		},
		{
			name:     "last item",
			slice:    []string{"first", "second", "third"},
			item:     "third",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v, expected %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

func TestHasAnyScenarioTag(t *testing.T) {
	tests := []struct {
		name         string
		scenarioTags []string
		filterTags   []string
		expected     bool
	}{
		{
			name:         "no scenario tags, no filter tags",
			scenarioTags: []string{},
			filterTags:   []string{},
			expected:     false,
		},
		{
			name:         "no scenario tags, some filter tags",
			scenarioTags: []string{},
			filterTags:   []string{"integration"},
			expected:     false,
		},
		{
			name:         "scenario has matching tag",
			scenarioTags: []string{"unit", "fast"},
			filterTags:   []string{"unit"},
			expected:     true,
		},
		{
			name:         "scenario has one of multiple filter tags",
			scenarioTags: []string{"unit", "fast"},
			filterTags:   []string{"integration", "fast"},
			expected:     true,
		},
		{
			name:         "scenario has none of filter tags",
			scenarioTags: []string{"unit", "fast"},
			filterTags:   []string{"integration", "slow"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &scenario.Scenario{
				Name: "test",
				Tags: tt.scenarioTags,
			}
			result := hasAnyScenarioTag(s, tt.filterTags)
			if result != tt.expected {
				t.Errorf("hasAnyScenarioTag() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterScenariosByArgs(t *testing.T) {
	// Create test scenarios
	scenarios := []*scenario.Scenario{
		{Name: "test-1", Tags: []string{"unit", "fast"}},
		{Name: "test-2", Tags: []string{"integration", "slow"}},
		{Name: "test-3", Tags: []string{"unit", "slow"}},
		{Name: "test-4", Tags: []string{"e2e"}},
	}

	tests := []struct {
		name           string
		names          []string
		tags           []string
		excludeTags    []string
		expectedCount  int
		expectedNames  []string
	}{
		{
			name:          "no filters returns all",
			names:         nil,
			tags:          nil,
			excludeTags:   nil,
			expectedCount: 4,
		},
		{
			name:          "filter by name",
			names:         []string{"test-1", "test-3"},
			tags:          nil,
			excludeTags:   nil,
			expectedCount: 2,
			expectedNames: []string{"test-1", "test-3"},
		},
		{
			name:          "filter by tag",
			names:         nil,
			tags:          []string{"unit"},
			excludeTags:   nil,
			expectedCount: 2,
			expectedNames: []string{"test-1", "test-3"},
		},
		{
			name:          "exclude by tag",
			names:         nil,
			tags:          nil,
			excludeTags:   []string{"slow"},
			expectedCount: 2,
			expectedNames: []string{"test-1", "test-4"},
		},
		{
			name:          "filter and exclude tags",
			names:         nil,
			tags:          []string{"unit"},
			excludeTags:   []string{"slow"},
			expectedCount: 1,
			expectedNames: []string{"test-1"},
		},
		{
			name:          "filter by name and tag",
			names:         []string{"test-1", "test-2"},
			tags:          []string{"unit"},
			excludeTags:   nil,
			expectedCount: 1,
			expectedNames: []string{"test-1"},
		},
		{
			name:          "no matching scenarios",
			names:         []string{"nonexistent"},
			tags:          nil,
			excludeTags:   nil,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterScenariosByArgs(scenarios, tt.names, tt.tags, tt.excludeTags)

			if len(result) != tt.expectedCount {
				t.Errorf("filterScenariosByArgs() returned %d scenarios, expected %d", len(result), tt.expectedCount)
			}

			if tt.expectedNames != nil {
				for i, name := range tt.expectedNames {
					if i >= len(result) || result[i].Name != name {
						t.Errorf("filterScenariosByArgs()[%d].Name = %v, expected %v", i, result[i].Name, name)
					}
				}
			}
		})
	}
}

func TestApplyModifiers(t *testing.T) {
	tests := []struct {
		name          string
		initialFlags  map[string]any
		newFlags      map[string]any
		options       []string
		chaosProfiles []string
		mockProfiles  []string
	}{
		{
			name:          "apply all modifiers",
			initialFlags:  map[string]any{"existing": "value"},
			newFlags:      map[string]any{"new": "flag"},
			options:       []string{"option1", "option2"},
			chaosProfiles: []string{"chaos1"},
			mockProfiles:  []string{"mock1"},
		},
		{
			name:          "override existing flag",
			initialFlags:  map[string]any{"key": "old"},
			newFlags:      map[string]any{"key": "new"},
			options:       nil,
			chaosProfiles: nil,
			mockProfiles:  nil,
		},
		{
			name:          "empty modifiers",
			initialFlags:  map[string]any{"key": "value"},
			newFlags:      nil,
			options:       nil,
			chaosProfiles: nil,
			mockProfiles:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &scenario.Scenario{
				Name:          "test",
				Flags:         make(map[string]any),
				Options:       []string{},
				ChaosProfiles: []string{},
				MockProfiles:  []string{},
			}

			// Copy initial flags
			for k, v := range tt.initialFlags {
				s.Flags[k] = v
			}

			applyModifiers(s, tt.newFlags, tt.options, tt.chaosProfiles, tt.mockProfiles)

			// Check flags were merged
			for k, v := range tt.newFlags {
				if s.Flags[k] != v {
					t.Errorf("applyModifiers() did not set flag %s = %v", k, v)
				}
			}

			// Check options were appended
			if len(s.Options) != len(tt.options) {
				t.Errorf("applyModifiers() options length = %d, expected %d", len(s.Options), len(tt.options))
			}

			// Check chaos profiles were appended
			if len(s.ChaosProfiles) != len(tt.chaosProfiles) {
				t.Errorf("applyModifiers() chaos profiles length = %d, expected %d", len(s.ChaosProfiles), len(tt.chaosProfiles))
			}

			// Check mock profiles were appended
			if len(s.MockProfiles) != len(tt.mockProfiles) {
				t.Errorf("applyModifiers() mock profiles length = %d, expected %d", len(s.MockProfiles), len(tt.mockProfiles))
			}
		})
	}
}

// Ensure scenario package types are used properly
var _ = core.ComponentSetup
