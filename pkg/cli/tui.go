package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/execution"
	"github.com/joshua-temple/chronicle/pkg/results"
	"github.com/joshua-temple/chronicle/pkg/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive terminal UI",
	Long: `Launch the Chronicle interactive terminal user interface.

The TUI provides:
- Scenario browsing and selection
- Live execution visualization
- Results browsing
- Tag-based filtering

Navigation:
  Arrow keys / j,k  Navigate lists
  Enter / r         Run selected scenario
  Tab               Switch between views
  /                 Filter scenarios
  ?                 Toggle help
  q                 Quit`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)

	tuiCmd.Flags().StringP("config", "c", "chronicle.yaml", "Path to configuration file")
	tuiCmd.Flags().StringP("results-dir", "r", ".chronicle/results", "Results storage directory")
}

func runTUI(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	resultsDir, _ := cmd.Flags().GetString("results-dir")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		// Config is optional for TUI
		fmt.Printf("Note: Could not load config from %s: %v\n", configPath, err)
		cfg = &config.Config{
			Name: "Chronicle",
		}
	}

	// Create executor
	executor := execution.NewExecutor()

	// Create storage
	storage, err := results.NewFileStorage(resultsDir)
	if err != nil {
		return fmt.Errorf("create storage: %w", err)
	}

	// Run TUI
	return tui.Run(cfg, executor, storage)
}
