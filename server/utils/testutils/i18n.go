package testutils

import (
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/matterpoll/matterpoll/server/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// GetLocalizer return an localizer with an empty bundle
func GetLocalizer() *i18n.Localizer {
	return GetBundle().GetServerLocalizer()
}

func GetBundle() *utils.Bundle {
	api := &plugintest.API{}
	api.On("GetBundlePath").Return(".", nil)
	api.On("GetConfig").Return(GetServerConfig())
	b, _ := utils.InitBundle(api, ".")
	return b
}
