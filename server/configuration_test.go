package main

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOnConfigurationChange(t *testing.T) {
	api1 := &plugintest.API{}
	api1.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*Config)
		arg.Trigger = "poll"
	})
	api1.On("UnregisterCommand", "", "oldTrigger").Return(nil)
	api1.On("RegisterCommand", getCommand("poll")).Return(nil)
	defer api1.AssertExpectations(t)

	api2 := &plugintest.API{}
	api2.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*Config)
		arg.Trigger = "poll"
	})
	api2.On("RegisterCommand", getCommand("poll")).Return(nil)
	defer api2.AssertExpectations(t)

	err3 := errors.New("LoadPluginConfiguration failed")
	api3 := &plugintest.API{}
	api3.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(err3)
	defer api3.AssertExpectations(t)

	api4 := &plugintest.API{}
	api4.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*Config)
		arg.Trigger = ""
	})
	defer api4.AssertExpectations(t)

	err5 := errors.New("UnregisterCommand failed")
	api5 := &plugintest.API{}
	api5.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*Config)
		arg.Trigger = "poll"
	})
	api5.On("UnregisterCommand", "", "oldTrigger").Return(err5)
	defer api5.AssertExpectations(t)

	err6 := errors.New("RegisterCommand failed")
	api6 := &plugintest.API{}
	api6.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*Config)
		arg.Trigger = "poll"
	})
	api6.On("UnregisterCommand", "", "oldTrigger").Return(nil)
	api6.On("RegisterCommand", getCommand("poll")).Return(err6)
	defer api6.AssertExpectations(t)

	for name, test := range map[string]struct {
		API            *plugintest.API
		Config         *Config
		ExpectedConfig *Config
		ExpectedError  error
	}{
		"Load and save succesfull, with old config": {
			API:            api1,
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "poll"},
			ExpectedError:  nil,
		},
		"Load and save succesfull, without old config": {
			API:            api2,
			Config:         nil,
			ExpectedConfig: &Config{Trigger: "poll"},
			ExpectedError:  nil,
		},
		"LoadPluginConfiguration fails": {
			API:            api3,
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ExpectedError:  err3,
		},
		"Load empty trigger": {
			API:            api4,
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ExpectedError:  errors.New("Empty trigger not allowed"),
		},
		"UnregisterCommand fails": {
			API:            api5,
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ExpectedError:  err5,
		},
		"RegisterCommand fails": {
			API:            api6,
			Config:         &Config{Trigger: "oldTrigger"},
			ExpectedConfig: &Config{Trigger: "oldTrigger"},
			ExpectedError:  err6,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			idGen := new(MockPollIDGenerator)
			p := &MatterpollPlugin{
				idGen:  idGen,
				Config: test.Config,
			}
			p.SetAPI(test.API)

			err := p.OnConfigurationChange()
			assert.Equal(test.ExpectedError, err)
			assert.Equal(test.ExpectedConfig, p.Config)
		})
	}
}
