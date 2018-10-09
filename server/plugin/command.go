package plugin

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/utils"
)

const (
	// Parameter: SiteURL, PluginId
	responseIconURL  = "%s/plugins/%s/logo_dark.png"
	responseUsername = "Matterpoll"

	// Parameter: Trigger
	commandHelpTextFormat = "To create a poll with the answer options \"Yes\" and \"No\" type `/%[1]s \"Question\"`.\n" +
		"You can customise the options by typing `/%[1]s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`\n" +
		"Poll Settings provider further customisation, e.g. `/%[1]s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --progress --anonymous`. The available Poll Settings are:\n" +
		"- `--anonymous`: Don't show who voted for what at the end\n" +
		"- `--progress`: During the poll, show how many votes each answer option got\n"

	// Parameter: Trigger
	commandInputErrorFormat = "Invalid input. Try `/%[1]s \"Question\"` or `/%[1]s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`"
	commandGenericError     = "Something went bad. Please try again later."
)

// ExecuteCommand parses a given input and creates a poll if the input is correct
func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	creatorID := args.UserId
	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	configuration := p.getConfiguration()

	q, o, s := utils.ParseInput(args.Command, configuration.Trigger)
	if q == "" || q == "help" {
		msg := fmt.Sprintf(commandHelpTextFormat, configuration.Trigger)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, msg, siteURL, nil), nil
	}
	if len(o) == 1 {
		msg := fmt.Sprintf(commandInputErrorFormat, configuration.Trigger)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, msg, siteURL, nil), nil
	}

	pollID := model.NewId()
	var poll *Poll
	var err error
	if len(o) == 0 {
		poll, err = NewPoll(creatorID, q, []string{"Yes", "No"}, s)
	} else {
		poll, err = NewPoll(creatorID, q, o, s)
	}
	if err != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Invalid input: "+err.Error(), siteURL, nil), nil
	}

	appErr := p.API.KVSet(pollID, poll.Encode())
	if appErr != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandGenericError, siteURL, nil), appErr
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(creatorID)
	if appErr != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandGenericError, siteURL, nil), appErr
	}

	response := poll.ToCommandResponse(*p.ServerConfig.ServiceSettings.SiteURL, pollID, displayName)
	p.API.LogDebug("Created a new poll", "response", response.ToJson())
	return response, nil
}

func getCommandResponse(responseType, text, siteURL string, attachments []*model.SlackAttachment) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     responseUsername,
		IconURL:      fmt.Sprintf(responseIconURL, siteURL, PluginId),
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
