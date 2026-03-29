package tui

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


func (i TitleInfo) Title() string       { return filepath.Base(i.) }
func (i TitleInfo) Description() string { return fmt.Sprintf("%.1f MB", float64(i.size)/1024/1024) }
func (i TitleInfo) FilterValue() string { return filepath.Base(i.path) }

type titleSelectModel struct {
	list list.Model
}

func newTitleSelectModel(titles []TitleInfo, width, height int) (titleSelectModel, tea.Cmd) {


	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(colorPrimary).
		BorderForeground(colorPrimary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(colorSubtext).
		BorderForeground(colorPrimary)

	w := width - 6
	h := height - 8
	if w < 20 {
		w = 60
	}
	if h < 5 {
		h = 10
	}

	l := list.New(titles, delegate, w, h)
	l.Title = "Select Title"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)

	return titleSelectModel{list: l}, nil
}

func (m Model) updateTitleSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.fileSelect.list.SetSize(msg.Width-6, msg.Height-8)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if item, ok := m.fileSelect.list.SelectedItem().(mkvItem); ok {
				m.mainMKV = item.path
				return m.transitionAfterFileSelect()
			}
		}
	}

	var cmd tea.Cmd
	m.fileSelect.list, cmd = m.fileSelect.list.Update(msg)
	return m, cmd
}

func (m Model) viewTitleSelect() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(m.fileSelect.list.View()),
		"",
		helpStyle.Render("  ↑/↓  navigate   enter  select   /  filter"),
	)
}
