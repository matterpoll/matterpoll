package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
)

const (
	infoMessage = "Thanks for using Matterpoll v" + PluginVersion + "\n"

	iconFilename = "logo_dark.png"

	addOptionKey = "answerOption"

	headerMattermostUserId = "Mattermost-User-Id"
)

type (
	postActionHandler   func(map[string]string, *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error)
	submitDialogHandler func(map[string]string, *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error)
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
	apiV1.HandleFunc("/configuration", p.handlePluginConfiguration).Methods("GET")

	pollRouter := apiV1.PathPrefix("/polls/{id:[a-z0-9]+}").Subrouter()
	pollRouter.HandleFunc("/vote/{optionNumber:[0-9]+}", p.handlePostActionIntegrationRequest(p.handleVote)).Methods("POST")
	pollRouter.HandleFunc("/option/add", p.handleSubmitDialogRequest(p.handleAddOption)).Methods("POST")
	pollRouter.HandleFunc("/option/add/request", p.handlePostActionIntegrationRequest(p.handleAddOptionDialogRequest)).Methods("POST")
	pollRouter.HandleFunc("/end", p.handlePostActionIntegrationRequest(p.handleEndPoll)).Methods("POST")
	pollRouter.HandleFunc("/delete", p.handlePostActionIntegrationRequest(p.handleDeletePoll)).Methods("POST")

	pollRouter.HandleFunc("/voted", p.handleUserVoted).Methods("GET")
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
	b, err := json.Marshal(configuration)
	if err != nil {
		p.API.LogWarn("failed to decode configuration object.", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(b); err != nil {
		p.API.LogWarn("failed to write response.", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *MatterpollPlugin) handlePostActionIntegrationRequest(handler postActionHandler) func(http.ResponseWriter, *http.Request) {
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

		response := &model.PostActionIntegrationResponse{}
		if msg != nil {
			response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, msg)
		}
		if update != nil {
			response.Update = update
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err = w.Write(response.ToJson()); err != nil {
			p.API.LogWarn("failed to write PostActionIntegrationResponse", "error", err.Error())
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (p *MatterpollPlugin) handleSubmitDialogRequest(handler submitDialogHandler) func(http.ResponseWriter, *http.Request) {
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
			if _, err = w.Write(response.ToJson()); err != nil {
				p.API.LogWarn("failed to write SubmitDialogRequest", "error", err.Error())
			}
		}
		w.WriteHeader(http.StatusOK)
	}
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

	hasVoted := poll.HasVoted(userID)
	if err = poll.UpdateVote(userID, optionNumber); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to update poll")
	}

	if err = p.Store.Poll().Save(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to save poll")
	}

	v, err := poll.GetVotedAnswer(userID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get voted answers")
	}
	p.API.PublishWebSocketEvent("has_voted", map[string]interface{}{
		"user_id":       v.UserID,
		"poll_id":       v.PollID,
		"voted_answers": v.VotedAnswers,
	}, &model.WebsocketBroadcast{UserId: userID})

	post := &model.Post{}
	publicLocalizer := p.getServerLocalizer()
	model.ParseSlackAttachment(post, poll.ToPostActions(publicLocalizer, *p.ServerConfig.ServiceSettings.SiteURL, PluginId, displayName))
	post.AddProp("poll_id", poll.ID)
	if hasVoted {
		return responseVoteUpdated, post, nil
	}
	return responseVoteCounted, post, nil
}

func (p *MatterpollPlugin) handleAddOption(vars map[string]string, request *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error) {
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
		return commandErrorGeneric, nil, errors.Wrapf(appErr, "failed to get submission key %s", addOptionKey)
	}

	if err = poll.AddAnswerOption(answerOption); err != nil {
		response := &model.SubmitDialogResponse{
			Errors: map[string]string{
				addOptionKey: err.Error(),
			},
		}
		return nil, response, nil
	}

	publicLocalizer := p.getServerLocalizer()
	model.ParseSlackAttachment(post, poll.ToPostActions(publicLocalizer, *p.ServerConfig.ServiceSettings.SiteURL, PluginId, displayName))
	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to update post")
	}

	if err = p.Store.Poll().Save(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get save poll")

	}

	return responseAddOptionSuccess, nil, nil
}

func (p *MatterpollPlugin) handleAddOptionDialogRequest(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error) {
	pollID := vars["id"]
	userLocalizer := p.getUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	if !poll.Settings.PublicAddOption {
		hasPermission, appErr := p.HasPermission(poll, request.UserId)
		if appErr != nil {
			return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to check permission")
		}
		if !hasPermission {
			return responseAddOptionInvalidPermission, nil, nil
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
			}},
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to open add option dialog")
	}
	return nil, nil, nil
}

// TODO: will confirm this func finally
func (p *MatterpollPlugin) handleUserVoted(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pollID := vars["id"]

	userID := r.Header.Get(headerMattermostUserId)

	response := &model.PostActionIntegrationResponse{}
	userLocalizer := p.getUserLocalizer(userID)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		response.EphemeralText = p.LocalizeDefaultMessage(userLocalizer, commandErrorGeneric)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(response.ToJson()); err != nil {
			p.API.LogWarn("failed to write response", "error", err.Error())
		}
		return
	}

	v, err := poll.GetVotedAnswer(userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		p.API.LogError(fmt.Sprintf("Failed to get voted answers with UserId: %s", userID))
		return
	}

	b, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.API.LogError(fmt.Sprintf("Failed to marshal voted answers: %v", v))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		p.API.LogWarn("failed to write response", "error", err.Error())
	}
}

func (p *MatterpollPlugin) handleEndPoll(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error) {
	pollID := vars["id"]

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	hasPermission, appErr := p.HasPermission(poll, request.UserId)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to check permission")
	}
	if !hasPermission {
		return responseEndPollInvalidPermission, nil, nil
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	post, appErr := poll.ToEndPollPost(p.getServerLocalizer(), displayName, p.ConvertUserIDToDisplayName)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get convert to end poll post")
	}

	if err := p.Store.Poll().Delete(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to delete poll")
	}

	p.postEndPollAnnouncement(request, poll.Question)
	return nil, post, nil
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
		UserId:    p.botUserID,
		ChannelId: channelID,
		RootId:    request.PostId,
		Message: p.LocalizeWithConfig(publicLocalizer, &i18n.LocalizeConfig{
			DefaultMessage: responseEndPollSuccessfully,
			TemplateData: map[string]interface{}{
				"Question": question,
				"Link":     link,
			}}),
		Type: model.POST_DEFAULT,
	}

	if _, err = p.API.CreatePost(endPost); err != nil {
		p.API.LogError(endPollAnnouncementPostError, "details", "failed to CreatePost")
	}
}

func (p *MatterpollPlugin) handleDeletePoll(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.Message, *model.Post, error) {
	pollID := vars["id"]

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	hasPermission, appErr := p.HasPermission(poll, request.UserId)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to check permission")
	}
	if !hasPermission {
		return responseDeletePollInvalidPermission, nil, nil
	}

	appErr = p.API.DeletePost(request.PostId)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to delete post")
	}

	if err := p.Store.Poll().Delete(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to delete poll")
	}

	return responseDeletePollSuccess, nil, nil
}
