package main

import (
	"github.com/mattermost/mattermost-server/model"
)

type IDGenerator interface {
	NewId() string
}

type PollIDGenerator struct{}

func (keks *PollIDGenerator) NewId() string {
	return model.NewId()
}
