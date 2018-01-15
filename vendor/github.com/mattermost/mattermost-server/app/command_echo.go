// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"strconv"
	"strings"
	"time"

	l4g "github.com/alecthomas/log4go"
	"github.com/mattermost/mattermost-server/model"
	goi18n "github.com/nicksnyder/go-i18n/i18n"
)

var echoSem chan bool

type EchoProvider struct {
}

const (
	CMD_ECHO = "echo"
)

func init() {
	RegisterCommandProvider(&EchoProvider{})
}

func (me *EchoProvider) GetTrigger() string {
	return CMD_ECHO
}

func (me *EchoProvider) GetCommand(a *App, T goi18n.TranslateFunc) *model.Command {
	return &model.Command{
		Trigger:          CMD_ECHO,
		AutoComplete:     true,
		AutoCompleteDesc: T("api.command_echo.desc"),
		AutoCompleteHint: T("api.command_echo.hint"),
		DisplayName:      T("api.command_echo.name"),
	}
}

func (me *EchoProvider) DoCommand(a *App, args *model.CommandArgs, message string) *model.CommandResponse {
	if len(message) == 0 {
		return &model.CommandResponse{Text: args.T("api.command_echo.message.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	maxThreads := 100

	delay := 0
	if endMsg := strings.LastIndex(message, "\""); string(message[0]) == "\"" && endMsg > 1 {
		if checkDelay, err := strconv.Atoi(strings.Trim(message[endMsg:], " \"")); err == nil {
			delay = checkDelay
		}
		message = message[1:endMsg]
	} else if strings.Contains(message, " ") {
		delayIdx := strings.LastIndex(message, " ")
		delayStr := strings.Trim(message[delayIdx:], " ")

		if checkDelay, err := strconv.Atoi(delayStr); err == nil {
			delay = checkDelay
			message = message[:delayIdx]
		}
	}

	if delay > 10000 {
		return &model.CommandResponse{Text: args.T("api.command_echo.delay.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	if echoSem == nil {
		// We want one additional thread allowed so we never reach channel lockup
		echoSem = make(chan bool, maxThreads+1)
	}

	if len(echoSem) >= maxThreads {
		return &model.CommandResponse{Text: args.T("api.command_echo.high_volume.app_error"), ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL}
	}

	echoSem <- true
	a.Go(func() {
		defer func() { <-echoSem }()
		post := &model.Post{}
		post.ChannelId = args.ChannelId
		post.RootId = args.RootId
		post.ParentId = args.ParentId
		post.Message = message
		post.UserId = args.UserId

		time.Sleep(time.Duration(delay) * time.Second)

		if _, err := a.CreatePostMissingChannel(post, true); err != nil {
			l4g.Error(args.T("api.command_echo.create.app_error"), err)
		}
	})

	return &model.CommandResponse{}
}
