package plugin

import "github.com/matterpoll/matterpoll/server/poll"

// applyConfiguration updates the settings with relevant configuration settings, e.g., ShowProgressBars
func (p *MatterpollPlugin) applyConfiguration(settings poll.Settings) poll.Settings {
	settings.ShowProgressBars = p.configuration.ShowProgressBars
	settings.ProgressBarLength = p.configuration.ProgressBarLength
	return settings
}
