package plugin

import (
	"testing"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/matterpoll/matterpoll/server/store"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestPlugin(t *testing.T, api *plugintest.API, siteURL string) *MatterpollPlugin {
	p := &MatterpollPlugin{
		ServerConfig: &model.Config{
			ServiceSettings: model.ServiceSettings{
				SiteURL: &siteURL,
			},
		},
	}
	p.setConfiguration(&configuration{
		Trigger: "poll",
	})

	p.SetAPI(api)
	// TODO: Dont hardcode the key
	api.On("KVGet", "version").Return([]byte(PluginVersion), nil)
	store, err := store.NewStore(api)
	require.Nil(t, err)
	require.NotNil(t, store)
	p.Store = store
	p.router = p.InitAPI()

	return p
}

func TestPluginOnActivate(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI    func(*plugintest.API) *plugintest.API
		ShouldError bool
	}{
		"greater minor version than minimumServerVersion": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				m := semver.MustParse(minimumServerVersion)
				m.Minor++
				m.Patch = 0

				api.On("GetServerVersion").Return(m.String())
				// TODO: Dont hardcode the key
				api.On("KVGet", "version").Return([]byte(PluginVersion), nil)
				return api
			},
			ShouldError: false,
		},
		"same version as minimumServerVersion": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetServerVersion").Return(minimumServerVersion)
				// TODO: Dont hardcode the key
				api.On("KVGet", "version").Return([]byte(PluginVersion), nil)
				return api
			},
			ShouldError: false,
		},
		"lesser minor version than minimumServerVersion": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				m := semver.MustParse(minimumServerVersion)
				if m.Minor == 0 {
					m.Major--
					m.Minor = 0
					m.Patch = 0
				} else {
					m.Minor--
					m.Patch = 0
				}
				api.On("GetServerVersion").Return(m.String())
				return api
			},
			ShouldError: true,
		},
		"GetServerVersion not implemented, returns empty string": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetServerVersion").Return("")
				return api
			},
			ShouldError: true,
		},
		"NewStore() fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetServerVersion").Return(minimumServerVersion)
				// TODO: Dont hardcode the key
				api.On("KVGet", "version").Return([]byte{}, &model.AppError{})
				return api
			},
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)

			p := &MatterpollPlugin{}
			p.setConfiguration(&configuration{
				Trigger: "poll",
			})
			p.SetAPI(api)
			err := p.OnActivate()

			if test.ShouldError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestPluginOnDeactivate(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		p := setupTestPlugin(t, api, testutils.GetSiteURL())
		api.On("UnregisterCommand", "", p.getConfiguration().Trigger).Return(nil)
		defer api.AssertExpectations(t)

		err := p.OnDeactivate()
		assert.Nil(t, err)
	})

	t.Run("UnregisterCommand fails", func(t *testing.T) {
		api := &plugintest.API{}
		p := setupTestPlugin(t, api, testutils.GetSiteURL())
		api.On("UnregisterCommand", "", p.getConfiguration().Trigger).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		err := p.OnDeactivate()
		assert.NotNil(t, err)
	})
}

func GetMockArgumentsWithType(typeString string, num int) []interface{} {
	ret := make([]interface{}, num)
	for i := 0; i < len(ret); i++ {
		ret[i] = mock.AnythingOfTypeArgument(typeString)
	}
	return ret
}
