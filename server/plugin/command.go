package plugin

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils"
)

const (
	// Parameter: SiteURL, manifest.ID
	responseIconURL = "%s/plugins/%s/logo_dark.png"
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
		Other: "To create a poll with the answer options \"{{.Yes}}\" and \"{{.No}}\" type `/{{.Trigger}} \"Question\"`",
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
		Other: "Don't show who voted for what when the poll ends",
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
	msg, appErr := p.executeCommand(args)
	if msg != "" {
		p.SendEphemeralPost(args.ChannelId, args.UserId, msg)
	}
	return &model.CommandResponse{}, appErr
}

func (p *MatterpollPlugin) executeCommand(args *model.CommandArgs) (string, *model.AppError) {
	creatorID := args.UserId
	configuration := p.getConfiguration()

	userLocalizer := p.getUserLocalizer(creatorID)
	publicLocalizer := p.getServerLocalizer()

	defaultYes := p.LocalizeDefaultMessage(publicLocalizer, commandDefaultYes)
	defaultNo := p.LocalizeDefaultMessage(publicLocalizer, commandDefaultNo)

	q, o, s := utils.ParseInput(args.Command, configuration.Trigger)
	if q == "" {
		siteURL := *p.ServerConfig.ServiceSettings.SiteURL
		dialog := model.OpenDialogRequest{
			TriggerId: args.TriggerId,
			URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/create", manifest.ID),
			Dialog:    p.getCreatePollDialog(siteURL, args.RootId, userLocalizer),
		}

		if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
			p.API.LogWarn("failed to open create poll dialog", "err", appErr.Error())
			return p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
		}
		return "", nil
	}

	if q == "help" {
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

		return msg, nil
	}

	if len(o) == 1 {
		return "", &model.AppError{
			Id:         p.LocalizeDefaultMessage(userLocalizer, commandErrorinvalidNumberOfOptions),
			StatusCode: http.StatusBadRequest,
			Where:      "ExecuteCommand",
		}
	}

	var newPoll *poll.Poll
	var errMsg *poll.ErrorMessage
	if len(o) == 0 {
		newPoll, errMsg = poll.NewPoll(creatorID, q, []string{defaultYes, defaultNo}, s)
	} else {
		newPoll, errMsg = poll.NewPoll(creatorID, q, o, s)
	}
	if errMsg != nil {
		appErr := &model.AppError{
			Id: p.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
				DefaultMessage: commandErrorInvalidInput,
				TemplateData: map[string]interface{}{
					"Error": p.LocalizeErrorMessage(userLocalizer, errMsg),
				}}),
			StatusCode: http.StatusBadRequest,
			Where:      "ExecuteCommand",
		}
		return "", appErr
	}

	if err := p.Store.Poll().Insert(newPoll); err != nil {
		p.API.LogWarn("failed to save poll", "error", err.Error())
		return p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(creatorID)
	if appErr != nil {
		p.API.LogWarn("failed to ConvertCreatorIDToDisplayName", "error", appErr.Error())
		return p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
	}

	actions := newPoll.ToPostActions(publicLocalizer, manifest.ID, displayName)
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Type:      MatterpollPostType,
		Props: map[string]interface{}{
			"poll_id": newPoll.ID,
		},
	}
	model.ParseSlackAttachment(post, actions)

	if _, appErr = p.API.CreatePost(post); appErr != nil {
		p.API.LogWarn("failed to post poll post", "error", appErr.Error())
		return p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
	}

	p.API.LogDebug("Created a new poll", "post", post.ToJson())
	return "", nil
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

func (p *MatterpollPlugin) getCreatePollDialog(siteURL, rootID string, l *i18n.Localizer) model.Dialog {
	elements := []model.DialogElement{{
		DisplayName: p.LocalizeDefaultMessage(l, &i18n.Message{
			ID:    "dialog.createPoll.question",
			Other: "Question",
		}),
		Name:    questionKey,
		Type:    "text",
		SubType: "text",
	}}
	for i := 1; i < 4; i++ {
		elements = append(elements, model.DialogElement{
			DisplayName: p.LocalizeWithConfig(l, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "dialog.createPoll.option",
					Other: "Option {{ .Number }}",
				},
				TemplateData: map[string]interface{}{
					"Number": i,
				}}),
			Name:     fmt.Sprintf("option%v", i),
			Type:     "text",
			SubType:  "text",
			Optional: i > 2,
		})
	}
	elements = append(elements, model.DialogElement{
		DisplayName: "Anonymous",
		Name:        "setting-anonymous",
		Type:        "bool",
		Placeholder: p.LocalizeDefaultMessage(l, commandHelpTextPollSettingAnonymous),
		Optional:    true,
	})
	elements = append(elements, model.DialogElement{
		DisplayName: "Progress",
		Name:        "setting-progress",
		Type:        "bool",
		Placeholder: p.LocalizeDefaultMessage(l, commandHelpTextPollSettingProgress),
		Optional:    true,
	})
	elements = append(elements, model.DialogElement{
		DisplayName: "Public Add Option",
		Name:        "setting-public-add-option",
		Type:        "bool",
		Placeholder: p.LocalizeDefaultMessage(l, commandHelpTextPollSettingPublicAddOption),
		Optional:    true,
	})

	dialog := model.Dialog{
		CallbackId: rootID,
		Title: p.LocalizeDefaultMessage(l, &i18n.Message{
			ID:    "dialog.create.title",
			Other: "Create Poll",
		}),
		IconURL: fmt.Sprintf(responseIconURL, siteURL, manifest.ID),
		SubmitLabel: p.LocalizeDefaultMessage(l, &i18n.Message{
			ID:    "dialog.create.submitLabel",
			Other: "Create",
		}),
		Elements: elements,
	}

	return dialog
}
