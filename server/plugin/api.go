package plugin

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	infoMessage = "Thanks for using Matterpoll v" + PluginVersion + "\n"

	iconFilename = "logo_dark.png"

	voteCounted = "Your vote has been counted."
	voteUpdated = "Your vote has been updated."

	// Parameter: Question, Permalink
	endPollSuccessfullyFormat    = "The poll **%s** has ended and the original post have been updated. You can jump to it by pressing [here](%s)."
	endPollAnnouncementPostError = "Failed to post the end poll announcement."
	endPollInvalidPermission     = "Only the creator of a poll and System Admins are allowed to end it."

	deletePollInvalidPermission = "Only the creator of a poll and System Admins are allowed to delete it."
	deletePollSuccess           = "Succefully deleted the poll."

	addOptionInvalidPermission = "Only the creator of a poll and System Admins are allowed to add options."
	addOptionSuccess           = "Succefully added an option."
	addOptionKey               = "answerOption"
)

// InitAPI initializes the REST API
func (p *MatterpollPlugin) InitAPI() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", p.handleInfo).Methods("GET")
	r.HandleFunc("/"+iconFilename, p.handleLogo).Methods("GET")

	apiV1 := r.PathPrefix("/api/v1").Subrouter()

	pollRouter := apiV1.PathPrefix("/polls/{id:[a-z0-9]+}").Subrouter()
	pollRouter.HandleFunc("/vote/{optionNumber:[0-9]+}", p.handleVote).Methods("POST")
	pollRouter.HandleFunc("/option/add", p.handleAddOption).Methods("POST")
	pollRouter.HandleFunc("/option/add/request", p.handleAddOptionDialogRequest).Methods("POST")
	pollRouter.HandleFunc("/end", p.handleEndPoll).Methods("POST")
	pollRouter.HandleFunc("/delete", p.handleDeletePoll).Methods("POST")
	return r
}

func (p *MatterpollPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)
	p.router.ServeHTTP(w, r)
}

func (p *MatterpollPlugin) handleInfo(w http.ResponseWriter, _ *http.Request) {
	_, _ = io.WriteString(w, infoMessage)
}

func (p *MatterpollPlugin) handleLogo(w http.ResponseWriter, r *http.Request) {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		p.API.LogWarn("failed to get bundle path", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, filepath.Join(bundlePath, "assets", iconFilename))
}

func (p *MatterpollPlugin) handleVote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pollID := vars["id"]
	optionNumber, _ := strconv.Atoi(vars["optionNumber"])
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequestFromJson(r.Body)
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
	if err = poll.UpdateVote(userID, optionNumber); err != nil {
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

func (p *MatterpollPlugin) handleAddOption(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]

	request := model.SubmitDialogRequestFromJson(r.Body)
	if request == nil {
		p.API.LogError("failed to decode request")
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandGenericError)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		p.API.LogError("failed to get poll", "err", err.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandGenericError)
		w.WriteHeader(http.StatusOK)
		return
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		p.API.LogError("failed to get display name for creator", "err", appErr.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandGenericError)
		w.WriteHeader(http.StatusOK)
		return
	}

	post, appErr := p.API.GetPost(request.CallbackId)
	if appErr != nil {
		p.API.LogError("failed to get post", "err", appErr.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandGenericError)
		w.WriteHeader(http.StatusOK)
		return
	}

	answerOption, ok := request.Submission[addOptionKey].(string)
	if !ok {
		p.API.LogError("failed to parse request")
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandGenericError)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := poll.AddAnswerOption(answerOption); err != nil {
		response := &model.SubmitDialogResponse{
			Errors: map[string]string{
				addOptionKey: err.Error(),
			},
		}
		writeSubmitDialogResponse(w, response)
		return
	}

	model.ParseSlackAttachment(post, poll.ToPostActions(*p.ServerConfig.ServiceSettings.SiteURL, PluginId, displayName))
	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		p.API.LogError("failed to update post", "err", appErr.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandGenericError)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err = p.Store.Poll().Save(poll); err != nil {
		p.API.LogError("failed to get save poll", "err", err.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandGenericError)
		w.WriteHeader(http.StatusOK)
		return
	}

	p.SendEphemeralPost(request.ChannelId, request.UserId, addOptionSuccess)
	w.WriteHeader(http.StatusOK)
}

func (p *MatterpollPlugin) handleAddOptionDialogRequest(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequestFromJson(r.Body)
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

	if !poll.Settings.PublicAddOption {
		hasPermission, appErr := p.HasPermission(poll, request.UserId)
		if appErr != nil {
			response.EphemeralText = commandGenericError
			writePostActionIntegrationResponse(w, response)
			return
		}
		if !hasPermission {
			response.EphemeralText = addOptionInvalidPermission
			writePostActionIntegrationResponse(w, response)
			return
		}
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("%s/plugins/%s/api/v1/polls/%s/option/add", siteURL, PluginId, pollID),
		Dialog: model.Dialog{
			Title:       "Add Option",
			IconURL:     fmt.Sprintf(responseIconURL, siteURL, PluginId),
			CallbackId:  request.PostId,
			SubmitLabel: "Add",
			Elements: []model.DialogElement{{
				DisplayName: "Option",
				Name:        addOptionKey,
				Type:        "text",
				SubType:     "text",
			},
			},
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		p.API.LogError("failed to open add option dialog ", "err", appErr.Error())
		response.EphemeralText = commandGenericError
		writePostActionIntegrationResponse(w, response)
		return
	}
	writePostActionIntegrationResponse(w, response)
}

func (p *MatterpollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequestFromJson(r.Body)
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

	if _, err = p.API.CreatePost(endPost); err != nil {
		p.API.LogError(endPollAnnouncementPostError, "details", "failed to CreatePost")
	}
}

func (p *MatterpollPlugin) handleDeletePoll(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]
	response := &model.PostActionIntegrationResponse{}

	request := model.PostActionIntegrationRequestFromJson(r.Body)
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
	_, _ = w.Write(response.ToJson())
}

func writeSubmitDialogResponse(w http.ResponseWriter, response *model.SubmitDialogResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response.ToJson())
}
