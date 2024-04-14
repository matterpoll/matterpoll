package plugin

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/nicksnyder/go-i18n/v2/i18n"

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
			p.endPoll(pollID, expireTime)
		}()
	} else {
		p.endPoll(pollID, expireTime)
	}
}

func (p *MatterpollPlugin) endPoll(pollID string, expireTime *time.Time) {
	errMsg := fmt.Sprintf("failed to end poll after expireTime %v", expireTime.String())
	poll, err := p.Store.Poll().Get(pollID)
	if err != nil {
		p.API.LogWarn(errMsg, "error", err.Error())
		return
	}

	displayName, appErr := p.ConvertCreatorIDToDisplayName(poll.Creator)
	if appErr != nil {
		p.API.LogWarn(errMsg, "error", appErr.Error())
		return
	}

	post, appErr := poll.ToEndPollPost(p.bundle, displayName, p.ConvertUserIDToDisplayName)
	if appErr != nil {
		p.API.LogWarn(errMsg, "error", appErr.Error())
		return
	}

	if poll.PostID == "" {
		p.API.LogWarn(errMsg, "error", "poll is created without a postID")
		p.sendEphemeralPost(poll, post)
		return
	}

	post.Id = poll.PostID
	if _, appErr = p.API.UpdatePost(post); appErr != nil {
		p.API.LogWarn(errMsg, "error", appErr.Error())
		p.sendEphemeralPost(poll, post)
		return
	}

	if err := p.Store.Poll().Delete(poll); err != nil {
		p.API.LogWarn(errMsg, "error", appErr.Error())
		p.sendEphemeralPost(poll, post)
		return
	}

	p.postEndPollAnnouncement(post.ChannelId, post.Id, poll.Question)
}

func (p *MatterpollPlugin) sendEphemeralPost(poll *poll.Poll, post *model.Post) {
	var rootID string
	if post.RootId != "" {
		rootID = post.RootId
	} else {
		rootID = post.Id
	}

	userLocalizer := p.bundle.GetUserLocalizer(poll.Creator)
	msg := &i18n.Message{
		ID:    "command.error.schedule_end",
		Other: "Something went wrong during automatically end of poll.",
	}

	p.SendEphemeralPost(post.ChannelId, poll.Creator, rootID, p.bundle.LocalizeDefaultMessage(userLocalizer, msg))
}
