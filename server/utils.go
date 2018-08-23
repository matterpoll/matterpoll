package main

import (
	"fmt"
	"strings"
)

func ParseInput(input string, trigger string) (string, []string) {
	o := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(input, fmt.Sprintf("/%s", trigger))), "\""), "\"")
	if o == "" {
		return "", []string{}
	}
	s := strings.Split(o, "\" \"")
	return s[0], s[1:]
}
