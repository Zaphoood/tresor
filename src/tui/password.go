package tui

import (
	"fmt"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type PasswordInput struct {
	input textinput.Model
	err   error

	windowWidth  int
	windowHeight int

	database *database.Database
}

func NewPasswordInput(database *database.Database, windowWidth, windowHeight int) PasswordInput {
	m := PasswordInput{
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

func (m PasswordInput) Init() tea.Cmd {
	return nil
}

func (m PasswordInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case decryptFailedMsg:
		m.err = msg.err
		m.input.SetValue("")
		return m, nil
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, decryptFileCmd(m.database, m.input.Value())
		}
	}
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m PasswordInput) viewError() string {
	if m.err != nil {
		return fmt.Sprintf("\n%s\n", m.err)
	}
	return ""
}

func (m PasswordInput) View() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Enter password for %s:\n\n", m.database.Path()))
	builder.WriteString(m.input.View())
	builder.WriteRune('\n')
	builder.WriteString(m.viewError())
	builder.WriteString("\n(Press 'Ctrl-c' to quit)")

	return centerInWindow(boxStyle.Render(builder.String()), m.windowWidth, m.windowHeight)
}
