package tui

import "github.com/charmbracelet/lipgloss"

// Palette
var (
	colorPrimary = lipgloss.Color("#7C3AED") // violet
	colorSuccess = lipgloss.Color("#10B981") // emerald
	colorError   = lipgloss.Color("#EF4444") // red
	colorMuted   = lipgloss.Color("#6B7280") // gray
	colorText    = lipgloss.Color("#F9FAFB") // near-white
	colorSubtext = lipgloss.Color("#9CA3AF") // light gray
	colorBorder  = lipgloss.Color("#4B5563") // dark gray
)

// Reusable styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
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
)
