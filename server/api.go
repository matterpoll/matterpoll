package main

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	infoMessage = "Thanks for using Matterpoll v" + PluginVersion + "\n"

	iconFilename = "logo_dark.png"
	iconPath     = "plugins/" + PluginId + "/"

	voteCounted = "Your vote has been counted."
	voteUpdated = "Your vote has been updated."

	endPollInvalidPermission = "Only the creator of a poll is allowed to end it."

	deletePollInvalidPermission = "Only the creator of a poll is allowed to delete it."
	deletePollSuccess           = "Succefully deleted the poll."
)

func (p *MatterpollPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)
	p.router.ServeHTTP(w, r)
}

func (p *MatterpollPlugin) handleInfo(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, infoMessage)
}

func (p *MatterpollPlugin) handleLogo(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, iconPath+iconFilename)
}

func (p *MatterpollPlugin) handleVote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pollID := vars["id"]
	optionNumber, _ := strconv.Atoi(vars["optionNumber"])
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequesteFromJson(r.Body)
	if request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID := request.UserId

	b, appErr := p.API.KVGet(pollID)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}
	poll := Decode(b)
	if poll == nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	hasVoted := poll.HasVoted(userID)
	err := poll.UpdateVote(userID, optionNumber)
	if err != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	appErr = p.API.KVSet(pollID, poll.Encode())
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	post := &model.Post{}
	post.AddProp("attachments", poll.ToPostActions(*p.ServerConfig.ServiceSettings.SiteURL, pollID, displayName))
	response.Update = post

	if hasVoted {
		response.EphemeralText = voteUpdated
	} else {
		response.EphemeralText = voteCounted
	}
	writePostActionIntegrationResponse(w, response)
}

func (p *MatterpollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequesteFromJson(r.Body)
	if request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID := request.UserId

	b, appErr := p.API.KVGet(pollID)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}
	poll := Decode(b)
	if poll == nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	if userID != poll.Creator {
		response.EphemeralText = endPollInvalidPermission
		writePostActionIntegrationResponse(w, response)
		return
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	response.Update, appErr = poll.ToEndPollPost(displayName, p.ConvertUserIDToDisplayName)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	appErr = p.API.KVDelete(pollID)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	writePostActionIntegrationResponse(w, response)
}

func (p *MatterpollPlugin) handleDeletePoll(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequesteFromJson(r.Body)
	if request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID := request.UserId

	b, appErr := p.API.KVGet(pollID)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}
	poll := Decode(b)
	if poll == nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	if userID != poll.Creator {
		response.EphemeralText = deletePollInvalidPermission
		writePostActionIntegrationResponse(w, response)
		return
	}

	appErr = p.API.DeletePost(request.PostId)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	appErr = p.API.KVDelete(pollID)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}
	response.EphemeralText = deletePollSuccess

	writePostActionIntegrationResponse(w, response)
}

func writePostActionIntegrationResponse(w http.ResponseWriter, response *model.PostActionIntegrationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, response.ToJson())
}