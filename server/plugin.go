package main

import (
	"net/http"
	"regexp"

	"github.com/mattermost/mattermost-server/plugin"
)

var (
	endPollRoute = regexp.MustCompile("/polls/([0-9a-z]+)/end")
	voteRoute    = regexp.MustCompile("/polls/([0-9a-z]+)/vote/([0-9]+)")
)

const (
	endPollInvalidPermission = "Only the creator of a poll is allowed to end it"
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	idGen IDGenerator
}

func (p *MatterpollPlugin) OnActivate() error {
	p.idGen = &PollIDGenerator{}
	return p.API.RegisterCommand(getCommand())
}

func (p *MatterpollPlugin) OnDeactivate() error {
	return p.API.UnregisterCommand("", trigger)
}

func (p *MatterpollPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch {
	case endPollRoute.MatchString(r.URL.Path):
		p.handleEndPoll(w, r)
	case voteRoute.MatchString(r.URL.Path):
		p.handleVote(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}
