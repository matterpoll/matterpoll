package main

import (
	"fmt"
	"testing"

	"github.com/bouk/monkey"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPluginExecuteCommand(t *testing.T) {
	trigger := "poll"

	api1 := &plugintest.API{}
	api1.On("KVSet", samplePollID, samplePoll_twoOptions.Encode()).Return(nil)
	api1.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	api1.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()
	defer api1.AssertExpectations(t)

	api2 := &plugintest.API{}
	api2.On("KVSet", samplePollID, samplePoll.Encode()).Return(nil)
	api2.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	api2.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("KVSet", samplePollID, samplePoll.Encode()).Return(&model.AppError{})
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVSet", samplePollID, samplePoll.Encode()).Return(nil)
	api4.On("GetUser", "userID1").Return(nil, &model.AppError{})
	defer api4.AssertExpectations(t)

	for name, test := range map[string]struct {
		API                  *plugintest.API
		Command              string
		ExpectedError        *model.AppError
		ExpectedResponseType string
		ExpectedText         string
		ExpectedAttachments  []*model.SlackAttachment
	}{
		"No argument": {
			Command:              fmt.Sprintf("/%s", trigger),
			API:                  &plugintest.API{},
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandInputErrorFormat, trigger, trigger),
			ExpectedAttachments:  nil,
		},
		"Help text": {
			Command:              fmt.Sprintf("/%s help", trigger),
			API:                  &plugintest.API{},
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandHelpTextFormat, trigger, trigger),
			ExpectedAttachments:  nil,
		},
		"Two arguments": {
			Command:              fmt.Sprintf("/%s \"Question\" \"Just one option\"", trigger),
			API:                  &plugintest.API{},
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandInputErrorFormat, trigger, trigger),
			ExpectedAttachments:  nil,
		},
		"Just question": {
			Command:              fmt.Sprintf("/%s \"Question\"", trigger),
			API:                  api1,
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			ExpectedText:         "",
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "Total votes: 0",
				Actions: []*model.PostAction{{
					Name: "Yes",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "No",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					}},
				},
			}},
		},
		"With 4 arguments": {
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			API:                  api2,
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			ExpectedText:         "",
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "Total votes: 0",
				Actions: []*model.PostAction{{
					Name: "Answer 1",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "Answer 2",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "Answer 3",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/2", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				},
				},
			}},
		},
		"KVSet fails": {
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			API:                  api3,
			ExpectedError:        &model.AppError{},
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         commandGenericError,
			ExpectedAttachments:  nil,
		},
		"GetUser fails": {
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			API:                  api4,
			ExpectedError:        &model.AppError{},
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         commandGenericError,
			ExpectedAttachments:  nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := setupTestPlugin(t, test.API, samplesiteURL)
			p.Config.Trigger = trigger
			patch1 := monkey.Patch(model.GetMillis, func() int64 { return 1234567890 })
			patch2 := monkey.Patch(model.NewId, func() string { return samplePollID })
			defer patch1.Unpatch()
			defer patch2.Unpatch()

			r, err := p.ExecuteCommand(nil, &model.CommandArgs{
				Command: test.Command,
				UserId:  "userID1",
			})

			assert.Equal(test.ExpectedError, err)
			require.NotNil(t, r)

			assert.Equal(model.POST_DEFAULT, r.Type)
			assert.Equal(responseUsername, r.Username)
			assert.Equal(fmt.Sprintf(responseIconURL, samplesiteURL, PluginId), r.IconURL)
			assert.Equal(test.ExpectedResponseType, r.ResponseType)
			assert.Equal(test.ExpectedText, r.Text)
			assert.Equal(test.ExpectedAttachments, r.Attachments)
		})
	}
}
