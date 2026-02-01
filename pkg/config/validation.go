package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects multiple validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("%d validation errors:\n  %s", len(e), strings.Join(msgs, "\n  "))
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Validate scenarios
	scenarioNames := make(map[string]bool)
	abstractScenarios := make(map[string]bool)
	for i, s := range c.Scenarios {
		// Check for duplicate names
		if scenarioNames[s.Name] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("scenarios[%d].name", i),
				Message: fmt.Sprintf("duplicate scenario name: %s", s.Name),
			})
		}
		scenarioNames[s.Name] = true

		if s.Abstract {
			abstractScenarios[s.Name] = true
		}

		// Validate scenario
		if errs := validateScenario(&s, i); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Check inheritance references
	for i, s := range c.Scenarios {
		if s.Extends != "" {
			if !scenarioNames[s.Extends] {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("scenarios[%d].extends", i),
					Message: fmt.Sprintf("extends non-existent scenario: %s", s.Extends),
				})
			}
			if !abstractScenarios[s.Extends] {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("scenarios[%d].extends", i),
					Message: fmt.Sprintf("can only extend abstract scenarios: %s is not abstract", s.Extends),
				})
			}
		}
	}

	// Validate infrastructure
	for name, infra := range c.Infrastructure {
		if errs := validateInfraConfig(name, &infra); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Validate chaos profiles
	for name, profile := range c.ChaosProfiles {
		if errs := validateChaosProfile(name, &profile); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Validate mock profiles
	for name, profile := range c.MockProfiles {
		if errs := validateMockProfile(name, &profile); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Validate suites
	for name, suite := range c.Suites {
		if errs := validateSuiteConfig(name, &suite, scenarioNames); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Validate execution config
	if errs := validateExecutionConfig(&c.Execution); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate results config
	if errs := validateResultsConfig(&c.Results); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func validateScenario(s *ScenarioConfig, index int) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("scenarios[%d]", index)

	// Name is required
	if s.Name == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".name",
			Message: "name is required",
		})
	} else if !isValidIdentifier(s.Name) {
		errors = append(errors, ValidationError{
			Field:   prefix + ".name",
			Message: "name must be a valid identifier (alphanumeric and hyphens)",
		})
	}

	// Non-abstract scenarios must have a flow
	if !s.Abstract && len(s.Flow) == 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".flow",
			Message: "non-abstract scenarios must have at least one flow item",
		})
	}

	// Validate flow items
	for i, item := range s.Flow {
		if errs := validateFlowItem(&item, fmt.Sprintf("%s.flow[%d]", prefix, i)); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Validate teardown flow items
	for i, item := range s.TeardownFlow {
		if errs := validateFlowItem(&item, fmt.Sprintf("%s.teardown[%d]", prefix, i)); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Validate conditions
	for i, cond := range s.SkipIf {
		if errs := validateCondition(&cond, fmt.Sprintf("%s.skip_if[%d]", prefix, i)); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}
	for i, cond := range s.SkipUnless {
		if errs := validateCondition(&cond, fmt.Sprintf("%s.skip_unless[%d]", prefix, i)); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	return errors
}

func validateFlowItem(item *FlowItemConfig, prefix string) ValidationErrors {
	var errors ValidationErrors

	// Count how many component types are set
	count := 0
	if item.Setup != "" {
		count++
	}
	if item.Task != "" {
		count++
	}
	if item.Validation != "" {
		count++
	}
	if item.Step != "" {
		count++
	}
	if item.Rollup != "" {
		count++
	}
	if item.Teardown != "" {
		count++
	}

	if count == 0 {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "flow item must specify exactly one component type (setup, task, validation, step, rollup, or teardown)",
		})
	} else if count > 1 {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "flow item must specify exactly one component type, found multiple",
		})
	}

	return errors
}

func validateCondition(cond *ConditionConfig, prefix string) ValidationErrors {
	var errors ValidationErrors

	// Must have at least one condition type
	if cond.Expression == "" && cond.Env == "" && cond.Flag == "" {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "condition must specify expression, env, or flag",
		})
	}

	return errors
}

func validateInfraConfig(name string, infra *InfraConfig) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("infrastructure.%s", name)

	if infra.Provider == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".provider",
			Message: "provider is required",
		})
	}

	// Validate health check
	if infra.HealthCheck.Interval > 0 && infra.HealthCheck.Timeout > infra.HealthCheck.Interval {
		errors = append(errors, ValidationError{
			Field:   prefix + ".health_check.timeout",
			Message: "timeout should not exceed interval",
		})
	}

	// Validate port configs
	for i, port := range infra.Ports {
		if port.Container <= 0 || port.Container > 65535 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.ports[%d].container", prefix, i),
				Message: "container port must be between 1 and 65535",
			})
		}
		if port.Host < 0 || port.Host > 65535 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.ports[%d].host", prefix, i),
				Message: "host port must be between 0 and 65535",
			})
		}
	}

	return errors
}

func validateChaosProfile(name string, profile *ChaosProfile) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("chaos_profiles.%s", name)

	// Validate latency config
	if profile.Network.Latency.Enabled {
		if profile.Network.Latency.Min <= 0 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".network.latency.min",
				Message: "min latency must be positive",
			})
		}
		if profile.Network.Latency.Max < profile.Network.Latency.Min {
			errors = append(errors, ValidationError{
				Field:   prefix + ".network.latency.max",
				Message: "max latency must be >= min latency",
			})
		}
	}

	// Validate packet loss config
	if profile.Network.PacketLoss.Enabled {
		if profile.Network.PacketLoss.Percentage <= 0 || profile.Network.PacketLoss.Percentage > 100 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".network.packet_loss.percentage",
				Message: "percentage must be between 0 and 100",
			})
		}
	}

	// Validate resource chaos
	if profile.Resource.CPU.Enabled {
		if profile.Resource.CPU.Percentage <= 0 || profile.Resource.CPU.Percentage > 100 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".resource.cpu.percentage",
				Message: "CPU percentage must be between 0 and 100",
			})
		}
	}
	if profile.Resource.Memory.Enabled {
		if profile.Resource.Memory.Percentage <= 0 || profile.Resource.Memory.Percentage > 100 {
			errors = append(errors, ValidationError{
				Field:   prefix + ".resource.memory.percentage",
				Message: "memory percentage must be between 0 and 100",
			})
		}
	}

	return errors
}

func validateMockProfile(name string, profile *MockProfile) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("mock_profiles.%s", name)

	if len(profile.Services) == 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".services",
			Message: "at least one service is required",
		})
	}

	for i, svc := range profile.Services {
		if svc.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.services[%d].name", prefix, i),
				Message: "service name is required",
			})
		}
		if svc.Type == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.services[%d].type", prefix, i),
				Message: "service type is required",
			})
		}
	}

	return errors
}

func validateSuiteConfig(name string, suite *SuiteConfig, scenarioNames map[string]bool) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("suites.%s", name)

	// Validate suite name
	if !isValidIdentifier(name) {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "suite name must be a valid identifier (alphanumeric and hyphens)",
		})
	}

	// A suite must have either scenarios or tags
	if len(suite.Scenarios) == 0 && len(suite.Tags) == 0 {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "suite must specify either scenarios or tags",
		})
	}

	// Validate scenario references
	for i, scenarioName := range suite.Scenarios {
		if !scenarioNames[scenarioName] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.scenarios[%d]", prefix, i),
				Message: fmt.Sprintf("references non-existent scenario: %s", scenarioName),
			})
		}
	}

	// Validate parallelism
	if suite.Parallel < 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".parallel",
			Message: "parallel cannot be negative",
		})
	}

	return errors
}

func validateExecutionConfig(exec *ExecutionConfig) ValidationErrors {
	var errors ValidationErrors
	prefix := "execution"

	if exec.Parallelism < 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".parallelism",
			Message: "parallelism cannot be negative",
		})
	}

	if exec.TeardownMode != "" {
		validModes := map[string]bool{
			"always":     true,
			"on_failure": true,
			"never":      true,
		}
		if !validModes[exec.TeardownMode] {
			errors = append(errors, ValidationError{
				Field:   prefix + ".teardown_mode",
				Message: "teardown_mode must be one of: always, on_failure, never",
			})
		}
	}

	if exec.RetryConfig.MaxRetries < 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".retry.max_retries",
			Message: "max_retries cannot be negative",
		})
	}

	if exec.RetryConfig.Backoff.Type != "" {
		validTypes := map[string]bool{
			"constant":    true,
			"exponential": true,
			"linear":      true,
		}
		if !validTypes[exec.RetryConfig.Backoff.Type] {
			errors = append(errors, ValidationError{
				Field:   prefix + ".retry.backoff.type",
				Message: "backoff type must be one of: constant, exponential, linear",
			})
		}
	}

	return errors
}

func validateResultsConfig(results *ResultsConfig) ValidationErrors {
	var errors ValidationErrors
	prefix := "results"

	if results.Storage.Type != "" {
		validTypes := map[string]bool{
			"file":     true,
			"s3":       true,
			"gcs":      true,
			"database": true,
		}
		if !validTypes[results.Storage.Type] {
			errors = append(errors, ValidationError{
				Field:   prefix + ".storage.type",
				Message: "storage type must be one of: file, s3, gcs, database",
			})
		}
	}

	for i, report := range results.Reports {
		validFormats := map[string]bool{
			"junit":    true,
			"json":     true,
			"html":     true,
			"markdown": true,
		}
		if !validFormats[report.Format] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.reports[%d].format", prefix, i),
				Message: "report format must be one of: junit, json, html, markdown",
			})
		}
	}

	return errors
}

// isValidIdentifier checks if a string is a valid identifier.
var identifierRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

func isValidIdentifier(s string) bool {
	return identifierRegex.MatchString(s)
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Name:    "chronicle",
		Version: "1.0",
		Discovery: DiscoveryConfig{
			Paths: []string{"."},
		},
		Infrastructure: make(map[string]InfraConfig),
		Suites:         make(map[string]SuiteConfig),
		ChaosProfiles:  make(map[string]ChaosProfile),
		MockProfiles:   make(map[string]MockProfile),
		Options:        make(map[string]OptionConfig),
		Execution: ExecutionConfig{
			Parallelism:    1,
			DefaultTimeout: 30 * time.Second,
			TeardownMode:   "always",
			FailFast:       false,
			RetryConfig: RetryConfig{
				MaxRetries: 0,
				Backoff: BackoffConfig{
					Type:       "exponential",
					Initial:    100 * time.Millisecond,
					Max:        5 * time.Second,
					Multiplier: 2.0,
					Jitter:     true,
				},
			},
		},
		Results: ResultsConfig{
			Storage: StorageConfig{
				Type: "file",
				Path: "./results",
			},
			Retention: RetentionConfig{
				Days:    30,
				Cleanup: true,
			},
		},
	}
}
