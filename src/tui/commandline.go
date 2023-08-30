package tui

import (
	"errors"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const INITIAL_MESSAGE = "Ready."
const PROMPT_COMMAND = ":"
const PROMPT_SEARCH = "/"
const PROMPT_REV_SEARCH = "?"

type CmdLineInputCallback func(string) tea.Cmd

type CommandLine struct {
	input    textinput.Model
	message  string
	callback CmdLineInputCallback
}

func NewCommandLine() CommandLine {
	input := textinput.New()
	input.Prompt = ""
	return CommandLine{
		input:    input,
		message:  INITIAL_MESSAGE,
		callback: nil,
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
			cmd = c.onEnter()
			return c, cmd
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

func (c *CommandLine) StartInput(prompt string, callback CmdLineInputCallback) tea.Cmd {
	c.input.SetValue("")
	c.callback = callback
	c.input.Prompt = prompt
	return c.input.Focus()
}

func (c *CommandLine) endInput() {
	c.input.Blur()
	c.message = ""
}

func (c *CommandLine) onEnter() tea.Cmd {
	if !c.input.Focused() {
		return nil
	}
	c.endInput()
	c.SetMessage(c.input.Prompt + c.input.Value())

	if c.callback == nil {
		log.Println("ERROR: In CommandLine.onEnter(): CommandLine.callback is nil")
		return nil
	}

	return c.callback(c.input.Value())
}

func parseInputAsSearch(input string) (string, error) {
	if len(input) == 0 {
		return "", errors.New("Empty search")
	}
	return input, nil
}

func (c CommandLine) View() string {
	if c.input.Focused() {
		return c.input.View()
	} else {
		return c.message
	}
}

func (c *CommandLine) SetMessage(msg string) {
	c.message = msg
}

func (c CommandLine) Focused() bool {
	return c.input.Focused()
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

func CommandCallback(s string) tea.Cmd {
	cmdAsStrings, err := parseInputAsCommand(s)
	if err != nil || len(cmdAsStrings) == 0 {
		return nil
	}
	return func() tea.Msg { return commandInputMsg{cmdAsStrings} }
}

func SearchCallback(reverse bool) CmdLineInputCallback {
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
