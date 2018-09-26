package main

import (
	"errors"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	router       *mux.Router
	Config       *Config
	ServerConfig *model.Config
}

func (p *MatterpollPlugin) OnActivate() error {
	if p.Config == nil {
		return errors.New("Config empty")
	}
	p.router = p.InitAPI()
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

func (p *MatterpollPlugin) HasPermission(poll *Poll, issuerID string) (bool, *model.AppError) {
	if issuerID == poll.Creator {
		return true, nil
	}

	user, appErr := p.API.GetUser(issuerID)
	if appErr != nil {
		return false, appErr
	}
	if user.IsInRole(model.SYSTEM_ADMIN_ROLE_ID) {
		return true, nil
	}
	return false, nil
}
