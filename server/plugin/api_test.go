package plugin

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
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
			p := setupTestPlugin(t, api, samplesiteURL)

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
	mkdirCmd := exec.Command("mkdir", "-p", iconPath)
	cpCmd := exec.Command("cp", "../../assets/"+iconFilename, iconPath+iconFilename)
	mkdirCmd.Run()
	cpCmd.Run()
	defer func() {
		rmCmd := exec.Command("rm", "-r", "plugins")
		rmCmd.Run()
	}()

	assert := assert.New(t)
	api := &plugintest.API{}
	api.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()
	defer api.AssertExpectations(t)
	p := setupTestPlugin(t, api, samplesiteURL)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", fmt.Sprintf("/%s", iconFilename), nil)
	p.ServeHTTP(nil, w, r)

	result := w.Result()
	require.NotNil(t, result)

	bodyBytes, err := ioutil.ReadAll(result.Body)
	require.Nil(t, err)

	assert.NotNil(bodyBytes)
	assert.Equal(http.StatusOK, result.StatusCode)
	assert.Contains([]string{"image/png"}, result.Header.Get("Content-Type"))
}

func TestHandleVote(t *testing.T) {
	poll1_in := samplePoll.Copy()
	poll1_out := poll1_in.Copy()
	poll1_out.UpdateVote("userID1", 0)
	expectedPost1 := &model.Post{}
	model.ParseSlackAttachment(expectedPost1, poll1_out.ToPostActions(samplesiteURL, samplePollID, "John Doe"))

	poll2_in := samplePoll.Copy()
	poll2_in.UpdateVote("userID1", 0)
	poll2_out := poll2_in.Copy()
	poll2_out.UpdateVote("userID1", 1)
	expectedPost2 := &model.Post{}
	model.ParseSlackAttachment(expectedPost2, poll2_out.ToPostActions(samplesiteURL, samplePollID, "John Doe"))

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		Request            *model.PostActionIntegrationRequest
		VoteIndex          int
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(poll1_in.Encode(), nil)
				api.On("KVSet", samplePollID, poll1_out.Encode()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteCounted, Update: expectedPost1},
		},
		"Valid request with vote": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(poll2_in.Encode(), nil)
				api.On("KVSet", samplePollID, poll2_out.Encode()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteUpdated, Update: expectedPost2},
		},

		"Valid request, KVGet fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(nil, &model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},

		"Valid request, Decode fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(nil, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, KVSet fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				poll_in := samplePoll.Copy()
				poll_out := poll_in.Copy()
				poll_out.UpdateVote("userID1", 0)

				api.On("KVGet", samplePollID).Return(poll_in.Encode(), nil)
				api.On("KVSet", samplePollID, poll_out.Encode()).Return(&model.AppError{})
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid index": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          3,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
			Request:            nil,
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
		},
		"Valid request, GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
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
			p := setupTestPlugin(t, api, samplesiteURL)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/vote/%d", samplePollID, test.VoteIndex), strings.NewReader(test.Request.ToJson()))
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

func TestHandleEndPoll(t *testing.T) {
	expectedPost := &model.Post{}
	model.ParseSlackAttachment(expectedPost, []*model.SlackAttachment{{
		AuthorName: "John Doe",
		Title:      "Question",
		Text:       "This poll has ended. The results are:",
		Fields: []*model.SlackAttachmentField{{
			Title: "Answer 1 (3 votes)",
			Value: "@user1, @user2 and @user3",
			Short: true,
		}, {
			Title: "Answer 2 (1 vote)",
			Value: "@user4",
			Short: true,
		}, {
			Title: "Answer 3 (0 votes)",
			Value: "",
			Short: true,
		}},
	}})

	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
				api.On("KVDelete", samplePollID).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channel_id"}, nil)
				api.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost},
		},
		"Valid request with no votes, issuer is system admin": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
				api.On("KVDelete", samplePollID).Return(nil)
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
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost},
		},
		"Valid request, KVGet fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(nil, &model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Decode fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(nil, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for issuer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Invalid permission": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2", Roles: model.SYSTEM_USER_ROLE_ID}, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: endPollInvalidPermission},
		},
		"Valid request, DeletePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				api.On("KVDelete", samplePollID).Return(&model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for poll creator": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for voter": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				return api
			},
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
			p := setupTestPlugin(t, api, samplesiteURL)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/end", samplePollID), strings.NewReader(test.Request.ToJson()))
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
			p := setupTestPlugin(t, test.SetupAPI(&plugintest.API{}), samplesiteURL)
			p.postEndPollAnnouncement(test.Request, "Question")
		})
	}
}
func TestHandleDeletePoll(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("DeletePost", "postID1").Return(nil)
				api.On("KVDelete", samplePollID).Return(nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollSuccess},
		},
		"Valid request with no votes, issuer is system admin": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("GetUser", "userID2").Return(&model.User{
					Username: "user2",
					Roles:    model.SYSTEM_ADMIN_ROLE_ID + " " + model.SYSTEM_USER_ROLE_ID,
				}, nil)
				api.On("DeletePost", "postID1").Return(nil)
				api.On("KVDelete", samplePollID).Return(nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollSuccess},
		},
		"Valid request, KVGet fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(nil, &model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Decode fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(nil, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for issuer": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("GetUser", "userID2").Return(nil, &model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Invalid permission": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2", Roles: model.SYSTEM_USER_ROLE_ID}, nil)
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollInvalidPermission},
		},
		"Valid request, DeletePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("DeletePost", "postID1").Return(&model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, KVDelete fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
				api.On("DeletePost", "postID1").Return(nil)
				api.On("KVDelete", samplePollID).Return(&model.AppError{})
				return api
			},
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			SetupAPI:           func(api *plugintest.API) *plugintest.API { return api },
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
			p := setupTestPlugin(t, api, samplesiteURL)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/delete", samplePollID), strings.NewReader(test.Request.ToJson()))
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
