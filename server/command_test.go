package main

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestPluginExecuteCommandHelp(t *testing.T) {
	api := &plugintest.API{}
	p := &MatterpollPlugin{}
	p.SetAPI(api)

	r, err := p.ExecuteCommand(nil, &model.CommandArgs{
		Command: "/matterpoll",
	})

	assert.Nil(t, err)
	assertHelpResponse(t, r)
}

func TestPluginExecuteTwoArguments(t *testing.T) {
	api := &plugintest.API{}
	p := &MatterpollPlugin{}
	p.SetAPI(api)

	r, err := p.ExecuteCommand(nil, &model.CommandArgs{
		Command: "/matterpoll \"Question\" \"Just one option\"",
	})
	assert.Nil(t, err)
	assertHelpResponse(t, r)
}

func TestPluginExecuteCommand(t *testing.T) {
	assert := assert.New(t)

	siteURL := "https://example.org/"
	idGen := new(MockPollIDGenerator)
	api := &plugintest.API{}

	api.On("KVSet", idGen.NewID(), samplePoll.Encode()).Return(nil)
	api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	defer api.AssertExpectations(t)
	p := &MatterpollPlugin{
		idGen: idGen,
	}
	p.SetAPI(api)

	r, err := p.ExecuteCommand(nil, &model.CommandArgs{
		Command: "/matterpoll \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"",
		SiteURL: siteURL,
		UserId:  "userID1",
	})

	assert.Nil(err)
	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, r.ResponseType)
	assert.Equal(model.POST_DEFAULT, r.Type)
	assert.Equal(responseUsername, r.Username)
	assert.Equal(responseIconURL, r.IconURL)
	assert.Equal([]*model.SlackAttachment{{
		AuthorName: "John Doe",
		Text:       "Question",
		Actions: []*model.PostAction{
			{Name: "Answer 1", Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/vote/0", siteURL, PluginId, p.idGen.NewID()),
			}},
			{Name: "Answer 2", Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/vote/1", siteURL, PluginId, p.idGen.NewID()),
			}},
			{Name: "Answer 3", Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/vote/2", siteURL, PluginId, p.idGen.NewID()),
			}},
			{Name: "End Poll", Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/end", siteURL, PluginId, p.idGen.NewID()),
			}},
		},
	}}, r.Attachments)
}

func TestPluginExecuteCommandWithQuestion(t *testing.T) {
	assert := assert.New(t)

	siteURL := "https://example.org/"
	poll := Poll{
		Creator:  "userID1",
		Question: "Question",
		Options: []*Option{
			{Answer: "Yes"},
			{Answer: "No"},
		},
	}
	idGen := new(MockPollIDGenerator)
	api := &plugintest.API{}
	api.On("KVSet", idGen.NewID(), poll.Encode()).Return(nil)
	api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	defer api.AssertExpectations(t)
	p := &MatterpollPlugin{
		idGen: idGen,
	}
	p.SetAPI(api)

	r, err := p.ExecuteCommand(nil, &model.CommandArgs{
		Command: "/matterpoll \"Question\"",
		SiteURL: siteURL,
		UserId:  "userID1",
	})

	assert.Nil(err)
	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, r.ResponseType)
	assert.Equal(model.POST_DEFAULT, r.Type)
	assert.Equal(responseUsername, r.Username)
	assert.Equal(responseIconURL, r.IconURL)
	assert.Equal([]*model.SlackAttachment{{
		AuthorName: "John Doe",
		Text:       "Question",
		Actions: []*model.PostAction{
			{Name: "Yes", Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/vote/0", siteURL, PluginId, p.idGen.NewID()),
			}},
			{Name: "No", Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/vote/1", siteURL, PluginId, p.idGen.NewID()),
			}},
			{Name: "End Poll", Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/end", siteURL, PluginId, p.idGen.NewID()),
			}},
		},
	}}, r.Attachments)
}

func assertHelpResponse(t *testing.T, r *model.CommandResponse) {
	assert := assert.New(t)

	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, r.ResponseType)
	assert.Equal(model.POST_DEFAULT, r.Type)
	assert.Equal(responseUsername, r.Username)
	assert.Equal(responseIconURL, r.IconURL)
	assert.Equal(commandInputError, r.Text)
	assert.Nil(r.Attachments)
}
