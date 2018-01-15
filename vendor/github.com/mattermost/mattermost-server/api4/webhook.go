// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

func (api *API) InitWebhook() {
	api.BaseRoutes.IncomingHooks.Handle("", api.ApiSessionRequired(createIncomingHook)).Methods("POST")
	api.BaseRoutes.IncomingHooks.Handle("", api.ApiSessionRequired(getIncomingHooks)).Methods("GET")
	api.BaseRoutes.IncomingHook.Handle("", api.ApiSessionRequired(getIncomingHook)).Methods("GET")
	api.BaseRoutes.IncomingHook.Handle("", api.ApiSessionRequired(updateIncomingHook)).Methods("PUT")
	api.BaseRoutes.IncomingHook.Handle("", api.ApiSessionRequired(deleteIncomingHook)).Methods("DELETE")

	api.BaseRoutes.OutgoingHooks.Handle("", api.ApiSessionRequired(createOutgoingHook)).Methods("POST")
	api.BaseRoutes.OutgoingHooks.Handle("", api.ApiSessionRequired(getOutgoingHooks)).Methods("GET")
	api.BaseRoutes.OutgoingHook.Handle("", api.ApiSessionRequired(getOutgoingHook)).Methods("GET")
	api.BaseRoutes.OutgoingHook.Handle("", api.ApiSessionRequired(updateOutgoingHook)).Methods("PUT")
	api.BaseRoutes.OutgoingHook.Handle("", api.ApiSessionRequired(deleteOutgoingHook)).Methods("DELETE")
	api.BaseRoutes.OutgoingHook.Handle("/regen_token", api.ApiSessionRequired(regenOutgoingHookToken)).Methods("POST")

	api.BaseRoutes.Root.Handle("/hooks/commands/{id:[A-Za-z0-9]+}", api.ApiHandler(commandWebhook)).Methods("POST")
	api.BaseRoutes.Root.Handle("/hooks/{id:[A-Za-z0-9]+}", api.ApiHandler(incomingWebhook)).Methods("POST")

	// Old endpoint for backwards compatibility
	api.BaseRoutes.Root.Handle("/api/v3/teams/{team_id:[A-Za-z0-9]+}/hooks/{id:[A-Za-z0-9]+}", api.ApiHandler(incomingWebhook)).Methods("POST")
}

func createIncomingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	hook := model.IncomingWebhookFromJson(r.Body)
	if hook == nil {
		c.SetInvalidParam("incoming_webhook")
		return
	}

	channel, err := c.App.GetChannel(hook.ChannelId)
	if err != nil {
		c.Err = err
		return
	}

	c.LogAudit("attempt")

	if !c.App.SessionHasPermissionToTeam(c.Session, channel.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if channel.Type != model.CHANNEL_OPEN && !c.App.SessionHasPermissionToChannel(c.Session, channel.Id, model.PERMISSION_READ_CHANNEL) {
		c.LogAudit("fail - bad channel permissions")
		c.SetPermissionError(model.PERMISSION_READ_CHANNEL)
		return
	}

	if incomingHook, err := c.App.CreateIncomingWebhookForChannel(c.Session.UserId, channel, hook); err != nil {
		c.Err = err
		return
	} else {
		c.LogAudit("success")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(incomingHook.ToJson()))
	}
}

func updateIncomingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hookId := c.Params.HookId

	updatedHook := model.IncomingWebhookFromJson(r.Body)
	if updatedHook == nil {
		c.SetInvalidParam("incoming_webhook")
		return
	}

	c.LogAudit("attempt")

	oldHook, err := c.App.GetIncomingWebhook(hookId)
	if err != nil {
		c.Err = err
		return
	}

	if updatedHook.TeamId == "" {
		updatedHook.TeamId = oldHook.TeamId
	}

	if updatedHook.TeamId != oldHook.TeamId {
		c.Err = model.NewAppError("updateIncomingHook", "api.webhook.team_mismatch.app_error", nil, "user_id="+c.Session.UserId, http.StatusBadRequest)
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, updatedHook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.Session.UserId != oldHook.UserId && !c.App.SessionHasPermissionToTeam(c.Session, updatedHook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.LogAudit("fail - inappropriate permissions")
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	channel, err := c.App.GetChannel(updatedHook.ChannelId)
	if err != nil {
		c.Err = err
		return
	}

	if channel.Type != model.CHANNEL_OPEN && !c.App.SessionHasPermissionToChannel(c.Session, channel.Id, model.PERMISSION_READ_CHANNEL) {
		c.LogAudit("fail - bad channel permissions")
		c.SetPermissionError(model.PERMISSION_READ_CHANNEL)
		return
	}

	if incomingHook, err := c.App.UpdateIncomingWebhook(oldHook, updatedHook); err != nil {
		c.Err = err
		return
	} else {
		c.LogAudit("success")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(incomingHook.ToJson()))
	}
}

func getIncomingHooks(c *Context, w http.ResponseWriter, r *http.Request) {
	teamId := r.URL.Query().Get("team_id")

	var hooks []*model.IncomingWebhook
	var err *model.AppError

	if len(teamId) > 0 {
		if !c.App.SessionHasPermissionToTeam(c.Session, teamId, model.PERMISSION_MANAGE_WEBHOOKS) {
			c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
			return
		}

		hooks, err = c.App.GetIncomingWebhooksForTeamPage(teamId, c.Params.Page, c.Params.PerPage)
	} else {
		if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_WEBHOOKS) {
			c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
			return
		}

		hooks, err = c.App.GetIncomingWebhooksPage(c.Params.Page, c.Params.PerPage)
	}

	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.IncomingWebhookListToJson(hooks)))
}

func getIncomingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hookId := c.Params.HookId

	var err *model.AppError
	var hook *model.IncomingWebhook
	var channel *model.Channel

	if hook, err = c.App.GetIncomingWebhook(hookId); err != nil {
		c.Err = err
		return
	} else {
		channel, err = c.App.GetChannel(hook.ChannelId)
		if err != nil {
			c.Err = err
			return
		}

		if !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) ||
			(channel.Type != model.CHANNEL_OPEN && !c.App.SessionHasPermissionToChannel(c.Session, hook.ChannelId, model.PERMISSION_READ_CHANNEL)) {
			c.LogAudit("fail - bad permissions")
			c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
			return
		} else {
			w.Write([]byte(hook.ToJson()))
			return
		}
	}
}

func deleteIncomingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hookId := c.Params.HookId

	var err *model.AppError
	var hook *model.IncomingWebhook
	var channel *model.Channel

	if hook, err = c.App.GetIncomingWebhook(hookId); err != nil {
		c.Err = err
		return
	} else {
		channel, err = c.App.GetChannel(hook.ChannelId)
		if err != nil {
			c.Err = err
			return
		}

		if !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) ||
			(channel.Type != model.CHANNEL_OPEN && !c.App.SessionHasPermissionToChannel(c.Session, hook.ChannelId, model.PERMISSION_READ_CHANNEL)) {
			c.LogAudit("fail - bad permissions")
			c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
			return
		} else {
			if err = c.App.DeleteIncomingWebhook(hookId); err != nil {
				c.Err = err
				return
			}

			ReturnStatusOK(w)
		}
	}
}

func updateOutgoingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	toUpdateHook := model.OutgoingWebhookFromJson(r.Body)
	if toUpdateHook == nil {
		c.SetInvalidParam("outgoing_webhook")
		return
	}

	c.LogAudit("attempt")

	toUpdateHook.CreatorId = c.Session.UserId

	if !c.App.SessionHasPermissionToTeam(c.Session, toUpdateHook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	oldHook, err := c.App.GetOutgoingWebhook(toUpdateHook.Id)
	if err != nil {
		c.Err = err
		return
	}

	if c.Session.UserId != oldHook.CreatorId && !c.App.SessionHasPermissionToTeam(c.Session, oldHook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.LogAudit("fail - inappropriate permissions")
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	rhook, err := c.App.UpdateOutgoingWebhook(oldHook, toUpdateHook)
	if err != nil {
		c.Err = err
		return
	}

	c.LogAudit("success")
	w.Write([]byte(rhook.ToJson()))
}

func createOutgoingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	hook := model.OutgoingWebhookFromJson(r.Body)
	if hook == nil {
		c.SetInvalidParam("outgoing_webhook")
		return
	}

	c.LogAudit("attempt")

	hook.CreatorId = c.Session.UserId

	if !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if rhook, err := c.App.CreateOutgoingWebhook(hook); err != nil {
		c.LogAudit("fail")
		c.Err = err
		return
	} else {
		c.LogAudit("success")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(rhook.ToJson()))
	}
}

func getOutgoingHooks(c *Context, w http.ResponseWriter, r *http.Request) {
	channelId := r.URL.Query().Get("channel_id")
	teamId := r.URL.Query().Get("team_id")

	var hooks []*model.OutgoingWebhook
	var err *model.AppError

	if len(channelId) > 0 {
		if !c.App.SessionHasPermissionToChannel(c.Session, channelId, model.PERMISSION_MANAGE_WEBHOOKS) {
			c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
			return
		}

		hooks, err = c.App.GetOutgoingWebhooksForChannelPage(channelId, c.Params.Page, c.Params.PerPage)
	} else if len(teamId) > 0 {
		if !c.App.SessionHasPermissionToTeam(c.Session, teamId, model.PERMISSION_MANAGE_WEBHOOKS) {
			c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
			return
		}

		hooks, err = c.App.GetOutgoingWebhooksForTeamPage(teamId, c.Params.Page, c.Params.PerPage)
	} else {
		if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_WEBHOOKS) {
			c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
			return
		}

		hooks, err = c.App.GetOutgoingWebhooksPage(c.Params.Page, c.Params.PerPage)
	}

	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.OutgoingWebhookListToJson(hooks)))
}

func getOutgoingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hook, err := c.App.GetOutgoingWebhook(c.Params.HookId)
	if err != nil {
		c.Err = err
		return
	}

	c.LogAudit("attempt")

	if !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.Session.UserId != hook.CreatorId && !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.LogAudit("fail - inappropriate permissions")
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	c.LogAudit("success")
	w.Write([]byte(hook.ToJson()))
}

func regenOutgoingHookToken(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hook, err := c.App.GetOutgoingWebhook(c.Params.HookId)
	if err != nil {
		c.Err = err
		return
	}

	c.LogAudit("attempt")

	if !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.Session.UserId != hook.CreatorId && !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.LogAudit("fail - inappropriate permissions")
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	if rhook, err := c.App.RegenOutgoingWebhookToken(hook); err != nil {
		c.Err = err
		return
	} else {
		w.Write([]byte(rhook.ToJson()))
	}
}

func deleteOutgoingHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hook, err := c.App.GetOutgoingWebhook(c.Params.HookId)
	if err != nil {
		c.Err = err
		return
	}

	c.LogAudit("attempt")

	if !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.Session.UserId != hook.CreatorId && !c.App.SessionHasPermissionToTeam(c.Session, hook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.LogAudit("fail - inappropriate permissions")
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	if err := c.App.DeleteOutgoingWebhook(hook.Id); err != nil {
		c.LogAudit("fail")
		c.Err = err
		return
	}

	c.LogAudit("success")
	ReturnStatusOK(w)
}

func incomingWebhook(c *Context, w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	r.ParseForm()

	var payload io.Reader
	contentType := r.Header.Get("Content-Type")
	if strings.Split(contentType, "; ")[0] == "application/x-www-form-urlencoded" {
		payload = strings.NewReader(r.FormValue("payload"))
	} else {
		payload = r.Body
	}

	if c.App.Config().LogSettings.EnableWebhookDebugging {
		var err error
		payload, err = utils.InfoReader(
			payload,
			utils.T("api.webhook.incoming.debug"),
		)
		if err != nil {
			c.Err = model.NewAppError("incomingWebhook", "api.webhook.incoming.debug.error", nil, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	parsedRequest, decodeError := model.IncomingWebhookRequestFromJson(payload)

	if decodeError != nil {
		c.Err = decodeError
		return
	}

	err := c.App.HandleIncomingWebhook(id, parsedRequest)
	if err != nil {
		c.Err = err
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

func commandWebhook(c *Context, w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	response := model.CommandResponseFromHTTPBody(r.Header.Get("Content-Type"), r.Body)

	err := c.App.HandleCommandWebhook(id, response)
	if err != nil {
		c.Err = err
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}
