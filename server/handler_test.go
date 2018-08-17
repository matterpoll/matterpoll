package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestServeHTTP(t *testing.T) {
	for name, test := range map[string]struct {
		RequestURL         string
		KV                 bool
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"InvalidRequestURL": {
			RequestURL:         "/not_found",
			KV:                 false,
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedHeader:     http.Header{},
		},
		"ValidEndPollRequest": {
			RequestURL:         fmt.Sprintf("/polls/%s/end", new(MockPollIDGenerator).NewID()),
			KV:                 true,
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader: map[string][]string{
				"Content-Type": []string{"application/json"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			if test.KV {
				api.On("KVDelete", new(MockPollIDGenerator).NewID()).Return(nil)
				api.On("KVGet", new(MockPollIDGenerator).NewID()).Return(samplePollWithVotes.Encode(), nil)
				api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
				api.On("GetUser", "userID2").Return(&model.User{Username: "user2"}, nil)
				api.On("GetUser", "userID3").Return(&model.User{Username: "user3"}, nil)
				api.On("GetUser", "userID4").Return(&model.User{Username: "user4"}, nil)
				defer api.AssertExpectations(t)
			}
			p := MatterpollPlugin{}
			p.SetAPI(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)
			assert.Equal(t, test.ExpectedStatusCode, w.Result().StatusCode)
			assert.Equal(t, test.ExpectedHeader, w.Result().Header)

			if test.KV {
				var response model.PostActionIntegrationResponse
				json.NewDecoder(w.Result().Body).Decode(&response)
				assert.Equal(t, "Poll is done.\nAnswer 1: @user1, @user2 and @user3\nAnswer 2: @user4\nAnswer 3:\n", response.Update.Message)
			}
		})
	}
}

func TestVoteRequest(t *testing.T) {
	api := &plugintest.API{}

	api.On("KVGet", new(MockPollIDGenerator).NewID()).Return(samplePoll.Encode(), nil)
	samplePoll.UpdateVote("userID", 1)
	api.On("KVSet", new(MockPollIDGenerator).NewID(), samplePoll.Encode()).Return(nil)

	defer api.AssertExpectations(t)
	p := MatterpollPlugin{}
	p.SetAPI(api)

	request := model.PostActionIntegrationRequest{UserId: "userID"}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", fmt.Sprintf("/polls/%s/vote/1", new(MockPollIDGenerator).NewID()), strings.NewReader(request.ToJson()))
	p.ServeHTTP(nil, w, r)
	CheckHeaderOK(t, w.Result())
}

func CheckHeaderOK(t *testing.T, r *http.Response) {
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, http.Header{
		"Content-Type": []string{"application/json"},
	}, r.Header)
}
