// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"net/http"

	"github.com/mattermost/mattermost-server/model"
)

func (api *API) InitStatus() {
	api.BaseRoutes.User.Handle("/status", api.ApiSessionRequired(getUserStatus)).Methods("GET")
	api.BaseRoutes.Users.Handle("/status/ids", api.ApiSessionRequired(getUserStatusesByIds)).Methods("POST")
	api.BaseRoutes.User.Handle("/status", api.ApiSessionRequired(updateUserStatus)).Methods("PUT")
}

func getUserStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	// No permission check required

	if statusMap, err := c.App.GetUserStatusesByIds([]string{c.Params.UserId}); err != nil {
		c.Err = err
		return
	} else {
		if len(statusMap) == 0 {
			c.Err = model.NewAppError("UserStatus", "api.status.user_not_found.app_error", nil, "", http.StatusNotFound)
			return
		} else {
			w.Write([]byte(statusMap[0].ToJson()))
		}
	}
}

func getUserStatusesByIds(c *Context, w http.ResponseWriter, r *http.Request) {
	userIds := model.ArrayFromJson(r.Body)

	if len(userIds) == 0 {
		c.SetInvalidParam("user_ids")
		return
	}

	// No permission check required

	if statusMap, err := c.App.GetUserStatusesByIds(userIds); err != nil {
		c.Err = err
		return
	} else {
		w.Write([]byte(model.StatusListToJson(statusMap)))
	}
}

func updateUserStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	status := model.StatusFromJson(r.Body)
	if status == nil {
		c.SetInvalidParam("status")
		return
	}

	if !c.App.SessionHasPermissionToUser(c.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	switch status.Status {
	case "online":
		c.App.SetStatusOnline(c.Params.UserId, "", true)
	case "offline":
		c.App.SetStatusOffline(c.Params.UserId, true)
	case "away":
		c.App.SetStatusAwayIfNeeded(c.Params.UserId, true)
	case "dnd":
		c.App.SetStatusDoNotDisturb(c.Params.UserId)
	default:
		c.SetInvalidParam("status")
		return
	}

	getUserStatus(c, w, r)
}
