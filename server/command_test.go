package main

import (
	"fmt"
	"testing"

	"github.com/bouk/monkey"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginExecuteCommand(t *testing.T) {
	trigger := "poll"

	/*
		api1 := &plugintest.API{}
		api1.On("KVSet", samplePollID, samplePoll_twoOptions.Encode()).Return(nil)
		api1.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
		api1.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
		defer api1.AssertExpectations(t)

		api2 := &plugintest.API{}
		api2.On("KVSet", samplePollID, samplePoll.Encode()).Return(nil)
		api2.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
		api2.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
		defer api2.AssertExpectations(t)

		poll3 := samplePoll.Copy()
		poll3.Settings.Progress = true
		api3 := &plugintest.API{}
		api3.On("KVSet", samplePollID, poll3.Encode()).Return(nil)
		api3.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
		api3.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
		defer api3.AssertExpectations(t)

		api4 := &plugintest.API{}
		api4.On("KVSet", samplePollID, samplePoll.Encode()).Return(&model.AppError{})
		defer api4.AssertExpectations(t)

		api5 := &plugintest.API{}
		api5.On("KVSet", samplePollID, samplePoll.Encode()).Return(nil)
		api5.On("GetUser", "userID1").Return(nil, &model.AppError{})
		defer api5.AssertExpectations(t)
	*/
	for name, test := range map[string]struct {
		SetupAPI             func(*plugintest.API) *plugintest.API
		Command              string
		ExpectedError        *model.AppError
		ExpectedResponseType string
		ExpectedText         string
		ExpectedAttachments  []*model.SlackAttachment
	}{
		"No argument": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s", trigger),
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandHelpTextFormat, trigger),
			ExpectedAttachments:  nil,
		},
		"Help text": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s help", trigger),
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandHelpTextFormat, trigger),
			ExpectedAttachments:  nil,
		},
		"Two arguments": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s \"Question\" \"Just one option\"", trigger),
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandInputErrorFormat, trigger),
			ExpectedAttachments:  nil,
		},
		"Just question": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", samplePollID, samplePoll_twoOptions.Encode()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\"", trigger),
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
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", samplePollID, samplePoll.Encode()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
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
		"With 4 arguments and settting progress": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				poll := samplePoll.Copy()
				poll.Settings.Progress = true

				api.On("KVSet", samplePollID, poll.Encode()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --progress", trigger),
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			ExpectedText:         "",
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "Total votes: 0",
				Actions: []*model.PostAction{{
					Name: "Answer 1 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "Answer 2 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", samplesiteURL, PluginId, CurrentApiVersion, samplePollID),
					},
				}, {
					Name: "Answer 3 (0)",
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
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", samplePollID, samplePoll.Encode()).Return(&model.AppError{})
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			ExpectedError:        &model.AppError{},
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         commandGenericError,
			ExpectedAttachments:  nil,
		},
		"GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", samplePollID, samplePoll.Encode()).Return(nil)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			ExpectedError:        &model.AppError{},
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         commandGenericError,
			ExpectedAttachments:  nil,
		},
		"Invalid setting": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --unkownOption", trigger),
			ExpectedError:        nil,
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf("Invalid input: Unrecognised poll setting unkownOption"),
			ExpectedAttachments:  nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, samplesiteURL)
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
