package tui

import (
	"fmt"
	"log"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/Zaphoood/tresor/src/keepass/parser"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"
)

/* Model for navigating the Database in order to view and edit entries */

const CLEAR_CLIPBOARD_DELAY = 10

var itemViewColumns []table.Column = []table.Column{
	{Title: "Name", Width: 0},
	{Title: "Entries", Width: 7},
}

type Navigate struct {
	parent       groupTable
	selector     groupTable
	groupPreview groupTable
	entryPreview entryTable
	focusedItem  parser.Item
	lastCursor   map[string]string
	cmdLine      CommandLine

	path []string
	err  error

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
	n.cmdLine = NewCommandLine(n.handleCommand)
	n.parent = newGroupTable(n.styles, itemViewColumns, true)
	n.selector = newGroupTable(n.styles, itemViewColumns, true, table.WithFocused(true))
	n.groupPreview = newGroupTable(n.styles, itemViewColumns, true)
	n.entryPreview = newEntryTable(table.Styles{
		Header:   n.styles.Header.Copy(),
		Cell:     n.styles.Cell.Copy(),
		Selected: lipgloss.NewStyle(),
	})

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
	height := n.windowHeight - n.cmdLine.GetHeight()

	n.parent.Resize(parentWidth, height)
	n.selector.Resize(selectorWidth, height)
	n.groupPreview.Resize(previewWidth, height)
	n.entryPreview.Resize(previewWidth, height)
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
	notifyChange := clipboard.Write(clipboard.FmtText, []byte(unlocked.Inner))

	return scheduleClearClipboard(CLEAR_CLIPBOARD_DELAY, notifyChange)
}

func (n *Navigate) handleCommand(cmd []string) (tea.Cmd, bool, string) {
	switch cmd[0] {
	case "q":
		// TODO: Handle trailing characters
		clearClipboard()
		return tea.Quit, true, "Bye-bye!"
	case "w":
		return saveCmd(n.database), true, "Saving..."
	default:
		return nil, true, fmt.Sprintf("Not a command: %s", cmd)
	}
}

func clearClipboard() {
	clipboard.Write(clipboard.FmtText, []byte(""))
}

func (n Navigate) Init() tea.Cmd {
	return nil
}

func (n Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case clearClipboardMsg:
		clearClipboard()
	case saveDoneMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Saved to %s", msg.path))
	case saveFailedMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Error while saving: %s", msg.err))
	case tea.WindowSizeMsg:
		n.windowWidth = msg.Width
		n.windowHeight = msg.Height
		n.resizeAll()
		return n, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		if !n.cmdLine.IsInputMode() {
			switch msg.String() {
			case "ctrl+c":
				n.cmdLine.SetMessage("Type  :q  and press <Enter> to exit tresor")
			case "enter":
				cmd := n.copyToClipboard()
				return n, cmd
			case "l":
				n.moveRight()
			case "h":
				n.moveLeft()
			}
		}
		n.cmdLine, cmd = n.cmdLine.Update(msg)
		cmds = append(cmds, cmd)
	}
	if !n.cmdLine.IsInputMode() {
		cursor := n.selector.Cursor()
		n.selector, cmd = n.selector.Update(msg)
		if cursor != n.selector.Cursor() {
			n.updatePreview()
		}
		cmds = append(cmds, cmd)
	}

	return n, tea.Batch(cmds...)
}

func (n Navigate) View() string {
	var preview string
	switch n.focusedItem.(type) {
	case parser.Group:
		preview = n.groupPreview.View()
	case parser.Entry:
		preview = n.entryPreview.View()
	}
	tables := lipgloss.JoinHorizontal(lipgloss.Top, n.parent.View(), n.selector.View(), preview)
	return lipgloss.JoinVertical(lipgloss.Left, tables, n.cmdLine.View())
}
