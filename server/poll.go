package main

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

type Poll struct {
	Question string
	Options  []*Option
}

type Option struct {
	Answer string
	Voter  []string
}

func NewPoll(question string, options []string) *Poll {
	p := Poll{Question: question}
	for _, o := range options {
		p.Options = append(p.Options, &Option{Answer: o})
	}
	return &p
}

func (p *Poll) ToCommandResponse(siteURL string, id string) *model.CommandResponse {
	actions := []*model.PostAction{}
	for _, o := range p.Options {
		actions = append(actions, &model.PostAction{
			Name: o.Answer,
		})
	}

	actions = append(actions, &model.PostAction{
		Name: `End Poll`,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf(`%s/plugins/%s/polls/%s/end`, siteURL, PluginId, id),
		},
	})

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, ``, []*model.SlackAttachment{{
		AuthorName: `Matterpoll`,
		Text:       p.Question,
		Actions:    actions,
	},
	})
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
