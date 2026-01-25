// Package cli provides the command-line interface for Chronicle.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version is set at build time.
	Version = "dev"

	// Commit is set at build time.
	Commit = "unknown"

	cfgFile string
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "chronicle",
	Short: "Chronicle - Integration Testing Framework",
	Long: `Chronicle is a powerful integration testing framework for Go.

It provides component discovery, scenario composition, infrastructure management,
chaos engineering, and comprehensive reporting for integration tests.

Use 'chronicle --help' for more information about available commands.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./chronicle.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")

	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(discoverCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(resultsCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(daemonCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.SetConfigName("chronicle")
		viper.SetConfigType("yaml")

		// Also try parent directories up to 3 levels
		wd, _ := os.Getwd()
		for i := 0; i < 3; i++ {
			parent := filepath.Dir(wd)
			if parent == wd {
				break
			}
			viper.AddConfigPath(parent)
			wd = parent
		}
	}

	viper.SetEnvPrefix("CHRONICLE")
	viper.AutomaticEnv()

	// Read config if exists
	_ = viper.ReadInConfig()
}

// versionCmd shows version information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Chronicle %s (commit: %s)\n", Version, Commit)
	},
}

// initCmd initializes a new Chronicle project.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Chronicle project",
	Long: `Initialize a new Chronicle project in the current directory.

This creates a chronicle.yaml configuration file and example directory structure.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if chronicle.yaml already exists
	if _, err := os.Stat("chronicle.yaml"); err == nil {
		return fmt.Errorf("chronicle.yaml already exists")
	}

	// Create default configuration
	defaultConfig := `# Chronicle Configuration
name: my-project
version: "1.0"

discovery:
  paths:
    - ./

infrastructure: {}

scenarios: []

flags:
  defaults:
    environment: local
`

	if err := os.WriteFile("chronicle.yaml", []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to create chronicle.yaml: %w", err)
	}

	fmt.Println("Created chronicle.yaml")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add component annotations to your Go files (@chronicle:setup, @chronicle:task, etc.)")
	fmt.Println("  2. Run 'chronicle discover' to see discovered components")
	fmt.Println("  3. Configure scenarios in chronicle.yaml")
	fmt.Println("  4. Run 'chronicle run' to execute your tests")

	return nil
}
