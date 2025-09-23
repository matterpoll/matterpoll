package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"

	"github.com/matterpoll/matterpoll/server/store/mockstore"
	"github.com/matterpoll/matterpoll/server/utils"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func setupTestPlugin(_ *testing.T, api *plugintest.API, store *mockstore.Store) *MatterpollPlugin {
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
	t.Run("SiteURL not set", func(t *testing.T) {
		api := &plugintest.API{}
		defer api.AssertExpectations(t)

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
