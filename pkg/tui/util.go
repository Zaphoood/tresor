package tui

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var boxStyle = lipgloss.NewStyle().
	Width(50).
	Padding(1, 2, 1).
	BorderStyle(lipgloss.NormalBorder())

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
