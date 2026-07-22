package console

import "github.com/charmbracelet/lipgloss"

// Theme defines the visual language for NightOps consoles.
type Theme struct {
	Accent               lipgloss.Color
	Muted                lipgloss.Color
	Surface              lipgloss.Color
	Text                 lipgloss.Color
	Good                 lipgloss.Color
	Warning              lipgloss.Color
	Error                lipgloss.Color
	Wordmark             lipgloss.Style
	Subtitle             lipgloss.Style
	Motto                lipgloss.Style
	PanelTitle           lipgloss.Style
	Panel                lipgloss.Style
	Status               lipgloss.Style
	Action               lipgloss.Style
	SelectedAction       lipgloss.Style
	LaunchAction         lipgloss.Style
	SelectedLaunchAction lipgloss.Style
	Footer               lipgloss.Style
}

// MissionControlTheme is the default near-black command-console palette.
func MissionControlTheme() Theme {
	theme := Theme{
		Accent:  lipgloss.Color("75"),
		Muted:   lipgloss.Color("244"),
		Surface: lipgloss.Color("235"),
		Text:    lipgloss.Color("252"),
		Good:    lipgloss.Color("42"),
		Warning: lipgloss.Color("214"),
		Error:   lipgloss.Color("196"),
	}
	return styleTheme(theme)
}

// ObservatoryTheme is a slightly brighter slate-and-cyan palette for dark observatory rooms.
func ObservatoryTheme() Theme {
	theme := Theme{
		Accent:  lipgloss.Color("81"),
		Muted:   lipgloss.Color("245"),
		Surface: lipgloss.Color("238"),
		Text:    lipgloss.Color("255"),
		Good:    lipgloss.Color("48"),
		Warning: lipgloss.Color("220"),
		Error:   lipgloss.Color("203"),
	}
	return styleTheme(theme)
}

// ThemeForName returns a configured supported theme, defaulting to mission control.
func ThemeForName(name string) Theme {
	if name == "observatory" {
		return ObservatoryTheme()
	}
	return MissionControlTheme()
}

func styleTheme(theme Theme) Theme {
	theme.Wordmark = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent).Align(lipgloss.Center)
	theme.Subtitle = lipgloss.NewStyle().Foreground(theme.Muted).Align(lipgloss.Center)
	theme.Motto = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("248")).Align(lipgloss.Center)
	theme.PanelTitle = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	theme.Panel = lipgloss.NewStyle().Foreground(theme.Text).Border(lipgloss.RoundedBorder()).BorderForeground(theme.Surface).Padding(1, 2)
	theme.Status = lipgloss.NewStyle().Foreground(theme.Text).BorderLeft(true).BorderForeground(theme.Surface).PaddingLeft(2)
	theme.Action = lipgloss.NewStyle().Foreground(theme.Text).Padding(0, 1)
	theme.SelectedAction = lipgloss.NewStyle().Foreground(theme.Text).Background(theme.Surface).BorderLeft(true).BorderForeground(theme.Accent).PaddingLeft(1)
	theme.LaunchAction = lipgloss.NewStyle().Bold(true).Foreground(theme.Good).BorderTop(true).BorderForeground(theme.Good).Padding(1, 1)
	theme.SelectedLaunchAction = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(theme.Good).BorderTop(true).BorderForeground(theme.Good).Padding(1, 1)
	theme.Footer = lipgloss.NewStyle().Foreground(theme.Muted).Align(lipgloss.Center)
	return theme
}

// AccentStyle returns the style used for active system information.
func (t Theme) AccentStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Accent) }

// GoodStyle returns the style used for healthy system information.
func (t Theme) GoodStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Good) }

// WarningStyle returns the style used for unavailable or pending systems.
func (t Theme) WarningStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Warning) }

// ErrorStyle returns the style used for actionable failures.
func (t Theme) ErrorStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Error) }

// MutedStyle returns the style used for secondary information.
func (t Theme) MutedStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Muted) }
