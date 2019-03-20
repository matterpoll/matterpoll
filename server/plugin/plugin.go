package plugin

import (
	"fmt"
	"sync"

	"github.com/blang/semver"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/store"
	"github.com/matterpoll/matterpoll/server/store/kvstore"
	"github.com/pkg/errors"
)

// MatterpollPlugin is the object to run the plugin
type MatterpollPlugin struct {
	plugin.MattermostPlugin
	router *mux.Router
	Store  store.Store

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
	ServerConfig  *model.Config
}

const minimumServerVersion = "5.6.0" // TODO: Update to 5.10.0 once it's available

// OnActivate ensures a configuration is set and initializes the API
func (p *MatterpollPlugin) OnActivate() error {
	if err := p.checkServerVersion(); err != nil {
		return err
	}

	store, err := kvstore.NewStore(p.API, PluginVersion)
	if err != nil {
		return err
	}
	p.Store = store

	p.router = p.InitAPI()
	return nil
}

// OnDeactivate unregisters the command
func (p *MatterpollPlugin) OnDeactivate() error {
	err := p.API.UnregisterCommand("", p.getConfiguration().Trigger)
	if err != nil {
		return errors.Wrap(err, "failed to dectivate command")
	}
	return nil
}

// checkServerVersion checks Mattermost Server has at least the required version
func (p *MatterpollPlugin) checkServerVersion() error {
	serverVersion, err := semver.Parse(p.API.GetServerVersion())
	if err != nil {
		return errors.Wrap(err, "failed to parse server version")
	}

	r := semver.MustParseRange(">=" + minimumServerVersion)
	if !r(serverVersion) {
		return fmt.Errorf("this plugin requires Mattermost v%s or later", minimumServerVersion)
	}

	return nil
}

// ConvertUserIDToDisplayName returns the display name to a given user ID
func (p *MatterpollPlugin) ConvertUserIDToDisplayName(userID string) (string, *model.AppError) {
	user, err := p.API.GetUser(userID)
	if err != nil {
		return "", err
	}
	displayName := user.GetDisplayName(model.SHOW_USERNAME)
	displayName = "@" + displayName
	return displayName, nil
}

// ConvertCreatorIDToDisplayName returns the display name to a given user ID of a poll creator
func (p *MatterpollPlugin) ConvertCreatorIDToDisplayName(creatorID string) (string, *model.AppError) {
	user, err := p.API.GetUser(creatorID)
	if err != nil {
		return "", err
	}
	displayName := user.GetDisplayName(model.SHOW_NICKNAME_FULLNAME)
	return displayName, nil
}

// HasPermission checks if a given user has the permission to end or delete a given poll
func (p *MatterpollPlugin) HasPermission(poll *poll.Poll, issuerID string) (bool, *model.AppError) {
	if issuerID == poll.Creator {
		return true, nil
	}

	user, appErr := p.API.GetUser(issuerID)
	if appErr != nil {
		return false, appErr
	}
	if user.IsInRole(model.SYSTEM_ADMIN_ROLE_ID) {
		return true, nil
	}
	return false, nil
}

func (p *MatterpollPlugin) SendEphemeralPost(channelID, userID, message string) {
	// This is mostly taken from https://github.com/mattermost/mattermost-server/blob/master/app/command.go#L304
	ephemeralPost := &model.Post{}
	ephemeralPost.ChannelId = channelID
	ephemeralPost.UserId = userID
	ephemeralPost.Message = message
	ephemeralPost.AddProp("override_username", responseUsername)
	ephemeralPost.AddProp("override_icon_url", fmt.Sprintf(responseIconURL, *p.ServerConfig.ServiceSettings.SiteURL, PluginId))
	ephemeralPost.AddProp("from_webhook", "true")
	_ = p.API.SendEphemeralPost(userID, ephemeralPost)
}
