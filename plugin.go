package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/rpcplugin"
)

type MatterpollPlugin struct {
	api plugin.API
}

func (p *MatterpollPlugin) OnActivate(api plugin.API) error {
	p.api = api
	return p.api.RegisterCommand(&model.Command{
		Trigger:          `matterpoll`,
		AutoComplete:     true,
		AutoCompleteDesc: `Create a poll`,
		AutoCompleteHint: `[Question] [Answer 1] [Answer 2]...`,
	})
}

func (p *MatterpollPlugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	input := ParseInput(args.Command)
	if len(input) < 2 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Username:     `Matterpoll`,
			Text:         `We need input. Try ` + "`" + `/matterpoll "Question" "Answer 1" "Answer 2"` + "`",
		}, nil
	}
	attachList := []*model.PostAction{}
	for index := 1; index < len(input); index++ {
		attachList = append(attachList, &model.PostAction{
			Name: input[index],
		})
	}
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Username:     `Matterpoll`,
		Attachments: []*model.SlackAttachment{{
			AuthorName: `Matterpoll`,
			Text:       input[0],
			Actions:    attachList,
		},
		},
	}, nil
}

func ParseInput(input string) []string {
	o := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(input, "/matterpoll")), "\""), "\"")
	if o == "" {
		return []string{}
	}
	return strings.Split(o, "\" \"")
}

func main() {
	rpcplugin.Main(&MatterpollPlugin{})
}
