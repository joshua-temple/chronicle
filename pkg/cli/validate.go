package cli

import (
	"fmt"
	"os"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/discovery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and components",
	Long: `Validate Chronicle configuration and discovered components.

Checks for:
- Valid YAML syntax in configuration files
- All required dependencies can be satisfied
- No circular dependencies between components
- Scenario references valid components`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().Bool("check-cycles", true, "check for circular dependencies")
	validateCmd.Flags().Bool("check-deps", true, "check that all dependencies can be satisfied")
	validateCmd.Flags().Bool("strict", false, "treat warnings as errors")
	validateCmd.Flags().StringP("scenario", "s", "", "validate specific scenario only")
}

type ValidationResult struct {
	Errors   []string
	Warnings []string
}

func (v *ValidationResult) AddError(format string, args ...any) {
	v.Errors = append(v.Errors, fmt.Sprintf(format, args...))
}

func (v *ValidationResult) AddWarning(format string, args ...any) {
	v.Warnings = append(v.Warnings, fmt.Sprintf(format, args...))
}

func (v *ValidationResult) HasErrors() bool {
	return len(v.Errors) > 0
}

func (v *ValidationResult) HasWarnings() bool {
	return len(v.Warnings) > 0
}

func runValidate(cmd *cobra.Command, args []string) error {
	checkCycles, _ := cmd.Flags().GetBool("check-cycles")
	checkDeps, _ := cmd.Flags().GetBool("check-deps")
	strict, _ := cmd.Flags().GetBool("strict")
	scenarioFilter, _ := cmd.Flags().GetString("scenario")
	verbose := viper.GetBool("verbose")

	result := &ValidationResult{}

	// Step 1: Load and validate configuration
	cfg, err := loadConfig()
	if err != nil {
		result.AddError("Configuration: %v", err)
	} else {
		if verbose {
			fmt.Printf("Loaded configuration: %s\n", cfg.Name)
		}
		validateConfig(cfg, result)
	}

	// Step 2: Discover components
	var registry *discovery.Registry
	if cfg != nil {
		parser := discovery.NewParser(cfg.Discovery.Paths...)
		registry, err = parser.Discover()
		if err != nil {
			result.AddError("Discovery: %v", err)
		} else if verbose {
			fmt.Printf("Discovered %d components, %d types\n", len(registry.Components), len(registry.Types))
		}
	}

	// Step 3: Validate registry
	if registry != nil {
		if checkDeps {
			validateDependencies(registry, result)
		}
		if checkCycles {
			validateCycles(registry, result)
		}
	}

	// Step 4: Validate scenarios
	if cfg != nil && registry != nil {
		validateScenarios(cfg, registry, scenarioFilter, result)
	}

	// Output results
	printValidationResults(result)

	// Determine exit code
	if result.HasErrors() {
		return fmt.Errorf("validation failed with %d error(s)", len(result.Errors))
	}
	if strict && result.HasWarnings() {
		return fmt.Errorf("validation failed with %d warning(s) (strict mode)", len(result.Warnings))
	}

	fmt.Println("\n✓ Validation passed")
	return nil
}

func loadConfig() (*config.Config, error) {
	configFile := viper.GetString("config")
	if configFile != "" {
		return config.Load(configFile)
	}

	// Try default locations
	for _, name := range []string{"chronicle.yaml", "chronicle.yml"} {
		if _, err := os.Stat(name); err == nil {
			return config.Load(name)
		}
	}

	return nil, fmt.Errorf("no configuration file found (expected chronicle.yaml)")
}

func validateConfig(cfg *config.Config, result *ValidationResult) {
	if cfg.Name == "" {
		result.AddWarning("Configuration: 'name' is not set")
	}
	if cfg.Version == "" {
		result.AddWarning("Configuration: 'version' is not set")
	}
	if len(cfg.Discovery.Paths) == 0 {
		result.AddWarning("Configuration: no discovery paths configured")
	}
}

func validateDependencies(registry *discovery.Registry, result *ValidationResult) {
	if err := registry.Validate(); err != nil {
		result.AddError("Dependencies: %v", err)
	}
}

func validateCycles(registry *discovery.Registry, result *ValidationResult) {
	cycles := registry.DetectCycles()
	for _, cycle := range cycles {
		// cycle.Path contains the component names in the cycle
		result.AddError("Cycle detected: %s", cycle.String())
	}
}

func validateScenarios(cfg *config.Config, registry *discovery.Registry, filter string, result *ValidationResult) {
	for _, scenarioCfg := range cfg.Scenarios {
		// Skip if filtering and not matching
		if filter != "" && scenarioCfg.Name != filter {
			continue
		}

		// Check that all flow items reference valid components
		for _, item := range scenarioCfg.Flow {
			componentName := item.GetComponentName()
			if componentName == "" {
				continue
			}

			if !registry.HasComponent(componentName) {
				result.AddError("Scenario '%s': references unknown component '%s'", scenarioCfg.Name, componentName)
			}
		}

		// Check extends references
		if scenarioCfg.Extends != "" {
			found := false
			for _, s := range cfg.Scenarios {
				if s.Name == scenarioCfg.Extends {
					found = true
					break
				}
			}
			if !found {
				result.AddError("Scenario '%s': extends unknown scenario '%s'", scenarioCfg.Name, scenarioCfg.Extends)
			}
		}

		// Check chaos profile references
		for _, profile := range scenarioCfg.ChaosProfiles {
			if _, ok := cfg.ChaosProfiles[profile]; !ok {
				result.AddWarning("Scenario '%s': references unknown chaos profile '%s'", scenarioCfg.Name, profile)
			}
		}

		// Check mock profile references
		for _, profile := range scenarioCfg.MockProfiles {
			if _, ok := cfg.MockProfiles[profile]; !ok {
				result.AddWarning("Scenario '%s': references unknown mock profile '%s'", scenarioCfg.Name, profile)
			}
		}
	}
}

func printValidationResults(result *ValidationResult) {
	if len(result.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, e := range result.Errors {
			fmt.Printf("   • %s\n", e)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("   • %s\n", w)
		}
	}
}
