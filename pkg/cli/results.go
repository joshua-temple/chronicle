package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/results"
	"github.com/spf13/cobra"
)

var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Query and display test results",
	Long: `Query historical test results from storage.

Results are stored according to the configured storage adapter.
Use subcommands to list, show details, or delete results.`,
}

var resultsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List test results",
	RunE:  runResultsList,
}

var resultsShowCmd = &cobra.Command{
	Use:   "show [run-id]",
	Short: "Show details of a specific run",
	Args:  cobra.ExactArgs(1),
	RunE:  runResultsShow,
}

var resultsDeleteCmd = &cobra.Command{
	Use:   "delete [run-id...]",
	Short: "Delete test results",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runResultsDelete,
}

func init() {
	resultsCmd.AddCommand(resultsListCmd)
	resultsCmd.AddCommand(resultsShowCmd)
	resultsCmd.AddCommand(resultsDeleteCmd)

	resultsListCmd.Flags().IntP("limit", "n", 20, "maximum number of results to show")
	resultsListCmd.Flags().String("since", "", "show results since date (e.g., 24h, 7d, 2024-01-01)")
	resultsListCmd.Flags().StringP("format", "f", "table", "output format (table, json)")
}

func runResultsList(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	since, _ := cmd.Flags().GetString("since")
	format, _ := cmd.Flags().GetString("format")

	// Load configuration and get storage
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	storage, err := getStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Build list options
	var opts []results.ListOption
	opts = append(opts, results.WithListLimit(limit))

	if since != "" {
		sinceTime, err := parseSince(since)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		opts = append(opts, results.WithListAfter(sinceTime))
	}

	// List results
	ctx := context.Background()
	ids, err := storage.List(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to list results: %w", err)
	}

	if len(ids) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Load result details for display
	var runResults []*results.RunResult
	for _, id := range ids {
		r, err := storage.Load(ctx, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load result %s: %v\n", id, err)
			continue
		}
		runResults = append(runResults, r)
	}

	// Sort by start time (newest first)
	sort.Slice(runResults, func(i, j int) bool {
		return runResults[i].StartTime.After(runResults[j].StartTime)
	})

	if format == "json" {
		return printResultsJSON(runResults)
	}
	return printResultsTable(runResults)
}

func getStorage(cfg *config.Config) (results.Storage, error) {
	// Get storage path from config or use default
	storagePath := cfg.Results.Storage.Path
	if storagePath == "" {
		storagePath = ".chronicle/results"
	}

	// Create file storage
	return results.NewFileStorage(storagePath)
}

func parseSince(s string) (time.Time, error) {
	// Try parsing as duration
	if strings.HasSuffix(s, "h") || strings.HasSuffix(s, "d") || strings.HasSuffix(s, "m") {
		var d time.Duration
		if strings.HasSuffix(s, "d") {
			days := strings.TrimSuffix(s, "d")
			var n int
			if _, err := fmt.Sscanf(days, "%d", &n); err != nil {
				return time.Time{}, err
			}
			d = time.Duration(n) * 24 * time.Hour
		} else {
			var err error
			d, err = time.ParseDuration(s)
			if err != nil {
				return time.Time{}, err
			}
		}
		return time.Now().Add(-d), nil
	}

	// Try parsing as date
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

func printResultsTable(runResults []*results.RunResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "ID\tNAME\tPASS/FAIL/SKIP\tDURATION\tSTARTED")
	fmt.Fprintln(w, "--\t----\t--------------\t--------\t-------")

	for _, r := range runResults {
		started := r.StartTime.Format("2006-01-02 15:04:05")
		duration := r.Duration.Round(time.Millisecond).String()
		stats := fmt.Sprintf("%d/%d/%d", r.Stats.Passed, r.Stats.Failed, r.Stats.Skipped)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			truncate(r.ID, 12),
			r.Name,
			stats,
			duration,
			started,
		)
	}

	return nil
}

func printResultsJSON(runResults []*results.RunResult) error {
	fmt.Println("[")
	for i, r := range runResults {
		comma := ","
		if i == len(runResults)-1 {
			comma = ""
		}

		fmt.Printf(`  {"id": "%s", "name": "%s", "passed": %d, "failed": %d, "skipped": %d, "duration": "%s", "started": "%s"}%s
`,
			r.ID,
			r.Name,
			r.Stats.Passed,
			r.Stats.Failed,
			r.Stats.Skipped,
			r.Duration.String(),
			r.StartTime.Format(time.RFC3339),
			comma,
		)
	}
	fmt.Println("]")
	return nil
}

func runResultsShow(cmd *cobra.Command, args []string) error {
	runID := args[0]

	// Load configuration and get storage
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	storage, err := getStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Load result
	ctx := context.Background()
	result, err := storage.Load(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load result: %w", err)
	}

	// Print details
	fmt.Printf("Run ID:    %s\n", result.ID)
	fmt.Printf("Name:      %s\n", result.Name)
	fmt.Printf("Started:   %s\n", result.StartTime.Format(time.RFC3339))
	fmt.Printf("Duration:  %s\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("Results:   %d passed, %d failed, %d skipped\n",
		result.Stats.Passed, result.Stats.Failed, result.Stats.Skipped)

	if result.Environment.Hostname != "" {
		fmt.Printf("Host:      %s\n", result.Environment.Hostname)
	}

	fmt.Println("\nScenarios:")
	for _, s := range result.Scenarios {
		icon := "✓"
		if s.State == "failed" {
			icon = "✗"
		} else if s.State == "skipped" {
			icon = "○"
		}

		fmt.Printf("  %s %s (%s)\n", icon, s.ScenarioName, s.Duration.Round(time.Millisecond))

		for _, item := range s.FlowResults {
			indent := "    "
			itemIcon := "✓"
			if item.State == "failed" {
				itemIcon = "✗"
			} else if item.State == "skipped" {
				itemIcon = "○"
			}

			fmt.Printf("%s%s [%s] %s (%s)\n",
				indent,
				itemIcon,
				item.Type,
				item.Name,
				item.Duration.Round(time.Millisecond),
			)

			if item.Error != "" {
				fmt.Printf("%s  Error: %s\n", indent, item.Error)
			}
		}
	}

	return nil
}

func runResultsDelete(cmd *cobra.Command, args []string) error {
	// Load configuration and get storage
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	storage, err := getStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	ctx := context.Background()
	deleted := 0

	for _, id := range args {
		if err := storage.Delete(ctx, id); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete %s: %v\n", id, err)
			continue
		}
		deleted++
	}

	fmt.Printf("Deleted %d result(s)\n", deleted)
	return nil
}

var reportCmd = &cobra.Command{
	Use:   "report [run-id]",
	Short: "Generate reports from test results",
	Long: `Generate reports in various formats from test results.

If no run-id is specified, generates a report from the most recent run.`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().StringP("format", "f", "text", "report format (text, json, junit, html, markdown)")
	reportCmd.Flags().StringP("output", "o", "", "output file (default: stdout)")
	reportCmd.Flags().Bool("latest", false, "use the most recent run")
}

func runReport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	latest, _ := cmd.Flags().GetBool("latest")

	// Load configuration and get storage
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	storage, err := getStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	ctx := context.Background()

	// Determine which run to report on
	var runID string
	if len(args) > 0 {
		runID = args[0]
	} else if latest {
		ids, err := storage.List(ctx, results.WithListLimit(1))
		if err != nil {
			return fmt.Errorf("failed to list results: %w", err)
		}
		if len(ids) == 0 {
			return fmt.Errorf("no results found")
		}
		runID = ids[0]
	} else {
		return fmt.Errorf("specify a run ID or use --latest")
	}

	// Load result
	result, err := storage.Load(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load result: %w", err)
	}

	// Generate report using the helper function
	reporter, err := results.GetReporter(format)
	if err != nil {
		return fmt.Errorf("invalid format: %w", err)
	}

	data, err := reporter.Generate(result)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Output
	if output != "" {
		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		fmt.Printf("Report written to %s\n", output)
	} else {
		fmt.Println(string(data))
	}

	return nil
}
