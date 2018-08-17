package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-server/model"
)

func (p *MatterpollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	json.NewDecoder(r.Body).Decode(&request)
	userID := request.UserId
	pollID := endPollRoute.FindStringSubmatch(r.URL.Path)[1]

	b, _ := p.API.KVGet(pollID)
	poll := Decode(b)

	if userID != poll.Creator {
		resp := &model.PostActionIntegrationResponse{
			EphemeralText: endPollInvalidPermission,
		}
		bytes, _ := json.Marshal(resp)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes)
		return
	}

	// TODO: Error handling
	_ = p.API.KVDelete(pollID)

	message := "Poll is done.\n"
	for _, o := range poll.Options {
		message += fmt.Sprintf("%s:", o.Answer)
		for i := 0; i < len(o.Voter); i++ {
			user, err := p.API.GetUser(o.Voter[i])
			if err != nil {
				//// TODO: Better error handling
				panic("Bad")
			}
			if i+1 == len(o.Voter) && len(o.Voter) > 1 {
				message += " and"
			} else if i != 0 {
				message += ","
			}

			message += fmt.Sprintf(" @%s", user.Username)
		}
		message += "\n"
	}

	resp := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: message,
		},
	}
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func (p *MatterpollPlugin) handleVote(w http.ResponseWriter, r *http.Request) {
	var request model.PostActionIntegrationRequest
	json.NewDecoder(r.Body).Decode(&request)
	userID := request.UserId

	matches := voteRoute.FindStringSubmatch(r.URL.Path)
	pollID := matches[1]
	optionNumber, _ := strconv.Atoi(matches[2])

	b, _ := p.API.KVGet(pollID)
	poll := Decode(b)

	hasVoted := poll.HasVoted(userID)
	_ = poll.UpdateVote(userID, optionNumber)
	p.API.KVSet(pollID, poll.Encode())

	var message string
	if hasVoted {
		message = "Your vote has been updated."
	} else {
		message = "Your vote has been counted."
	}
	resp := &model.PostActionIntegrationResponse{
		EphemeralText: message,
	}
	bytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
