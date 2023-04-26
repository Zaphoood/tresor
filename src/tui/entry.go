package tui

import (
	"log"
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

func isDefaultEntryField(key string) bool {
	for _, field := range defaultEntryFields {
		if field.key == key {
			return true
		}
	}
	return false
}

type entryTable struct {
	model         table.Model
	stylesFocused table.Styles
	stylesBlurred table.Styles

	entry parser.Entry
	// The keys of the currently viewed entry's string fields, in order they are displayed
	fieldKeys []string
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
	t.entry = entry
	t.fieldKeys = make([]string, 0, len(entry.Strings))
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
		t.fieldKeys = append(t.fieldKeys, field.key)
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
		t.fieldKeys = append(t.fieldKeys, field.Key)
	}
	t.model.SetRows(rows)
}

func (t entryTable) Update(msg tea.Msg) (entryTable, tea.Cmd) {
	if !t.Focused() {
		return t, nil
	}

	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "h", "esc":
			return t, func() tea.Msg { return leaveEntryEditor{} }
		case "y":
			cmd = t.copyFocusedToClipboard()
			return t, cmd
		case "d":
			cmd = t.deleteFocused()
			return t, cmd
		}
	}
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

// copyFocusedToClipboard copies the value of the currently focused field to the clipboard
func (t *entryTable) copyFocusedToClipboard() tea.Cmd {
	// We have to get the key by indexing the table rows,
	// since the display order of strings may be different
	// from the order in t.entry.Strings
	// TODO: This is a bit hacky, maybe find a less confusing solution
	key := t.fieldKeys[t.model.Cursor()]
	value, err := t.entry.Get(key)
	if err != nil {
		log.Printf("ERROR: Could not retrieve value for key '%s' of entry '%s'", key, t.entry.GetUUID())
		return nil
	}

	clipboardDelay := 0
	if value.Protected {
		clipboardDelay = CLEAR_CLIPBOARD_DELAY
	}
	return copyToClipboard(value.Inner, clipboardDelay)
}

func (t *entryTable) deleteFocused() tea.Cmd {
	focusedKey := t.fieldKeys[t.model.Cursor()]
	if isDefaultEntryField(focusedKey) {
		return nil
	}

	t.entry.DeleteField(focusedKey)

	if t.model.Cursor() >= len(t.entry.Strings) {
		t.model.SetCursor(len(t.entry.Strings) - 1)
	}

	return func() tea.Msg { return updateEntryMsg{t.entry} }
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
