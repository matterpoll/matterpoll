package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var samplePoll = Poll{
	Creator:           "userID1",
	DataSchemaVersion: "v1",
	Question:          "Question",
	AnswerOptions: []*AnswerOption{
		{Answer: "Answer 1"},
		{Answer: "Answer 2"},
		{Answer: "Answer 3"},
	},
}

var samplePollWithVotes = Poll{
	Creator:           "userID1",
	DataSchemaVersion: "v1",
	Question:          "Question",
	AnswerOptions: []*AnswerOption{
		{Answer: "Answer 1",
			Voter: []string{"userID1", "userID2", "userID3"}},
		{Answer: "Answer 2",
			Voter: []string{"userID4"}},
		{Answer: "Answer 3"},
	},
}

type MockPollIDGenerator struct {
	mock.Mock
}

func (m *MockPollIDGenerator) NewID() string {
	return "1234567890abcdefghij"
}

/*
func TestPluginOnActivate(t *testing.T) {
	api := &plugintest.API{}
	api.On("RegisterCommand", &model.Command{
		Trigger:          trigger,
		DisplayName:      "Matterpoll",
		Description:      "Polling feature by https://github.com/matterpoll/matterpoll",
		AutoComplete:     true,
		AutoCompleteDesc: "Create a poll",
		AutoCompleteHint: "[Question] [Answer 1] [Answer 2]...",
	}).Return(nil)
	defer api.AssertExpectations(t)
	p := &MatterpollPlugin{}
	p.SetAPI(api)

	err := p.OnActivate()
	assert.Nil(t, err)
}
*/

func TestPluginOnDeactivate(t *testing.T) {
	p := &MatterpollPlugin{
		Config: &Config{Trigger: "poll"},
	}
	api := &plugintest.API{}
	api.On("UnregisterCommand", "", p.Config.Trigger).Return(nil)
	defer api.AssertExpectations(t)
	p.SetAPI(api)

	err := p.OnDeactivate()
	assert.Nil(t, err)
}
