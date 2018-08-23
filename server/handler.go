package main

import (
	"io"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-server/model"
)

const (
	voteCounted = "Your vote has been counted."
	voteUpdated = "Your vote has been updated."

	endPollInvalidPermission = "Only the creator of a poll is allowed to end it."

	deletePollInvalidPermission   = "Only the creator of a poll is allowed to delete it."
	deletePollFeatureNotAvailable = "This feature is only available on Mattermost v5.3."
	deletePollSuccess             = "Succefully deleted the poll."
)

func (p *MatterpollPlugin) handleVote(w http.ResponseWriter, r *http.Request) {

	matches := voteRoute.FindStringSubmatch(r.URL.Path)
	pollID := matches[1]
	optionNumber, _ := strconv.Atoi(matches[2])
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

	if hasVoted {
		response.EphemeralText = voteUpdated
	} else {
		response.EphemeralText = voteCounted
	}
	writePostActionIntegrationResponse(w, response)
}

func (p *MatterpollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	pollID := endPollRoute.FindStringSubmatch(r.URL.Path)[1]
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

	user, err := p.API.GetUser(poll.Creator)
	if err != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	convert := func(userID string) (string, *model.AppError) {
		user, err := p.API.GetUser(userID)
		if err != nil {
			return "", err
		}
		return user.Username, nil
	}

	response.Update, appErr = poll.ToEndPollPost(user.GetFullName(), convert)
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
	pollID := deletePollRoute.FindStringSubmatch(r.URL.Path)[1]
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

	if request.PostId == "" {
		response.EphemeralText = deletePollFeatureNotAvailable
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
