package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Helper to test if a key matches a binding
	testKeyMatch := func(t *testing.T, binding key.Binding, testKey string, desc string) {
		t.Helper()
		var msg tea.KeyMsg
		switch testKey {
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEscape}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(testKey)}
		}
		if !key.Matches(msg, binding) {
			t.Errorf("%s should match '%s' key", desc, testKey)
		}
	}

	t.Run("Up key binding", func(t *testing.T) {
		testKeyMatch(t, km.Up, "up", "Up")
	})

	t.Run("Down key binding", func(t *testing.T) {
		testKeyMatch(t, km.Down, "down", "Down")
	})

	t.Run("Enter key binding", func(t *testing.T) {
		testKeyMatch(t, km.Enter, "enter", "Enter")
	})

	t.Run("Back key binding", func(t *testing.T) {
		testKeyMatch(t, km.Back, "esc", "Back")
	})

	t.Run("Quit key binding", func(t *testing.T) {
		testKeyMatch(t, km.Quit, "q", "Quit")
	})

	t.Run("Help key binding", func(t *testing.T) {
		testKeyMatch(t, km.Help, "?", "Help")
	})

	t.Run("Tab key binding", func(t *testing.T) {
		testKeyMatch(t, km.Tab, "tab", "Tab")
	})

	t.Run("Run key binding", func(t *testing.T) {
		testKeyMatch(t, km.Run, "r", "Run")
	})

	t.Run("Refresh key binding", func(t *testing.T) {
		testKeyMatch(t, km.Refresh, "R", "Refresh")
	})

	t.Run("Filter key binding", func(t *testing.T) {
		testKeyMatch(t, km.Filter, "/", "Filter")
	})

	t.Run("Select key binding", func(t *testing.T) {
		testKeyMatch(t, km.Select, " ", "Select")
	})
}

func TestKeyMapShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	shortHelp := km.ShortHelp()

	if len(shortHelp) == 0 {
		t.Error("ShortHelp should return bindings")
	}

	// Verify specific bindings are included (7 bindings: Up, Down, Enter, Run, RunAll, Tab, Quit)
	expectedBindings := []key.Binding{km.Up, km.Down, km.Enter, km.Run, km.RunAll, km.Tab, km.Quit}
	if len(shortHelp) != len(expectedBindings) {
		t.Errorf("ShortHelp returned %d bindings, expected %d", len(shortHelp), len(expectedBindings))
	}
}

func TestKeyMapFullHelp(t *testing.T) {
	km := DefaultKeyMap()
	fullHelp := km.FullHelp()

	if len(fullHelp) == 0 {
		t.Error("FullHelp should return binding groups")
	}

	// Should have 3 groups
	if len(fullHelp) != 3 {
		t.Errorf("FullHelp returned %d groups, expected 3", len(fullHelp))
	}

	// First group: navigation (Up, Down, Enter, Back)
	if len(fullHelp[0]) != 4 {
		t.Errorf("First group should have 4 bindings, got %d", len(fullHelp[0]))
	}

	// Second group: actions (Run, RunAll, RunSelected, Select)
	if len(fullHelp[1]) != 4 {
		t.Errorf("Second group should have 4 bindings, got %d", len(fullHelp[1]))
	}

	// Third group: general (Tab, Refresh, Filter, Help, Quit)
	if len(fullHelp[2]) != 5 {
		t.Errorf("Third group should have 5 bindings, got %d", len(fullHelp[2]))
	}
}

func TestNewModel(t *testing.T) {
	// Test creating a model with nil parameters (should not panic)
	t.Run("creates model with nil config", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewModel panicked with nil config: %v", r)
			}
		}()

		model := NewModel(nil, nil, nil)

		if model.view != ViewScenarios {
			t.Errorf("Initial view should be ViewScenarios, got %d", model.view)
		}

		if model.executing {
			t.Error("Model should not be executing initially")
		}

		if model.quitting {
			t.Error("Model should not be quitting initially")
		}

		if model.executionLogs == nil {
			t.Error("Execution logs should be initialized")
		}
	})
}

func TestModelView_NotReady(t *testing.T) {
	model := NewModel(nil, nil, nil)
	model.ready = false

	view := model.View()

	if view != "Loading...\n" {
		t.Errorf("View when not ready should be 'Loading...\\n', got %q", view)
	}
}

func TestModelView_Quitting(t *testing.T) {
	model := NewModel(nil, nil, nil)
	model.quitting = true

	view := model.View()

	if view != "Goodbye!\n" {
		t.Errorf("View when quitting should be 'Goodbye!\\n', got %q", view)
	}
}
