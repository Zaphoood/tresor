package tui

import (
	"fmt"

	"github.com/Zaphoood/tresor/lib/keepass/parser"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	GROUP_PLACEH     = "(No entries)"
	TITLE_PLACEH     = "(No title)"
	ENCRYPTED_PLACEH = "******"
)

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

//type sizedColumn struct {
//	name  string
//	width int
//}

type itemTable struct {
	table.Model
	styles      table.Styles
	stylesEmpty table.Styles
	columns     []table.Column
}

func newItemTable(styles table.Styles, columns []table.Column, options ...table.Option) itemTable {
	return itemTable{
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
	}
}

// Set size will set all columns to their given size and additionally scale oone column with width 0 dynamically
// to fit the width of the table. Don't set the width to 0 for more than one column -- it won't work
func (t *itemTable) SetSize(width, height int) {
	t.SetWidth(width)
	t.SetHeight(height)
	totalFixed := 0
	dynamicIndex := -1
	for i, column := range t.columns {
		if column.Width == 0 {
			dynamicIndex = i
		}
		totalFixed += column.Width
	}

	newColumns := make([]table.Column, len(t.columns))
	for i := range newColumns {
		newColumns[i] = table.Column{
			Title: t.columns[i].Title,
		}
		if i == dynamicIndex {
			newColumns[i].Width = width - totalFixed - 2*t.styles.Header.GetPaddingLeft() - 2*t.styles.Header.GetPaddingLeft()
		} else {
			newColumns[i].Width = t.columns[i].Width
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

func (t *itemTable) Load(d *parser.Document, path []int, lastSelected *map[string]int) {
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
	t.LoadGroup(group, lastSelected)
}

func (t *itemTable) LoadGroup(group parser.Group, lastSelected *map[string]int) {
	if len(group.Groups) == 0 && len(group.Entries) == 0 {
		t.SetRows([]table.Row{
			{GROUP_PLACEH, ""},
		})
		t.SetStyles(t.stylesEmpty)
	} else {
		t.SetItems(group.Groups, group.Entries)
		t.SetStyles(t.styles)
		index, exists := (*lastSelected)[group.UUID]
		if exists && index < len(t.Model.Rows()) {
			t.SetCursor(index)
		} else {
			t.SetCursor(0)
		}
	}

}

func (t *itemTable) SetItems(groups []parser.Group, entries []parser.Entry) {
	rows := make([]table.Row, len(groups)+len(entries))
	for i, group := range groups {
		rows[i] = table.Row{group.Name, fmt.Sprint(len(group.Groups) + len(group.Entries))}
	}
	for i, entry := range entries {
		title := entry.TryGet("Title", TITLE_PLACEH).Chardata
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
			value = ENCRYPTED_PLACEH
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
			rows = append(rows, table.Row{field.Key, ENCRYPTED_PLACEH})
		} else {
			rows = append(rows, table.Row{field.Key, field.Value.Chardata})
		}
	}
	t.SetRows(rows)
}
