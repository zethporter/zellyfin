package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem struct {
	label    string
	sublabel string
}

type menuModel struct {
	cursor int
	items  []menuItem
}

func newMenuModel() menuModel {
	return menuModel{
		items: []menuItem{
			{"Full pipeline", "TMDB metadata + rip + upload to server"},
			{"Local rip only", "rip to local drive, no upload"},
			{"Edit config", "modify connection and path settings"},
			{"Quit", ""},
		},
	}
}

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		if m.menu.cursor > 0 {
			m.menu.cursor--
		} else {
			m.menu.cursor = len(m.menu.items) - 1
		}
	case "down", "j":
		if m.menu.cursor < len(m.menu.items)-1 {
			m.menu.cursor++
		} else {
			m.menu.cursor = 0
		}
	case "enter", " ":
		return m.selectMenuItem()
	case "1":
		m.menu.cursor = 0
		return m.selectMenuItem()
	case "2":
		m.menu.cursor = 1
		return m.selectMenuItem()
	case "3":
		m.menu.cursor = 2
		return m.selectMenuItem()
	case "q", "4":
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) selectMenuItem() (tea.Model, tea.Cmd) {
	switch m.menu.cursor {
	case 0: // Full pipeline
		m.fullPipeline = true
		sm, cmd := newSearchModel()
		m.search = sm
		m.state = StateTMDBSearch
		return m, cmd
	case 1: // Local rip only
		m.fullPipeline = false
		sm, cmd := newSearchModel()
		m.search = sm
		m.state = StateTMDBSearch
		return m, cmd
	case 2: // Edit config
		ce, cmd := newConfigEditorModel(m.cfg)
		m.configEditor = ce
		m.state = StateConfigEditor
		return m, cmd
	case 3: // Quit
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewMenu() string {
	header := titleStyle.Render("  Jellyfin Disk Ripper  ")

	var sb strings.Builder
	sb.WriteString(helpStyle.Render("What would you like to do?"))
	sb.WriteString("\n\n")

	const labelWidth = 20

	for i, item := range m.menu.items {
		var row strings.Builder

		if i == m.menu.cursor {
			row.WriteString(selectedItemStyle.Render("▸ "))
			row.WriteString(selectedItemStyle.Width(labelWidth).Render(item.label))
			if item.sublabel != "" {
				row.WriteString(sublabelStyle.Render(item.sublabel))
			}
		} else {
			row.WriteString("  ")
			row.WriteString(labelStyle.Width(labelWidth).Render(item.label))
			if item.sublabel != "" {
				row.WriteString(sublabelStyle.Render(item.sublabel))
			}
		}

		sb.WriteString(row.String())
		if i < len(m.menu.items)-1 {
			sb.WriteString("\n")
		}
	}

	box := boxStyle.Render(sb.String())
	help := helpStyle.Render("↑/↓  j/k  navigate   enter  select   1-4  shortcut   q  quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(box),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(help),
	)
}
