package tui

import (
	"fmt"
	"log"

	kp "github.com/Zaphoood/tresor/lib/keepass"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* Model for navigating the Database in order to view and edit entries */

type Navigate struct {
	left         table.Model
	center       table.Model
	right        table.Model
	groupsCenter int
	path         []int
	err          error

	windowWidth  int
	windowHeight int

	database *kp.Database
}

var columns []table.Column = []table.Column{
	{Title: "Name", Width: 25},
	{Title: "Entries", Width: 10},
}

var columnsWide []table.Column = []table.Column{
	{Title: "Name", Width: 50},
	{Title: "Entries", Width: 10},
}

func NewNavigate(database *kp.Database, windowWidth, windowHeight int) Navigate {
	n := Navigate{
		path:         []int{0},
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		database:     database,
	}

	n.left = table.New(
		table.WithWidth(int(float64(windowWidth)*0.2)),
		table.WithColumns(columns),
	)
	n.center = table.New(
		table.WithWidth(int(float64(windowWidth)*0.4)),
		table.WithColumns(columns),
		table.WithFocused(true),
	)
	n.right = table.New(
		table.WithWidth(windowWidth-n.left.Width()-n.center.Width()),
		table.WithColumns(columnsWide),
	)
	n.populateAllTables()

	return n
}

func (m *Navigate) populateAllTables() {
	if len(m.path) == 0 {
		m.left.SetRows([]table.Row{})
	} else {
		_, _, err := m.populateTable(&m.left, m.path[:len(m.path)-1])
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
		m.left.SetCursor(m.path[len(m.path)-1])
	}
	var err error
	m.groupsCenter, _, err = m.populateTable(&m.center, m.path)
	if err != nil {
		log.Printf("ERROR: %s", err)
	}
	m.center.SetCursor(0)

	m.populateRight()
}

func (m *Navigate) populateRight() {
	cursor := m.center.Cursor()
	// If a table is empty and the 'down' or 'up' key is pressed, the cursor becomes -1
	// This may be a bug in Bubbles? Might also be intended
	if cursor < 0 {
		m.right.SetRows([]table.Row{})
		return
	}
	if cursor < m.groupsCenter {
		// A group is focused
		_, _, err := m.populateTable(&m.right, append(m.path, cursor))
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
	} else {
		// An entry is focused
		m.populateEntry(&m.right, append(m.path, cursor))
	}
}

func (m *Navigate) populateEntry(t *table.Model, path []int) {
	_, entries, err := m.database.Parsed().GetPath(path[:len(path)-1])
	entry := entries[path[len(path)-1]-m.groupsCenter]
	title, err := entry.Get("Title")
	if err != nil {
		title = "(No title)"
	}
	rows := []table.Row{
		{fmt.Sprintf("Title: %s", title), ""},
	}
	m.right.SetRows(rows)
}

func (m *Navigate) populateTable(t *table.Model, path []int) (int, int, error) {
	groups, entries, err := m.database.Parsed().GetPath(path)
	if err != nil {
		return 0, 0, err
	}
	rows := make([]table.Row, len(groups)+len(entries))
	for i, group := range groups {
		rows[i] = table.Row{group.Name, fmt.Sprint(len(group.Groups) + len(group.Entries))}
	}
	for i, entry := range entries {
		title, err := entry.Get("Title")
		if err != nil {
			title = "(No title)"
		}

		rows[len(groups)+i] = table.Row{title, ""}
	}
	t.SetRows(rows)

	return len(groups), len(entries), nil
}

func (m *Navigate) moveLeft() {
	if len(m.path) == 0 {
		return
	}
	m.path = m.path[:len(m.path)-1]
	m.populateAllTables()
}

func (m *Navigate) moveRight() {
	cursor := m.center.Cursor()
	if cursor >= m.groupsCenter {
		return
	}
	m.path = append(m.path, m.center.Cursor())
	m.populateAllTables()
}

func (m Navigate) Init() tea.Cmd {
	return nil
}

func (m Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "l":
			m.moveRight()
		case "h":
			m.moveLeft()
		}
	}
	cursor := m.center.Cursor()
	var cmd tea.Cmd
	m.center, cmd = m.center.Update(msg)
	if cursor != m.center.Cursor() {
		m.populateRight()
	}

	return m, cmd
}

func (m Navigate) View() string {
	return lipgloss.JoinHorizontal(0, m.left.View(), m.center.View(), m.right.View())
}
