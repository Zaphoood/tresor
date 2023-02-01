package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

/* Inital model where you enter the path to the database */

type SelectFile struct {
	focusIndex int
	input      textinput.Model
	err        error

	windowWidth  int
	windowHeight int
}

func NewSelectFile() SelectFile {
	m := SelectFile{input: textinput.New()}

	m.input.Width = 32
	m.input.Placeholder = "File"
	m.input.Focus()

	return m
}

func (m SelectFile) Init() tea.Cmd {
	return nil
}

func (m SelectFile) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case loadFailedMsg:
		m.err = msg.err
		m.focusIndex = 0
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
				return m, fileSelectedCmd(m.input.Value())
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

func (m SelectFile) View() string {
	var builder strings.Builder
	builder.WriteString("Select file:\n\n")

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
