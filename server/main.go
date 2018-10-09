package main

import (
	mmplugin "github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/plugin"
)

func main() {
	mmplugin.ClientMain(&plugin.MatterpollPlugin{})
}
