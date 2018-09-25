package main

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
	api1 := &plugintest.API{}

	for name, test := range map[string]struct {
		API                *plugintest.API
		RequestURL         string
		ExpectedStatusCode int
		ExpectedHeader     http.Header
		ExpectedbodyString string
	}{
		"Request info": {
			API:                api1,
			RequestURL:         "/",
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
			ExpectedbodyString: infoMessage,
		},
		"InvalidRequestURL": {
			API:                api1,
			RequestURL:         "/not_found",
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
			ExpectedbodyString: "404 page not found\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := setupTestPlugin(t, test.API, samplesiteURL)
			test.API.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()

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
	cpCmd := exec.Command("cp", "../assets/"+iconFilename, iconPath+iconFilename)
	mkdirCmd.Run()
	cpCmd.Run()
	defer func() {
		rmCmd := exec.Command("rm", "-r", "plugins")
		rmCmd.Run()
	}()

	assert := assert.New(t)
	api1 := &plugintest.API{}
	p := setupTestPlugin(t, api1, samplesiteURL)
	api1.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()

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
	poll1 := samplePoll.Copy()
	api1 := &plugintest.API{}
	api1.On("KVGet", samplePollID).Return(poll1.Encode(), nil)
	poll1.UpdateVote("userID1", 0)
	api1.On("KVSet", samplePollID, poll1.Encode()).Return(nil)
	api1.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	defer api1.AssertExpectations(t)
	expectedPost1 := &model.Post{}
	model.ParseSlackAttachment(expectedPost1, poll1.ToPostActions(samplesiteURL, samplePollID, "John Doe"))

	poll2 := samplePoll.Copy()
	api2 := &plugintest.API{}
	poll2.UpdateVote("userID1", 0)
	api2.On("KVGet", samplePollID).Return(poll2.Encode(), nil)
	poll2.UpdateVote("userID1", 1)
	api2.On("KVSet", samplePollID, poll2.Encode()).Return(nil)
	api2.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	defer api2.AssertExpectations(t)
	expectedPost2 := &model.Post{}
	model.ParseSlackAttachment(expectedPost2, poll2.ToPostActions(samplesiteURL, samplePollID, "John Doe"))

	api3 := &plugintest.API{}
	api3.On("KVGet", samplePollID).Return(nil, &model.AppError{})
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVGet", samplePollID).Return(nil, nil)
	defer api4.AssertExpectations(t)

	poll5 := samplePoll.Copy()
	api5 := &plugintest.API{}
	api5.On("KVGet", samplePollID).Return(poll5.Encode(), nil)
	poll5.UpdateVote("userID1", 0)
	api5.On("KVSet", samplePollID, poll5.Encode()).Return(&model.AppError{})
	api5.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	defer api5.AssertExpectations(t)

	api6 := &plugintest.API{}
	api6.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
	api6.On("GetUser", "userID1").Return(nil, &model.AppError{})
	defer api6.AssertExpectations(t)

	for name, test := range map[string]struct {
		API                *plugintest.API
		Request            *model.PostActionIntegrationRequest
		VoteIndex          int
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			API:                api1,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteCounted, Update: expectedPost1},
		},
		"Valid request with vote": {
			API:                api2,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteUpdated, Update: expectedPost2},
		},
		"Valid request, KVGet fails": {
			API:                api3,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Decode fails": {
			API:                api4,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, KVDelete fails": {
			API:                api5,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid index": {
			API:                api1,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          3,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			API:                &plugintest.API{},
			Request:            nil,
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
		},
		"Valid request, GetUser fails": {
			API:                api6,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          0,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := setupTestPlugin(t, test.API, samplesiteURL)
			test.API.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()

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
	api1 := &plugintest.API{}
	api1.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
	api1.On("KVDelete", samplePollID).Return(nil)
	api1.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
	api1.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
	api1.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
	api1.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
	api1.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channel_id"}, nil)
	api1.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
	api1.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
	defer api1.AssertExpectations(t)

	expectedattachments1 := []*model.SlackAttachment{{
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
	}}
	expectedPost1 := &model.Post{}
	model.ParseSlackAttachment(expectedPost1, expectedattachments1)

	api2 := &plugintest.API{}
	api2.On("KVGet", samplePollID).Return(nil, &model.AppError{})
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("KVGet", samplePollID).Return(nil, nil)
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
	defer api4.AssertExpectations(t)

	api5 := &plugintest.API{}
	api5.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
	api5.On("KVDelete", samplePollID).Return(&model.AppError{})
	api5.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
	api5.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
	api5.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
	api5.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
	defer api5.AssertExpectations(t)

	api6 := &plugintest.API{}
	api6.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
	api6.On("GetUser", "userID1").Return(nil, &model.AppError{})
	defer api6.AssertExpectations(t)

	api7 := &plugintest.API{}
	api7.On("KVGet", samplePollID).Return(samplePollWithVotes.Encode(), nil)
	api7.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
	api7.On("GetUser", "userID2").Return(nil, &model.AppError{})
	defer api7.AssertExpectations(t)

	for name, test := range map[string]struct {
		API                *plugintest.API
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			API:                api1,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: expectedPost1},
		},
		"Valid request, KVGet fails": {
			API:                api2,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Decode fails": {
			API:                api3,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid permission": {
			API:                api4,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: endPollInvalidPermission},
		},
		"Valid request, DeletePost fails": {
			API:                api5,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for poll creator": {
			API:                api6,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, GetUser fails for voter": {
			API:                api7,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			API:                &plugintest.API{},
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := setupTestPlugin(t, test.API, samplesiteURL)
			test.API.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()

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
	api1 := &plugintest.API{}
	api1.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
	api1.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channelID1"}, nil)
	api1.On("CreatePost", &model.Post{
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
	defer api1.AssertExpectations(t)

	api2 := &plugintest.API{}
	api2.On("GetTeam", "teamID1").Return(nil, &model.AppError{})
	api2.On("LogError", GetMockArgumentsWithType("string", 3)...).Return(nil)
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
	api3.On("GetPost", "postID1").Return(nil, &model.AppError{})
	api3.On("LogError", GetMockArgumentsWithType("string", 3)...).Return(nil)
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("GetTeam", "teamID1").Return(&model.Team{Name: "team1"}, nil)
	api4.On("GetPost", "postID1").Return(&model.Post{ChannelId: "channelID1"}, nil)
	api4.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{})
	api4.On("LogError", GetMockArgumentsWithType("string", 3)...).Return(nil)
	defer api4.AssertExpectations(t)

	for name, test := range map[string]struct {
		API     *plugintest.API
		Request *model.PostActionIntegrationRequest
	}{
		"Valid request": {
			API:     api1,
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
		"Valid request, GetTeam fails": {
			API:     api2,
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
		"Valid request, GetPost fails": {
			API:     api3,
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
		"Valid request, CreatePost fails": {
			API:     api4,
			Request: &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1", TeamId: "teamID1"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := setupTestPlugin(t, test.API, samplesiteURL)
			p.postEndPollAnnouncement(test.Request, "Question")
		})
	}
}
func TestHandleDeletePoll(t *testing.T) {
	api1 := &plugintest.API{}
	api1.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
	api1.On("DeletePost", "postID1").Return(nil)
	api1.On("KVDelete", samplePollID).Return(nil)
	defer api1.AssertExpectations(t)

	api2 := &plugintest.API{}
	api2.On("KVGet", samplePollID).Return(nil, &model.AppError{})
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("KVGet", samplePollID).Return(nil, nil)
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
	defer api4.AssertExpectations(t)

	api5 := &plugintest.API{}
	api5.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
	api5.On("DeletePost", "postID1").Return(&model.AppError{})
	defer api1.AssertExpectations(t)

	api6 := &plugintest.API{}
	api6.On("KVGet", samplePollID).Return(samplePoll.Encode(), nil)
	api6.On("DeletePost", "postID1").Return(nil)
	api6.On("KVDelete", samplePollID).Return(&model.AppError{})
	defer api6.AssertExpectations(t)

	for name, test := range map[string]struct {
		API                *plugintest.API
		Request            *model.PostActionIntegrationRequest
		ExpectedStatusCode int
		ExpectedResponse   *model.PostActionIntegrationResponse
	}{
		"Valid request with no votes": {
			API:                api1,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollSuccess},
		},
		"Valid request, KVGet fails": {
			API:                api2,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Decode fails": {
			API:                api3,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, Invalid permission": {
			API:                api4,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID2", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: deletePollInvalidPermission},
		},
		"Valid request, DeletePost fails": {
			API:                api5,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Valid request, KVDelete fails": {
			API:                api6,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: commandGenericError},
		},
		"Invalid request": {
			API:                &plugintest.API{},
			Request:            nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedResponse:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := setupTestPlugin(t, test.API, samplesiteURL)
			test.API.On("LogDebug", GetMockArgumentsWithType("string", 7)...).Return()

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
