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
		"Without any quote": {
			Input:            `/poll Is Matterpoll great?`,
			Trigger:          "poll",
			ExpectedQuestion: "Is Matterpoll great?",
			ExpectedOptions:  []string{},
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
		"With quotationmark in unquoted option": {
			Input:            `/poll "AAA" "BBB" \"CCC\"`,
			Trigger:          "poll",
			ExpectedQuestion: `AAA`,
			ExpectedOptions:  []string{`BBB`, `"CCC"`},
			ExpectedSettings: []string{},
		},
		"With escaped space": {
			Input:            `/poll Option\ Type "-h" /h \\h "\\?"`,
			Trigger:          "poll",
			ExpectedQuestion: `Option Type`,
			ExpectedOptions:  []string{`-h`, `/h`, `\h`, `\?`},
			ExpectedSettings: []string{},
		},
		"With whitespace characters in arguments": {
			Input:            `/poll "Poll Title" "Choice A" "Choice B"`,
			Trigger:          "poll",
			ExpectedQuestion: `Poll Title`,
			ExpectedOptions:  []string{`Choice A`, `Choice B`},
			ExpectedSettings: []string{},
		},
		"With quoted or escaped whitespace characters in arguments": {
			Input:            `/poll "Poll Title" Choice" "A Choice\ B`,
			Trigger:          "poll",
			ExpectedQuestion: `Poll Title`,
			ExpectedOptions:  []string{`Choice A`, `Choice B`},
			ExpectedSettings: []string{},
		},
		"With backlash": {
			Input:            `/poll "A" "B\\C" "D\\E"`,
			Trigger:          "poll",
			ExpectedQuestion: `A`,
			ExpectedOptions:  []string{`B\C`, `D\E`},
			ExpectedSettings: []string{},
		},
		"Trim whitespace": {
			Input:            `/poll  "A" "B" "C"  `,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{},
		},
		"Trim whitespace before options": {
			Input:            `/poll  "A"  "B" "C"  `,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{},
		},
		"Trim whitespace between options": {
			Input:            `/poll  "A" "B"  "C"  `,
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
		"No options, with trainling spaces": {
			Input:            `/poll  `,
			Trigger:          "poll",
			ExpectedQuestion: "",
			ExpectedOptions:  []string{},
			ExpectedSettings: []string{},
		},
		"No options": {
			Input:            `/poll`,
			Trigger:          "poll",
			ExpectedQuestion: "",
			ExpectedOptions:  []string{},
			ExpectedSettings: []string{},
		},
		"Weird quoting": {
			Input:            `/poll A" "? "B !" C" !"`,
			Trigger:          "poll",
			ExpectedQuestion: `A ?`,
			ExpectedOptions:  []string{"B !", "C !"},
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
			Input:            `/poll "A" "B" "C" --anonymous --abc`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous", "abc"},
		},
		"With two settings, multipile whitespaces": {
			Input:            `/poll "A" "B" "C"    --anonymous   --abc   `,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous", "abc"},
		},
		"With two settings, no whitespaces": {
			Input:            `/poll "A" "B" "C" --anonymous --abc`,
			Trigger:          "poll",
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous", "abc"},
		},
		"With two settings, dashes in question": {
			Input:            `/poll "--A" "B" "C" --anonymous-abc`,
			Trigger:          "poll",
			ExpectedQuestion: "--A",
			ExpectedOptions:  []string{"B", "C"},
			ExpectedSettings: []string{"anonymous-abc"},
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
