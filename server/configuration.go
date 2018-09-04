package main

import "errors"

type Config struct {
	Trigger string
}

func (p *MatterpollPlugin) OnConfigurationChange() error {
	c := &Config{}
	if err := p.API.LoadPluginConfiguration(c); err != nil {
		return err
	}

	if c.Trigger == "" {
		return errors.New("Empty trigger not allowed")
	}

	if p.Config != nil {
		if err := p.API.UnregisterCommand("", p.Config.Trigger); err != nil {
			return err
		}
	}
	if err := p.API.RegisterCommand(getCommand(c.Trigger)); err != nil {
		return err
	}

	p.ServerConfig = p.API.GetConfig()
	p.Config = c
	return nil
}
