package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"

	"github.com/matterpoll/matterpoll/server/poll"
)

const (
	iconFilename = "logo_dark.png"

	addOptionKey = "answerOption"
	questionKey  = "question"
)

type (
	postActionHandler   func(map[string]string, *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error)
	submitDialogHandler func(map[string]string, *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error)
)

var (
	infoMessage = "Thanks for using Matterpoll v" + manifest.Version + "\n"

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

	responseEndPollSuccessfully = &i18n.Message{
		ID:    "response.endPoll.successfully",
		Other: "The poll **{{.Question}}** has ended and the original post has been updated. You can jump to it by pressing [here]({{.Link}}).",
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
	r.HandleFunc("/", p.handleInfo).Methods(http.MethodGet)
	r.HandleFunc("/"+iconFilename, p.handleLogo).Methods(http.MethodGet)

	apiV1 := r.PathPrefix("/api/v1").Subrouter()
	apiV1.Use(checkAuthenticity)
	apiV1.HandleFunc("/configuration", p.handlePluginConfiguration).Methods(http.MethodGet)

	apiV1.HandleFunc("/polls/create", p.handleSubmitDialogRequest(p.handleCreatePoll)).Methods(http.MethodPost)
	pollRouter := apiV1.PathPrefix("/polls/{id:[a-z0-9]+}").Subrouter()
	pollRouter.HandleFunc("/vote/{optionNumber:[0-9]+}", p.handlePostActionIntegrationRequest(p.handleVote)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/option/add/request", p.handlePostActionIntegrationRequest(p.handleAddOption)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/option/add", p.handleSubmitDialogRequest(p.handleAddOptionConfirm)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/end", p.handlePostActionIntegrationRequest(p.handleEndPoll)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/end/confirm", p.handleSubmitDialogRequest(p.handleEndPollConfirm)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/delete", p.handlePostActionIntegrationRequest(p.handleDeletePoll)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/delete/confirm", p.handleSubmitDialogRequest(p.handleDeletePollConfirm)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/metadata", p.handlePollMetadata).Methods(http.MethodGet)
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

func (p *MatterpollPlugin) handlePluginConfiguration(w http.ResponseWriter, r *http.Request) {
	configuration := p.getConfiguration()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(configuration)
	if err != nil {
		p.API.LogWarn("failed to write configuration response.", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func checkAuthenticity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Mattermost-User-ID") == "" {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *MatterpollPlugin) handlePostActionIntegrationRequest(handler postActionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := model.PostActionIntegrationRequestFromJson(r.Body)
		if request == nil {
			p.API.LogWarn("failed to decode PostActionIntegrationRequest")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		userLocalizer := p.getUserLocalizer(request.UserId)

		msg, update, err := handler(mux.Vars(r), request)
		if err != nil {
			p.API.LogWarn("failed to handle PostActionIntegrationRequest", "error", err.Error())
		}

		if msg != nil {
			p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, msg))
		}

		response := &model.PostActionIntegrationResponse{}
		if update != nil {
			response.Update = update
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			p.API.LogWarn("failed to write PostActionIntegrationResponse", "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func (p *MatterpollPlugin) handleSubmitDialogRequest(handler submitDialogHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := model.SubmitDialogRequestFromJson(r.Body)
		if request == nil {
			p.API.LogWarn("failed to decode SubmitDialogRequest")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		msg, response, err := handler(mux.Vars(r), request)
		if err != nil {
			p.API.LogWarn("failed to handle SubmitDialogRequest", "error", err.Error())
		}

		if msg != nil {
			userLocalizer := p.getUserLocalizer(request.UserId)
			p.SendEphemeralPost(request.ChannelId, request.UserId, p.LocalizeDefaultMessage(userLocalizer, msg))
		}

		if response != nil {
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(response)
			if err != nil {
				p.API.LogWarn("failed to write SubmitDialogRequest", "error", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

func (p *MatterpollPlugin) handleCreatePoll(_ map[string]string, request *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error) {
	publicLocalizer := p.getServerLocalizer()
	creatorID := request.UserId

	question, ok := request.Submission[questionKey].(string)
	if !ok {
		return commandErrorGeneric, nil, errors.Errorf("failed to get question key. Value is: %v", request.Submission[questionKey])
	}

	var answerOptions []string
	o1, ok := request.Submission["option1"].(string)
	if !ok {
		return commandErrorGeneric, nil, errors.Errorf("failed to get option1 key. Value is: %v", request.Submission["option1"])
	}
	answerOptions = append(answerOptions, o1)

	o2, ok := request.Submission["option2"].(string)
	if !ok {
		return commandErrorGeneric, nil, errors.Errorf("failed to get option2 key. Value is: %v", request.Submission["option2"])
	}
	answerOptions = append(answerOptions, o2)

	o3, ok := request.Submission["option3"].(string)
	if ok {
		answerOptions = append(answerOptions, o3)
	}

	var settings []string
	for k, v := range request.Submission {
		if strings.HasPrefix(k, "setting-") {
			b, ok := v.(bool)
			if b && ok {
				settings = append(settings, strings.TrimPrefix(k, "setting-"))
			}
		}
	}

	userLocalizer := p.getUserLocalizer(creatorID)
	poll, errMsg := poll.NewPoll(creatorID, question, answerOptions, settings)
	if errMsg != nil {
		response := &model.SubmitDialogResponse{
			Error: p.LocalizeErrorMessage(userLocalizer, errMsg),
		}
		return nil, response, nil
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(creatorID)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	if err := p.Store.Poll().Insert(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to save poll")
	}

	actions := poll.ToPostActions(publicLocalizer, manifest.ID, displayName)
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: request.ChannelId,
		RootId:    request.CallbackId,
		Type:      MatterpollPostType,
		Props: map[string]interface{}{
			"poll_id": poll.ID,
		},
	}
	model.ParseSlackAttachment(post, actions)

	if _, appErr = p.API.CreatePost(post); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to create poll post")
	}

	return nil, nil, nil
}

func (p *MatterpollPlugin) handleVote(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error) {
	pollID := vars["id"]
	optionNumber, _ := strconv.Atoi(vars["optionNumber"])
	userID := request.UserId

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	prev := poll.Copy()
	hasVoted := poll.HasVoted(userID)
	if err = poll.UpdateVote(userID, optionNumber); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to update poll")
	}

	if err = p.Store.Poll().Update(prev, poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to save poll")
	}

	go p.publishPollMetadata(poll, userID)

	post := &model.Post{}
	publicLocalizer := p.getServerLocalizer()
	model.ParseSlackAttachment(post, poll.ToPostActions(publicLocalizer, manifest.ID, displayName))
	post.AddProp("poll_id", poll.ID)

	if hasVoted {
		return responseVoteUpdated, post, nil
	}
	return responseVoteCounted, post, nil
}

func (p *MatterpollPlugin) publishPollMetadata(poll *poll.Poll, userID string) {
	hasAdminPermission, appErr := p.HasAdminPermission(poll, userID)
	if appErr != nil {
		p.API.LogWarn("Failed to check admin permission", "userID", userID, "pollID", poll.ID, "error", appErr.Error())
		hasAdminPermission = false
	}
	metadata, err := poll.GetMetadata(userID, hasAdminPermission)
	if err != nil {
		p.API.LogWarn("Failed to get poll metadata", "userID", userID, "pollID", poll.ID, "error", appErr.Error())
		return
	}

	p.API.PublishWebSocketEvent("has_voted", metadata.ToMap(), &model.WebsocketBroadcast{UserId: userID})
}

func (p *MatterpollPlugin) handleAddOption(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error) {
	pollID := vars["id"]
	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	if !poll.Settings.PublicAddOption {
		hasAdmminPermission, appErr := p.HasAdminPermission(poll, request.UserId)
		if appErr != nil {
			return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to check permission")
		}
		if !hasAdmminPermission {
			return responseAddOptionInvalidPermission, nil, nil
		}
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/option/add", manifest.ID, pollID),
		Dialog: model.Dialog{
			Title: p.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.addOption.title",
				Other: "Add Option",
			}),
			IconURL:    fmt.Sprintf(responseIconURL, siteURL, manifest.ID),
			CallbackId: request.PostId,
			SubmitLabel: p.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.addOption.submitLabel",
				Other: "Add",
			}),
			Elements: []model.DialogElement{{
				DisplayName: p.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
					ID:    "dialog.addOption.element.displayName",
					Other: "Option",
				}),
				Name:    addOptionKey,
				Type:    "text",
				SubType: "text",
			}},
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to open add option dialog")
	}
	return nil, nil, nil
}

func (p *MatterpollPlugin) handleAddOptionConfirm(vars map[string]string, request *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error) {
	pollID := vars["id"]

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	post, appErr := p.API.GetPost(request.CallbackId)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get post")
	}

	answerOption, ok := request.Submission[addOptionKey].(string)
	if !ok {
		return commandErrorGeneric, nil, errors.Errorf("failed to get submission key: %s", addOptionKey)
	}

	prev := poll.Copy()
	userLocalizer := p.getUserLocalizer(poll.Creator)

	if errMsg := poll.AddAnswerOption(answerOption); errMsg != nil {
		response := &model.SubmitDialogResponse{
			Errors: map[string]string{
				addOptionKey: p.LocalizeErrorMessage(userLocalizer, errMsg),
			},
		}
		return nil, response, nil
	}

	publicLocalizer := p.getServerLocalizer()
	model.ParseSlackAttachment(post, poll.ToPostActions(publicLocalizer, manifest.ID, displayName))
	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to update post")
	}

	if err = p.Store.Poll().Update(prev, poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get save poll")
	}

	return responseAddOptionSuccess, nil, nil
}

func (p *MatterpollPlugin) handleEndPoll(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error) {
	pollID := vars["id"]
	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	hasAdmminPermission, appErr := p.HasAdminPermission(poll, request.UserId)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to check permission")
	}
	if !hasAdmminPermission {
		return responseEndPollInvalidPermission, nil, nil
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/end/confirm", manifest.ID, pollID),
		Dialog: model.Dialog{
			Title: p.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.end.title",
				Other: "Confirm Poll End",
			}),
			IconURL:    fmt.Sprintf(responseIconURL, siteURL, manifest.ID),
			CallbackId: request.PostId,
			SubmitLabel: p.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.end.submitLabel",
				Other: "End",
			}),
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to open end poll dialog")
	}
	return nil, nil, nil
}

func (p *MatterpollPlugin) handleEndPollConfirm(vars map[string]string, request *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error) {
	pollID := vars["id"]

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	post, appErr := poll.ToEndPollPost(p.getServerLocalizer(), displayName, p.ConvertUserIDToDisplayName)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get convert to end poll post")
	}

	post.Id = request.CallbackId
	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to update post")
	}

	if err := p.Store.Poll().Delete(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to delete poll")
	}

	p.postEndPollAnnouncement(request.ChannelId, post.Id, poll.Question)

	return nil, nil, nil
}

func (p *MatterpollPlugin) postEndPollAnnouncement(channelID, postID, question string) {
	endPost := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelID,
		RootId:    postID,
		Message: p.LocalizeWithConfig(p.getServerLocalizer(), &i18n.LocalizeConfig{
			DefaultMessage: responseEndPollSuccessfully,
			TemplateData: map[string]interface{}{
				"Question": question,
				"Link":     fmt.Sprintf("%s/_redirect/pl/%s", *p.ServerConfig.ServiceSettings.SiteURL, postID),
			}}),
		Type: model.POST_DEFAULT,
	}

	if _, err := p.API.CreatePost(endPost); err != nil {
		p.API.LogWarn("Failed to post the end poll announcement", "details", "failed to CreatePost", "error", err.Error())
	}
}

func (p *MatterpollPlugin) handleDeletePoll(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error) {
	pollID := vars["id"]
	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	hasAdmminPermission, appErr := p.HasAdminPermission(poll, request.UserId)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to check permission")
	}
	if !hasAdmminPermission {
		return responseDeletePollInvalidPermission, nil, nil
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/delete/confirm", manifest.ID, pollID),
		Dialog: model.Dialog{
			Title: p.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.delete.title",
				Other: "Confirm Poll Delete",
			}),
			IconURL:    fmt.Sprintf(responseIconURL, siteURL, manifest.ID),
			CallbackId: request.PostId,
			SubmitLabel: p.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.delete.submitLabel",
				Other: "Delete",
			}),
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to open delete poll dialog")
	}

	return nil, nil, nil
}

func (p *MatterpollPlugin) handleDeletePollConfirm(vars map[string]string, request *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error) {
	pollID := vars["id"]

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	if appErr := p.API.DeletePost(request.CallbackId); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to delete post")
	}

	if err := p.Store.Poll().Delete(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to delete poll")
	}

	return responseDeletePollSuccess, nil, nil
}

func (p *MatterpollPlugin) handlePollMetadata(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pollID := vars["id"]
	userID := r.Header.Get("Mattermost-User-Id")

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.API.LogWarn("failed to get poll", "error", err.Error())
		return
	}

	hasAdminPermission, appErr := p.HasAdminPermission(poll, userID)
	if appErr != nil {
		p.API.LogWarn("Failed to check permission", "userID", userID, "error", appErr.Error())
		hasAdminPermission = false
	}
	metadata, err := poll.GetMetadata(userID, hasAdminPermission)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		p.API.LogWarn("Failed to get poll metadata", "userID", userID, "error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		p.API.LogWarn("failed to write response", "error", err.Error())
	}
}
