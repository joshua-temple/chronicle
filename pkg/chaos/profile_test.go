package chaos

import (
	"context"
	"testing"
)

func TestNewProfile(t *testing.T) {
	profile := NewProfile("test-profile")

	if profile.Name != "test-profile" {
		t.Errorf("expected name 'test-profile', got %s", profile.Name)
	}
	if !profile.Enabled {
		t.Error("expected profile to be enabled by default")
	}
	if len(profile.Faults) != 0 {
		t.Errorf("expected no faults, got %d", len(profile.Faults))
	}
}

func TestProfileWithOptions(t *testing.T) {
	fault := NewErrorFault(ErrChaosInjected, 1.0)
	profile := NewProfile("test",
		WithDescription("A test profile"),
		WithFaults(fault),
		WithSelector(AllSelector{}),
	)

	if profile.Description != "A test profile" {
		t.Errorf("expected description, got %s", profile.Description)
	}
	if len(profile.Faults) != 1 {
		t.Errorf("expected 1 fault, got %d", len(profile.Faults))
	}
}

func TestProfileDisabled(t *testing.T) {
	profile := NewProfile("test", Disabled())

	if profile.Enabled {
		t.Error("expected profile to be disabled")
	}
}

func TestProfileEnableDisable(t *testing.T) {
	profile := NewProfile("test")

	profile.Disable()
	if profile.Enabled {
		t.Error("expected disabled")
	}

	profile.Enable()
	if !profile.Enabled {
		t.Error("expected enabled")
	}
}

func TestProfileAddFault(t *testing.T) {
	profile := NewProfile("test")
	fault := NewErrorFault(ErrChaosInjected, 1.0)

	result := profile.AddFault(fault)

	if result != profile {
		t.Error("expected fluent interface")
	}
	if len(profile.Faults) != 1 {
		t.Errorf("expected 1 fault, got %d", len(profile.Faults))
	}
}

func TestProfileApplyDisabled(t *testing.T) {
	profile := NewProfile("test", Disabled())
	profile.AddFault(NewErrorFault(ErrChaosInjected, 1.0))

	target := NewSimpleTarget("test-target", "component")
	err := profile.Apply(context.Background(), target)

	if err != nil {
		t.Errorf("expected no error for disabled profile, got %v", err)
	}
}

func TestProfileApplyWithSelector(t *testing.T) {
	profile := NewProfile("test",
		WithSelector(NameSelector{Pattern: "other"}),
		WithFaults(NewErrorFault(ErrChaosInjected, 1.0)),
	)

	// Should not apply - name doesn't match
	target := NewSimpleTarget("test-target", "component")
	err := profile.Apply(context.Background(), target)
	if err != nil {
		t.Errorf("expected no error (selector doesn't match), got %v", err)
	}

	// Should apply - name matches
	target = NewSimpleTarget("other", "component")
	err = profile.Apply(context.Background(), target)
	if err == nil {
		t.Error("expected error (selector matches)")
	}
}

func TestAllSelector(t *testing.T) {
	selector := AllSelector{}

	targets := []*SimpleTarget{
		NewSimpleTarget("target1", "type1"),
		NewSimpleTarget("target2", "type2"),
		NewSimpleTarget("", ""),
	}

	for _, target := range targets {
		if !selector.Matches(target) {
			t.Errorf("AllSelector should match all targets, failed for %s", target.Name())
		}
	}
}

func TestNameSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector NameSelector
		target   string
		want     bool
	}{
		{"exact match", NameSelector{Pattern: "test"}, "test", true},
		{"exact no match", NameSelector{Pattern: "test"}, "other", false},
		{"prefix match", NameSelector{Pattern: "test", Prefix: true}, "test-foo", true},
		{"prefix no match", NameSelector{Pattern: "test", Prefix: true}, "foo-test", false},
		{"suffix match", NameSelector{Pattern: "test", Suffix: true}, "foo-test", true},
		{"suffix no match", NameSelector{Pattern: "test", Suffix: true}, "test-foo", false},
		{"contains", NameSelector{Pattern: "test", Prefix: true, Suffix: true}, "foo-test-bar", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := NewSimpleTarget(tc.target, "type")
			got := tc.selector.Matches(target)
			if got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestTypeSelector(t *testing.T) {
	selector := TypeSelector{Types: []string{"setup", "teardown"}}

	tests := []struct {
		typ  string
		want bool
	}{
		{"setup", true},
		{"teardown", true},
		{"task", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.typ, func(t *testing.T) {
			target := NewSimpleTarget("test", tc.typ)
			got := selector.Matches(target)
			if got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestTagSelector(t *testing.T) {
	t.Run("match any", func(t *testing.T) {
		selector := TagSelector{Tags: []string{"slow", "flaky"}, MatchAll: false}

		target := NewSimpleTarget("test", "type", "slow")
		if !selector.Matches(target) {
			t.Error("expected match (has 'slow')")
		}

		target = NewSimpleTarget("test", "type", "fast")
		if selector.Matches(target) {
			t.Error("expected no match (doesn't have 'slow' or 'flaky')")
		}
	})

	t.Run("match all", func(t *testing.T) {
		selector := TagSelector{Tags: []string{"slow", "integration"}, MatchAll: true}

		target := NewSimpleTarget("test", "type", "slow", "integration")
		if !selector.Matches(target) {
			t.Error("expected match (has both tags)")
		}

		target = NewSimpleTarget("test", "type", "slow")
		if selector.Matches(target) {
			t.Error("expected no match (missing 'integration')")
		}
	})
}

func TestProbabilitySelector(t *testing.T) {
	// Test always match
	selector := NewProbabilitySelector(1.0)
	target := NewSimpleTarget("test", "type")

	matched := 0
	for i := 0; i < 100; i++ {
		if selector.Matches(target) {
			matched++
		}
	}
	if matched != 100 {
		t.Errorf("expected 100%% match with probability 1.0, got %d%%", matched)
	}

	// Test never match
	selector = NewProbabilitySelector(0.0)
	matched = 0
	for i := 0; i < 100; i++ {
		if selector.Matches(target) {
			matched++
		}
	}
	if matched != 0 {
		t.Errorf("expected 0%% match with probability 0.0, got %d%%", matched)
	}
}

func TestCompositeSelector(t *testing.T) {
	t.Run("mode and", func(t *testing.T) {
		selector := CompositeSelector{
			Selectors: []Selector{
				NameSelector{Pattern: "test", Prefix: true},
				TypeSelector{Types: []string{"setup"}},
			},
			Mode: ModeAnd,
		}

		// Both match
		target := NewSimpleTarget("test-foo", "setup")
		if !selector.Matches(target) {
			t.Error("expected match (both selectors match)")
		}

		// Only name matches
		target = NewSimpleTarget("test-foo", "task")
		if selector.Matches(target) {
			t.Error("expected no match (type doesn't match)")
		}
	})

	t.Run("mode or", func(t *testing.T) {
		selector := CompositeSelector{
			Selectors: []Selector{
				NameSelector{Pattern: "test"},
				TypeSelector{Types: []string{"setup"}},
			},
			Mode: ModeOr,
		}

		// Name matches
		target := NewSimpleTarget("test", "task")
		if !selector.Matches(target) {
			t.Error("expected match (name matches)")
		}

		// Type matches
		target = NewSimpleTarget("other", "setup")
		if !selector.Matches(target) {
			t.Error("expected match (type matches)")
		}

		// Neither matches
		target = NewSimpleTarget("other", "task")
		if selector.Matches(target) {
			t.Error("expected no match (neither matches)")
		}
	})

	t.Run("empty selectors", func(t *testing.T) {
		selector := CompositeSelector{Selectors: nil, Mode: ModeAnd}
		target := NewSimpleTarget("test", "type")
		if !selector.Matches(target) {
			t.Error("expected match for empty selectors")
		}
	})
}

func TestSimpleTarget(t *testing.T) {
	target := NewSimpleTarget("my-target", "my-type", "tag1", "tag2")

	if target.Name() != "my-target" {
		t.Errorf("expected name 'my-target', got %s", target.Name())
	}
	if target.Type() != "my-type" {
		t.Errorf("expected type 'my-type', got %s", target.Type())
	}
	tags := target.Tags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello", "hello", true},
		{"hello", "ell", true},
		{"hello", "xyz", false},
		{"", "a", false},
		{"hello", "", true},
	}

	for _, tc := range tests {
		got := contains(tc.s, tc.substr)
		if got != tc.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tc.s, tc.substr, got, tc.want)
		}
	}
}
