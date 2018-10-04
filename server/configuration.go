package main

import (
	"github.com/pkg/errors"
)

// Config captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
type Config struct {
	Trigger string
}

// OnConfigurationChange loads the plugin configuration, validates it and saves it.
func (p *MatterpollPlugin) OnConfigurationChange() error {
	config := new(Config)

	if err := p.API.LoadPluginConfiguration(config); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	if config.Trigger == "" {
		return errors.New("Empty trigger not allowed")
	}

	if p.Config != nil {
		if err := p.API.UnregisterCommand("", p.Config.Trigger); err != nil {
			return errors.Wrap(err, "failed to unregister old command")
		}
	}
	if err := p.API.RegisterCommand(getCommand(config.Trigger)); err != nil {
		return errors.Wrap(err, "failed to register new command")
	}

	p.ServerConfig = p.API.GetConfig()
	p.setConfiguration(config)
	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *MatterpollPlugin) getConfiguration() *Config {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.Config == nil {
		return &Config{}
	}
	return p.Config
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
func (p *MatterpollPlugin) setConfiguration(config *Config) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if config != nil && p.Config == config {
		panic("setConfiguration called with the existing configuration")
	}
	p.Config = config
}
