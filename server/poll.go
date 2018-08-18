package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

type Poll struct {
	Creator  string
	Question string
	Options  []*Option
}

type Option struct {
	Answer string
	Voter  []string
}

func NewPoll(creator string, question string, options []string) *Poll {
	p := Poll{Creator: creator, Question: question}
	for _, o := range options {
		p.Options = append(p.Options, &Option{Answer: o})
	}
	return &p
}

func (p *Poll) ToCommandResponse(siteURL, authorName, pollID string) *model.CommandResponse {
	actions := []*model.PostAction{}
	for i, o := range p.Options {
		actions = append(actions, &model.PostAction{
			Name: o.Answer,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/polls/%s/vote/%v", siteURL, PluginId, pollID, i),
			},
		})
	}

	actions = append(actions, &model.PostAction{
		Name: "Delete Poll",
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/polls/%s/delete", siteURL, PluginId, pollID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "End Poll",
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/polls/%s/end", siteURL, PluginId, pollID),
		},
	})

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "", []*model.SlackAttachment{{
		AuthorName: authorName,
		Text:       p.Question,
		Actions:    actions,
	},
	})
}

func (p *Poll) UpdateVote(userID string, index int) error {
	if len(p.Options) <= index || index < 0 {
		return errors.New("invalid index")
	}
	if userID == "" {
		return errors.New("invalid userID")
	}
	for _, o := range p.Options {
		for i := 0; i < len(o.Voter); i++ {
			if userID == o.Voter[i] {
				o.Voter = append(o.Voter[:i], o.Voter[i+1:]...)
			}
		}
	}
	p.Options[index].Voter = append(p.Options[index].Voter, userID)
	return nil
}

func (p *Poll) HasVoted(userID string) bool {
	for _, o := range p.Options {
		for i := 0; i < len(o.Voter); i++ {
			if userID == o.Voter[i] {
				return true
			}
		}
	}
	return false
}

func (p *Poll) Encode() []byte {
	b, _ := json.Marshal(p)
	return b
}

func Decode(b []byte) *Poll {
	p := Poll{}
	_ = json.Unmarshal(b, &p)
	return &p
}
