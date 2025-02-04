package testutils

import (
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/matterpoll/matterpoll/server/utils"
)

// GetLocalizer return an localizer with an empty bundle
func GetLocalizer() *i18n.Localizer {
	return GetBundle().GetServerLocalizer()
}

func GetBundle() *utils.Bundle {
	api := &plugintest.API{}
	api.On("GetBundlePath").Return(".", nil)
	api.On("GetConfig").Return(GetServerConfig())
	api.On("LogWarn", GetMockArgumentsWithType("string", 3)...)
	b, _ := utils.InitBundle(api, ".")
	return b
}
