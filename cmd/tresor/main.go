package main

import (
	"fmt"
	"os"

	"github.com/Zaphoood/tresor/lib/keepass/database"
	"github.com/Zaphoood/tresor/pkg/tui"
	"github.com/Zaphoood/tresor/pkg/util"
	tea "github.com/charmbracelet/bubbletea"
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

	var d *database.Database = nil
	if len(path) > 0 {
		d = database.New(path)
		err = d.Load()
		if err != nil {
			fmt.Printf("Error while opening %s: %s\n", path, err)
			return
		}
	}

	p := tea.NewProgram(tui.NewMainModel(d), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error while running TUI: %s", err)
		os.Exit(1)
	}
}
