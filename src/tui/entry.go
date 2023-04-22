package tui

import (
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/Zaphoood/tresor/src/keepass/parser"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

const ENCRYPTED_PLACEH = "••••••"

var defaultEntryFields []entryField = []entryField{
	{"Title", "Title", TITLE_PLACEH},
	{"UserName", "Username", ""},
	{"Password", "Password", ""},
}

type entryTable struct {
	model         table.Model
	stylesFocused table.Styles
	stylesBlurred table.Styles
}

func newEntryTable(stylesFocused table.Styles, stylesBlurred table.Styles, options ...table.Option) entryTable {
	return entryTable{
		model:         table.New(append(options, table.WithStyles(stylesBlurred))...),
		stylesFocused: stylesFocused,
		stylesBlurred: stylesBlurred,
	}
}

func (t *entryTable) Resize(width, height int) {
	t.model.SetWidth(width)
	t.model.SetHeight(height)
	frameWidth, _ := t.stylesFocused.Header.GetFrameSize()
	firstColWidth := (width - frameWidth) * 4 / 10
	secondColWidth := width - firstColWidth - 2*frameWidth
	newColumns := []table.Column{
		// These Titles don't matter, since table headers will be truncated anyway
		{Title: "Key", Width: firstColWidth},
		{Title: "Value", Width: secondColWidth},
	}

	t.model.SetColumns(newColumns)
}

func (t *entryTable) LoadEntry(entry parser.Entry, d *database.Database) {
	rows := make([]table.Row, 0, len(entry.Strings))
	visited := make(map[string]struct{})
	var value string
	for _, field := range defaultEntryFields {
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
		if _, skip := visited[field.Key]; skip {
			continue
		}
		if field.Value.Protected {
			value = ENCRYPTED_PLACEH
		} else {
			value = field.Value.Inner
		}
		rows = append(rows, table.Row{field.Key, value})
	}
	t.model.SetRows(rows)
}

func (t entryTable) Update(msg tea.Msg) (entryTable, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return t, cmd
}

func (t entryTable) View() string {
	return truncateHeader(t.model.View())
}

func (t *entryTable) Focus() {
	t.model.SetCursor(0)
	t.model.Focus()
	t.model.SetStyles(t.stylesFocused)
}

func (t *entryTable) Blur() {
	t.model.Blur()
	t.model.SetStyles(t.stylesBlurred)
}

func (t *entryTable) Focused() bool {
	return t.model.Focused()
}

// truncateHeader removes the header of a bubbles table by
// deleting everything up to (and including) the first newline
func truncateHeader(s string) string {
	split := strings.SplitN(s, "\n", 2)
	if len(split) < 2 {
		return split[0]
	}
	return split[1]
}
