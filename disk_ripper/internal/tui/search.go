package tui

import (
	"fmt"
	"path/filepath"
	"strconv"

	"ripper/internal/tmdb"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type searchStep int

const (
	searchStepInput searchStep = iota
	searchStepLoading
	searchStepConfirm
)

type tmdbSearchDoneMsg struct {
	result *tmdb.SearchResult
	err    error
}

type searchModel struct {
	step        searchStep
	form        *huh.Form
	spinner     spinner.Model
	query       *string
	result      *tmdb.SearchResult
	err         error
	confirmForm *huh.Form
	editName    *string
	editYear    *string
	editId      *string
}

func newSearchModel() (searchModel, tea.Cmd) {
	query := ""
	s := searchModel{query: &query}
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Movie title").
				Placeholder("e.g. The Goonies").
				Value(s.query),
		),
	).WithShowHelp(true).WithTheme(formTheme())

	sp := spinner.New()
	sp.Spinner = spinner.Moon
	s.spinner = sp

	return s, s.form.Init()
}

func doTMDBSearch(apiKey, title string) tea.Cmd {
	return func() tea.Msg {
		result, err := tmdb.Search(apiKey, title)
		return tmdbSearchDoneMsg{result: result, err: err}
	}
}

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.search.step {
	case searchStepInput:
		return m.updateSearchInput(msg)
	case searchStepLoading:
		return m.updateSearchLoading(msg)
	case searchStepConfirm:
		return m.updateSearchConfirm(msg)
	}
	return m, nil
}

func (m Model) updateSearchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.search.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.search.form = f
	}

	switch m.search.form.State {
	case huh.StateCompleted:
		m.search.step = searchStepLoading
		return m, tea.Batch(
			m.search.spinner.Tick,
			doTMDBSearch(m.cfg.TMDB.APIKey, *m.search.query),
		)
	case huh.StateAborted:
		m.state = StateMainMenu
		return m, nil
	}

	return m, cmd
}

func (m Model) updateSearchLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tmdbSearchDoneMsg:
		if msg.err != nil {
			// reset to input with error
			sm, cmd := newSearchModel()
			sm.err = msg.err
			m.search = sm
			return m, cmd
		}
		editName := tmdb.SanitizeFilename(msg.result.Title)
		editYear := tmdb.ExtractYear(msg.result.ReleaseDate)
		editId := strconv.Itoa(msg.result.ID)
		m.search.result = msg.result
		m.search.editName = &editName
		m.search.editYear = &editYear
		m.search.editId = &editId

		m.search.confirmForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Movie name").Value(m.search.editName),
				huh.NewInput().Title("Year").Value(m.search.editYear),
				huh.NewInput().Title("TMBD ID").Value(m.search.editId),
			),
		).WithShowHelp(true).WithTheme(formTheme())

		m.search.step = searchStepConfirm
		return m, m.search.confirmForm.Init()

	default:
		var cmd tea.Cmd
		m.search.spinner, cmd = m.search.spinner.Update(msg)
		return m, cmd
	}
}

func (m Model) updateSearchConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.search.confirmForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.search.confirmForm = f
	}

	switch m.search.confirmForm.State {
	case huh.StateCompleted:
		editId, err := strconv.Atoi(*m.search.editId)
		if err != nil {
			m.err = err
			m.id = 0
		} else {
			m.id = editId
		}
		m.movieName = *m.search.editName
		m.year = *m.search.editYear
		m.folderName = *m.search.editName + " (" + *m.search.editYear + ") [tmdbid-" + *m.search.editId + "]"
		m.outputDir = filepath.Join(m.cfg.Output.Dir, m.folderName)

		if !m.fullPipeline {
			lc, lcCmd := newLocalConfirmModel(m.outputDir)
			m.localConfirm = lc
			m.state = StateLocalOnlyConfirm
			return m, lcCmd
		}

		rip, ripCmd := newRippingModel(m.cfg.Drive.Device, m.outputDir)
		m.ripping = rip
		m.state = StateRipping
		return m, ripCmd

	case huh.StateAborted:
		sm, initCmd := newSearchModel()
		m.search = sm
		return m, initCmd
	}

	return m, cmd
}

func (m Model) viewSearch() string {
	header := titleStyle.Render("  TMDB Search  ")

	var content string
	switch m.search.step {
	case searchStepInput:
		errLine := ""
		if m.search.err != nil {
			errLine = errorStyle.Render("  "+m.search.err.Error()) + "\n\n"
		}
		content = errLine + m.search.form.View()

	case searchStepLoading:
		content = fmt.Sprintf("\n  %s Searching TMDB…\n", m.search.spinner.View())

	case searchStepConfirm:
		result := m.search.result
		id := strconv.Itoa(result.ID)
		info := fmt.Sprintf("  Found: %s  %s %s\n\n  Edit if needed, then confirm:\n\n",
			successStyle.Render(result.Title),
			sublabelStyle.Render(tmdb.ExtractYear(result.ReleaseDate)),
			sublabelStyle.Render(id),
		)
		content = info + m.search.confirmForm.View()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(content),
	)
}
