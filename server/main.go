package main

import (
	mmplugin "github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/matterpoll/matterpoll/server/plugin"
)

func main() {
	mmplugin.ClientMain(&plugin.MatterpollPlugin{})
}
