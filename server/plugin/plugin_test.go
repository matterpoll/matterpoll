package plugin

import (
	"errors"
	"path/filepath"
	"testing"

	"bou.ke/monkey"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"

	"github.com/matterpoll/matterpoll/server/store"
	"github.com/matterpoll/matterpoll/server/store/kvstore"
	"github.com/matterpoll/matterpoll/server/store/mockstore"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func setupTestPlugin(_ *testing.T, api *plugintest.API, store *mockstore.Store) *MatterpollPlugin { //nolint:interfacer
	p := &MatterpollPlugin{
		ServerConfig: testutils.GetServerConfig(),
		getIconData:  getIconDataMock,
	}
	p.setConfiguration(&configuration{
		Trigger:        "poll",
		ExperimentalUI: true,
	})

	p.SetAPI(api)
	p.botUserID = testutils.GetBotUserID()
	p.bundle = i18n.NewBundle(language.English)
	p.Store = store
	p.router = p.InitAPI()
	p.setActivated(true)

	return p
}

func getIconDataMock() (string, error) {
	return "someIconData", nil
}

func TestPluginOnActivate(t *testing.T) {
	bot := &model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
	}

	command := &model.Command{
		Trigger:              "poll",
		AutoComplete:         true,
		AutoCompleteDesc:     "Create a poll",
		AutoCompleteHint:     `"[Question]" "[Answer 1]" "[Answer 2]"...`,
		AutocompleteIconData: "someIconData",
	}

	for name, test := range map[string]struct {
		SetupAPI     func(*plugintest.API) *plugintest.API
		SetupHelpers func(*plugintest.Helpers) *plugintest.Helpers
		ShouldError  bool
	}{
		// server version tests
		"all fine": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				path, err := filepath.Abs("../..")
				require.Nil(t, err)
				api.On("GetBundlePath").Return(path, nil)
				api.On("PatchBot", testutils.GetBotUserID(), &model.BotPatch{Description: &botDescription.Other}).Return(nil, nil)
				api.On("RegisterCommand", command).Return(nil)
				return api
			},
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot, mock.AnythingOfType("plugin.EnsureBotOption")).Return(testutils.GetBotUserID(), nil)
				return helpers
			},
			ShouldError: false,
		},
		// i18n bundle tests
		"GetBundlePath fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return("", errors.New(""))
				return api
			},
			ShouldError: true,
		},
		"i18n directory doesn't exist ": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return("/tmp", nil)
				return api
			},
			ShouldError: true,
		},
		// Bot tests
		"EnsureBot fails ": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				path, err := filepath.Abs("../..")
				require.Nil(t, err)
				api.On("GetBundlePath").Return(path, nil)
				return api
			},
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot, mock.AnythingOfType("plugin.EnsureBotOption")).Return("", &model.AppError{})
				return helpers
			},
			ShouldError: true,
		},
		"patch bot description fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				path, err := filepath.Abs("../..")
				require.Nil(t, err)
				api.On("GetBundlePath").Return(path, nil)
				api.On("PatchBot", testutils.GetBotUserID(), &model.BotPatch{Description: &botDescription.Other}).Return(nil, &model.AppError{})
				return api
			},
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot, mock.AnythingOfType("plugin.EnsureBotOption")).Return(testutils.GetBotUserID(), nil)
				return helpers
			},
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)

			helpers := &plugintest.Helpers{}
			if test.SetupHelpers != nil {
				helpers = test.SetupHelpers(helpers)
				defer helpers.AssertExpectations(t)
			}

			patch := monkey.Patch(kvstore.NewStore, func(plugin.API, string) (store.Store, error) {
				return &mockstore.Store{}, nil
			})
			defer patch.Unpatch()

			siteURL := testutils.GetSiteURL()
			defaultClientLocale := "en"
			p := &MatterpollPlugin{
				ServerConfig: &model.Config{
					LocalizationSettings: model.LocalizationSettings{
						DefaultClientLocale: &defaultClientLocale,
					},
					ServiceSettings: model.ServiceSettings{
						SiteURL: &siteURL,
					},
				},
				getIconData: getIconDataMock,
			}
			p.setConfiguration(&configuration{
				Trigger: "poll",
			})
			p.SetAPI(api)
			p.SetHelpers(helpers)
			err := p.OnActivate()

			if test.ShouldError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
	t.Run("NewStore() fails", func(t *testing.T) {
		api := &plugintest.API{}
		defer api.AssertExpectations(t)

		patch := monkey.Patch(kvstore.NewStore, func(plugin.API, string) (store.Store, error) {
			return nil, &model.AppError{}
		})
		defer patch.Unpatch()

		siteURL := testutils.GetSiteURL()
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
		err := p.OnActivate()

		assert.NotNil(t, err)
	})
	t.Run("SiteURL not set", func(t *testing.T) {
		api := &plugintest.API{}
		defer api.AssertExpectations(t)

		patch := monkey.Patch(kvstore.NewStore, func(plugin.API, string) (store.Store, error) {
			return nil, &model.AppError{}
		})
		defer patch.Unpatch()

		p := &MatterpollPlugin{
			ServerConfig: &model.Config{
				ServiceSettings: model.ServiceSettings{
					SiteURL: nil,
				},
			},
		}
		p.setConfiguration(&configuration{
			Trigger: "poll",
		})
		p.SetAPI(api)
		err := p.OnActivate()

		assert.NotNil(t, err)
	})
}

func TestPluginOnDeactivate(t *testing.T) {
	p := setupTestPlugin(t, &plugintest.API{}, &mockstore.Store{})

	err := p.OnDeactivate()
	assert.Nil(t, err)
}
