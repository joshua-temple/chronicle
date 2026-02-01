package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "passed status",
			status:   "passed",
			expected: "✓",
		},
		{
			name:     "completed status",
			status:   "completed",
			expected: "✓",
		},
		{
			name:     "success status",
			status:   "success",
			expected: "✓",
		},
		{
			name:     "failed status",
			status:   "failed",
			expected: "✗",
		},
		{
			name:     "error status",
			status:   "error",
			expected: "✗",
		},
		{
			name:     "skipped status",
			status:   "skipped",
			expected: "○",
		},
		{
			name:     "running status",
			status:   "running",
			expected: "●",
		},
		{
			name:     "in_progress status",
			status:   "in_progress",
			expected: "●",
		},
		{
			name:     "unknown status",
			status:   "unknown",
			expected: "·",
		},
		{
			name:     "empty status",
			status:   "",
			expected: "·",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StatusIcon(tt.status)
			if result != tt.expected {
				t.Errorf("StatusIcon(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	// Helper to check if a style can render text (non-nil check)
	styleCanRender := func(s lipgloss.Style, name string) {
		t.Helper()
		// If the style can render without panic, it's valid
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s style failed to render: %v", name, r)
			}
		}()
		_ = s.Render("test")
	}

	t.Run("creates valid styles", func(t *testing.T) {
		styleCanRender(styles.App, "App")
		styleCanRender(styles.Header, "Header")
		styleCanRender(styles.Title, "Title")
		styleCanRender(styles.Subtitle, "Subtitle")
		styleCanRender(styles.StatusBar, "StatusBar")
		styleCanRender(styles.Error, "Error")
		styleCanRender(styles.Success, "Success")
		styleCanRender(styles.Warning, "Warning")
		styleCanRender(styles.Muted, "Muted")
	})

	t.Run("creates list styles", func(t *testing.T) {
		styleCanRender(styles.List.Item, "List.Item")
		styleCanRender(styles.List.SelectedItem, "List.SelectedItem")
		styleCanRender(styles.List.Tag, "List.Tag")
		styleCanRender(styles.List.Description, "List.Description")
	})

	t.Run("creates execution styles", func(t *testing.T) {
		styleCanRender(styles.Execution.Running, "Execution.Running")
		styleCanRender(styles.Execution.Passed, "Execution.Passed")
		styleCanRender(styles.Execution.Failed, "Execution.Failed")
		styleCanRender(styles.Execution.Skipped, "Execution.Skipped")
		styleCanRender(styles.Execution.Pending, "Execution.Pending")
	})

	t.Run("creates results styles", func(t *testing.T) {
		styleCanRender(styles.Results.Summary, "Results.Summary")
		styleCanRender(styles.Results.PassRate, "Results.PassRate")
		styleCanRender(styles.Results.FailRate, "Results.FailRate")
	})
}

func TestStylesStatusStyle(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		name          string
		status        string
		expectedStyle lipgloss.Style
	}{
		{
			name:          "passed returns Execution.Passed",
			status:        "passed",
			expectedStyle: styles.Execution.Passed,
		},
		{
			name:          "completed returns Execution.Passed",
			status:        "completed",
			expectedStyle: styles.Execution.Passed,
		},
		{
			name:          "success returns Execution.Passed",
			status:        "success",
			expectedStyle: styles.Execution.Passed,
		},
		{
			name:          "failed returns Execution.Failed",
			status:        "failed",
			expectedStyle: styles.Execution.Failed,
		},
		{
			name:          "error returns Execution.Failed",
			status:        "error",
			expectedStyle: styles.Execution.Failed,
		},
		{
			name:          "skipped returns Execution.Skipped",
			status:        "skipped",
			expectedStyle: styles.Execution.Skipped,
		},
		{
			name:          "running returns Execution.Running",
			status:        "running",
			expectedStyle: styles.Execution.Running,
		},
		{
			name:          "in_progress returns Execution.Running",
			status:        "in_progress",
			expectedStyle: styles.Execution.Running,
		},
		{
			name:          "unknown returns Execution.Pending",
			status:        "unknown",
			expectedStyle: styles.Execution.Pending,
		},
		{
			name:          "empty returns Execution.Pending",
			status:        "",
			expectedStyle: styles.Execution.Pending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := styles.StatusStyle(tt.status)
			// Compare rendered output since lipgloss.Style doesn't have simple equality
			if result.Render("test") != tt.expectedStyle.Render("test") {
				t.Errorf("StatusStyle(%q) returned unexpected style", tt.status)
			}
		})
	}
}
