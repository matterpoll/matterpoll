package main

import (
	"fmt"
	"strings"
)

func ParseInput(input string, trigger string) (string, []string, []string) {
	/*
			reg := regexp.MustCompile(fmt.Sprintf(`/%s(?:\s*"([0-9a-zA-Z]+)"\s*)*`, trigger))
			matches := reg.FindAllStringSubmatch(input, -1000)

		  return "", []string{}, []string{}

			reg := regexp.MustCompile(`\s*--([[:alpha:]]+)\s*`)

			matches := reg.FindAllStringSubmatch(input, -1)
			for i := len(matches) - 1; i >= 0; i-- {
				in = strings.TrimRight(in, matches[i][0])
				setting = append(setting, matches[i][1])
			}
	*/
	setting := []string{}
	in := input

	prossedInput := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(in, fmt.Sprintf("/%s", trigger))), "\""), "\"")
	if prossedInput == "" {
		return "", []string{}, []string{}
	}

	split := strings.Split(prossedInput, "\" \"")
	q := split[0]
	o := split[1:]

	return q, o, setting
}
