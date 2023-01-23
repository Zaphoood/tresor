package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func openDatabaseCmd(path, password string) tea.Cmd {
    return func() tea.Msg {
        return openDatabaseMsg{path, password}
    }
}

type openDatabaseMsg struct {
    path     string
    password string
}
