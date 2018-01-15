// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-server/model"
	goi18n "github.com/nicksnyder/go-i18n/i18n"
)

type SettingsProvider struct {
}

const (
	CMD_SETTINGS = "settings"
)

func init() {
	RegisterCommandProvider(&SettingsProvider{})
}

func (settings *SettingsProvider) GetTrigger() string {
	return CMD_SETTINGS
}

func (settings *SettingsProvider) GetCommand(a *App, T goi18n.TranslateFunc) *model.Command {
	return &model.Command{
		Trigger:          CMD_SETTINGS,
		AutoComplete:     true,
		AutoCompleteDesc: T("api.command_settings.desc"),
		AutoCompleteHint: "",
		DisplayName:      T("api.command_settings.name"),
	}
}

func (settings *SettingsProvider) DoCommand(a *App, args *model.CommandArgs, message string) *model.CommandResponse {
	// This command is handled client-side and shouldn't hit the server.
	return &model.CommandResponse{
		Text:         args.T("api.command_settings.unsupported.app_error"),
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
	}
}
