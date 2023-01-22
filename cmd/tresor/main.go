package main

import (
	"fmt"
	"os"
	tea "github.com/charmbracelet/bubbletea"
    "github.com/Zaphoood/tresor/pkg/ui"
)


func main() {
    p := tea.NewProgram(ui.NewModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %s", err)
        os.Exit(1)
    }
}
