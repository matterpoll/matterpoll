package plugin

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"

	root "github.com/matterpoll/matterpoll"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils"
)

const (
	// Parameter: SiteURL, manifest.Id
	responseIconURL = "%s/plugins/%s/logo_dark-bg.png"
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
	commandHelpTextPollSettingAnonymousCreator = &i18n.Message{
		ID:    "command.help.text.pollSetting.anonymous-creator",
		Other: "Don't show author of the poll",
	}
	commandHelpTextPollSettingProgress = &i18n.Message{
		ID:    "command.help.text.pollSetting.progress",
		Other: "During the poll, show how many votes each answer option got",
	}
	commandHelpTextPollSettingPublicAddOption = &i18n.Message{
		ID:    "command.help.text.pollSetting.public-add-option",
		Other: "Allow all users to add additional options",
	}
	commandHelpTextPollSettingMultiVote = &i18n.Message{
		ID:    "command.help.text.pollSetting.multi-vote",
		Other: "Allow users to vote for X options. Default is 1. If X is 0, users have an unlimited amount of votes.",
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
func (p *MatterpollPlugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	msg, appErr := p.executeCommand(args)
	if msg != "" {
		p.SendEphemeralPost(args.ChannelId, args.UserId, args.RootId, msg)
	}
	return &model.CommandResponse{}, appErr
}

func (p *MatterpollPlugin) executeCommand(args *model.CommandArgs) (string, *model.AppError) {
	creatorID := args.UserId
	configuration := p.getConfiguration()

	userLocalizer := p.bundle.GetUserLocalizer(creatorID)
	publicLocalizer := p.bundle.GetServerLocalizer()

	defaultYes := p.bundle.LocalizeDefaultMessage(publicLocalizer, commandDefaultYes)
	defaultNo := p.bundle.LocalizeDefaultMessage(publicLocalizer, commandDefaultNo)

	q, o, s := utils.ParseInput(args.Command, configuration.Trigger)
	if q == "" {
		siteURL := *p.ServerConfig.ServiceSettings.SiteURL
		dialog := model.OpenDialogRequest{
			TriggerId: args.TriggerId,
			URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/create", root.Manifest.Id),
			Dialog:    p.getCreatePollDialog(siteURL, args.RootId, userLocalizer, configuration),
		}

		if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
			p.API.LogWarn("failed to open create poll dialog", "err", appErr.Error())
			return p.bundle.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
		}
		return "", nil
	}

	if q == "help" {
		msg := p.bundle.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: commandHelpTextSimple,
			TemplateData:   map[string]interface{}{"Trigger": configuration.Trigger, "Yes": defaultYes, "No": defaultNo},
		}) + "\n"
		msg += p.bundle.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: commandHelpTextOptions,
			TemplateData:   map[string]interface{}{"Trigger": configuration.Trigger},
		}) + "\n"
		msg += p.bundle.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: commandHelpTextPollSettingIntroduction,
			TemplateData:   map[string]interface{}{"Trigger": configuration.Trigger},
		}) + "\n"
		msg += "- `--anonymous`: " + p.bundle.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingAnonymous) + "\n"
		msg += "- `--anonymous-creator`: " + p.bundle.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingAnonymousCreator) + "\n"
		msg += "- `--progress`: " + p.bundle.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingProgress) + "\n"
		msg += "- `--public-add-option`: " + p.bundle.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingPublicAddOption) + "\n"
		msg += "- `--votes=X`: " + p.bundle.LocalizeDefaultMessage(userLocalizer, commandHelpTextPollSettingMultiVote)

		return msg, nil
	}

	if len(o) == 1 {
		return "", &model.AppError{
			Id:         p.bundle.LocalizeDefaultMessage(userLocalizer, commandErrorinvalidNumberOfOptions),
			StatusCode: http.StatusBadRequest,
			Where:      "ExecuteCommand",
		}
	}

	settings, errMsg := poll.NewSettingsFromStrings(s)
	if errMsg != nil {
		appErr := &model.AppError{
			Id: p.bundle.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
				DefaultMessage: commandErrorInvalidInput,
				TemplateData: map[string]interface{}{
					"Error": p.bundle.LocalizeErrorMessage(userLocalizer, errMsg),
				}}),
			StatusCode: http.StatusBadRequest,
			Where:      "ExecuteCommand",
		}
		return "", appErr
	}

	var newPoll *poll.Poll
	if len(o) == 0 {
		newPoll, errMsg = poll.NewPoll(creatorID, q, []string{defaultYes, defaultNo}, settings)
	} else {
		newPoll, errMsg = poll.NewPoll(creatorID, q, o, settings)
	}
	if errMsg != nil {
		appErr := &model.AppError{
			Id: p.bundle.LocalizeWithConfig(userLocalizer, &i18n.LocalizeConfig{
				DefaultMessage: commandErrorInvalidInput,
				TemplateData: map[string]interface{}{
					"Error": p.bundle.LocalizeErrorMessage(userLocalizer, errMsg),
				}}),
			StatusCode: http.StatusBadRequest,
			Where:      "ExecuteCommand",
		}
		return "", appErr
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(creatorID)
	if appErr != nil {
		p.API.LogWarn("failed to ConvertCreatorIDToDisplayName", "error", appErr.Error())
		return p.bundle.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
	}

	actions := newPoll.ToPostActions(p.bundle, root.Manifest.Id, displayName)
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
	if newPoll.Settings.Progress {
		post.AddProp("card", newPoll.ToCard(p.bundle, p.ConvertUserIDToDisplayName))
	}

	rPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		p.API.LogWarn("failed to post poll post", "error", appErr.Error())
		return p.bundle.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
	}

	newPoll.PostID = rPost.Id

	if err := p.Store.Poll().Insert(newPoll); err != nil {
		p.API.LogWarn("failed to save poll", "error", err.Error())
		return p.bundle.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric), nil
	}

	rPostJSON, _ := rPost.ToJSON()
	p.API.LogDebug("Created a new poll", "post", rPostJSON)

	return "", nil
}

func (p *MatterpollPlugin) getCommand(trigger string) (*model.Command, error) {
	iconData, err := p.getIconData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	localizer := p.bundle.GetServerLocalizer()

	return &model.Command{
		Trigger:              trigger,
		AutoComplete:         true,
		AutoCompleteDesc:     p.bundle.LocalizeDefaultMessage(localizer, commandAutoCompleteDesc),
		AutoCompleteHint:     p.bundle.LocalizeDefaultMessage(localizer, commandAutoCompleteHint),
		AutocompleteIconData: iconData,
	}, nil
}

func (p *MatterpollPlugin) getCreatePollDialog(siteURL, rootID string, l *i18n.Localizer, c *configuration) model.Dialog {
	elements := []model.DialogElement{{
		DisplayName: p.bundle.LocalizeDefaultMessage(l, &i18n.Message{
			ID:    "dialog.createPoll.question",
			Other: "Question",
		}),
		Name:    questionKey,
		Type:    "text",
		SubType: "text",
	}}
	for i := 1; i < 4; i++ {
		elements = append(elements, model.DialogElement{
			DisplayName: p.bundle.LocalizeWithConfig(l, &i18n.LocalizeConfig{
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
		DisplayName: "Number of Votes",
		Name:        "setting-multi",
		Type:        "text",
		SubType:     "number",
		Default:     "1",
		HelpText: p.bundle.LocalizeWithConfig(l, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "dialog.createPoll.setting.multi",
				Other: "The number of options that a user can vote on. 0 means that users can vote for all options even after adding options.",
			}}),
		Optional: false,
	})
	elements = append(elements, model.DialogElement{
		DisplayName: "Anonymous",
		Name:        "setting-anonymous",
		Type:        "bool",
		Placeholder: p.bundle.LocalizeDefaultMessage(l, commandHelpTextPollSettingAnonymous),
		Default:     fmt.Sprintf("%t", c.DefaultSettings["anonymous"]),
		Optional:    true,
	})
	elements = append(elements, model.DialogElement{
		DisplayName: "Anonymous creator",
		Name:        "setting-anonymous-creator",
		Type:        "bool",
		Placeholder: p.bundle.LocalizeDefaultMessage(l, commandHelpTextPollSettingAnonymousCreator),
		Default:     fmt.Sprintf("%t", c.DefaultSettings["anonymousCreator"]),
		Optional:    true,
	})
	elements = append(elements, model.DialogElement{
		DisplayName: "Progress",
		Name:        "setting-progress",
		Type:        "bool",
		Placeholder: p.bundle.LocalizeDefaultMessage(l, commandHelpTextPollSettingProgress),
		Default:     fmt.Sprintf("%t", c.DefaultSettings["progress"]),
		Optional:    true,
	})
	elements = append(elements, model.DialogElement{
		DisplayName: "Public Add Option",
		Name:        "setting-public-add-option",
		Type:        "bool",
		Placeholder: p.bundle.LocalizeDefaultMessage(l, commandHelpTextPollSettingPublicAddOption),
		Default:     fmt.Sprintf("%t", c.DefaultSettings["publicAddOption"]),
		Optional:    true,
	})
	dialog := model.Dialog{
		CallbackId: rootID,
		Title: p.bundle.LocalizeDefaultMessage(l, &i18n.Message{
			ID:    "dialog.create.title",
			Other: "Create Poll",
		}),
		IconURL: fmt.Sprintf(responseIconURL, siteURL, root.Manifest.Id),
		SubmitLabel: p.bundle.LocalizeDefaultMessage(l, &i18n.Message{
			ID:    "dialog.create.submitLabel",
			Other: "Create",
		}),
		Elements: elements,
	}

	return dialog
}
