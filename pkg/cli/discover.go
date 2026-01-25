package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover and list components",
	Long: `Discover components annotated with Chronicle annotations.

Scans the configured paths for Go files containing component annotations
like @chronicle:setup, @chronicle:task, @chronicle:validation, etc.`,
	RunE: runDiscover,
}

func init() {
	discoverCmd.Flags().StringSliceP("paths", "p", []string{"./"}, "paths to scan for components")
	discoverCmd.Flags().StringP("type", "t", "", "filter by component type (setup, task, validation, step, rollup, teardown)")
	discoverCmd.Flags().StringSliceP("tags", "T", nil, "filter by tags")
	discoverCmd.Flags().StringP("format", "f", "table", "output format (table, json, yaml)")
	discoverCmd.Flags().Bool("show-deps", false, "show component dependencies")
	discoverCmd.Flags().Bool("types-only", false, "only show discovered types")
}

func runDiscover(cmd *cobra.Command, args []string) error {
	paths, _ := cmd.Flags().GetStringSlice("paths")
	typeFilter, _ := cmd.Flags().GetString("type")
	tagsFilter, _ := cmd.Flags().GetStringSlice("tags")
	format, _ := cmd.Flags().GetString("format")
	showDeps, _ := cmd.Flags().GetBool("show-deps")
	typesOnly, _ := cmd.Flags().GetBool("types-only")
	verbose := viper.GetBool("verbose")

	// Use paths from config if available
	if cfgPaths := viper.GetStringSlice("discovery.paths"); len(cfgPaths) > 0 && len(paths) == 1 && paths[0] == "./" {
		paths = cfgPaths
	}

	if verbose {
		fmt.Printf("Scanning paths: %v\n", paths)
	}

	// Create parser and discover
	parser := discovery.NewParser(paths...)
	registry, err := parser.Discover()
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if typesOnly {
		return printTypes(registry, format)
	}

	// Filter components
	components := filterComponents(registry, typeFilter, tagsFilter)

	// Output based on format
	switch format {
	case "json":
		return printComponentsJSON(components, showDeps)
	case "yaml":
		return printComponentsYAML(components, showDeps)
	default:
		return printComponentsTable(components, showDeps)
	}
}

func filterComponents(registry *discovery.Registry, typeFilter string, tagsFilter []string) []*core.Component {
	var result []*core.Component

	for _, comp := range registry.Components {
		// Type filter
		if typeFilter != "" && string(comp.Type) != typeFilter {
			continue
		}

		// Tags filter
		if len(tagsFilter) > 0 && !hasAnyTag(comp.Tags, tagsFilter) {
			continue
		}

		result = append(result, comp)
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

func hasAnyTag(componentTags []string, filterTags []string) bool {
	tagSet := make(map[string]bool)
	for _, t := range componentTags {
		tagSet[t] = true
	}
	for _, t := range filterTags {
		if tagSet[t] {
			return true
		}
	}
	return false
}

func printComponentsTable(components []*core.Component, showDeps bool) error {
	if len(components) == 0 {
		fmt.Println("No components discovered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	if showDeps {
		fmt.Fprintln(w, "NAME\tTYPE\tPRODUCES\tREQUIRES\tTAGS")
		fmt.Fprintln(w, "----\t----\t--------\t--------\t----")
		for _, comp := range components {
			produces := formatDependencies(comp.Produces)
			requires := formatDependencies(comp.Requires)
			tags := strings.Join(comp.Tags, ", ")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", comp.Name, comp.Type, produces, requires, tags)
		}
	} else {
		fmt.Fprintln(w, "NAME\tTYPE\tTAGS\tDESCRIPTION")
		fmt.Fprintln(w, "----\t----\t----\t-----------")
		for _, comp := range components {
			tags := strings.Join(comp.Tags, ", ")
			desc := truncate(comp.Description, 50)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", comp.Name, comp.Type, tags, desc)
		}
	}

	fmt.Fprintf(os.Stdout, "\nFound %d component(s)\n", len(components))
	return nil
}

func formatDependencies(deps []core.Dependency) string {
	if len(deps) == 0 {
		return "-"
	}
	var parts []string
	for _, d := range deps {
		parts = append(parts, fmt.Sprintf("%s:%s", d.Key, d.Type))
	}
	return strings.Join(parts, ", ")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func printComponentsJSON(components []*core.Component, showDeps bool) error {
	fmt.Println("[")
	for i, comp := range components {
		comma := ","
		if i == len(components)-1 {
			comma = ""
		}
		if showDeps {
			produces := formatDependenciesJSON(comp.Produces)
			requires := formatDependenciesJSON(comp.Requires)
			fmt.Printf(`  {"name": "%s", "type": "%s", "produces": %s, "requires": %s}%s
`, comp.Name, comp.Type, produces, requires, comma)
		} else {
			fmt.Printf(`  {"name": "%s", "type": "%s", "tags": %s}%s
`, comp.Name, comp.Type, formatTagsJSON(comp.Tags), comma)
		}
	}
	fmt.Println("]")
	return nil
}

func formatDependenciesJSON(deps []core.Dependency) string {
	if len(deps) == 0 {
		return "[]"
	}
	var parts []string
	for _, d := range deps {
		parts = append(parts, fmt.Sprintf(`{"key": "%s", "type": "%s"}`, d.Key, d.Type))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatTagsJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	var parts []string
	for _, t := range tags {
		parts = append(parts, fmt.Sprintf(`"%s"`, t))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func printComponentsYAML(components []*core.Component, showDeps bool) error {
	fmt.Println("components:")
	for _, comp := range components {
		fmt.Printf("  - name: %s\n", comp.Name)
		fmt.Printf("    type: %s\n", comp.Type)
		if len(comp.Tags) > 0 {
			fmt.Printf("    tags: [%s]\n", strings.Join(comp.Tags, ", "))
		}
		if showDeps {
			if len(comp.Produces) > 0 {
				fmt.Println("    produces:")
				for _, d := range comp.Produces {
					fmt.Printf("      - key: %s\n        type: %s\n", d.Key, d.Type)
				}
			}
			if len(comp.Requires) > 0 {
				fmt.Println("    requires:")
				for _, d := range comp.Requires {
					fmt.Printf("      - key: %s\n        type: %s\n", d.Key, d.Type)
				}
			}
		}
	}
	return nil
}

func printTypes(registry *discovery.Registry, format string) error {
	var typeNames []string
	for name := range registry.Types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	if len(typeNames) == 0 {
		fmt.Println("No types discovered.")
		return nil
	}

	switch format {
	case "json":
		fmt.Println("[")
		for i, name := range typeNames {
			comma := ","
			if i == len(typeNames)-1 {
				comma = ""
			}
			info := registry.Types[name]
			fmt.Printf(`  {"name": "%s", "package": "%s", "file": "%s"}%s
`, name, info.PackagePath, info.SourceFile, comma)
		}
		fmt.Println("]")
	case "yaml":
		fmt.Println("types:")
		for _, name := range typeNames {
			info := registry.Types[name]
			fmt.Printf("  - name: %s\n    package: %s\n    file: %s\n", name, info.PackagePath, info.SourceFile)
		}
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPACKAGE\tFILE")
		fmt.Fprintln(w, "----\t-------\t----")
		for _, name := range typeNames {
			info := registry.Types[name]
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, info.PackagePath, info.SourceFile)
		}
		w.Flush()
		fmt.Printf("\nFound %d type(s)\n", len(typeNames))
	}

	return nil
}
