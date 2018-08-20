package main

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	trigger          = "matterpoll"
	responseIconURL  = "https://github.com/matterpoll/matterpoll/raw/rewrite/assets/logo_dark.png"
	responseUsername = "Matterpoll"

	commandHelpText     = "To create a poll with the answer options \"Yes\" and \"No\" type `/matterpoll \"Question\"`.\nYou can customise the options by typing `/matterpoll \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"` "
	commandInputError   = "Invalid Input. Try `/matterpoll \"Question\"` or `/matterpoll \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`"
	commandGenericError = "Something went bad. Please try again later."
)

func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	q, o := ParseInput(args.Command)
	userID := args.UserId
	if len(o) == 0 && q == "help" {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandHelpText, nil), nil

	}
	if len(o) == 1 || q == "" {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandInputError, nil), nil
	}

	pollID := p.idGen.NewID()
	var poll *Poll
	if len(o) == 0 {
		poll = NewPoll(userID, q, []string{"Yes", "No"})
	} else {
		poll = NewPoll(userID, q, o)
	}

	err := p.API.KVSet(pollID, poll.Encode())
	if err != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandGenericError, nil), err
	}
	user, err := p.API.GetUser(userID)
	if err != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandGenericError, nil), err
	}
	return poll.ToCommandResponse(args.SiteURL, user.GetFullName(), pollID), nil
}

func getCommandResponse(responseType, text string, attachments []*model.SlackAttachment) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     responseUsername,
		IconURL:      responseIconURL,
		Type:         model.POST_DEFAULT,
		Attachments:  attachments,
	}
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          trigger,
		DisplayName:      "Matterpoll",
		Description:      "Polling feature by https://github.com/matterpoll/matterpoll",
		AutoComplete:     true,
		AutoCompleteDesc: "Create a poll",
		AutoCompleteHint: "[Question] [Answer 1] [Answer 2]...",
	}
}
