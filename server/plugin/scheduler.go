package plugin

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/matterpoll/matterpoll/server/poll"
)

func (p *MatterpollPlugin) Scheduler() {
	a := p.Store.Poll()
	allKeys, err := a.GetAllPollIDs()
	if err != nil {
		p.API.LogWarn("failed to fetch all polls", "error", err.Error())
	}

	for _, k := range allKeys {
		poll, err := p.Store.Poll().Get(k)
		if err != nil {
			p.API.LogWarn("Failed to get poll", "error", err.Error(), "pollID", k)
			continue
		}

		if poll.Settings.End != nil {
			p.StartScheduler(k, poll.Settings.End)
		}
	}
}

func (p *MatterpollPlugin) StartScheduler(pollID string, expireTime *time.Time) {
	duration := time.Until(*expireTime)

	if duration >= 0 {
		go func() {
			<-time.NewTimer(duration).C
			p.endPollScheduler(pollID, expireTime)
		}()
	} else {
		p.endPollScheduler(pollID, expireTime)
	}
}

func (p *MatterpollPlugin) endPollScheduler(pollID string, expireTime *time.Time) {
	poll, post, err := p.EndPoll(pollID, nil)
	if err != nil {
		errMsg := fmt.Sprintf("failed to end poll after expireTime %v", expireTime.String())
		p.API.LogWarn(errMsg, "error", err.Error())
		p.sendEphemeralPostEndError(poll, post)
		return
	}
}

func (p *MatterpollPlugin) sendEphemeralPostEndError(poll *poll.Poll, post *model.Post) {
	var rootID string
	if post.RootId != "" {
		rootID = post.RootId
	} else {
		rootID = post.Id
	}

	userLocalizer := p.bundle.GetUserLocalizer(poll.Creator)

	p.SendEphemeralPost(post.ChannelId, poll.Creator, rootID, p.bundle.LocalizeDefaultMessage(userLocalizer, commandErrorSchedulerEnd))
}
