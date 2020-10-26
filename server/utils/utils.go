package utils

import (
	"fmt"
	"strings"
)

// ParseInput pares a given input and tries to extract the poll question and poll options
func ParseInput(input, trigger string) (string, []string, []string) {
	settings := []string{}

	// Transform curly quotes to straight quotes
	input = strings.Map(func(in rune) rune {
		switch in {
		case '“', '”':
			return '"'
		}

		return in
	}, input)

	// Remove Trigger prefix and spaces
	in := strings.TrimSpace(strings.TrimPrefix(input, fmt.Sprintf("/%s", trigger)))
	// Remove first "
	in = strings.TrimLeft(in, `"`)

	// Split between options
	split := strings.Split(in, `" "`)
	lastIndex := len(split) - 1

	// Everything behind the last " are  Settings
	l := strings.Split(split[lastIndex], string('"'))
	split[lastIndex] = l[0]
	if len(l) == 2 && l[1] != "" {
		ops := strings.TrimPrefix(strings.TrimSpace(l[1]), "--")
		// Split between Settings
		opsList := strings.Split(ops, "--")
		for i := 0; i < len(opsList); i++ {
			s := strings.TrimSpace(opsList[i])
			settings = append(settings, s)
		}
	}

	// Unescape " in question and options
	question := strings.ReplaceAll(split[0], `\"`, `"`)
	options := split[1:]
	for i := 0; i < len(options); i++ {
		options[i] = strings.ReplaceAll(options[i], `\"`, `"`)
	}
	return question, options, settings
}
