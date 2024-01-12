package main

import (
	mmplugin "github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/matterpoll/matterpoll/server/plugin"
)

func main() {
	mmplugin.ClientMain(plugin.NewMatterpollPlugin())
}
