package plugin

import (
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"

	root "github.com/matterpoll/matterpoll"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/store"
	"github.com/matterpoll/matterpoll/server/store/kvstore"
	"github.com/matterpoll/matterpoll/server/utils"
)

// MatterpollPlugin is the object to run the plugin
type MatterpollPlugin struct {
	plugin.MattermostPlugin
	botUserID string
	bundle    *utils.Bundle
	router    *mux.Router
	Store     store.Store

	// activated is used to track whether or not OnActivate has initialized the plugin state.
	activated bool

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
	ServerConfig  *model.Config

	// getIconData provides access to command.GetIconData in a way that is mockable for unit testing.
	getIconData func() (string, error)

	pf poll.Factory
}

var botDescription = &i18n.Message{
	ID:    "bot.description",
	Other: "Poll Bot",
}

const (
	botUserName    = "matterpoll"
	botDisplayName = "Matterpoll"

	// MatterpollPostType is post_type of posts generated by Matterpoll
	MatterpollPostType = "custom_matterpoll"
)

func NewMatterpollPlugin() *MatterpollPlugin {
	plugin := &MatterpollPlugin{}

	getIconData := func() (string, error) {
		return command.GetIconData(plugin.API, "assets/logo_dark-bg.svg")
	}
	plugin.getIconData = getIconData

	return plugin
}

// OnActivate ensures a configuration is set and initializes the API
func (p *MatterpollPlugin) OnActivate() error {
	if p.ServerConfig.ServiceSettings.SiteURL == nil {
		return errors.New("siteURL is not set. Please set a siteURL and restart the plugin")
	}

	var err error
	p.Store, err = kvstore.NewStore(p.API, root.Manifest.Version)
	if err != nil {
		return errors.Wrap(err, "failed to create store")
	}

	p.bundle, err = utils.InitBundle(p.API, filepath.Join("assets", "i18n"))
	if err != nil {
		return errors.Wrap(err, "failed to init localisation bundle")
	}

	bot := &model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
	}
	pluginAPI := pluginapi.NewClient(p.API, p.Driver)
	botUserID, err := pluginAPI.Bot.EnsureBot(bot, pluginapi.ProfileImagePath("assets/logo_dark-bg.png"))
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot user")
	}
	p.botUserID = botUserID

	if err = p.patchBotDescription(); err != nil {
		return errors.Wrap(err, "failed to patch bot description")
	}

	command, err := p.getCommand(p.getConfiguration().Trigger)
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	if err := p.API.RegisterCommand(command); err != nil {
		return errors.Wrap(err, "failed to register  command")
	}

	p.router = p.InitAPI()

	p.setActivated(true)

	return nil
}

// OnDeactivate marks the plugin as deactivated
func (p *MatterpollPlugin) OnDeactivate() error {
	p.setActivated(false)

	return nil
}

func (p *MatterpollPlugin) setActivated(activated bool) {
	p.activated = activated
}

func (p *MatterpollPlugin) isActivated() bool {
	return p.activated
}

// patchBotDescription updates the bot description based on the servers local
func (p *MatterpollPlugin) patchBotDescription() error {
	publicLocalizer := p.bundle.GetServerLocalizer()
	description := p.bundle.LocalizeDefaultMessage(publicLocalizer, botDescription)

	// Update description with server local
	botPatch := &model.BotPatch{
		Description: &description,
	}
	if _, appErr := p.API.PatchBot(p.botUserID, botPatch); appErr != nil {
		return errors.Wrap(appErr, "failed to patch bot")
	}

	return nil
}

// ConvertUserIDToDisplayName returns the display name to a given user ID
func (p *MatterpollPlugin) ConvertUserIDToDisplayName(userID string) (string, *model.AppError) {
	user, err := p.API.GetUser(userID)
	if err != nil {
		return "", err
	}
	displayName := user.GetDisplayName(model.ShowUsername)
	displayName = "@" + displayName
	return displayName, nil
}

// ConvertCreatorIDToDisplayName returns the display name to a given user ID of a poll creator
func (p *MatterpollPlugin) ConvertCreatorIDToDisplayName(creatorID string) (string, *model.AppError) {
	user, err := p.API.GetUser(creatorID)
	if err != nil {
		return "", err
	}
	setting := p.ServerConfig.PrivacySettings.ShowFullName
	// Need to check if settings value is nil pointer, because PrivacySettings.ShowFullName
	// can be nil pointer when ShowFullName setting is false.
	if setting == nil || !*setting {
		return user.GetDisplayName(model.ShowUsername), nil
	}
	return user.GetDisplayName(model.ShowNicknameFullName), nil
}

// CanManagePoll checks if a given user has the permission to manage i.e. end or delete a given poll
func (p *MatterpollPlugin) CanManagePoll(poll *poll.Poll, issuerID string) (bool, *model.AppError) {
	if issuerID == poll.Creator {
		return true, nil
	}

	user, appErr := p.API.GetUser(issuerID)
	if appErr != nil {
		return false, appErr
	}
	if user.IsInRole(model.SystemAdminRoleId) {
		return true, nil
	}
	return false, nil
}

// SendEphemeralPost sends an ephemeral post to a user as the bot account
func (p *MatterpollPlugin) SendEphemeralPost(channelID, userID, rootID, message string) {
	ephemeralPost := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		RootId:    rootID,
		Message:   message,
	}
	_ = p.API.SendEphemeralPost(userID, ephemeralPost)
}
