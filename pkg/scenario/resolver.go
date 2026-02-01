package scenario

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
)

// Resolver resolves scenarios from configuration and registry.
type Resolver struct {
	config   *config.Config
	registry *discovery.Registry
}

// NewResolver creates a new scenario resolver.
func NewResolver(cfg *config.Config, reg *discovery.Registry) *Resolver {
	return &Resolver{
		config:   cfg,
		registry: reg,
	}
}

// ResolveAll resolves all scenarios from configuration.
func (r *Resolver) ResolveAll() ([]*Scenario, error) {
	var scenarios []*Scenario

	// Build scenario map for inheritance lookup
	scenarioConfigs := make(map[string]*config.ScenarioConfig)
	for i := range r.config.Scenarios {
		cfg := &r.config.Scenarios[i]
		scenarioConfigs[cfg.Name] = cfg
	}

	// Resolve each scenario
	for i := range r.config.Scenarios {
		cfg := &r.config.Scenarios[i]

		// Skip abstract scenarios
		if cfg.Abstract {
			continue
		}

		resolved, err := r.resolveScenario(cfg, scenarioConfigs)
		if err != nil {
			return nil, fmt.Errorf("resolving scenario %s: %w", cfg.Name, err)
		}

		// Handle matrix expansion
		if len(cfg.Matrix) > 0 {
			expanded, err := r.expandMatrix(resolved)
			if err != nil {
				return nil, fmt.Errorf("expanding matrix for %s: %w", cfg.Name, err)
			}
			scenarios = append(scenarios, expanded...)
		} else {
			scenarios = append(scenarios, resolved)
		}
	}

	return scenarios, nil
}

// Resolve resolves a single scenario by name.
func (r *Resolver) Resolve(name string) (*Scenario, error) {
	cfg, ok := r.config.GetScenario(name)
	if !ok {
		return nil, fmt.Errorf("scenario not found: %s", name)
	}

	scenarioConfigs := make(map[string]*config.ScenarioConfig)
	for i := range r.config.Scenarios {
		c := &r.config.Scenarios[i]
		scenarioConfigs[c.Name] = c
	}

	return r.resolveScenario(cfg, scenarioConfigs)
}

// resolveScenario resolves a single scenario with inheritance.
func (r *Resolver) resolveScenario(cfg *config.ScenarioConfig, all map[string]*config.ScenarioConfig) (*Scenario, error) {
	s := NewScenario(cfg.Name)
	s.Description = cfg.Description
	s.Timeout = cfg.Timeout
	s.Tags = cfg.Tags
	s.Abstract = cfg.Abstract
	s.Extends = cfg.Extends

	// Handle inheritance
	if cfg.Extends != "" {
		parent, ok := all[cfg.Extends]
		if !ok {
			return nil, fmt.Errorf("parent scenario not found: %s", cfg.Extends)
		}
		if !parent.Abstract {
			return nil, fmt.Errorf("can only extend abstract scenarios: %s", cfg.Extends)
		}

		// Resolve parent first
		parentScenario, err := r.resolveScenario(parent, all)
		if err != nil {
			return nil, fmt.Errorf("resolving parent %s: %w", cfg.Extends, err)
		}

		// Inherit from parent
		r.inheritFrom(s, parentScenario)
	}

	// Resolve flow items
	for _, flowCfg := range cfg.Flow {
		item, err := r.resolveFlowItem(&flowCfg)
		if err != nil {
			return nil, fmt.Errorf("resolving flow item: %w", err)
		}
		s.AddFlow(item)
	}

	// Resolve teardown flow
	for _, flowCfg := range cfg.TeardownFlow {
		item, err := r.resolveFlowItem(&flowCfg)
		if err != nil {
			return nil, fmt.Errorf("resolving teardown item: %w", err)
		}
		s.AddTeardown(item)
	}

	// Apply flags
	for k, v := range cfg.Flags {
		s.SetFlag(k, v)
	}

	// Copy options and profiles
	s.Options = cfg.Options
	s.ChaosProfiles = cfg.ChaosProfiles
	s.MockProfiles = cfg.MockProfiles

	// Convert conditions
	for _, c := range cfg.SkipIf {
		s.SkipIf = append(s.SkipIf, r.convertCondition(&c))
	}
	for _, c := range cfg.SkipUnless {
		s.SkipUnless = append(s.SkipUnless, r.convertCondition(&c))
	}

	// Copy matrix
	for k, v := range cfg.Matrix {
		s.Matrix[k] = v
	}

	return s, nil
}

// inheritFrom copies properties from parent to child scenario.
func (r *Resolver) inheritFrom(child, parent *Scenario) {
	// Inherit flow (parent flow comes first)
	child.Flow = append(parent.Flow, child.Flow...)

	// Inherit teardown (child teardown comes first, then parent)
	child.TeardownFlow = append(child.TeardownFlow, parent.TeardownFlow...)

	// Inherit tags
	tagSet := make(map[string]bool)
	for _, t := range parent.Tags {
		tagSet[t] = true
	}
	for _, t := range child.Tags {
		tagSet[t] = true
	}
	child.Tags = nil
	for t := range tagSet {
		child.Tags = append(child.Tags, t)
	}

	// Inherit flags (child overrides parent)
	for k, v := range parent.Flags {
		if _, exists := child.Flags[k]; !exists {
			child.Flags[k] = v
		}
	}

	// Inherit options (merge unique)
	optSet := make(map[string]bool)
	for _, o := range parent.Options {
		optSet[o] = true
	}
	for _, o := range child.Options {
		optSet[o] = true
	}
	child.Options = nil
	for o := range optSet {
		child.Options = append(child.Options, o)
	}

	// Inherit timeout (child overrides if set)
	if child.Timeout == 0 && parent.Timeout > 0 {
		child.Timeout = parent.Timeout
	}

	// Inherit description if not set
	if child.Description == "" {
		child.Description = parent.Description
	}
}

// resolveFlowItem converts a config flow item to a resolved flow item.
func (r *Resolver) resolveFlowItem(cfg *config.FlowItemConfig) (FlowItem, error) {
	item := FlowItem{
		Timeout:   cfg.Timeout,
		DependsOn: cfg.DependsOn,
		Params:    cfg.Params,
		Parallel:  cfg.Parallel,
	}

	// Determine component type and name
	switch {
	case cfg.Setup != "":
		item.Type = core.ComponentSetup
		item.Name = cfg.Setup
	case cfg.Task != "":
		item.Type = core.ComponentTask
		item.Name = cfg.Task
	case cfg.Validation != "":
		item.Type = core.ComponentValidation
		item.Name = cfg.Validation
	case cfg.Step != "":
		item.Type = core.ComponentStep
		item.Name = cfg.Step
	case cfg.Rollup != "":
		item.Type = core.ComponentRollup
		item.Name = cfg.Rollup
	case cfg.Teardown != "":
		item.Type = core.ComponentTeardown
		item.Name = cfg.Teardown
	default:
		return item, fmt.Errorf("flow item has no component type specified")
	}

	// Resolve component reference if registry is available
	if r.registry != nil {
		if comp, ok := r.registry.GetComponentByName(item.Name); ok {
			item.Component = comp
		}
	}

	return item, nil
}

// convertCondition converts a config condition to a scenario condition.
func (r *Resolver) convertCondition(cfg *config.ConditionConfig) Condition {
	cond := Condition{
		Reason: cfg.Reason,
	}

	// Build expression from different condition types
	if cfg.Expression != "" {
		cond.Expression = cfg.Expression
	} else if cfg.Env != "" {
		cond.Expression = fmt.Sprintf("env.%s is set", cfg.Env)
	} else if cfg.Flag != "" {
		cond.Expression = fmt.Sprintf("flags.%s == true", cfg.Flag)
	}

	return cond
}

// expandMatrix expands a scenario with matrix parameters into multiple scenarios.
func (r *Resolver) expandMatrix(s *Scenario) ([]*Scenario, error) {
	if len(s.Matrix) == 0 {
		return []*Scenario{s}, nil
	}

	// Get all matrix keys and their values
	keys := make([]string, 0, len(s.Matrix))
	for k := range s.Matrix {
		keys = append(keys, k)
	}

	// Generate all combinations
	combinations := r.generateCombinations(s.Matrix, keys)

	// Create a scenario for each combination
	scenarios := make([]*Scenario, 0, len(combinations))
	for i, combo := range combinations {
		clone := s.Clone()

		// Set matrix index values
		clone.MatrixIndex = combo

		// Update scenario name to include matrix values
		var parts []string
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%v", combo[k]))
		}
		clone.Name = fmt.Sprintf("%s[%s]", s.Name, strings.Join(parts, ","))
		clone.ID = core.NewScenarioID()

		// Substitute matrix values in flow item params
		for j := range clone.Flow {
			clone.Flow[j].Params = r.substituteParams(clone.Flow[j].Params, combo)
		}

		_ = i // Unused, just for iteration
		scenarios = append(scenarios, clone)
	}

	return scenarios, nil
}

// generateCombinations generates all combinations of matrix values.
func (r *Resolver) generateCombinations(matrix map[string][]any, keys []string) []map[string]any {
	if len(keys) == 0 {
		return []map[string]any{{}}
	}

	key := keys[0]
	rest := keys[1:]
	restCombos := r.generateCombinations(matrix, rest)

	var result []map[string]any
	for _, value := range matrix[key] {
		for _, combo := range restCombos {
			newCombo := make(map[string]any)
			for k, v := range combo {
				newCombo[k] = v
			}
			newCombo[key] = value
			result = append(result, newCombo)
		}
	}

	return result
}

// substituteParams substitutes matrix values in parameters.
// Supports ${{ matrix.key }} syntax.
var matrixVarRegex = regexp.MustCompile(`\$\{\{\s*matrix\.(\w+)\s*\}\}`)

func (r *Resolver) substituteParams(params map[string]any, matrixValues map[string]any) map[string]any {
	if params == nil {
		return nil
	}

	result := make(map[string]any)
	for k, v := range params {
		result[k] = r.substituteValue(v, matrixValues)
	}
	return result
}

func (r *Resolver) substituteValue(value any, matrixValues map[string]any) any {
	switch v := value.(type) {
	case string:
		return matrixVarRegex.ReplaceAllStringFunc(v, func(match string) string {
			submatches := matrixVarRegex.FindStringSubmatch(match)
			if len(submatches) > 1 {
				key := submatches[1]
				if val, ok := matrixValues[key]; ok {
					return fmt.Sprintf("%v", val)
				}
			}
			return match
		})
	case map[string]any:
		return r.substituteParams(v, matrixValues)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = r.substituteValue(item, matrixValues)
		}
		return result
	default:
		return value
	}
}
