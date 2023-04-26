package tui

import (
	"errors"
	"fmt"
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
	InputNone inputMode = iota
	InputCommand
	InputSearch
)

type CmdLineInputCallback func(string) tea.Cmd

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
		inputMode: InputNone,
		message:   DEFAULT_MESSAGE,
	}
}

func (c CommandLine) Init() tea.Cmd {
	return nil
}

func (c CommandLine) Update(msg tea.Msg) (CommandLine, tea.Cmd) {
	if !c.Focused() {
		return c, nil
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case setCommandLineMessageMsg:
		// TODO: Consider calling it the command line's 'status' instead in order to avoid these unfortunate variable names
		c.SetMessage(msg.msg)
		return c, nil
	case tea.KeyMsg:
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

func (c *CommandLine) StartInput(mode inputMode, prompt string) tea.Cmd {
	c.inputMode = mode
	c.input.SetValue("")
	c.input.Prompt = prompt
	return c.input.Focus()
}

func (c *CommandLine) endInput() {
	c.inputMode = InputNone
	c.input.Blur()
	c.message = DEFAULT_MESSAGE
}

func (c *CommandLine) onEnter() tea.Cmd {
	if c.inputMode == InputNone {
		return nil
	}
	inputMode := c.inputMode
	c.endInput()
	c.message = c.input.Prompt + c.input.Value()

	switch inputMode {
	case InputCommand:
		return commandCallback(c.input.Value())
	case InputSearch:
		// TODO: Remove hard-coded reverse=false
		return searchCallback(false)(c.input.Value())
	}
	return nil
}

func commandCallback(s string) tea.Cmd {
	cmdAsStrings, err := parseInputAsCommand(s)
	if err != nil || len(cmdAsStrings) == 0 {
		return nil
	}
	return func() tea.Msg { return commandInputMsg{cmdAsStrings} }
}

func searchCallback(reverse bool) CmdLineInputCallback {
	return func(s string) tea.Cmd {
		inputAsSearch, err := parseInputAsSearch(s)
		if err != nil || len(inputAsSearch) == 0 {
			return nil
		}
		return func() tea.Msg { return searchInputMsg{inputAsSearch, reverse} }
	}
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
	case InputNone:
		return c.message
	case InputCommand, InputSearch:
		return c.input.View()
	default:
		panic(fmt.Sprintf("ERROR: Invalid input mode %d", c.inputMode))
	}
}

func (c *CommandLine) SetMessage(msg string) {
	c.message = msg
}

func (c CommandLine) Focused() bool {
	return c.inputMode != InputNone
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
