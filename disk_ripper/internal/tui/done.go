package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type cleanupDoneMsg struct{}

type doneModel struct {
	movieName   string
	localPath   string
	localDir    string
	uploadErr   error
	flow        Flow
	cleanup     *bool
	cleanupForm *huh.Form
	cleaned     bool
}

func newDoneModel(localDir, movieName, localPath string, uploadErr error, flow Flow) (doneModel, tea.Cmd) {
	cleanup := false
	d := doneModel{
		movieName: movieName,
		localPath: localPath,
		localDir:  localDir,
		uploadErr: uploadErr,
		flow:      flow,
		cleanup:   &cleanup,
	}
	d.cleanupForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Delete local temp files?").
				Value(d.cleanup),
		),
	).WithShowHelp(true).WithTheme(formTheme())
	return d, d.cleanupForm.Init()
}

func doCleanup(dir string) tea.Cmd {
	return func() tea.Msg {
		_ = os.RemoveAll(dir)
		return cleanupDoneMsg{}
	}
}

func (m Model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case cleanupDoneMsg:
		m.done.cleaned = true
		return m, nil
	}

	if m.done.cleaned {
		if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "q" || key.String() == "enter") {
			m.state = StateMainMenu
		}
		return m, nil
	}

	form, cmd := m.done.cleanupForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.done.cleanupForm = f
	}

	switch m.done.cleanupForm.State {
	case huh.StateCompleted:
		if *m.done.cleanup && m.done.localDir != "" {
			return m, doCleanup(m.done.localDir)
		}
		m.done.cleaned = true
	case huh.StateAborted:
		m.done.cleaned = true
	}

	return m, cmd
}

func (m Model) viewDone() string {
	d := m.done

	var sb strings.Builder

	if d.uploadErr != nil {
		sb.WriteString(errorStyle.Render("  Upload failed: "+d.uploadErr.Error()) + "\n\n")
	} else if d.flow != Ripping {
		sb.WriteString(successStyle.Render("  Upload complete!") + "\n\n")
	} else {
		sb.WriteString(successStyle.Render("  Rip complete!") + "\n\n")
	}

	if d.movieName != "" {
		sb.WriteString(fmt.Sprintf("  Movie:  %s\n", labelStyle.Render(d.movieName)))
	}
	if d.localPath != "" {
		sb.WriteString(fmt.Sprintf("  File:   %s\n", sublabelStyle.Render(filepath.Base(d.localPath))))
	}
	if d.localDir != "" {
		sb.WriteString(fmt.Sprintf("  Dir:    %s\n", sublabelStyle.Render(d.localDir)))
	}

	sb.WriteString("\n")

	if d.cleaned {
		sb.WriteString(dimStyle.Render("  press q or enter to return to menu"))
	} else {
		sb.WriteString(d.cleanupForm.View())
	}

	header := titleStyle.Render("  Done  ")
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(sb.String()),
	)
}
