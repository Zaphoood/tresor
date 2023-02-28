package tui

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Zaphoood/tresor/src/keepass/crypto"
	"github.com/Zaphoood/tresor/src/keepass/database"
	tea "github.com/charmbracelet/bubbletea"
)

func fileSelectedCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if len(path) == 0 {
			return loadFailedMsg{errors.New("Empty path")}
		}
		// Expand file path
		pathExpanded, err := expand(path)
		if err != nil {
			return err
		}
		if fileInfo, err := os.Stat(pathExpanded); err != nil {
			return loadFailedMsg{fmt.Errorf("File '%s' does not exist", path)}
		} else if fileInfo.IsDir() {
			return loadFailedMsg{fmt.Errorf("'%s' is directory", path)}
		}
		db := database.New(pathExpanded)
		err = db.Load()
		if err != nil {
			return loadFailedMsg{err}
		}
		return loadDoneMsg{db}
	}
}

func decryptFileCmd(database *database.Database, password string) tea.Cmd {
	return func() tea.Msg {
		database.SetPassword(password)
		err := database.Decrypt()
		if err != nil {
			if _, ok := err.(crypto.DecryptError); ok {
				return decryptFailedMsg{errors.New("Incorrect password")}
			}
		}
		err = database.Parse()
		if err != nil {
			return decryptFailedMsg{err}
		}
		valid, err := database.VerifyHeaderHash()
		if err != nil {
			log.Println("Could not verify header hash")
		}
		if !valid {
			return decryptFailedMsg{errors.New("Invalid header hash")}
		}
		return decryptDoneMsg{database}
	}
}

type loadDoneMsg struct {
	database *database.Database
}

type loadFailedMsg struct {
	err error
}

type decryptDoneMsg struct {
	database *database.Database
}

type decryptFailedMsg struct {
	err error
}

func scheduleClearClipboard(delay int, notify <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-notify:
			return nil
		case <-time.After(time.Duration(delay) * time.Second):
			return clearClipboardMsg{}
		}
	}
}

type clearClipboardMsg struct{}

/* When any model receives a tea.WindowSizeMsg, it should emit this command
in order to alert the main model of the resize. The main model will store the new
window size and pass it to other models upon initialization */
func globalResizeCmd(width, height int) tea.Cmd {
	return func() tea.Msg {
		return globalResizeMsg{width, height}
	}
}

type globalResizeMsg struct {
	width  int
	height int
}
