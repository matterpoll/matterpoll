package plugin

import (
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
type configuration struct {
	Trigger        string `json:"trigger"`
	ExperimentalUI bool   `json:"experimentalui"`
}

// OnConfigurationChange loads the plugin configuration, validates it and saves it.
func (p *MatterpollPlugin) OnConfigurationChange() error {
	configuration := new(configuration)
	oldConfiguration := p.getConfiguration()
	p.ServerConfig = p.API.GetConfig()

	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	if configuration.Trigger == "" {
		return errors.New("empty trigger not allowed")
	}

	// This require a loaded i18n bundle
	if p.isActivated() {
		// Update slash command help text
		if oldConfiguration.Trigger != "" {
			if err := p.API.UnregisterCommand("", oldConfiguration.Trigger); err != nil {
				return errors.Wrap(err, "failed to unregister old command")
			}
		}
		if err := p.API.RegisterCommand(p.getCommand(configuration.Trigger)); err != nil {
			return errors.Wrap(err, "failed to register new command")
		}
		// Update bot description
		if err := p.patchBotDescription(); err != nil {
			return errors.Wrap(err, "failed to patch bot description")
		}
	}

	p.setConfiguration(configuration)

	// Emit experimental settings to client if changed
	if oldConfiguration.ExperimentalUI != configuration.ExperimentalUI {
		p.API.PublishWebSocketEvent("configuration_change", map[string]interface{}{
			"experimentalui": configuration.ExperimentalUI,
		}, &model.WebsocketBroadcast{})
	}

	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *MatterpollPlugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}
	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *MatterpollPlugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		panic("setConfiguration called with the existing configuration")
	}
	p.configuration = configuration
}
