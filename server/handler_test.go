package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServeHTTP(t *testing.T) {
	idGen := new(MockPollIDGenerator)

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
			ExpectedHeader:     http.Header{},
			ExpectedbodyString: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			p := &MatterpollPlugin{
				idGen: idGen,
			}
			AllowRequestLogging(test.API)
			p.SetAPI(test.API)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()

			bodyBytes, err := ioutil.ReadAll(result.Body)
			assert.Nil(err)
			bodyString := string(bodyBytes)

			assert.Equal(test.ExpectedbodyString, bodyString)
			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestHandleVote(t *testing.T) {
	idGen := new(MockPollIDGenerator)

	api1 := &plugintest.API{}
	api1.On("KVGet", idGen.NewID()).Return(samplePoll.Encode(), nil)
	samplePoll.UpdateVote("userID1", 0)
	api1.On("KVSet", idGen.NewID(), samplePoll.Encode()).Return(nil)
	defer api1.AssertExpectations(t)

	api2 := &plugintest.API{}
	samplePoll.UpdateVote("userID1", 0)
	api2.On("KVGet", idGen.NewID()).Return(samplePoll.Encode(), nil)
	samplePoll.UpdateVote("userID1", 1)
	api2.On("KVSet", idGen.NewID(), samplePoll.Encode()).Return(nil)
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("KVGet", idGen.NewID()).Return(nil, &model.AppError{})
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVGet", idGen.NewID()).Return(nil, nil)
	defer api4.AssertExpectations(t)

	api5 := &plugintest.API{}
	api5.On("KVGet", idGen.NewID()).Return(samplePoll.Encode(), nil)
	samplePoll.UpdateVote("userID1", 0)
	api5.On("KVSet", idGen.NewID(), samplePoll.Encode()).Return(&model.AppError{})
	defer api5.AssertExpectations(t)

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
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteCounted},
		},
		"Valid request with vote": {
			API:                api2,
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			VoteIndex:          1,
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{EphemeralText: voteUpdated},
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
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			p := &MatterpollPlugin{
				idGen: idGen,
			}
			AllowRequestLogging(test.API)
			p.SetAPI(test.API)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/vote/%d", idGen.NewID(), test.VoteIndex), strings.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			response := model.PostActionIntegrationResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
			}
			assert.Equal(test.ExpectedResponse, response)
		})
	}
}

func TestHandleEndPoll(t *testing.T) {
	idGen := new(MockPollIDGenerator)

	api1 := &plugintest.API{}
	api1.On("KVGet", idGen.NewID()).Return(samplePollWithVotes.Encode(), nil)
	api1.On("KVDelete", idGen.NewID()).Return(nil)
	api1.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
	api1.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
	api1.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
	api1.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
	defer api1.AssertExpectations(t)

	expectedattachments1 := []*model.SlackAttachment{{
		AuthorName: "John Doue",
		Title:      "Question",
		Text:       "This poll has ended. The results are:",
		Fields: []*model.SlackAttachmentField{
			{
				Title: "Answer 1 Answer 1 (3 votes)",
				Value: "user1, user2 and user3",
				Short: true,
			},
			{
				Title: "Answer 1 (1 vote)",
				Value: "user4",
				Short: true,
			},
			{
				Title: "Answer 3 (0 votes)",
				Value: "",
				Short: true,
			},
		},
	}}
	expectedPost1 := model.Post{}
	expectedPost1.AddProp("attachments", expectedattachments1)

	api2 := &plugintest.API{}
	api2.On("KVGet", idGen.NewID()).Return(nil, &model.AppError{})
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("KVGet", idGen.NewID()).Return(nil, nil)
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVGet", idGen.NewID()).Return(samplePollWithVotes.Encode(), nil)
	defer api4.AssertExpectations(t)

	api5 := &plugintest.API{}
	api5.On("KVGet", idGen.NewID()).Return(samplePollWithVotes.Encode(), nil)
	api5.On("KVDelete", idGen.NewID()).Return(&model.AppError{})
	api5.On("GetUser", "userID1").Return(&model.User{Username: "user1", FirstName: "John", LastName: "Doe"}, nil)
	api5.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
	api5.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
	api5.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
	defer api5.AssertExpectations(t)

	api6 := &plugintest.API{}
	api6.On("KVGet", idGen.NewID()).Return(samplePollWithVotes.Encode(), nil)
	api6.On("GetUser", "userID1").Return(nil, &model.AppError{})
	defer api6.AssertExpectations(t)

	api7 := &plugintest.API{}
	api7.On("KVGet", idGen.NewID()).Return(samplePollWithVotes.Encode(), nil)
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
			Request:            &model.PostActionIntegrationRequest{UserId: "userID1", PostId: "postID1"},
			ExpectedStatusCode: http.StatusOK,
			ExpectedResponse:   &model.PostActionIntegrationResponse{Update: &expectedPost1},
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
			p := &MatterpollPlugin{
				idGen: idGen,
			}
			AllowRequestLogging(test.API)
			p.SetAPI(test.API)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/end", idGen.NewID()), strings.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			response := model.PostActionIntegrationResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
				assert.Equal(test.ExpectedResponse.EphemeralText, response.EphemeralText)
				//// FIXME:response.Update.SlackAttachment is map[string]interface {} not []*model.SlackAttachment
				// assert.Equal(test.ExpectedResponse.Update, response.Update)
			} else {
				assert.Equal(test.ExpectedResponse, response)
			}
		})
	}
}

func TestHandleDeletePoll(t *testing.T) {
	idGen := new(MockPollIDGenerator)

	api1 := &plugintest.API{}
	api1.On("KVGet", idGen.NewID()).Return(samplePoll.Encode(), nil)
	api1.On("DeletePost", "postID1").Return(nil)
	api1.On("KVDelete", idGen.NewID()).Return(nil)
	defer api1.AssertExpectations(t)

	api2 := &plugintest.API{}
	api2.On("KVGet", idGen.NewID()).Return(nil, &model.AppError{})
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("KVGet", idGen.NewID()).Return(nil, nil)
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVGet", idGen.NewID()).Return(samplePoll.Encode(), nil)
	defer api4.AssertExpectations(t)

	api5 := &plugintest.API{}
	api5.On("KVGet", idGen.NewID()).Return(samplePoll.Encode(), nil)
	api5.On("DeletePost", "postID1").Return(&model.AppError{})
	defer api1.AssertExpectations(t)

	api6 := &plugintest.API{}
	api6.On("KVGet", idGen.NewID()).Return(samplePoll.Encode(), nil)
	api6.On("DeletePost", "postID1").Return(nil)
	api6.On("KVDelete", idGen.NewID()).Return(&model.AppError{})
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
			p := &MatterpollPlugin{
				idGen: idGen,
			}
			AllowRequestLogging(test.API)
			p.SetAPI(test.API)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/polls/%s/delete", idGen.NewID()), strings.NewReader(test.Request.ToJson()))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			response := model.PostActionIntegrationResponseFromJson(result.Body)

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if result.StatusCode == http.StatusOK {
				assert.Equal(http.Header{
					"Content-Type": []string{"application/json"},
				}, result.Header)
			}
			assert.Equal(test.ExpectedResponse, response)
		})
	}
}

func AllowRequestLogging(api *plugintest.API) {
	api.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()
}
