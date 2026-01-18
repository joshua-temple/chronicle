package chronicle

import (
	"bytes"
	"fmt"
)

// MarkdownRenderer outputs formatted markdown
type MarkdownRenderer struct {
	IncludeTimings bool
	IncludeErrors  bool
}

// Render renders an entry to Markdown
func (r *MarkdownRenderer) Render(entry *Entry) ([]byte, error) {
	var buf bytes.Buffer

	// Title
	buf.WriteString(fmt.Sprintf("# %s\n", entry.Name))
	if entry.Description != "" {
		buf.WriteString(fmt.Sprintf("%s\n", entry.Description))
	}
	buf.WriteString("\n")

	// Render children
	r.renderChildren(&buf, entry.Children, 0)

	return buf.Bytes(), nil
}

// RenderSuite renders a suite chronicle to Markdown
func (r *MarkdownRenderer) RenderSuite(suite *SuiteChronicle) ([]byte, error) {
	var buf bytes.Buffer

	// Title
	buf.WriteString(fmt.Sprintf("# %s\n", suite.Name))
	if suite.Description != "" {
		buf.WriteString(fmt.Sprintf("%s\n", suite.Description))
	}
	buf.WriteString("\n")

	// Summary
	passed, failed, skipped := suite.Summary()
	buf.WriteString(fmt.Sprintf("**Summary:** %d passed, %d failed, %d skipped", passed, failed, skipped))
	if r.IncludeTimings {
		buf.WriteString(fmt.Sprintf(" | Duration: %s", suite.Duration))
	}
	buf.WriteString("\n\n")

	// Scenarios
	buf.WriteString("## Scenarios\n\n")
	for _, scenario := range suite.Scenarios {
		r.renderScenario(&buf, scenario)
	}

	return buf.Bytes(), nil
}

func (r *MarkdownRenderer) renderScenario(buf *bytes.Buffer, entry *Entry) {
	icon := r.statusIcon(entry.Status)
	buf.WriteString(fmt.Sprintf("### %s %s", icon, entry.Name))
	if r.IncludeTimings && entry.Duration > 0 {
		buf.WriteString(fmt.Sprintf(" (%s)", entry.Duration))
	}
	if entry.Status == Skipped && entry.Description != "" {
		buf.WriteString(fmt.Sprintf(" — skipped: %s", entry.Description))
	}
	buf.WriteString("\n")

	if r.IncludeErrors && entry.Error != "" {
		buf.WriteString(fmt.Sprintf("\n**Error:** %s\n", entry.Error))
	}

	if len(entry.Children) > 0 {
		r.renderChildren(buf, entry.Children, 0)
	}
	buf.WriteString("\n")
}

func (r *MarkdownRenderer) renderChildren(buf *bytes.Buffer, children []*Entry, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	for _, child := range children {
		icon := r.statusIcon(child.Status)
		buf.WriteString(fmt.Sprintf("%s- %s **%s**", indent, icon, child.Name))

		if r.IncludeTimings && child.Duration > 0 {
			buf.WriteString(fmt.Sprintf(" (%s)", child.Duration))
		}

		if child.Status == Skipped && child.Description != "" {
			buf.WriteString(fmt.Sprintf(" — skipped: %s", child.Description))
		}

		if r.IncludeErrors && child.Error != "" {
			buf.WriteString(fmt.Sprintf(" — error: %s", child.Error))
		}

		buf.WriteString("\n")

		if len(child.Children) > 0 {
			r.renderChildren(buf, child.Children, depth+1)
		}
	}
}

func (r *MarkdownRenderer) statusIcon(status Status) string {
	switch status {
	case Passed:
		return "✅"
	case Failed:
		return "❌"
	case Skipped:
		return "⏭️"
	default:
		return "❓"
	}
}
