package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const DEFAULT_MESSAGE = "Ready."

type inputMode int

const (
	inputNone inputMode = iota
	inputCommand
	inputSearch
)

type CommandLine struct {
	input     textinput.Model
	inputMode inputMode
	message   string
}

func NewCommandLine() CommandLine {
	input := textinput.New()
	input.Prompt = ""
	return CommandLine{
		input:     input,
		inputMode: inputNone,
		message:   DEFAULT_MESSAGE,
	}
}

func (c CommandLine) Init() tea.Cmd {
	return nil
}

func (c CommandLine) Update(msg tea.Msg) (CommandLine, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
		if c.inputMode == inputNone {
			switch msg.String() {
			case ":":
				c.inputMode = inputCommand
				c.input.Focus()
				c.resetPrompt()
			case "/":
				c.inputMode = inputSearch
				c.input.Focus()
				c.resetPrompt()
			}
			return c, nil
		}

		switch msg.String() {
		case "esc", "ctrl+c":
			c.endInputMode()
			return c, nil
		case "enter":
			return c, c.onEnter()
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
	}
	return c, nil
}

func (c *CommandLine) resetPrompt() {
	switch c.inputMode {
	case inputCommand:
		c.input.SetValue(":")
		c.input.SetCursor(1)
	case inputSearch:
		c.input.SetValue("/")
		c.input.SetCursor(1)
	}
}

func (c *CommandLine) onEnter() tea.Cmd {
	switch c.inputMode {
	case inputCommand:
		return c.onCommandInput()
	case inputSearch:
		return c.onSearchInput()
	}
	return nil
}

func (c *CommandLine) onCommandInput() tea.Cmd {
	c.endInputMode()
	c.message = c.input.Value()
	cmdAsStrings, err := parseInputAsCommand(c.input.Value())
	if err != nil {
		// Input could not be parsed as command
		// TODO: Consider displaying error message here
		return nil
	}
	return func() tea.Msg { return commandInputMsg{cmdAsStrings} }
}

func (c *CommandLine) onSearchInput() tea.Cmd {
	c.endInputMode()
	c.message = c.input.Value()
	inputAsSearch, err := parseInputAsSearch(c.input.Value())
	if err != nil {
		return nil
	}
	return func() tea.Msg { return searchInputMsg{inputAsSearch} }
}

func parseInputAsCommand(input string) ([]string, error) {
	if len(input) == 0 || input[0] != byte(':') {
		return nil, fmt.Errorf("ERROR: Commands must start with ':', got '%s'\n", input)
	}
	// Clean out empty strings
	split := strings.Split(input[1:], " ")
	cmd := make([]string, 0)
	for _, s := range split {
		if len(s) > 0 {
			cmd = append(cmd, s)
		}
	}
	return cmd, nil
}

func parseInputAsSearch(input string) (string, error) {
	if len(input) == 0 || input[0] != byte('/') {
		return "", fmt.Errorf("ERROR: Search must start with '/', got '%s'\n", input)
	}
	return input[1:], nil
}

func (c *CommandLine) endInputMode() {
	c.inputMode = inputNone
	c.input.Blur()
	c.message = DEFAULT_MESSAGE
}

func (c CommandLine) View() string {
	switch c.inputMode {
	case inputNone:
		return c.message
	case inputCommand, inputSearch:
		return c.input.View()
	default:
		panic(fmt.Sprintf("ERROR: Invalid input mode %d", c.inputMode))
	}
}

func (c *CommandLine) SetMessage(msg string) {
	c.message = msg
}

func (c CommandLine) IsInputActive() bool {
	return c.inputMode != inputNone
}

func (c CommandLine) GetHeight() int {
	return 1
}

type commandInputMsg struct {
	cmd []string
}

type searchInputMsg struct {
	query string
}
