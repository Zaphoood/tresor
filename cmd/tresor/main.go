package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Zaphoood/tresor/pkg/tui"
	"github.com/Zaphoood/tresor/pkg/util"
)

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	path, err := util.ParseCommandLineArgs(os.Args)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(path)

	p := tea.NewProgram(tui.NewMainModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}
}
