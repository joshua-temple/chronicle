package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/execution"
	"github.com/joshua-temple/chronicle/pkg/results"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

// View represents the current view in the TUI.
type View int

const (
	ViewScenarios View = iota
	ViewExecution
	ViewResults
	ViewHelp
)

// Model is the main TUI application model.
type Model struct {
	// Configuration
	config   *config.Config
	executor *execution.Executor

	// UI state
	view         View
	width        int
	height       int
	ready        bool
	err          error
	styles       Styles
	help         help.Model
	keys         KeyMap
	quitting     bool

	// Scenario list
	scenarioList list.Model
	scenarios    []*scenario.Scenario

	// Execution state
	spinner       spinner.Model
	executing     bool
	currentResult *execution.ScenarioResult
	executionLogs []string

	// Results view
	resultsViewport viewport.Model
	resultsList     []*results.RunResult

	// Storage
	storage results.Storage
}

// ScenarioItem represents a scenario in the list.
type ScenarioItem struct {
	scenario *scenario.Scenario
}

func (i ScenarioItem) Title() string       { return i.scenario.Name }
func (i ScenarioItem) Description() string { return i.scenario.Description }
func (i ScenarioItem) FilterValue() string { return i.scenario.Name }

// KeyMap defines the keybindings for the TUI.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Tab      key.Binding
	Run      key.Binding
	Refresh  key.Binding
	Filter   key.Binding
	Select   key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch view"),
		),
		Run: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run scenario"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle select"),
		),
	}
}

// ShortHelp returns the short help text.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Run, k.Tab, k.Quit}
}

// FullHelp returns the full help text.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Run, k.Refresh, k.Filter},
		{k.Tab, k.Help, k.Quit},
	}
}

// NewModel creates a new TUI model.
func NewModel(cfg *config.Config, executor *execution.Executor, storage results.Storage) Model {
	styles := DefaultStyles()
	keys := DefaultKeyMap()

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	// Create help
	h := help.New()

	// Create scenario list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(primaryColor).
		BorderLeftForeground(primaryColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(mutedColor).
		BorderLeftForeground(primaryColor)

	scenarioList := list.New([]list.Item{}, delegate, 0, 0)
	scenarioList.Title = "Scenarios"
	scenarioList.SetShowStatusBar(true)
	scenarioList.SetFilteringEnabled(true)
	scenarioList.Styles.Title = styles.Title
	scenarioList.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 1)

	// Create viewport for results
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Padding(1)

	return Model{
		config:          cfg,
		executor:        executor,
		storage:         storage,
		view:            ViewScenarios,
		styles:          styles,
		keys:            keys,
		help:            h,
		spinner:         s,
		scenarioList:    scenarioList,
		resultsViewport: vp,
		executionLogs:   make([]string, 0),
	}
}

// Init initializes the TUI.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadScenarios,
		m.loadResults,
	)
}

// Messages
type scenariosLoadedMsg struct {
	scenarios []*scenario.Scenario
}

type resultsLoadedMsg struct {
	results []*results.RunResult
}

type executionFinishedMsg struct {
	result *execution.ScenarioResult
	err    error
}

type errMsg struct {
	err error
}

func (m Model) loadScenarios() tea.Msg {
	if m.config == nil {
		return scenariosLoadedMsg{scenarios: nil}
	}

	var scenarios []*scenario.Scenario
	for _, scenarioCfg := range m.config.Scenarios {
		s := scenario.NewBuilder(scenarioCfg.Name).
			Description(scenarioCfg.Description)

		for _, tag := range scenarioCfg.Tags {
			s.Tags(tag)
		}

		scenarios = append(scenarios, s.Build())
	}

	return scenariosLoadedMsg{scenarios: scenarios}
}

func (m Model) loadResults() tea.Msg {
	if m.storage == nil {
		return resultsLoadedMsg{results: nil}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultIDs, err := m.storage.List(ctx)
	if err != nil {
		return errMsg{err: err}
	}

	// Fetch each result by ID
	var resultsList []*results.RunResult
	for _, id := range resultIDs {
		result, err := m.storage.Load(ctx, id)
		if err != nil {
			continue // Skip results that can't be loaded
		}
		resultsList = append(resultsList, result)
	}

	return resultsLoadedMsg{results: resultsList}
}

func (m Model) runScenario(s *scenario.Scenario) tea.Cmd {
	return func() tea.Msg {
		if m.executor == nil {
			return executionFinishedMsg{err: fmt.Errorf("no executor configured")}
		}

		ctx := context.Background()
		result := m.executor.Execute(ctx, s)

		return executionFinishedMsg{result: result}
	}
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			// Cycle through views
			switch m.view {
			case ViewScenarios:
				m.view = ViewResults
			case ViewResults:
				m.view = ViewScenarios
			}
			return m, nil

		case key.Matches(msg, m.keys.Back):
			if m.view == ViewExecution && !m.executing {
				m.view = ViewScenarios
				return m, nil
			}
		}

		// Handle view-specific keys
		switch m.view {
		case ViewScenarios:
			if key.Matches(msg, m.keys.Run) || key.Matches(msg, m.keys.Enter) {
				if item, ok := m.scenarioList.SelectedItem().(ScenarioItem); ok {
					m.view = ViewExecution
					m.executing = true
					m.executionLogs = []string{"Starting execution..."}
					return m, tea.Batch(
						m.spinner.Tick,
						m.runScenario(item.scenario),
					)
				}
			}

		case ViewResults:
			// Viewport handles scrolling
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update component sizes
		headerHeight := 3
		helpHeight := 4
		contentHeight := m.height - headerHeight - helpHeight - 4

		m.scenarioList.SetSize(m.width-4, contentHeight)
		m.resultsViewport.Width = m.width - 4
		m.resultsViewport.Height = contentHeight

	case spinner.TickMsg:
		if m.executing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case scenariosLoadedMsg:
		m.scenarios = msg.scenarios
		items := make([]list.Item, len(msg.scenarios))
		for i, s := range msg.scenarios {
			items[i] = ScenarioItem{scenario: s}
		}
		m.scenarioList.SetItems(items)

	case resultsLoadedMsg:
		m.resultsList = msg.results
		m.resultsViewport.SetContent(m.renderResultsList())

	case executionFinishedMsg:
		m.executing = false
		m.currentResult = msg.result
		if msg.err != nil {
			m.err = msg.err
			m.executionLogs = append(m.executionLogs, fmt.Sprintf("Error: %v", msg.err))
		} else {
			m.executionLogs = append(m.executionLogs, "Execution complete!")
		}
		// Reload results
		cmds = append(cmds, m.loadResults)

	case errMsg:
		m.err = msg.err
	}

	// Update child components based on view
	switch m.view {
	case ViewScenarios:
		var cmd tea.Cmd
		m.scenarioList, cmd = m.scenarioList.Update(msg)
		cmds = append(cmds, cmd)

	case ViewResults:
		var cmd tea.Cmd
		m.resultsViewport, cmd = m.resultsViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return "Loading...\n"
	}

	var content string
	switch m.view {
	case ViewScenarios:
		content = m.viewScenarios()
	case ViewExecution:
		content = m.viewExecution()
	case ViewResults:
		content = m.viewResults()
	}

	// Build the full view
	header := m.renderHeader()
	statusBar := m.renderStatusBar()
	helpView := m.help.View(m.keys)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		statusBar,
		helpView,
	)
}

func (m Model) renderHeader() string {
	title := m.styles.Title.Render("Chronicle")
	subtitle := m.styles.Subtitle.Render(" - Test Orchestration Framework")

	var viewIndicator string
	switch m.view {
	case ViewScenarios:
		viewIndicator = " [Scenarios]"
	case ViewExecution:
		viewIndicator = " [Execution]"
	case ViewResults:
		viewIndicator = " [Results]"
	}

	return m.styles.Header.Width(m.width - 4).Render(
		title + subtitle + m.styles.Muted.Render(viewIndicator),
	)
}

func (m Model) renderStatusBar() string {
	var status string
	if m.executing {
		status = m.spinner.View() + " Running..."
	} else if m.err != nil {
		status = m.styles.Error.Render("Error: " + m.err.Error())
	} else {
		status = fmt.Sprintf("%d scenarios | %d results", len(m.scenarios), len(m.resultsList))
	}

	return m.styles.StatusBar.Width(m.width - 4).Render(status)
}

func (m Model) viewScenarios() string {
	return m.scenarioList.View()
}

func (m Model) viewExecution() string {
	var b strings.Builder

	if m.executing {
		b.WriteString(m.spinner.View())
		b.WriteString(" Executing scenario...\n\n")
	} else if m.currentResult != nil {
		icon := StatusIcon(m.currentResult.State.String())
		style := m.styles.StatusStyle(m.currentResult.State.String())
		b.WriteString(style.Render(icon + " " + m.currentResult.ScenarioName))
		b.WriteString("\n")
		b.WriteString(m.styles.Muted.Render(fmt.Sprintf("Duration: %s", m.currentResult.Duration)))
		b.WriteString("\n\n")

		// Show flow results
		for _, fr := range m.currentResult.FlowResults {
			frIcon := StatusIcon(fr.State.String())
			frStyle := m.styles.StatusStyle(fr.State.String())
			b.WriteString(frStyle.Render(fmt.Sprintf("  %s %s [%s]", frIcon, fr.Name, fr.Type)))
			b.WriteString("\n")
			if fr.Error != nil {
				b.WriteString(m.styles.Error.Render(fmt.Sprintf("    Error: %v", fr.Error)))
				b.WriteString("\n")
			}
		}
	}

	// Show logs
	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("Execution Log:"))
	b.WriteString("\n")
	for _, log := range m.executionLogs {
		b.WriteString("  " + log + "\n")
	}

	return b.String()
}

func (m Model) viewResults() string {
	return m.resultsViewport.View()
}

func (m Model) renderResultsList() string {
	if len(m.resultsList) == 0 {
		return m.styles.Muted.Render("No results found. Run some scenarios first!")
	}

	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Recent Results"))
	b.WriteString("\n\n")

	for _, r := range m.resultsList {
		// Result header
		icon := "●"
		style := m.styles.Success
		if r.Stats.Failed > 0 {
			icon = "✗"
			style = m.styles.Error
		}

		header := style.Render(fmt.Sprintf("%s %s", icon, r.Name))
		b.WriteString(header)
		b.WriteString("\n")

		// Stats
		stats := fmt.Sprintf("  Passed: %d | Failed: %d | Skipped: %d",
			r.Stats.Passed, r.Stats.Failed, r.Stats.Skipped)
		b.WriteString(m.styles.Muted.Render(stats))
		b.WriteString("\n")

		// Duration and time
		meta := fmt.Sprintf("  Duration: %s | Started: %s",
			r.Duration.Round(time.Millisecond),
			r.StartTime.Format("2006-01-02 15:04:05"))
		b.WriteString(m.styles.Muted.Render(meta))
		b.WriteString("\n\n")
	}

	return b.String()
}

// Run starts the TUI application.
func Run(cfg *config.Config, executor *execution.Executor, storage results.Storage) error {
	model := NewModel(cfg, executor, storage)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
