package tui

import (
	"log"

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

	windowWidth  int
	windowHeight int
}

func NewMainModel() MainModel {
	return MainModel{view: selectFileView, selectFile: NewSelectFile()}
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case loadDoneMsg:
		m.view = decryptView
		m.decrypt = NewDecrypt(msg.database, m.windowWidth, m.windowHeight)
		cmd = m.decrypt.Init()
		cmds = append(cmds, cmd)
	case decryptDoneMsg:
		m.view = navigateView
		m.navigate = NewNavigate(msg.database, m.windowWidth, m.windowHeight)
		cmd = m.navigate.Init()
		cmds = append(cmds, cmd)
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

func (m MainModel) View() string {
	switch m.view {
	case selectFileView:
		return m.selectFile.View()
	case decryptView:
		return m.decrypt.View()
	case navigateView:
		return m.navigate.View()
	default:
		log.Fatalf("Invalid view: %d", m.view)
		return "Invalid view: %d"
	}
}
