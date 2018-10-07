package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	samplePollID  = "1234567890abcdefghij"
	samplesiteURL = "https://example.org"
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

var samplePollTwoOptions = Poll{
	CreatedAt:         1234567890,
	Creator:           "userID1",
	DataSchemaVersion: "v1",
	Question:          "Question",
	AnswerOptions: []*AnswerOption{
		{Answer: "Yes"},
		{Answer: "No"},
	},
}

func setupTestPlugin(t *testing.T, api *plugintest.API, siteURL string) *MatterpollPlugin {
	p := &MatterpollPlugin{
		ServerConfig: &model.Config{
			ServiceSettings: model.ServiceSettings{
				SiteURL: &siteURL,
			},
		},
	}
	p.setConfiguration(&configuration{
		Trigger: "poll",
	})
	p.SetAPI(api)
	err := p.OnActivate()
	require.Nil(t, err)
	return p
}

func TestPluginOnActivate(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		p := &MatterpollPlugin{}
		p.setConfiguration(&configuration{
			Trigger: "poll",
		})
		err := p.OnActivate()
		assert.Nil(t, err)
	})
}

func TestPluginOnDeactivate(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		p := setupTestPlugin(t, api, samplesiteURL)
		api.On("UnregisterCommand", "", p.getConfiguration().Trigger).Return(nil)
		defer api.AssertExpectations(t)

		err := p.OnDeactivate()
		assert.Nil(t, err)
	})

	t.Run("UnregisterCommand fails", func(t *testing.T) {
		api := &plugintest.API{}
		p := setupTestPlugin(t, api, samplesiteURL)
		api.On("UnregisterCommand", "", p.getConfiguration().Trigger).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		err := p.OnDeactivate()
		assert.NotNil(t, err)
	})
}

func GetMockArgumentsWithType(typeString string, num int) []interface{} {
	ret := make([]interface{}, num)
	for i := 0; i < len(ret); i++ {
		ret[i] = mock.AnythingOfTypeArgument(typeString)
	}
	return ret
}
