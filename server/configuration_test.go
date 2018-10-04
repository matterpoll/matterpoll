package main

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOnConfigurationChange(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI       func(*plugintest.API) *plugintest.API
		Config         *Config
		ExpectedConfig *Config
		ShouldError    bool
	}{
		"Load and save succesfull, with old config": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*Config)
					arg.Trigger = "poll"
				})
				api.On("UnregisterCommand", "", "oldTrigger").Return(nil)
				api.On("RegisterCommand", getCommand("poll")).Return(nil)
				api.On("GetConfig").Return(&model.Config{})
				return api
			},
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "poll"},
			ShouldError:    false,
		},
		"Load and save succesfull, without old config": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*Config)
					arg.Trigger = "poll"
				})
				api.On("RegisterCommand", getCommand("poll")).Return(nil)
				api.On("GetConfig").Return(&model.Config{})
				return api
			},
			Config:         nil,
			ExpectedConfig: &Config{Trigger: "poll"},
			ShouldError:    false,
		},
		"LoadPluginConfiguration fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(errors.New("LoadPluginConfiguration failed"))
				return api
			},
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ShouldError:    true,
		},
		"Load empty trigger": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*Config)
					arg.Trigger = ""
				})
				return api
			},
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ShouldError:    true,
		},
		"UnregisterCommand fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*Config)
					arg.Trigger = "poll"
				})
				api.On("UnregisterCommand", "", "oldTrigger").Return(errors.New("UnregisterCommand failed"))
				return api
			},
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ShouldError:    true,
		},
		"RegisterCommand fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(0).(*Config)
					arg.Trigger = "poll"
				})
				api.On("UnregisterCommand", "", "oldTrigger").Return(nil)
				api.On("RegisterCommand", getCommand("poll")).Return(errors.New("RegisterCommand failed"))
				return api
			},
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ShouldError:    true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, samplesiteURL)
			p.Config = test.Config

			err := p.OnConfigurationChange()
			assert.Equal(test.ExpectedConfig, p.Config)
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

		assert.Equal(t, &Config{}, plugin.getConfiguration())
	})

	t.Run("changing configuration", func(t *testing.T) {
		plugin := &MatterpollPlugin{}
		config1 := &Config{Trigger: "poll"}

		plugin.setConfiguration(config1)

		assert.Equal(t, config1, plugin.getConfiguration())

		config2 := &Config{Trigger: "otherTrigger"}
		plugin.setConfiguration(config2)

		assert.Equal(t, config2, plugin.getConfiguration())
		assert.NotEqual(t, config1, plugin.getConfiguration())
		assert.False(t, plugin.getConfiguration() == config1)
		assert.True(t, plugin.getConfiguration() == config2)
	})

	t.Run("setting same configuration", func(t *testing.T) {
		plugin := &MatterpollPlugin{}
		config := &Config{}
		plugin.setConfiguration(config)

		assert.Panics(t, func() {
			plugin.setConfiguration(config)
		})
	})

	t.Run("clearing configuration", func(t *testing.T) {
		plugin := &MatterpollPlugin{}
		config := &Config{Trigger: "poll"}
		plugin.setConfiguration(config)

		assert.NotPanics(t, func() {
			plugin.setConfiguration(nil)
		})

		assert.NotNil(t, plugin.getConfiguration())
		assert.NotEqual(t, plugin, plugin.getConfiguration())
	})
}
