package cli

import (
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
)

func TestValidationResult_AddError(t *testing.T) {
	result := &ValidationResult{}

	result.AddError("error %d", 1)
	result.AddError("error %d", 2)

	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}

	if result.Errors[0] != "error 1" {
		t.Errorf("Expected 'error 1', got %q", result.Errors[0])
	}

	if result.Errors[1] != "error 2" {
		t.Errorf("Expected 'error 2', got %q", result.Errors[1])
	}
}

func TestValidationResult_AddWarning(t *testing.T) {
	result := &ValidationResult{}

	result.AddWarning("warning %s", "one")
	result.AddWarning("warning %s", "two")

	if len(result.Warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(result.Warnings))
	}

	if result.Warnings[0] != "warning one" {
		t.Errorf("Expected 'warning one', got %q", result.Warnings[0])
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []string
		expected bool
	}{
		{
			name:     "no errors",
			errors:   nil,
			expected: false,
		},
		{
			name:     "empty errors slice",
			errors:   []string{},
			expected: false,
		},
		{
			name:     "has errors",
			errors:   []string{"error1", "error2"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{Errors: tt.errors}
			if result.HasErrors() != tt.expected {
				t.Errorf("HasErrors() = %v, expected %v", result.HasErrors(), tt.expected)
			}
		})
	}
}

func TestValidationResult_HasWarnings(t *testing.T) {
	tests := []struct {
		name     string
		warnings []string
		expected bool
	}{
		{
			name:     "no warnings",
			warnings: nil,
			expected: false,
		},
		{
			name:     "empty warnings slice",
			warnings: []string{},
			expected: false,
		},
		{
			name:     "has warnings",
			warnings: []string{"warn1"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{Warnings: tt.warnings}
			if result.HasWarnings() != tt.expected {
				t.Errorf("HasWarnings() = %v, expected %v", result.HasWarnings(), tt.expected)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name             string
		cfg              *config.Config
		expectedWarnings int
	}{
		{
			name: "fully valid config",
			cfg: &config.Config{
				Name:    "test-project",
				Version: "1.0.0",
				Discovery: config.DiscoveryConfig{
					Paths: []string{"./"},
				},
			},
			expectedWarnings: 0,
		},
		{
			name: "missing name",
			cfg: &config.Config{
				Version: "1.0.0",
				Discovery: config.DiscoveryConfig{
					Paths: []string{"./"},
				},
			},
			expectedWarnings: 1,
		},
		{
			name: "missing version",
			cfg: &config.Config{
				Name: "test-project",
				Discovery: config.DiscoveryConfig{
					Paths: []string{"./"},
				},
			},
			expectedWarnings: 1,
		},
		{
			name: "no discovery paths",
			cfg: &config.Config{
				Name:      "test-project",
				Version:   "1.0.0",
				Discovery: config.DiscoveryConfig{},
			},
			expectedWarnings: 1,
		},
		{
			name:             "all fields missing",
			cfg:              &config.Config{},
			expectedWarnings: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateConfig(tt.cfg, result)

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("validateConfig() produced %d warnings, expected %d",
					len(result.Warnings), tt.expectedWarnings)
			}
		})
	}
}

func TestValidateDependencies(t *testing.T) {
	tests := []struct {
		name           string
		registry       *discovery.Registry
		expectedErrors int
	}{
		{
			name: "valid dependencies",
			registry: &discovery.Registry{
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
			},
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateDependencies(tt.registry, result)

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("validateDependencies() produced %d errors, expected %d",
					len(result.Errors), tt.expectedErrors)
			}
		})
	}
}

func TestValidateScenarios(t *testing.T) {
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"ValidComponent": {
				Name: "ValidComponent",
				Type: core.ComponentTask,
			},
		},
	}

	tests := []struct {
		name           string
		cfg            *config.Config
		filter         string
		expectedErrors int
	}{
		{
			name: "valid scenario with existing component",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name: "test-scenario",
						Flow: []config.FlowItemConfig{
							{Task: "ValidComponent"},
						},
					},
				},
			},
			filter:         "",
			expectedErrors: 0,
		},
		{
			name: "scenario references unknown component",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name: "test-scenario",
						Flow: []config.FlowItemConfig{
							{Task: "NonExistent"},
						},
					},
				},
			},
			filter:         "",
			expectedErrors: 1,
		},
		{
			name: "scenario extends unknown scenario",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name:    "test-scenario",
						Extends: "unknown-base",
					},
				},
			},
			filter:         "",
			expectedErrors: 1,
		},
		{
			name: "filter matches scenario",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name: "test-scenario",
						Flow: []config.FlowItemConfig{
							{Task: "NonExistent"},
						},
					},
					{
						Name: "other-scenario",
						Flow: []config.FlowItemConfig{
							{Task: "AlsoNonExistent"},
						},
					},
				},
			},
			filter:         "test-scenario",
			expectedErrors: 1, // Only validates filtered scenario
		},
		{
			name: "filter excludes scenario",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name: "test-scenario",
						Flow: []config.FlowItemConfig{
							{Task: "NonExistent"},
						},
					},
				},
			},
			filter:         "other-scenario",
			expectedErrors: 0, // Scenario is filtered out
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateScenarios(tt.cfg, registry, tt.filter, result)

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("validateScenarios() produced %d errors, expected %d: %v",
					len(result.Errors), tt.expectedErrors, result.Errors)
			}
		})
	}
}

func TestValidateScenariosWithChaosProfiles(t *testing.T) {
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{},
	}

	tests := []struct {
		name             string
		cfg              *config.Config
		expectedWarnings int
	}{
		{
			name: "unknown chaos profile",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name:          "test-scenario",
						ChaosProfiles: []string{"unknown-chaos"},
					},
				},
				ChaosProfiles: map[string]config.ChaosProfile{},
			},
			expectedWarnings: 1,
		},
		{
			name: "valid chaos profile reference",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name:          "test-scenario",
						ChaosProfiles: []string{"network-failure"},
					},
				},
				ChaosProfiles: map[string]config.ChaosProfile{
					"network-failure": {
						Name:        "network-failure",
						Description: "Network failure simulation",
						Network: config.NetworkChaosConfig{
							Latency: config.LatencyConfig{
								Enabled: true,
								Min:     time.Millisecond * 100,
								Max:     time.Second,
							},
						},
					},
				},
			},
			expectedWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateScenarios(tt.cfg, registry, "", result)

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("validateScenarios() produced %d warnings, expected %d: %v",
					len(result.Warnings), tt.expectedWarnings, result.Warnings)
			}
		})
	}
}

func TestValidateScenariosWithMockProfiles(t *testing.T) {
	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{},
	}

	tests := []struct {
		name             string
		cfg              *config.Config
		expectedWarnings int
	}{
		{
			name: "unknown mock profile",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name:         "test-scenario",
						MockProfiles: []string{"unknown-mock"},
					},
				},
				MockProfiles: map[string]config.MockProfile{},
			},
			expectedWarnings: 1,
		},
		{
			name: "valid mock profile reference",
			cfg: &config.Config{
				Scenarios: []config.ScenarioConfig{
					{
						Name:         "test-scenario",
						MockProfiles: []string{"http-mock"},
					},
				},
				MockProfiles: map[string]config.MockProfile{
					"http-mock": {
						Name:        "http-mock",
						Description: "HTTP mock service",
						Services: []config.MockConfig{
							{
								Name: "api-mock",
								Type: "http",
							},
						},
					},
				},
			},
			expectedWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateScenarios(tt.cfg, registry, "", result)

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("validateScenarios() produced %d warnings, expected %d: %v",
					len(result.Warnings), tt.expectedWarnings, result.Warnings)
			}
		})
	}
}
