package tui

import (
	"fmt"
	"os"

	types "ripper/internal"
	"ripper/internal/ripper"
	"ripper/internal/transfer"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Ripping ───────────────────────────────────────────────────────────────────

type ripStartedMsg struct {
	progressCh <-chan int
	errCh      <-chan error
}

type ripProgressMsg struct{ pct int }
type ripDoneMsg struct{ err error }

type rippingModel struct {
	bar        progress.Model
	pct        int
	progressCh <-chan int
	errCh      <-chan error
	tempDir    string
}

func newRippingModel(device, selectedTitle, tempDir string) (rippingModel, tea.Cmd) {
	rm := rippingModel{
		bar:     newProgressBar(),
		tempDir: tempDir,
	}
	cmd := func() tea.Msg {
		progressCh := make(chan int, 20)
		errCh := make(chan error, 1)
		go func() {
			errCh <- ripper.RipDisc(device, selectedTitle, tempDir, progressCh)
		}()
		return ripStartedMsg{progressCh: progressCh, errCh: errCh}
	}
	return rm, cmd
}

func pollRip(progressCh <-chan int, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case pct, ok := <-progressCh:
			if !ok {
				return ripDoneMsg{}
			}
			return ripProgressMsg{pct: pct}
		case err := <-errCh:
			return ripDoneMsg{err: err}
		}
	}
}

func (m Model) updateRipping(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ripStartedMsg:
		m.ripping.progressCh = msg.progressCh
		m.ripping.errCh = msg.errCh
		return m, pollRip(msg.progressCh, msg.errCh)

	case ripProgressMsg:
		m.ripping.pct = msg.pct
		return m, pollRip(m.ripping.progressCh, m.ripping.errCh)

	case ripDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = StateDone
			dm, cmd := newDoneModel(m.ripping.tempDir, m.movieName, m.mainMKV, msg.err, m.flow)
			m.done = dm
			return m, cmd
		}
		files, err := ripper.FindMKVFiles(m.ripping.tempDir)
		if err != nil || len(files) == 0 {
			ripErr := fmt.Errorf("no MKV files found in %s", m.ripping.tempDir)
			m.err = ripErr
			dm, cmd := newDoneModel(m.ripping.tempDir, m.movieName, "", ripErr, m.flow)
			m.done = dm
			m.state = StateDone
			return m, cmd
		}
		if len(files) == 1 {
			m.mainMKV = files[0]
			return m.transitionAfterFileSelect()
		}
		fs, cmd := newFileSelectModel(files, m.width, m.height)
		m.fileSelect = fs
		m.state = StateFileSelect
		return m, cmd
	}
	return m, nil
}

func (m Model) viewRipping() string {
	header := titleStyle.Render("  Ripping Disc  ")
	bar := m.ripping.bar.ViewAs(float64(m.ripping.pct) / 100)
	content := fmt.Sprintf(
		"  Ripping to:\n  %s\n\n  %s\n  %s\n",
		sublabelStyle.Render(m.ripping.tempDir),
		bar,
		dimStyle.Render(fmt.Sprintf("%d%%", m.ripping.pct)),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(content),
		"",
		helpStyle.Render("  ctrl+c  abort"),
	)
}

// ── Fetching Titles ───────────────────────────────────────────────────────────────────

type fetchStartedMsg struct {
	progressCh <-chan types.FetchingProgress
	doneCh     <-chan fetchDoneMsg
}

type fetchProgressMsg struct {
	pct    int
	titles map[int]types.TitleInfo
}
type fetchDoneMsg struct {
	titles []types.TitleInfo
	err    error
}

type fetchingModel struct {
	bar        progress.Model
	pct        int
	progressCh <-chan types.FetchingProgress
	lastTitles map[int]types.TitleInfo
	doneCh     <-chan fetchDoneMsg
}

func newTitleFetchModel(device string) (fetchingModel, tea.Cmd) {
	fm := fetchingModel{
		bar: newProgressBar(),
	}

	cmd := func() tea.Msg {
		progressCh := make(chan types.FetchingProgress, 100)
		doneCh := make(chan fetchDoneMsg, 1)

		go func() {
			titles, err := ripper.FindTitles(device, progressCh)
			doneCh <- fetchDoneMsg{titles: titles, err: err}
		}()

		return fetchStartedMsg{
			progressCh: progressCh,
			doneCh:     doneCh,
		}
	}

	return fm, cmd
}

func pollFetch(
	progressCh <-chan types.FetchingProgress,
	doneCh <-chan fetchDoneMsg,
) tea.Cmd {
	return func() tea.Msg {
		for progressCh != nil || doneCh != nil {
			select {
			case p, ok := <-progressCh:
				if !ok {
					progressCh = nil
					continue
				}
				return fetchProgressMsg{
					pct:    p.Pct,
					titles: p.Titles,
				}

			case done, ok := <-doneCh:
				if !ok {
					doneCh = nil
					continue
				}
				return done
			}
		}
		return nil
	}
}

func (m Model) updateFetching(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchStartedMsg:
		m.fetching.progressCh = msg.progressCh
		m.fetching.doneCh = msg.doneCh
		return m, pollFetch(msg.progressCh, msg.doneCh)

	case fetchProgressMsg:
		m.fetching.pct = msg.pct
		m.fetching.lastTitles = msg.titles
		return m, pollFetch(m.fetching.progressCh, m.fetching.doneCh)

	case fetchDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = StateDone
			dm, cmd := newDoneModel("", m.movieName, m.mainMKV, msg.err, m.flow)
			m.done = dm
			return m, cmd
		}

		m.diskTitles = msg.titles
		ts, cmd := newTitleSelectModel(m.diskTitles, m.width, m.height)
		m.titleSelect = ts
		m.state = StateTitleSelect
		return m, cmd
	}

	return m, nil
}

func (m Model) viewFetching() string {
	header := titleStyle.Render("  Fetching Titles  ")
	bar := m.fetching.bar.ViewAs(float64(m.fetching.pct) / 100)
	content := fmt.Sprintf(
		"%s %s",
		bar,
		dimStyle.Render(fmt.Sprintf("%d%%", m.fetching.pct)),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(content),
		"",
		helpStyle.Render("  ctrl+c  abort"),
	)
}

// ── Uploading ─────────────────────────────────────────────────────────────────

type uploadStartedMsg struct {
	progressCh <-chan int64
	errCh      <-chan error
	totalBytes int64
}

type uploadProgressMsg struct{ bytes int64 }
type uploadDoneMsg struct{ err error }

type uploadModel struct {
	bar        progress.Model
	bytes      int64
	totalBytes int64
	progressCh <-chan int64
	errCh      <-chan error
	remotePath string
}

func startUploadCmd(localFile, remotePath string) tea.Cmd {
	return func() tea.Msg {
		info, err := os.Stat(localFile)
		if err != nil {
			return uploadDoneMsg{err: err}
		}
		progressCh := make(chan int64, 20)
		errCh := make(chan error, 1)
		go func() {
			errCh <- transfer.Upload(localFile, remotePath, progressCh)
		}()
		return uploadStartedMsg{progressCh: progressCh, errCh: errCh, totalBytes: info.Size()}
	}
}

func pollUpload(progressCh <-chan int64, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case b, ok := <-progressCh:
			if !ok {
				return uploadDoneMsg{}
			}
			return uploadProgressMsg{bytes: b}
		case err := <-errCh:
			return uploadDoneMsg{err: err}
		}
	}
}

func (m Model) updateUploading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case uploadStartedMsg:
		m.uploading.progressCh = msg.progressCh
		m.uploading.errCh = msg.errCh
		m.uploading.totalBytes = msg.totalBytes
		return m, pollUpload(msg.progressCh, msg.errCh)

	case uploadProgressMsg:
		m.uploading.bytes = msg.bytes
		return m, pollUpload(m.uploading.progressCh, m.uploading.errCh)

	case uploadDoneMsg:
		dm, cmd := newDoneModel(m.ripping.tempDir, m.movieName, m.mainMKV, msg.err, m.flow)
		m.done = dm
		m.state = StateDone
		return m, cmd
	}
	return m, nil
}

func (m Model) viewUploading() string {
	header := titleStyle.Render("  Uploading  ")
	var pct float64
	if m.uploading.totalBytes > 0 {
		pct = float64(m.uploading.bytes) / float64(m.uploading.totalBytes)
	}
	bar := m.uploading.bar.ViewAs(pct)
	mb := float64(m.uploading.bytes) / 1024 / 1024
	total := float64(m.uploading.totalBytes) / 1024 / 1024
	content := fmt.Sprintf(
		"  Uploading to:\n  %s\n\n  %s\n  %s\n",
		sublabelStyle.Render(m.uploading.remotePath),
		bar,
		dimStyle.Render(fmt.Sprintf("%.1f / %.1f MB", mb, total)),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(content),
		"",
		helpStyle.Render("  ctrl+c  abort"),
	)
}

// transitionAfterFileSelect decides the next step once mainMKV is set.
func (m Model) transitionAfterFileSelect() (tea.Model, tea.Cmd) {
	if m.flow != Unknown {
		m.uploading = uploadModel{
			bar:        newProgressBar(),
			remotePath: m.outputDir,
		}
		m.state = StateUploading
		return m, startUploadCmd(m.mainMKV, m.outputDir)
	}
	dm, cmd := newDoneModel(m.ripping.tempDir, m.movieName, m.mainMKV, nil, Unknown)
	m.done = dm
	m.state = StateDone
	return m, cmd
}
