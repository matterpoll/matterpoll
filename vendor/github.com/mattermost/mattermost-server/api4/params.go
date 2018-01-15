// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	PAGE_DEFAULT          = 0
	PER_PAGE_DEFAULT      = 60
	PER_PAGE_MAXIMUM      = 200
	LOGS_PER_PAGE_DEFAULT = 10000
	LOGS_PER_PAGE_MAXIMUM = 10000
)

type ApiParams struct {
	UserId         string
	TeamId         string
	InviteId       string
	TokenId        string
	ChannelId      string
	PostId         string
	FileId         string
	PluginId       string
	CommandId      string
	HookId         string
	ReportId       string
	EmojiId        string
	AppId          string
	Email          string
	Username       string
	TeamName       string
	ChannelName    string
	PreferenceName string
	EmojiName      string
	Category       string
	Service        string
	JobId          string
	JobType        string
	ActionId       string
	Page           int
	PerPage        int
	LogsPerPage    int
	Permanent      bool
}

func ApiParamsFromRequest(r *http.Request) *ApiParams {
	params := &ApiParams{}

	props := mux.Vars(r)

	if val, ok := props["user_id"]; ok {
		params.UserId = val
	}

	if val, ok := props["team_id"]; ok {
		params.TeamId = val
	}

	if val, ok := props["invite_id"]; ok {
		params.InviteId = val
	}

	if val, ok := props["token_id"]; ok {
		params.TokenId = val
	}

	if val, ok := props["channel_id"]; ok {
		params.ChannelId = val
	}

	if val, ok := props["post_id"]; ok {
		params.PostId = val
	}

	if val, ok := props["file_id"]; ok {
		params.FileId = val
	}

	if val, ok := props["plugin_id"]; ok {
		params.PluginId = val
	}

	if val, ok := props["command_id"]; ok {
		params.CommandId = val
	}

	if val, ok := props["hook_id"]; ok {
		params.HookId = val
	}

	if val, ok := props["report_id"]; ok {
		params.ReportId = val
	}

	if val, ok := props["emoji_id"]; ok {
		params.EmojiId = val
	}

	if val, ok := props["app_id"]; ok {
		params.AppId = val
	}

	if val, ok := props["email"]; ok {
		params.Email = val
	}

	if val, ok := props["username"]; ok {
		params.Username = val
	}

	if val, ok := props["team_name"]; ok {
		params.TeamName = val
	}

	if val, ok := props["channel_name"]; ok {
		params.ChannelName = val
	}

	if val, ok := props["category"]; ok {
		params.Category = val
	}

	if val, ok := props["service"]; ok {
		params.Service = val
	}

	if val, ok := props["preference_name"]; ok {
		params.PreferenceName = val
	}

	if val, ok := props["emoji_name"]; ok {
		params.EmojiName = val
	}

	if val, ok := props["job_id"]; ok {
		params.JobId = val
	}

	if val, ok := props["job_type"]; ok {
		params.JobType = val
	}

	if val, ok := props["action_id"]; ok {
		params.ActionId = val
	}

	if val, err := strconv.Atoi(r.URL.Query().Get("page")); err != nil || val < 0 {
		params.Page = PAGE_DEFAULT
	} else {
		params.Page = val
	}

	if val, err := strconv.ParseBool(r.URL.Query().Get("permanent")); err != nil {
		params.Permanent = val
	}

	if val, err := strconv.Atoi(r.URL.Query().Get("per_page")); err != nil || val < 0 {
		params.PerPage = PER_PAGE_DEFAULT
	} else if val > PER_PAGE_MAXIMUM {
		params.PerPage = PER_PAGE_MAXIMUM
	} else {
		params.PerPage = val
	}

	if val, err := strconv.Atoi(r.URL.Query().Get("logs_per_page")); err != nil || val < 0 {
		params.LogsPerPage = LOGS_PER_PAGE_DEFAULT
	} else if val > LOGS_PER_PAGE_MAXIMUM {
		params.LogsPerPage = LOGS_PER_PAGE_MAXIMUM
	} else {
		params.LogsPerPage = val
	}

	return params
}
