package tui

import (
	"strconv"

	types "ripper/internal"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type titleItem struct {
	id          int
	title, desc string
}

func (i titleItem) Title() string       { return i.title }
func (i titleItem) Description() string { return i.desc }
func (i titleItem) FilterValue() string { return i.title }

type titleSelectModel struct {
	list list.Model
}

func newTitleSelectModel(titles []types.TitleInfo, width, height int) (titleSelectModel, tea.Cmd) {

	items := make([]list.Item, len(titles))
	for i, f := range titles {
		lenMin, err := strconv.Atoi(f.Duration)
		if err != nil {
			seconds := lenMin % 60
			minutes := (lenMin - seconds) / 60
			items[i] = titleItem{
				id:    f.ID,
				title: f.Name,
				desc:  f.SizeHuman + " | Duration: " + strconv.Itoa(minutes) + ":" + strconv.Itoa(seconds),
			}
		} else {
			items[i] = titleItem{
				id:    f.ID,
				title: f.Name,
				desc:  f.SizeHuman,
			}
		}

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
	l.Title = "Select Title"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)

	return titleSelectModel{list: l}, nil
}

func (m Model) updateTitleSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.titleSelect.list.SetSize(msg.Width-6, msg.Height-8)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if item, ok := m.titleSelect.list.SelectedItem().(titleItem); ok {
				selectedTitle := strconv.Itoa(item.id)
				m.selectedTitle = selectedTitle
				sm, cmd := newSearchModel(item.title)
				m.search = sm
				m.state = StateTMDBConfirm
				return m, cmd
			}
		}
	}

	var cmd tea.Cmd
	m.titleSelect.list, cmd = m.titleSelect.list.Update(msg)
	return m, cmd
}

func (m Model) viewTitleSelect() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(m.titleSelect.list.View()),
		"",
		helpStyle.Render("  ↑/↓  navigate   enter  select   /  filter"),
	)
}
