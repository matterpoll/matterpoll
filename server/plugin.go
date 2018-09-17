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

const (
	iconFilename = "logo_dark.png"
	iconPath     = "plugins/" + PluginId + "/"
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	idGen        IDGenerator
	Config       *Config
	ServerConfig *model.Config
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
	case r.URL.Path == "/"+iconFilename:
		http.ServeFile(w, r, iconPath+iconFilename)
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

func (p *MatterpollPlugin) ConvertUserIDToDisplayName(userID string) (string, *model.AppError) {
	user, err := p.API.GetUser(userID)
	if err != nil {
		return "", err
	}
	displayName := user.GetDisplayName(model.SHOW_USERNAME)
	displayName = "@" + displayName
	return displayName, nil
}

func (p *MatterpollPlugin) ConvertCreatorIDToDisplayName(creatorID string) (string, *model.AppError) {
	user, err := p.API.GetUser(creatorID)
	if err != nil {
		return "", err
	}
	displayName := user.GetDisplayName(model.SHOW_NICKNAME_FULLNAME)
	return displayName, nil
}
