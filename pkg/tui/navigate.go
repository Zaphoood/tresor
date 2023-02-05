package tui

import (
	"log"

	"github.com/Zaphoood/tresor/lib/keepass"
	"github.com/Zaphoood/tresor/lib/keepass/parser"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* Model for navigating the Database in order to view and edit entries */

var itemViewColumns []sizedColumn = []sizedColumn{
	{"Name", 0},
	{"Entries", 7},
}

var entryViewColumns []sizedColumn = []sizedColumn{
	{"Key", 40},
	{"Value", 0},
}

type Navigate struct {
	parent       itemTable
	selector     itemTable
	groupPreview itemTable
	entryPreview itemTable
	// previewEntry is true when an Entry is focused, false when a Group is focused
	previewEntry bool
	styles       table.Styles
	path         []int
	err          error

	windowWidth  int
	windowHeight int

	database *keepass.Database
}

func NewNavigate(database *keepass.Database, windowWidth, windowHeight int) Navigate {
	n := Navigate{
		styles:       table.DefaultStyles(),
		path:         []int{0},
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		database:     database,
	}
	n.parent = newItemTable(n.styles, itemViewColumns)
	n.selector = newItemTable(n.styles, itemViewColumns, table.WithFocused(true))
	n.groupPreview = newItemTable(n.styles, itemViewColumns)
	n.entryPreview = newItemTable(table.Styles{
		Header: n.styles.Header.Copy(),
		Cell: n.styles.Cell.Copy(),
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
		n.parent.Load(n.database.Parsed(), n.path[:len(n.path)-1])
		n.parent.SetCursor(n.path[len(n.path)-1])
	}

	n.selector.Load(n.database.Parsed(), n.path)
	n.selector.SetCursor(0)

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
		n.groupPreview.LoadGroup(item)
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
	if (n.previewEntry) {
		preview = n.entryPreview.View()
	} else {
		preview = n.groupPreview.View()
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, n.parent.View(), n.selector.View(), preview)
}
