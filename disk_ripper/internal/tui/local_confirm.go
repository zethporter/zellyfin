package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type localConfirmFields struct {
	outputDir string
}

type localConfirmModel struct {
	form   *huh.Form
	fields *localConfirmFields
}

func newLocalConfirmModel(defaultDir string) (localConfirmModel, tea.Cmd) {
	f := &localConfirmFields{outputDir: defaultDir}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Output directory").
				Description("Where to save the ripped MKV files").
				Value(&f.outputDir),
		),
	).WithShowHelp(true).WithTheme(formTheme())
	lc := localConfirmModel{form: form, fields: f}
	return lc, form.Init()
}

func (m Model) updateLocalConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.localConfirm.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.localConfirm.form = f
	}

	switch m.localConfirm.form.State {
	case huh.StateCompleted:
		m.outputDir = m.localConfirm.fields.outputDir
		rip, ripCmd := newRippingModel(m.cfg.Drive.Device, "all", m.outputDir)
		m.ripping = rip
		m.state = StateRipping
		return m, ripCmd
	case huh.StateAborted:
		sm, initCmd := newSearchModel("")
		m.search = sm
		m.state = StateTMDBSearch
		return m, initCmd
	}

	return m, cmd
}

func (m Model) viewLocalConfirm() string {
	header := titleStyle.Render("  Local Rip — Set Output  ")
	info := sublabelStyle.Render("  Movie: "+m.folderName) + "\n\n"
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(info+m.localConfirm.form.View()),
	)
}
