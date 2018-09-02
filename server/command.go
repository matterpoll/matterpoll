package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	// Parameter: SiteURL, PluginId
	responseIconURL  = "%s/plugins/%s/logo_dark.png"
	responseUsername = "Matterpoll"

	// Parameter: Trigger
	commandHelpTextFormat = "To create a poll with the answer options \"Yes\" and \"No\" type `/%s \"Question\"`.\nYou can customise the options by typing `/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`"
	// Parameter: Trigger, Trigger
	commandInputErrorFormat = "Invalid Input. Try `/%s \"Question\"` or `/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`"
	commandGenericError     = "Something went bad. Please try again later."
)

func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	userID := args.UserId

	q, o := ParseInput(args.Command, p.Config.Trigger)
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

	appErr := p.API.KVSet(pollID, poll.Encode())
	if appErr != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandGenericError, nil), appErr
	}

	displayName, appErr := p.ConvertUserToDisplayName(userID)
	if appErr != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandGenericError, nil), appErr
	}
	response := poll.ToCommandResponse(*p.ServerConfig.ServiceSettings.SiteURL, pollID, displayName)
	p.API.LogDebug("Created a new poll", "response", response.ToJson())
	return response, nil
}

func getCommandResponse(responseType, text string, attachments []*model.SlackAttachment) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     responseUsername,
		IconURL:      fmt.Sprintf(responseIconURL, "http://localhost:8065", PluginId),
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
