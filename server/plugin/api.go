package plugin

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	infoMessage = "Thanks for using Matterpoll v" + PluginVersion + "\n"

	iconFilename = "logo_dark.png"

	addOptionKey = "answerOption"
)

var (
	responseVoteCounted = &i18n.Message{
		ID:    "response.vote.counted",
		Other: "Your vote has been counted.",
	}
	responseVoteUpdated = &i18n.Message{
		ID:    "response.vote.updated",
		Other: "Your vote has been updated.",
	}

	responseAddOptionSuccess = &i18n.Message{
		ID:    "response.addOption.success",
		Other: "Successfully added the option.",
	}
	responseAddOptionInvalidPermission = &i18n.Message{
		ID:    "response.addOption.invalidPermission",
		Other: "Only the creator of a poll and System Admins are allowed to add options.",
	}

	dialogAddOptionTitle = &i18n.Message{
		ID:    "dialog.addOption.title",
		Other: "Add Option",
	}
	dialogAddOptionSubmitLabel = &i18n.Message{
		ID:    "dialog.addOption.submitLabel",
		Other: "Add",
	}
	dialogAddOptionElementDisplayName = &i18n.Message{
		ID:    "dialog.addOption.element.displayName",
		Other: "Option",
	}

	responseEndPollSuccessfully = &i18n.Message{
		ID:    "response.endPoll.successfully",
		Other: "The poll **{{.Question}}** has ended and the original post have been updated. You can jump to it by pressing [here]({{.Link}}).",
	}
	responseEndPollInvalidPermission = &i18n.Message{
		ID:    "response.endPoll.invalidPermission",
		Other: "Only the creator of a poll and System Admins are allowed to end it.",
	}

	responseDeletePollSuccess = &i18n.Message{
		ID:    "response.deletePoll.success",
		Other: "Successfully deleted the poll.",
	}
	responseDeletePollInvalidPermission = &i18n.Message{
		ID:    "response.deletePoll.invalidPermission",
		Other: "Only the creator of a poll and System Admins are allowed to delete it.",
	}
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
	root := utils.GetPluginRootPath()
	if root == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	iconPath := root + "/" + iconFilename
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, iconPath)
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
	userLocalizer := p.getUserLocalizer(userID)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	hasVoted := poll.HasVoted(userID)
	if err = poll.UpdateVote(userID, optionNumber); err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	if err = p.Store.Poll().Save(poll); err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	post := &model.Post{}
	pollLocalizer := i18n.NewLocalizer(p.bundle, *p.ServerConfig.LocalizationSettings.DefaultServerLocale)
	model.ParseSlackAttachment(post, poll.ToPostActions(pollLocalizer, *p.ServerConfig.ServiceSettings.SiteURL, PluginId, displayName))
	response.Update = post

	if hasVoted {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, responseVoteUpdated)
	} else {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, responseVoteCounted)
	}
	writePostActionIntegrationResponse(w, response)
}

func (p *MatterpollPlugin) handleAddOption(w http.ResponseWriter, r *http.Request) {
	pollID := mux.Vars(r)["id"]

	request := model.SubmitDialogRequestFromJson(r.Body)
	if request == nil {
		p.API.LogError("failed to decode request")
		p.SendEphemeralPost(request.ChannelId, request.UserId, commandErrorGeneric.Other)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		p.API.LogError("failed to get poll", "err", err.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric))
		w.WriteHeader(http.StatusOK)
		return
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		p.API.LogError("failed to get display name for creator", "err", appErr.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric))
		w.WriteHeader(http.StatusOK)
		return
	}

	post, appErr := p.API.GetPost(request.CallbackId)
	if appErr != nil {
		p.API.LogError("failed to get post", "err", appErr.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric))
		w.WriteHeader(http.StatusOK)
		return
	}

	answerOption, ok := request.Submission[addOptionKey].(string)
	if !ok {
		p.API.LogError("failed to parse request")
		p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric))
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

	publicLocalizer := p.getServerLocalizer()
	model.ParseSlackAttachment(post, poll.ToPostActions(publicLocalizer, *p.ServerConfig.ServiceSettings.SiteURL, PluginId, displayName))
	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		p.API.LogError("failed to update post", "err", appErr.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric))
		w.WriteHeader(http.StatusOK)
		return
	}

	if err = p.Store.Poll().Save(poll); err != nil {
		p.API.LogError("failed to get save poll", "err", err.Error())
		p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric))
		w.WriteHeader(http.StatusOK)
		return
	}

	p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, responseAddOptionSuccess))
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
	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	if !poll.Settings.PublicAddOption {
		hasPermission, appErr := p.HasPermission(poll, request.UserId)
		if appErr != nil {
			response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
			writePostActionIntegrationResponse(w, response)
			return
		}
		if !hasPermission {
			response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, responseAddOptionInvalidPermission)
			writePostActionIntegrationResponse(w, response)
			return
		}
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("%s/plugins/%s/api/v1/polls/%s/option/add", siteURL, PluginId, pollID),
		Dialog: model.Dialog{
			Title:       p.LocalizeDefaultMessage(userLocalizer, dialogAddOptionTitle),
			IconURL:     fmt.Sprintf(responseIconURL, siteURL, PluginId),
			CallbackId:  request.PostId,
			SubmitLabel: p.LocalizeDefaultMessage(userLocalizer, dialogAddOptionSubmitLabel),
			Elements: []model.DialogElement{{
				DisplayName: p.LocalizeDefaultMessage(userLocalizer, dialogAddOptionElementDisplayName),
				Name:        addOptionKey,
				Type:        "text",
				SubType:     "text",
			},
			},
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		p.API.LogError("failed to open add option dialog ", "err", appErr.Error())
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
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

	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	hasPermission, appErr := p.HasPermission(poll, request.UserId)
	if appErr != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}
	if !hasPermission {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, responseEndPollInvalidPermission)
		writePostActionIntegrationResponse(w, response)
		return
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	publicLocalizer := p.getServerLocalizer()
	response.Update, appErr = poll.ToEndPollPost(publicLocalizer, displayName, p.ConvertUserIDToDisplayName)
	if appErr != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	if err := p.Store.Poll().Delete(poll); err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	p.postEndPollAnnouncement(request, poll.Question)

	writePostActionIntegrationResponse(w, response)
}

func (p *MatterpollPlugin) postEndPollAnnouncement(request *model.PostActionIntegrationRequest, question string) {
	endPollAnnouncementPostError := "Failed to post the end poll announcement."

	team, err := p.API.GetTeam(request.TeamId)
	if err != nil {
		p.API.LogError(endPollAnnouncementPostError, "details", fmt.Sprintf("failed to GetTeam with TeamId: %s", request.TeamId))
		return
	}
	link := fmt.Sprintf("%s/%s/pl/%s", *p.ServerConfig.ServiceSettings.SiteURL, team.Name, request.PostId)

	pollPost, err := p.API.GetPost(request.PostId)
	if err != nil {
		p.API.LogError(endPollAnnouncementPostError, "details", fmt.Sprintf("failed to GetPost with PostId: %s", request.PostId))
		return
	}
	channelID := pollPost.ChannelId

	publicLocalizer := p.getServerLocalizer()

	endPost := &model.Post{
		UserId:    request.UserId,
		ChannelId: channelID,
		RootId:    request.PostId,
		Message: p.LocalizeWithConfig(publicLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: responseEndPollSuccessfully,
			TemplateData: map[string]interface{}{
				"Question": question,
				"Link":     link,
			}}),
		Type: model.POST_DEFAULT,
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
	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	hasPermission, appErr := p.HasPermission(poll, request.UserId)
	if appErr != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}
	if !hasPermission {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, responseDeletePollInvalidPermission)
		writePostActionIntegrationResponse(w, response)
		return
	}

	appErr = p.API.DeletePost(request.PostId)
	if appErr != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}

	if err := p.Store.Poll().Delete(poll); err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		writePostActionIntegrationResponse(w, response)
		return
	}
	response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, responseDeletePollSuccess)

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
