package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/Zaphoood/tresor/src/keepass/parser"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	GROUP_PLACEH     = "(No entries)"
	TITLE_PLACEH     = "(No title)"
	ENCRYPTED_PLACEH = "******"
	NUM_COL_WIDTH    = 3
)

var numberStyle lipgloss.Style = lipgloss.NewStyle().
	Width(NUM_COL_WIDTH).
	AlignHorizontal(lipgloss.Right)

type entryField struct {
	key          string
	displayName  string
	defaultValue string
}

var defaultFields []entryField = []entryField{
	{"Title", "Title", TITLE_PLACEH},
	{"UserName", "Username", ""},
	{"Password", "Password", ""},
}

type groupTable struct {
	table.Model
	styles      table.Styles
	stylesEmpty table.Styles
	columns     []table.Column
	sorted      bool
	// TODO: Instead of storing a separate list of UUIDs, store a list
	// of shallow copies of Items (includes only metadata like title etc.
	// but not supgroups, child entries or history)
	uuids []string
}

func newGroupTable(styles table.Styles, columns []table.Column, sorted bool, options ...table.Option) groupTable {
	return groupTable{
		Model:  table.New(append(options, table.WithStyles(styles))...),
		styles: styles,
		stylesEmpty: table.Styles{
			Header: styles.Header,
			Cell:   styles.Cell,
			Selected: styles.Selected.Copy().
				Foreground(styles.Cell.GetForeground()).
				Bold(false),
		},
		columns: columns,
		sorted:  sorted,
	}
}

// Resize will set all columns to their given size and additionally scale oone column with width 0 dynamically
// to fit the width of the table. Don't set the width to 0 for more than one column -- it won't work
func (t *groupTable) Resize(width, height int) {
	t.SetWidth(width)
	t.SetHeight(height)
	frameWidth, _ := t.styles.Header.GetFrameSize()
	numberColWidth := NUM_COL_WIDTH

	t.SetColumns([]table.Column{
		{Title: "Name", Width: width - 2*frameWidth - numberColWidth},
		{Title: "Val", Width: numberColWidth},
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
	t.SetRows([]table.Row{{"", ""}})
	t.uuids = []string{}
}

func (t *groupTable) Init() tea.Cmd {
	return nil
}

func (t groupTable) Update(msg tea.Msg) (groupTable, tea.Cmd) {
	var cmd tea.Cmd
	t.Model, cmd = t.Model.Update(msg)
	return t, cmd
}

func (t *groupTable) View() string {
	return truncateHeader(t.Model.View())
}

func (t *groupTable) Load(d *parser.Document, path []string, lastSelected *map[string]string) {
	item, err := d.GetItem(path)
	if err != nil {
		t.SetRows([]table.Row{
			{err.Error(), ""},
		})
		t.uuids = []string{}
	}
	group, ok := item.(parser.Group)
	if !ok {
		t.Clear()
		return
	}
	t.LoadGroup(group, lastSelected)
}

func (t *groupTable) LoadGroup(group parser.Group, lastCursors *map[string]string) {
	if len(group.Groups)+len(group.Entries) == 0 {
		t.SetRows([]table.Row{
			{GROUP_PLACEH, ""},
		})
		t.SetStyles(t.stylesEmpty)
		t.uuids = []string{}
		return
	}
	t.SetItems(group.Groups, group.Entries)
	t.SetStyles(t.styles)
	t.SetCursor(0)
	lastCursor, ok := (*lastCursors)[group.UUID]
	if !ok {
		return
	}
	for i, uuid := range t.uuids {
		if uuid == lastCursor {
			t.SetCursor(i)
		}
	}
}

func (t *groupTable) SetItems(groups []parser.Group, entries []parser.Entry) {
	rows := make([]table.Row, 0, len(groups)+len(entries))
	t.uuids = make([]string, 0, len(groups)+len(entries))
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
			firstTitle, err1 := entriesSorted[i].Get("Title")
			secondTitle, err2 := entriesSorted[j].Get("Title")
			if err1 != nil {
				return false
			} else if err2 != nil {
				return true
			}
			return strings.ToLower(firstTitle.Inner) < strings.ToLower(secondTitle.Inner)
		})
	}
	for _, group := range groupsSorted {
		rows = append(rows, table.Row{group.Name, numberStyle.Render(fmt.Sprint(len(group.Groups) + len(group.Entries)))})
		t.uuids = append(t.uuids, group.UUID)
	}
	for _, entry := range entriesSorted {
		title := entry.TryGet("Title", TITLE_PLACEH)
		rows = append(rows, table.Row{title, ""})
		t.uuids = append(t.uuids, entry.UUID)
	}
	t.SetRows(rows)
}

func (t *groupTable) FocusedUUID() string {
	if len(t.uuids) == 0 {
		return ""
	}
	return t.uuids[t.Cursor()]
}

type entryTable struct {
	table.Model
	styles table.Styles
}

func newEntryTable(styles table.Styles, options ...table.Option) entryTable {
	return entryTable{
		Model:  table.New(append(options, table.WithStyles(styles))...),
		styles: styles,
	}
}

func (t *entryTable) Resize(width, height int) {
	t.SetWidth(width)
	t.SetHeight(height)
	frameWidth, _ := t.styles.Header.GetFrameSize()
	firstColWidth := (width - frameWidth) * 4 / 10
	secondColWidth := width - firstColWidth - 2*frameWidth
	newColumns := []table.Column{
		{Title: "Key", Width: firstColWidth},
		{Title: "Value", Width: secondColWidth},
	}

	t.SetColumns(newColumns)
}

func (t *entryTable) LoadEntry(entry parser.Entry, d *database.Database) {
	rows := make([]table.Row, 0, len(entry.Strings))
	visited := make(map[string]struct{})
	var value string
	for _, field := range defaultFields {
		r, err := entry.Get(field.key)
		if err != nil {
			value = field.defaultValue
		} else if r.Protected {
			value = ENCRYPTED_PLACEH
		} else {
			value = r.Inner
		}
		rows = append(rows, table.Row{field.displayName, value})
		visited[field.key] = struct{}{}
	}
	for _, field := range entry.Strings {
		if _, v := visited[field.Key]; v {
			continue
		}
		if field.Value.Protected {
			value = ENCRYPTED_PLACEH
		} else {
			value = field.Value.Inner
		}
		rows = append(rows, table.Row{field.Key, value})
	}
	t.SetRows(rows)
}

func (t entryTable) View() string {
	return truncateHeader(t.Model.View())
}

// truncateHeader removes the header of a bubbles.Table by
// deleting everything up to (and including) the first newline
func truncateHeader(s string) string {
	split := strings.SplitN(s, "\n", 2)
	if len(split) < 2 {
		return split[0]
	}
	return split[1]
}
