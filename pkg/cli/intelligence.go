package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/joshua-temple/chronicle/pkg/intelligence"
)

var intelligenceCmd = &cobra.Command{
	Use:   "intelligence",
	Short: "Test intelligence and analysis tools",
	Long: `Commands for test intelligence features including flaky test detection,
performance regression analysis, and test impact analysis.`,
	Aliases: []string{"intel"},
}

var flakyCmd = &cobra.Command{
	Use:   "flaky",
	Short: "Flaky test detection and management",
}

var flakyReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate flaky test report",
	RunE:  runFlakyReport,
}

var flakyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List flaky tests",
	RunE:  runFlakyList,
}

var perfCmd = &cobra.Command{
	Use:   "performance",
	Short: "Performance regression analysis",
	Aliases: []string{"perf"},
}

var perfReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate performance report",
	RunE:  runPerfReport,
}

var perfRegressionsCmd = &cobra.Command{
	Use:   "regressions",
	Short: "List performance regressions",
	RunE:  runPerfRegressions,
}

var impactCmd = &cobra.Command{
	Use:   "impact",
	Short: "Test impact analysis",
}

var impactAnalyzeCmd = &cobra.Command{
	Use:   "analyze [base-ref] [head-ref]",
	Short: "Analyze impact of code changes",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runImpactAnalyze,
}

func init() {
	rootCmd.AddCommand(intelligenceCmd)

	// Flaky detection commands
	intelligenceCmd.AddCommand(flakyCmd)
	flakyCmd.AddCommand(flakyReportCmd)
	flakyCmd.AddCommand(flakyListCmd)

	flakyReportCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	flakyReportCmd.Flags().String("storage", ".chronicle/flaky", "Flaky data storage path")

	flakyListCmd.Flags().String("status", "", "Filter by status (suspected, confirmed, quarantined)")
	flakyListCmd.Flags().String("storage", ".chronicle/flaky", "Flaky data storage path")

	// Performance commands
	intelligenceCmd.AddCommand(perfCmd)
	perfCmd.AddCommand(perfReportCmd)
	perfCmd.AddCommand(perfRegressionsCmd)

	perfReportCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	perfReportCmd.Flags().String("storage", ".chronicle/performance", "Performance data storage path")

	perfRegressionsCmd.Flags().String("storage", ".chronicle/performance", "Performance data storage path")
	perfRegressionsCmd.Flags().Float64("threshold", 20.0, "Degradation threshold percentage")

	// Impact analysis commands
	intelligenceCmd.AddCommand(impactCmd)
	impactCmd.AddCommand(impactAnalyzeCmd)

	impactAnalyzeCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	impactAnalyzeCmd.Flags().Bool("uncommitted", false, "Analyze uncommitted changes")
	impactAnalyzeCmd.Flags().String("mappings", ".chronicle/impact", "Test mappings storage path")
}

func runFlakyReport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	storagePath, _ := cmd.Flags().GetString("storage")

	config := intelligence.DefaultFlakyDetectorConfig()
	config.StoragePath = storagePath

	detector := intelligence.NewFlakyDetector(config)
	report := detector.GenerateReport(context.Background())

	if format == "json" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Text format
	fmt.Println("Flaky Test Report")
	fmt.Println("=================")
	fmt.Printf("Generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339))

	fmt.Printf("Total Scenarios:    %d\n", report.TotalScenarios)
	fmt.Printf("Stable:             %d\n", report.StableScenarios)
	fmt.Printf("Suspected Flaky:    %d\n", report.SuspectedFlaky)
	fmt.Printf("Confirmed Flaky:    %d\n", report.ConfirmedFlaky)
	fmt.Printf("Quarantined:        %d\n", report.QuarantinedTests)

	if len(report.FlakyTests) > 0 {
		fmt.Println("\nFlaky Tests:")
		fmt.Println("------------")
		for _, ft := range report.FlakyTests {
			statusIcon := "?"
			switch ft.Status {
			case intelligence.FlakyStatusSuspected:
				statusIcon = "~"
			case intelligence.FlakyStatusConfirmed:
				statusIcon = "!"
			case intelligence.FlakyStatusQuarantined:
				statusIcon = "X"
			}
			fmt.Printf("  %s %-40s (score: %.2f, pass rate: %.1f%%, flips: %d)\n",
				statusIcon, ft.Name, ft.FlakyScore, ft.PassRate*100, ft.Flips)
		}
	}

	if len(report.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for _, rec := range report.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}

	return nil
}

func runFlakyList(cmd *cobra.Command, args []string) error {
	storagePath, _ := cmd.Flags().GetString("storage")
	statusFilter, _ := cmd.Flags().GetString("status")

	config := intelligence.DefaultFlakyDetectorConfig()
	config.StoragePath = storagePath

	detector := intelligence.NewFlakyDetector(config)

	var tests []*intelligence.TestHistory

	switch statusFilter {
	case "quarantined":
		tests = detector.GetQuarantinedTests()
	case "":
		tests = detector.GetFlakyTests()
	default:
		// Filter by status
		allFlaky := detector.GetFlakyTests()
		for _, t := range allFlaky {
			if strings.EqualFold(t.Status.String(), statusFilter) {
				tests = append(tests, t)
			}
		}
	}

	if len(tests) == 0 {
		fmt.Println("No flaky tests found.")
		return nil
	}

	fmt.Printf("Found %d flaky test(s):\n\n", len(tests))
	for _, t := range tests {
		fmt.Printf("  %-40s [%s] (score: %.2f)\n", t.ScenarioName, t.Status, t.FlakyScore)
	}

	return nil
}

func runPerfReport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	storagePath, _ := cmd.Flags().GetString("storage")

	config := intelligence.DefaultPerformanceTrackerConfig()
	config.StoragePath = storagePath

	tracker := intelligence.NewPerformanceTracker(config)
	report := tracker.GenerateReport(context.Background())

	if format == "json" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Text format
	fmt.Println("Performance Report")
	fmt.Println("==================")
	fmt.Printf("Generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339))

	fmt.Printf("Total Scenarios:    %d\n", report.TotalScenarios)
	fmt.Printf("Normal:             %d\n", report.NormalCount)
	fmt.Printf("Improved:           %d\n", report.ImprovedCount)
	fmt.Printf("Degraded:           %d\n", report.DegradedCount)
	fmt.Printf("Critical:           %d\n", report.CriticalCount)

	if len(report.Regressions) > 0 {
		fmt.Println("\nRegressions:")
		fmt.Println("-----------")
		for _, r := range report.Regressions {
			fmt.Printf("  %-40s %.1fms -> %.1fms (%+.1f%%) %s\n",
				r.ScenarioName, r.BaselineMean, r.CurrentMean, r.ChangePercent, r.TrendDirection)
		}
	}

	if len(report.Improvements) > 0 {
		fmt.Println("\nImprovements:")
		fmt.Println("------------")
		for _, r := range report.Improvements {
			fmt.Printf("  %-40s %.1fms -> %.1fms (%.1f%%) %s\n",
				r.ScenarioName, r.BaselineMean, r.CurrentMean, r.ChangePercent, r.TrendDirection)
		}
	}

	if len(report.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for _, rec := range report.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}

	return nil
}

func runPerfRegressions(cmd *cobra.Command, args []string) error {
	storagePath, _ := cmd.Flags().GetString("storage")

	config := intelligence.DefaultPerformanceTrackerConfig()
	config.StoragePath = storagePath

	tracker := intelligence.NewPerformanceTracker(config)
	regressions := tracker.GetRegressions()

	if len(regressions) == 0 {
		fmt.Println("No performance regressions detected.")
		return nil
	}

	fmt.Printf("Found %d performance regression(s):\n\n", len(regressions))
	for _, r := range regressions {
		status := r.Status.String()
		fmt.Printf("  [%s] %-40s (mean: %.1fms, p95: %.1fms)\n",
			strings.ToUpper(status), r.ScenarioName, r.Mean, r.P95)
	}

	return nil
}

func runImpactAnalyze(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	uncommitted, _ := cmd.Flags().GetBool("uncommitted")
	mappingsPath, _ := cmd.Flags().GetString("mappings")

	config := intelligence.DefaultImpactAnalyzerConfig()
	config.TestMappingsPath = mappingsPath

	analyzer := intelligence.NewImpactAnalyzer(config)

	var report *intelligence.ImpactReport
	var err error

	if uncommitted {
		report, err = analyzer.AnalyzeUncommitted(context.Background())
	} else {
		baseRef := "main"
		headRef := "HEAD"
		if len(args) >= 1 {
			baseRef = args[0]
		}
		if len(args) >= 2 {
			headRef = args[1]
		}
		report, err = analyzer.AnalyzeGitDiff(context.Background(), baseRef, headRef)
	}

	if err != nil {
		return fmt.Errorf("impact analysis failed: %w", err)
	}

	if format == "json" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Text format
	fmt.Println("Impact Analysis Report")
	fmt.Println("======================")
	fmt.Printf("Generated: %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Printf("Comparing: %s...%s\n\n", report.BaseRef, report.HeadRef)

	fmt.Printf("Files Changed:      %d\n", report.FilesChanged)
	fmt.Printf("Total Changes:      %d lines\n", report.TotalChanges)
	fmt.Printf("Impact Level:       %s\n", strings.ToUpper(report.ImpactLevel.String()))
	fmt.Printf("Affected Tests:     %d\n", report.AffectedTests)

	if len(report.SuggestedTests) > 0 {
		fmt.Println("\nSuggested Tests to Run:")
		for _, t := range report.SuggestedTests {
			fmt.Printf("  - %s\n", t)
		}
	}

	if len(report.SkippableTests) > 0 && len(report.SkippableTests) <= 10 {
		fmt.Println("\nTests That Can Be Skipped:")
		for _, t := range report.SkippableTests {
			fmt.Printf("  - %s\n", t)
		}
	} else if len(report.SkippableTests) > 10 {
		fmt.Printf("\n%d tests can be safely skipped.\n", len(report.SkippableTests))
	}

	if len(report.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for _, rec := range report.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}

	return nil
}

// PrintIntelligenceSummary prints a quick summary of test intelligence.
func PrintIntelligenceSummary(flakyPath, perfPath string) {
	// Flaky detection summary
	flakyConfig := intelligence.DefaultFlakyDetectorConfig()
	flakyConfig.StoragePath = flakyPath
	flakyDetector := intelligence.NewFlakyDetector(flakyConfig)
	flakyTests := flakyDetector.GetFlakyTests()
	quarantinedTests := flakyDetector.GetQuarantinedTests()

	// Performance summary
	perfConfig := intelligence.DefaultPerformanceTrackerConfig()
	perfConfig.StoragePath = perfPath
	perfTracker := intelligence.NewPerformanceTracker(perfConfig)
	regressions := perfTracker.GetRegressions()

	fmt.Println("\n📊 Test Intelligence Summary")
	fmt.Println("----------------------------")

	if len(flakyTests) > 0 || len(quarantinedTests) > 0 {
		fmt.Printf("⚠️  Flaky: %d suspected/confirmed, %d quarantined\n",
			len(flakyTests), len(quarantinedTests))
	} else {
		fmt.Println("✅ No flaky tests detected")
	}

	if len(regressions) > 0 {
		fmt.Printf("📉 Performance: %d regressions detected\n", len(regressions))
	} else {
		fmt.Println("✅ No performance regressions")
	}
}
