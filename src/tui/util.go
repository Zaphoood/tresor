package tui

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/parser"
	"github.com/Zaphoood/tresor/src/keepass/undo"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var boxStyle = lipgloss.NewStyle().
	Width(50).
	Padding(1, 2, 1).
	BorderStyle(lipgloss.NormalBorder())

// Modulo that works properly with negative numbers
func mod(a, b int) int {
	return ((a % b) + b) % b
}

func centerInWindow(text string, windowWidth, windowHeight int) string {
	return lipgloss.Place(windowWidth, windowHeight, lipgloss.Center, lipgloss.Center, text)
}

func expand(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		user, err := user.Current()
		if err != nil {
			return "", err
		}
		if len(path) == 1 {
			return user.HomeDir, nil
		} else if strings.HasPrefix(path, "~/") {
			return filepath.Join(user.HomeDir, path[2:]), nil
		} else {
			// We don't care about handling paths like '~user/...' for now
			return "", fmt.Errorf("Expanding of path '%s' is no supported", path)
		}
	}
	return path, nil
}

func completePath(path string) ([]string, error) {
	var head, tail string
	if len(path) == 0 {
		head = "."
		tail = ""
	} else if path == "~" {
		head = "~/"
		tail = ""
	} else {
		lastSlashIndex := -1
		for i := len(path) - 1; i > 0; i-- {
			if path[i] == filepath.Separator {
				lastSlashIndex = i
				break
			}
		}
		if lastSlashIndex == -1 {
			head = "."
			tail = path
		} else {
			head = path[:lastSlashIndex]
			tail = path[lastSlashIndex+1:]
		}
	}
	expanded, err := expand(head)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(expanded)
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, tail) {
			if entry.IsDir() {
				name += string(filepath.Separator)
			}
			results = append(results, name)
		}
	}
	return results, nil
}

// Call filepath.Join but keep a trailing path separator if present
func joinRetainTrailingSep(elem ...string) string {
	var hasTrailingSeparator bool
	if len(elem) == 0 {
		hasTrailingSeparator = false
	} else {
		hasTrailingSeparator = strings.HasSuffix(elem[len(elem)-1], string(filepath.Separator))
	}
	joined := filepath.Join(elem...)
	if hasTrailingSeparator {
		return joined + string(filepath.Separator)
	}
	return joined
}

//tableFocusCursor makes sure that cursor of a table.Model is visible
func tableFocusCursor(t *table.Model) {
	// This shitty workaround is necessary, since when the cursor is set using t.SetCursor(), it may go off screen
	t.MoveUp(0)
	t.MoveDown(0)
}

func makeChangeFieldAction(entry parser.Entry, field string, newValue string, returnValue interface{}) tea.Cmd {
	newEntry := entry
	newEntry.UpdateField(field, newValue)

	return func() tea.Msg {
		return undoableActionMsg{undo.NewUpdateEntryAction(
			newEntry,
			entry,
			focusChangedItemCmd(newEntry.UUID),
			fmt.Sprintf("Change value of '%s' to '%s'", field, newValue),
		)}
	}
}
