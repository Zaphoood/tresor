package tui

import "github.com/charmbracelet/lipgloss"

var boxStyle = lipgloss.NewStyle().
	Width(50).
	Padding(1, 2, 1).
	BorderStyle(lipgloss.NormalBorder())

func centerInWindow(text string, windowWidth, windowHeight int) string {
	return lipgloss.Place(windowWidth, windowHeight, lipgloss.Center, lipgloss.Center, text)
}
