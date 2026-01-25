package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Show dependency graph",
	Long: `Display the dependency graph of components.

Supports multiple output formats:
- ascii (default): ASCII art tree representation
- dot: Graphviz DOT format
- mermaid: Mermaid diagram format`,
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().StringP("scenario", "s", "", "show graph for specific scenario")
	graphCmd.Flags().StringP("component", "c", "", "show graph for specific component")
	graphCmd.Flags().StringP("format", "f", "ascii", "output format (ascii, dot, mermaid)")
	graphCmd.Flags().String("depends-on", "", "show components that depend on a key")
	graphCmd.Flags().Bool("show-requires", false, "show only what a component requires")
	graphCmd.Flags().Bool("show-produces", false, "show only what a component produces")
	graphCmd.Flags().Bool("reverse", false, "show reverse dependencies (what depends on this)")
}

func runGraph(cmd *cobra.Command, args []string) error {
	scenarioName, _ := cmd.Flags().GetString("scenario")
	componentName, _ := cmd.Flags().GetString("component")
	format, _ := cmd.Flags().GetString("format")
	dependsOn, _ := cmd.Flags().GetString("depends-on")
	showRequires, _ := cmd.Flags().GetBool("show-requires")
	showProduces, _ := cmd.Flags().GetBool("show-produces")
	reverse, _ := cmd.Flags().GetBool("reverse")
	verbose := viper.GetBool("verbose")

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

	// Handle depends-on query
	if dependsOn != "" {
		return printDependsOn(registry, dependsOn, format)
	}

	// Handle component-specific query
	if componentName != "" {
		comp, ok := registry.Components[core.ComponentID(componentName)]
		if !ok {
			return fmt.Errorf("component '%s' not found", componentName)
		}

		if showRequires {
			return printComponentRequires(comp, format)
		}
		if showProduces {
			return printComponentProduces(comp, format)
		}
		if reverse {
			return printReverseDependencies(registry, comp, format)
		}
		return printComponentGraph(registry, comp, format)
	}

	// Handle scenario-specific query
	if scenarioName != "" {
		for _, s := range cfg.Scenarios {
			if s.Name == scenarioName {
				return printScenarioGraph(registry, &s, format)
			}
		}
		return fmt.Errorf("scenario '%s' not found", scenarioName)
	}

	// Default: show full dependency graph
	return printFullGraph(registry, format)
}

func printDependsOn(registry *discovery.Registry, key string, format string) error {
	var dependents []*core.Component

	for _, comp := range registry.Components {
		for _, req := range comp.Requires {
			if req.Key == key {
				dependents = append(dependents, comp)
				break
			}
		}
	}

	if len(dependents) == 0 {
		fmt.Printf("No components depend on '%s'\n", key)
		return nil
	}

	fmt.Printf("Components that depend on '%s':\n", key)
	for _, comp := range dependents {
		fmt.Printf("  • %s (%s)\n", comp.Name, comp.Type)
	}
	return nil
}

func printComponentRequires(comp *core.Component, format string) error {
	if len(comp.Requires) == 0 {
		fmt.Printf("%s requires nothing\n", comp.Name)
		return nil
	}

	fmt.Printf("%s requires:\n", comp.Name)
	for _, req := range comp.Requires {
		fmt.Printf("  • %s:%s\n", req.Key, req.Type)
	}
	return nil
}

func printComponentProduces(comp *core.Component, format string) error {
	if len(comp.Produces) == 0 {
		fmt.Printf("%s produces nothing\n", comp.Name)
		return nil
	}

	fmt.Printf("%s produces:\n", comp.Name)
	for _, prod := range comp.Produces {
		fmt.Printf("  • %s:%s\n", prod.Key, prod.Type)
	}
	return nil
}

func printReverseDependencies(registry *discovery.Registry, comp *core.Component, format string) error {
	// Find what keys this component produces
	producedKeys := make(map[string]bool)
	for _, prod := range comp.Produces {
		producedKeys[prod.Key] = true
	}

	var dependents []*core.Component
	for _, other := range registry.Components {
		if other.Name == comp.Name {
			continue
		}
		for _, req := range other.Requires {
			if producedKeys[req.Key] {
				dependents = append(dependents, other)
				break
			}
		}
	}

	if len(dependents) == 0 {
		fmt.Printf("No components depend on %s\n", comp.Name)
		return nil
	}

	fmt.Printf("Components that depend on %s:\n", comp.Name)
	for _, dep := range dependents {
		fmt.Printf("  • %s (%s)\n", dep.Name, dep.Type)
	}
	return nil
}

func printComponentGraph(registry *discovery.Registry, comp *core.Component, format string) error {
	switch format {
	case "dot":
		return printComponentDOT(registry, comp)
	case "mermaid":
		return printComponentMermaid(registry, comp)
	default:
		return printComponentASCII(registry, comp)
	}
}

func printComponentASCII(registry *discovery.Registry, comp *core.Component) error {
	fmt.Printf("%s [%s]\n", comp.Name, comp.Type)

	if len(comp.Produces) > 0 {
		fmt.Println("├── produces:")
		for i, prod := range comp.Produces {
			prefix := "│   ├── "
			if i == len(comp.Produces)-1 && len(comp.Requires) == 0 {
				prefix = "    └── "
			} else if i == len(comp.Produces)-1 {
				prefix = "│   └── "
			}
			fmt.Printf("%s%s:%s\n", prefix, prod.Key, prod.Type)
		}
	}

	if len(comp.Requires) > 0 {
		fmt.Println("└── requires:")
		for i, req := range comp.Requires {
			prefix := "    ├── "
			if i == len(comp.Requires)-1 {
				prefix = "    └── "
			}
			// Find provider
			provider := findProvider(registry, req.Key)
			if provider != "" {
				fmt.Printf("%s%s:%s ← %s\n", prefix, req.Key, req.Type, provider)
			} else {
				fmt.Printf("%s%s:%s (unresolved)\n", prefix, req.Key, req.Type)
			}
		}
	}

	return nil
}

func findProvider(registry *discovery.Registry, key string) string {
	for _, comp := range registry.Components {
		for _, prod := range comp.Produces {
			if prod.Key == key {
				return comp.Name
			}
		}
	}
	return ""
}

func printScenarioGraph(registry *discovery.Registry, scenario *config.ScenarioConfig, format string) error {
	switch format {
	case "dot":
		return printScenarioDOT(registry, scenario)
	case "mermaid":
		return printScenarioMermaid(registry, scenario)
	default:
		return printScenarioASCII(registry, scenario)
	}
}

func printScenarioASCII(registry *discovery.Registry, scenario *config.ScenarioConfig) error {
	fmt.Printf("%s\n", scenario.Name)

	for i, item := range scenario.Flow {
		compName := item.GetComponentName()
		comp, ok := registry.Components[core.ComponentID(compName)]

		prefix := "├── "
		childPrefix := "│   "
		if i == len(scenario.Flow)-1 {
			prefix = "└── "
			childPrefix = "    "
		}

		if ok {
			fmt.Printf("%s%s [%s]\n", prefix, comp.Name, comp.Type)
			if len(comp.Produces) > 0 {
				for _, prod := range comp.Produces {
					fmt.Printf("%s└── produces: %s:%s\n", childPrefix, prod.Key, prod.Type)
				}
			}
			if len(comp.Requires) > 0 {
				for _, req := range comp.Requires {
					provider := findProvider(registry, req.Key)
					if provider != "" {
						fmt.Printf("%s└── requires: %s:%s ← %s\n", childPrefix, req.Key, req.Type, provider)
					}
				}
			}
		} else {
			fmt.Printf("%s%s [unknown]\n", prefix, compName)
		}
	}

	return nil
}

func printFullGraph(registry *discovery.Registry, format string) error {
	switch format {
	case "dot":
		return printFullDOT(registry)
	case "mermaid":
		return printFullMermaid(registry)
	default:
		return printFullASCII(registry)
	}
}

func printFullASCII(registry *discovery.Registry) error {
	// Group by type
	byType := make(map[core.ComponentType][]*core.Component)
	for _, comp := range registry.Components {
		byType[comp.Type] = append(byType[comp.Type], comp)
	}

	// Sort types for consistent output
	types := []core.ComponentType{
		core.ComponentSetup,
		core.ComponentTask,
		core.ComponentValidation,
		core.ComponentStep,
		core.ComponentRollup,
		core.ComponentTeardown,
	}

	for _, t := range types {
		comps := byType[t]
		if len(comps) == 0 {
			continue
		}

		// Sort by name
		sort.Slice(comps, func(i, j int) bool {
			return comps[i].Name < comps[j].Name
		})

		fmt.Printf("\n%s:\n", strings.ToUpper(string(t)))
		for _, comp := range comps {
			fmt.Printf("  %s\n", comp.Name)
			if len(comp.Produces) > 0 {
				for _, prod := range comp.Produces {
					fmt.Printf("    └── produces: %s:%s\n", prod.Key, prod.Type)
				}
			}
			if len(comp.Requires) > 0 {
				for _, req := range comp.Requires {
					provider := findProvider(registry, req.Key)
					if provider != "" {
						fmt.Printf("    └── requires: %s:%s ← %s\n", req.Key, req.Type, provider)
					} else {
						fmt.Printf("    └── requires: %s:%s (unresolved)\n", req.Key, req.Type)
					}
				}
			}
		}
	}

	return nil
}

func printFullDOT(registry *discovery.Registry) error {
	fmt.Println("digraph chronicle {")
	fmt.Println("  rankdir=TB;")
	fmt.Println("  node [shape=box];")
	fmt.Println()

	// Define nodes with colors by type
	typeColors := map[core.ComponentType]string{
		core.ComponentSetup:      "lightblue",
		core.ComponentTask:       "lightgreen",
		core.ComponentValidation: "lightyellow",
		core.ComponentStep:       "lightgray",
		core.ComponentRollup:     "lightpink",
		core.ComponentTeardown:   "lightsalmon",
	}

	for _, comp := range registry.Components {
		color := typeColors[comp.Type]
		if color == "" {
			color = "white"
		}
		fmt.Printf("  \"%s\" [label=\"%s\\n[%s]\", fillcolor=%s, style=filled];\n",
			comp.Name, comp.Name, comp.Type, color)
	}

	fmt.Println()

	// Define edges based on dependencies
	for _, comp := range registry.Components {
		for _, req := range comp.Requires {
			provider := findProvider(registry, req.Key)
			if provider != "" {
				fmt.Printf("  \"%s\" -> \"%s\" [label=\"%s\"];\n", provider, comp.Name, req.Key)
			}
		}
	}

	fmt.Println("}")
	return nil
}

func printFullMermaid(registry *discovery.Registry) error {
	fmt.Println("graph TD")
	fmt.Println()

	// Define nodes
	for _, comp := range registry.Components {
		style := getMermaidStyle(comp.Type)
		fmt.Printf("  %s[\"%s<br>[%s]\"]%s\n", sanitizeID(comp.Name), comp.Name, comp.Type, style)
	}

	fmt.Println()

	// Define edges
	for _, comp := range registry.Components {
		for _, req := range comp.Requires {
			provider := findProvider(registry, req.Key)
			if provider != "" {
				fmt.Printf("  %s -->|%s| %s\n", sanitizeID(provider), req.Key, sanitizeID(comp.Name))
			}
		}
	}

	return nil
}

func printComponentDOT(registry *discovery.Registry, comp *core.Component) error {
	fmt.Println("digraph component {")
	fmt.Println("  rankdir=LR;")
	fmt.Printf("  \"%s\" [shape=box, style=filled, fillcolor=lightblue];\n", comp.Name)

	for _, prod := range comp.Produces {
		key := fmt.Sprintf("%s:%s", prod.Key, prod.Type)
		fmt.Printf("  \"%s\" [shape=ellipse, style=filled, fillcolor=lightgreen];\n", key)
		fmt.Printf("  \"%s\" -> \"%s\" [label=\"produces\"];\n", comp.Name, key)
	}

	for _, req := range comp.Requires {
		key := fmt.Sprintf("%s:%s", req.Key, req.Type)
		fmt.Printf("  \"%s\" [shape=ellipse, style=filled, fillcolor=lightyellow];\n", key)
		fmt.Printf("  \"%s\" -> \"%s\" [label=\"requires\"];\n", key, comp.Name)
	}

	fmt.Println("}")
	return nil
}

func printComponentMermaid(registry *discovery.Registry, comp *core.Component) error {
	fmt.Println("graph LR")
	fmt.Printf("  %s[\"%s\"]:::component\n", sanitizeID(comp.Name), comp.Name)

	for _, prod := range comp.Produces {
		key := sanitizeID(prod.Key + "_" + prod.Type)
		fmt.Printf("  %s((\"%s:%s\")):::produced\n", key, prod.Key, prod.Type)
		fmt.Printf("  %s -->|produces| %s\n", sanitizeID(comp.Name), key)
	}

	for _, req := range comp.Requires {
		key := sanitizeID(req.Key + "_" + req.Type + "_req")
		fmt.Printf("  %s((\"%s:%s\")):::required\n", key, req.Key, req.Type)
		fmt.Printf("  %s -->|requires| %s\n", key, sanitizeID(comp.Name))
	}

	fmt.Println()
	fmt.Println("  classDef component fill:#6baed6,stroke:#333")
	fmt.Println("  classDef produced fill:#74c476,stroke:#333")
	fmt.Println("  classDef required fill:#fd8d3c,stroke:#333")

	return nil
}

func printScenarioDOT(registry *discovery.Registry, scenario *config.ScenarioConfig) error {
	fmt.Println("digraph scenario {")
	fmt.Println("  rankdir=TB;")
	fmt.Printf("  label=\"%s\";\n", scenario.Name)
	fmt.Println()

	var prevName string
	for _, item := range scenario.Flow {
		compName := item.GetComponentName()
		comp, ok := registry.Components[core.ComponentID(compName)]

		if ok {
			fmt.Printf("  \"%s\" [label=\"%s\\n[%s]\"];\n", comp.Name, comp.Name, comp.Type)
			if prevName != "" {
				fmt.Printf("  \"%s\" -> \"%s\";\n", prevName, comp.Name)
			}
			prevName = comp.Name
		}
	}

	fmt.Println("}")
	return nil
}

func printScenarioMermaid(registry *discovery.Registry, scenario *config.ScenarioConfig) error {
	fmt.Println("graph TD")
	fmt.Printf("  subgraph %s\n", sanitizeID(scenario.Name))

	var prevName string
	for _, item := range scenario.Flow {
		compName := item.GetComponentName()
		comp, ok := registry.Components[core.ComponentID(compName)]

		if ok {
			style := getMermaidStyle(comp.Type)
			fmt.Printf("    %s[\"%s\"]%s\n", sanitizeID(comp.Name), comp.Name, style)
			if prevName != "" {
				fmt.Printf("    %s --> %s\n", sanitizeID(prevName), sanitizeID(comp.Name))
			}
			prevName = comp.Name
		}
	}

	fmt.Println("  end")
	return nil
}

func sanitizeID(s string) string {
	// Replace characters that aren't valid in Mermaid IDs
	r := strings.NewReplacer("-", "_", " ", "_", ":", "_")
	return r.Replace(s)
}

func getMermaidStyle(t core.ComponentType) string {
	switch t {
	case core.ComponentSetup:
		return ":::setup"
	case core.ComponentTask:
		return ":::task"
	case core.ComponentValidation:
		return ":::validation"
	case core.ComponentTeardown:
		return ":::teardown"
	default:
		return ""
	}
}
