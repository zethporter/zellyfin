package tui

import (
	"ripper/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

// State represents which view is currently active.
type State int

const (
	StateMainMenu State = iota
	StateConfigEditor
	StateTMDBSearch
	StateTMDBConfirm
	StateLocalOnlyConfirm
	StateRipping
	StateFileSelect
	StateUploading
	StateDone
)

// Model is the root Bubbletea model. It owns all sub-models and shared
// workflow state, delegating Update/View to the active sub-model.
type Model struct {
	state   State
	cfg     config.Config
	cfgPath string
	width   int
	height  int

	// workflow choice set at main menu
	fullPipeline bool

	// workflow data accumulated across steps
	movieName  string
	year       string
	folderName string
	outputDir  string
	mainMKV    string

	// sub-models (initialised lazily per-phase)
	menu         menuModel
	configEditor configEditorModel
	search       searchModel
	localConfirm localConfirmModel
	ripping      rippingModel
	fileSelect   fileSelectModel
	uploading    uploadModel
	done         doneModel

	err error
}

func New(cfg config.Config, cfgPath string) Model {
	return Model{
		state:   StateMainMenu,
		cfg:     cfg,
		cfgPath: cfgPath,
		menu:    newMenuModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global handlers
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// fall through so active sub-model also gets the resize
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.state {
	case StateMainMenu:
		return m.updateMenu(msg)
	case StateConfigEditor:
		return m.updateConfigEditor(msg)
	case StateTMDBSearch:
		return m.updateSearch(msg)
	case StateLocalOnlyConfirm:
		return m.updateLocalConfirm(msg)
	case StateRipping:
		return m.updateRipping(msg)
	case StateFileSelect:
		return m.updateFileSelect(msg)
	case StateUploading:
		return m.updateUploading(msg)
	case StateDone:
		return m.updateDone(msg)
	default:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "q" {
			m.state = StateMainMenu
		}
		return m, nil
	}
}

func (m Model) View() string {
	switch m.state {
	case StateMainMenu:
		return m.viewMenu()
	case StateConfigEditor:
		return m.viewConfigEditor()
	case StateTMDBSearch:
		return m.viewSearch()
	case StateLocalOnlyConfirm:
		return m.viewLocalConfirm()
	case StateRipping:
		return m.viewRipping()
	case StateFileSelect:
		return m.viewFileSelect()
	case StateUploading:
		return m.viewUploading()
	case StateDone:
		return m.viewDone()
	default:
		return dimStyle.Render("\n  not yet implemented — press q to return to menu\n")
	}
}
