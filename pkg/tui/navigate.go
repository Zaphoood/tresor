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
	left   table.Model
	center table.Model
	right  table.Model
	path   []int
	err    error

	windowWidth  int
	windowHeight int

	database *kp.Database
}

var columns []table.Column = []table.Column{
	{Title: "Name", Width: 25},
	{Title: "Entries", Width: 10},
}

var columnsWide []table.Column = []table.Column{
	{Title: "Name", Width: 60},
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
	n.populateAll()

	return n
}

func (m *Navigate) populateAll() {
	if len(m.path) == 0 {
		m.left.SetRows([]table.Row{})
	} else {
		err := m.populate(&m.left, m.path[:len(m.path)-1])
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
		m.left.SetCursor(m.path[len(m.path)-1])
	}
	err := m.populate(&m.center, m.path)
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
	err := m.populate(&m.right, append(m.path, cursor))
	if err != nil {
		log.Printf("ERROR: %s", err)
	}
}

func (m *Navigate) populate(t *table.Model, path []int) error {
	groups, entries, err := m.database.Parsed().GetPath(path)
	if err != nil {
		return err
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

	return nil
}

func (m *Navigate) moveLeft() {
	if len(m.path) == 0 {
		return
	}
	m.path = m.path[:len(m.path)-1]
	m.populateAll()
}

func (m *Navigate) moveRight() {
	if len(m.center.Rows()) == 0 {
		return
	}
	m.path = append(m.path, m.center.Cursor())
	m.populateAll()
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
