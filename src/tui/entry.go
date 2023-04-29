package tui

import (
	"log"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/database"
	"github.com/Zaphoood/tresor/src/keepass/parser"
	"github.com/Zaphoood/tresor/src/keepass/undo"
	"github.com/Zaphoood/tresor/src/util/set"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

const ENCRYPTED_PLACEH = "••••••"

var defaultEntryFields []entryField = []entryField{
	{"Title", "Title", NO_TITLE_PLACEHOLDER},
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
	visited := set.New[string]()

	var value string
	for _, field := range defaultEntryFields {
		r, err := entry.Get(field.key)
		if err != nil {
			value = field.defaultValue
		} else if r.Protected {
			if r.Inner == "" {
				value = ""
			} else {
				value = ENCRYPTED_PLACEH
			}
		} else {
			value = r.Inner
		}
		rows = append(rows, table.Row{field.displayName, value})
		t.fieldKeys = append(t.fieldKeys, field.key)
		visited.Insert(field.key)
	}
	for _, field := range entry.Strings {
		if visited.Contains(field.Key) {
			continue
		}
		if field.Value.Protected {
			if field.Value.Inner == "" {
				value = ""
			} else {
				value = ENCRYPTED_PLACEH
			}
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
	newEntry := t.entry
	if isDefaultEntryField(focusedKey) {
		changed := newEntry.UpdateField(focusedKey, "")
		if !changed {
			return nil
		}
	} else {
		changed := newEntry.DeleteField(focusedKey)
		if !changed {
			log.Printf("ERROR: Tried to delete field '%s' from entry '%s' but no change made\n", focusedKey, t.entry.UUID)
		}
	}

	if t.model.Cursor() >= len(newEntry.Strings) {
		t.model.SetCursor(len(newEntry.Strings) - 1)
	}

	return func() tea.Msg {
		return undoableActionMsg{undo.NewUpdateEntryAction(newEntry, t.entry)}
	}
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
