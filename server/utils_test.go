package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInput(t *testing.T) {
	for name, test := range map[string]struct {
		Input            string
		ExpectedQuestion string
		ExpectedOptions  []string
	}{
		"Normal test": {
			Input:            "/matterpoll \"A\" \"B\" \"C\"",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
		},
		"Trim whitespace": {
			Input:            "/matterpoll   \"A\" \"B\" \"C\"",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
		},
		"No options": {
			Input:            "/matterpoll  ",
			ExpectedQuestion: "",
			ExpectedOptions:  []string{},
		},
	} {
		t.Run(name, func(t *testing.T) {

			q, o := ParseInput(test.Input)
			assert.Equal(t, test.ExpectedQuestion, q)
			assert.Equal(t, test.ExpectedOptions, o)
		})
	}
}
