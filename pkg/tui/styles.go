package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors for the TUI theme.
var (
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	warningColor   = lipgloss.Color("#F59E0B") // Yellow
	mutedColor     = lipgloss.Color("#6B7280") // Gray
)

// Styles defines all styles used in the TUI.
type Styles struct {
	App           lipgloss.Style
	Header        lipgloss.Style
	Title         lipgloss.Style
	Subtitle      lipgloss.Style
	StatusBar     lipgloss.Style
	Help          lipgloss.Style
	List          ListStyles
	Execution     ExecutionStyles
	Results       ResultStyles
	Spinner       lipgloss.Style
	Error         lipgloss.Style
	Success       lipgloss.Style
	Warning       lipgloss.Style
	Muted         lipgloss.Style
	Selected      lipgloss.Style
	Focused       lipgloss.Style
}

// ListStyles defines styles for the list view.
type ListStyles struct {
	Item         lipgloss.Style
	SelectedItem lipgloss.Style
	Tag          lipgloss.Style
	Description  lipgloss.Style
}

// ExecutionStyles defines styles for the execution view.
type ExecutionStyles struct {
	Running   lipgloss.Style
	Passed    lipgloss.Style
	Failed    lipgloss.Style
	Skipped   lipgloss.Style
	Pending   lipgloss.Style
	Progress  lipgloss.Style
	Component lipgloss.Style
	Duration  lipgloss.Style
}

// ResultStyles defines styles for the results view.
type ResultStyles struct {
	Summary   lipgloss.Style
	PassRate  lipgloss.Style
	FailRate  lipgloss.Style
	Duration  lipgloss.Style
	Timestamp lipgloss.Style
	Details   lipgloss.Style
}

// DefaultStyles returns the default TUI styles.
func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(mutedColor).
			MarginBottom(1).
			Padding(0, 1),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor),

		Subtitle: lipgloss.NewStyle().
			Foreground(mutedColor),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1),

		Help: lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1),

		List: ListStyles{
			Item: lipgloss.NewStyle().
				PaddingLeft(2),
			SelectedItem: lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(primaryColor).
				Bold(true),
			Tag: lipgloss.NewStyle().
				Foreground(secondaryColor).
				Background(lipgloss.Color("#064E3B")).
				Padding(0, 1).
				MarginRight(1),
			Description: lipgloss.NewStyle().
				Foreground(mutedColor),
		},

		Execution: ExecutionStyles{
			Running: lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true),
			Passed: lipgloss.NewStyle().
				Foreground(secondaryColor),
			Failed: lipgloss.NewStyle().
				Foreground(errorColor),
			Skipped: lipgloss.NewStyle().
				Foreground(warningColor),
			Pending: lipgloss.NewStyle().
				Foreground(mutedColor),
			Progress: lipgloss.NewStyle().
				Foreground(primaryColor),
			Component: lipgloss.NewStyle().
				Bold(true),
			Duration: lipgloss.NewStyle().
				Foreground(mutedColor),
		},

		Results: ResultStyles{
			Summary: lipgloss.NewStyle().
				Bold(true).
				Padding(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor),
			PassRate: lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true),
			FailRate: lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true),
			Duration: lipgloss.NewStyle().
				Foreground(mutedColor),
			Timestamp: lipgloss.NewStyle().
				Foreground(mutedColor),
			Details: lipgloss.NewStyle().
				PaddingLeft(2),
		},

		Spinner: lipgloss.NewStyle().
			Foreground(primaryColor),

		Error: lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true),

		Warning: lipgloss.NewStyle().
			Foreground(warningColor),

		Muted: lipgloss.NewStyle().
			Foreground(mutedColor),

		Selected: lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")),

		Focused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor),
	}
}

// StatusIcon returns an icon for the given status.
func StatusIcon(status string) string {
	switch status {
	case "passed", "completed", "success":
		return "✓"
	case "failed", "error":
		return "✗"
	case "skipped":
		return "○"
	case "running", "in_progress":
		return "●"
	default:
		return "·"
	}
}

// StatusStyle returns a style for the given status.
func (s Styles) StatusStyle(status string) lipgloss.Style {
	switch status {
	case "passed", "completed", "success":
		return s.Execution.Passed
	case "failed", "error":
		return s.Execution.Failed
	case "skipped":
		return s.Execution.Skipped
	case "running", "in_progress":
		return s.Execution.Running
	default:
		return s.Execution.Pending
	}
}
