package tui

import (
    "strings"
	tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/lipgloss"
)

/* Inital model where you enter the path to the database and the password */

type Input struct {
    focusIndex int
    inputs     []textinput.Model

    winWidth  int
    winHeight int
}

var boxStyle = lipgloss.NewStyle().
    Width(50).
    Padding(1, 2, 1).
    BorderStyle(lipgloss.NormalBorder())

func NewInput() Input {
    m := Input{
       inputs: make([]textinput.Model, 2),
   }

   var input textinput.Model
   for i := range m.inputs {
       input = textinput.New()
       input.CharLimit = 32
       switch i {
       case 0:
           input.Placeholder = "File"
           input.Focus()
       case 1:
           input.Placeholder = "Password"
           input.EchoMode = textinput.EchoPassword
           input.EchoCharacter = '•'
       }
       m.inputs[i] = input
   }
   return m
}

func (m Input) Init() tea.Cmd {
    return nil
}

func (m Input) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.winWidth = msg.Width
        m.winHeight = msg.Height
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c":
            return m, tea.Quit
        case "tab", "shift+tab", "enter", "up", "down":
            s := msg.String();

            if s == "enter" && m.focusIndex >= len(m.inputs) - 1 {
                return m, openDatabaseCmd(m.inputs[0].Value(), m.inputs[1].Value())
            }

            // Cycle indices
            if s == "down" || s == "tab" {
                m.focusIndex++
            } else if s == "up" || s == "shift+tab" {
                m.focusIndex--
            }

            if m.focusIndex > len(m.inputs) {
                m.focusIndex = 0
            } else if m.focusIndex < 0 {
                m.focusIndex = len(m.inputs)
            }

            cmds := make([]tea.Cmd, len(m.inputs))
            for i := 0; i <= len(m.inputs) - 1; i++ {
                if i == m.focusIndex {
                    // Set focused state
                    cmds[i] = m.inputs[i].Focus()
                    continue
                }
                // Remove focused state
                m.inputs[i].Blur()
            }
            return m, tea.Batch(cmds...)
        }
    }
    cmd := m.updateInputs(msg)

    return m, cmd
}

func (m Input) updateInputs(msg tea.Msg) tea.Cmd {
    cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m Input) View() string {
    var builder strings.Builder
    builder.WriteString("Open database:\n\n")

    for i := range m.inputs {
        builder.WriteString(m.inputs[i].View())
        builder.WriteRune('\n')
    }

    if m.focusIndex == len(m.inputs) {
        builder.WriteString("\n[ OK ]\n")
    } else {
        builder.WriteString("\n  OK  \n")
    }
    builder.WriteString("\n(Press 'Ctrl-c' to quit)")

    return m.centerInWindow(boxStyle.Render(builder.String()))
}

func (m Input) centerInWindow(text string) string {
    return lipgloss.Place(m.winWidth, m.winHeight, lipgloss.Center, lipgloss.Center, text)
}
