package tui

import (
	"log"

	"github.com/Zaphoood/tresor/lib/keepass/database"
	"github.com/Zaphoood/tresor/lib/keepass/parser"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* Model for navigating the Database in order to view and edit entries */

var itemViewColumns []table.Column = []table.Column{
	{Title: "Name", Width: 0},
	{Title: "Entries", Width: 7},
}

var entryViewColumns []table.Column = []table.Column{
	{Title: "Key", Width: 40},
	{Title: "Value", Width: 0},
}

type Navigate struct {
	parent       itemTable
	selector     itemTable
	groupPreview itemTable
	entryPreview itemTable
	// previewEntry is true when an Entry is focused, false when a Group is focused
	previewEntry bool
	lastSelected map[string]int
	styles       table.Styles
	path         []int
	err          error

	windowWidth  int
	windowHeight int

	database *database.Database
}

func NewNavigate(database *database.Database, windowWidth, windowHeight int) Navigate {
	n := Navigate{
		styles:       table.DefaultStyles(),
		path:         []int{0},
		lastSelected: make(map[string]int),
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		database:     database,
	}
	n.parent = newItemTable(n.styles, itemViewColumns)
	n.selector = newItemTable(n.styles, itemViewColumns, table.WithFocused(true))
	n.groupPreview = newItemTable(n.styles, itemViewColumns)
	n.entryPreview = newItemTable(table.Styles{
		Header:   n.styles.Header.Copy(),
		Cell:     n.styles.Cell.Copy(),
		Selected: lipgloss.NewStyle(),
	}, entryViewColumns)
	n.previewEntry = false

	n.resizeAll()
	n.updateAll()

	return n
}

func (n *Navigate) resizeAll() {
	selectorWidth := int(float64(n.windowWidth) * 0.3)
	previewWidth := int(float64(n.windowWidth) * 0.5)
	parentWidth := n.windowWidth - selectorWidth - previewWidth
	height := n.windowHeight - 1

	n.parent.SetSize(parentWidth, height)
	n.selector.SetSize(selectorWidth, height)
	n.groupPreview.SetSize(previewWidth, height)
	n.entryPreview.SetSize(previewWidth, height)
}

func (n *Navigate) updateAll() {
	if len(n.path) == 0 {
		n.parent.Clear()
	} else {
		n.parent.Load(n.database.Parsed(), n.path[:len(n.path)-1], &n.lastSelected)
	}

	n.selector.Load(n.database.Parsed(), n.path, &n.lastSelected)

	n.updatePreview()
}

func (n *Navigate) updatePreview() {
	cursor := n.selector.Cursor()
	// If a table is empty and the 'down' or 'up' key is pressed, the cursor becomes -1
	// This may be a bug in Bubbles? Might also be intended
	if cursor < 0 {
		n.groupPreview.Clear()
		return
	}
	item, err := n.database.Parsed().GetItem(append(n.path, cursor))
	if err != nil {
		switch err := err.(type) {
		case parser.PathOutOfRange:
			n.groupPreview.Clear()
		default:
			log.Printf("ERROR: %s\n", err)
		}
		return
	}
	switch item := item.(type) {
	case parser.Group:
		// A group is focused
		n.groupPreview.LoadGroup(item, &n.lastSelected)
		n.previewEntry = false
	case parser.Entry:
		// An entry is focused
		n.entryPreview.LoadEntry(item)
		n.previewEntry = true
	default:
		log.Printf("ERROR: Expected Group or Entry in updatePreview")
	}
}

func (n *Navigate) moveLeft() {
	if len(n.path) == 0 {
		return
	}
	n.rememberSelected()
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
	n.rememberSelected()
	n.path = append(n.path, n.selector.Cursor())
	n.updateAll()
}

func (n *Navigate) rememberSelected() {
	currentGroup, err := n.database.Parsed().GetItem(n.path)
	if currentGroup, ok := currentGroup.(parser.Group); err == nil && ok {
		n.lastSelected[currentGroup.UUID] = n.selector.Cursor()
	}
}

func (n Navigate) Init() tea.Cmd {
	return nil
}

func (n Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		n.windowWidth = msg.Width
		n.windowHeight = msg.Height
		n.resizeAll()
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

	t := table.New()
	t, cmd = t.Update(msg)

	return n, cmd
}

func (n Navigate) View() string {
	var preview string
	if n.previewEntry {
		preview = n.entryPreview.View()
	} else {
		preview = n.groupPreview.View()
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, n.parent.View(), n.selector.View(), preview)
}
