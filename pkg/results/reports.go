package results

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"
)

// Reporter generates reports in various formats.
type Reporter interface {
	Generate(result *RunResult) ([]byte, error)
	ContentType() string
	FileExtension() string
}

// JUnitReporter generates JUnit XML reports.
type JUnitReporter struct{}

// NewJUnitReporter creates a new JUnit reporter.
func NewJUnitReporter() *JUnitReporter {
	return &JUnitReporter{}
}

// junitTestSuites is the root element for JUnit XML.
type junitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Name     string           `xml:"name,attr"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Skipped  int              `xml:"skipped,attr"`
	Time     float64          `xml:"time,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// Generate creates a JUnit XML report.
func (r *JUnitReporter) Generate(result *RunResult) ([]byte, error) {
	suite := junitTestSuite{
		Name:      result.Name,
		Tests:     result.Stats.Total,
		Failures:  result.Stats.Failed,
		Errors:    0,
		Skipped:   result.Stats.Skipped,
		Time:      result.Duration.Seconds(),
		Timestamp: result.StartTime.Format(time.RFC3339),
		TestCases: make([]junitTestCase, 0, len(result.Scenarios)),
	}

	for _, s := range result.Scenarios {
		tc := junitTestCase{
			Name:      s.ScenarioName,
			ClassName: result.Name,
			Time:      s.Duration.Seconds(),
		}

		switch s.State {
		case "failed":
			tc.Failure = &junitFailure{
				Message: s.Error,
				Type:    "AssertionError",
				Content: buildFailureContent(s),
			}
		case "skipped":
			tc.Skipped = &junitSkipped{
				Message: s.SkipReason,
			}
		}

		suite.TestCases = append(suite.TestCases, tc)
	}

	suites := junitTestSuites{
		Name:     result.Name,
		Tests:    result.Stats.Total,
		Failures: result.Stats.Failed,
		Errors:   0,
		Skipped:  result.Stats.Skipped,
		Time:     result.Duration.Seconds(),
		Suites:   []junitTestSuite{suite},
	}

	return xml.MarshalIndent(suites, "", "  ")
}

// ContentType returns the MIME type for JUnit XML.
func (r *JUnitReporter) ContentType() string {
	return "application/xml"
}

// FileExtension returns the file extension for JUnit XML.
func (r *JUnitReporter) FileExtension() string {
	return ".xml"
}

// buildFailureContent creates detailed failure content for JUnit.
func buildFailureContent(s ScenarioRunResult) string {
	var b strings.Builder

	if s.Error != "" {
		b.WriteString(s.Error)
		b.WriteString("\n\n")
	}

	for _, fr := range s.FlowResults {
		if fr.State == "failed" {
			b.WriteString(fmt.Sprintf("%s [%s]: %s\n", fr.Name, fr.Type, fr.Error))
		}
	}

	return b.String()
}

// JSONReporter generates JSON reports.
type JSONReporter struct {
	Compact bool
}

// NewJSONReporter creates a new JSON reporter.
func NewJSONReporter(compact bool) *JSONReporter {
	return &JSONReporter{Compact: compact}
}

// Generate creates a JSON report.
func (r *JSONReporter) Generate(result *RunResult) ([]byte, error) {
	if r.Compact {
		return json.Marshal(result)
	}
	return json.MarshalIndent(result, "", "  ")
}

// ContentType returns the MIME type for JSON.
func (r *JSONReporter) ContentType() string {
	return "application/json"
}

// FileExtension returns the file extension for JSON.
func (r *JSONReporter) FileExtension() string {
	return ".json"
}

// HTMLReporter generates HTML reports.
type HTMLReporter struct {
	template *template.Template
}

// NewHTMLReporter creates a new HTML reporter.
func NewHTMLReporter() *HTMLReporter {
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"formatDuration": func(d time.Duration) string {
			return d.Round(time.Millisecond).String()
		},
		"statusClass": func(state string) string {
			switch state {
			case "completed":
				return "success"
			case "failed":
				return "failure"
			case "skipped":
				return "skipped"
			default:
				return "unknown"
			}
		},
		"statusIcon": func(state string) string {
			switch state {
			case "completed":
				return "✓"
			case "failed":
				return "✗"
			case "skipped":
				return "○"
			default:
				return "?"
			}
		},
	}).Parse(htmlTemplate))

	return &HTMLReporter{template: tmpl}
}

// Generate creates an HTML report.
func (r *HTMLReporter) Generate(result *RunResult) ([]byte, error) {
	var buf strings.Builder
	if err := r.template.Execute(&buf, result); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return []byte(buf.String()), nil
}

// ContentType returns the MIME type for HTML.
func (r *HTMLReporter) ContentType() string {
	return "text/html"
}

// FileExtension returns the file extension for HTML.
func (r *HTMLReporter) FileExtension() string {
	return ".html"
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chronicle Report: {{.Name}}</title>
    <style>
        :root {
            --success-color: #28a745;
            --failure-color: #dc3545;
            --skipped-color: #6c757d;
            --bg-color: #f8f9fa;
            --card-bg: #ffffff;
            --border-color: #dee2e6;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: var(--bg-color);
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            background: var(--card-bg);
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .header h1 {
            margin: 0 0 10px 0;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-weight: bold;
            font-size: 14px;
        }
        .status-badge.success {
            background-color: var(--success-color);
            color: white;
        }
        .status-badge.failure {
            background-color: var(--failure-color);
            color: white;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 15px;
            margin: 20px 0;
        }
        .stat-card {
            background: var(--card-bg);
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .stat-card .value {
            font-size: 32px;
            font-weight: bold;
        }
        .stat-card .label {
            color: #6c757d;
            font-size: 14px;
        }
        .stat-card.passed .value { color: var(--success-color); }
        .stat-card.failed .value { color: var(--failure-color); }
        .stat-card.skipped .value { color: var(--skipped-color); }
        .scenarios {
            background: var(--card-bg);
            border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .scenarios h2 {
            margin: 0;
            padding: 15px 20px;
            background: #f1f3f5;
            border-bottom: 1px solid var(--border-color);
        }
        .scenario {
            border-bottom: 1px solid var(--border-color);
            padding: 15px 20px;
        }
        .scenario:last-child {
            border-bottom: none;
        }
        .scenario-header {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .scenario-icon {
            font-size: 18px;
        }
        .scenario-icon.success { color: var(--success-color); }
        .scenario-icon.failure { color: var(--failure-color); }
        .scenario-icon.skipped { color: var(--skipped-color); }
        .scenario-name {
            font-weight: 500;
            flex: 1;
        }
        .scenario-duration {
            color: #6c757d;
            font-size: 14px;
        }
        .scenario-error {
            margin-top: 10px;
            padding: 10px;
            background: #fff5f5;
            border-left: 3px solid var(--failure-color);
            font-family: monospace;
            font-size: 13px;
            overflow-x: auto;
        }
        .flow-items {
            margin-top: 10px;
            padding-left: 30px;
            font-size: 14px;
        }
        .flow-item {
            padding: 5px 0;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .flow-item-type {
            background: #e9ecef;
            padding: 2px 6px;
            border-radius: 4px;
            font-size: 12px;
        }
        footer {
            text-align: center;
            padding: 20px;
            color: #6c757d;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Name}}</h1>
            <span class="status-badge {{if eq .Stats.Failed 0}}success{{else}}failure{{end}}">
                {{if eq .Stats.Failed 0}}PASS{{else}}FAIL{{end}}
            </span>
            <div style="margin-top: 10px; color: #6c757d;">
                Duration: {{formatDuration .Duration}} |
                Started: {{.StartTime.Format "2006-01-02 15:04:05"}}
            </div>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="value">{{.Stats.Total}}</div>
                <div class="label">Total</div>
            </div>
            <div class="stat-card passed">
                <div class="value">{{.Stats.Passed}}</div>
                <div class="label">Passed</div>
            </div>
            <div class="stat-card failed">
                <div class="value">{{.Stats.Failed}}</div>
                <div class="label">Failed</div>
            </div>
            <div class="stat-card skipped">
                <div class="value">{{.Stats.Skipped}}</div>
                <div class="label">Skipped</div>
            </div>
        </div>

        <div class="scenarios">
            <h2>Scenarios</h2>
            {{range .Scenarios}}
            <div class="scenario">
                <div class="scenario-header">
                    <span class="scenario-icon {{statusClass .State}}">{{statusIcon .State}}</span>
                    <span class="scenario-name">{{.ScenarioName}}</span>
                    <span class="scenario-duration">{{formatDuration .Duration}}</span>
                </div>
                {{if .Error}}
                <div class="scenario-error">{{.Error}}</div>
                {{end}}
                {{if .SkipReason}}
                <div class="scenario-error" style="background: #f8f9fa; border-color: var(--skipped-color);">
                    Skipped: {{.SkipReason}}
                </div>
                {{end}}
                {{if .FlowResults}}
                <div class="flow-items">
                    {{range .FlowResults}}
                    <div class="flow-item">
                        <span class="scenario-icon {{statusClass .State}}">{{statusIcon .State}}</span>
                        <span>{{.Name}}</span>
                        <span class="flow-item-type">{{.Type}}</span>
                        <span class="scenario-duration">{{formatDuration .Duration}}</span>
                    </div>
                    {{if .Error}}
                    <div class="scenario-error" style="margin-left: 25px;">{{.Error}}</div>
                    {{end}}
                    {{end}}
                </div>
                {{end}}
            </div>
            {{end}}
        </div>

        <footer>
            Generated by Chronicle at {{.EndTime.Format "2006-01-02 15:04:05"}}
        </footer>
    </div>
</body>
</html>`

// TextReporter generates plain text reports.
type TextReporter struct {
	Style NarrativeStyle
}

// NewTextReporter creates a new text reporter.
func NewTextReporter(style NarrativeStyle) *TextReporter {
	return &TextReporter{Style: style}
}

// Generate creates a text report.
func (r *TextReporter) Generate(result *RunResult) ([]byte, error) {
	ng := NewNarrativeGenerator(WithStyle(r.Style))
	return []byte(ng.Generate(result)), nil
}

// ContentType returns the MIME type for text.
func (r *TextReporter) ContentType() string {
	return "text/plain"
}

// FileExtension returns the file extension for text.
func (r *TextReporter) FileExtension() string {
	return ".txt"
}

// MarkdownReporter generates Markdown reports.
type MarkdownReporter struct{}

// NewMarkdownReporter creates a new Markdown reporter.
func NewMarkdownReporter() *MarkdownReporter {
	return &MarkdownReporter{}
}

// Generate creates a Markdown report.
func (r *MarkdownReporter) Generate(result *RunResult) ([]byte, error) {
	ng := NewNarrativeGenerator(WithStyle(StyleMarkdown))
	return []byte(ng.Generate(result)), nil
}

// ContentType returns the MIME type for Markdown.
func (r *MarkdownReporter) ContentType() string {
	return "text/markdown"
}

// FileExtension returns the file extension for Markdown.
func (r *MarkdownReporter) FileExtension() string {
	return ".md"
}

// ReportWriter helps write reports to various destinations.
type ReportWriter struct {
	reporter Reporter
}

// NewReportWriter creates a new report writer.
func NewReportWriter(reporter Reporter) *ReportWriter {
	return &ReportWriter{reporter: reporter}
}

// Write generates a report and writes it to the given writer.
func (w *ReportWriter) Write(result *RunResult, out io.Writer) error {
	data, err := w.reporter.Generate(result)
	if err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	_, err = out.Write(data)
	return err
}

// GenerateAll generates reports in multiple formats.
func GenerateAll(result *RunResult, reporters ...Reporter) (map[string][]byte, error) {
	reports := make(map[string][]byte)

	for _, r := range reporters {
		data, err := r.Generate(result)
		if err != nil {
			return nil, fmt.Errorf("generate %s report: %w", r.FileExtension(), err)
		}
		reports[r.FileExtension()] = data
	}

	return reports, nil
}

// GetReporter returns a reporter by format name.
func GetReporter(format string) (Reporter, error) {
	switch strings.ToLower(format) {
	case "json":
		return NewJSONReporter(false), nil
	case "json-compact":
		return NewJSONReporter(true), nil
	case "junit", "xml":
		return NewJUnitReporter(), nil
	case "html":
		return NewHTMLReporter(), nil
	case "text", "txt":
		return NewTextReporter(StyleStandard), nil
	case "markdown", "md":
		return NewMarkdownReporter(), nil
	case "brief":
		return NewTextReporter(StyleBrief), nil
	case "verbose":
		return NewTextReporter(StyleVerbose), nil
	default:
		return nil, fmt.Errorf("unknown report format: %s", format)
	}
}
