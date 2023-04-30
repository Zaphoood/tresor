package tui

import (
	"fmt"
	"log"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/Zaphoood/tresor/src/keepass/parser"
	"github.com/Zaphoood/tresor/src/keepass/undo"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* Model for navigating the Database in order to view and edit entries */

const TABLE_SPACING = 1

var tablePadding lipgloss.Style = lipgloss.NewStyle().PaddingRight(TABLE_SPACING)

type Navigate struct {
	parent       groupTable
	selector     groupTable
	groupPreview groupTable
	entryPreview entryTable
	lastCursors  map[string]string
	cmdLine      CommandLine

	search        []string
	searchIndex   int
	searchForward bool

	path []string
	err  error

	windowWidth  int
	windowHeight int

	database *database.Database
	undoman  undo.UndoManager[parser.Document]
}

func NewNavigate(database *database.Database, windowWidth, windowHeight int) Navigate {
	tableStyles := table.Styles{
		Header: lipgloss.NewStyle().Bold(true),
		Cell:   lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle().
			Reverse(true).
			Bold(true).
			Foreground(lipgloss.Color("#9dcbf4")),
	}
	tableStylesBlurred := table.Styles{
		Header:   tableStyles.Header.Copy(),
		Cell:     tableStyles.Cell.Copy(),
		Selected: lipgloss.NewStyle(),
	}
	n := Navigate{
		path:         []string{},
		lastCursors:  make(map[string]string),
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		database:     database,
		undoman:      undo.NewUndoManager[parser.Document](),
	}
	n.cmdLine = NewCommandLine()
	n.parent = newGroupTable(tableStyles, true, false)
	n.selector = newGroupTable(tableStyles, true, true, table.WithFocused(true))
	n.groupPreview = newGroupTable(tableStyles, true, false)
	n.entryPreview = newEntryTable(
		tableStyles,
		tableStylesBlurred,
	)

	n.resizeAll()
	n.reopenLastGroup()
	n.loadAllTables(true)

	initClipboard()

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

func (n *Navigate) loadAllTables(updateCursor bool) {
	if len(n.path) == 0 {
		n.parent.Clear()
	} else {
		n.parent.Load(n.database.Parsed(), n.path[:len(n.path)-1])
		if updateCursor {
			err := n.parent.LoadLastCursor(&n.lastCursors)
			if err != nil {
				log.Println(err)
			}
		}
	}

	n.selector.Load(n.database.Parsed(), n.path)
	if updateCursor {
		err := n.selector.LoadLastCursor(&n.lastCursors)
		if err != nil {
			log.Println(err)
		}
	}

	n.loadPreviewTable(updateCursor)

	// Reset search results
	n.search = []string{}
}

func (n *Navigate) loadPreviewTable(updateCursor bool) {
	focusedItem := n.getFocusedItem()
	if focusedItem == nil {
		return
	}
	switch focusedItem := (*focusedItem).(type) {
	case parser.Group:
		n.groupPreview.LoadGroup(focusedItem)
		if !updateCursor {
			return
		}
		err := n.groupPreview.LoadLastCursor(&n.lastCursors)
		if err != nil {
			log.Println(err)
		}
	case parser.Entry:
		n.entryPreview.LoadEntry(focusedItem, n.database)
	default:
		log.Printf("ERROR in updatePreview: Expected Group or Entry as focused item")
		return
	}
}

func (n *Navigate) reopenLastGroup() {
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
		n.lastCursors[path[i]] = path[i+1]
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
	focused := n.selector.FocusedUUID()
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

func (n *Navigate) moveLeft() {
	if len(n.path) == 0 {
		return
	}
	n.rememberCursor()
	n.path = n.path[:len(n.path)-1]
	n.loadAllTables(true)
}

func (n *Navigate) moveRight() {
	newPath := append(n.path, n.selector.FocusedUUID())
	focusedItem, err := n.database.Parsed().GetItem(newPath)
	if err != nil {
		return
	}

	switch focusedItem.(type) {
	case parser.Group:
		n.rememberCursor()
		n.path = newPath
		n.loadAllTables(true)
	case parser.Entry:
		n.selector.Blur()
		n.entryPreview.Focus()
	}
}

func (n *Navigate) rememberCursor() {
	if parentFocusedUUID := n.parent.FocusedUUID(); len(parentFocusedUUID) > 0 {
		n.lastCursors[parentFocusedUUID] = n.selector.FocusedUUID()
	}
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

	cmd, err := copyEntryFieldToClipboard(focusedEntry, "Password", CLEAR_CLIPBOARD_DELAY)
	if err != nil {
		log.Println(err)
		return nil
	}
	return cmd
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

func (n *Navigate) handleSearch(query string, reverse bool) tea.Cmd {
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
		return nil
	}
	n.searchForward = !reverse
	if reverse {
		n.searchIndex = len(n.search) - 1
	} else {
		n.searchIndex = 0
	}

	cmd, err := n.selector.SetCursorToUUID(n.search[n.searchIndex])
	if err != nil {
		log.Println(err)
		return nil
	}

	return cmd
}

func (n *Navigate) nextSearchResult() {
	if n.searchForward {
		n.incSearchIndex()
	} else {
		n.decSearchIndex()
	}
}

func (n *Navigate) previousSearchResult() {
	if n.searchForward {
		n.decSearchIndex()
	} else {
		n.incSearchIndex()
	}
}

func (n *Navigate) incSearchIndex() tea.Cmd {
	if len(n.search) == 0 {
		return nil
	}
	n.searchIndex = (n.searchIndex + 1) % len(n.search)

	cmd, err := n.selector.SetCursorToUUID(n.search[n.searchIndex])
	if err != nil {
		log.Println(err)
		return nil
	}

	return cmd
}

func (n *Navigate) decSearchIndex() tea.Cmd {
	if len(n.search) == 0 {
		return nil
	}
	n.searchIndex = (n.searchIndex + len(n.search) - 1) % len(n.search)

	cmd, err := n.selector.SetCursorToUUID(n.search[n.searchIndex])
	if err != nil {
		log.Println(err)
		return nil
	}

	return cmd
}

func (n Navigate) Init() tea.Cmd {
	return nil
}

func (n Navigate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case clearClipboardMsg:
		clearClipboard()
	case clearClipboardAndQuitMsg:
		clearClipboard()
		return n, tea.Quit
	case groupTableCursorChanged:
		n.loadPreviewTable(true)
	case focusItemMsg:
		path, found := n.database.Parsed().FindPath(msg.uuid)
		if !found {
			log.Printf("ERROR: Cannot focus on item '%s' (not found)", msg.uuid)
		}
		if len(path) > 0 {
			n.path = path[:len(path)-1]
			n.loadAllTables(false)
			cmd, err := n.selector.SetCursorToUUID(msg.uuid)
			if err != nil {
				log.Println(err)
			}
			return n, cmd
		}
	case commandInputMsg:
		cmd = n.handleCommand(msg.cmd)
		return n, cmd
	case searchInputMsg:
		cmd = n.handleSearch(msg.query, msg.reverse)
		return n, cmd
	case saveDoneMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Saved to %s", msg.path))
		return n, msg.andThen
	case saveFailedMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Error while saving: %s", msg.err))
	case loadFailedMsg:
		n.cmdLine.SetMessage(fmt.Sprintf("Error while loading: %s", msg.err))
	case undoableActionMsg:
		result := n.undoman.Do(n.database.Parsed(), msg.action)
		n.loadAllTables(false)
		if cmd, ok := result.(tea.Cmd); ok {
			return n, cmd
		}
	case leaveEntryEditor:
		n.selector.Focus()
		n.entryPreview.Blur()
	case tea.WindowSizeMsg:
		n.windowWidth = msg.Width
		n.windowHeight = msg.Height
		n.resizeAll()
		return n, globalResizeCmd(msg.Width, msg.Height)
	case tea.KeyMsg:
		if n.cmdLine.Focused() {
			// Key events should not be handled by Navigate in case the command line is active
			break
		}
		if handled, cmd := n.handleKeyAnyFocus(msg); handled {
			return n, cmd
		}
		if handled, cmd := n.handleKeyCmdLineTrigger(msg); handled {
			return n, cmd
		}
		if handled, cmd := n.handleKeyDefault(msg); handled {
			return n, cmd
		}
	}

	if n.cmdLine.Focused() {
		n.cmdLine, cmd = n.cmdLine.Update(msg)
		return n, cmd
	} else if n.entryPreview.Focused() {
		n.entryPreview, cmd = n.entryPreview.Update(msg)
		return n, cmd
	} else {
		n.selector, cmd = n.selector.Update(msg)
		return n, cmd
	}
}

func (n *Navigate) handleKeyAnyFocus(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		n.cmdLine.SetMessage("Type  :q  and press <Enter> to exit tresor")
		return true, nil
	case "u":
		// TODO: handle undo and redo in separate methods
		result, err := n.undoman.Undo(n.database.Parsed())
		if err != nil {
			if _, ok := err.(undo.AtOldestChange); ok {
				n.cmdLine.SetMessage(err.Error())
			} else {
				log.Println(err)
			}
			return true, nil
		}
		n.loadAllTables(false)
		if cmd, ok := result.(tea.Cmd); ok {
			return true, cmd
		}
		return true, nil
	case "ctrl+r":
		result, err := n.undoman.Redo(n.database.Parsed())
		if err != nil {
			if _, ok := err.(undo.AtNewestChange); ok {
				n.cmdLine.SetMessage(err.Error())
			} else {
				log.Println(err)
			}
			return true, nil
		}
		n.loadAllTables(false)
		if cmd, ok := result.(tea.Cmd); ok {
			return true, cmd
		}
		return true, nil
	}
	return false, nil
}

// handleKeyDefault handles key events when no other components are focused (such as command line, entry preview)
func (n *Navigate) handleKeyDefault(msg tea.KeyMsg) (bool, tea.Cmd) {
	if n.entryPreview.Focused() {
		return false, nil
	}

	switch msg.String() {
	case "y":
		return true, n.copyToClipboard()
	case "l":
		n.moveRight()
		return true, nil
	case "h":
		n.moveLeft()
		return true, nil
	case "n":
		n.nextSearchResult()
		return true, nil
	case "N":
		n.previousSearchResult()
		return true, nil
	}
	return false, nil
}

func (n *Navigate) handleKeyCmdLineTrigger(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case PROMPT_COMMAND:
		return true, n.cmdLine.StartInput(PROMPT_COMMAND, CommandCallback)
	case PROMPT_SEARCH:
		return true, n.cmdLine.StartInput(PROMPT_SEARCH, SearchCallback(false))
	case PROMPT_REV_SEARCH:
		return true, n.cmdLine.StartInput(PROMPT_REV_SEARCH, SearchCallback(true))
	}
	return false, nil
}

func (n Navigate) View() string {
	var preview string
	focusedItem := n.getFocusedItem()
	if focusedItem == nil {
		preview = ""
	} else {
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
