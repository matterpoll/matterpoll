package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var samplePoll = Poll{
	CreatedAt:         1234567890,
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
	CreatedAt:         1234567890,
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

func setupTestPlugin(t *testing.T, api *plugintest.API, siteURL string) *MatterpollPlugin {
	p := &MatterpollPlugin{
		Config: &Config{
			Trigger: "poll",
		},
		ServerConfig: &model.Config{
			ServiceSettings: model.ServiceSettings{
				SiteURL: &siteURL,
			},
		},
	}
	p.SetAPI(api)
	err := p.OnActivate()
	require.Nil(t, err)
	p.idGen = new(MockPollIDGenerator)
	return p
}

func TestPluginOnActivate(t *testing.T) {
	p := &MatterpollPlugin{
		Config: &Config{},
	}
	err := p.OnActivate()
	assert.Nil(t, err)
}

func TestPluginOnActivateEmptyConfig(t *testing.T) {
	p := &MatterpollPlugin{}
	err := p.OnActivate()
	assert.NotNil(t, err)
}

func TestPluginOnDeactivate(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(t, api, "https://example.org")
	api.On("UnregisterCommand", "", p.Config.Trigger).Return(nil)
	defer api.AssertExpectations(t)

	err := p.OnDeactivate()
	assert.Nil(t, err)
}
