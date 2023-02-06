package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

/* Inital model where you enter the path to the database */

type SelectFile struct {
	focusIndex int
	input      textinput.Model
	err        error

	pathWithoutCompletion string
	completions           []string
	completionIndex       int
	cyclingCompletions    bool

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
		case "tab":
			if !m.cyclingCompletions {
				m.cyclingCompletions = true
				err := m.loadCompletions()
				if err != nil {
					m.err = err
				}
				m.cycleCompletion(0)
			} else {
				m.cycleCompletion(1)
			}
		case "shift+tab":
			if m.cyclingCompletions && len(m.completions) > 0 {
				m.cycleCompletion(-1)
			}
		case "enter", "up", "down":
			s := msg.String()

			if s == "enter" {
				return m, fileSelectedCmd(m.input.Value())
			}

			if s == "down" {
				m.focusIndex++
			} else if s == "up" {
				m.focusIndex--
			}
			m.focusIndex = mod(m.focusIndex, 2)
			if m.focusIndex == 0 {
				cmd = m.input.Focus()
				return m, cmd
			} else {
				m.input.Blur()
			}
			return m, nil
		}
	}
	oldValue := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != oldValue {
		m.cyclingCompletions = false
	}

	return m, cmd
}

func (m *SelectFile) loadCompletions() error {
	input := m.input.Value()
	var err error
	m.completions, err = completePath(input)
	if err != nil {
		return err
	}
	if input == "~" {
		input = "~/"
	}
	m.pathWithoutCompletion = filepath.Dir(input)
	return nil
}

func (m *SelectFile) cycleCompletion(n int) {
	if n != 1 && n != 0 && n != -1 {
		return
	}
	switch len(m.completions) {
	case 0:
		return
	case 1:
		m.cyclingCompletions = false
		fallthrough
	default:
		m.completionIndex = mod(m.completionIndex+n, len(m.completions))
		m.input.SetValue(joinRetainTrailingSep(m.pathWithoutCompletion, m.completions[m.completionIndex]))
		m.input.SetCursor(len(m.input.Value()))
	}
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
