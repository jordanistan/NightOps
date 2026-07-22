// Package console implements the Bubble Tea application shell.
package console

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenStartup screen = iota
	screenOrigin
	screenHome
	screenSettings
)

// Model is the top-level Bubble Tea state machine.
type Model struct {
	screen   screen
	spinner  spinner.Model
	theme    Theme
	width    int
	height   int
	origin   string
	started  time.Time
	quitting bool
}

type startupDoneMsg struct{}

// New creates the NightOps console model.
func New(theme Theme) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Model{screen: screenStartup, spinner: s, theme: theme, started: time.Now()}
}

// Init starts the startup animation.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return startupDoneMsg{} })
}

// Update handles keyboard input and application messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case startupDoneMsg:
		m.screen = screenOrigin
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q"))) {
			m.quitting = true
			return m, tea.Quit
		}
		switch m.screen {
		case screenOrigin:
			switch msg.String() {
			case "1", "h":
				m.origin, m.screen = "Home Base", screenHome
			case "2", "z":
				m.origin, m.screen = "ZIP Code", screenHome
			case "3", "g":
				m.origin, m.screen = "Current GPS", screenHome
			}
		case screenHome:
			if msg.String() == "s" {
				m.screen = screenSettings
			}
		case screenSettings:
			if msg.String() == "esc" {
				m.screen = screenHome
			}
		}
	}
	return m, nil
}

// View renders the active console.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Accent).Render("NIGHTOPS // MISSION CONTROL")
	footer := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("q quit  •  s settings")
	switch m.screen {
	case screenStartup:
		return lipgloss.JoinVertical(lipgloss.Left, "", title, "", m.spinner.View()+"  Initializing flight systems...", "", footer)
	case screenOrigin:
		return lipgloss.JoinVertical(lipgloss.Left, "", title, "", "MISSION ORIGIN", "", "[1] Home Base", "[2] ZIP Code", "[3] Current GPS", "", lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Choose the launch site for this operation."), "", footer)
	case screenSettings:
		return lipgloss.JoinVertical(lipgloss.Left, "", title, "", "SETTINGS // SYSTEM CONFIGURATION", "", "Configuration is loaded from YAML.", "", "esc return", "", footer)
	default:
		return lipgloss.JoinVertical(lipgloss.Left, "", title, "", "HOME BASE // "+m.origin, "", "No mission is active.", "", "[s] Settings", "", footer)
	}
}
