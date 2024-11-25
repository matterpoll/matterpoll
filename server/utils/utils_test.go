package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/matterpoll/matterpoll/server/utils"
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
		"Replace curlyquotes": {
			Input:            `/poll “A“ ”BBB” “CCC”`,
			Trigger:          "poll",
			ExpectedQuestion: `A`,
			ExpectedOptions:  []string{"BBB", "CCC"},
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
			Input:            `/poll "A" "B" "C" --anonymous`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous"},
		},
		"With two settings": {
			Input:            `/poll "A" "B" "C" --anonymous --votes=2`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous", "votes=2"},
		},
		"With two settings, multiple whitespaces": {
			Input:            `/poll "A" "B" "C"    --anonymous   --abc   `,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous", "abc"},
		},
		"With two settings, no whitespaces": {
			Input:            `/poll "A" "B" "C"--anonymous--abc`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous", "abc"},
		},
		"With two settings, dashes in question": {
			Input:            `/poll "--A" "B" "C"--anonymous--abc`,
			Trigger:          "poll",
			ExpectedQuestion: "--A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous", "abc"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			q, o, s := utils.ParseInput(test.Input, test.Trigger)

			assert.Equal(test.ExpectedQuestion, q)
			assert.Equal(test.ExpectedOptions, o)
			assert.Equal(test.ExpectedSettings, s)
		})
	}
}
