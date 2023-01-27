package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"log"
)

type viewState int

const (
	inputView viewState = iota
	navigateView
)

type MainModel struct {
	// Which sub-model we are currently viewing
	view     viewState
	input    tea.Model
	navigate tea.Model
}

func NewMainModel() MainModel {
	return MainModel{view: inputView, input: NewInput()}
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case openDatabaseMsg:
		m.view = navigateView
		m.navigate = NewNavigate(msg.path, msg.password)
		cmd = m.navigate.Init()
		cmds = append(cmds, cmd)
	}

	switch m.view {
	case inputView:
		newInput, newCmd := m.input.Update(msg)
		newInput, ok := newInput.(Input)
		if !ok {
			panic("Could not assert that newInput is of type Input after Update()")
		}
		m.input = newInput
		cmd = newCmd
	case navigateView:
		newNavigate, newCmd := m.navigate.Update(msg)
		newNavigate, ok := newNavigate.(Navigate)
		if !ok {
			panic("Could not assert that newNavigate is of type Input after Update()")
		}
		m.navigate = newNavigate
		cmd = newCmd
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	switch m.view {
	case inputView:
		return m.input.View()
	case navigateView:
		return m.navigate.View()
	default:
		log.Fatalf("Invalid view: %d", m.view)
		return "Invalid view: %d"
	}
}
