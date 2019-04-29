package plugin

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	// Parameter: SiteURL, PluginId
	responseIconURL  = "%s/plugins/%s/logo_dark.png"
	responseUsername = "Matterpoll"
)

var (
	commandAutoCompleteDesc = &i18n.Message{
		ID:    "command.autoComplete.desc",
		Other: "Create a poll",
	}
	commandAutoCompleteHint = &i18n.Message{
		ID:    "command.autoComplete.hint",
		Other: `"[Question]" "[Answer 1]" "[Answer 2]"...`,
	}

	commandDefaultYes = &i18n.Message{
		ID:    "command.default.yes",
		Other: "Yes",
	}
	commandDefaultNo = &i18n.Message{
		ID:    "command.default.no",
		Other: "No",
	}

	commandHelpTextSimple = &i18n.Message{
		ID:    "command.help.text.simple",
		Other: "To create a poll with the answer options \"{{.Yes}}\" and \"{{.No}}\" type `/{{.Trigger}} \"Question\"`.",
	}
	commandHelpTextOptions = &i18n.Message{
		ID:    "command.help.text.options",
		Other: "You can customize the options by typing `/{{.Trigger}} \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`",
	}
	commandHelpTextPollSettingIntroduction = &i18n.Message{
		ID:    "command.help.text.pollSetting.introduction",
		Other: "Poll Settings provider further customization, e.g. `/{{.Trigger}} \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --progress --anonymous`. The available Poll Settings are:",
	}
	commandHelpTextPollSettingAnonymous = &i18n.Message{
		ID:    "command.help.text.pollSetting.anonymous",
		Other: "Don't show who voted for what",
	}
	commandHelpTextPollSettingProgress = &i18n.Message{
		ID:    "command.help.text.pollSetting.progress",
		Other: "During the poll, show how many votes each answer option got",
	}
	commandHelpTextPollSettingPublicAddOption = &i18n.Message{
		ID:    "command.help.text.pollSetting.public-add-option",
		Other: "Allow all users to add additional options",
	}

	commandErrorGeneric = &i18n.Message{
		ID:    "command.error.generic",
		Other: "Something went wrong. Please try again later.",
	}
	commandErrorinvalidNumberOfOptions = &i18n.Message{
		ID:    "command.error.invalidNumberOfOptions",
		Other: "You must provide either no answer or at least two answers.",
	}
	commandErrorInvalidInput = &i18n.Message{
		ID:    "command.error.invalidInput",
		Other: "Invalid input: {{.Error}}",
	}
)

// ExecuteCommand parses a given input and creates a poll if the input is correct
func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	creatorID := args.UserId
	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	configuration := p.getConfiguration()

	userLocalizer := p.getUserLocalizer(creatorID)
	publicLocalizer := p.getServerLocalizer()

	defaultYes := p.LocalizeDefaultMessage(publicLocalizer, commandDefaultYes)
	defaultNo := p.LocalizeDefaultMessage(publicLocalizer, commandDefaultNo)

	q, o, s := utils.ParseInput(args.Command, configuration.Trigger)
	if q == "" || q == "help" {
		msg := p.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: commandHelpTextSimple,
			TemplateData:   map[string]interface{}{"Trigger": configuration.Trigger, "Yes": defaultYes, "No": defaultNo},
		}) + "\n"
		msg += p.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: commandHelpTextOptions,
			TemplateData:   map[string]interface{}{"Trigger": configuration.Trigger},
		}) + "\n"
		msg += p.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: commandHelpTextPollSettingIntroduction,
			TemplateData:   map[string]interface{}{"Trigger": configuration.Trigger},
		}) + "\n"
		msg += "- `--anonymous`: " + p.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingAnonymous) + "\n"
		msg += "- `--progress`: " + p.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingProgress) + "\n"
		msg += "- `--public-add-option`: " + p.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingPublicAddOption)

		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, msg, siteURL, nil), nil
	}
	if len(o) == 1 {
		return nil, &model.AppError{
			Id:         p.LocalizeDefaultMessage(userLocalizer, commandErrorinvalidNumberOfOptions),
			StatusCode: http.StatusBadRequest,
			Where:      "ExecuteCommand",
		}
	}

	var newPoll *poll.Poll
	var err error
	if len(o) == 0 {
		newPoll, err = poll.NewPoll(creatorID, q, []string{defaultYes, defaultNo}, s)
	} else {
		newPoll, err = poll.NewPoll(creatorID, q, o, s)
	}
	if err != nil {
		return nil, &model.AppError{
			Id: p.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
				DefaultMessage: commandErrorInvalidInput,
				TemplateData: map[string]interface{}{
					"Error": err.Error(),
				}}),
			StatusCode: http.StatusBadRequest,
			Where:      "ExecuteCommand",
		}
	}

	if err := p.Store.Poll().Save(newPoll); err != nil {
		p.API.LogError("failed to save poll", "err", err.Error())
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), siteURL, nil), nil
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(creatorID)
	if appErr != nil {
		p.API.LogError("failed to ConvertCreatorIDToDisplayName", "err", appErr.Error())
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), siteURL, nil), nil
	}

	actions := newPoll.ToPostActions(publicLocalizer, *p.ServerConfig.ServiceSettings.SiteURL, PluginId, displayName)
	response := getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "", *p.ServerConfig.ServiceSettings.SiteURL, actions)
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

func (p *MatterpollPlugin) getCommand(trigger string) *model.Command {
	localizer := p.getServerLocalizer()

	return &model.Command{
		Trigger:          trigger,
		AutoComplete:     true,
		AutoCompleteDesc: p.LocalizeDefaultMessage(localizer, commandAutoCompleteDesc),
		AutoCompleteHint: p.LocalizeDefaultMessage(localizer, commandAutoCompleteHint),
	}
}
