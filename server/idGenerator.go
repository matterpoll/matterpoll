package main

import (
	"github.com/mattermost/mattermost-server/model"
)

type IDGenerator interface {
	NewID() string
}

type PollIDGenerator struct{}

func (keks *PollIDGenerator) NewID() string {
	return model.NewId()
}
