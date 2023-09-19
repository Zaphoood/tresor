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
const ERR_TOO_FEW_ARGS = "Error: Too few arguments"
const ERR_TOO_MANY_ARGS = "Error: Too many arguments"

var tablePadding lipgloss.Style = lipgloss.NewStyle().PaddingRight(TABLE_SPACING)

type Navigate struct {
	// Shows the parent Group of the active Group (displayed by the center table)
	leftTable groupTable
	// Shows the active Group. This table's cursor can be controlled using j/k or up/down
	centerTable groupTable
	// Shows a preview of the currently selected item of the center table if it's a Group
	rightGroupTable groupTable
	// Same as above, but if the current item is an Entry
	rightEntryTable entryTable
	// Map the UUID of each group that has been visited to the UUID which was last hovered by the cursor
	lastCursors map[string]string
	cmdLine     CommandLine

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
	n.leftTable = newGroupTable(tableStyles, true, false)
	n.centerTable = newGroupTable(tableStyles, true, true, table.WithFocused(true))
	n.rightGroupTable = newGroupTable(tableStyles, true, false)
	n.rightEntryTable = newEntryTable(
		tableStyles,
		tableStylesBlurred,
	)

	n.resizeAll()
	n.reopenLastGroup()
	n.loadAllTables()

	initClipboard()

	return n
}

func (n *Navigate) resizeAll() {
	totalWidth := n.windowWidth - 2*TABLE_SPACING
	totalHeight := n.windowHeight - n.cmdLine.GetHeight()
	centerTableWidth := int(float64(totalWidth) * 0.3)
	rightTableWidth := int(float64(totalWidth) * 0.5)
	leftTableWidth := totalWidth - centerTableWidth - rightTableWidth
	height := totalHeight

	n.leftTable.Resize(leftTableWidth, height)
	n.centerTable.Resize(centerTableWidth, height)
	n.rightGroupTable.Resize(rightTableWidth, height)
	n.rightEntryTable.Resize(rightTableWidth, height)
}

func (n *Navigate) loadAllTables() {
	if len(n.path) == 0 {
		n.leftTable.Clear()
	} else {
		n.leftTable.Load(n.database.Parsed(), n.path[:len(n.path)-1])
		err := n.leftTable.LoadLastCursor(&n.lastCursors)
		if err != nil {
			log.Println(err)
		}
	}

	n.centerTable.Load(n.database.Parsed(), n.path)
	err := n.centerTable.LoadLastCursor(&n.lastCursors)
	if err != nil {
		log.Println(err)
	}

	n.loadPreviewTable()

	// Reset search results
	n.search = []string{}
}

func (n *Navigate) loadPreviewTable() {
	focusedItem := n.getFocusedItem()
	if focusedItem == nil {
		return
	}
	switch focusedItem := (*focusedItem).(type) {
	case parser.Group:
		n.rightGroupTable.LoadGroup(focusedItem)
		err := n.rightGroupTable.LoadLastCursor(&n.lastCursors)
		if err != nil {
			log.Println(err)
		}
	case parser.Entry:
		n.rightEntryTable.LoadEntry(focusedItem, n.database)
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
	currentGroupUUID := n.leftTable.FocusedUUID()
	if len(currentGroupUUID) == 0 {
		currentGroupUUID = n.centerTable.FocusedUUID()
	}
	n.database.Parsed().Meta.LastSelectedGroup = currentGroupUUID
}

// getFocusedItem returns the currently focused database item if it exists, otherwise nil
func (n *Navigate) getFocusedItem() *parser.Item {
	focused := n.centerTable.FocusedUUID()
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

func (n *Navigate) focusItem(uuid string) {
	path, found := n.database.Parsed().FindPath(uuid)
	if !found {
		log.Printf("ERROR: Cannot focus on item '%s' (not found)", uuid)
	}
	if len(path) > 0 {
		for i := 0; i < len(path)-1; i++ {
			n.lastCursors[path[i]] = path[i+1]
		}
		// Omit last item of path, which is the UUID of the item to be focused. This is because
		// the path relates to the group shown in selector, not to its selected item
		n.path = path[:len(path)-1]
		n.loadAllTables()
	}
}

func (n *Navigate) moveLeft() {
	if len(n.path) == 0 {
		return
	}
	n.path = n.path[:len(n.path)-1]
	n.loadAllTables()
}

func (n *Navigate) moveRight() {
	newPath := append(n.path, n.centerTable.FocusedUUID())
	focusedItem, err := n.database.Parsed().GetItem(newPath)
	if err != nil {
		return
	}

	switch focusedItem.(type) {
	case parser.Group:
		n.path = newPath
		n.loadAllTables()
	case parser.Entry:
		n.centerTable.Blur()
		n.rightEntryTable.Focus()
	}
}

// rememberCursor stores the currently focused UUID of the selector to table
// which maps group UUIDs to the last selected item UUID
func (n *Navigate) rememberCursor() {
	if parentFocusedUUID := n.leftTable.FocusedUUID(); len(parentFocusedUUID) > 0 {
		n.lastCursors[parentFocusedUUID] = n.centerTable.FocusedUUID()
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
	case "change":
		return n.handleChangeCmd(cmd)
	default:
		n.cmdLine.SetMessage(fmt.Sprintf("Not a command: %s", cmd[0]))
		return nil
	}
}

func (n *Navigate) handleQuitCmd(cmd []string) tea.Cmd {
	if len(cmd) > 1 && len(cmd[1]) > 1 {
		n.cmdLine.SetMessage(ERR_TOO_MANY_ARGS)
		return nil
	}
	return func() tea.Msg { return clearClipboardAndQuitMsg{} }
}

func (n *Navigate) handleSaveCmd(cmd []string, quit bool) tea.Cmd {
	if len(cmd) > 2 {
		n.cmdLine.SetMessage(ERR_TOO_MANY_ARGS)
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
		n.cmdLine.SetMessage(ERR_TOO_MANY_ARGS)
		return nil
	}
	path := n.database.Path()
	if len(cmd) == 2 {
		path = cmd[1]
	}
	n.cmdLine.SetMessage("Reloading...")
	return fileSelectedCmd(path)
}

func (n *Navigate) handleChangeCmd(cmd []string) tea.Cmd {
	if len(cmd) < 1 {
		n.cmdLine.SetMessage(ERR_TOO_FEW_ARGS)
		return nil
	}
	// TODO: This merges multiple consecutive spaces into one. This is because
	// commandline breaks along whitespace. A solution would be to either pass
	// along the original command input (or just handle command parsing here
	// instead of in command line), or alternatively allow wrapping command
	// arguments in parantheses
	newValue := strings.Join(cmd[1:], " ")

	if n.rightEntryTable.Focused() {
		return n.rightEntryTable.changeFocused(newValue)
	}

	// If right table not focused an cursor is on an Entry, change that Entry's
	// title
	focusedItem := n.getFocusedItem()
	if focusedItem == nil {
		return nil
	}
	focusedEntry, ok := (*focusedItem).(parser.Entry)
	if !ok {
		return nil
	}
	return makeChangeFieldAction(focusedEntry, "Title", newValue, focusChangedItemCmd(focusedEntry.UUID))
}

func (n *Navigate) handleSearch(query string, reverse bool) tea.Cmd {
	n.search = n.centerTable.FindAll(func(item parser.Item) bool {
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

	cmd, err := n.centerTable.SetCursorToUUID(n.search[n.searchIndex])
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

	cmd, err := n.centerTable.SetCursorToUUID(n.search[n.searchIndex])
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

	cmd, err := n.centerTable.SetCursorToUUID(n.search[n.searchIndex])
	if err != nil {
		log.Println(err)
		return nil
	}

	return cmd
}

func (n *Navigate) handleUndo() tea.Cmd {
	result, description, err := n.undoman.Undo(n.database.Parsed())
	if err != nil {
		if _, ok := err.(undo.AtOldestChange); ok {
			n.cmdLine.SetMessage(err.Error())
		} else {
			log.Println(err)
		}
		return nil
	}

	n.cmdLine.SetMessage(fmt.Sprintf("Undo: %s", description))
	n.loadAllTables()

	// An undoable action may ask for a tea.Cmd to be executed after it is undone, such as focusing a changed item
	if cmd, ok := result.(tea.Cmd); ok {
		return cmd
	}

	return nil
}

func (n *Navigate) handleRedo() tea.Cmd {
	result, description, err := n.undoman.Redo(n.database.Parsed())
	if err != nil {
		if _, ok := err.(undo.AtNewestChange); ok {
			n.cmdLine.SetMessage(err.Error())
		} else {
			log.Println(err)
		}
		return nil
	}

	n.cmdLine.SetMessage(fmt.Sprintf("Redo: %s", description))
	n.loadAllTables()

	if cmd, ok := result.(tea.Cmd); ok {
		return cmd
	}

	return nil
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
		n.loadPreviewTable()
	case focusItemMsg:
		n.focusItem(msg.uuid)
		return n, nil
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
		result, _ := n.undoman.Do(n.database.Parsed(), msg.action)
		n.loadAllTables()
		if cmd, ok := result.(tea.Cmd); ok {
			return n, cmd
		}
	case leaveEntryEditor:
		n.centerTable.Focus()
		n.rightEntryTable.Blur()
	case setCommandLineMessageMsg:
		// TODO: Consider calling it the command line's 'status' instead in order to avoid these unfortunate variable names
		n.cmdLine.SetMessage(msg.msg)
		return n, nil
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

		if n.rightEntryTable.Focused() {
			break
		}
		if handled, cmd := n.handleKeyDefault(msg); handled {
			return n, cmd
		}
	}

	if n.cmdLine.Focused() {
		n.cmdLine, cmd = n.cmdLine.Update(msg)
		return n, cmd
	} else if n.rightEntryTable.Focused() {
		n.rightEntryTable, cmd = n.rightEntryTable.Update(msg)
		return n, cmd
	} else {
		n.centerTable, cmd = n.centerTable.Update(msg)
		n.rememberCursor()
		return n, cmd
	}
}

// handleKeyAnyFocus takes care of key events that should always be handled, except if the command line is active
// All key handling functions return a boolean as their first return value which indicates wether the given key
// event was handled by this function, in which case it should not be handled by other handler functions
func (n *Navigate) handleKeyAnyFocus(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		n.cmdLine.SetMessage("Type  :q  and press <Enter> to exit tresor")
		return true, nil
	case "u":
		return true, n.handleUndo()
	case "ctrl+r":
		return true, n.handleRedo()
	}
	return false, nil
}

// handleKeyDefault handles key events when no other components are focused (such as command line, entry preview)
func (n *Navigate) handleKeyDefault(msg tea.KeyMsg) (bool, tea.Cmd) {
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
	case "c":
		// Note that we handle this keypress here even though it might seem
		// like the right entry/group table should handle it. However, the
		// selected item of the center table can also be changed, in addition,
		// this does not actually change any items but only starts the command
		// prompt. The actual change is then handled by the corresponding table.
		return true, n.cmdLine.StartInputWithValue(PROMPT_COMMAND, CommandCallback, "change ")
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
			preview = n.rightGroupTable.View()
		case parser.Entry:
			preview = n.rightEntryTable.View()
		}
	}
	tables := lipgloss.JoinHorizontal(
		lipgloss.Top,
		tablePadding.Render(n.leftTable.View()),
		tablePadding.Render(n.centerTable.View()),
		preview,
	)
	return lipgloss.JoinVertical(lipgloss.Left, tables, n.cmdLine.View())
}
