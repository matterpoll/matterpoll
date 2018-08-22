package main

import "strings"

func ParseInput(input string) (string, []string) {
	o := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(input, "/matterpoll")), "\""), "\"")
	if o == "" {
		return "", []string{}
	}
	s := strings.Split(o, "\" \"")
	return s[0], s[1:]
}
