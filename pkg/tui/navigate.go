package tui

import (
	"fmt"
	"log"

	"github.com/Zaphoood/tresor/lib/keepass"
	"github.com/Zaphoood/tresor/lib/keepass/parser"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* Model for navigating the Database in order to view and edit entries */

type Navigate struct {
	parent   table.Model
	selector table.Model
	preview  table.Model
	path     []int
	err      error

	windowWidth  int
	windowHeight int

	database *keepass.Database
}

var columns []table.Column = []table.Column{
	{Title: "Name", Width: 25},
	{Title: "Entries", Width: 10},
}

var columnsWide []table.Column = []table.Column{
	{Title: "Name", Width: 50},
	{Title: "Entries", Width: 10},
}

func NewNavigate(database *keepass.Database, windowWidth, windowHeight int) Navigate {
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
	n.updateAll()

	return n
}

func (n *Navigate) updateAll() {
	if len(n.path) == 0 {
		n.parent.SetRows([]table.Row{})
	} else {
		updateTable(&n.parent, n.database.Parsed(), n.path[:len(n.path)-1])
		n.parent.SetCursor(n.path[len(n.path)-1])
	}
	updateTable(&n.selector, n.database.Parsed(), n.path)
	n.selector.SetCursor(0)

	n.updatePreview()
}

func (n *Navigate) updatePreview() {
	cursor := n.selector.Cursor()
	// If a table is empty and the 'down' or 'up' key is pressed, the cursor becomes -1
	// This may be a bug in Bubbles? Might also be intended
	if cursor < 0 {
		n.preview.SetRows([]table.Row{})
		return
	}
	item, err := n.database.Parsed().GetItem(append(n.path, cursor))
	if err != nil {
		switch err := err.(type) {
		case parser.PathOutOfRange:
			n.preview.SetRows([]table.Row{
				{"", ""},
			})
		default:
			n.preview.SetRows([]table.Row{
				{err.Error(), ""},
			})
		}
		return
	}
	switch item := item.(type) {
	case parser.Group:
		// A group is focused
		setItems(&n.preview, item.Groups, item.Entries)
	case parser.Entry:
		// An entry is focused
		n.loadEntry(&n.preview, item)
	default:
		log.Printf("ERROR: Expected Group or Entry in updatePreview")
	}
}

func updateTable(t *table.Model, d *parser.Document, path []int) {
	groups, entries, err := d.ListPath(path)
	if err != nil {
		t.SetRows([]table.Row{
			{err.Error(), ""},
		})
	}
	if len(groups)+len(entries) > 0 {
		setItems(t, groups, entries)
	} else {
		t.SetRows([]table.Row{
			{"(No entries)", ""},
		})
	}
}

func (n *Navigate) loadEntry(t *table.Model, entry parser.Entry) {
	title := entry.TryGet("Title", "(No title)")
	rows := []table.Row{
		{fmt.Sprintf("Title: %s", title), ""},
	}
	n.preview.SetRows(rows)
}

func setItems(t *table.Model, groups []parser.Group, entries []parser.Entry) {
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

func (n *Navigate) moveLeft() {
	if len(n.path) == 0 {
		return
	}
	n.path = n.path[:len(n.path)-1]
	n.updateAll()
}

func (n *Navigate) moveRight() {
	cursor := n.selector.Cursor()
	selected, err := n.database.Parsed().GetItem(append(n.path, cursor))
	if err != nil {
		return
	}
	if _, ok := selected.(parser.Group); !ok {
		return
	}
	n.path = append(n.path, n.selector.Cursor())
	n.updateAll()
}

func (n Navigate) Init() tea.Cmd {
	return nil
}

func (n Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		n.windowWidth = msg.Width
		n.windowHeight = msg.Height
		return n, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return n, tea.Quit
		case "l":
			n.moveRight()
		case "h":
			n.moveLeft()
		}
	}
	var cmd tea.Cmd
	cursor := n.selector.Cursor()
	n.selector, cmd = n.selector.Update(msg)
	if cursor != n.selector.Cursor() {
		n.updatePreview()
	}

	return n, cmd
}

func (n Navigate) View() string {
	return lipgloss.JoinHorizontal(0, n.parent.View(), n.selector.View(), n.preview.View())
}
