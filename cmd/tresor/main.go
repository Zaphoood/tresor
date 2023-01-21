package main

import (
	"fmt"
	"os"

	kp "github.com/Zaphoood/tresor/lib/keepass"
	tea "github.com/charmbracelet/bubbletea"
)


type Status int
const (
    Init    Status = iota
    Loading Status = iota
    Done    Status = iota
    Failed  Status = iota
)

type model struct {
    status Status
    content string
    err    error
}

func initalModel() model {
    return model{status: 0}
}

func (m model) Init() tea.Cmd {
    return tea.EnterAltScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case doneMsg:
        m.status = Done
        m.content = msg.content
    case errMsg:
        m.status = Failed
        m.err = msg
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "enter":
            if m.status == Init {
                m.status = Loading
                return m, openDatabase("/home/foo/secrets/bar.kdbx")
            }
            return m, nil
        }
    }
    return m, nil
}

func (m model) View() string {
    switch m.status {
    case Init:
        return "Press enter to open database"
    case Loading:
        return "Loading..."
    case Done:
        return fmt.Sprintf("Done\n%s", m.content)
    case Failed:
        return "Error: " + m.err.Error()
    default:
        return "Invalid status: " + fmt.Sprintf("%d", m.status)
    }
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


func main() {
    p := tea.NewProgram(initalModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %s", err)
        os.Exit(1)
    }
}
