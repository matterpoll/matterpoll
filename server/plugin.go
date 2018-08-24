package main

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

var (
	voteRoute       = regexp.MustCompile("/api/v1/polls/([0-9a-z]+)/vote/([0-9]+)")
	endPollRoute    = regexp.MustCompile("/api/v1/polls/([0-9a-z]+)/end")
	deletePollRoute = regexp.MustCompile("/api/v1/polls/([0-9a-z]+)/delete")
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	idGen   IDGenerator
	Config  *Config
	SiteURL string
}

func (p *MatterpollPlugin) OnActivate() error {
	p.idGen = &PollIDGenerator{}
	if p.Config == nil {
		return errors.New("Config empty")
	}
	return nil
}

func (p *MatterpollPlugin) OnDeactivate() error {
	return p.API.UnregisterCommand("", p.Config.Trigger)
}

func (p *MatterpollPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)
	switch {
	case r.URL.Path == "/":
		p.handleInfo(w, r)
	case voteRoute.MatchString(r.URL.Path):
		p.handleVote(w, r)
	case endPollRoute.MatchString(r.URL.Path):
		p.handleEndPoll(w, r)
	case deletePollRoute.MatchString(r.URL.Path):
		p.handleDeletePoll(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *MatterpollPlugin) ConvertUserToDisplayName(userID string) (string, *model.AppError) {
	user, err := p.API.GetUser(userID)
	if err != nil {
		return "", err
	}
	// NOTE: We should better fetch the server config and us this instead of model.SHOW_FULLNAME
	return user.GetDisplayName(model.SHOW_FULLNAME), nil
}
