package tui

import (
	"fmt"
	"log"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/Zaphoood/tresor/src/keepass/parser"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"
)

/* Model for navigating the Database in order to view and edit entries */

const CLEAR_CLIPBOARD_DELAY = 10

type Navigate struct {
	parent       groupTable
	selector     groupTable
	groupPreview groupTable
	entryPreview entryTable
	focusedItem  parser.Item
	lastCursor   map[string]string
	cmdLine      CommandLine

	search      []string
	searchIndex int

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
	n.cmdLine = NewCommandLine()
	n.parent = newGroupTable(n.styles, true)
	n.selector = newGroupTable(n.styles, true, table.WithFocused(true))
	n.groupPreview = newGroupTable(n.styles, true)
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
	// Reset search results
	n.search = []string{}
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
		n.groupPreview.LoadGroup(item, &n.lastCursor)
	case parser.Entry:
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

func (n *Navigate) handleCommand(cmd []string) tea.Cmd {
	if len(cmd) == 0 {
		return nil
	}
	switch cmd[0] {
	case "q":
		return n.handleQuitCmd(cmd)
	case "w":
		return n.handleSaveCmd(cmd, false)
	case "wq", "x":
		return n.handleSaveCmd(cmd, true)
	case "e":
		return n.handleEditCmd(cmd)
	default:
		n.cmdLine.SetMessage(fmt.Sprintf("Not a command: %s", cmd[0]))
		return nil
	}
}

func (n *Navigate) handleQuitCmd(cmd []string) tea.Cmd {
	if len(cmd) > 1 && len(cmd[1]) > 1 {
		n.cmdLine.SetMessage("Error: Too many arguments")
		return nil
	}
	return func() tea.Msg { return clearClipboardAndQuitMsg{} }
}

func (n *Navigate) handleSaveCmd(cmd []string, quit bool) tea.Cmd {
	if len(cmd) > 2 {
		n.cmdLine.SetMessage("Error: Too many arguments")
		return nil
	}

	var andThen tea.Cmd
	if quit {
		andThen = func() tea.Msg { return clearClipboardAndQuitMsg{} }
	}
	path := ""
	if len(cmd) == 2 {
		path = cmd[1]
	}
	n.cmdLine.SetMessage("Saving...")
	return saveToPathCmd(n.database, path, andThen)
}

func (n *Navigate) handleEditCmd(cmd []string) tea.Cmd {
	if len(cmd) > 2 {
		n.cmdLine.SetMessage("Error: Too many arguments")
		return nil
	}
	path := n.database.Path()
	if len(cmd) == 2 {
		path = cmd[1]
	}
	n.cmdLine.SetMessage("Reloading...")
	return fileSelectedCmd(path)
}

func (n *Navigate) handleSearch(query string) tea.Cmd {
	n.search = n.selector.FindAll(func(item parser.Item) bool {
		switch item := item.(type) {
		case parser.Group:
			return strings.Contains(strings.ToLower(item.Name), strings.ToLower(query))
		case parser.Entry:
			return strings.Contains(strings.ToLower(item.TryGet("Title", "")), strings.ToLower(query))
		}
		return false
	})
	if len(n.search) == 0 {
		n.cmdLine.SetMessage(fmt.Sprintf("Not found: %s", query))
	} else {
		n.searchIndex = 0
		n.selector.SetFocusToUUID(n.search[0])
	}
	return nil
}

func (n *Navigate) nextSearchResult() {
	if len(n.search) == 0 {
		return
	}
	n.searchIndex = (n.searchIndex + 1) % len(n.search)
	n.selector.SetFocusToUUID(n.search[n.searchIndex])
}

func (n *Navigate) previousSearchResult() {
	if len(n.search) == 0 {
		return
	}
	n.searchIndex = (n.searchIndex + len(n.search) - 1) % len(n.search)
	n.selector.SetFocusToUUID(n.search[n.searchIndex])
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
	case clearClipboardAndQuitMsg:
		clearClipboard()
		return n, tea.Quit
	case commandInputMsg:
		cmds = append(cmds, n.handleCommand(msg.cmd))
	case searchInputMsg:
		cmds = append(cmds, n.handleSearch(msg.query))
	case saveDoneMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Saved to %s", msg.path))
		cmds = append(cmds, msg.andThen)
	case saveFailedMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Error while saving: %s", msg.err))
	case loadFailedMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Error while loading: %s", msg.err))
	case tea.WindowSizeMsg:
		n.windowWidth = msg.Width
		n.windowHeight = msg.Height
		n.resizeAll()
		return n, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		if !n.cmdLine.IsInputActive() {
			switch msg.String() {
			case "ctrl+c":
				n.cmdLine.SetMessage("Type  :q  and press <Enter> to exit tresor")
			case "y":
				cmd := n.copyToClipboard()
				return n, cmd
			case "l":
				n.moveRight()
			case "h":
				n.moveLeft()
			case "n":
				n.nextSearchResult()
			case "N":
				n.previousSearchResult()
			}
		}
		n.cmdLine, cmd = n.cmdLine.Update(msg)
		cmds = append(cmds, cmd)
	}
	if !n.cmdLine.IsInputActive() {
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
