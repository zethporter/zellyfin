package tui

import (
	"ripper/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type configEditorFields struct {
	apiKey    string
	device    string
	outputDir string
	tempDir   string
}

type configEditorModel struct {
	form   *huh.Form
	fields *configEditorFields
}

func newConfigEditorModel(cfg config.Config) (configEditorModel, tea.Cmd) {
	f := &configEditorFields{
		apiKey:    cfg.TMDB.APIKey,
		device:    cfg.Drive.Device,
		outputDir: cfg.Output.Dir,
		tempDir:   cfg.Output.TempDir,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("API Key").Value(&f.apiKey).EchoMode(huh.EchoModePassword),
		).Title("TMDB"),
		huh.NewGroup(
			huh.NewInput().Title("Device").Value(&f.device),
		).Title("Drive"),
		huh.NewGroup(
			huh.NewInput().Title("Movie Dir").Value(&f.outputDir),
			huh.NewInput().Title("Temp Dir").Value(&f.tempDir),
		).Title("Output"),
	).WithShowHelp(true).WithTheme(formTheme())

	return configEditorModel{form: form, fields: f}, form.Init()
}

func (ce configEditorModel) toConfig(base config.Config) config.Config {
	base.TMDB.APIKey = ce.fields.apiKey
	base.Drive.Device = ce.fields.device
	base.Output.Dir = ce.fields.outputDir
	return base
}

func (m Model) updateConfigEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.configEditor.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.configEditor.form = f
	}

	switch m.configEditor.form.State {
	case huh.StateCompleted:
		m.cfg = m.configEditor.toConfig(m.cfg)
		_ = config.Save(m.cfgPath, m.cfg)
		m.state = StateMainMenu
		return m, nil
	case huh.StateAborted:
		m.state = StateMainMenu
		return m, nil
	}

	return m, cmd
}

func (m Model) viewConfigEditor() string {
	header := titleStyle.Render("  Edit Configuration  ")
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(m.configEditor.form.View()),
	)
}
