package tui

import (
	"fmt"

	"github.com/Zaphoood/tresor/src/keepass/parser"
	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
)

const CLEAR_CLIPBOARD_DELAY = 10

func initClipboard() {
	err := clipboard.Init()
	if err != nil {
		// TODO: We should handle this more gracefully
		panic(err)
	}
}

func copyToClipboard(value string, clearClipboardDelay int) tea.Cmd {
	notifyChangeChan := clipboard.Write(clipboard.FmtText, []byte(value))

	commandLineMsg := "Copied to clipboard."
	var clearClipboardCmd tea.Cmd = nil
	if clearClipboardDelay > 0 {
		commandLineMsg += fmt.Sprintf(" (Clearing in %d seconds)", CLEAR_CLIPBOARD_DELAY)
		clearClipboardCmd = scheduleClearClipboard(CLEAR_CLIPBOARD_DELAY, notifyChangeChan)
	}
	setMsgCmd := func() tea.Msg {
		return setCommandLineMessageMsg{commandLineMsg}
	}
	return tea.Batch(setMsgCmd, clearClipboardCmd)
}

func copyEntryFieldToClipboard(entry parser.Entry, field string, clearClipboardDelay int) (tea.Cmd, error) {
	value, err := entry.Get(field)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Cannot copy to clipboard. Failed to get field '%s' for Entry '%s'\n", field, entry.UUID)
	}
	return copyToClipboard(value.Inner, clearClipboardDelay), nil
}

func clearClipboard() {
	clipboard.Write(clipboard.FmtText, []byte(""))
}
