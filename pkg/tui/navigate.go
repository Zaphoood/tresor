package tui

import (
	"fmt"

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
		table.WithColumns(columns),
	)
	n.populateAll()

	return n
}

func (m *Navigate) populateAll() {
	root := &m.database.Parsed().Root

	rows := make([]table.Row, len(root.Groups))
	for i, group := range root.Groups {
		rows[i] = table.Row{group.Name, fmt.Sprint(len(group.Groups) + len(group.Entries))}
	}
	m.left.SetRows(rows)

	second := root.Groups[0]
	rows = make([]table.Row, len(second.Groups))
	for i, group := range second.Groups {
		rows[i] = table.Row{group.Name, fmt.Sprint(len(group.Groups) + len(group.Entries))}
	}
	m.center.SetRows(rows)

	m.populateRight()
}

func (m *Navigate) populateRight() {
	group_right := &m.database.Parsed().Root.Groups[0].Groups[m.center.Cursor()]
	rows := make([]table.Row, len(group_right.Groups))
	for i, group := range group_right.Groups {
		rows[i] = table.Row{group.Name, fmt.Sprint(len(group.Groups) + len(group.Entries))}
	}
	m.right.SetRows(rows)
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
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
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
