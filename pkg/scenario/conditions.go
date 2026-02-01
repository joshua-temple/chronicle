package scenario

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ConditionEvaluator evaluates skip conditions.
type ConditionEvaluator struct {
	env     map[string]string
	flags   map[string]any
	now     time.Time
	weekday string
}

// NewConditionEvaluator creates a new condition evaluator.
func NewConditionEvaluator(flags map[string]any) *ConditionEvaluator {
	// Capture environment variables
	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	now := time.Now()
	weekdays := []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

	return &ConditionEvaluator{
		env:     env,
		flags:   flags,
		now:     now,
		weekday: weekdays[now.Weekday()],
	}
}

// NewConditionEvaluatorWithEnv creates an evaluator with explicit environment.
func NewConditionEvaluatorWithEnv(env map[string]string, flags map[string]any) *ConditionEvaluator {
	e := NewConditionEvaluator(flags)
	e.env = env
	return e
}

// SetTime sets the time used for time-based conditions (for testing).
func (e *ConditionEvaluator) SetTime(t time.Time) {
	e.now = t
	weekdays := []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}
	e.weekday = weekdays[t.Weekday()]
}

// Evaluate evaluates a condition and returns whether it's satisfied.
func (e *ConditionEvaluator) Evaluate(cond Condition) (bool, error) {
	return e.EvaluateExpression(cond.Expression)
}

// EvaluateExpression evaluates a condition expression string.
func (e *ConditionEvaluator) EvaluateExpression(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}

	// Handle logical operators
	if result, handled, err := e.evaluateLogical(expr); handled {
		return result, err
	}

	// Handle specific expression patterns
	parsers := []func(string) (bool, bool, error){
		e.parseEnvIsSet,
		e.parseEnvIsEmpty,
		e.parseEnvEquals,
		e.parseEnvNotEquals,
		e.parseEnvIn,
		e.parseFlagEquals,
		e.parseFlagNotEquals,
		e.parseTimeComparison,
		e.parseWeekdayIn,
	}

	for _, parser := range parsers {
		result, matched, err := parser(expr)
		if matched {
			return result, err
		}
	}

	return false, fmt.Errorf("unsupported expression: %s", expr)
}

// evaluateLogical handles AND/OR logical operators.
func (e *ConditionEvaluator) evaluateLogical(expr string) (bool, bool, error) {
	// Simple handling of AND (&&) and OR (||)
	// Note: This is a basic implementation; a full parser would handle precedence

	// Check for OR first (lower precedence)
	if parts := strings.Split(expr, " || "); len(parts) > 1 {
		for _, part := range parts {
			result, err := e.EvaluateExpression(part)
			if err != nil {
				return false, true, err
			}
			if result {
				return true, true, nil
			}
		}
		return false, true, nil
	}

	// Check for AND
	if parts := strings.Split(expr, " && "); len(parts) > 1 {
		for _, part := range parts {
			result, err := e.EvaluateExpression(part)
			if err != nil {
				return false, true, err
			}
			if !result {
				return false, true, nil
			}
		}
		return true, true, nil
	}

	// Check for NOT
	if strings.HasPrefix(expr, "!") || strings.HasPrefix(expr, "not ") {
		var inner string
		if strings.HasPrefix(expr, "!") {
			inner = strings.TrimPrefix(expr, "!")
		} else {
			inner = strings.TrimPrefix(expr, "not ")
		}
		result, err := e.EvaluateExpression(strings.TrimSpace(inner))
		return !result, true, err
	}

	return false, false, nil
}

// Pattern matchers for different expression types
var (
	envIsSetPattern     = regexp.MustCompile(`^env\.(\w+)\s+is\s+set$`)
	envIsEmptyPattern   = regexp.MustCompile(`^env\.(\w+)\s+is\s+empty$`)
	envEqualsPattern    = regexp.MustCompile(`^env\.(\w+)\s*==\s*"([^"]*)"$`)
	envNotEqualsPattern = regexp.MustCompile(`^env\.(\w+)\s*!=\s*"([^"]*)"$`)
	envInPattern        = regexp.MustCompile(`^env\.(\w+)\s+in\s+\[([^\]]+)\]$`)
	flagEqualsPattern   = regexp.MustCompile(`^flags\.(\w+)\s*==\s*(.+)$`)
	flagNotEqualsPattern = regexp.MustCompile(`^flags\.(\w+)\s*!=\s*(.+)$`)
	timeHourPattern     = regexp.MustCompile(`^time\.hour\s*(>=|<=|>|<|==)\s*(\d+)$`)
	timeMinutePattern   = regexp.MustCompile(`^time\.minute\s*(>=|<=|>|<|==)\s*(\d+)$`)
	weekdayInPattern    = regexp.MustCompile(`^weekday\s+in\s+\[([^\]]+)\]$`)
)

func (e *ConditionEvaluator) parseEnvIsSet(expr string) (bool, bool, error) {
	matches := envIsSetPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	_, exists := e.env[matches[1]]
	return exists, true, nil
}

func (e *ConditionEvaluator) parseEnvIsEmpty(expr string) (bool, bool, error) {
	matches := envIsEmptyPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	val, exists := e.env[matches[1]]
	return !exists || val == "", true, nil
}

func (e *ConditionEvaluator) parseEnvEquals(expr string) (bool, bool, error) {
	matches := envEqualsPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	val, exists := e.env[matches[1]]
	return exists && val == matches[2], true, nil
}

func (e *ConditionEvaluator) parseEnvNotEquals(expr string) (bool, bool, error) {
	matches := envNotEqualsPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	val, exists := e.env[matches[1]]
	return !exists || val != matches[2], true, nil
}

func (e *ConditionEvaluator) parseEnvIn(expr string) (bool, bool, error) {
	matches := envInPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	val, exists := e.env[matches[1]]
	if !exists {
		return false, true, nil
	}
	values := parseStringList(matches[2])
	for _, v := range values {
		if val == v {
			return true, true, nil
		}
	}
	return false, true, nil
}

func (e *ConditionEvaluator) parseFlagEquals(expr string) (bool, bool, error) {
	matches := flagEqualsPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	flagName := matches[1]
	expected := strings.TrimSpace(matches[2])

	val, exists := e.flags[flagName]
	if !exists {
		return false, true, nil
	}

	// Handle different value types
	return compareValues(val, expected), true, nil
}

func (e *ConditionEvaluator) parseFlagNotEquals(expr string) (bool, bool, error) {
	matches := flagNotEqualsPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	flagName := matches[1]
	expected := strings.TrimSpace(matches[2])

	val, exists := e.flags[flagName]
	if !exists {
		return true, true, nil
	}

	return !compareValues(val, expected), true, nil
}

func (e *ConditionEvaluator) parseTimeComparison(expr string) (bool, bool, error) {
	// Time hour comparison
	if matches := timeHourPattern.FindStringSubmatch(expr); matches != nil {
		op := matches[1]
		expected, _ := strconv.Atoi(matches[2])
		hour := e.now.Hour()
		return compareInts(hour, op, expected), true, nil
	}

	// Time minute comparison
	if matches := timeMinutePattern.FindStringSubmatch(expr); matches != nil {
		op := matches[1]
		expected, _ := strconv.Atoi(matches[2])
		minute := e.now.Minute()
		return compareInts(minute, op, expected), true, nil
	}

	return false, false, nil
}

func (e *ConditionEvaluator) parseWeekdayIn(expr string) (bool, bool, error) {
	matches := weekdayInPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false, false, nil
	}
	days := parseStringList(matches[1])
	for _, d := range days {
		if strings.EqualFold(e.weekday, d) {
			return true, true, nil
		}
	}
	return false, true, nil
}

// Helper functions

func parseStringList(s string) []string {
	var result []string
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func compareValues(actual any, expected string) bool {
	// Remove quotes from expected if present
	expected = strings.Trim(expected, `"'`)

	switch v := actual.(type) {
	case bool:
		return v == (expected == "true")
	case int:
		e, err := strconv.Atoi(expected)
		return err == nil && v == e
	case int64:
		e, err := strconv.ParseInt(expected, 10, 64)
		return err == nil && v == e
	case float64:
		e, err := strconv.ParseFloat(expected, 64)
		return err == nil && v == e
	case string:
		return v == expected
	default:
		return fmt.Sprintf("%v", actual) == expected
	}
}

func compareInts(actual int, op string, expected int) bool {
	switch op {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case ">":
		return actual > expected
	case "<":
		return actual < expected
	case ">=":
		return actual >= expected
	case "<=":
		return actual <= expected
	default:
		return false
	}
}

// EvaluateSkipConditions evaluates skip conditions for a scenario.
// Returns (shouldSkip, reason) where reason is the first matched condition's reason.
func EvaluateSkipConditions(s *Scenario, flags map[string]any) (bool, string) {
	if flags == nil {
		flags = make(map[string]any)
	}
	evaluator := NewConditionEvaluator(flags)

	// Merge scenario flags with provided flags (scenario flags override)
	for k, v := range s.Flags {
		evaluator.flags[k] = v
	}

	// Check skip_if conditions (skip if ANY condition is true)
	for _, cond := range s.SkipIf {
		result, err := evaluator.Evaluate(cond)
		if err == nil && result {
			reason := cond.Reason
			if reason == "" {
				reason = fmt.Sprintf("skip_if condition matched: %s", cond.Expression)
			}
			return true, reason
		}
	}

	// Check skip_unless conditions (skip unless ALL conditions are true)
	for _, cond := range s.SkipUnless {
		result, err := evaluator.Evaluate(cond)
		if err == nil && !result {
			reason := cond.Reason
			if reason == "" {
				reason = fmt.Sprintf("skip_unless condition not met: %s", cond.Expression)
			}
			return true, reason
		}
	}

	return false, ""
}
