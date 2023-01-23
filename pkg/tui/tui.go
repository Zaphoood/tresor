package tui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
	kp "github.com/Zaphoood/tresor/lib/keepass"
)

type Status int
const (
    Init    Status = iota
    Loading Status = iota
    Done    Status = iota
    Failed  Status = iota
)

type model struct {
    status  Status
    content string
    err     error

    focusIndex int
    inputs     []textinput.Model

    winWidth  int
    winHeight int
}

var boxStyle = lipgloss.NewStyle().
    Width(50).
    Padding(1, 2, 1).
    BorderStyle(lipgloss.NormalBorder())

func NewModel() model {
    m := model{
       status: 0,
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

func (m model) Init() tea.Cmd {
    return tea.EnterAltScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.winWidth = msg.Width
        m.winHeight = msg.Height
    case doneMsg:
        m.status = Done
        m.content = msg.content
    case errMsg:
        m.status = Failed
        m.err = msg
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c":
            return m, tea.Quit
        case "tab", "shift+tab", "enter", "up", "down":
            if m.status == Init {
                s := msg.String();

                if s == "enter" && m.focusIndex >= len(m.inputs) - 1 {
                    m.status = Loading
                    return m, openDatabase(m.inputs[0].Value())
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

            return m, nil
        }
    }
    cmd := m.updateInputs(msg)

    return m, cmd
}

func (m model) updateInputs(msg tea.Msg) tea.Cmd {
    cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
    switch m.status {
    case Init:
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
    case Loading:
        return m.centerInWindow(boxStyle.Render("Loading..."))
    case Done:
        return m.centerInWindow(boxStyle.Render(fmt.Sprintf("Done\n%s", m.content)))
    case Failed:
        return m.centerInWindow(boxStyle.Render("Error: " + m.err.Error()))
    default:
        return m.centerInWindow(boxStyle.Render("Invalid status: " + fmt.Sprintf("%d", m.status)))
    }
}

func (m model) centerInWindow(text string) string {
    return lipgloss.Place(m.winWidth, m.winHeight, lipgloss.Center, lipgloss.Center, text)
}

func openDatabase(path string) tea.Cmd {
    return func() tea.Msg {
        content, err := kp.LoadDB(path)
        if err != nil {
            return errMsg{err}
        }
        return doneMsg{content}
    }
}

type doneMsg struct {
    content string
}

type errMsg struct { err error }

func (e errMsg) Error() string { return e.err.Error() }
