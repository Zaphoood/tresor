package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

/* Inital model where you enter the path to the database */

type FileSelector struct {
	input textinput.Model
	err   error

	completionBase     string
	completions        []string
	completionIndex    int
	cyclingCompletions bool

	windowWidth  int
	windowHeight int
}

func NewFileSelector() FileSelector {
	m := FileSelector{input: textinput.New()}

	m.input.Width = 32
	m.input.Placeholder = "File"
	m.input.Focus()

	return m
}

func (m FileSelector) Init() tea.Cmd {
	return nil
}

func (m FileSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case loadFailedMsg:
		m.err = msg.err
		return m, nil
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.cyclingCompletions {
				m.cycleCompletion(1)
			} else {
				m.cyclingCompletions = true
				err := m.loadCompletions()
				if err != nil {
					m.err = err
				}
				m.cycleCompletion(0)
			}
		case "shift+tab":
			if m.cyclingCompletions && len(m.completions) > 0 {
				m.cycleCompletion(-1)
			}
		case "enter":
			return m, fileSelectedCmd(m.input.Value())
		}
	}
	oldValue := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != oldValue {
		m.cyclingCompletions = false
	}

	return m, cmd
}

func (m *FileSelector) loadCompletions() error {
	input := m.input.Value()
	var err error
	m.completions, err = completePath(input)
	if err != nil {
		return err
	}
	if input == "~" {
		input = "~/"
	}
	m.completionBase = filepath.Dir(input)
	return nil
}

func (m *FileSelector) cycleCompletion(n int) {
	if !(n == 1 || n == 0 || n == -1) {
		panic(fmt.Sprintf("Cannot cycle completions by %d steps", n))
	}
	switch len(m.completions) {
	case 0:
		return
	case 1:
		m.cyclingCompletions = false
		fallthrough
	default:
		m.completionIndex = mod(m.completionIndex+n, len(m.completions))
		m.input.SetValue(joinRetainTrailingSep(m.completionBase, m.completions[m.completionIndex]))
		m.input.SetCursor(len(m.input.Value()))
	}
}

func (m FileSelector) viewError() string {
	if m.err != nil {
		return fmt.Sprintf("\n%s\n", m.err)
	}
	return ""
}

func (m FileSelector) View() string {
	var builder strings.Builder
	builder.WriteString("Select file:\n\n")
	builder.WriteString(m.input.View())
	builder.WriteRune('\n')
	builder.WriteString(m.viewError())
	builder.WriteString("\n(Press 'Ctrl-c' to quit)")

	return centerInWindow(boxStyle.Render(builder.String()), m.windowWidth, m.windowHeight)
}
