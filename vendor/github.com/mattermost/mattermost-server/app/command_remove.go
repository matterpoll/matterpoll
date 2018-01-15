// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"strings"

	l4g "github.com/alecthomas/log4go"
	goi18n "github.com/nicksnyder/go-i18n/i18n"

	"github.com/mattermost/mattermost-server/model"
)

type RemoveProvider struct {
}

type KickProvider struct {
}

const (
	CMD_REMOVE = "remove"
	CMD_KICK   = "kick"
)

func init() {
	RegisterCommandProvider(&RemoveProvider{})
	RegisterCommandProvider(&KickProvider{})
}

func (me *RemoveProvider) GetTrigger() string {
	return CMD_REMOVE
}

func (me *KickProvider) GetTrigger() string {
	return CMD_KICK
}

func (me *RemoveProvider) GetCommand(a *App, T goi18n.TranslateFunc) *model.Command {
	return &model.Command{
		Trigger:          CMD_REMOVE,
		AutoComplete:     true,
		AutoCompleteDesc: T("api.command_remove.desc"),
		AutoCompleteHint: T("api.command_remove.hint"),
		DisplayName:      T("api.command_remove.name"),
	}
}

func (me *KickProvider) GetCommand(a *App, T goi18n.TranslateFunc) *model.Command {
	return &model.Command{
		Trigger:          CMD_KICK,
		AutoComplete:     true,
		AutoCompleteDesc: T("api.command_remove.desc"),
		AutoCompleteHint: T("api.command_remove.hint"),
		DisplayName:      T("api.command_kick.name"),
	}
}

func (me *RemoveProvider) DoCommand(a *App, args *model.CommandArgs, message string) *model.CommandResponse {
	return doCommand(a, args, message)
}

func (me *KickProvider) DoCommand(a *App, args *model.CommandArgs, message string) *model.CommandResponse {
	return doCommand(a, args, message)
}

func doCommand(a *App, args *model.CommandArgs, message string) *model.CommandResponse {
	channel, err := a.GetChannel(args.ChannelId)
	if err != nil {
		return &model.CommandResponse{Text: args.T("api.command_channel_rename.channel.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	if channel.Type == model.CHANNEL_OPEN && !a.SessionHasPermissionToChannel(args.Session, args.ChannelId, model.PERMISSION_MANAGE_PUBLIC_CHANNEL_MEMBERS) {
		return &model.CommandResponse{Text: args.T("api.command_remove.permission.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	if channel.Type == model.CHANNEL_PRIVATE && !a.SessionHasPermissionToChannel(args.Session, args.ChannelId, model.PERMISSION_MANAGE_PRIVATE_CHANNEL_MEMBERS) {
		return &model.CommandResponse{Text: args.T("api.command_remove.permission.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	if channel.Type == model.CHANNEL_GROUP || channel.Type == model.CHANNEL_DIRECT {
		return &model.CommandResponse{Text: args.T("api.command_remove.direct_group.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	if len(message) == 0 {
		return &model.CommandResponse{Text: args.T("api.command_remove.message.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	targetUsername := ""

	targetUsername = strings.SplitN(message, " ", 2)[0]
	targetUsername = strings.TrimPrefix(targetUsername, "@")

	var userProfile *model.User
	if result := <-a.Srv.Store.User().GetByUsername(targetUsername); result.Err != nil {
		l4g.Error(result.Err.Error())
		return &model.CommandResponse{Text: args.T("api.command_remove.missing.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	} else {
		userProfile = result.Data.(*model.User)
	}

	_, err = a.GetChannelMember(args.ChannelId, userProfile.Id)
	if err != nil {
		nameFormat := *a.Config().TeamSettings.TeammateNameDisplay
		return &model.CommandResponse{Text: args.T("api.command_remove.user_not_in_channel", map[string]interface{}{"Username": userProfile.GetDisplayName(nameFormat)}), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	if err = a.RemoveUserFromChannel(userProfile.Id, args.UserId, channel); err != nil {
		return &model.CommandResponse{Text: args.T(err.Id, map[string]interface{}{"Channel": model.DEFAULT_CHANNEL}), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	return &model.CommandResponse{}
}
