package tui

import (
	"log"

	database "github.com/Zaphoood/tresor/src/keepass/database"
	tea "github.com/charmbracelet/bubbletea"
)

type viewState int

const (
	fileSelectorView viewState = iota
	passwordView
	navigateView
)

type MainModel struct {
	// Which sub-model we are currently viewing
	view          viewState
	fileSelector  tea.Model
	passwordInput tea.Model
	navigate      tea.Model
	// Instead of asking the user for input, a database can be passed upon construction
	// This is useful when files are openend via command line arguments
	database *database.Database

	windowWidth  int
	windowHeight int
}

func NewMainModel(d *database.Database) MainModel {
	return MainModel{view: fileSelectorView,
		fileSelector: NewFileSelector(),
		database:     d,
	}
}

func (m MainModel) Init() tea.Cmd {
	if m.database != nil {
		return func() tea.Msg { return loadDoneMsg{m.database} }
	}
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	case loadDoneMsg:
		cmds = append(cmds, m.initPasswordView(msg.database))
	case decryptDoneMsg:
		cmds = append(cmds, m.initNavigateView(msg.database))
	case globalResizeMsg:
		m.windowWidth = msg.width
		m.windowHeight = msg.height
	}

	switch m.view {
	case fileSelectorView:
		m.fileSelector, cmd = m.fileSelector.Update(msg)
	case passwordView:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	case navigateView:
		m.navigate, cmd = m.navigate.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *MainModel) initPasswordView(d *database.Database) tea.Cmd {
	m.view = passwordView
	m.passwordInput = NewPasswordInput(d, m.windowWidth, m.windowHeight)
	return m.passwordInput.Init()
}

func (m *MainModel) initNavigateView(d *database.Database) tea.Cmd {
	m.view = navigateView
	m.navigate = NewNavigate(d, m.windowWidth, m.windowHeight)
	return m.navigate.Init()
}

func (m MainModel) View() string {
	switch m.view {
	case fileSelectorView:
		return m.fileSelector.View()
	case passwordView:
		return m.passwordInput.View()
	case navigateView:
		return m.navigate.View()
	default:
		log.Printf("ERROR: Invalid view: %d", m.view)
		return "Invalid view: %d"
	}
}
