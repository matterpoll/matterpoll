package plugin

import (
	"fmt"
	"testing"

	"github.com/bouk/monkey"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginExecuteCommand(t *testing.T) {
	trigger := "poll"

	for name, test := range map[string]struct {
		SetupAPI             func(*plugintest.API) *plugintest.API
		Command              string
		ExpectedResponseType string
		ExpectedText         string
		ExpectedAttachments  []*model.SlackAttachment
	}{
		"No argument": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandHelpTextFormat, trigger),
			ExpectedAttachments:  nil,
		},
		"Help text": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s help", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandHelpTextFormat, trigger),
			ExpectedAttachments:  nil,
		},
		"Two arguments": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s \"Question\" \"Just one option\"", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf(commandInputErrorFormat, trigger),
			ExpectedAttachments:  nil,
		},
		"Just question": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", testutils.GetPollID(), testutils.GetPollTwoOptions().EncodeToByte()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\"", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			ExpectedText:         "",
			ExpectedAttachments:  testutils.GetPollTwoOptions().ToPostActions(testutils.GetSiteURL(), PluginId, "John Doe"),
		},
		"With 4 arguments": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", testutils.GetPollID(), testutils.GetPoll().EncodeToByte()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			ExpectedText:         "",
			ExpectedAttachments:  testutils.GetPoll().ToPostActions(testutils.GetSiteURL(), PluginId, "John Doe"),
		},
		"With 4 arguments and settting progress": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				poll := testutils.GetPollWithSettings(poll.PollSettings{Progress: true})

				api.On("KVSet", testutils.GetPollID(), poll.EncodeToByte()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --progress", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			ExpectedAttachments:  testutils.GetPollWithSettings(poll.PollSettings{Progress: true}).ToPostActions(testutils.GetSiteURL(), PluginId, "John Doe"),
		},
		"With 4 arguments and settting anonymous and progress": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				poll := testutils.GetPollWithSettings(poll.PollSettings{Progress: true, Anonymous: true})

				api.On("KVSet", testutils.GetPollID(), poll.EncodeToByte()).Return(nil)
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --anonymous --progress", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			ExpectedText:         "",
			ExpectedAttachments:  testutils.GetPollWithSettings(poll.PollSettings{Progress: true, Anonymous: true}).ToPostActions(testutils.GetSiteURL(), PluginId, "John Doe"),
		},
		"KVSet fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", testutils.GetPollID(), testutils.GetPoll().EncodeToByte()).Return(&model.AppError{})
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         commandGenericError,
			ExpectedAttachments:  nil,
		},
		"GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVSet", testutils.GetPollID(), testutils.GetPoll().EncodeToByte()).Return(nil)
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				return api
			},
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         commandGenericError,
			ExpectedAttachments:  nil,
		},
		"Invalid setting": {
			SetupAPI:             func(api *plugintest.API) *plugintest.API { return api },
			Command:              fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --unkownOption", trigger),
			ExpectedResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			ExpectedText:         fmt.Sprintf("Invalid input: Unrecognised poll setting unkownOption"),
			ExpectedAttachments:  nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, testutils.GetSiteURL())
			p.configuration.Trigger = trigger

			patch1 := monkey.Patch(model.GetMillis, func() int64 { return 1234567890 })
			patch2 := monkey.Patch(model.NewId, func() string { return testutils.GetPollID() })
			defer patch1.Unpatch()
			defer patch2.Unpatch()

			r, err := p.ExecuteCommand(nil, &model.CommandArgs{
				Command: test.Command,
				UserId:  "userID1",
			})

			assert.Nil(err)
			require.NotNil(t, r)

			assert.Equal(model.POST_DEFAULT, r.Type)
			assert.Equal(responseUsername, r.Username)
			assert.Equal(fmt.Sprintf(responseIconURL, testutils.GetSiteURL(), PluginId), r.IconURL)
			assert.Equal(test.ExpectedResponseType, r.ResponseType)
			assert.Equal(test.ExpectedText, r.Text)
			assert.Equal(test.ExpectedAttachments, r.Attachments)
		})
	}
}
