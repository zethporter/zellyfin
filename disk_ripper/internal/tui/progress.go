package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"ripper/internal/config"
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
	outputDir  string
}

func newRippingModel(device, outputDir string) (rippingModel, tea.Cmd) {
	rm := rippingModel{
		bar:       newProgressBar(),
		outputDir: outputDir,
	}
	cmd := func() tea.Msg {
		progressCh := make(chan int, 20)
		errCh := make(chan error, 1)
		go func() {
			errCh <- ripper.RipDisc(device, outputDir, progressCh)
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
			dm, cmd := newDoneModel(m.ripping.outputDir, m.movieName, m.mainMKV, msg.err, m.fullPipeline)
			m.done = dm
			return m, cmd
		}
		files, err := ripper.FindMKVFiles(m.ripping.outputDir)
		if err != nil || len(files) == 0 {
			ripErr := fmt.Errorf("no MKV files found in %s", m.ripping.outputDir)
			m.err = ripErr
			dm, cmd := newDoneModel(m.ripping.outputDir, m.movieName, "", ripErr, m.fullPipeline)
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
		sublabelStyle.Render(m.ripping.outputDir),
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

func startUploadCmd(sftp config.SFTPConfig, localFile, remotePath string) tea.Cmd {
	return func() tea.Msg {
		info, err := os.Stat(localFile)
		if err != nil {
			return uploadDoneMsg{err: err}
		}
		progressCh := make(chan int64, 20)
		errCh := make(chan error, 1)
		go func() {
			errCh <- transfer.Upload(sftp, localFile, remotePath, progressCh)
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
		dm, cmd := newDoneModel(m.ripping.outputDir, m.movieName, m.mainMKV, msg.err, m.fullPipeline)
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
	if m.fullPipeline {
		folderName := filepath.Base(m.ripping.outputDir)
		remotePath := filepath.Join(
			m.cfg.SFTP.RemotePath,
			folderName,
			folderName+".mkv",
		)
		m.uploading = uploadModel{
			bar:        newProgressBar(),
			remotePath: remotePath,
		}
		m.state = StateUploading
		return m, startUploadCmd(m.cfg.SFTP, m.mainMKV, remotePath)
	}
	dm, cmd := newDoneModel(m.ripping.outputDir, m.movieName, m.mainMKV, nil, false)
	m.done = dm
	m.state = StateDone
	return m, cmd
}
