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
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"

	root "github.com/matterpoll/matterpoll"
	"github.com/matterpoll/matterpoll/server/poll"
)

const (
	iconFilename = "logo_dark-bg.png"

	addOptionKey = "answerOption"
	questionKey  = "question"

	infoMessage = "Thanks for using Matterpoll v"
)

type (
	postActionHandler   func(map[string]string, *model.PostActionIntegrationRequest) (*i18n.LocalizeConfig, *model.Post, error)
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
	pollRouter.HandleFunc("/votes/reset", p.handlePostActionIntegrationRequest(p.handleResetVotes)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/option/add/request", p.handlePostActionIntegrationRequest(p.handleAddOption)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/option/add", p.handleSubmitDialogRequest(p.handleAddOptionConfirm)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/end", p.handlePostActionIntegrationRequest(p.handleEndPoll)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/end/confirm", p.handleSubmitDialogRequest(p.handleEndPollConfirm)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/delete", p.handlePostActionIntegrationRequest(p.handleDeletePoll)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/delete/confirm", p.handleSubmitDialogRequest(p.handleDeletePollConfirm)).Methods(http.MethodPost)
	pollRouter.HandleFunc("/metadata", p.handlePollMetadata).Methods(http.MethodGet)
	return r
}

func (p *MatterpollPlugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)
	p.router.ServeHTTP(w, r)
}

func (p *MatterpollPlugin) handleInfo(w http.ResponseWriter, _ *http.Request) {
	_, _ = io.WriteString(w, infoMessage+root.Manifest.Version+"\n")
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

func (p *MatterpollPlugin) handlePluginConfiguration(w http.ResponseWriter, _ *http.Request) {
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
		var request *model.PostActionIntegrationRequest
		decodeErr := json.NewDecoder(r.Body).Decode(&request)
		if decodeErr != nil || request == nil {
			p.API.LogWarn("failed to decode PostActionIntegrationRequest")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if request.UserId != r.Header.Get("Mattermost-User-ID") {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		pollID := vars["id"]
		poll, err := p.Store.Poll().Get(pollID)
		if err != nil {
			http.Error(w, "failed to get poll", http.StatusInternalServerError)
			return
		}

		var rootID string
		postID := poll.PostID
		if postID != "" {
			post, appEerr := p.API.GetPost(postID)
			if appEerr != nil {
				http.Error(w, "failed to get post", http.StatusInternalServerError)
				return
			}

			if request.ChannelId != post.ChannelId {
				http.Error(w, "not authorized", http.StatusUnauthorized)
				return
			}

			if post.RootId != "" {
				rootID = post.RootId
			} else {
				rootID = post.Id
			}
		}

		if !p.API.HasPermissionToChannel(request.UserId, request.ChannelId, model.PermissionReadChannel) {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		userLocalizer := p.bundle.GetUserLocalizer(request.UserId)

		lc, update, err := handler(mux.Vars(r), request)
		if err != nil {
			p.API.LogWarn("failed to handle PostActionIntegrationRequest", "error", err.Error())
		}

		if lc != nil {
			p.SendEphemeralPost(request.ChannelId, request.UserId, rootID, p.bundle.LocalizeWithConfig(userLocalizer, lc))
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
		var request *model.SubmitDialogRequest
		decodeErr := json.NewDecoder(r.Body).Decode(&request)
		if decodeErr != nil || request == nil {
			p.API.LogWarn("failed to decode SubmitDialogRequest")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if request.UserId != r.Header.Get("Mattermost-User-ID") {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		var rootID string

		vars := mux.Vars(r)
		pollID := vars["id"]
		if pollID != "" {
			poll, err := p.Store.Poll().Get(pollID)
			if err != nil {
				http.Error(w, "failed to get poll", http.StatusInternalServerError)
				return
			}

			postID := poll.PostID
			if postID != "" {
				post, appEerr := p.API.GetPost(postID)
				if appEerr != nil {
					http.Error(w, "failed to get post", http.StatusInternalServerError)
					return
				}

				if request.ChannelId != post.ChannelId {
					http.Error(w, "not authorized", http.StatusUnauthorized)
					return
				}

				if post.RootId != "" {
					rootID = post.RootId
				} else {
					rootID = post.Id
				}
			}
		}

		if !p.API.HasPermissionToChannel(request.UserId, request.ChannelId, model.PermissionReadChannel) {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		msg, response, err := handler(vars, request)
		if err != nil {
			p.API.LogWarn("failed to handle SubmitDialogRequest", "error", err.Error())
		}

		if msg != nil {
			userLocalizer := p.bundle.GetUserLocalizer(request.UserId)
			p.SendEphemeralPost(request.ChannelId, request.UserId, rootID, p.bundle.LocalizeDefaultMessage(userLocalizer, msg))
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
	if ok && o3 != "" {
		answerOptions = append(answerOptions, o3)
	}

	userLocalizer := p.bundle.GetUserLocalizer(creatorID)

	settings := poll.NewSettingsFromSubmission(request.Submission)
	poll, errMsg := poll.NewPoll(creatorID, question, answerOptions, settings)
	if errMsg != nil {
		response := &model.SubmitDialogResponse{
			Error: p.bundle.LocalizeErrorMessage(userLocalizer, errMsg),
		}
		return nil, response, nil
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(creatorID)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	actions := poll.ToPostActions(p.bundle, root.Manifest.Id, displayName)
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
	if poll.Settings.Progress {
		post.AddProp("card", poll.ToCard(p.bundle, p.ConvertUserIDToDisplayName))
	}

	rPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to create poll post")
	}

	poll.PostID = rPost.Id

	if err := p.Store.Poll().Insert(poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to save poll")
	}

	return nil, nil, nil
}

func (p *MatterpollPlugin) handleVote(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.LocalizeConfig, *model.Post, error) {
	pollID := vars["id"]
	optionNumber, _ := strconv.Atoi(vars["optionNumber"])
	userID := request.UserId

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to get poll")
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	prev := poll.Copy()
	previouslyVoted := poll.HasVoted(userID)
	msg, err := poll.UpdateVote(userID, optionNumber)
	if msg != nil {
		return &i18n.LocalizeConfig{DefaultMessage: msg}, nil, nil
	}
	if err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to update poll")
	}

	if err = p.Store.Poll().Update(prev, poll); err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to save poll")
	}

	go p.publishPollMetadata(poll, userID)

	post := &model.Post{}
	model.ParseSlackAttachment(post, poll.ToPostActions(p.bundle, root.Manifest.Id, displayName))
	post.AddProp("poll_id", poll.ID)
	if poll.Settings.Progress {
		post.AddProp("card", poll.ToCard(p.bundle, p.ConvertUserIDToDisplayName))
	}

	// Multi Answer Mode
	if poll.IsMultiVote() {
		var remains int
		votedAnswers := poll.GetVotedAnswers(userID)
		if poll.Settings.MaxVotes == 0 {
			remains = len(poll.AnswerOptions) - len(votedAnswers)
		} else {
			remains = poll.Settings.MaxVotes - len(votedAnswers)
		}
		return &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "response.vote.multi.updated",
				One:   "Your vote has been counted. You have {{.Remains}} vote left.",
				Few:   "Your vote has been counted. You have {{.Remains}} votes left.",
				Many:  "Your vote has been counted. You have {{.Remains}} votes left.",
				Other: "Your vote has been counted. You have {{.Remains}} votes left.",
			},
			TemplateData: map[string]interface{}{"Remains": remains},
			PluralCount:  remains,
		}, post, nil
	}

	// Single Answer Mode
	if previouslyVoted {
		return &i18n.LocalizeConfig{DefaultMessage: responseVoteUpdated}, post, nil
	}
	return &i18n.LocalizeConfig{DefaultMessage: responseVoteCounted}, post, nil
}

func (p *MatterpollPlugin) publishPollMetadata(poll *poll.Poll, userID string) {
	canManagePoll, appErr := p.CanManagePoll(poll, userID)
	if appErr != nil {
		p.API.LogWarn("Failed to check permission to manage poll", "userID", userID, "pollID", poll.ID, "error", appErr.Error())
		canManagePoll = false
	}
	metadata := poll.GetMetadata(userID, canManagePoll)
	p.API.PublishWebSocketEvent("has_voted", metadata.ToMap(), &model.WebsocketBroadcast{UserId: userID})
}

func (p *MatterpollPlugin) handleResetVotes(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.LocalizeConfig, *model.Post, error) {
	pollID := vars["id"]
	userID := request.UserId

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to get poll")
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to get display name for creator")
	}

	votedAnswers := poll.GetVotedAnswers(userID)
	if len(votedAnswers) == 0 {
		return &i18n.LocalizeConfig{DefaultMessage: &i18n.Message{
			ID:    "response.resetVotes.noVotes",
			Other: "There are no votes to reset.",
		}}, nil, nil
	}

	prev := poll.Copy()

	poll.ResetVotes(userID)

	if err = p.Store.Poll().Update(prev, poll); err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to save poll")
	}

	go p.publishPollMetadata(poll, userID)

	post := &model.Post{}
	model.ParseSlackAttachment(post, poll.ToPostActions(p.bundle, root.Manifest.Id, displayName))
	post.AddProp("poll_id", poll.ID)
	if poll.Settings.Progress {
		post.AddProp("card", poll.ToCard(p.bundle, p.ConvertUserIDToDisplayName))
	}

	return &i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "response.resetVotes.success",
			Other: "All votes are cleared. Your previous votes were [{{.ClearedVotes}}].",
		},
		TemplateData: map[string]interface{}{"ClearedVotes": strings.Join(votedAnswers, ", ")},
	}, post, nil
}

func (p *MatterpollPlugin) handleAddOption(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.LocalizeConfig, *model.Post, error) {
	pollID := vars["id"]
	userLocalizer := p.bundle.GetUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to get poll")
	}

	if !poll.Settings.PublicAddOption {
		canManagePoll, appErr := p.CanManagePoll(poll, request.UserId)
		if appErr != nil {
			return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to check permission")
		}
		if !canManagePoll {
			return &i18n.LocalizeConfig{DefaultMessage: responseAddOptionInvalidPermission}, nil, nil
		}
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/option/add", root.Manifest.Id, pollID),
		Dialog: model.Dialog{
			Title: p.bundle.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.addOption.title",
				Other: "Add Option",
			}),
			IconURL:    fmt.Sprintf(responseIconURL, siteURL, root.Manifest.Id),
			CallbackId: request.PostId,
			SubmitLabel: p.bundle.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.addOption.submitLabel",
				Other: "Add",
			}),
			Elements: []model.DialogElement{{
				DisplayName: p.bundle.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
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
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to open add option dialog")
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

	var postID string
	if poll.PostID != "" {
		postID = poll.PostID
	} else {
		// Legacy check if polls created without a postID
		postID = request.CallbackId
	}

	post, appErr := p.API.GetPost(postID)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get post")
	}

	answerOption, ok := request.Submission[addOptionKey].(string)
	if !ok {
		return commandErrorGeneric, nil, errors.Errorf("failed to get submission key: %s", addOptionKey)
	}

	prev := poll.Copy()
	userLocalizer := p.bundle.GetUserLocalizer(poll.Creator)

	if errMsg := poll.AddAnswerOption(answerOption); errMsg != nil {
		response := &model.SubmitDialogResponse{
			Errors: map[string]string{
				addOptionKey: p.bundle.LocalizeErrorMessage(userLocalizer, errMsg),
			},
		}
		return nil, response, nil
	}

	model.ParseSlackAttachment(post, poll.ToPostActions(p.bundle, root.Manifest.Id, displayName))
	if poll.Settings.Progress {
		post.AddProp("card", poll.ToCard(p.bundle, p.ConvertUserIDToDisplayName))
	}

	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to update post")
	}

	if err = p.Store.Poll().Update(prev, poll); err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get save poll")
	}

	return responseAddOptionSuccess, nil, nil
}

func (p *MatterpollPlugin) handleEndPoll(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.LocalizeConfig, *model.Post, error) {
	pollID := vars["id"]
	userLocalizer := p.bundle.GetUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to get poll")
	}

	canManagePoll, appErr := p.CanManagePoll(poll, request.UserId)
	if appErr != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to check permission")
	}
	if !canManagePoll {
		return &i18n.LocalizeConfig{DefaultMessage: responseEndPollInvalidPermission}, nil, nil
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/end/confirm", root.Manifest.Id, pollID),
		Dialog: model.Dialog{
			Title: p.bundle.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.end.title",
				Other: "Confirm Poll End",
			}),
			IconURL:    fmt.Sprintf(responseIconURL, siteURL, root.Manifest.Id),
			CallbackId: request.PostId,
			SubmitLabel: p.bundle.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.end.submitLabel",
				Other: "End",
			}),
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to open end poll dialog")
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

	post, appErr := poll.ToEndPollPost(p.bundle, displayName, p.ConvertUserIDToDisplayName)
	if appErr != nil {
		return commandErrorGeneric, nil, errors.Wrap(appErr, "failed to get convert to end poll post")
	}

	var postID string
	if poll.PostID != "" {
		postID = poll.PostID
	} else {
		// Legacy check if polls created without a postID
		postID = request.CallbackId
	}

	post.Id = postID
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
		Message: p.bundle.LocalizeWithConfig(p.bundle.GetServerLocalizer(), &i18n.LocalizeConfig{
			DefaultMessage: responseEndPollSuccessfully,
			TemplateData: map[string]interface{}{
				"Question": question,
				"Link":     fmt.Sprintf("%s/_redirect/pl/%s", *p.ServerConfig.ServiceSettings.SiteURL, postID),
			}}),
		Type: model.PostTypeDefault,
	}

	if _, err := p.API.CreatePost(endPost); err != nil {
		p.API.LogWarn("Failed to post the end poll announcement", "details", "failed to CreatePost", "error", err.Error())
	}
}

func (p *MatterpollPlugin) handleDeletePoll(vars map[string]string, request *model.PostActionIntegrationRequest) (*i18n.LocalizeConfig, *model.Post, error) {
	pollID := vars["id"]
	userLocalizer := p.bundle.GetUserLocalizer(request.UserId)

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(err, "failed to get poll")
	}

	canManagePoll, appErr := p.CanManagePoll(poll, request.UserId)
	if appErr != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to check permission")
	}
	if !canManagePoll {
		return &i18n.LocalizeConfig{DefaultMessage: responseDeletePollInvalidPermission}, nil, nil
	}

	siteURL := *p.ServerConfig.ServiceSettings.SiteURL
	dialog := model.OpenDialogRequest{
		TriggerId: request.TriggerId,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/delete/confirm", root.Manifest.Id, pollID),
		Dialog: model.Dialog{
			Title: p.bundle.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.delete.title",
				Other: "Confirm Poll Delete",
			}),
			IconURL:    fmt.Sprintf(responseIconURL, siteURL, root.Manifest.Id),
			CallbackId: request.PostId,
			SubmitLabel: p.bundle.LocalizeDefaultMessage(userLocalizer, &i18n.Message{
				ID:    "dialog.delete.submitLabel",
				Other: "Delete",
			}),
		},
	}

	if appErr := p.API.OpenInteractiveDialog(dialog); appErr != nil {
		return &i18n.LocalizeConfig{DefaultMessage: commandErrorGeneric}, nil, errors.Wrap(appErr, "failed to open delete poll dialog")
	}

	return nil, nil, nil
}

func (p *MatterpollPlugin) handleDeletePollConfirm(vars map[string]string, request *model.SubmitDialogRequest) (*i18n.Message, *model.SubmitDialogResponse, error) {
	pollID := vars["id"]

	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		return commandErrorGeneric, nil, errors.Wrap(err, "failed to get poll")
	}

	var postID string
	if poll.PostID != "" {
		postID = poll.PostID
	} else {
		// Legacy check if polls created without a postID
		postID = request.CallbackId
	}

	if appErr := p.API.DeletePost(postID); appErr != nil {
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

	canManagePoll, appErr := p.CanManagePoll(poll, userID)
	if appErr != nil {
		p.API.LogWarn("Failed to check permission", "userID", userID, "error", appErr.Error())
		canManagePoll = false
	}
	metadata := poll.GetMetadata(userID, canManagePoll)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		p.API.LogWarn("failed to write response", "error", err.Error())
	}
}
