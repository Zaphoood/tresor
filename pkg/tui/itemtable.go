package tui

import (
	"fmt"

	"github.com/Zaphoood/tresor/lib/keepass/parser"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type sizedColumn struct {
	name  string
	width int
}

type entryField struct {
	key          string
	displayName  string
	defaultValue string
}

var defaultFields []entryField = []entryField{
	{"Title", "Title", "(No title)"},
	{"UserName", "Username", ""},
	{"Password", "Password", ""},
}

type itemTable struct {
	table.Model
	styles  table.Styles
	columns []sizedColumn
}

func newItemTable(styles table.Styles, columns []sizedColumn, options ...table.Option) itemTable {
	return itemTable{
		Model:   table.New(append(options, table.WithStyles(styles))...),
		styles:  styles,
		columns: columns,
	}
}

// Set size will set fixed columns to their size and scale a column with width 0 dynamically
// Don't set the width to 0 for more than one column -- it won't work
func (t *itemTable) SetSize(width, height int) {
	t.SetWidth(width)
	t.SetHeight(height)
	totalFixed := 0
	dynamicIndex := -1
	for i, column := range t.columns {
		if column.width == 0 {
			dynamicIndex = i
		}
		totalFixed += column.width
	}

	newColumns := make([]table.Column, len(t.columns))
	for i := range newColumns {
		newColumns[i] = table.Column{
			Title: t.columns[i].name,
		}
		if i == dynamicIndex {
			newColumns[i].Width = width - totalFixed - 2*t.styles.Header.GetPaddingLeft() - 2*t.styles.Header.GetPaddingLeft()
		} else {
			newColumns[i].Width = t.columns[i].width
		}
	}

	t.SetColumns(newColumns)
}

func (t *itemTable) Clear() {
	t.SetRows([]table.Row{})
}

func (t *itemTable) Init() tea.Cmd {
	return nil
}

func (t itemTable) Update(msg tea.Msg) (itemTable, tea.Cmd) {
	var cmd tea.Cmd
	t.Model, cmd = t.Model.Update(msg)
	return t, cmd
}

func (t *itemTable) View() string {
	return t.Model.View()
}

func (t *itemTable) Load(d *parser.Document, path []int) {
	item, err := d.GetItem(path)
	if err != nil {
		t.SetRows([]table.Row{
			{err.Error(), ""},
		})
	}
	group, ok := item.(parser.Group)
	if !ok {
		t.Clear()
		return
	}
	if len(group.Groups) == 0 && len(group.Entries) == 0 {
		t.SetRows([]table.Row{
			{"(No entries)", ""},
		})
	} else {
		t.SetItems(group.Groups, group.Entries)
	}
}

func (t *itemTable) LoadGroup(group parser.Group) {
	t.SetItems(group.Groups, group.Entries)
}

func (t *itemTable) SetItems(groups []parser.Group, entries []parser.Entry) {
	rows := make([]table.Row, len(groups)+len(entries))
	for i, group := range groups {
		rows[i] = table.Row{group.Name, fmt.Sprint(len(group.Groups) + len(group.Entries))}
	}
	for i, entry := range entries {
		title := entry.TryGet("Title", "(No title)").Chardata
		rows[len(groups)+i] = table.Row{title, ""}
	}
	t.SetRows(rows)
}

func (t *itemTable) LoadEntry(entry parser.Entry) {
	rows := make([]table.Row, 0, len(entry.Strings))
	visited := make(map[string]struct{})
	for _, field := range defaultFields {
		r := entry.TryGet(field.key, field.defaultValue)
		var value string
		if r.IsProtected() {
			value = "******"
		} else {
			value = r.Chardata
		}
		rows = append(rows, table.Row{field.displayName, value})
		visited[field.key] = struct{}{}
	}
	for _, field := range entry.Strings {
		if _, v := visited[field.Key]; v {
			continue
		}
		if field.Value.IsProtected() {
			rows = append(rows, table.Row{field.Key, "******"})
		} else {
			rows = append(rows, table.Row{field.Key, field.Value.Chardata})
		}
	}
	t.SetRows(rows)
}
