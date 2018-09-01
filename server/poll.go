package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

type Poll struct {
	CreatedAt         int64
	Creator           string
	DataSchemaVersion string
	Question          string
	AnswerOptions     []*AnswerOption
}

type AnswerOption struct {
	Answer string
	Voter  []string
}

func NewPoll(creator string, question string, answerOptions []string) *Poll {
	p := Poll{CreatedAt: model.GetMillis(), Creator: creator, DataSchemaVersion: CurrentDataSchemaVersion, Question: question}
	for _, o := range answerOptions {
		p.AnswerOptions = append(p.AnswerOptions, &AnswerOption{Answer: o})
	}
	return &p
}

func (p *Poll) ToCommandResponse(siteURL, authorName, pollID string) *model.CommandResponse {
	actions := []*model.PostAction{}
	for i, o := range p.AnswerOptions {
		actions = append(actions, &model.PostAction{
			Name: o.Answer,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/%v", siteURL, PluginId, CurrentApiVersion, pollID, i),
			},
		})
	}

	actions = append(actions, &model.PostAction{
		Name: "Delete Poll",
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginId, CurrentApiVersion, pollID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "End Poll",
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginId, CurrentApiVersion, pollID),
		},
	})

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "", []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Actions:    actions,
	},
	})
}

func (p *Poll) ToEndPollPost(authorName string, convert func(string) (string, *model.AppError)) (*model.Post, *model.AppError) {
	post := model.Post{}

	fields := []*model.SlackAttachmentField{}

	for _, o := range p.AnswerOptions {
		var voter string
		for i := 0; i < len(o.Voter); i++ {
			userName, err := convert(o.Voter[i])
			if err != nil {
				return nil, err
			}
			if i+1 == len(o.Voter) && len(o.Voter) > 1 {
				voter += " and "
			} else if i != 0 {
				voter += ", "
			}
			voter += fmt.Sprintf("@%s", userName)
		}
		var voteText string
		if len(o.Voter) == 1 {
			voteText = "vote"
		} else {
			voteText = "votes"
		}
		fields = append(fields, &model.SlackAttachmentField{
			Short: true,
			Title: fmt.Sprintf("%s (%d %s)", o.Answer, len(o.Voter), voteText),
			Value: voter,
		})
	}

	attachments := []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       "This poll has ended. The results are:",
		Fields:     fields,
	}}
	post.AddProp("attachments", attachments)

	return &post, nil
}

func (p *Poll) UpdateVote(userID string, index int) error {
	if len(p.AnswerOptions) <= index || index < 0 {
		return errors.New("invalid index")
	}
	if userID == "" {
		return errors.New("invalid userID")
	}
	for _, o := range p.AnswerOptions {
		for i := 0; i < len(o.Voter); i++ {
			if userID == o.Voter[i] {
				o.Voter = append(o.Voter[:i], o.Voter[i+1:]...)
			}
		}
	}
	p.AnswerOptions[index].Voter = append(p.AnswerOptions[index].Voter, userID)
	return nil
}

func (p *Poll) HasVoted(userID string) bool {
	for _, o := range p.AnswerOptions {
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
	err := json.Unmarshal(b, &p)
	if err != nil {
		return nil
	}
	return &p
}
