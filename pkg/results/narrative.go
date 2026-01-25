package results

import (
	"fmt"
	"strings"
	"time"
)

// NarrativeStyle defines the output style for narratives.
type NarrativeStyle int

const (
	// StyleBrief produces minimal output.
	StyleBrief NarrativeStyle = iota
	// StyleStandard produces readable output with key details.
	StyleStandard
	// StyleVerbose produces detailed output with all information.
	StyleVerbose
	// StyleMarkdown produces markdown-formatted output.
	StyleMarkdown
)

// NarrativeGenerator generates human-readable narratives from results.
type NarrativeGenerator struct {
	style      NarrativeStyle
	showTiming bool
	showLogs   bool
	maxErrors  int
}

// NarrativeOption configures a NarrativeGenerator.
type NarrativeOption func(*NarrativeGenerator)

// NewNarrativeGenerator creates a new narrative generator.
func NewNarrativeGenerator(opts ...NarrativeOption) *NarrativeGenerator {
	ng := &NarrativeGenerator{
		style:      StyleStandard,
		showTiming: true,
		showLogs:   false,
		maxErrors:  5,
	}

	for _, opt := range opts {
		opt(ng)
	}

	return ng
}

// WithStyle sets the narrative style.
func WithStyle(style NarrativeStyle) NarrativeOption {
	return func(ng *NarrativeGenerator) {
		ng.style = style
	}
}

// WithTiming enables/disables timing information.
func WithTiming(enabled bool) NarrativeOption {
	return func(ng *NarrativeGenerator) {
		ng.showTiming = enabled
	}
}

// WithLogs enables/disables log output.
func WithLogs(enabled bool) NarrativeOption {
	return func(ng *NarrativeGenerator) {
		ng.showLogs = enabled
	}
}

// WithMaxErrors sets the maximum number of errors to show.
func WithMaxErrors(max int) NarrativeOption {
	return func(ng *NarrativeGenerator) {
		ng.maxErrors = max
	}
}

// Generate creates a narrative from run results.
func (ng *NarrativeGenerator) Generate(result *RunResult) string {
	switch ng.style {
	case StyleBrief:
		return ng.generateBrief(result)
	case StyleVerbose:
		return ng.generateVerbose(result)
	case StyleMarkdown:
		return ng.generateMarkdown(result)
	default:
		return ng.generateStandard(result)
	}
}

func (ng *NarrativeGenerator) generateBrief(result *RunResult) string {
	status := "✓ PASS"
	if !result.IsSuccess() {
		status = "✗ FAIL"
	}

	return fmt.Sprintf("%s: %d/%d scenarios passed (%.1f%%) in %v",
		status,
		result.Stats.Passed,
		result.Stats.Total,
		result.PassRate(),
		result.Duration.Round(time.Millisecond))
}

func (ng *NarrativeGenerator) generateStandard(result *RunResult) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("Chronicle Run: %s\n", result.Name))
	b.WriteString(strings.Repeat("=", 50))
	b.WriteString("\n\n")

	// Summary
	if result.IsSuccess() {
		b.WriteString("Status: ✓ PASS\n")
	} else {
		b.WriteString("Status: ✗ FAIL\n")
	}

	b.WriteString(fmt.Sprintf("Results: %d passed, %d failed, %d skipped of %d total\n",
		result.Stats.Passed, result.Stats.Failed, result.Stats.Skipped, result.Stats.Total))

	if ng.showTiming {
		b.WriteString(fmt.Sprintf("Duration: %v (avg: %v)\n",
			result.Duration.Round(time.Millisecond),
			result.Stats.AvgDuration.Round(time.Millisecond)))
	}

	b.WriteString("\n")

	// Scenario details
	if result.Stats.Failed > 0 {
		b.WriteString("Failed Scenarios:\n")
		b.WriteString(strings.Repeat("-", 30))
		b.WriteString("\n")

		errorCount := 0
		for _, s := range result.FailedScenarios() {
			if errorCount >= ng.maxErrors {
				b.WriteString(fmt.Sprintf("  ... and %d more failures\n",
					result.Stats.Failed-ng.maxErrors))
				break
			}

			b.WriteString(fmt.Sprintf("  ✗ %s", s.ScenarioName))
			if ng.showTiming {
				b.WriteString(fmt.Sprintf(" (%v)", s.Duration.Round(time.Millisecond)))
			}
			b.WriteString("\n")

			if s.Error != "" {
				b.WriteString(fmt.Sprintf("    Error: %s\n", truncate(s.Error, 100)))
			}

			// Show failed flow items
			for _, f := range s.FlowResults {
				if f.State == "failed" {
					b.WriteString(fmt.Sprintf("    → %s [%s]: %s\n",
						f.Name, f.Type, truncate(f.Error, 80)))
				}
			}

			b.WriteString("\n")
			errorCount++
		}
	}

	// Passed scenarios (brief)
	if result.Stats.Passed > 0 {
		b.WriteString("Passed Scenarios:\n")
		b.WriteString(strings.Repeat("-", 30))
		b.WriteString("\n")

		for _, s := range result.Scenarios {
			if s.State == "completed" {
				b.WriteString(fmt.Sprintf("  ✓ %s", s.ScenarioName))
				if ng.showTiming {
					b.WriteString(fmt.Sprintf(" (%v)", s.Duration.Round(time.Millisecond)))
				}
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	// Skipped scenarios
	if result.Stats.Skipped > 0 {
		b.WriteString("Skipped Scenarios:\n")
		b.WriteString(strings.Repeat("-", 30))
		b.WriteString("\n")

		for _, s := range result.Scenarios {
			if s.State == "skipped" {
				b.WriteString(fmt.Sprintf("  ○ %s", s.ScenarioName))
				if s.SkipReason != "" {
					b.WriteString(fmt.Sprintf(" (%s)", s.SkipReason))
				}
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (ng *NarrativeGenerator) generateVerbose(result *RunResult) string {
	var b strings.Builder

	// Header
	b.WriteString("Chronicle Run Report\n")
	b.WriteString(fmt.Sprintf("Run: %s (ID: %s)\n", result.Name, result.ID))
	b.WriteString(strings.Repeat("=", 60))
	b.WriteString("\n\n")

	// Timing
	b.WriteString("Timing:\n")
	b.WriteString(fmt.Sprintf("  Started:  %s\n", result.StartTime.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("  Finished: %s\n", result.EndTime.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("  Duration: %v\n", result.Duration))
	b.WriteString("\n")

	// Configuration
	if result.Config.Parallelism > 0 || result.Config.FailFast {
		b.WriteString("Configuration:\n")
		if result.Config.Parallelism > 0 {
			b.WriteString(fmt.Sprintf("  Parallelism: %d\n", result.Config.Parallelism))
		}
		if result.Config.FailFast {
			b.WriteString("  Fail-fast: enabled\n")
		}
		if len(result.Config.Tags) > 0 {
			b.WriteString(fmt.Sprintf("  Tags: %v\n", result.Config.Tags))
		}
		b.WriteString("\n")
	}

	// Environment
	if result.Environment.Hostname != "" || result.Environment.CI {
		b.WriteString("Environment:\n")
		if result.Environment.Hostname != "" {
			b.WriteString(fmt.Sprintf("  Hostname: %s\n", result.Environment.Hostname))
		}
		if result.Environment.OS != "" {
			b.WriteString(fmt.Sprintf("  OS: %s/%s\n", result.Environment.OS, result.Environment.Arch))
		}
		if result.Environment.GoVersion != "" {
			b.WriteString(fmt.Sprintf("  Go: %s\n", result.Environment.GoVersion))
		}
		if result.Environment.CI {
			b.WriteString(fmt.Sprintf("  CI: %s\n", result.Environment.CIProvider))
			if result.Environment.Branch != "" {
				b.WriteString(fmt.Sprintf("  Branch: %s\n", result.Environment.Branch))
			}
			if result.Environment.Commit != "" {
				b.WriteString(fmt.Sprintf("  Commit: %s\n", result.Environment.Commit))
			}
		}
		b.WriteString("\n")
	}

	// Summary
	b.WriteString("Summary:\n")
	b.WriteString(fmt.Sprintf("  Status: %s\n", statusEmoji(result.IsSuccess())))
	b.WriteString(fmt.Sprintf("  Total:   %d\n", result.Stats.Total))
	b.WriteString(fmt.Sprintf("  Passed:  %d (%.1f%%)\n", result.Stats.Passed, result.PassRate()))
	b.WriteString(fmt.Sprintf("  Failed:  %d\n", result.Stats.Failed))
	b.WriteString(fmt.Sprintf("  Skipped: %d\n", result.Stats.Skipped))
	b.WriteString(fmt.Sprintf("  Avg Duration: %v\n", result.Stats.AvgDuration.Round(time.Millisecond)))
	b.WriteString("\n")

	// All scenarios in detail
	b.WriteString("Scenario Details:\n")
	b.WriteString(strings.Repeat("-", 60))
	b.WriteString("\n\n")

	for i, s := range result.Scenarios {
		b.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, statusEmoji(s.State == "completed"), s.ScenarioName))
		b.WriteString(fmt.Sprintf("   State: %s | Duration: %v\n", s.State, s.Duration.Round(time.Millisecond)))

		if s.Error != "" {
			b.WriteString(fmt.Sprintf("   Error: %s\n", s.Error))
		}
		if s.SkipReason != "" {
			b.WriteString(fmt.Sprintf("   Skip reason: %s\n", s.SkipReason))
		}

		// Flow items
		if len(s.FlowResults) > 0 {
			b.WriteString("   Flow:\n")
			for _, f := range s.FlowResults {
				var icon string
				switch f.State {
				case "failed":
					icon = "✗"
				case "skipped":
					icon = "○"
				default:
					icon = "✓"
				}
				b.WriteString(fmt.Sprintf("     %s %s [%s] %v\n",
					icon, f.Name, f.Type, f.Duration.Round(time.Millisecond)))
				if f.Error != "" {
					b.WriteString(fmt.Sprintf("       → %s\n", f.Error))
				}
			}
		}

		// Teardown
		if len(s.TeardownResults) > 0 {
			b.WriteString("   Teardown:\n")
			for _, t := range s.TeardownResults {
				icon := "✓"
				if t.State == "failed" {
					icon = "✗"
				}
				b.WriteString(fmt.Sprintf("     %s %s %v\n",
					icon, t.Name, t.Duration.Round(time.Millisecond)))
				if t.Error != "" {
					b.WriteString(fmt.Sprintf("       → %s\n", t.Error))
				}
			}
		}

		// Logs
		if ng.showLogs && len(s.Logs) > 0 {
			b.WriteString("   Logs:\n")
			for _, l := range s.Logs {
				b.WriteString(fmt.Sprintf("     [%s] %s: %s\n", l.Level, l.Component, l.Message))
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}

func (ng *NarrativeGenerator) generateMarkdown(result *RunResult) string {
	var b strings.Builder

	// Title
	b.WriteString(fmt.Sprintf("# Chronicle Run: %s\n\n", result.Name))

	// Status badge
	if result.IsSuccess() {
		b.WriteString("![Status](https://img.shields.io/badge/status-PASS-success)\n\n")
	} else {
		b.WriteString("![Status](https://img.shields.io/badge/status-FAIL-critical)\n\n")
	}

	// Summary table
	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Total | %d |\n", result.Stats.Total))
	b.WriteString(fmt.Sprintf("| Passed | %d (%.1f%%) |\n", result.Stats.Passed, result.PassRate()))
	b.WriteString(fmt.Sprintf("| Failed | %d |\n", result.Stats.Failed))
	b.WriteString(fmt.Sprintf("| Skipped | %d |\n", result.Stats.Skipped))
	b.WriteString(fmt.Sprintf("| Duration | %v |\n", result.Duration.Round(time.Millisecond)))
	b.WriteString("\n")

	// Failed scenarios
	if result.Stats.Failed > 0 {
		b.WriteString("## Failed Scenarios\n\n")
		for _, s := range result.FailedScenarios() {
			b.WriteString(fmt.Sprintf("### ❌ %s\n\n", s.ScenarioName))
			if s.Error != "" {
				b.WriteString(fmt.Sprintf("**Error:** `%s`\n\n", s.Error))
			}

			// Failed steps table
			hasFailedSteps := false
			for _, f := range s.FlowResults {
				if f.State == "failed" {
					if !hasFailedSteps {
						b.WriteString("| Step | Type | Error |\n")
						b.WriteString("|------|------|-------|\n")
						hasFailedSteps = true
					}
					b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", f.Name, f.Type, truncate(f.Error, 50)))
				}
			}
			if hasFailedSteps {
				b.WriteString("\n")
			}
		}
	}

	// Passed scenarios
	if result.Stats.Passed > 0 {
		b.WriteString("## Passed Scenarios\n\n")
		for _, s := range result.Scenarios {
			if s.State == "completed" {
				b.WriteString(fmt.Sprintf("- ✅ %s (%v)\n", s.ScenarioName, s.Duration.Round(time.Millisecond)))
			}
		}
		b.WriteString("\n")
	}

	// Skipped scenarios
	if result.Stats.Skipped > 0 {
		b.WriteString("## Skipped Scenarios\n\n")
		for _, s := range result.Scenarios {
			if s.State == "skipped" {
				reason := ""
				if s.SkipReason != "" {
					reason = fmt.Sprintf(" - %s", s.SkipReason)
				}
				b.WriteString(fmt.Sprintf("- ⏭️ %s%s\n", s.ScenarioName, reason))
			}
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("*Generated at %s*\n", result.EndTime.Format(time.RFC3339)))

	return b.String()
}

func statusEmoji(success bool) string {
	if success {
		return "✓ PASS"
	}
	return "✗ FAIL"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GenerateNarrative is a convenience function to generate a standard narrative.
func GenerateNarrative(result *RunResult) string {
	return NewNarrativeGenerator().Generate(result)
}

// GenerateBriefNarrative generates a one-line summary.
func GenerateBriefNarrative(result *RunResult) string {
	return NewNarrativeGenerator(WithStyle(StyleBrief)).Generate(result)
}

// GenerateMarkdownReport generates a markdown report.
func GenerateMarkdownReport(result *RunResult) string {
	return NewNarrativeGenerator(WithStyle(StyleMarkdown)).Generate(result)
}
