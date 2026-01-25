package discovery

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// AnnotationPrefix is the prefix for Chronicle annotations.
const AnnotationPrefix = "@chronicle:"

// Annotation types.
const (
	AnnotationType        = "type"
	AnnotationSetup       = "setup"
	AnnotationTask        = "task"
	AnnotationValidation  = "validation"
	AnnotationStep        = "step"
	AnnotationRollup      = "rollup"
	AnnotationTeardown    = "teardown"
	AnnotationMiddleware  = "middleware"
	AnnotationDescription = "description"
	AnnotationTags        = "tags"
	AnnotationOwner       = "owner"
	AnnotationVersion     = "version"
	AnnotationDeprecated  = "deprecated"
	AnnotationRequires    = "requires"
	AnnotationProduces    = "produces"
	AnnotationExample     = "example"
	AnnotationSince       = "since"
)

// Parser discovers components from Go source files using AST parsing.
type Parser struct {
	paths       []string
	fileSet     *token.FileSet
	annotations map[string][]Annotation
}

// Annotation represents a parsed annotation.
type Annotation struct {
	Type       string            // type, setup, task, etc.
	Attributes map[string]string // key="value" pairs
	Value      string            // Value for simple annotations (e.g., description "text")
	File       string            // Source file
	Line       int               // Line number
}

// NewParser creates a new Parser for the given paths.
func NewParser(paths ...string) *Parser {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	return &Parser{
		paths:       paths,
		fileSet:     token.NewFileSet(),
		annotations: make(map[string][]Annotation),
	}
}

// Discover scans the configured paths and builds a Registry.
func (p *Parser) Discover() (*Registry, error) {
	registry := NewRegistry()

	for _, path := range p.paths {
		if err := p.scanPath(path, registry); err != nil {
			return nil, fmt.Errorf("scanning path %s: %w", path, err)
		}
	}

	return registry, nil
}

func (p *Parser) scanPath(path string, registry *Registry) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return p.scanDirectory(path, registry)
	}
	return p.scanFile(path, registry)
}

func (p *Parser) scanDirectory(dir string, registry *Registry) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and vendor
		if info.IsDir() {
			name := info.Name()
			// Don't skip the root directory "." or "./"
			if name == "." {
				return nil
			}
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files for component discovery (they can have their own tests)
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return p.scanFile(path, registry)
	})
}

func (p *Parser) scanFile(path string, registry *Registry) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	file, err := parser.ParseFile(p.fileSet, path, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	// Process all declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			p.processGenDecl(d, path, registry)
		case *ast.FuncDecl:
			p.processFuncDecl(d, path, registry)
		}
	}

	return nil
}

func (p *Parser) processGenDecl(decl *ast.GenDecl, path string, registry *Registry) {
	if decl.Doc == nil {
		return
	}

	annotations := p.parseCommentGroup(decl.Doc, path)
	if len(annotations) == 0 {
		return
	}

	// Handle type declarations
	for _, spec := range decl.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			for _, ann := range annotations {
				if ann.Type == AnnotationType {
					typeInfo := &core.TypeInfo{
						Name:       ts.Name.Name,
						SourceFile: path,
						SourceLine: p.fileSet.Position(ts.Pos()).Line,
					}
					if alias := ann.Attributes["alias"]; alias != "" {
						typeInfo.IsAlias = true
						typeInfo.AliasOf = alias
					}
					registry.Types[ts.Name.Name] = typeInfo
				}
			}
		}
	}
}

func (p *Parser) processFuncDecl(decl *ast.FuncDecl, path string, registry *Registry) {
	if decl.Doc == nil {
		return
	}

	annotations := p.parseCommentGroup(decl.Doc, path)
	if len(annotations) == 0 {
		return
	}

	// Find the primary component annotation
	var component *core.Component
	for _, ann := range annotations {
		switch ann.Type {
		case AnnotationSetup:
			component = p.createComponent(ann, core.ComponentSetup, decl.Name.Name, path)
		case AnnotationTask:
			component = p.createComponent(ann, core.ComponentTask, decl.Name.Name, path)
		case AnnotationValidation:
			component = p.createComponent(ann, core.ComponentValidation, decl.Name.Name, path)
		case AnnotationStep:
			component = p.createComponent(ann, core.ComponentStep, decl.Name.Name, path)
		case AnnotationRollup:
			component = p.createComponent(ann, core.ComponentRollup, decl.Name.Name, path)
		case AnnotationTeardown:
			component = p.createComponent(ann, core.ComponentTeardown, decl.Name.Name, path)
		case AnnotationMiddleware:
			mi := &MiddlewareInfo{
				Name:       ann.Attributes["name"],
				SourceFile: path,
				SourceLine: p.fileSet.Position(decl.Pos()).Line,
			}
			if mi.Name == "" {
				mi.Name = decl.Name.Name
			}
			registry.Middleware[mi.Name] = mi
		}
	}

	if component == nil {
		return
	}

	// Apply additional annotations to the component
	for _, ann := range annotations {
		switch ann.Type {
		case AnnotationDescription:
			if component.Description == "" {
				component.Description = ann.Value
			} else {
				component.Description += "\n" + ann.Value
			}
		case AnnotationTags:
			tags := parseTags(ann.Value)
			component.Tags = append(component.Tags, tags...)
		case AnnotationOwner:
			component.Owner = ann.Value
		case AnnotationVersion:
			component.Version = ann.Value
		case AnnotationDeprecated:
			component.Deprecated = ann.Value
			if sunset := ann.Attributes["sunset"]; sunset != "" {
				if t, err := time.Parse("2006-01-02", sunset); err == nil {
					component.Sunset = t
				}
			}
		case AnnotationRequires:
			// Additional requires (beyond the main annotation)
			deps := parseDependencies(ann.Value)
			component.Requires = append(component.Requires, deps...)
		case AnnotationProduces:
			// Additional produces (beyond the main annotation)
			deps := parseDependencies(ann.Value)
			component.Produces = append(component.Produces, deps...)
		}
	}

	component.SourceLine = p.fileSet.Position(decl.Pos()).Line
	registry.Components[component.ID] = component
}

func (p *Parser) createComponent(ann Annotation, componentType core.ComponentType, funcName, path string) *core.Component {
	name := ann.Attributes["name"]
	if name == "" {
		name = funcName
	}

	component := core.NewComponent(name, componentType)
	component.SourceFile = path

	// Parse produces
	if produces := ann.Attributes["produces"]; produces != "" {
		component.Produces = parseDependencies(produces)
	}

	// Parse requires
	if requires := ann.Attributes["requires"]; requires != "" {
		component.Requires = parseDependencies(requires)
	}

	// Parse teardown pairing
	if teardown := ann.Attributes["teardown"]; teardown != "" {
		component.Teardown = teardown
	}

	// Parse version
	if version := ann.Attributes["version"]; version != "" {
		component.Version = version
	}

	return component
}

func (p *Parser) parseCommentGroup(cg *ast.CommentGroup, path string) []Annotation {
	var annotations []Annotation

	for _, comment := range cg.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		if !strings.HasPrefix(text, AnnotationPrefix) {
			continue
		}

		ann := p.parseAnnotation(text, path, p.fileSet.Position(comment.Pos()).Line)
		if ann != nil {
			annotations = append(annotations, *ann)
		}
	}

	return annotations
}

// Regex patterns for annotation parsing.
var (
	annotationTypeRe = regexp.MustCompile(`^@chronicle:(\w+)`)
	attributeRe      = regexp.MustCompile(`(\w+)="([^"]*)"`)
	simpleValueRe    = regexp.MustCompile(`^@chronicle:\w+\s+"([^"]+)"`)
	tagListRe        = regexp.MustCompile(`^@chronicle:\w+\s+(.+)$`)
)

func (p *Parser) parseAnnotation(text, path string, line int) *Annotation {
	// Extract annotation type
	match := annotationTypeRe.FindStringSubmatch(text)
	if match == nil {
		return nil
	}

	ann := &Annotation{
		Type:       match[1],
		Attributes: make(map[string]string),
		File:       path,
		Line:       line,
	}

	// For simple value annotations (description, tags, owner, version)
	switch ann.Type {
	case AnnotationDescription, AnnotationExample:
		if m := simpleValueRe.FindStringSubmatch(text); m != nil {
			ann.Value = m[1]
			return ann
		}
	case AnnotationTags, AnnotationOwner, AnnotationVersion, AnnotationSince:
		// Extract everything after the annotation type
		rest := strings.TrimPrefix(text, AnnotationPrefix+ann.Type)
		rest = strings.TrimSpace(rest)
		// Remove quotes if present
		rest = strings.Trim(rest, `"`)
		ann.Value = rest
		return ann
	case AnnotationDeprecated:
		// Extract the quoted message and any attributes
		if m := simpleValueRe.FindStringSubmatch(text); m != nil {
			ann.Value = m[1]
		}
		// Also parse attributes like sunset="date"
		matches := attributeRe.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			ann.Attributes[m[1]] = m[2]
		}
		return ann
	}

	// Parse key="value" attributes
	matches := attributeRe.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		ann.Attributes[m[1]] = m[2]
	}

	return ann
}

// parseDependencies parses a dependency string like "user:User,cart:Cart".
func parseDependencies(s string) []core.Dependency {
	var deps []core.Dependency
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			deps = append(deps, core.Dependency{
				Key:  strings.TrimSpace(kv[0]),
				Type: strings.TrimSpace(kv[1]),
			})
		} else {
			// Just a key without type
			deps = append(deps, core.Dependency{
				Key: strings.TrimSpace(kv[0]),
			})
		}
	}
	return deps
}

// parseTags parses a comma-separated tag list.
func parseTags(s string) []string {
	var tags []string
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}
