package utils

import (
	"strings"
	"unicode"
)

// ParseInput pares a given input and tries to extract the poll question and poll options
func ParseInput(input, trigger string) (string, []string, []string) {
	// Transform curly quotes to straight quotes
	input = strings.Map(func(in rune) rune {
		switch in {
		case '“', '”':
			return '"'
		}

		return in
	}, input)

	// Remove Trigger prefix and spaces
	input = strings.TrimSpace(strings.TrimPrefix(input, "/"+trigger))

	// If there are no quotes, according to the documentation
	// the input is interpreted as a single question...
	if !strings.Contains(input, `"`) {
		return input, []string{}, []string{}
	}

	settings := []string{}
	var fields []string

	escaped := false
	quoted := false
	tainted := false // a tainted word cannot be a setting name
	var word string

	addField := func() {
		word = strings.TrimSpace(word)
		if len(word) == 0 {
			return
		}
		if !tainted && len(word) > 2 && strings.HasPrefix(word, "--") {
			settings = append(settings, word[2:])
		} else {
			fields = append(fields, word)
		}
		word = ""
		tainted = false
	}

	for _, c := range input {
		switch {
		case c == '"':
			if escaped {
				word += string(c)
			} else {
				if quoted {
					quoted = false
				} else {
					quoted = true
				}
			}
			tainted = true
		case c == '\\':
			if escaped {
				word += `\`
			}
			tainted = true
		case unicode.IsSpace(c) && !quoted && !escaped:
			addField() // End of word
		default:
			word += string(c)
		}
		escaped = c == '\\' && !escaped
	}
	if len(word) > 0 {
		addField() // Add last field
	}

	if len(fields) == 0 {
		return "", []string{}, settings // Question is missing
	}
	return fields[0], fields[1:], settings
}
