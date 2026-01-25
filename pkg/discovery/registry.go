package discovery

import (
	"fmt"
	"sort"
	"strings"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// Registry holds all discovered components, types, and middleware.
type Registry struct {
	Components map[core.ComponentID]*core.Component
	Types      map[string]*core.TypeInfo
	Middleware map[string]*MiddlewareInfo
}

// MiddlewareInfo represents information about discovered middleware.
type MiddlewareInfo struct {
	Name        string
	Description string
	SourceFile  string
	SourceLine  int
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		Components: make(map[core.ComponentID]*core.Component),
		Types:      make(map[string]*core.TypeInfo),
		Middleware: make(map[string]*MiddlewareInfo),
	}
}

// GetComponent retrieves a component by ID.
func (r *Registry) GetComponent(id core.ComponentID) (*core.Component, bool) {
	c, ok := r.Components[id]
	return c, ok
}

// GetComponentByName retrieves a component by name.
func (r *Registry) GetComponentByName(name string) (*core.Component, bool) {
	return r.GetComponent(core.ComponentID(name))
}

// GetType retrieves type info by name.
func (r *Registry) GetType(name string) (*core.TypeInfo, bool) {
	t, ok := r.Types[name]
	return t, ok
}

// GetMiddleware retrieves middleware info by name.
func (r *Registry) GetMiddleware(name string) (*MiddlewareInfo, bool) {
	m, ok := r.Middleware[name]
	return m, ok
}

// ComponentsByType returns all components of a given type.
func (r *Registry) ComponentsByType(ct core.ComponentType) []*core.Component {
	var result []*core.Component
	for _, c := range r.Components {
		if c.Type == ct {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ComponentsByTag returns all components with a given tag.
func (r *Registry) ComponentsByTag(tag string) []*core.Component {
	var result []*core.Component
	for _, c := range r.Components {
		if c.HasTag(tag) {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ValidationError represents a validation error in the registry.
type ValidationError struct {
	Component string
	Field     string
	Message   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s.%s: %s", e.Component, e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no errors"
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// Validate checks the registry for consistency and completeness.
func (r *Registry) Validate() error {
	var errors ValidationErrors

	// Check each component
	for _, c := range r.Components {
		// Validate component itself
		if err := c.Validate(); err != nil {
			errors = append(errors, ValidationError{
				Component: c.Name,
				Field:     "component",
				Message:   err.Error(),
			})
		}

		// Check that all required dependencies can be satisfied
		for _, req := range c.Requires {
			if !r.canSatisfy(req) {
				errors = append(errors, ValidationError{
					Component: c.Name,
					Field:     "requires",
					Message:   fmt.Sprintf("dependency %s:%s cannot be satisfied by any component", req.Key, req.Type),
				})
			}
		}

		// Check that teardown component exists (if specified)
		if c.Teardown != "" {
			if _, ok := r.GetComponentByName(c.Teardown); !ok {
				errors = append(errors, ValidationError{
					Component: c.Name,
					Field:     "teardown",
					Message:   fmt.Sprintf("teardown component %q not found", c.Teardown),
				})
			}
		}

		// Check that types are registered (optional, just warnings)
		// Types may come from external packages, so we don't error on unregistered types
		for _, prod := range c.Produces {
			_ = prod // Explicit: type checking is intentionally a no-op for external types
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// canSatisfy checks if any component can satisfy a dependency.
func (r *Registry) canSatisfy(dep core.Dependency) bool {
	for _, c := range r.Components {
		for _, prod := range c.Produces {
			if prod.Key == dep.Key {
				// If types are specified, they should match
				if dep.Type != "" && prod.Type != "" {
					if dep.Type == prod.Type || strings.TrimPrefix(dep.Type, "*") == strings.TrimPrefix(prod.Type, "*") {
						return true
					}
				} else {
					return true
				}
			}
		}
	}
	return false
}

// Cycle represents a dependency cycle.
type Cycle struct {
	Path []string
}

func (c Cycle) String() string {
	return strings.Join(c.Path, " -> ") + " -> " + c.Path[0]
}

// DetectCycles finds any dependency cycles in the registry.
func (r *Registry) DetectCycles() []Cycle {
	var cycles []Cycle
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, c := range r.Components {
		path := []string{}
		if cycle := r.detectCyclesDFS(c.Name, visited, recStack, path); cycle != nil {
			cycles = append(cycles, Cycle{Path: cycle})
		}
	}

	return cycles
}

func (r *Registry) detectCyclesDFS(name string, visited, recStack map[string]bool, path []string) []string {
	visited[name] = true
	recStack[name] = true
	path = append(path, name)

	c, ok := r.GetComponentByName(name)
	if !ok {
		recStack[name] = false
		return nil
	}

	// Find components that could satisfy our requirements
	for _, req := range c.Requires {
		for _, other := range r.Components {
			if other.ProducesKey(req.Key) {
				if !visited[other.Name] {
					if cycle := r.detectCyclesDFS(other.Name, visited, recStack, path); cycle != nil {
						return cycle
					}
				} else if recStack[other.Name] {
					// Found a cycle
					cycleStart := -1
					for i, p := range path {
						if p == other.Name {
							cycleStart = i
							break
						}
					}
					if cycleStart >= 0 {
						return path[cycleStart:]
					}
				}
			}
		}
	}

	recStack[name] = false
	return nil
}

// Graph represents the dependency graph of components.
type Graph struct {
	Nodes map[string]*GraphNode
	Edges []GraphEdge
}

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	Name      string
	Component *core.Component
}

// GraphEdge represents a dependency edge.
type GraphEdge struct {
	From       string
	To         string
	Dependency string
}

// DependencyGraph builds a dependency graph from the registry.
func (r *Registry) DependencyGraph() *Graph {
	g := &Graph{
		Nodes: make(map[string]*GraphNode),
		Edges: make([]GraphEdge, 0),
	}

	// Add all components as nodes
	for _, c := range r.Components {
		g.Nodes[c.Name] = &GraphNode{
			Name:      c.Name,
			Component: c,
		}
	}

	// Add edges for dependencies
	for _, c := range r.Components {
		for _, req := range c.Requires {
			// Find which component produces this
			for _, other := range r.Components {
				if other.ProducesKey(req.Key) && other.Name != c.Name {
					g.Edges = append(g.Edges, GraphEdge{
						From:       other.Name, // Producer
						To:         c.Name,     // Consumer
						Dependency: req.Key,
					})
				}
			}
		}
	}

	return g
}

// TopologicalSort returns components in execution order.
func (g *Graph) TopologicalSort() ([]string, error) {
	// Count incoming edges for each node
	inDegree := make(map[string]int)
	for name := range g.Nodes {
		inDegree[name] = 0
	}
	for _, edge := range g.Edges {
		inDegree[edge.To]++
	}

	// Start with nodes that have no dependencies
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Deterministic order

	var result []string
	for len(queue) > 0 {
		// Dequeue
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Reduce in-degree for dependent nodes
		for _, edge := range g.Edges {
			if edge.From == node {
				inDegree[edge.To]--
				if inDegree[edge.To] == 0 {
					queue = append(queue, edge.To)
					sort.Strings(queue)
				}
			}
		}
	}

	if len(result) != len(g.Nodes) {
		return nil, fmt.Errorf("cycle detected: could not complete topological sort")
	}

	return result, nil
}

// Stats returns statistics about the registry.
func (r *Registry) Stats() map[string]int {
	stats := map[string]int{
		"components": len(r.Components),
		"types":      len(r.Types),
		"middleware": len(r.Middleware),
	}

	for ct := range map[core.ComponentType]bool{
		core.ComponentSetup:      true,
		core.ComponentTask:       true,
		core.ComponentValidation: true,
		core.ComponentStep:       true,
		core.ComponentRollup:     true,
		core.ComponentTeardown:   true,
	} {
		stats[string(ct)+"_count"] = len(r.ComponentsByType(ct))
	}

	return stats
}

// Merge combines another registry into this one.
// Returns an error if there are conflicting definitions.
func (r *Registry) Merge(other *Registry) error {
	for id, c := range other.Components {
		if existing, ok := r.Components[id]; ok {
			return fmt.Errorf("duplicate component: %s (in %s and %s)", c.Name, existing.SourceFile, c.SourceFile)
		}
		r.Components[id] = c
	}

	for name, t := range other.Types {
		if existing, ok := r.Types[name]; ok {
			return fmt.Errorf("duplicate type: %s (in %s and %s)", name, existing.SourceFile, t.SourceFile)
		}
		r.Types[name] = t
	}

	for name, m := range other.Middleware {
		if existing, ok := r.Middleware[name]; ok {
			return fmt.Errorf("duplicate middleware: %s (in %s and %s)", name, existing.SourceFile, m.SourceFile)
		}
		r.Middleware[name] = m
	}

	return nil
}
