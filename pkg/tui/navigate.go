package tui

import (
	"fmt"

	kp "github.com/Zaphoood/tresor/lib/keepass"
	tea "github.com/charmbracelet/bubbletea"
)

/* Model for navigating the Database in order to view and edit entries */

type Navigate struct {
	database *kp.Database
	err      error

	windowWidth  int
	windowHeight int
}

func NewNavigate(database *kp.Database, windowWidth, windowHeight int) Navigate {
	return Navigate{
		database:     database,
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
	}
}

func (m Navigate) Init() tea.Cmd {
	return nil
}

func (m Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Navigate) View() string {
	return fmt.Sprintf("Navigate %s\n\n%s", m.database.Path(), m.database.Content())
}
