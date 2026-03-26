package tui

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Palette (ANSI – used for lipgloss styles)
var (
	colorPrimary = lipgloss.Color("5") // violet
	colorSuccess = lipgloss.Color("2") // emerald
	colorError   = lipgloss.Color("1") // red
	colorMuted   = lipgloss.Color("4") // gray
	colorText    = lipgloss.Color("7") // near-white
	colorSubtext = lipgloss.Color("4") // light gray
	colorBorder  = lipgloss.Color("3") // dark gray
	colorBlack   = lipgloss.Color("0") // black
)

// Hex equivalents for components that require true-color strings (e.g. progress bars).
const (
	hexPrimary = "#af5fd7" // violet  (~ANSI 95)
	hexMuted   = "#5f87af" // steel   (~ANSI 67)
)

// Reusable styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Padding(0, 2)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 3)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	sublabelStyle = lipgloss.NewStyle().
			Foreground(colorSubtext)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorText)
	unselectedSubLabelStyle = lipgloss.NewStyle().
				Foreground(colorBlack)
)

// newProgressBar returns a progress.Model styled with the app palette.
func newProgressBar() progress.Model {
	return progress.New(progress.WithGradient(hexMuted, hexPrimary))
}

// formTheme returns a huh theme using the app palette.
func formTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Title = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(colorSubtext)
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(colorPrimary)
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(colorText)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(colorMuted)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(colorError)
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(colorError)
	t.Focused.FocusedButton = lipgloss.NewStyle().Foreground(colorText).Background(colorPrimary).Padding(0, 1)
	t.Focused.BlurredButton = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(colorPrimary)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(colorPrimary)

	t.Blurred.Title = lipgloss.NewStyle().Foreground(colorMuted)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(colorSubtext)
	t.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(colorSubtext)
	t.Blurred.TextInput.Placeholder = lipgloss.NewStyle().Foreground(colorMuted)
	t.Blurred.FocusedButton = lipgloss.NewStyle().Foreground(colorText).Background(colorPrimary).Padding(0, 1)
	t.Blurred.BlurredButton = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)

	return t
}
