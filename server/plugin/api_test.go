package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/undefinedlabs/go-mpatch"

	root "github.com/matterpoll/matterpoll"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/store/mockstore"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestServeHTTP(t *testing.T) {
	for name, test := range map[string]struct {
		RequestURL         string
		ExpectedStatusCode int
		ExpectedHeader     http.Header
		ExpectedbodyString string
	}{
		"Request info": {
			RequestURL:         "/",
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
			ExpectedbodyString: infoMessage + root.Manifest.Version + "\n",
		},
		"InvalidRequestURL": {
			RequestURL:         "/not_found",
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
			ExpectedbodyString: "404 page not found\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := &plugintest.API{}
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, &mockstore.Store{})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			bodyBytes, err := io.ReadAll(result.Body)
			require.Nil(t, err)
			bodyString := string(bodyBytes)

			assert.Equal(test.ExpectedbodyString, bodyString)
			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestServeFile(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ShouldError        bool
	}{
		"all fine": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				path, err := filepath.Abs("../..")
				require.Nil(t, err)
				api.On("GetBundlePath").Return(path, nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ShouldError:        false,
		},
		"failed to get executable": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return("", errors.New(""))
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ShouldError:        true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, &mockstore.Store{})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", iconFilename), nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			bodyBytes, err := io.ReadAll(result.Body)
			require.Nil(t, err)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if test.ShouldError {
				assert.Equal([]byte{}, bodyBytes)
				assert.Equal(http.Header{}, result.Header)
			} else {
				assert.NotNil(bodyBytes)
				assert.Contains([]string{"image/png"}, result.Header.Get("Content-Type"))
			}
		})
	}
}

func TestHandlePluginConfiguration(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ShouldError        bool
	}{
		"all fine": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			ExpectedStatusCode: http.StatusOK,
			ShouldError:        false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, &mockstore.Store{})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/configuration", nil)
			r.Header.Add("Mattermost-User-ID", model.NewId())
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			bodyBytes, err := io.ReadAll(result.Body)
			require.Nil(t, err)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if test.ShouldError {
				assert.Equal([]byte{}, bodyBytes)
				assert.Equal(http.Header{}, result.Header)
			} else {
				assert.NotNil(bodyBytes)
				assert.Contains([]string{"application/json"}, result.Header.Get("Content-Type"))
			}
		})
	}
}

func TestHandleCreatePoll(t *testing.T) {
	converter := func(userID string) (string, *model.AppError) {
		switch userID {
		case "userID1":
			return "@jhDoe", nil
		default:
			return "", &model.AppError{}
		}
	}

	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.PostActionIntegrationRequest{UserId: "userID1", TeamId: "teamID1"}

		w := httptest.NewRecorder()
		url := "/api/v1/polls/create"
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)
		result := w.Result()
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	expectedPoll := testutils.GetPoll()
	userID := expectedPoll.Creator
	channelID := model.NewId()
	rootID := model.NewId()
	expectedPost := &model.Post{
		UserId:    testutils.GetBotUserID(),
		ChannelId: channelID,
		RootId:    rootID,
		Type:      MatterpollPostType,
		Props: model.StringInterface{
			"poll_id": testutils.GetPollID(),
		},
	}
	model.ParseSlackAttachment(expectedPost, expectedPoll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	pollWithTwoOptions := testutils.GetPoll()
	pollWithTwoOptions.AnswerOptions = pollWithTwoOptions.AnswerOptions[0:2]
	expectedPostTwoOptions := &model.Post{
		UserId:    testutils.GetBotUserID(),
		ChannelId: channelID,
		RootId:    rootID,
		Type:      MatterpollPostType,
		Props: model.StringInterface{
			"poll_id": testutils.GetPollID(),
		},
	}
	model.ParseSlackAttachment(expectedPostTwoOptions, pollWithTwoOptions.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	pollWithSettings := testutils.GetPollWithSettings(poll.Settings{Progress: true, Anonymous: true, PublicAddOption: true, MaxVotes: 3})
	expectedPostWithSettings := &model.Post{
		UserId:    testutils.GetBotUserID(),
		ChannelId: channelID,
		RootId:    rootID,
		Type:      MatterpollPostType,
		Props: model.StringInterface{
			"poll_id": testutils.GetPollID(),
		},
	}
	expectedPostWithSettings.AddProp("card", pollWithSettings.ToCard(testutils.GetBundle(), converter))
	model.ParseSlackAttachment(expectedPostWithSettings, pollWithSettings.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.SubmitDialogRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.SubmitDialogResponse
		ExpectedMsg        string
	}{
		"Valid request, two options": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)

				rPost := expectedPostTwoOptions.Clone()
				rPost.Id = "postID1"
				api.On("CreatePost", expectedPostTwoOptions).Return(rPost, nil)

				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Insert", pollWithTwoOptions).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": pollWithTwoOptions.Question,
					"option1":  pollWithTwoOptions.AnswerOptions[0].Answer,
					"option2":  pollWithTwoOptions.AnswerOptions[1].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
		"Valid request, three options": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)

				rPost := expectedPost.Clone()
				rPost.Id = "postID1"
				api.On("CreatePost", expectedPost).Return(rPost, nil)

				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Insert", expectedPoll).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": expectedPoll.Question,
					"option1":  expectedPoll.AnswerOptions[0].Answer,
					"option2":  expectedPoll.AnswerOptions[1].Answer,
					"option3":  expectedPoll.AnswerOptions[2].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
		"Valid request with settings": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe", Username: "jhDoe"}, nil)

				rPost := expectedPostWithSettings.Clone()
				rPost.Id = "postID1"
				api.On("CreatePost", expectedPostWithSettings).Return(rPost, nil)

				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Insert", pollWithSettings).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question":                  pollWithSettings.Question,
					"option1":                   pollWithSettings.AnswerOptions[0].Answer,
					"option2":                   pollWithSettings.AnswerOptions[1].Answer,
					"option3":                   pollWithSettings.AnswerOptions[2].Answer,
					"setting-multi":             3,
					"setting-anonymous":         true,
					"setting-progress":          true,
					"setting-public-add-option": true,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
		"Invalid request, question not set": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store { return store },
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"option1": expectedPoll.AnswerOptions[0].Answer,
					"option2": expectedPoll.AnswerOptions[1].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, option 1 not set": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store { return store },
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": expectedPoll.Question,
					"option2":  expectedPoll.AnswerOptions[1].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, option 2 not set": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store { return store },
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": expectedPoll.Question,
					"option1":  expectedPoll.AnswerOptions[0].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, duplicate option": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store { return store },
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": expectedPoll.Question,
					"option1":  "abc",
					"option2":  "abc",
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse: &model.SubmitDialogResponse{
				Error: "Duplicate option: abc",
			},
			ExpectedMsg: "",
		},
		"Valid request, GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store { return store },
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": pollWithTwoOptions.Question,
					"option1":  pollWithTwoOptions.AnswerOptions[0].Answer,
					"option2":  pollWithTwoOptions.AnswerOptions[1].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, PollStore.Save fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)

				rPost := expectedPostTwoOptions.Clone()
				rPost.Id = "postID1"
				api.On("CreatePost", expectedPostTwoOptions).Return(rPost, nil)

				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Insert", pollWithTwoOptions).Return(errors.New(""))
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": pollWithTwoOptions.Question,
					"option1":  pollWithTwoOptions.AnswerOptions[0].Answer,
					"option2":  pollWithTwoOptions.AnswerOptions[1].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, createPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("CreatePost", expectedPostTwoOptions).Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store { return store },
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"question": pollWithTwoOptions.Question,
					"option1":  pollWithTwoOptions.AnswerOptions[0].Answer,
					"option2":  pollWithTwoOptions.AnswerOptions[1].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store { return store },
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: rootID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"option1": expectedPoll.AnswerOptions[0].Answer,
					"option2": expectedPoll.AnswerOptions[1].Answer,
				},
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Empty request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store)

			patch1, _ := mpatch.PatchMethod(model.GetMillis, func() int64 { return 1234567890 })
			patch2, _ := mpatch.PatchMethod(model.NewId, testutils.GetPollID)
			defer func() { require.NoError(t, patch1.Unpatch()) }()
			defer func() { require.NoError(t, patch2.Unpatch()) }()

			w := httptest.NewRecorder()
			url := "/api/v1/polls/create"
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.SubmitDialogResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			assert.Equal(test.ExpectedResponse, response)

			if test.ExpectedResponse != nil {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
			}
		})
	}
}

func TestHandleVote(t *testing.T) {
	converter := func(userID string) (string, *model.AppError) {
		switch userID {
		case "userID1":
			return "@jhDoe", nil
		case "userID2":
			return "@jhDoe2", nil
		default:
			return "", &model.AppError{}
		}
	}

	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.PostActionIntegrationRequest{UserId: "userID1", TeamId: "teamID1"}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/vote/0", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)

		result := w.Result()
		require.NotNil(t, result)
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	poll1In := testutils.GetPoll()
	poll1Out := poll1In.Copy()
	msg, err := poll1Out.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	expectedPost1 := &model.Post{}
	model.ParseSlackAttachment(expectedPost1, poll1Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll2In := testutils.GetPoll()
	msg, err = poll2In.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	poll2Out := poll2In.Copy()
	msg, err = poll2Out.UpdateVote("userID1", 1)
	require.Nil(t, msg)
	require.Nil(t, err)
	expectedPost2 := &model.Post{}
	model.ParseSlackAttachment(expectedPost2, poll2Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll3In := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 2})
	poll3Out := poll3In.Copy()
	msg, err = poll3Out.UpdateVote("userID2", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	expectedPost3 := &model.Post{}
	model.ParseSlackAttachment(expectedPost3, poll3Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll4In := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 2})
	msg, err = poll4In.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	poll4Out := poll4In.Copy()
	msg, err = poll4Out.UpdateVote("userID1", 1)
	require.Nil(t, msg)
	require.Nil(t, err)
	expectedPost4 := &model.Post{}
	model.ParseSlackAttachment(expectedPost4, poll4Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll5In := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 2})
	msg, err = poll5In.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	msg, err = poll5In.UpdateVote("userID1", 1)
	require.Nil(t, msg)
	require.Nil(t, err)

	poll6In := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 2})
	poll6Out := poll6In.Copy()
	msg, err = poll6Out.UpdateVote("userID2", 1)
	require.Nil(t, msg)
	require.Nil(t, err)
	expectedPost6 := &model.Post{}
	model.ParseSlackAttachment(expectedPost6, poll6Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll7In := testutils.GetPollWithSettings(poll.Settings{Progress: true, MaxVotes: 1})
	msg, err = poll7In.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	poll7Out := poll7In.Copy()
	msg, err = poll7Out.UpdateVote("userID1", 1)
	require.Nil(t, msg)
	require.Nil(t, err)
	expectedPost7 := &model.Post{}
	expectedPost7.AddProp("card", poll7Out.ToCard(testutils.GetBundle(), converter))
	model.ParseSlackAttachment(expectedPost7, poll7Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll8In := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 0})
	poll8Out := poll8In.Copy()
	msg, err = poll8Out.UpdateVote("userID2", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	expectedPost8 := &model.Post{}
	model.ParseSlackAttachment(expectedPost8, poll8Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	post := &model.Post{
		ChannelId: "channelID1",
	}

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		VoteIndex          int
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
		ExpectedMsg        string
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"voted_answers":             []string{"Answer 1"},
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"can_manage_poll":           true,
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll1In.Copy(), nil)
				store.PollStore.On("Update", poll1In, poll1Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost1},
			ExpectedMsg:        "Your vote has been counted.",
		},
		"Valid request with no votes, poll without postID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"voted_answers":             []string{"Answer 1"},
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"can_manage_poll":           true,
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				pollIn := poll1In.Copy()
				pollIn.PostID = ""
				store.PollStore.On("Get", testutils.GetPollID()).Return(pollIn.Copy(), nil)
				pollOut := poll1Out.Copy()
				pollOut.PostID = ""
				store.PollStore.On("Update", pollIn, pollOut).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost1},
			ExpectedMsg:        "Your vote has been counted.",
		},
		"Valid request, with multi setting, first vote": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"can_manage_poll":           false,
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID2",
					"voted_answers":             []string{"Answer 1"},
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID2"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll3In.Copy(), nil)
				store.PollStore.On("Update", poll3In, poll3Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost3},
			ExpectedMsg:        "Your vote has been counted. You have 1 vote left.",
		},
		"Valid request, with multi setting, second vote": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"can_manage_poll":           true,
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"voted_answers":             []string{"Answer 1", "Answer 2"},
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll4In.Copy(), nil)
				store.PollStore.On("Update", poll4In, poll4Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost4},
			ExpectedMsg:        "Your vote has been counted. You have 0 votes left.",
		},
		"Valid request, with multi setting, over the max": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll5In.Copy(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          2,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "You could't vote for this option, because you don't have any votes left. Use the reset button to reset your votes.",
		},
		"Valid request, with multi setting (--votes=0), first vote": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"can_manage_poll":           false,
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID2",
					"voted_answers":             []string{"Answer 1"},
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID2"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll8In.Copy(), nil)
				store.PollStore.On("Update", poll8In, poll8Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost8},
			ExpectedMsg:        "Your vote has been counted. You have 2 votes left.",
		},
		"Valid request with vote": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"voted_answers":             []string{"Answer 2"},
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"can_manage_poll":           true,
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll2In.Copy(), nil)
				store.PollStore.On("Update", poll2In, poll2Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost2},
			ExpectedMsg:        "Your vote has been updated.",
		},
		"Valid request with vote, with Progress setting true": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe", Username: "jhDoe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"voted_answers":             []string{"Answer 2"},
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"can_manage_poll":           true,
					"setting_progress":          true,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll7In.Copy(), nil)
				store.PollStore.On("Update", poll7In, poll7Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost7},
			ExpectedMsg:        "Your vote has been updated.",
		},
		"Valid request, PollStore.Save fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				pollIn := testutils.GetPoll()
				pollOut := pollIn.Copy()
				msg, err := pollOut.UpdateVote("userID1", 0)
				require.Nil(t, msg)
				require.Nil(t, err)

				store.PollStore.On("Get", testutils.GetPollID()).Return(pollIn.Copy(), nil)
				store.PollStore.On("Update", pollIn, pollOut).Return(&model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request with vote, CanManagePoll fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 7)...).Return().Maybe()
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"voted_answers":             []string{"Answer 2"},
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID2",
					"can_manage_poll":           false,
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID2"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll6In.Copy(), nil)
				store.PollStore.On("Update", poll6In, poll6Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost6},
			ExpectedMsg:        "Your vote has been counted. You have 1 vote left.",
		},
		"Invalid index": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          3,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll1In.Copy(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, post with invalid channelID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				p := post.Clone()
				p.ChannelId = "channelID2"
				api.On("GetPost", "postID1").Return(p, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll1In.Copy(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					RootId:    post.Id,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)

			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)

			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/vote/%d", testutils.GetPollID(), test.VoteIndex)
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.PostActionIntegrationResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				require.NotNil(t, response)
				assert.Equal(test.ExpectedResponse.EphemeralText, response.EphemeralText)
				if test.ExpectedResponse.Update != nil {
					assert.Equal(test.ExpectedResponse.Update.Props["card"], response.Update.Props["card"])
					assert.Equal(test.ExpectedResponse.Update.Attachments(), response.Update.Attachments())
				}
			} else {
				assert.Equal(test.ExpectedResponse, response)
			}
		})
	}
}

func TestHandleResetVotes(t *testing.T) {
	converter := func(userID string) (string, *model.AppError) {
		switch userID {
		case "userID1":
			return "@jhDoe", nil
		case "userID2":
			return "@jhDoe2", nil
		default:
			return "", &model.AppError{}
		}
	}

	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.PostActionIntegrationRequest{UserId: "userID1", TeamId: "teamID1"}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/votes/reset", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)

		result := w.Result()
		require.NotNil(t, result)
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	pollEmptyWithProgress := &poll.Poll{
		ID:      testutils.GetPollID(),
		Creator: "userID1",
		AnswerOptions: []*poll.AnswerOption{
			{Answer: "Answer 1", Voter: []string{}},
			{Answer: "Answer 2", Voter: []string{}},
			{Answer: "Answer 3", Voter: []string{}},
		},
		Settings: poll.Settings{Progress: true, MaxVotes: 3},
	}

	poll2WithVotesWithProgress := pollEmptyWithProgress.Copy()
	msg, err := poll2WithVotesWithProgress.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)

	poll := &poll.Poll{
		ID:      testutils.GetPollID(),
		Creator: "userID1",
		AnswerOptions: []*poll.AnswerOption{
			{Answer: "Answer 1", Voter: []string{}},
			{Answer: "Answer 2", Voter: []string{}},
			{Answer: "Answer 3", Voter: []string{}},
		},
		Settings: poll.Settings{MaxVotes: 3},
	}

	expectedPost := &model.Post{}
	model.ParseSlackAttachment(expectedPost, poll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	expectedPostWithProgress := &model.Post{}
	expectedPostWithProgress.AddProp("card", pollEmptyWithProgress.ToCard(testutils.GetBundle(), converter))
	model.ParseSlackAttachment(expectedPostWithProgress, pollEmptyWithProgress.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll2WithVotes := poll.Copy()
	msg, err = poll2WithVotes.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)

	poll3WithVotes := poll.Copy()
	msg, err = poll3WithVotes.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)
	msg, err = poll3WithVotes.UpdateVote("userID1", 1)
	require.Nil(t, msg)
	require.Nil(t, err)
	msg, err = poll3WithVotes.UpdateVote("userID1", 2)
	require.Nil(t, msg)
	require.Nil(t, err)

	poll4WithVotes := poll.Copy()
	msg, err = poll4WithVotes.UpdateVote("userID1", 0)
	require.Nil(t, msg)
	require.Nil(t, err)

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
		ExpectedMsg        string
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll.Copy(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "There are no votes to reset.",
		},
		"Valid request, reset 1 vote": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"can_manage_poll":           true,
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"voted_answers":             []string{},
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll2WithVotes.Copy(), nil)
				store.PollStore.On("Update", poll2WithVotes, poll).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost},
			ExpectedMsg:        "All votes are cleared. Your previous votes were [Answer 1].",
		},
		"Valid request, reset 1 vote, with Settings Progress to true": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe", Username: "jhDoe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"can_manage_poll":           true,
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"voted_answers":             []string{},
					"setting_progress":          true,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll2WithVotesWithProgress.Copy(), nil)
				store.PollStore.On("Update", poll2WithVotesWithProgress, pollEmptyWithProgress).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPostWithProgress},
			ExpectedMsg:        "All votes are cleared. Your previous votes were [Answer 1].",
		},
		"Valid request, reset multi votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("PublishWebSocketEvent", "has_voted", map[string]interface{}{
					"can_manage_poll":           true,
					"poll_id":                   testutils.GetPollID(),
					"user_id":                   "userID1",
					"voted_answers":             []string{},
					"setting_progress":          false,
					"setting_public_add_option": false,
				}, &model.WebsocketBroadcast{UserId: "userID1"}).Return()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll3WithVotes.Copy(), nil)
				store.PollStore.On("Update", poll3WithVotes, poll).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost},
			ExpectedMsg:        "All votes are cleared. Your previous votes were [Answer 1, Answer 2, Answer 3].",
		},
		"Failed to get poll": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
		"Failed to get display name": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll.Copy(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"invalid user id": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll.Copy(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"failed to save poll": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll4WithVotes.Copy(), nil)
				store.PollStore.On("Update", poll4WithVotes, poll).Return(&model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)

			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)

			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/votes/reset", testutils.GetPollID())
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.PostActionIntegrationResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				require.NotNil(t, response)
				assert.Equal(test.ExpectedResponse.EphemeralText, response.EphemeralText)
				if test.ExpectedResponse.Update != nil {
					assert.Equal(test.ExpectedResponse.Update.Props["card"], response.Update.Props["card"])
					assert.Equal(test.ExpectedResponse.Update.Attachments(), response.Update.Attachments())
				}
			} else {
				assert.Equal(test.ExpectedResponse, response)
			}
		})
	}
}
func TestHandleAddOption(t *testing.T) {
	userID := testutils.GetPollWithVotes().Creator
	triggerID := model.NewId()

	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.PostActionIntegrationRequest{UserId: userID, PostId: "postID1", TriggerId: triggerID}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/option/add/request", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)

		result := w.Result()
		require.NotNil(t, result)
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	dialogRequest := model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/option/add", root.Manifest.Id, testutils.GetPollID()),
		Dialog: model.Dialog{
			Title:       "Add Option",
			IconURL:     fmt.Sprintf(responseIconURL, testutils.GetSiteURL(), root.Manifest.Id),
			CallbackId:  "postID1",
			SubmitLabel: "Add",
			Elements: []model.DialogElement{{
				DisplayName: "Option",
				Name:        "answerOption",
				Type:        "text",
				SubType:     "text",
			},
			},
		},
	}
	post := &model.Post{
		ChannelId: "channelID1",
	}

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedMsg        string
	}{
		"Valid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", userID, "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("OpenInteractiveDialog", dialogRequest).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request, poll without postID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("OpenInteractiveDialog", dialogRequest).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithoutPostID(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request, issuer is system admin": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(&model.User{
					Username: "user2",
					Roles:    model.SystemAdminRoleId + " " + model.SystemUserRoleId,
				}, nil)
				api.On("OpenInteractiveDialog", dialogRequest).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request, Invalid permission": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Only the creator of a poll and System Admins are allowed to add options.",
		},
		"Valid request, GetUser fails for issuer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, OpenInteractiveDialog fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", userID, "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("OpenInteractiveDialog", dialogRequest).Return(&model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, errors.New(""))
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, post with invalid channelID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				p := post.Clone()
				p.ChannelId = "channelID2"
				api.On("GetPost", "postID1").Return(p, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", userID, "channelID1", model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Empty request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)

			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)

			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/option/add/request", testutils.GetPollID())
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.PostActionIntegrationResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				assert.Equal(response, &model.PostActionIntegrationResponse{})
			} else {
				assert.Nil(response)
			}
		})
	}
}

func TestHandleAddOptionConfirm(t *testing.T) {
	converter := func(userID string) (string, *model.AppError) {
		switch userID {
		case "userID1":
			return "@jhDoe", nil
		case "userID2":
			return "@jhDoe2", nil
		case "userID3":
			return "@jhDoe3", nil
		case "userID4":
			return "@jhDoe4", nil
		default:
			return "", &model.AppError{}
		}
	}

	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.PostActionIntegrationRequest{UserId: "userID1", TeamId: "teamID1"}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/option/add", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)
		result := w.Result()
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	userID := testutils.GetPollWithVotes().Creator
	channelID := model.NewId()
	postID := model.NewId()

	poll1In := testutils.GetPollWithVotes()
	poll1In.PostID = postID
	poll1Out := poll1In.Copy()
	err := poll1Out.AddAnswerOption("New Option")
	require.Nil(t, err)
	expectedPost1 := &model.Post{
		ChannelId: channelID,
	}
	model.ParseSlackAttachment(expectedPost1, poll1Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll2In := testutils.GetPollWithoutPostID()
	poll2Out := poll2In.Copy()
	err = poll2Out.AddAnswerOption("New Option")
	require.Nil(t, err)
	expectedPost2 := &model.Post{}
	model.ParseSlackAttachment(expectedPost2, poll2Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	poll3In := testutils.GetPollWithVotesAndSettings(poll.Settings{Progress: true, MaxVotes: 2})
	poll3In.PostID = postID
	poll3Out := poll3In.Copy()
	err = poll3Out.AddAnswerOption("New Option")
	require.Nil(t, err)
	expectedPost3 := &model.Post{
		ChannelId: channelID,
	}
	expectedPost3.AddProp("card", poll3Out.ToCard(testutils.GetBundle(), converter))
	model.ParseSlackAttachment(expectedPost3, poll3Out.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe"))

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.SubmitDialogRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.SubmitDialogResponse
		ExpectedMsg        string
	}{
		"Valid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost1, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("UpdatePost", expectedPost1).Return(expectedPost1, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll1In.Copy(), nil)
				store.PollStore.On("Update", poll1In, poll1Out).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Successfully added the option.",
		},
		"Valid request, with Progress settings": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost3, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe", Username: "jhDoe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "jhDoe2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "jhDoe3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "jhDoe4"}, nil)
				api.On("UpdatePost", expectedPost3).Return(expectedPost3, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll3In.Copy(), nil)
				store.PollStore.On("Update", poll3In, poll3Out).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption":      "New Option",
					"settings-progress": true,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Successfully added the option.",
		},
		"Valid request, poll without postID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetPost", postID).Return(&model.Post{}, nil)
				api.On("UpdatePost", expectedPost2).Return(expectedPost2, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll2In.Copy(), nil)
				store.PollStore.On("Update", poll2In, poll2Out).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Successfully added the option.",
		},
		"Valid request, GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost1, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithVotes().Copy()
				poll.PostID = postID
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll, nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request with integer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost1, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetPost", postID).Return(&model.Post{}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithVotes().Copy()
				poll.PostID = postID
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll, nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": 1,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, duplicate new answeroption": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost1, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetPost", postID).Return(&model.Post{}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithVotes().Copy()
				poll.PostID = postID
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll, nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": poll1In.AnswerOptions[0].Answer,
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse: &model.SubmitDialogResponse{
				Errors: map[string]string{
					"answerOption": "Duplicate option: Answer 1",
				},
			},
			ExpectedMsg: "",
		},
		"Valid request, UpdatePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost1, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetPost", postID).Return(&model.Post{}, nil)
				api.On("UpdatePost", expectedPost1).Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithVotes().Copy()
				poll.PostID = postID
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll, nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, PollStore.Save fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost1, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(true)
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetPost", postID).Return(&model.Post{}, nil)
				api.On("UpdatePost", expectedPost1).Return(expectedPost1, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll1In.Copy(), nil)
				store.PollStore.On("Update", poll1In, poll1Out).Return(errors.New(""))
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, errors.New(""))
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithVotes().Copy()
				poll.PostID = postID
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll, nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, post with invalid channelId": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				p := expectedPost1.Clone()
				p.ChannelId = "channelID2"
				api.On("GetPost", postID).Return(p, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithVotes().Copy()
				poll.PostID = postID
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll, nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", postID).Return(expectedPost1, nil)
				api.On("HasPermissionToChannel", userID, channelID, model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll1In.Copy(), nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					"answerOption": "New Option",
				},
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Empty request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/option/add", testutils.GetPollID())
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.SubmitDialogResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			assert.Equal(test.ExpectedResponse, response)
			if test.ExpectedResponse != nil {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
			}
		})
	}
}

func TestHandleEndPoll(t *testing.T) {
	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.PostActionIntegrationRequest{UserId: "userID1", ChannelId: "channelID1", PostId: "postID1"}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/end", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)

		result := w.Result()
		require.NotNil(t, result)
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	triggerID := model.NewId()
	dialog := model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/end/confirm", root.Manifest.Id, testutils.GetPollID()),
		Dialog: model.Dialog{
			Title:       "Confirm Poll End",
			IconURL:     fmt.Sprintf(responseIconURL, testutils.GetSiteURL(), root.Manifest.Id),
			CallbackId:  "postID1",
			SubmitLabel: "End",
		},
	}

	post := &model.Post{
		ChannelId: "channelID1",
	}

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedMsg        string
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request, poll without postID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithoutPostID(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request with no votes, issuer is system admin": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(&model.User{
					Username: "user2",
					Roles:    model.SystemAdminRoleId + " " + model.SystemUserRoleId,
				}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request, GetUser fails for issuer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, Invalid permission": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2", Roles: model.SystemUserRoleId}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Only the creator of a poll and System Admins are allowed to end it.",
		},
		"Valid request, OpenInteractiveDialog fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(&model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, Store.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, post with invalid channelID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				p := post.Clone()
				p.ChannelId = "channelID2"
				api.On("GetPost", "postID1").Return(p, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)

			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)

			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/end", testutils.GetPollID())
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.PostActionIntegrationResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				assert.Equal(response, &model.PostActionIntegrationResponse{})
			} else {
				assert.Nil(response)
			}
		})
	}
}

func TestHandleEndPollConfirm(t *testing.T) {
	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.SubmitDialogRequest{}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/end/confirm", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)

		result := w.Result()
		require.NotNil(t, result)
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	converter := func(userID string) (string, *model.AppError) {
		switch userID {
		case "userID1":
			return "@user1", nil
		case "userID2":
			return "@user2", nil
		case "userID3":
			return "@user3", nil
		case "userID4":
			return "@user4", nil
		default:
			return "", &model.AppError{}
		}
	}
	expectedPost, err := testutils.GetPollWithVotes().ToEndPollPost(testutils.GetBundle(), "John Doe", converter)
	require.Nil(t, err)
	expectedPost.Id = "postID1"

	post := &model.Post{
		ChannelId: "channelID1",
	}

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.SubmitDialogRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.SubmitDialogResponse
		ExpectedMsg        string
	}{
		"Valid request with votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("UpdatePost", expectedPost).Return(nil, nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				store.PollStore.On("Delete", testutils.GetPollWithVotes()).Return(nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID1", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
		"Valid request, poll without postID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("UpdatePost", expectedPost).Return(nil, nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithVotes().Copy()
				poll.PostID = ""
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll, nil)
				store.PollStore.On("Delete", poll).Return(nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID1", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
		"Valid request, GetUser fails for poll creator": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID1", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, GetUser fails for voter": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID2", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, UpdatePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("UpdatePost", expectedPost).Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID2", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, PollStore.Delete fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("UpdatePost", expectedPost).Return(nil, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				store.PollStore.On("Delete", testutils.GetPollWithVotes()).Return(&model.AppError{})
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID2", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID1", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID1", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, post with invalid channelID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				p := post.Clone()
				p.ChannelId = "channelID2"
				api.On("GetPost", "postID1").Return(p, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID1", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.SubmitDialogRequest{UserId: "userID1", ChannelId: "channelID1", CallbackId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)

			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)

			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/end/confirm", testutils.GetPollID())
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.SubmitDialogResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			assert.Equal(test.ExpectedResponse, response)
			if test.ExpectedResponse != nil {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
			}
		})
	}
}

func TestPostEndPollAnnouncement(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI func(*plugintest.API) *plugintest.API
	}{
		"Valid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("CreatePost", &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    "postID1",
					Message: "The poll **Question** has ended and the original post has been updated. " +
						"You can jump to it by pressing [here](https://example.org/_redirect/pl/postID1).",
					Type: model.PostTypeDefault,
				}).Return(nil, nil)
				return api
			},
		},
		"Valid request, CreatePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{})
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 5)...).Return()
				return api
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})

			p := setupTestPlugin(t, api, &mockstore.Store{})
			p.postEndPollAnnouncement("channelID1", "postID1", "Question")
		})
	}
}
func TestHandleDeletePoll(t *testing.T) {
	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.PostActionIntegrationRequest{}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/delete", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)

		result := w.Result()
		require.NotNil(t, result)
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	triggerID := model.NewId()
	dialog := model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/%s/delete/confirm", root.Manifest.Id, testutils.GetPollID()),
		Dialog: model.Dialog{
			Title:       "Confirm Poll Delete",
			IconURL:     fmt.Sprintf(responseIconURL, testutils.GetSiteURL(), root.Manifest.Id),
			CallbackId:  "postID1",
			SubmitLabel: "Delete",
		},
	}
	post := &model.Post{
		ChannelId: "channelID1",
	}

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedMsg        string
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request, poll without postID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithoutPostID(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request with no votes, issuer is system admin": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(&model.User{
					Username: "user2",
					Roles:    model.SystemAdminRoleId + " " + model.SystemUserRoleId,
				}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "",
		},
		"Valid request, GetUser fails for issuer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, Invalid permission": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID2", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2", Roles: model.SystemUserRoleId}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID2",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Only the creator of a poll and System Admins are allowed to delete it.",
		},
		"Valid request, OpenInteractiveDialog fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("OpenInteractiveDialog", dialog).Return(&model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, Store.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, post with invalid channelID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				p := post.Clone()
				p.ChannelId = "channelID2"
				api.On("GetPost", "postID1").Return(p, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    "userID1",
				ChannelId: "channelID1",
				PostId:    "postID1",
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)

			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)

			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/delete", testutils.GetPollID())
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.PostActionIntegrationResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				assert.Equal(response, &model.PostActionIntegrationResponse{})
			} else {
				assert.Nil(response)
			}
		})
	}
}

func TestHandleDeletePollConfirm(t *testing.T) {
	t.Run("not-authorized", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		defer api.AssertExpectations(t)
		p := setupTestPlugin(t, api, &mockstore.Store{})
		request := &model.SubmitDialogRequest{}

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/polls/%s/delete/confirm", testutils.GetPollID())
		b, err := json.Marshal(request)
		require.Nil(t, err)
		body := bytes.NewReader(b)
		r := httptest.NewRequest(http.MethodPost, url, body)
		p.ServeHTTP(nil, w, r)
		result := w.Result()
		defer result.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	post := &model.Post{
		ChannelId: "channelID1",
	}

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.SubmitDialogRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.SubmitDialogResponse
		ExpectedMsg        string
	}{
		"Valid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("DeletePost", "postID1").Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				store.PollStore.On("Delete", testutils.GetPoll()).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID1",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Successfully deleted the poll.",
		},
		"Valid request, poll without postID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("DeletePost", "postID2").Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithoutPostID(), nil)
				store.PollStore.On("Delete", testutils.GetPollWithoutPostID()).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID2",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Successfully deleted the poll.",
		},
		"Valid request, DeletePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				api.On("DeletePost", "postID1").Return(&model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID1",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Valid request, PollStore.Delete fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(true)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("DeletePost", "postID1").Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				store.PollStore.On("Delete", testutils.GetPoll()).Return(&model.AppError{})
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID1",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
			ExpectedMsg:        "Something went wrong. Please try again later.",
		},
		"Invalid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, errors.New(""))
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID1",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID1",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"Invalid request, post with invalid channelID": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				p := post.Clone()
				p.ChannelId = "channelID2"
				api.On("GetPost", "postID1").Return(p, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID1",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Invalid request, without permission to read channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetPost", "postID1").Return(post, nil)
				api.On("HasPermissionToChannel", "userID1", "channelID1", model.PermissionReadChannel).Return(false)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     "userID1",
				CallbackId: "postID1",
				ChannelId:  "channelID1",
				Submission: map[string]interface{}{},
			},
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		"Empty request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
			ExpectedMsg:        "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			if test.ExpectedMsg != "" {
				ephemeralPost := &model.Post{
					ChannelId: test.Request.ChannelId,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedMsg,
				}
				api.On("SendEphemeralPost", test.Request.UserId, ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/polls/%s/delete/confirm", testutils.GetPollID())
			b, err := json.Marshal(test.Request)
			require.Nil(t, err)
			body := bytes.NewReader(b)
			r := httptest.NewRequest(http.MethodPost, url, body)
			if test.Request != nil {
				r.Header.Add("Mattermost-User-ID", test.Request.UserId)
			} else {
				r.Header.Add("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)

			var response *model.SubmitDialogResponse
			// Don't check if the response typed error is nil in order to do additional assertions.
			_ = json.NewDecoder(result.Body).Decode(&response)

			assert.Equal(test.ExpectedResponse, response)
			if test.ExpectedResponse != nil {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
			}
		})
	}
}

func TestHandlePollMetadata(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		UserID             string
		ShouldError        bool
		ExpectedStatusCode int
		ExpectedBody       *poll.Metadata
	}{
		"Valid request with votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			UserID:             "userID1",
			ShouldError:        false,
			ExpectedStatusCode: http.StatusOK,
			ExpectedBody: (&poll.Metadata{
				PollID:        testutils.GetPollID(),
				UserID:        "userID1",
				CanManagePoll: true,
				VotedAnswers:  []string{"Answer 1"},
			}),
		},
		"Valid request without votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID5").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			UserID:             "userID5",
			ShouldError:        false,
			ExpectedStatusCode: http.StatusOK,
			ExpectedBody: (&poll.Metadata{
				PollID:        testutils.GetPollID(),
				UserID:        "userID5",
				CanManagePoll: false,
				VotedAnswers:  []string{},
			}),
		},
		"Valid request without votes, CanManagePoll fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID5").Return(nil, &model.AppError{})
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 5)...).Return().Maybe()
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			UserID:             "userID5",
			ShouldError:        false,
			ExpectedStatusCode: http.StatusOK,
			ExpectedBody: (&poll.Metadata{
				PollID:        testutils.GetPollID(),
				UserID:        "userID5",
				CanManagePoll: false,
				VotedAnswers:  []string{},
			}),
		},
		"Valid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			UserID:             "userID1",
			ShouldError:        true,
			ExpectedStatusCode: http.StatusInternalServerError,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return().Maybe()
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/polls/%s/metadata", testutils.GetPollID()), nil)
			r.Header.Add("Mattermost-User-ID", test.UserID)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			bodyBytes, err := io.ReadAll(result.Body)
			require.Nil(t, err)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if test.ShouldError {
				assert.Equal([]byte{}, bodyBytes)
				assert.Equal(http.Header{}, result.Header)
			} else {
				assert.Contains([]string{"application/json"}, result.Header.Get("Content-Type"))
				b := new(bytes.Buffer)
				err = json.NewEncoder(b).Encode(test.ExpectedBody)
				assert.Nil(err)
				assert.Equal(b.Bytes(), bodyBytes)
			}
		})
	}
}
