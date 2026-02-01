package results

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestJUnitReporter(t *testing.T) {
	result := createTestResultWithFailure()
	reporter := NewJUnitReporter()

	data, err := reporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify it's valid XML
	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	if suites.Name != result.Name {
		t.Errorf("expected name %s, got %s", result.Name, suites.Name)
	}
	if suites.Tests != result.Stats.Total {
		t.Errorf("expected %d tests, got %d", result.Stats.Total, suites.Tests)
	}
	if suites.Failures != result.Stats.Failed {
		t.Errorf("expected %d failures, got %d", result.Stats.Failed, suites.Failures)
	}
	if len(suites.Suites) != 1 {
		t.Errorf("expected 1 suite, got %d", len(suites.Suites))
	}
	if len(suites.Suites[0].TestCases) != len(result.Scenarios) {
		t.Errorf("expected %d test cases, got %d", len(result.Scenarios), len(suites.Suites[0].TestCases))
	}
}

func TestJUnitReporterContentType(t *testing.T) {
	reporter := NewJUnitReporter()

	if reporter.ContentType() != "application/xml" {
		t.Errorf("expected application/xml, got %s", reporter.ContentType())
	}
	if reporter.FileExtension() != ".xml" {
		t.Errorf("expected .xml, got %s", reporter.FileExtension())
	}
}

func TestJUnitReporterFailure(t *testing.T) {
	result := &RunResult{
		Name:      "test",
		StartTime: time.Now(),
		Duration:  100 * time.Millisecond,
		Stats:     RunStats{Total: 1, Failed: 1},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "failing-test",
				State:        "failed",
				Error:        "assertion failed",
				Duration:     100 * time.Millisecond,
				FlowResults: []FlowItemRunResult{
					{Name: "step1", Type: "task", State: "failed", Error: "step error"},
				},
			},
		},
	}

	reporter := NewJUnitReporter()
	data, err := reporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	tc := suites.Suites[0].TestCases[0]
	if tc.Failure == nil {
		t.Fatal("expected failure element")
	}
	if tc.Failure.Message != "assertion failed" {
		t.Errorf("expected failure message 'assertion failed', got %s", tc.Failure.Message)
	}
	if !strings.Contains(tc.Failure.Content, "step error") {
		t.Errorf("expected failure content to contain 'step error', got %s", tc.Failure.Content)
	}
}

func TestJUnitReporterSkipped(t *testing.T) {
	result := &RunResult{
		Name:      "test",
		StartTime: time.Now(),
		Duration:  100 * time.Millisecond,
		Stats:     RunStats{Total: 1, Skipped: 1},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "skipped-test",
				State:        "skipped",
				SkipReason:   "not applicable",
			},
		},
	}

	reporter := NewJUnitReporter()
	data, err := reporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	tc := suites.Suites[0].TestCases[0]
	if tc.Skipped == nil {
		t.Fatal("expected skipped element")
	}
	if tc.Skipped.Message != "not applicable" {
		t.Errorf("expected skip message 'not applicable', got %s", tc.Skipped.Message)
	}
}

func TestJSONReporter(t *testing.T) {
	result := createTestResult()

	// Test pretty JSON
	reporter := NewJSONReporter(false)
	data, err := reporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(string(data), "\n") {
		t.Error("expected pretty-printed JSON to have newlines")
	}

	// Test compact JSON
	compactReporter := NewJSONReporter(true)
	compactData, err := compactReporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate compact failed: %v", err)
	}

	if strings.Contains(string(compactData), "\n  ") {
		t.Error("expected compact JSON to not have formatted newlines")
	}
}

func TestJSONReporterContentType(t *testing.T) {
	reporter := NewJSONReporter(false)

	if reporter.ContentType() != "application/json" {
		t.Errorf("expected application/json, got %s", reporter.ContentType())
	}
	if reporter.FileExtension() != ".json" {
		t.Errorf("expected .json, got %s", reporter.FileExtension())
	}
}

func TestHTMLReporter(t *testing.T) {
	result := createTestResultWithFailure()
	reporter := NewHTMLReporter()

	data, err := reporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	html := string(data)

	// Check for key HTML elements
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(html, "<title>Chronicle Report:") {
		t.Error("expected title element")
	}
	if !strings.Contains(html, result.Name) {
		t.Errorf("expected report to contain run name %s", result.Name)
	}
	if !strings.Contains(html, "Passed") {
		t.Error("expected 'Passed' in report")
	}
	if !strings.Contains(html, "Failed") {
		t.Error("expected 'Failed' in report")
	}
}

func TestHTMLReporterContentType(t *testing.T) {
	reporter := NewHTMLReporter()

	if reporter.ContentType() != "text/html" {
		t.Errorf("expected text/html, got %s", reporter.ContentType())
	}
	if reporter.FileExtension() != ".html" {
		t.Errorf("expected .html, got %s", reporter.FileExtension())
	}
}

func TestTextReporter(t *testing.T) {
	result := createTestResult()

	tests := []struct {
		style NarrativeStyle
		check string
	}{
		{StyleBrief, "PASS"},
		{StyleStandard, "Chronicle Run"},
		{StyleVerbose, "Chronicle Run Report"},
	}

	for _, tc := range tests {
		reporter := NewTextReporter(tc.style)
		data, err := reporter.Generate(result)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if !strings.Contains(string(data), tc.check) {
			t.Errorf("expected text to contain %s for style %v", tc.check, tc.style)
		}
	}
}

func TestTextReporterContentType(t *testing.T) {
	reporter := NewTextReporter(StyleStandard)

	if reporter.ContentType() != "text/plain" {
		t.Errorf("expected text/plain, got %s", reporter.ContentType())
	}
	if reporter.FileExtension() != ".txt" {
		t.Errorf("expected .txt, got %s", reporter.FileExtension())
	}
}

func TestMarkdownReporter(t *testing.T) {
	result := createTestResult()
	reporter := NewMarkdownReporter()

	data, err := reporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	md := string(data)

	if !strings.Contains(md, "# Chronicle Run") {
		t.Error("expected markdown header")
	}
	if !strings.Contains(md, "## Summary") {
		t.Error("expected summary section")
	}
}

func TestMarkdownReporterContentType(t *testing.T) {
	reporter := NewMarkdownReporter()

	if reporter.ContentType() != "text/markdown" {
		t.Errorf("expected text/markdown, got %s", reporter.ContentType())
	}
	if reporter.FileExtension() != ".md" {
		t.Errorf("expected .md, got %s", reporter.FileExtension())
	}
}

func TestReportWriter(t *testing.T) {
	result := createTestResult()
	reporter := NewJSONReporter(false)
	writer := NewReportWriter(reporter)

	var buf bytes.Buffer
	err := writer.Write(result, &buf)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestGenerateAll(t *testing.T) {
	result := createTestResult()

	reporters := []Reporter{
		NewJSONReporter(false),
		NewJUnitReporter(),
		NewHTMLReporter(),
	}

	reports, err := GenerateAll(result, reporters...)
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	if len(reports) != 3 {
		t.Errorf("expected 3 reports, got %d", len(reports))
	}

	if _, ok := reports[".json"]; !ok {
		t.Error("expected .json report")
	}
	if _, ok := reports[".xml"]; !ok {
		t.Error("expected .xml report")
	}
	if _, ok := reports[".html"]; !ok {
		t.Error("expected .html report")
	}
}

func TestGetReporter(t *testing.T) {
	tests := []struct {
		format     string
		wantExt    string
		shouldFail bool
	}{
		{"json", ".json", false},
		{"json-compact", ".json", false},
		{"junit", ".xml", false},
		{"xml", ".xml", false},
		{"html", ".html", false},
		{"text", ".txt", false},
		{"txt", ".txt", false},
		{"markdown", ".md", false},
		{"md", ".md", false},
		{"brief", ".txt", false},
		{"verbose", ".txt", false},
		{"unknown", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			reporter, err := GetReporter(tc.format)

			if tc.shouldFail {
				if err == nil {
					t.Error("expected error for unknown format")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if reporter.FileExtension() != tc.wantExt {
				t.Errorf("expected extension %s, got %s", tc.wantExt, reporter.FileExtension())
			}
		})
	}
}

func TestGetReporterCaseInsensitive(t *testing.T) {
	formats := []string{"JSON", "Json", "jSoN"}

	for _, format := range formats {
		reporter, err := GetReporter(format)
		if err != nil {
			t.Errorf("GetReporter(%q) failed: %v", format, err)
		}
		if reporter == nil {
			t.Errorf("GetReporter(%q) returned nil", format)
		}
	}
}

func createTestResultWithFailure() *RunResult {
	return &RunResult{
		ID:        "test-id",
		Name:      "test-run",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Stats: RunStats{
			Total:       3,
			Passed:      1,
			Failed:      1,
			Skipped:     1,
			AvgDuration: 33 * time.Millisecond,
		},
		Scenarios: []ScenarioRunResult{
			{
				ScenarioName: "passing-scenario",
				State:        "completed",
				Duration:     30 * time.Millisecond,
				FlowResults: []FlowItemRunResult{
					{Name: "setup", Type: "setup", State: "completed", Duration: 10 * time.Millisecond},
					{Name: "action", Type: "task", State: "completed", Duration: 20 * time.Millisecond},
				},
			},
			{
				ScenarioName: "failing-scenario",
				State:        "failed",
				Error:        "assertion failed: expected true",
				Duration:     50 * time.Millisecond,
				FlowResults: []FlowItemRunResult{
					{Name: "setup", Type: "setup", State: "completed", Duration: 10 * time.Millisecond},
					{Name: "action", Type: "task", State: "failed", Error: "assertion failed", Duration: 40 * time.Millisecond},
				},
			},
			{
				ScenarioName: "skipped-scenario",
				State:        "skipped",
				SkipReason:   "feature not available",
			},
		},
	}
}
