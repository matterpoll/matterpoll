package main

import (
	"fmt"
	"testing"

	"github.com/bouk/monkey"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPluginExecuteCommand(t *testing.T) {
	siteURL := "https://example.org/"
	trigger := "poll"
	idGen := new(MockPollIDGenerator)
	poll := Poll{
		CreatedAt:         1234567890,
		Creator:           "userID1",
		DataSchemaVersion: "v1",
		Question:          "Question",
		AnswerOptions: []*AnswerOption{
			{Answer: "Yes"},
			{Answer: "No"},
		},
	}

	api1 := &plugintest.API{}
	api1.On("KVSet", idGen.NewID(), poll.Encode()).Return(nil)
	api1.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	api1.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()
	defer api1.AssertExpectations(t)

	api2 := &plugintest.API{}
	api2.On("KVSet", idGen.NewID(), samplePoll.Encode()).Return(nil)
	api2.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
	api2.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()
	defer api2.AssertExpectations(t)

	api3 := &plugintest.API{}
	api3.On("KVSet", idGen.NewID(), samplePoll.Encode()).Return(&model.AppError{})
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("KVSet", idGen.NewID(), samplePoll.Encode()).Return(nil)
	api4.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, &model.AppError{})
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
				Actions: []*model.PostAction{{
					Name: "Yes",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
					},
				}, {
					Name: "No",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
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
				Actions: []*model.PostAction{{
					Name: "Answer 1",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
					},
				}, {
					Name: "Answer 2",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
					},
				}, {
					Name: "Answer 3",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/2", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginId, CurrentApiVersion, idGen.NewID()),
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
			patch := monkey.Patch(model.GetMillis, func() int64 { return 1234567890 })
			defer patch.Unpatch()

			idGen := new(MockPollIDGenerator)
			p := &MatterpollPlugin{
				idGen: idGen,
				Config: &Config{
					Trigger: trigger,
				},
			}
			p.SetAPI(test.API)

			r, err := p.ExecuteCommand(nil, &model.CommandArgs{
				Command: test.Command,
				SiteURL: siteURL,
				UserId:  "userID1",
			})

			assert.Equal(test.ExpectedError, err)
			assert.NotNil(r)

			assert.Equal(model.POST_DEFAULT, r.Type)
			assert.Equal(responseUsername, r.Username)
			assert.Equal(responseIconURL, r.IconURL)
			assert.Equal(test.ExpectedResponseType, r.ResponseType)
			assert.Equal(test.ExpectedText, r.Text)
			assert.Equal(test.ExpectedAttachments, r.Attachments)
		})
	}
}
