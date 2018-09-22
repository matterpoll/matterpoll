package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInput(t *testing.T) {
	for name, test := range map[string]struct {
		Input            string
		Trigger          string
		ExpectedQuestion string
		ExpectedOptions  []string
	}{
		"Normal test": {
			Input:            "/poll \"A\" \"B\" \"C\"",
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
		},
		"Trim whitespace": {
			Input:            "/poll   \"A\" \"B\" \"C\"",
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
		},
		"No options": {
			Input:            "/poll  ",
			Trigger:          "poll",
			ExpectedQuestion: "",
			ExpectedOptions:  []string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			q, o := ParseInput(test.Input, test.Trigger)

			assert.Equal(test.ExpectedQuestion, q)
			assert.Equal(test.ExpectedOptions, o)
		})
	}
}
