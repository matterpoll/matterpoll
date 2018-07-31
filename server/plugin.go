package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
}

func (p *MatterpollPlugin) OnActivate() error {
	return p.API.RegisterCommand(&model.Command{
		Trigger:          `matterpoll`,
		AutoComplete:     true,
		AutoCompleteDesc: `Create a poll`,
		AutoCompleteHint: `[Question] [Answer 1] [Answer 2]...`,
	})
}

func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
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
