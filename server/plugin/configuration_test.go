package plugin

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/matterpoll/matterpoll/server/store/mockstore"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestOnConfigurationChange(t *testing.T) {
	command := &model.Command{
		Trigger:          "poll",
		AutoComplete:     true,
		AutoCompleteDesc: "Create a poll",
		AutoCompleteHint: `"[Question]" "[Answer 1]" "[Answer 2]"...`,
	}

	botPatch := &model.BotPatch{
		Description: &botDescription.Other,
	}

	for name, test := range map[string]struct {
		SetupAPI              func(*plugintest.API) *plugintest.API
		Configuration         *configuration
		ExpectedConfiguration *configuration
		ShouldError           bool
	}{
		"Load and save successful, with old configuration": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetConfig").Return(testutils.GetServerConfig())
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*plugin.configuration")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*configuration)
					arg.Trigger = "poll"
					arg.ExperimentalUI = true
				})
				api.On("UnregisterCommand", "", "oldTrigger").Return(nil)
				api.On("RegisterCommand", command).Return(nil)
				api.On("PatchBot", testutils.GetBotUserID(), botPatch).Return(nil, nil)
				api.On("PublishWebSocketEvent", "configuration_change", map[string]interface{}{
					"experimentalui": true,
				}, &model.WebsocketBroadcast{}).Return()
				return api
			},
			Configuration:         &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ExpectedConfiguration: &configuration{Trigger: "poll", ExperimentalUI: true},
			ShouldError:           false,
		},
		"Load and save successful, without old configuration": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetConfig").Return(testutils.GetServerConfig())
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*plugin.configuration")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*configuration)
					arg.Trigger = "poll"
					arg.ExperimentalUI = true
				})
				api.On("RegisterCommand", command).Return(nil)
				api.On("PatchBot", testutils.GetBotUserID(), botPatch).Return(nil, nil)
				api.On("PublishWebSocketEvent", "configuration_change", map[string]interface{}{
					"experimentalui": true,
				}, &model.WebsocketBroadcast{}).Return()
				return api
			},
			Configuration:         nil,
			ExpectedConfiguration: &configuration{Trigger: "poll", ExperimentalUI: true},
			ShouldError:           false,
		},
		"LoadPluginConfiguration fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetConfig").Return(testutils.GetServerConfig())
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*plugin.configuration")).Return(errors.New(""))
				return api
			},
			Configuration:         &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ExpectedConfiguration: &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ShouldError:           true,
		},
		"Load empty trigger": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetConfig").Return(testutils.GetServerConfig())
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*plugin.configuration")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*configuration)
					arg.Trigger = ""
				})
				return api
			},
			Configuration:         &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ExpectedConfiguration: &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ShouldError:           true,
		},
		"UnregisterCommand fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetConfig").Return(testutils.GetServerConfig())
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*plugin.configuration")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*configuration)
					arg.Trigger = "poll"
				})
				api.On("UnregisterCommand", "", "oldTrigger").Return(errors.New(""))
				return api
			},
			Configuration:         &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ExpectedConfiguration: &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ShouldError:           true,
		},
		"RegisterCommand fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetConfig").Return(testutils.GetServerConfig())
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*plugin.configuration")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*configuration)
					arg.Trigger = "poll"
				})
				api.On("UnregisterCommand", "", "oldTrigger").Return(nil)
				api.On("RegisterCommand", command).Return(errors.New(""))
				return api
			},
			Configuration:         &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ExpectedConfiguration: &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ShouldError:           true,
		},
		"patchBotDescription fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetConfig").Return(testutils.GetServerConfig())
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*plugin.configuration")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*configuration)
					arg.Trigger = "poll"
				})
				api.On("UnregisterCommand", "", "oldTrigger").Return(nil)
				api.On("RegisterCommand", command).Return(nil)
				api.On("PatchBot", testutils.GetBotUserID(), botPatch).Return(nil, &model.AppError{})
				return api
			},
			Configuration:         &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ExpectedConfiguration: &configuration{Trigger: "oldTrigger", ExperimentalUI: false},
			ShouldError:           true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, &mockstore.Store{})
			p.setConfiguration(test.Configuration)

			err := p.OnConfigurationChange()
			assert.Equal(test.ExpectedConfiguration, p.getConfiguration())
			if test.ShouldError {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
		})
	}
}

func TestConfiguration(t *testing.T) {
	t.Run("null configuration", func(t *testing.T) {
		plugin := &MatterpollPlugin{}

		assert.Equal(t, &configuration{}, plugin.getConfiguration())
	})

	t.Run("changing configuration", func(t *testing.T) {
		plugin := &MatterpollPlugin{}
		config1 := &configuration{Trigger: "poll"}

		plugin.setConfiguration(config1)

		assert.Equal(t, config1, plugin.getConfiguration())

		config2 := &configuration{Trigger: "otherTrigger"}
		plugin.setConfiguration(config2)

		assert.Equal(t, config2, plugin.getConfiguration())
		assert.NotEqual(t, config1, plugin.getConfiguration())
		assert.False(t, plugin.getConfiguration() == config1)
		assert.True(t, plugin.getConfiguration() == config2)
	})

	t.Run("setting same configuration", func(t *testing.T) {
		plugin := &MatterpollPlugin{}
		config := &configuration{}
		plugin.setConfiguration(config)

		assert.Panics(t, func() {
			plugin.setConfiguration(config)
		})
	})

	t.Run("clearing configuration", func(t *testing.T) {
		plugin := &MatterpollPlugin{}
		config := &configuration{Trigger: "poll"}
		plugin.setConfiguration(config)

		assert.NotPanics(t, func() {
			plugin.setConfiguration(nil)
		})

		assert.NotNil(t, plugin.getConfiguration())
		assert.NotEqual(t, plugin, plugin.getConfiguration())
	})
}
