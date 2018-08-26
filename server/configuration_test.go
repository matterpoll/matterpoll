package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOnConfigurationChange(t *testing.T) {
	api := &plugintest.API{}
	api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Config")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*Config)
		arg.Trigger = "poll"
	})

	api.On("RegisterCommand", &model.Command{
		Trigger:          "poll",
		DisplayName:      "Matterpoll",
		Description:      "Polling feature by https://github.com/matterpoll/matterpoll",
		AutoComplete:     true,
		AutoCompleteDesc: "Create a poll",
		AutoCompleteHint: "[Question] [Answer 1] [Answer 2]...",
	}).Return(nil)

	defer api.AssertExpectations(t)
	p := &MatterpollPlugin{}
	p.SetAPI(api)

	err := p.OnConfigurationChange()
	assert.Nil(t, err)
}
