package tui

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const DEFAULT_MESSAGE = "Ready."
const PROMPT_COMMAND = ":"
const PROMPT_SEARCH = "/"
const PROMPT_REV_SEARCH = "?"

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
			case PROMPT_COMMAND:
				return c, c.startInput(inputCommand, PROMPT_COMMAND)
			case PROMPT_SEARCH:
				return c, c.startInput(inputSearch, PROMPT_SEARCH)
			case PROMPT_REV_SEARCH:
				return c, c.startInput(inputSearch, PROMPT_REV_SEARCH)
			}
			return c, nil
		}

		switch msg.String() {
		case "esc", "ctrl+c":
			c.endInput()
			return c, nil
		case "enter":
			return c, c.onEnter()
		case "backspace":
			if len(c.input.Value()) == 0 {
				c.endInput()
				return c, nil
			}
		}
		c.input, cmd = c.input.Update(msg)
		return c, cmd
	}
	return c, nil
}

func (c *CommandLine) startInput(mode inputMode, prompt string) tea.Cmd {
	c.inputMode = mode
	c.input.SetValue("")
	c.input.Prompt = prompt
	return c.input.Focus()
}

func (c *CommandLine) endInput() {
	log.Println("foo")
	c.inputMode = inputNone
	c.input.Blur()
	c.message = DEFAULT_MESSAGE
}

func (c *CommandLine) onEnter() tea.Cmd {
	if c.inputMode == inputNone {
		return nil
	}
	inputMode := c.inputMode
	c.endInput()
	c.message = c.input.Prompt + c.input.Value()

	switch inputMode {
	case inputCommand:
		return c.onCommandInput()
	case inputSearch:
		return c.onSearchInput()
	}
	return nil
}

func (c *CommandLine) onCommandInput() tea.Cmd {
	cmdAsStrings, err := parseInputAsCommand(c.input.Value())
	if err != nil || len(cmdAsStrings) == 0 {
		return nil
	}
	return func() tea.Msg { return commandInputMsg{cmdAsStrings} }
}

func (c *CommandLine) onSearchInput() tea.Cmd {
	var reverse bool
	switch c.input.Prompt {
	case PROMPT_SEARCH:
		reverse = false
	case PROMPT_REV_SEARCH:
		reverse = true
	default:
		panic(fmt.Sprintf("Invalid Prompt after search input: '%s'", c.input.Prompt))
	}
	inputAsSearch, err := parseInputAsSearch(c.input.Value())
	if err != nil || len(inputAsSearch) == 0 {
		return nil
	}
	return func() tea.Msg { return searchInputMsg{inputAsSearch, reverse} }
}

func parseInputAsCommand(input string) ([]string, error) {
	if len(input) == 0 {
		return nil, errors.New("Empty command")
	}
	split := strings.Split(input, " ")
	cmd := make([]string, 0)
	for _, s := range split {
		if len(s) > 0 {
			cmd = append(cmd, s)
		}
	}
	return cmd, nil
}

func parseInputAsSearch(input string) (string, error) {
	if len(input) == 0 {
		return "", errors.New("Empty search")
	}
	return input, nil
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
	query   string
	reverse bool
}
