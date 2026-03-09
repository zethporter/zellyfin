package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mkvItem struct {
	path string
	size int64
}

func (i mkvItem) Title() string       { return filepath.Base(i.path) }
func (i mkvItem) Description() string { return fmt.Sprintf("%.1f MB", float64(i.size)/1024/1024) }
func (i mkvItem) FilterValue() string { return filepath.Base(i.path) }

type fileSelectModel struct {
	list list.Model
}

func newFileSelectModel(files []string, width, height int) (fileSelectModel, tea.Cmd) {
	items := make([]list.Item, len(files))
	for i, f := range files {
		var sz int64
		if info, err := os.Stat(f); err == nil {
			sz = info.Size()
		}
		items[i] = mkvItem{path: f, size: sz}
	}

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

	l := list.New(items, delegate, w, h)
	l.Title = "Select MKV File"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)

	return fileSelectModel{list: l}, nil
}

func (m Model) updateFileSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m Model) viewFileSelect() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(m.fileSelect.list.View()),
		"",
		helpStyle.Render("  ↑/↓  navigate   enter  select   /  filter"),
	)
}
