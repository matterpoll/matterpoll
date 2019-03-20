package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/matterpoll/matterpoll/server/store/mockstore"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
			ExpectedbodyString: infoMessage,
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
			api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, &mockstore.Store{}, testutils.GetSiteURL())

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)

			bodyBytes, err := ioutil.ReadAll(result.Body)
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
				api.On("LogWarn", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ShouldError:        true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, &mockstore.Store{}, testutils.GetSiteURL())

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", fmt.Sprintf("/%s", iconFilename), nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)

			bodyBytes, err := ioutil.ReadAll(result.Body)
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

func TestHandleVote(t *testing.T) {
	poll1In := testutils.GetPoll()
	poll1Out := poll1In.Copy()
	err := poll1Out.UpdateVote("userID1", 0)
	require.Nil(t, err)
	expectedPost1 := &model.Post{}
	model.ParseSlackAttachment(expectedPost1, poll1Out.ToPostActions(testutils.GetSiteURL(), PluginId, "John Doe"))

	poll2In := testutils.GetPoll()
	err = poll2In.UpdateVote("userID1", 0)
	require.Nil(t, err)
	poll2Out := poll2In.Copy()
	err = poll2Out.UpdateVote("userID1", 1)
	require.Nil(t, err)
	expectedPost2 := &model.Post{}
	model.ParseSlackAttachment(expectedPost2, poll2Out.ToPostActions(testutils.GetSiteURL(), PluginId, "John Doe"))

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		VoteIndex          int
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll1In, nil)
				store.PollStore.On("Save", poll1Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteCounted, Update: expectedPost1},
		},
		"Valid request with vote": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(poll2In, nil)
				store.PollStore.On("Save", poll2Out).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteUpdated, Update: expectedPost2},
		},
		"Valid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, PollStore.Save fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				pollIn := testutils.GetPoll()
				pollOut := pollIn.Copy()
				err := pollOut.UpdateVote("userID1", 0)
				require.Nil(t, err)

				store.PollStore.On("Get", testutils.GetPollID()).Return(pollIn, nil)
				store.PollStore.On("Save", pollOut).Return(&model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid index": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          3,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
		},
		"Valid request, GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store, testutils.GetSiteURL())

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/vote/%d", testutils.GetPollID(), test.VoteIndex), bytes.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			response := model.PostActionIntegrationResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				require.NotNil(t, response)
				assert.Equal(test.ExpectedResponse.EphemeralText, response.EphemeralText)
				if test.ExpectedResponse.Update != nil {
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
	channelID := model.NewId()
	postID := model.NewId()

	responsePost := &model.Post{
		ChannelId: channelID,
		UserId:    userID,
		Message:   addOptionSuccess,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_icon_url": fmt.Sprintf(responseIconURL, testutils.GetSiteURL(), PluginId),
			"override_username": responseUsername,
		},
	}

	poll1In := testutils.GetPollWithVotes()
	poll1Out := poll1In.Copy()
	err := poll1Out.AddAnswerOption("New Option")
	require.Nil(t, err)
	expectedPost1 := &model.Post{}
	model.ParseSlackAttachment(expectedPost1, poll1Out.ToPostActions(testutils.GetSiteURL(), PluginId, "John Doe"))

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.SubmitDialogRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.SubmitDialogResponse
	}{
		"Valid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", userID).Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetPost", postID).Return(&model.Post{}, nil)
				api.On("UpdatePost", expectedPost1).Return(expectedPost1, nil)
				api.On("SendEphemeralPost", userID, responsePost).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				store.PollStore.On("Save", poll1Out).Return(nil)
				return store
			},
			Request: &model.SubmitDialogRequest{
				UserId:     userID,
				CallbackId: postID,
				ChannelId:  channelID,
				Submission: map[string]interface{}{
					addOptionKey: "New Option",
				},
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store, testutils.GetSiteURL())

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/option/add", testutils.GetPollID()), bytes.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			response := model.SubmitDialogResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedResponse, response)
			if test.ExpectedResponse != nil {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
			}
		})
	}
}

func TestHandleAddOptionDialogRequest(t *testing.T) {
	userID := testutils.GetPollWithVotes().Creator
	triggerID := model.NewId()
	postID := model.NewId()

	dialogRequest := model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       fmt.Sprintf("%s/plugins/%s/api/v1/polls/%s/option/add", testutils.GetSiteURL(), PluginId, testutils.GetPollID()),
		Dialog: model.Dialog{
			Title:       "Add Option",
			IconURL:     fmt.Sprintf(responseIconURL, testutils.GetSiteURL(), PluginId),
			CallbackId:  postID,
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

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("OpenInteractiveDialog", dialogRequest).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				PostId:    postID,
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{},
		},
		"Valid request, OpenInteractiveDialog fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("OpenInteractiveDialog", dialogRequest).Return(&model.AppError{})
				api.On("LogError", GetMockArgumentsWithType("string", 3)...).Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request: &model.PostActionIntegrationRequest{
				UserId:    userID,
				PostId:    postID,
				TriggerId: triggerID,
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store, testutils.GetSiteURL())

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/option/add/request", testutils.GetPollID()), bytes.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			response := model.PostActionIntegrationResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
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
	expectedPost, err := testutils.GetPollWithVotes().ToEndPollPost("John Doe", converter)
	require.Nil(t, err)

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channel_id"}, nil)
				api.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				store.PollStore.On("Delete", testutils.GetPollWithVotes()).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost},
		},
		"Valid request with votes, issuer is system admin": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{
					Username: "user2",
					Roles:    model.SYSTEM_ADMIN_ROLE_ID + " " + model.SYSTEM_USER_ROLE_ID,
				}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channel_id"}, nil)
				api.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				store.PollStore.On("Delete", testutils.GetPollWithVotes()).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost},
		},
		"Valid request, PollStore.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for issuer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Invalid permission": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2", Roles: model.SYSTEM_USER_ROLE_ID}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: endPollInvalidPermission},
		},
		"Valid request, PollStore.Delete fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				store.PollStore.On("Delete", testutils.GetPollWithVotes()).Return(&model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for poll creator": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for voter": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPollWithVotes(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store, testutils.GetSiteURL())

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/end", testutils.GetPollID()), bytes.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			response := model.PostActionIntegrationResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				require.NotNil(t, response)
				assert.Equal(test.ExpectedResponse.EphemeralText, response.EphemeralText)
				if test.ExpectedResponse.Update != nil {
					assert.Equal(test.ExpectedResponse.Update.Attachments(), response.Update.Attachments())
				}
			} else {
				assert.Equal(test.ExpectedResponse, response)
			}
		})
	}
}

func TestPostEndPollAnnouncement(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI func(*plugintest.API) *plugintest.API
		Request  *model.PostActionIntegrationRequest
	}{
		"Valid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
				api.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channelID1"}, nil)
				api.On("CreatePost", &model.Post{
					UserId:    "userID1",
					ChannelId: "channelID1",
					RootId:    "postID1",
					Message:   fmt.Sprintf(endPollSuccessfullyFormat, "Question", "https://example.org/team1/pl/postID1"),
					Type:      model.POST_DEFAULT,
					Props: model.StringInterface{
						"override_username": responseUsername,
						"override_icon_url": fmt.Sprintf(responseIconURL, "https://example.org", PluginId),
						"from_webhook":      "true",
					},
				}).Return(nil, nil)
				return api
			},
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
		"Valid request, GetTeam fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetTeam", "teamID1").Return(nil, &model.AppError{})
				api.On("LogError", GetMockArgumentsWithType("string", 3)...).Return(nil)
				return api
			},
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
		"Valid request, GetPost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
				api.On("GetPost", "postID1").Return(nil, &model.AppError{})
				api.On("LogError", GetMockArgumentsWithType("string", 3)...).Return(nil)
				return api
			},
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
		"Valid request, CreatePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
				api.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channelID1"}, nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{})
				api.On("LogError", GetMockArgumentsWithType("string", 3)...).Return(nil)
				return api
			},
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := setupTestPlugin(t, test.SetupAPI(&plugintest.API{}), &mockstore.Store{}, testutils.GetSiteURL())
			p.postEndPollAnnouncement(test.Request, "Question")
		})
	}
}
func TestHandleDeletePoll(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		SetupStore         func(*mockstore.Store) *mockstore.Store
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("DeletePost", "postID1").Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				store.PollStore.On("Delete", testutils.GetPoll()).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollSuccess},
		},
		"Valid request with no votes, issuer is system admin": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID2").Return(&model.User{
					Username: "user2",
					Roles:    model.SYSTEM_ADMIN_ROLE_ID + " " + model.SYSTEM_USER_ROLE_ID,
				}, nil)
				api.On("DeletePost", "postID1").Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				store.PollStore.On("Delete", testutils.GetPoll()).Return(nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollSuccess},
		},
		"Valid request, Store.Get fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(nil, &model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for issuer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Invalid permission": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2", Roles: model.SYSTEM_USER_ROLE_ID}, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollInvalidPermission},
		},
		"Valid request, DeletePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("DeletePost", "postID1").Return(&model.AppError{})
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, KVDelete fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("DeletePost", "postID1").Return(nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Get", testutils.GetPollID()).Return(testutils.GetPoll(), nil)
				store.PollStore.On("Delete", testutils.GetPoll()).Return(&model.AppError{})
				return store
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:         func(store *mockstore.Store) *mockstore.Store { return store },
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store, testutils.GetSiteURL())

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/delete", testutils.GetPollID()), bytes.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			response := model.PostActionIntegrationResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				require.NotNil(t, response)
				assert.Equal(test.ExpectedResponse.EphemeralText, response.EphemeralText)
				if test.ExpectedResponse.Update != nil {
					assert.Equal(test.ExpectedResponse.Update.Attachments(), response.Update.Attachments())
				}
			}
			assert.Equal(test.ExpectedResponse, response)
		})
	}
}
