package tui

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/parser"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	EMPTY_GROUP_PLACEHOLDER = "(No entries)"
	NO_TITLE_PLACEHOLDER    = "(No title)"
	NUM_COL_WIDTH           = 3
)

var numberStyle lipgloss.Style = lipgloss.NewStyle().
	Width(NUM_COL_WIDTH).
	AlignHorizontal(lipgloss.Right)

type entryField struct {
	key          string
	displayName  string
	defaultValue string
}

type groupTable struct {
	model              table.Model
	styles             table.Styles
	stylesEmpty        table.Styles
	sorted             bool
	notifyCursorChange bool
	uuid               string
	// items is a list of copies of the database items currently being displayed;
	// only metadata is copied, not sub-items
	items []parser.Item
}

func newGroupTable(styles table.Styles, sorted bool, notifyCursorChange bool, options ...table.Option) groupTable {
	return groupTable{
		model:  table.New(append(options, table.WithStyles(styles))...),
		styles: styles,
		stylesEmpty: table.Styles{
			Header:   styles.Header,
			Cell:     styles.Cell,
			Selected: styles.Cell.Copy(),
		},
		sorted:             sorted,
		notifyCursorChange: notifyCursorChange,
		uuid:               "",
		items:              []parser.Item{},
	}
}

func (t *groupTable) Resize(width, height int) {
	t.model.SetWidth(width)
	t.model.SetHeight(height)
	frameWidth, _ := t.styles.Header.GetFrameSize()

	t.model.SetColumns([]table.Column{
		{Title: "Name", Width: width - 2*frameWidth - NUM_COL_WIDTH},
		{Title: "Val", Width: NUM_COL_WIDTH},
	})
}

func (t *groupTable) SetSorted(v bool) {
	t.sorted = v
}

func (t *groupTable) Sorted() bool {
	return t.sorted
}

func (t *groupTable) Clear() {
	// Must set empty row, in order for truncateHeader to work
	// Otherwise an empty string would be returned from View(), which messes up the formatting
	t.model.SetRows([]table.Row{{"", ""}})
	t.items = []parser.Item{}
	t.model.SetStyles(t.stylesEmpty)
	t.uuid = ""
}

func (t groupTable) Update(msg tea.Msg) (groupTable, tea.Cmd) {
	if !t.Focused() {
		return t, nil
	}

	var cmd tea.Cmd
	oldCursor := t.model.Cursor()
	t.model, cmd = t.model.Update(msg)
	if t.notifyCursorChange && oldCursor != t.model.Cursor() {
		return t, tea.Batch(cmd, func() tea.Msg { return groupTableCursorChanged{} })
	}
	return t, cmd
}

func (t *groupTable) View() string {
	return truncateHeader(t.model.View())
}

func (t *groupTable) Focus() {
	t.model.Focus()
}

func (t *groupTable) Blur() {
	t.model.Blur()
}

func (t *groupTable) Focused() bool {
	return t.model.Focused()
}

func (t *groupTable) Load(d *parser.Document, path []string) {
	item, err := d.GetItem(path)
	if err != nil {
		t.Clear()
		t.model.SetRows([]table.Row{
			{err.Error(), ""},
		})
		log.Println(err)
		return
	}
	group, ok := item.(parser.Group)
	if !ok {
		t.Clear()
		log.Println("ERROR: Cannot load item that is not a Group into groupTable")
		return
	}
	t.LoadGroup(group)
}

// LoadGroup loads a given `parser.Group` as the table content
func (t *groupTable) LoadGroup(group parser.Group) {
	t.model.SetStyles(t.styles)
	if len(group.Groups)+len(group.Entries) == 0 {
		t.Clear()
		t.model.SetRows([]table.Row{
			{EMPTY_GROUP_PLACEHOLDER, ""},
		})
		return
	}
	t.uuid = group.UUID
	t.LoadItems(group.Groups, group.Entries)
}

// LoadLastCursor sets the cursor to the last remembered position
func (t *groupTable) LoadLastCursor(lastCursors *map[string]string) error {
	if len(t.uuid) == 0 {
		t.model.SetCursor(0)
		return nil
	}
	lastCursor, ok := (*lastCursors)[t.uuid]
	if !ok {
		t.model.SetCursor(0)
		return nil
	}
	for i, item := range t.items {
		if item.GetUUID() == lastCursor {
			t.model.SetCursor(i)
			return nil
		}
	}
	return fmt.Errorf("ERROR: Failed to find last selected item '%s' in group '%s'", lastCursor, t.uuid)
}

func (t *groupTable) LoadItems(groups []parser.Group, entries []parser.Entry) {
	rows := make([]table.Row, 0, len(groups)+len(entries))
	t.items = make([]parser.Item, 0, len(groups)+len(entries))
	groupsSorted := make([]*parser.Group, 0, len(groups))
	entriesSorted := make([]*parser.Entry, 0, len(entries))
	for i := range groups {
		groupsSorted = append(groupsSorted, &groups[i])
	}
	for i := range entries {
		entriesSorted = append(entriesSorted, &entries[i])
	}
	if t.sorted {
		sort.Slice(groupsSorted, func(i, j int) bool {
			return strings.ToLower(groupsSorted[i].Name) < strings.ToLower(groupsSorted[j].Name)
		})
		sort.Slice(entriesSorted, func(i, j int) bool {
			firstTitle, err := entriesSorted[i].Get("Title")
			if err != nil {
				return true
			}
			secondTitle, err := entriesSorted[j].Get("Title")
			if err != nil {
				return false
			}
			return strings.ToLower(firstTitle.Inner) < strings.ToLower(secondTitle.Inner)
		})
	}
	for _, group := range groupsSorted {
		rows = append(rows, table.Row{group.Name, numberStyle.Render(fmt.Sprint(len(group.Groups) + len(group.Entries)))})
		t.items = append(t.items, group.CopyMeta())
	}
	for _, entry := range entriesSorted {
		title := entry.TryGet("Title", NO_TITLE_PLACEHOLDER)
		if len(title) == 0 {
			title = NO_TITLE_PLACEHOLDER
		}
		rows = append(rows, table.Row{title, ""})
		t.items = append(t.items, entry.CopyMeta())
	}
	t.model.SetRows(rows)
}

func (t *groupTable) FindAll(predicate func(parser.Item) bool) []string {
	uuids := []string{}
	for _, item := range t.items {
		if predicate(item) {
			uuids = append(uuids, item.GetUUID())
		}
	}
	return uuids
}

func (t *groupTable) FocusedUUID() string {
	if len(t.items) == 0 {
		return ""
	}
	return t.items[t.model.Cursor()].GetUUID()
}

func (t *groupTable) SetCursorToUUID(uuid string) (tea.Cmd, error) {
	if len(t.items) == 0 {
		return nil, fmt.Errorf("Failed set cursor to UUID %s: Group is empty", uuid)
	}
	for i, item := range t.items {
		if uuid == item.GetUUID() {
			if i == t.model.Cursor() {
				return nil, nil
			}
			t.model.SetCursor(i)
			tableFocusCursor(&t.model)
			return func() tea.Msg { return groupTableCursorChanged{} }, nil
		}
	}
	return nil, fmt.Errorf("Failed to set cursor to UUID %s: Not found", uuid)
}
