package tui

import (
	"log"
	"time"

	"github.com/Zaphoood/tresor/lib/keepass/database"
	"github.com/Zaphoood/tresor/lib/keepass/parser"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"
)

/* Model for navigating the Database in order to view and edit entries */

const CLEAR_CLIPBOARD_DELAY = 3

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
	focusedItem  parser.Item
	lastCursor   map[string]string

	lastCopy time.Time
	path     []string
	err      error

	styles       table.Styles
	windowWidth  int
	windowHeight int

	database *database.Database
}

func NewNavigate(database *database.Database, windowWidth, windowHeight int) Navigate {
	n := Navigate{
		styles:       table.DefaultStyles(),
		path:         []string{},
		lastCursor:   make(map[string]string),
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		database:     database,
	}
	n.parent = newItemTable(n.styles, itemViewColumns, true)
	n.selector = newItemTable(n.styles, itemViewColumns, true, table.WithFocused(true))
	n.groupPreview = newItemTable(n.styles, itemViewColumns, true)
	n.entryPreview = newItemTable(table.Styles{
		Header:   n.styles.Header.Copy(),
		Cell:     n.styles.Cell.Copy(),
		Selected: lipgloss.NewStyle(),
	}, entryViewColumns, true)

	n.resizeAll()
	n.loadLastSelected()
	n.updateAll()

	err := clipboard.Init()
	if err != nil {
		// TODO: We should handle this more gracefully
		panic(err)
	}

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
		n.parent.Load(n.database.Parsed(), n.path[:len(n.path)-1], &n.lastCursor)
	}
	n.selector.Load(n.database.Parsed(), n.path, &n.lastCursor)
	n.updatePreview()
}

func (n *Navigate) loadLastSelected() {
	n.path = []string{n.database.Parsed().Root.Groups[0].UUID}
	lastSelected := n.database.Parsed().Meta.LastSelectedGroup
	if len(lastSelected) == 0 {
		return
	}
	path, found := n.database.Parsed().FindPath(lastSelected)
	if !found {
		return
	}
	n.path = path
	for i := 0; i < len(path)-1; i++ {
		n.lastCursor[path[i]] = path[i+1]
	}
}

func (n *Navigate) updatePreview() {
	focused := n.focusedUUID()
	if len(focused) == 0 {
		n.groupPreview.Clear()
		return
	}
	item, err := n.database.Parsed().GetItem(append(n.path, focused))
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		n.groupPreview.Clear()
		return
	}
	switch item := item.(type) {
	case parser.Group:
		// A group is focused
		n.groupPreview.LoadGroup(item, &n.lastCursor)
	case parser.Entry:
		// An entry is focused
		n.entryPreview.LoadEntry(item, n.database)
	default:
		log.Printf("ERROR in updatePreview: Expected Group or Entry from GetItem()")
		return
	}
	n.focusedItem = item
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
	selected, err := n.database.Parsed().GetItem(append(n.path, n.focusedUUID()))
	if err != nil {
		return
	}
	if _, ok := selected.(parser.Group); !ok {
		return
	}
	n.rememberSelected()
	n.path = append(n.path, n.focusedUUID())
	n.updateAll()
}

func (n *Navigate) rememberSelected() {
	if parentFocusedUUID := n.parent.FocusedUUID(); len(parentFocusedUUID) > 0 {
		n.lastCursor[parentFocusedUUID] = n.focusedUUID()
	}
}

func (n Navigate) focusedUUID() string {
	return n.selector.FocusedUUID()
}

func (n *Navigate) copyToClipboard() tea.Cmd {
	focusedEntry, ok := n.focusedItem.(parser.Entry)
	if !ok {
		return nil
	}
	unlocked, err := focusedEntry.Get("Password")
	if err != nil {
		log.Printf("Failed to get Password for '%s'\n", focusedEntry.UUID)
		return nil
	}
	clipboard.Write(clipboard.FmtText, []byte(unlocked.Inner))

	timestamp := time.Now()
	n.lastCopy = timestamp
	return scheduleClearClipboard(CLEAR_CLIPBOARD_DELAY, timestamp)
}

func (n *Navigate) clearClipboard(timestamp time.Time) {
	if n.lastCopy == timestamp {
		clipboard.Write(clipboard.FmtText, []byte(""))
	}
}

func (n Navigate) Init() tea.Cmd {
	return nil
}

func (n Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearClipboardMsg:
		n.clearClipboard(msg.timestamp)
	case tea.WindowSizeMsg:
		n.windowWidth = msg.Width
		n.windowHeight = msg.Height
		n.resizeAll()
		return n, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return n, tea.Quit
		case "enter":
			cmd := n.copyToClipboard()
			return n, cmd
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
	var preview string
	switch n.focusedItem.(type) {
	case parser.Group:
		preview = n.groupPreview.View()
	case parser.Entry:
		preview = n.entryPreview.View()
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, n.parent.View(), n.selector.View(), preview)
}
