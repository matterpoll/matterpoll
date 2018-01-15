package app

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
)

func TestCodeProviderDoCommand(t *testing.T) {
	cp := CodeProvider{}
	args := &model.CommandArgs{
		T: func(s string, args ...interface{}) string { return s },
	}

	for msg, expected := range map[string]string{
		"":           "api.command_code.message.app_error",
		"foo":        "    foo",
		"foo\nbar":   "    foo\n    bar",
		"foo\nbar\n": "    foo\n    bar\n    ",
	} {
		actual := cp.DoCommand(nil, args, msg).Text
		if actual != expected {
			t.Errorf("expected `%v`, got `%v`", expected, actual)
		}
	}
}
