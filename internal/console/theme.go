package console

import "github.com/charmbracelet/lipgloss"

// Theme defines the visual language for NightOps consoles.
type Theme struct {
	Accent  lipgloss.Color
	Muted   lipgloss.Color
	Surface lipgloss.Color
	Text    lipgloss.Color
	Good    lipgloss.Color
	Warning lipgloss.Color
}

// MissionControlTheme is the default dark command-console palette.
func MissionControlTheme() Theme {
	return Theme{Accent: lipgloss.Color("205"), Muted: lipgloss.Color("244"), Surface: lipgloss.Color("235"), Text: lipgloss.Color("255"), Good: lipgloss.Color("42"), Warning: lipgloss.Color("214")}
}
