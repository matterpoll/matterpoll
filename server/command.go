package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	responseIconURL  = "https://raw.githubusercontent.com/matterpoll/matterpoll/master/assets/logo_dark.png"
	responseUsername = "Matterpoll"

	commandHelpTextFormat   = "To create a poll with the answer options \"Yes\" and \"No\" type `/%s \"Question\"`.\nYou can customise the options by typing `/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"` "
	commandInputErrorFormat = "Invalid Input. Try `/%s \"Question\"` or `/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`"
	commandGenericError     = "Something went bad. Please try again later."
)

func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	q, o := ParseInput(args.Command, p.Config.Trigger)
	userID := args.UserId
	if len(o) == 0 && q == "help" {
		msg := fmt.Sprintf(commandHelpTextFormat, p.Config.Trigger, p.Config.Trigger)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, msg, nil), nil
	}
	if len(o) == 1 || q == "" {
		msg := fmt.Sprintf(commandInputErrorFormat, p.Config.Trigger, p.Config.Trigger)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, msg, nil), nil
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
	response := poll.ToCommandResponse(args.SiteURL, user.GetFullName(), pollID)
	p.API.LogDebug("Created a new poll", "response", response.ToJson())
	return response, nil
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

func getCommand(trigger string) *model.Command {
	return &model.Command{
		Trigger:          trigger,
		DisplayName:      "Matterpoll",
		Description:      "Polling feature by https://github.com/matterpoll/matterpoll",
		AutoComplete:     true,
		AutoCompleteDesc: "Create a poll",
		AutoCompleteHint: "[Question] [Answer 1] [Answer 2]...",
	}
}
