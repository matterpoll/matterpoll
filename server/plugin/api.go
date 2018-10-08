package plugin

import (
	"fmt"
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

	// Parameter: Question, Permalink
	endPollSuccessfullyFormat    = "The poll **%s** has ended and the original post have been updated. You can jump to it by pressing [here](%s)."
	endPollAnnouncementPostError = "Failed to post the end poll announcement."
	endPollInvalidPermission     = "Only the creator of a poll and System Admins are allowed to end it."

	deletePollInvalidPermission = "Only the creator of a poll and System Admins are allowed to delete it."
	deletePollSuccess           = "Succefully deleted the poll."
)

// InitAPI initializes the REST API
func (p *MatterpollPlugin) InitAPI() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", p.handleInfo).Methods("GET")
	r.HandleFunc("/"+iconFilename, p.handleLogo).Methods("GET")
	s := r.PathPrefix("/api/v1").Subrouter()
	s.HandleFunc("/polls/{id:[a-z0-9]+}/vote/{optionNumber:[0-9]+}", p.handleVote).Methods("POST")
	s.HandleFunc("/polls/{id:[a-z0-9]+}/end", p.handleEndPoll).Methods("POST")
	s.HandleFunc("/polls/{id:[a-z0-9]+}/delete", p.handleDeletePoll).Methods("POST")
	return r
}

func (p *MatterpollPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)
	p.router.ServeHTTP(w, r)
}

func (p *MatterpollPlugin) handleInfo(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, infoMessage)
}

func (p *MatterpollPlugin) handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=604800")
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

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
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
	if err := poll.UpdateVote(userID, optionNumber); err != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	if err = p.Store.Poll().Save(poll); err != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	post := &model.Post{}
	model.ParseSlackAttachment(post, poll.ToPostActions(*p.ServerConfig.ServiceSettings.SiteURL, PluginId, displayName))
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

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	hasPermission, appErr := p.HasPermission(poll, request.UserId)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}
	if !hasPermission {
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

	if err := p.Store.Poll().Delete(poll); err != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	p.postEndPollAnnouncement(request, poll.Question)

	writePostActionIntegrationResponse(w, response)
}

func (p *MatterpollPlugin) postEndPollAnnouncement(request *model.PostActionIntegrationRequest, question string) {
	team, err := p.API.GetTeam(request.TeamId)
	if err != nil {
		p.API.LogError(endPollAnnouncementPostError, "details", fmt.Sprintf("failed to GetTeam with TeamId: %s", request.TeamId))
		return
	}
	permalink := fmt.Sprintf("%s/%s/pl/%s", *p.ServerConfig.ServiceSettings.SiteURL, team.Name, request.PostId)

	pollPost, err := p.API.GetPost(request.PostId)
	if err != nil {
		p.API.LogError(endPollAnnouncementPostError, "details", fmt.Sprintf("failed to GetPost with PostId: %s", request.PostId))
		return
	}
	channelID := pollPost.ChannelId

	endPost := &model.Post{
		UserId:    request.UserId,
		ChannelId: channelID,
		RootId:    request.PostId,
		Message:   fmt.Sprintf(endPollSuccessfullyFormat, question, permalink),
		Type:      model.POST_DEFAULT,
		Props: model.StringInterface{
			"override_username": responseUsername,
			"override_icon_url": fmt.Sprintf(responseIconURL, *p.ServerConfig.ServiceSettings.SiteURL, PluginId),
			"from_webhook":      "true",
		},
	}
	_, err = p.API.CreatePost(endPost)
	if err != nil {
		p.API.LogError(endPollAnnouncementPostError, "details", "failed to CreatePost")
	}
}

func (p *MatterpollPlugin) handleDeletePoll(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequesteFromJson(r.Body)
	if request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}

	hasPermission, appErr := p.HasPermission(poll, request.UserId)
	if appErr != nil {
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}
	if !hasPermission {
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

	if err := p.Store.Poll().Delete(poll); err != nil {
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
