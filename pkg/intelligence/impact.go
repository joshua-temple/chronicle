package intelligence

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ImpactLevel represents the impact level of a change.
type ImpactLevel int

const (
	ImpactLevelNone ImpactLevel = iota
	ImpactLevelLow
	ImpactLevelMedium
	ImpactLevelHigh
	ImpactLevelCritical
)

// String returns the string representation of the impact level.
func (l ImpactLevel) String() string {
	switch l {
	case ImpactLevelNone:
		return "none"
	case ImpactLevelLow:
		return "low"
	case ImpactLevelMedium:
		return "medium"
	case ImpactLevelHigh:
		return "high"
	case ImpactLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// FileChange represents a change to a file.
type FileChange struct {
	Path       string   `json:"path"`
	ChangeType string   `json:"change_type"` // added, modified, deleted, renamed
	LinesAdded int      `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	Functions  []string `json:"functions,omitempty"`
}

// TestMapping maps tests to the code they test.
type TestMapping struct {
	ScenarioName  string   `json:"scenario_name"`
	ComponentName string   `json:"component_name,omitempty"`
	TestedFiles   []string `json:"tested_files"`
	TestedFuncs   []string `json:"tested_functions"`
	Tags          []string `json:"tags,omitempty"`
	Priority      int      `json:"priority"`
}

// ImpactResult represents the result of impact analysis.
type ImpactResult struct {
	Changes         []FileChange    `json:"changes"`
	AffectedTests   []TestMapping   `json:"affected_tests"`
	ImpactLevel     ImpactLevel     `json:"impact_level"`
	RunRecommendation string        `json:"run_recommendation"`
	Confidence      float64         `json:"confidence"`
	AnalyzedAt      time.Time       `json:"analyzed_at"`
}

// ImpactReport provides a summary of test impact analysis.
type ImpactReport struct {
	GeneratedAt       time.Time       `json:"generated_at"`
	BaseRef           string          `json:"base_ref"`
	HeadRef           string          `json:"head_ref"`
	TotalChanges      int             `json:"total_changes"`
	FilesChanged      int             `json:"files_changed"`
	AffectedTests     int             `json:"affected_tests"`
	ImpactLevel       ImpactLevel     `json:"impact_level"`
	Results           []ImpactResult  `json:"results"`
	SuggestedTests    []string        `json:"suggested_tests"`
	SkippableTests    []string        `json:"skippable_tests"`
	Recommendations   []string        `json:"recommendations"`
}

// ImpactAnalyzerConfig configures the impact analyzer.
type ImpactAnalyzerConfig struct {
	// RootPath is the root path of the codebase.
	RootPath string `json:"root_path"`

	// TestMappingsPath is where test mappings are stored.
	TestMappingsPath string `json:"test_mappings_path"`

	// IgnorePatterns are file patterns to ignore.
	IgnorePatterns []string `json:"ignore_patterns"`

	// CriticalPaths are paths that always trigger full test runs.
	CriticalPaths []string `json:"critical_paths"`

	// DefaultImpactLevel is the default impact level for unmapped files.
	DefaultImpactLevel ImpactLevel `json:"default_impact_level"`
}

// DefaultImpactAnalyzerConfig returns the default configuration.
func DefaultImpactAnalyzerConfig() ImpactAnalyzerConfig {
	return ImpactAnalyzerConfig{
		RootPath:         ".",
		TestMappingsPath: ".chronicle/impact",
		IgnorePatterns: []string{
			"*.md",
			"*.txt",
			".git/*",
			"vendor/*",
			"node_modules/*",
		},
		CriticalPaths: []string{
			"go.mod",
			"go.sum",
			"Makefile",
			"Dockerfile",
			".chronicle.yaml",
		},
		DefaultImpactLevel: ImpactLevelMedium,
	}
}

// ImpactAnalyzer analyzes the impact of code changes on tests.
type ImpactAnalyzer struct {
	config   ImpactAnalyzerConfig
	mappings map[string]*TestMapping
	mu       sync.RWMutex
}

// NewImpactAnalyzer creates a new impact analyzer.
func NewImpactAnalyzer(config ImpactAnalyzerConfig) *ImpactAnalyzer {
	ia := &ImpactAnalyzer{
		config:   config,
		mappings: make(map[string]*TestMapping),
	}

	// Load existing mappings
	_ = ia.loadMappings()

	return ia
}

// RegisterMapping registers a test-to-code mapping.
func (ia *ImpactAnalyzer) RegisterMapping(mapping *TestMapping) {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	key := mapping.ScenarioName
	if mapping.ComponentName != "" {
		key = fmt.Sprintf("%s.%s", mapping.ScenarioName, mapping.ComponentName)
	}

	ia.mappings[key] = mapping
	_ = ia.saveMappings()
}

// AnalyzeGitDiff analyzes the impact of changes between two git refs.
func (ia *ImpactAnalyzer) AnalyzeGitDiff(ctx context.Context, baseRef, headRef string) (*ImpactReport, error) {
	// Get changed files from git
	changes, err := ia.getGitDiff(ctx, baseRef, headRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get git diff: %w", err)
	}

	return ia.AnalyzeChanges(ctx, changes, baseRef, headRef)
}

// AnalyzeUncommitted analyzes the impact of uncommitted changes.
func (ia *ImpactAnalyzer) AnalyzeUncommitted(ctx context.Context) (*ImpactReport, error) {
	changes, err := ia.getUncommittedChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get uncommitted changes: %w", err)
	}

	return ia.AnalyzeChanges(ctx, changes, "HEAD", "working-tree")
}

// AnalyzeChanges analyzes the impact of a set of file changes.
func (ia *ImpactAnalyzer) AnalyzeChanges(ctx context.Context, changes []FileChange, baseRef, headRef string) (*ImpactReport, error) {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	report := &ImpactReport{
		GeneratedAt:    time.Now(),
		BaseRef:        baseRef,
		HeadRef:        headRef,
		TotalChanges:   0,
		FilesChanged:   len(changes),
		Results:        make([]ImpactResult, 0),
		SuggestedTests: make([]string, 0),
		SkippableTests: make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Check for critical path changes
	criticalChange := false
	for _, change := range changes {
		report.TotalChanges += change.LinesAdded + change.LinesRemoved

		for _, critPath := range ia.config.CriticalPaths {
			if matchPattern(change.Path, critPath) {
				criticalChange = true
				break
			}
		}
	}

	if criticalChange {
		report.ImpactLevel = ImpactLevelCritical
		report.Recommendations = append(report.Recommendations,
			"Critical configuration file changed - recommend running full test suite")

		// Suggest all tests
		for _, mapping := range ia.mappings {
			report.SuggestedTests = append(report.SuggestedTests, mapping.ScenarioName)
		}
		report.AffectedTests = len(report.SuggestedTests)
		return report, nil
	}

	// Analyze each change
	affectedMap := make(map[string]bool)
	maxImpact := ImpactLevelNone

	for _, change := range changes {
		// Skip ignored patterns
		if ia.shouldIgnore(change.Path) {
			continue
		}

		result := ia.analyzeFileChange(change)
		report.Results = append(report.Results, result)

		if result.ImpactLevel > maxImpact {
			maxImpact = result.ImpactLevel
		}

		for _, test := range result.AffectedTests {
			affectedMap[test.ScenarioName] = true
		}
	}

	report.ImpactLevel = maxImpact
	report.AffectedTests = len(affectedMap)

	// Build suggested tests list
	for testName := range affectedMap {
		report.SuggestedTests = append(report.SuggestedTests, testName)
	}

	// Build skippable tests list (tests not affected)
	for _, mapping := range ia.mappings {
		if !affectedMap[mapping.ScenarioName] {
			report.SkippableTests = append(report.SkippableTests, mapping.ScenarioName)
		}
	}

	// Generate recommendations
	switch report.ImpactLevel {
	case ImpactLevelHigh:
		report.Recommendations = append(report.Recommendations,
			"High impact changes detected - recommend running affected tests before merge")
	case ImpactLevelMedium:
		report.Recommendations = append(report.Recommendations,
			"Medium impact changes detected - consider running affected tests")
	case ImpactLevelLow:
		report.Recommendations = append(report.Recommendations,
			"Low impact changes - minimal testing required")
	}

	if len(report.SkippableTests) > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("%d tests can be safely skipped based on change analysis", len(report.SkippableTests)))
	}

	return report, nil
}

// analyzeFileChange analyzes the impact of a single file change.
func (ia *ImpactAnalyzer) analyzeFileChange(change FileChange) ImpactResult {
	result := ImpactResult{
		Changes:       []FileChange{change},
		AffectedTests: make([]TestMapping, 0),
		Confidence:    0.8, // Default confidence
		AnalyzedAt:    time.Now(),
	}

	// Find tests that map to this file
	for _, mapping := range ia.mappings {
		for _, testedFile := range mapping.TestedFiles {
			if matchPattern(change.Path, testedFile) {
				result.AffectedTests = append(result.AffectedTests, *mapping)
				break
			}
		}

		// Also check function-level mapping
		for _, testedFunc := range mapping.TestedFuncs {
			for _, changedFunc := range change.Functions {
				if testedFunc == changedFunc {
					// Avoid duplicates
					found := false
					for _, at := range result.AffectedTests {
						if at.ScenarioName == mapping.ScenarioName {
							found = true
							break
						}
					}
					if !found {
						result.AffectedTests = append(result.AffectedTests, *mapping)
					}
				}
			}
		}
	}

	// Determine impact level based on change size and affected tests
	linesChanged := change.LinesAdded + change.LinesRemoved
	numAffected := len(result.AffectedTests)

	switch {
	case numAffected >= 10 || linesChanged >= 500:
		result.ImpactLevel = ImpactLevelHigh
		result.RunRecommendation = "Run all affected tests"
	case numAffected >= 5 || linesChanged >= 100:
		result.ImpactLevel = ImpactLevelMedium
		result.RunRecommendation = "Run affected tests"
	case numAffected >= 1 || linesChanged >= 20:
		result.ImpactLevel = ImpactLevelLow
		result.RunRecommendation = "Run affected tests if time permits"
	default:
		result.ImpactLevel = ImpactLevelNone
		result.RunRecommendation = "No testing required"
	}

	// If no mappings found, use default impact level
	if numAffected == 0 && linesChanged > 0 {
		result.ImpactLevel = ia.config.DefaultImpactLevel
		result.Confidence = 0.5
		result.RunRecommendation = "No explicit mapping - recommend running related tests"
	}

	return result
}

// getGitDiff gets changed files between two git refs.
func (ia *ImpactAnalyzer) getGitDiff(ctx context.Context, baseRef, headRef string) ([]FileChange, error) {
	// Get list of changed files
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-status", baseRef, headRef)
	cmd.Dir = ia.config.RootPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return ia.parseGitDiff(ctx, string(output), baseRef, headRef)
}

// getUncommittedChanges gets uncommitted changes.
func (ia *ImpactAnalyzer) getUncommittedChanges(ctx context.Context) ([]FileChange, error) {
	// Get staged and unstaged changes
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-status", "HEAD")
	cmd.Dir = ia.config.RootPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return ia.parseGitDiff(ctx, string(output), "HEAD", "working-tree")
}

// parseGitDiff parses git diff output into FileChange structs.
func (ia *ImpactAnalyzer) parseGitDiff(ctx context.Context, output, baseRef, headRef string) ([]FileChange, error) {
	var changes []FileChange

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		changeType := parts[0]
		path := parts[len(parts)-1]

		change := FileChange{
			Path: path,
		}

		switch changeType[0] {
		case 'A':
			change.ChangeType = "added"
		case 'M':
			change.ChangeType = "modified"
		case 'D':
			change.ChangeType = "deleted"
		case 'R':
			change.ChangeType = "renamed"
		default:
			change.ChangeType = "unknown"
		}

		// Get line counts for modified files
		if change.ChangeType == "modified" || change.ChangeType == "added" {
			numCmd := exec.CommandContext(ctx, "git", "diff", "--numstat", baseRef, headRef, "--", path)
			numCmd.Dir = ia.config.RootPath
			numOutput, err := numCmd.Output()
			if err == nil {
				numParts := strings.Fields(string(numOutput))
				if len(numParts) >= 2 {
					_, _ = fmt.Sscanf(numParts[0], "%d", &change.LinesAdded)
					_, _ = fmt.Sscanf(numParts[1], "%d", &change.LinesRemoved)
				}
			}

			// Extract changed functions for Go files
			if strings.HasSuffix(path, ".go") {
				change.Functions = ia.extractChangedFunctions(ctx, path, baseRef, headRef)
			}
		}

		changes = append(changes, change)
	}

	return changes, scanner.Err()
}

// extractChangedFunctions extracts function names from changed Go code.
func (ia *ImpactAnalyzer) extractChangedFunctions(ctx context.Context, path, baseRef, headRef string) []string {
	var functions []string

	fullPath := filepath.Join(ia.config.RootPath, path)

	// Parse the Go file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fullPath, nil, 0)
	if err != nil {
		return functions
	}

	// Extract all function names
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			functions = append(functions, fn.Name.Name)
		}
	}

	return functions
}

// shouldIgnore checks if a file should be ignored.
func (ia *ImpactAnalyzer) shouldIgnore(path string) bool {
	for _, pattern := range ia.config.IgnorePatterns {
		if matchPattern(path, pattern) {
			return true
		}
	}
	return false
}

// loadMappings loads test mappings from storage.
func (ia *ImpactAnalyzer) loadMappings() error {
	path := filepath.Join(ia.config.TestMappingsPath, "mappings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &ia.mappings)
}

// saveMappings saves test mappings to storage.
func (ia *ImpactAnalyzer) saveMappings() error {
	if err := os.MkdirAll(ia.config.TestMappingsPath, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ia.mappings, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(ia.config.TestMappingsPath, "mappings.json")
	return os.WriteFile(path, data, 0644)
}

// matchPattern matches a path against a glob-like pattern.
func matchPattern(path, pattern string) bool {
	// Simple matching - could be enhanced with proper glob support
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/")
	}
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(path, suffix)
	}
	return path == pattern || strings.Contains(path, pattern)
}

// AutoMapFromAnnotations automatically creates test mappings from Chronicle annotations.
func (ia *ImpactAnalyzer) AutoMapFromAnnotations(ctx context.Context, componentPath string) error {
	// Walk the component path and find files with @chronicle annotations
	return filepath.Walk(componentPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse the file
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		// Look for @chronicle annotations
		for _, comment := range file.Comments {
			for _, c := range comment.List {
				text := c.Text
				if strings.Contains(text, "@chronicle:") {
					// Extract scenario/component info and create mapping
					relPath, _ := filepath.Rel(ia.config.RootPath, path)

					// Find the associated function
					for _, decl := range file.Decls {
						if fn, ok := decl.(*ast.FuncDecl); ok {
							pos := fset.Position(fn.Pos())
							commentPos := fset.Position(c.Pos())

							// Check if this comment is near this function
							if pos.Line-commentPos.Line < 5 {
								mapping := &TestMapping{
									ScenarioName:  fn.Name.Name,
									ComponentName: fn.Name.Name,
									TestedFiles:   []string{relPath},
									TestedFuncs:   []string{fn.Name.Name},
									Priority:      1,
								}
								ia.RegisterMapping(mapping)
							}
						}
					}
				}
			}
		}

		return nil
	})
}
