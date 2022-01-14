package testutils

import (
	"github.com/mattermost/mattermost-plugin-api/i18n"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
)

// GetLocalizer return an localizer with an empty bundle
func GetLocalizer() *i18n.Localizer {
	api := &plugintest.API{}
	api.On("GetBundlePath").Return(".", nil)
	api.On("GetConfig").Return(GetServerConfig())
	b, _ := i18n.InitBundle(api, ".")
	return b.GetServerLocalizer()
}
