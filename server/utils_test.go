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
		ExpectedSettings []string
	}{
		"Normal test": {
			Input:            `/poll "A" "B" "C"`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{},
		},
		"With quotationmark in question": {
			Input:            `/poll "A\"AA" "BBB" "CCC"`,
			Trigger:          "poll",
			ExpectedQuestion: `A"AA`,
			ExpectedOptions:  []string{"BBB", "CCC"},
			ExpectedSettings: []string{},
		},
		"With quotationmark in option": {
			Input:            `/poll "AAA" "\"BBB" "CCC"`,
			Trigger:          "poll",
			ExpectedQuestion: `AAA`,
			ExpectedOptions:  []string{`"BBB`, `CCC`},
			ExpectedSettings: []string{},
		},
		"Trim whitespace": {
			Input:            `/poll  "A" "B" "C"  `,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{},
		},
		"No options": {
			Input:            `/poll  `,
			Trigger:          "poll",
			ExpectedQuestion: "",
			ExpectedOptions:  []string{},
			ExpectedSettings: []string{},
		},
		"With one setting": {
			Input:            `/poll "A" "B" "C" --secret`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"secret"},
		},
		"With two settings": {
			Input:            `/poll "A" "B" "C" --secret --abc`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"secret", "abc"},
		},
		"With two settings, multipile whitespaces": {
			Input:            `/poll "A" "B" "C"    --secret   --abc   `,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"secret", "abc"},
		},
		"With two settings, no whitespaces": {
			Input:            `/poll "A" "B" "C"--secret--abc`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"secret", "abc"},
		},
		"With two settings, dashes in question": {
			Input:            `/poll "--A" "B" "C"--secret--abc`,
			Trigger:          "poll",
			ExpectedQuestion: "--A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"secret", "abc"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			q, o, s := ParseInput(test.Input, test.Trigger)

			assert.Equal(test.ExpectedQuestion, q)
			assert.Equal(test.ExpectedOptions, o)
			assert.Equal(test.ExpectedSettings, s)
		})
	}
}
