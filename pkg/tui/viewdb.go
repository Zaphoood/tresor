package tui

import (
    "fmt"
    "strings"
    tea "github.com/charmbracelet/bubbletea"
	kp "github.com/Zaphoood/tresor/lib/keepass"
)

/* Model for navigating the Database in order to view and edit entries */

type Navigate struct {
    status loadingStatus
    path     string
    password string
    err      error
}

type loadingStatus int

const (
    Loading loadingStatus = iota
    Finished
    Failed
)

func NewNavigate(path, password string) Navigate {
    return Navigate{Loading, path, password, nil}
}

func (m Navigate) Init() tea.Cmd {
    return m.loadDatabase()
}

func (m Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case doneMsg:
        m.status = Finished
    case errMsg:
        m.status = Failed
        m.err = msg.err
    case tea.KeyMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m Navigate) View() string {
    var b strings.Builder
    switch m.status {
    case Loading:
        b.WriteString(fmt.Sprintf("Loading %s...", m.path))
    case Finished:
        b.WriteString("View database:\n\n")
        b.WriteString(fmt.Sprintf("path=%s\n", m.path))
        b.WriteString(fmt.Sprintf("password=%s\n", m.password))
    case Failed:
        b.WriteString(fmt.Sprintf("Error: %s", m.err.Error()))
    default:
        b.WriteString("Oops")
    }
    return b.String()
}

func (m *Navigate) loadDatabase() tea.Cmd {
    return func() tea.Msg {
        db := kp.NewDatabase(m.path)
        err := db.Load(m.password)
        if err != nil {
            return errMsg{err}
        }
        return doneMsg{db.Content()}
    }
}

type doneMsg struct {
    content string
}

type errMsg struct { err error }

func (e errMsg) Error() string { return e.err.Error() }
