package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joshua-temple/chronicle/pkg/discovery"
	"github.com/joshua-temple/chronicle/pkg/execution"
	"github.com/joshua-temple/chronicle/pkg/results"
	"github.com/joshua-temple/chronicle/pkg/scenario"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run [scenario...]",
	Short: "Run scenarios",
	Long: `Run one or more scenarios.

If no scenarios are specified, all scenarios in the configuration are run.
Use --tags to filter scenarios by tags.`,
	RunE: runScenarios,
}

func init() {
	runCmd.Flags().StringSliceP("tags", "T", nil, "run only scenarios with matching tags")
	runCmd.Flags().StringSliceP("exclude-tags", "X", nil, "exclude scenarios with matching tags")
	runCmd.Flags().StringSliceP("flag", "F", nil, "set runtime flag (key=value)")
	runCmd.Flags().StringSlice("option", nil, "enable option bundle")
	runCmd.Flags().StringSlice("chaos", nil, "enable chaos profile")
	runCmd.Flags().StringSlice("mock", nil, "enable mock profile")
	runCmd.Flags().DurationP("timeout", "t", 30*time.Minute, "global timeout")
	runCmd.Flags().Int("parallel", 1, "number of scenarios to run in parallel")
	runCmd.Flags().Bool("fail-fast", false, "stop on first failure")
	runCmd.Flags().StringP("output", "o", "", "output directory for results")
	runCmd.Flags().StringP("format", "f", "text", "output format (text, json, junit)")
	runCmd.Flags().Bool("dry-run", false, "show what would run without executing")
}

func runScenarios(cmd *cobra.Command, args []string) error {
	// Parse flags
	tags, _ := cmd.Flags().GetStringSlice("tags")
	excludeTags, _ := cmd.Flags().GetStringSlice("exclude-tags")
	flagStrs, _ := cmd.Flags().GetStringSlice("flag")
	options, _ := cmd.Flags().GetStringSlice("option")
	chaosProfiles, _ := cmd.Flags().GetStringSlice("chaos")
	mockProfiles, _ := cmd.Flags().GetStringSlice("mock")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	parallel, _ := cmd.Flags().GetInt("parallel")
	failFast, _ := cmd.Flags().GetBool("fail-fast")
	outputDir, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose := viper.GetBool("verbose")

	// Parse flag key=value pairs
	flags := parseFlags(flagStrs)

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Discover components
	parser := discovery.NewParser(cfg.Discovery.Paths...)
	registry, err := parser.Discover()
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if verbose {
		fmt.Printf("Discovered %d components\n", len(registry.Components))
	}

	// Resolve and filter scenarios
	resolver := scenario.NewResolver(cfg, registry)
	allScenarios, err := resolver.ResolveAll()
	if err != nil {
		return fmt.Errorf("failed to resolve scenarios: %w", err)
	}

	// Filter scenarios by name, tags, etc.
	scenarios := filterScenariosByArgs(allScenarios, args, tags, excludeTags)
	if len(scenarios) == 0 {
		fmt.Println("No scenarios to run.")
		return nil
	}

	// Apply modifiers to scenarios
	for _, s := range scenarios {
		applyModifiers(s, flags, options, chaosProfiles, mockProfiles)
	}

	if dryRun {
		return printDryRun(scenarios)
	}

	// Setup signal handling
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, canceling...")
		cancel()
	}()

	// Create executor with options
	executor := execution.NewExecutor(
		execution.WithParallelism(parallel),
		execution.WithFailFast(failFast),
		execution.WithDefaultTimeout(cfg.Execution.DefaultTimeout),
	)

	// Register components from registry
	for _, comp := range registry.Components {
		executor.RegisterComponent(comp)
	}

	// Run scenarios
	fmt.Printf("Running %d scenario(s)...\n\n", len(scenarios))

	startTime := time.Now()
	execResults := executor.ExecuteMultiple(ctx, scenarios)
	duration := time.Since(startTime)

	// Setup results collector
	collector := results.NewCollector(cfg.Name)
	collector.AddAll(execResults)
	collector.SetConfig(results.RunConfig{
		Tags:        tags,
		Flags:       flags,
		Parallelism: parallel,
		FailFast:    failFast,
		Timeout:     timeout,
	})

	// Build final result
	runResult := collector.Build()

	// Generate output
	if err := outputResults(runResult, format, outputDir, verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to output results: %v\n", err)
	}

	// Print summary
	printRunSummary(runResult, duration)

	// Check for failures
	if runResult.Stats.Failed > 0 {
		return fmt.Errorf("%d scenario(s) failed", runResult.Stats.Failed)
	}

	return nil
}

func parseFlags(flagStrs []string) map[string]any {
	flags := make(map[string]any)
	for _, f := range flagStrs {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) == 2 {
			flags[parts[0]] = parts[1]
		} else {
			flags[parts[0]] = true
		}
	}
	return flags
}

func filterScenariosByArgs(scenarios []*scenario.Scenario, names []string, tags []string, excludeTags []string) []*scenario.Scenario {
	var result []*scenario.Scenario

	for _, s := range scenarios {
		// Filter by name if specified
		if len(names) > 0 && !contains(names, s.Name) {
			continue
		}

		// Filter by tags
		if len(tags) > 0 && !hasAnyScenarioTag(s, tags) {
			continue
		}

		// Exclude by tags
		if len(excludeTags) > 0 && hasAnyScenarioTag(s, excludeTags) {
			continue
		}

		result = append(result, s)
	}

	return result
}

func hasAnyScenarioTag(s *scenario.Scenario, tags []string) bool {
	for _, t := range tags {
		if s.HasTag(t) {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func applyModifiers(s *scenario.Scenario, flags map[string]any, options []string, chaosProfiles []string, mockProfiles []string) {
	// Merge flags
	for k, v := range flags {
		s.Flags[k] = v
	}

	// Add options
	s.Options = append(s.Options, options...)

	// Add chaos profiles
	s.ChaosProfiles = append(s.ChaosProfiles, chaosProfiles...)

	// Add mock profiles
	s.MockProfiles = append(s.MockProfiles, mockProfiles...)
}

func printDryRun(scenarios []*scenario.Scenario) error {
	fmt.Println("Dry run - would execute the following scenarios:")
	fmt.Println()

	for _, s := range scenarios {
		fmt.Printf("Scenario: %s\n", s.Name)
		if s.Description != "" {
			fmt.Printf("  Description: %s\n", s.Description)
		}
		if len(s.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(s.Tags, ", "))
		}
		if s.Timeout > 0 {
			fmt.Printf("  Timeout: %s\n", s.Timeout)
		}

		fmt.Println("  Flow:")
		for i, item := range s.Flow {
			fmt.Printf("    %d. [%s] %s\n", i+1, item.Type, item.Name)
		}

		if len(s.Flags) > 0 {
			fmt.Printf("  Flags: %v\n", s.Flags)
		}
		if len(s.ChaosProfiles) > 0 {
			fmt.Printf("  Chaos: %s\n", strings.Join(s.ChaosProfiles, ", "))
		}
		if len(s.MockProfiles) > 0 {
			fmt.Printf("  Mocks: %s\n", strings.Join(s.MockProfiles, ", "))
		}

		fmt.Println()
	}

	return nil
}

func outputResults(runResult *results.RunResult, format, outputDir string, verbose bool) error {
	// For text format, we just print the summary
	if format == "text" {
		return nil
	}

	// Get reporter for the format
	reporter, err := results.GetReporter(format)
	if err != nil {
		return err
	}

	data, err := reporter.Generate(runResult)
	if err != nil {
		return err
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return err
		}
		filepath := outputDir + "/results" + reporter.FileExtension()
		if err := os.WriteFile(filepath, data, 0644); err != nil {
			return err
		}
		if verbose {
			fmt.Printf("Results written to %s\n", filepath)
		}
	} else {
		fmt.Println(string(data))
	}

	return nil
}

func printRunSummary(runResult *results.RunResult, duration time.Duration) {
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Printf("Completed in %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Total: %d  Passed: %d  Failed: %d  Skipped: %d\n",
		runResult.Stats.Total, runResult.Stats.Passed, runResult.Stats.Failed, runResult.Stats.Skipped)

	if runResult.Stats.Failed > 0 {
		fmt.Println("\nFailed scenarios:")
		for _, s := range runResult.Scenarios {
			if s.State == "failed" {
				fmt.Printf("   - %s", s.ScenarioName)
				if s.Error != "" {
					fmt.Printf(": %s", s.Error)
				}
				fmt.Println()
			}
		}
	}
}
