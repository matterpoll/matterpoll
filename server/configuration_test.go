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
		ExpectedError  error
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
			ExpectedError:  nil,
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
			ExpectedError:  nil,
		},
		"LoadPluginConfiguration fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(errors.New("LoadPluginConfiguration failed"))
				return api
			},
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ExpectedError:  errors.New("LoadPluginConfiguration failed"),
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
			ExpectedError:  errors.New("Empty trigger not allowed"),
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
			ExpectedError:  errors.New("UnregisterCommand failed"),
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
			ExpectedError:  errors.New("RegisterCommand failed"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(t, api, samplesiteURL)
			p.Config = test.Config

			err := p.OnConfigurationChange()
			assert.Equal(test.ExpectedError, err)
			assert.Equal(test.ExpectedConfig, p.Config)
		})
	}
}
