package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/undefinedlabs/go-mpatch"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/matterpoll/matterpoll/server/store"
	"github.com/matterpoll/matterpoll/server/store/kvstore"
	"github.com/matterpoll/matterpoll/server/store/mockstore"
	"github.com/matterpoll/matterpoll/server/utils"
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
	api.On("GetConfig").Return(testutils.GetServerConfig()).Maybe()
	api.On("GetBundlePath").Return(".", nil)
	p.bundle, _ = utils.InitBundle(api, ".")
	p.Store = store
	p.router = p.InitAPI()
	p.setActivated(true)

	return p
}

func getIconDataMock() (string, error) {
	return "someIconData", nil
}

func TestPluginOnActivate(t *testing.T) {
	command := &model.Command{
		Trigger:              "poll",
		AutoComplete:         true,
		AutoCompleteDesc:     "Create a poll",
		AutoCompleteHint:     `"[Question]" "[Answer 1]" "[Answer 2]"...`,
		AutocompleteIconData: "someIconData",
	}

	for name, test := range map[string]struct {
		SetupAPI       func(*plugintest.API) *plugintest.API
		SetupPluginAPI func(*pluginapi.Client) (*pluginapi.Client, []*mpatch.Patch)
		ShouldError    bool
	}{
		// server version tests
		"all fine": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("PatchBot", testutils.GetBotUserID(), &model.BotPatch{Description: &botDescription.Other}).Return(nil, nil)
				api.On("RegisterCommand", command).Return(nil)
				return api
			},
			SetupPluginAPI: func(client *pluginapi.Client) (*pluginapi.Client, []*mpatch.Patch) {
				p1, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client.Bot), "EnsureBot", func(*pluginapi.BotService, *model.Bot, ...pluginapi.EnsureBotOption) (string, error) {
					return testutils.GetBotUserID(), nil
				})
				require.NoError(t, err)

				return client, []*mpatch.Patch{p1}
			},
			ShouldError: false,
		},
		// i18n bundle tests
		"GetBundlePath fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return("", errors.New(""))
				return api
			},
			SetupPluginAPI: func(client *pluginapi.Client) (*pluginapi.Client, []*mpatch.Patch) {
				p1, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client.Bot), "EnsureBot", func(*pluginapi.BotService, *model.Bot, ...pluginapi.EnsureBotOption) (string, error) {
					return testutils.GetBotUserID(), nil
				})
				require.NoError(t, err)

				return client, []*mpatch.Patch{p1}
			},
			ShouldError: true,
		},
		// Bot tests
		"EnsureBot fails ": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				return api
			},
			SetupPluginAPI: func(client *pluginapi.Client) (*pluginapi.Client, []*mpatch.Patch) {
				p1, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client.Bot), "EnsureBot", func(*pluginapi.BotService, *model.Bot, ...pluginapi.EnsureBotOption) (string, error) {
					return "", errors.New("")
				})
				require.NoError(t, err)

				return client, []*mpatch.Patch{p1}
			},
			ShouldError: true,
		},
		"patch bot description fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("PatchBot", testutils.GetBotUserID(), &model.BotPatch{Description: &botDescription.Other}).Return(nil, &model.AppError{})
				return api
			},
			SetupPluginAPI: func(client *pluginapi.Client) (*pluginapi.Client, []*mpatch.Patch) {
				p1, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client.Bot), "EnsureBot", func(*pluginapi.BotService, *model.Bot, ...pluginapi.EnsureBotOption) (string, error) {
					return testutils.GetBotUserID(), nil
				})
				require.NoError(t, err)

				return client, []*mpatch.Patch{p1}
			},
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "")
			require.NoError(t, err)

			defer os.RemoveAll(dir)

			// Create assets/i18n dir
			i18nDir := filepath.Join(dir, "assets", "i18n")
			err = os.MkdirAll(i18nDir, 0700)
			require.NoError(t, err)

			file := filepath.Join(i18nDir, "active.de.json")
			content := []byte("{}")
			err = os.WriteFile(file, content, 0600)
			require.NoError(t, err)

			api := test.SetupAPI(&plugintest.API{})
			api.On("GetBundlePath").Return(dir, nil)
			api.On("GetConfig").Return(testutils.GetServerConfig()).Maybe()
			defer api.AssertExpectations(t)

			patch1, _ := mpatch.PatchMethod(kvstore.NewStore, func(plugin.API, string) (store.Store, error) {
				return &mockstore.Store{}, nil
			})
			defer func() { require.NoError(t, patch1.Unpatch()) }()

			// Setup pluginapi client
			mClient := pluginapi.NewClient(api, &plugintest.Driver{})
			patch2, err := mpatch.PatchMethod(
				pluginapi.NewClient,
				func(plugin.API, plugin.Driver) *pluginapi.Client { return mClient },
			)
			require.NoError(t, err)
			defer func() { require.NoError(t, patch2.Unpatch()) }()

			if test.SetupPluginAPI != nil {
				_, patches := test.SetupPluginAPI(mClient)
				t.Cleanup(func() {
					for _, p := range patches {
						require.NoError(t, p.Unpatch())
					}
				})
			}

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
			err = p.OnActivate()

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

		patch, _ := mpatch.PatchMethod(kvstore.NewStore, func(plugin.API, string) (store.Store, error) {
			return nil, &model.AppError{}
		})
		defer func() { require.NoError(t, patch.Unpatch()) }()

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

		patch, _ := mpatch.PatchMethod(kvstore.NewStore, func(plugin.API, string) (store.Store, error) {
			return nil, &model.AppError{}
		})
		defer func() { require.NoError(t, patch.Unpatch()) }()

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

func TestConvertCreatorIDToDisplayName(t *testing.T) {
	user := &model.User{
		Id:        "userID1",
		Username:  "user1",
		FirstName: "John",
		LastName:  "Doe",
	}
	for name, test := range map[string]struct {
		UserID              string
		SettingShowFullName bool
		SetupAPI            func(*plugintest.API) *plugintest.API
		ShouldError         bool
		ExpectedName        string
	}{
		"all fine, ShowFullName is true": {
			UserID:              user.Id,
			SettingShowFullName: true,
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", user.Id).Return(user, nil)
				return api
			},
			ShouldError:  false,
			ExpectedName: user.GetFullName(),
		},
		"all fine, ShowFullName is false": {
			UserID:              user.Id,
			SettingShowFullName: false,
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", user.Id).Return(user, nil)
				return api
			},
			ShouldError:  false,
			ExpectedName: user.Username,
		},
		"GetUser fails": {
			UserID:              user.Id,
			SettingShowFullName: true,
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", user.Id).Return(nil, &model.AppError{})
				return api
			},
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)

			p := setupTestPlugin(t, api, &mockstore.Store{})
			fn := test.SettingShowFullName
			p.ServerConfig.PrivacySettings.ShowFullName = &fn

			name, err := p.ConvertCreatorIDToDisplayName(test.UserID)

			if test.ShouldError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.ExpectedName, name)
			}
		})
	}
}
