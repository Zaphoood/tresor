package tui

import (
	"fmt"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Decrypt struct {
	focusIndex int
	input      textinput.Model
	err        error

	windowWidth  int
	windowHeight int

	database *database.Database
}

func NewDecrypt(database *database.Database, windowWidth, windowHeight int) Decrypt {
	m := Decrypt{
		input:        textinput.New(),
		database:     database,
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
	}

	m.input.Width = 32
	m.input.Placeholder = "Password"
	m.input.EchoMode = textinput.EchoPassword
	m.input.EchoCharacter = '•'
	m.input.Focus()

	return m
}

func (m Decrypt) Init() tea.Cmd {
	return nil
}

func (m Decrypt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case decryptFailedMsg:
		m.err = msg.err
		m.focusIndex = 0
		m.input.SetValue("")
		cmd = m.input.Focus()
		return m, cmd
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" {
				return m, decryptFileCmd(m.database, m.input.Value())
			}

			if s == "down" || s == "tab" {
				m.focusIndex++
			} else if s == "up" || s == "shift+tab" {
				m.focusIndex--
			}

			if m.focusIndex > 1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 1
			}

			if m.focusIndex == 0 {
				cmd = m.input.Focus()
				return m, cmd
			} else {
				m.input.Blur()
			}
			return m, nil
		}
	}
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m Decrypt) View() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Enter password for %s:\n\n", m.database.Path()))

	builder.WriteString(m.input.View())
	builder.WriteRune('\n')

	if m.err != nil {
		builder.WriteRune('\n')
		builder.WriteString(m.err.Error())
		builder.WriteRune('\n')
	}

	if m.focusIndex == 1 {
		builder.WriteString("\n[ OK ]\n")
	} else {
		builder.WriteString("\n  OK  \n")
	}
	builder.WriteString("\n(Press 'Ctrl-c' to quit)")

	return centerInWindow(boxStyle.Render(builder.String()), m.windowWidth, m.windowHeight)
}
