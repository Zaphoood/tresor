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

const TABLE_SPACING = 1

var tablePadding lipgloss.Style = lipgloss.NewStyle().PaddingRight(TABLE_SPACING)

type Navigate struct {
	parent       groupTable
	selector     groupTable
	groupPreview groupTable
	entryPreview entryTable
	lastCursor   map[string]string
	cmdLine      CommandLine

	search        []string
	searchIndex   int
	searchForward bool

	path []string
	err  error

	windowWidth  int
	windowHeight int

	database *database.Database
}

func NewNavigate(database *database.Database, windowWidth, windowHeight int) Navigate {
	styles := table.Styles{
		Header: lipgloss.NewStyle().Bold(true),
		Cell:   lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle().
			Reverse(true).
			Bold(true).
			Foreground(lipgloss.Color("#9dcbf4")),
	}
	n := Navigate{
		path:         []string{},
		lastCursor:   make(map[string]string),
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		database:     database,
	}
	n.cmdLine = NewCommandLine()
	n.parent = newGroupTable(styles, true, false)
	n.selector = newGroupTable(styles, true, true, table.WithFocused(true))
	n.groupPreview = newGroupTable(styles, true, false)
	n.entryPreview = newEntryTable(table.Styles{
		Header:   styles.Header.Copy(),
		Cell:     styles.Cell.Copy(),
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
	totalWidth := n.windowWidth - 2*TABLE_SPACING
	totalHeight := n.windowHeight - n.cmdLine.GetHeight()
	selectorWidth := int(float64(totalWidth) * 0.3)
	previewWidth := int(float64(totalWidth) * 0.5)
	parentWidth := totalWidth - selectorWidth - previewWidth
	height := totalHeight

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

func (n *Navigate) saveLastSelected() {
	currentGroupUUID := n.parent.FocusedUUID()
	if len(currentGroupUUID) == 0 {
		currentGroupUUID = n.selector.FocusedUUID()
	}
	n.database.Parsed().Meta.LastSelectedGroup = currentGroupUUID
}

// getFocusedItem returns the currently focused database item if it exists, otherwise nil
func (n *Navigate) getFocusedItem() *parser.Item {
	focused := n.focusedUUID()
	if len(focused) == 0 {
		return nil
	}
	item, err := n.database.Parsed().GetItem(append(n.path, focused))
	if err != nil {
		log.Printf("ERROR: Failed to get focused item: %s\n", err)
		return nil
	}
	return &item
}

func (n *Navigate) updatePreview() {
	focusedItem := n.getFocusedItem()
	if focusedItem == nil {
		return
	}
	switch focusedItem := (*focusedItem).(type) {
	case parser.Group:
		n.groupPreview.LoadGroup(focusedItem, &n.lastCursor)
	case parser.Entry:
		n.entryPreview.LoadEntry(focusedItem, n.database)
	default:
		log.Printf("ERROR in updatePreview: Expected Group or Entry from GetItem()")
		return
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
	focusedItem := n.getFocusedItem()
	if focusedItem == nil {
		return nil
	}
	focusedEntry, ok := (*focusedItem).(parser.Entry)
	if !ok {
		return nil
	}
	unlocked, err := focusedEntry.Get("Password")
	if err != nil {
		log.Printf("Failed to get Password for '%s'\n", focusedEntry.UUID)
		return nil
	}
	notifyChange := clipboard.Write(clipboard.FmtText, []byte(unlocked.Inner))
	n.cmdLine.SetMessage(fmt.Sprintf("Copied to clipboard. (Clearing in %d seconds)", CLEAR_CLIPBOARD_DELAY))

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
	n.saveLastSelected()
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

func (n *Navigate) handleSearch(query string, reverse bool) {
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
		return
	}
	n.searchForward = !reverse
	if reverse {
		n.searchIndex = len(n.search) - 1
	} else {
		n.searchIndex = 0
	}
	n.selector.SetFocusToUUID(n.search[n.searchIndex])
}

func (n *Navigate) nextSearchResult() {
	if n.searchForward {
		n.incSearchIndex()
	} else {
		n.redSearchIndex()
	}
}

func (n *Navigate) previousSearchResult() {
	if n.searchForward {
		n.redSearchIndex()
	} else {
		n.incSearchIndex()
	}
}

func (n *Navigate) incSearchIndex() {
	if len(n.search) == 0 {
		return
	}
	n.searchIndex = (n.searchIndex + 1) % len(n.search)
	n.selector.SetFocusToUUID(n.search[n.searchIndex])
}

func (n *Navigate) redSearchIndex() {
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
	case groupTableCursorChanged:
		n.updatePreview()
	case commandInputMsg:
		cmds = append(cmds, n.handleCommand(msg.cmd))
	case searchInputMsg:
		n.handleSearch(msg.query, msg.reverse)
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
		if !n.cmdLine.Focused() {
			cmds = append(cmds, n.handleKey(msg))
		}
		n.cmdLine, cmd = n.cmdLine.Update(msg)
		cmds = append(cmds, cmd)
		if n.cmdLine.Focused() {
			n.selector.model.Blur()
		} else {
			n.selector.model.Focus()
		}
	}
	n.selector, cmd = n.selector.Update(msg)
	cmds = append(cmds, cmd)

	return n, tea.Batch(cmds...)
}

func (n *Navigate) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		n.cmdLine.SetMessage("Type  :q  and press <Enter> to exit tresor")
	case "y":
		return n.copyToClipboard()
	case "l":
		n.moveRight()
	case "h":
		n.moveLeft()
	case "n":
		n.nextSearchResult()
	case "N":
		n.previousSearchResult()
	}
	return nil
}

func (n Navigate) View() string {
	preview := ""
	focusedItem := n.getFocusedItem()
	if focusedItem != nil {
		switch (*focusedItem).(type) {
		case parser.Group:
			preview = n.groupPreview.View()
		case parser.Entry:
			preview = n.entryPreview.View()
		}
	}
	tables := lipgloss.JoinHorizontal(
		lipgloss.Top,
		tablePadding.Render(n.parent.View()),
		tablePadding.Render(n.selector.View()),
		preview,
	)
	return lipgloss.JoinVertical(lipgloss.Left, tables, n.cmdLine.View())
}
