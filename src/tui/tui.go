package tui

import (
	"log"

	database "github.com/Zaphoood/tresor/src/keepass/database"
	tea "github.com/charmbracelet/bubbletea"
)

type viewState int

const (
	selectFileView viewState = iota
	decryptView
	navigateView
)

type MainModel struct {
	// Which sub-model we are currently viewing
	view       viewState
	selectFile tea.Model
	decrypt    tea.Model
	navigate   tea.Model
	// Instead of asking the user for input, a database can be passed upon construction
	// This is useful when files are openend via command line arguments
	database *database.Database

	windowWidth  int
	windowHeight int
}

func NewMainModel(d *database.Database) MainModel {
	return MainModel{view: selectFileView, selectFile: NewSelectFile(), database: d}
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
		cmds = append(cmds, m.initDecryptView(msg.database))
	case decryptDoneMsg:
		cmds = append(cmds, m.initNavigateView(msg.database))
	case globalResizeMsg:
		m.windowWidth = msg.width
		m.windowHeight = msg.height
	}

	switch m.view {
	case selectFileView:
		newSelectFile, newCmd := m.selectFile.Update(msg)
		newSelectFile, ok := newSelectFile.(SelectFile)
		if !ok {
			panic("Could not assert that newSelectFile is of type SelectFile after Update()")
		}
		m.selectFile = newSelectFile
		cmd = newCmd
	case decryptView:
		newDecrypt, newCmd := m.decrypt.Update(msg)
		newDecrypt, ok := newDecrypt.(Decrypt)
		if !ok {
			panic("Could not assert that newDecrypt is of type Decrypt after Update()")
		}
		m.decrypt = newDecrypt
		cmd = newCmd
	case navigateView:
		newNavigate, newCmd := m.navigate.Update(msg)
		newNavigate, ok := newNavigate.(Navigate)
		if !ok {
			panic("Could not assert that newNavigate is of type Navigate after Update()")
		}
		m.navigate = newNavigate
		cmd = newCmd
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *MainModel) initDecryptView(d *database.Database) tea.Cmd {
	m.view = decryptView
	m.decrypt = NewDecrypt(d, m.windowWidth, m.windowHeight)
	return m.decrypt.Init()
}

func (m *MainModel) initNavigateView(d *database.Database) tea.Cmd {
	m.view = navigateView
	m.navigate = NewNavigate(d, m.windowWidth, m.windowHeight)
	return m.navigate.Init()
}

func (m MainModel) View() string {
	switch m.view {
	case selectFileView:
		return m.selectFile.View()
	case decryptView:
		return m.decrypt.View()
	case navigateView:
		return m.navigate.View()
	default:
		log.Printf("ERROR: Invalid view: %d", m.view)
		return "Invalid view: %d"
	}
}
