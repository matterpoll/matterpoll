// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-server/model"
	goi18n "github.com/nicksnyder/go-i18n/i18n"
)

type MeProvider struct {
}

const (
	CMD_ME = "me"
)

func init() {
	RegisterCommandProvider(&MeProvider{})
}

func (me *MeProvider) GetTrigger() string {
	return CMD_ME
}

func (me *MeProvider) GetCommand(a *App, T goi18n.TranslateFunc) *model.Command {
	return &model.Command{
		Trigger:          CMD_ME,
		AutoComplete:     true,
		AutoCompleteDesc: T("api.command_me.desc"),
		AutoCompleteHint: T("api.command_me.hint"),
		DisplayName:      T("api.command_me.name"),
	}
}

func (me *MeProvider) DoCommand(a *App, args *model.CommandArgs, message string) *model.CommandResponse {
	return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, Text: "*" + message + "*"}
}
