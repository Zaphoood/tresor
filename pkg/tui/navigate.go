package tui

import (
	"fmt"
	"log"

	kp "github.com/Zaphoood/tresor/lib/keepass"
	"github.com/Zaphoood/tresor/lib/keepass/parser"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* Model for navigating the Database in order to view and edit entries */

type Navigate struct {
	parent       table.Model
	selector     table.Model
	preview      table.Model
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

	n.parent = table.New(
		table.WithWidth(int(float64(windowWidth)*0.2)),
		table.WithColumns(columns),
	)
	n.selector = table.New(
		table.WithWidth(int(float64(windowWidth)*0.4)),
		table.WithColumns(columns),
		table.WithFocused(true),
	)
	n.preview = table.New(
		table.WithWidth(windowWidth-n.parent.Width()-n.selector.Width()),
		table.WithColumns(columnsWide),
	)
	n.updateAllTables()

	return n
}

func (m *Navigate) updateAllTables() {
	if len(m.path) == 0 {
		m.parent.SetRows([]table.Row{})
	} else {
		_, _, err := m.updateTable(&m.parent, m.path[:len(m.path)-1])
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
		m.parent.SetCursor(m.path[len(m.path)-1])
	}
	var err error
	m.groupsCenter, _, err = m.updateTable(&m.selector, m.path)
	if err != nil {
		log.Printf("ERROR: %s", err)
	}
	m.selector.SetCursor(0)

	m.updateRightTable()
}

func (m *Navigate) updateRightTable() {
	cursor := m.selector.Cursor()
	// If a table is empty and the 'down' or 'up' key is pressed, the cursor becomes -1
	// This may be a bug in Bubbles? Might also be intended
	if cursor < 0 {
		m.preview.SetRows([]table.Row{})
		return
	}
	item, err := m.database.Parsed().GetItem(append(m.path, cursor))
	if err != nil {
		log.Printf("ERROR: %s", err)
		return
	}
	switch item := item.(type) {
	case parser.Group:
		// A group is focused
		m.setItems(&m.preview, item.Groups, item.Entries)
	case parser.Entry:
		// An entry is focused
		m.loadEntry(&m.preview, item)
	default:
		log.Printf("ERROR: Expected Group or Entry")
	}
}

func (m *Navigate) updateTable(t *table.Model, path []int) (int, int, error) {
	groups, entries, err := m.database.Parsed().ListPath(path)
	if err != nil {
		return 0, 0, err
	}
	if len(groups)+len(entries) > 0 {
		m.setItems(t, groups, entries)
	} else {
		t.SetRows([]table.Row{
			{"(No entries)", ""},
		})
	}
	return len(groups), len(entries), nil
}

func (m *Navigate) loadEntry(t *table.Model, entry parser.Entry) {
	title := entry.TryGet("Title", "(No title)")
	rows := []table.Row{
		{fmt.Sprintf("Title: %s", title), ""},
	}
	m.preview.SetRows(rows)
}

func (m *Navigate) setItems(t *table.Model, groups []parser.Group, entries []parser.Entry) {
	rows := make([]table.Row, len(groups)+len(entries))
	for i, group := range groups {
		rows[i] = table.Row{group.Name, fmt.Sprint(len(group.Groups) + len(group.Entries))}
	}
	for i, entry := range entries {
		title := entry.TryGet("Title", "(No title)")
		rows[len(groups)+i] = table.Row{title, ""}
	}
	t.SetRows(rows)
}

func (m *Navigate) moveLeft() {
	if len(m.path) == 0 {
		return
	}
	m.path = m.path[:len(m.path)-1]
	m.updateAllTables()
}

func (m *Navigate) moveRight() {
	cursor := m.selector.Cursor()
	if cursor < 0 || cursor >= m.groupsCenter {
		return
	}
	m.path = append(m.path, m.selector.Cursor())
	m.updateAllTables()
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
	cursor := m.selector.Cursor()
	var cmd tea.Cmd
	m.selector, cmd = m.selector.Update(msg)
	if cursor != m.selector.Cursor() {
		m.updateRightTable()
	}

	return m, cmd
}

func (m Navigate) View() string {
	return lipgloss.JoinHorizontal(0, m.parent.View(), m.selector.View(), m.preview.View())
}
