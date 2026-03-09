package tui

import (
	"ripper/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type configEditorFields struct {
	apiKey     string
	device     string
	outputDir  string
	sftpHost   string
	sftpPort   string
	sftpUser   string
	sftpKey    string
	sftpRemote string
}

type configEditorModel struct {
	form   *huh.Form
	fields *configEditorFields
}

func newConfigEditorModel(cfg config.Config) (configEditorModel, tea.Cmd) {
	f := &configEditorFields{
		apiKey:     cfg.TMDB.APIKey,
		device:     cfg.Drive.Device,
		outputDir:  cfg.Output.Dir,
		sftpHost:   cfg.SFTP.Host,
		sftpPort:   cfg.SFTP.Port,
		sftpUser:   cfg.SFTP.User,
		sftpKey:    cfg.SFTP.KeyPath,
		sftpRemote: cfg.SFTP.RemotePath,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("API Key").Value(&f.apiKey).EchoMode(huh.EchoModePassword),
		).Title("TMDB"),
		huh.NewGroup(
			huh.NewInput().Title("Device").Value(&f.device),
		).Title("Drive"),
		huh.NewGroup(
			huh.NewInput().Title("Local Dir").Value(&f.outputDir),
		).Title("Output"),
		huh.NewGroup(
			huh.NewInput().Title("Host").Value(&f.sftpHost),
			huh.NewInput().Title("Port").Value(&f.sftpPort),
			huh.NewInput().Title("User").Value(&f.sftpUser),
			huh.NewInput().Title("Key Path").Value(&f.sftpKey),
			huh.NewInput().Title("Remote Path").Value(&f.sftpRemote),
		).Title("SFTP"),
	).WithShowHelp(true)

	return configEditorModel{form: form, fields: f}, form.Init()
}

func (ce configEditorModel) toConfig(base config.Config) config.Config {
	base.TMDB.APIKey = ce.fields.apiKey
	base.Drive.Device = ce.fields.device
	base.Output.Dir = ce.fields.outputDir
	base.SFTP.Host = ce.fields.sftpHost
	base.SFTP.Port = ce.fields.sftpPort
	base.SFTP.User = ce.fields.sftpUser
	base.SFTP.KeyPath = ce.fields.sftpKey
	base.SFTP.RemotePath = ce.fields.sftpRemote
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
