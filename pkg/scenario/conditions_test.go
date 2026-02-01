package scenario

import (
	"testing"
	"time"
)

func TestConditionEvaluator_EnvConditions(t *testing.T) {
	env := map[string]string{
		"APP_ENV":   "production",
		"DEBUG":     "true",
		"EMPTY_VAR": "",
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		// env.VAR is set
		{"env is set - exists", "env.APP_ENV is set", true},
		{"env is set - not exists", "env.NONEXISTENT is set", false},
		{"env is set - empty but exists", "env.EMPTY_VAR is set", true},

		// env.VAR is empty
		{"env is empty - not set", "env.NONEXISTENT is empty", true},
		{"env is empty - empty value", "env.EMPTY_VAR is empty", true},
		{"env is empty - has value", "env.APP_ENV is empty", false},

		// env.VAR == "value"
		{"env equals - match", `env.APP_ENV == "production"`, true},
		{"env equals - no match", `env.APP_ENV == "development"`, false},
		{"env equals - not exists", `env.NONEXISTENT == "value"`, false},

		// env.VAR != "value"
		{"env not equals - different", `env.APP_ENV != "development"`, true},
		{"env not equals - same", `env.APP_ENV != "production"`, false},
		{"env not equals - not exists", `env.NONEXISTENT != "value"`, true},

		// env.VAR in [list]
		{"env in list - match", `env.APP_ENV in ["development", "production"]`, true},
		{"env in list - no match", `env.APP_ENV in ["staging", "test"]`, false},
		{"env in list - not exists", `env.NONEXISTENT in ["a", "b"]`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewConditionEvaluatorWithEnv(env, nil)
			result, err := e.EvaluateExpression(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v for expr: %s", tt.expected, result, tt.expr)
			}
		})
	}
}

func TestConditionEvaluator_FlagConditions(t *testing.T) {
	flags := map[string]any{
		"debug":    true,
		"verbose":  false,
		"count":    5,
		"name":     "test",
		"float":    3.14,
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		// flags.name == value
		{"flag equals bool true", "flags.debug == true", true},
		{"flag equals bool false", "flags.verbose == true", false},
		{"flag equals int", "flags.count == 5", true},
		{"flag equals int wrong", "flags.count == 10", false},
		{"flag equals string", `flags.name == "test"`, true},
		{"flag equals string quoted", `flags.name == 'test'`, true},
		{"flag equals nonexistent", "flags.nonexistent == true", false},

		// flags.name != value
		{"flag not equals - different", "flags.count != 10", true},
		{"flag not equals - same", "flags.count != 5", false},
		{"flag not equals - nonexistent", "flags.nonexistent != true", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewConditionEvaluatorWithEnv(nil, flags)
			result, err := e.EvaluateExpression(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v for expr: %s", tt.expected, result, tt.expr)
			}
		})
	}
}

func TestConditionEvaluator_TimeConditions(t *testing.T) {
	// Test at 14:30 on a Wednesday
	testTime := time.Date(2024, 1, 10, 14, 30, 0, 0, time.UTC) // Wednesday

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		// time.hour comparisons
		{"hour equals", "time.hour == 14", true},
		{"hour not equals", "time.hour == 10", false},
		{"hour greater", "time.hour > 12", true},
		{"hour less", "time.hour < 12", false},
		{"hour gte", "time.hour >= 14", true},
		{"hour lte", "time.hour <= 14", true},

		// time.minute comparisons
		{"minute equals", "time.minute == 30", true},
		{"minute greater", "time.minute > 15", true},

		// weekday in [list]
		{"weekday in list - match", `weekday in ["mon", "wed", "fri"]`, true},
		{"weekday in list - no match", `weekday in ["sat", "sun"]`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewConditionEvaluatorWithEnv(nil, nil)
			e.SetTime(testTime)
			result, err := e.EvaluateExpression(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v for expr: %s", tt.expected, result, tt.expr)
			}
		})
	}
}

func TestConditionEvaluator_LogicalOperators(t *testing.T) {
	env := map[string]string{
		"ENV": "production",
		"CI":  "true",
	}
	flags := map[string]any{
		"debug": true,
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		// AND (&&)
		{"and both true", `env.ENV == "production" && env.CI == "true"`, true},
		{"and first false", `env.ENV == "dev" && env.CI == "true"`, false},
		{"and second false", `env.ENV == "production" && env.CI == "false"`, false},
		{"and both false", `env.ENV == "dev" && env.CI == "false"`, false},

		// OR (||)
		{"or both true", `env.ENV == "production" || env.CI == "true"`, true},
		{"or first true", `env.ENV == "production" || env.CI == "false"`, true},
		{"or second true", `env.ENV == "dev" || env.CI == "true"`, true},
		{"or both false", `env.ENV == "dev" || env.CI == "false"`, false},

		// NOT (!)
		{"not true", `!env.NONEXISTENT is set`, true},
		{"not false", `!env.ENV is set`, false},
		{"not with keyword", "not env.NONEXISTENT is set", true},

		// Note: Complex expressions with parentheses are not yet supported
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewConditionEvaluatorWithEnv(env, flags)
			result, err := e.EvaluateExpression(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v for expr: %s", tt.expected, result, tt.expr)
			}
		})
	}
}

func TestConditionEvaluator_EmptyExpression(t *testing.T) {
	e := NewConditionEvaluatorWithEnv(nil, nil)

	result, err := e.EvaluateExpression("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("empty expression should return true")
	}

	result, err = e.EvaluateExpression("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("whitespace-only expression should return true")
	}
}

func TestConditionEvaluator_UnsupportedExpression(t *testing.T) {
	e := NewConditionEvaluatorWithEnv(nil, nil)

	_, err := e.EvaluateExpression("some.unknown.syntax")
	if err == nil {
		t.Error("expected error for unsupported expression")
	}
}

func TestEvaluateSkipConditions(t *testing.T) {
	t.Run("skip_if matches", func(t *testing.T) {
		s := NewScenario("test")
		s.SkipIf = append(s.SkipIf, Condition{
			Expression: "env.SKIP_TEST is set",
			Reason:     "Test explicitly skipped",
		})

		// When env is not set
		skip, reason := EvaluateSkipConditions(s, nil)
		if skip {
			t.Error("should not skip when env is not set")
		}

		// When env is set - need to set it temporarily
		// Since we can't easily mock os.Environ, we'll test the evaluator directly
		e := NewConditionEvaluatorWithEnv(map[string]string{"SKIP_TEST": "1"}, nil)
		result, _ := e.EvaluateExpression("env.SKIP_TEST is set")
		if !result {
			t.Error("condition should match when env is set")
		}
		_ = reason
	})

	t.Run("skip_unless not met", func(t *testing.T) {
		s := NewScenario("test")
		s.SkipUnless = append(s.SkipUnless, Condition{
			Expression: "flags.enabled == true",
			Reason:     "Feature not enabled",
		})

		// Without the flag
		skip, reason := EvaluateSkipConditions(s, nil)
		if !skip {
			t.Error("should skip when condition is not met")
		}
		if reason != "Feature not enabled" {
			t.Errorf("wrong reason: %s", reason)
		}

		// With the flag
		skip, _ = EvaluateSkipConditions(s, map[string]any{"enabled": true})
		if skip {
			t.Error("should not skip when condition is met")
		}
	})

	t.Run("multiple skip_if conditions", func(t *testing.T) {
		s := NewScenario("test")
		s.SkipIf = append(s.SkipIf,
			Condition{Expression: "flags.skip1 == true", Reason: "reason1"},
			Condition{Expression: "flags.skip2 == true", Reason: "reason2"},
		)

		// Neither true - should not skip
		skip, _ := EvaluateSkipConditions(s, nil)
		if skip {
			t.Error("should not skip when no conditions match")
		}

		// First true - should skip
		skip, reason := EvaluateSkipConditions(s, map[string]any{"skip1": true})
		if !skip {
			t.Error("should skip when first condition matches")
		}
		if reason != "reason1" {
			t.Error("should use first matching condition's reason")
		}

		// Second true - should skip
		skip, reason = EvaluateSkipConditions(s, map[string]any{"skip2": true})
		if !skip {
			t.Error("should skip when second condition matches")
		}
		if reason != "reason2" {
			t.Errorf("should use second condition's reason, got: %s", reason)
		}
	})

	t.Run("multiple skip_unless conditions", func(t *testing.T) {
		s := NewScenario("test")
		s.SkipUnless = append(s.SkipUnless,
			Condition{Expression: "flags.req1 == true", Reason: "need req1"},
			Condition{Expression: "flags.req2 == true", Reason: "need req2"},
		)

		// Neither true - should skip
		skip, _ := EvaluateSkipConditions(s, nil)
		if !skip {
			t.Error("should skip when conditions not met")
		}

		// Only first true - should skip (need both)
		skip, _ = EvaluateSkipConditions(s, map[string]any{"req1": true})
		if !skip {
			t.Error("should skip when only first condition met")
		}

		// Both true - should not skip
		skip, _ = EvaluateSkipConditions(s, map[string]any{"req1": true, "req2": true})
		if skip {
			t.Error("should not skip when all conditions met")
		}
	})

	t.Run("uses scenario flags", func(t *testing.T) {
		s := NewScenario("test")
		s.SetFlag("internal", true)
		s.SkipUnless = append(s.SkipUnless, Condition{
			Expression: "flags.internal == true",
			Reason:     "internal flag required",
		})

		// Should use scenario's flag
		skip, _ := EvaluateSkipConditions(s, nil)
		if skip {
			t.Error("should not skip when scenario has the required flag")
		}
	})
}

func TestCondition_Evaluate(t *testing.T) {
	cond := Condition{
		Expression: `env.APP_ENV == "production"`,
		Reason:     "Only run in production",
	}

	e := NewConditionEvaluatorWithEnv(map[string]string{"APP_ENV": "production"}, nil)
	result, err := e.Evaluate(cond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("condition should evaluate to true")
	}

	e = NewConditionEvaluatorWithEnv(map[string]string{"APP_ENV": "development"}, nil)
	result, err = e.Evaluate(cond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("condition should evaluate to false")
	}
}
