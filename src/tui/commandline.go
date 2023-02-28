package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const DEFAULT_MESSAGE = "Ready."

type CommandLine struct {
	input     textinput.Model
	inputMode bool
	message   string
	// callback is called after a command is entered. If the bool returned by the function
	// is true, then the string return value will be set as the new status message
	callback func(string) (tea.Cmd, bool, string)
}

func NewCommandLine(callback func(string) (tea.Cmd, bool, string)) CommandLine {
	input := textinput.New()
	input.Prompt = ""
	return CommandLine{
		input:     input,
		inputMode: false,
		message:   DEFAULT_MESSAGE,
		callback:  callback,
	}
}

func (c CommandLine) Init() tea.Cmd {
	return nil
}

func (c CommandLine) Update(msg tea.Msg) (CommandLine, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
		if c.inputMode {
			switch msg.String() {
			case "esc", "ctrl+c":
				c.endInputMode()
				return c, nil
			case "enter":
				return c, c.onCommandInput()
			}

			c.input, cmd = c.input.Update(msg)

			switch msg.String() {
			case "backspace":
				if len(c.input.Value()) == 0 {
					c.endInputMode()
					return c, nil
				}
			case "ctrl+w":
				if len(c.input.Value()) == 0 {
					c.resetPrompt()
					return c, nil
				}
			}

			return c, cmd
		} else {
			if msg.String() == ":" {
				c.inputMode = true
				c.input.Focus()
				c.resetPrompt()
				return c, nil
			}
		}
	}
	return c, nil
}

func (c *CommandLine) resetPrompt() {
	c.input.SetValue(":")
	c.input.SetCursor(1)
}

func (c *CommandLine) onCommandInput() tea.Cmd {
	c.endInputMode()
	cmd, updateMessage, message := c.callback(c.input.Value())
	if updateMessage {
		c.message = message
	}
	return cmd
}

func (c *CommandLine) endInputMode() {
	c.inputMode = false
	c.input.Blur()
	c.message = DEFAULT_MESSAGE
}

func (c CommandLine) View() string {
	if c.inputMode {
		return c.input.View()
	}
	return c.message
}

func (c *CommandLine) SetMessage(msg string) {
	c.message = msg
}

func (c CommandLine) IsInputMode() bool {
	return c.inputMode
}

func (c CommandLine) GetHeight() int {
	return 1
}