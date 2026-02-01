package intelligence

import (
	"context"
	"path/filepath"
	"testing"
)

func TestImpactLevel_String(t *testing.T) {
	tests := []struct {
		level    ImpactLevel
		expected string
	}{
		{ImpactLevelNone, "none"},
		{ImpactLevelLow, "low"},
		{ImpactLevelMedium, "medium"},
		{ImpactLevelHigh, "high"},
		{ImpactLevelCritical, "critical"},
		{ImpactLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("ImpactLevel.String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestNewImpactAnalyzer(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:           tmpDir,
		TestMappingsPath:   filepath.Join(tmpDir, "mappings"),
		IgnorePatterns:     []string{"*.md"},
		CriticalPaths:      []string{"go.mod"},
		DefaultImpactLevel: ImpactLevelMedium,
	}

	ia := NewImpactAnalyzer(cfg)

	if ia == nil {
		t.Fatal("NewImpactAnalyzer() returned nil")
	}

	if ia.mappings == nil {
		t.Error("NewImpactAnalyzer() did not initialize mappings map")
	}
}

func TestImpactAnalyzer_RegisterMapping(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
	}

	ia := NewImpactAnalyzer(cfg)

	mapping := &TestMapping{
		ScenarioName:  "test-scenario",
		ComponentName: "test-component",
		TestedFiles:   []string{"pkg/foo/bar.go"},
		TestedFuncs:   []string{"DoSomething"},
		Priority:      1,
	}

	ia.RegisterMapping(mapping)

	// Verify mapping was registered
	ia.mu.RLock()
	key := "test-scenario.test-component"
	registered, exists := ia.mappings[key]
	ia.mu.RUnlock()

	if !exists {
		t.Fatal("RegisterMapping() did not register mapping")
	}

	if registered.ScenarioName != mapping.ScenarioName {
		t.Errorf("Registered ScenarioName = %q, expected %q", registered.ScenarioName, mapping.ScenarioName)
	}
}

func TestImpactAnalyzer_RegisterMapping_WithoutComponent(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
	}

	ia := NewImpactAnalyzer(cfg)

	mapping := &TestMapping{
		ScenarioName: "test-scenario",
		TestedFiles:  []string{"pkg/foo/bar.go"},
		Priority:     1,
	}

	ia.RegisterMapping(mapping)

	// Verify mapping was registered with just scenario name as key
	ia.mu.RLock()
	_, exists := ia.mappings["test-scenario"]
	ia.mu.RUnlock()

	if !exists {
		t.Error("RegisterMapping() did not register mapping with scenario name key")
	}
}

func TestImpactAnalyzer_AnalyzeChanges_CriticalPath(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
		CriticalPaths:    []string{"go.mod", "Makefile"},
		IgnorePatterns:   []string{"*.md"},
	}

	ia := NewImpactAnalyzer(cfg)

	// Register some test mappings
	ia.RegisterMapping(&TestMapping{
		ScenarioName: "test-1",
		TestedFiles:  []string{"pkg/foo/bar.go"},
	})
	ia.RegisterMapping(&TestMapping{
		ScenarioName: "test-2",
		TestedFiles:  []string{"pkg/baz/qux.go"},
	})

	changes := []FileChange{
		{Path: "go.mod", ChangeType: "modified", LinesAdded: 5, LinesRemoved: 2},
	}

	report, err := ia.AnalyzeChanges(context.Background(), changes, "base", "head")
	if err != nil {
		t.Fatalf("AnalyzeChanges() error: %v", err)
	}

	if report.ImpactLevel != ImpactLevelCritical {
		t.Errorf("ImpactLevel = %v, expected Critical", report.ImpactLevel)
	}

	// Should suggest all tests when critical path changes
	if len(report.SuggestedTests) != 2 {
		t.Errorf("SuggestedTests length = %d, expected 2", len(report.SuggestedTests))
	}

	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations for critical change")
	}
}

func TestImpactAnalyzer_AnalyzeChanges_AffectedTests(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:           tmpDir,
		TestMappingsPath:   filepath.Join(tmpDir, "mappings"),
		CriticalPaths:      []string{},
		IgnorePatterns:     []string{"*.md"},
		DefaultImpactLevel: ImpactLevelMedium,
	}

	ia := NewImpactAnalyzer(cfg)

	// Register test mappings
	ia.RegisterMapping(&TestMapping{
		ScenarioName: "foo-test",
		TestedFiles:  []string{"pkg/foo/bar.go"},
	})
	ia.RegisterMapping(&TestMapping{
		ScenarioName: "baz-test",
		TestedFiles:  []string{"pkg/baz/qux.go"},
	})
	ia.RegisterMapping(&TestMapping{
		ScenarioName: "unaffected-test",
		TestedFiles:  []string{"pkg/other/file.go"},
	})

	changes := []FileChange{
		{Path: "pkg/foo/bar.go", ChangeType: "modified", LinesAdded: 10, LinesRemoved: 5},
	}

	report, err := ia.AnalyzeChanges(context.Background(), changes, "base", "head")
	if err != nil {
		t.Fatalf("AnalyzeChanges() error: %v", err)
	}

	// Should identify foo-test as affected
	if report.AffectedTests != 1 {
		t.Errorf("AffectedTests = %d, expected 1", report.AffectedTests)
	}

	foundFooTest := false
	for _, test := range report.SuggestedTests {
		if test == "foo-test" {
			foundFooTest = true
		}
	}

	if !foundFooTest {
		t.Error("SuggestedTests should include foo-test")
	}

	// Unaffected test should be skippable
	foundUnaffected := false
	for _, test := range report.SkippableTests {
		if test == "unaffected-test" {
			foundUnaffected = true
		}
	}

	if !foundUnaffected {
		t.Error("SkippableTests should include unaffected-test")
	}
}

func TestImpactAnalyzer_AnalyzeChanges_IgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
		CriticalPaths:    []string{},
		IgnorePatterns:   []string{"*.md", "*.txt"},
	}

	ia := NewImpactAnalyzer(cfg)

	changes := []FileChange{
		{Path: "README.md", ChangeType: "modified", LinesAdded: 100, LinesRemoved: 50},
		{Path: "CHANGELOG.txt", ChangeType: "modified", LinesAdded: 20, LinesRemoved: 10},
	}

	report, err := ia.AnalyzeChanges(context.Background(), changes, "base", "head")
	if err != nil {
		t.Fatalf("AnalyzeChanges() error: %v", err)
	}

	// Should have no results since all files are ignored
	if len(report.Results) != 0 {
		t.Errorf("Results length = %d, expected 0 (ignored files)", len(report.Results))
	}
}

func TestImpactAnalyzer_AnalyzeFileChange_HighImpact(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
	}

	ia := NewImpactAnalyzer(cfg)

	// Register many test mappings for the file
	for i := 0; i < 15; i++ {
		ia.RegisterMapping(&TestMapping{
			ScenarioName: "test-" + string(rune('a'+i)),
			TestedFiles:  []string{"pkg/core/important.go"},
		})
	}

	change := FileChange{
		Path:         "pkg/core/important.go",
		ChangeType:   "modified",
		LinesAdded:   600, // Many lines changed
		LinesRemoved: 200,
	}

	result := ia.analyzeFileChange(change)

	if result.ImpactLevel != ImpactLevelHigh {
		t.Errorf("ImpactLevel = %v, expected High", result.ImpactLevel)
	}
}

func TestImpactAnalyzer_ShouldIgnore(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:       tmpDir,
		IgnorePatterns: []string{"*.md", "vendor/*", ".git/*"},
	}

	ia := NewImpactAnalyzer(cfg)

	tests := []struct {
		path     string
		expected bool
	}{
		{"README.md", true},
		{"docs/GUIDE.md", true},
		{"vendor/github.com/foo/bar.go", true},
		{".git/config", true},
		{"pkg/foo/bar.go", false},
		{"main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ia.shouldIgnore(tt.path)
			if result != tt.expected {
				t.Errorf("shouldIgnore(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		path     string
		pattern  string
		expected bool
	}{
		{"README.md", "*.md", true},
		{"docs/GUIDE.md", "*.md", true},
		{"main.go", "*.md", false},
		{"vendor/foo/bar.go", "vendor/*", true},
		{"pkg/foo/bar.go", "vendor/*", false},
		{"go.mod", "go.mod", true},
		{"go.sum", "go.mod", false},
		{"pkg/foo/bar.go", "foo", true}, // Contains match
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			result := matchPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, expected %v", tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestImpactAnalyzer_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
	}

	// Create analyzer and add mappings
	ia1 := NewImpactAnalyzer(cfg)
	ia1.RegisterMapping(&TestMapping{
		ScenarioName: "test-1",
		TestedFiles:  []string{"file1.go"},
	})
	ia1.RegisterMapping(&TestMapping{
		ScenarioName: "test-2",
		TestedFiles:  []string{"file2.go"},
	})

	// Create new analyzer and verify mappings were loaded
	ia2 := NewImpactAnalyzer(cfg)

	ia2.mu.RLock()
	count := len(ia2.mappings)
	ia2.mu.RUnlock()

	if count != 2 {
		t.Errorf("Expected 2 mappings from persistence, got %d", count)
	}
}

func TestImpactAnalyzer_AnalyzeChanges_EmptyChanges(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
	}

	ia := NewImpactAnalyzer(cfg)

	report, err := ia.AnalyzeChanges(context.Background(), []FileChange{}, "base", "head")
	if err != nil {
		t.Fatalf("AnalyzeChanges() error: %v", err)
	}

	if report.FilesChanged != 0 {
		t.Errorf("FilesChanged = %d, expected 0", report.FilesChanged)
	}

	if report.TotalChanges != 0 {
		t.Errorf("TotalChanges = %d, expected 0", report.TotalChanges)
	}
}

func TestImpactAnalyzer_FunctionLevelMapping(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ImpactAnalyzerConfig{
		RootPath:         tmpDir,
		TestMappingsPath: filepath.Join(tmpDir, "mappings"),
	}

	ia := NewImpactAnalyzer(cfg)

	// Register mapping with function-level granularity
	ia.RegisterMapping(&TestMapping{
		ScenarioName: "foo-func-test",
		TestedFiles:  []string{},
		TestedFuncs:  []string{"DoFoo", "HandleFoo"},
	})

	change := FileChange{
		Path:       "pkg/foo/bar.go",
		ChangeType: "modified",
		Functions:  []string{"DoFoo", "OtherFunc"},
	}

	result := ia.analyzeFileChange(change)

	// Should match based on function name
	if len(result.AffectedTests) != 1 {
		t.Errorf("AffectedTests length = %d, expected 1", len(result.AffectedTests))
	}

	if len(result.AffectedTests) > 0 && result.AffectedTests[0].ScenarioName != "foo-func-test" {
		t.Errorf("AffectedTest = %q, expected 'foo-func-test'", result.AffectedTests[0].ScenarioName)
	}
}
