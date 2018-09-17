package main

import (
	"errors"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	idGen        IDGenerator
	router       *mux.Router
	Config       *Config
	ServerConfig *model.Config
}

func (p *MatterpollPlugin) OnActivate() error {
	p.idGen = &PollIDGenerator{}
	if p.Config == nil {
		return errors.New("Config empty")
	}
	p.router = mux.NewRouter()
	p.router.HandleFunc("/", p.handleInfo)
	p.router.HandleFunc("/"+iconFilename, p.handleLogo)
	p.router.HandleFunc("/api/v1/polls/{id:[a-z0-9]+}/vote/{optionNumber:[0-9]+}", p.handleVote)
	p.router.HandleFunc("/api/v1/polls/{id:[a-z0-9]+}/end", p.handleEndPoll)
	p.router.HandleFunc("/api/v1/polls/{id:[a-z0-9]+}/delete", p.handleDeletePoll)
	return nil
}

func (p *MatterpollPlugin) OnDeactivate() error {
	return p.API.UnregisterCommand("", p.Config.Trigger)
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
