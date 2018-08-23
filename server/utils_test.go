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

			q, o := ParseInput(test.Input, test.Trigger)
			assert.Equal(t, test.ExpectedQuestion, q)
			assert.Equal(t, test.ExpectedOptions, o)
		})
	}
}
