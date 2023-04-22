package tui

import (
	"fmt"

	"github.com/Zaphoood/tresor/src/keepass/parser"
	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
)

const CLEAR_CLIPBOARD_DELAY = 10

func copyEntryFieldToClipboard(entry parser.Entry, field string, clearClipboardDelay int) (tea.Cmd, error) {
	value, err := entry.Get(field)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Cannot copy to clipboard. Failed to get field '%s' for Entry '%s'\n", field, entry.UUID)
	}
	notifyChangeChan := clipboard.Write(clipboard.FmtText, []byte(value.Inner))
	if clearClipboardDelay > 0 {
		return scheduleClearClipboard(CLEAR_CLIPBOARD_DELAY, notifyChangeChan), nil
	}
	return nil, nil
}
