// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

type AutoChannelCreator struct {
	client             *model.Client
	team               *model.Team
	Fuzzy              bool
	DisplayNameLen     utils.Range
	DisplayNameCharset string
	NameLen            utils.Range
	NameCharset        string
	ChannelType        string
}

func NewAutoChannelCreator(client *model.Client, team *model.Team) *AutoChannelCreator {
	return &AutoChannelCreator{
		client:             client,
		team:               team,
		Fuzzy:              false,
		DisplayNameLen:     CHANNEL_DISPLAY_NAME_LEN,
		DisplayNameCharset: utils.ALPHANUMERIC,
		NameLen:            CHANNEL_NAME_LEN,
		NameCharset:        utils.LOWERCASE,
		ChannelType:        CHANNEL_TYPE,
	}
}

func (cfg *AutoChannelCreator) createRandomChannel() (*model.Channel, bool) {
	var displayName string
	if cfg.Fuzzy {
		displayName = utils.FuzzName()
	} else {
		displayName = utils.RandomName(cfg.NameLen, cfg.NameCharset)
	}
	name := utils.RandomName(cfg.NameLen, cfg.NameCharset)

	channel := &model.Channel{
		TeamId:      cfg.team.Id,
		DisplayName: displayName,
		Name:        name,
		Type:        cfg.ChannelType}

	println(cfg.client.GetTeamRoute())
	result, err := cfg.client.CreateChannel(channel)
	if err != nil {
		err.Translate(utils.T)
		println(err.Error())
		println(err.DetailedError)
		return nil, false
	}
	return result.Data.(*model.Channel), true
}

func (cfg *AutoChannelCreator) CreateTestChannels(num utils.Range) ([]*model.Channel, bool) {
	numChannels := utils.RandIntFromRange(num)
	channels := make([]*model.Channel, numChannels)

	for i := 0; i < numChannels; i++ {
		var err bool
		channels[i], err = cfg.createRandomChannel()
		if !err {
			return channels, false
		}
	}

	return channels, true
}
