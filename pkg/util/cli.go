package util

import (
	"errors"
	"fmt"
)

const USAGE = "Usage: %s [FILE]"

// If there is exactly one command line argument, use it as the file path
func ParseCommandLineArgs(args []string) (string, error) {
	if len(args) < 2 {
		return "", nil
	} else if len(args) == 2 {
		return args[1], nil
	}
	return "", errors.New(fmt.Sprintf(USAGE, args[0]))
}
