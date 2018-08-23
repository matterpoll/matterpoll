package main

type Config struct {
	Trigger string
}

func (p *MatterpollPlugin) OnConfigurationChange() error {
	return p.LoadCConfigurationAndRegisterCommand()
}

func (p *MatterpollPlugin) LoadCConfigurationAndRegisterCommand() error {
	c := &Config{}
	if err := p.API.LoadPluginConfiguration(c); err != nil {
		return err
	}
	if p.Config != nil && p.Config.Trigger != "" {
		if err := p.API.UnregisterCommand("", p.Config.Trigger); err != nil {
			return err
		}
	}
	if err := p.API.RegisterCommand(getCommand(c.Trigger)); err != nil {
		return err
	}
	p.Config = c

	return nil
}
