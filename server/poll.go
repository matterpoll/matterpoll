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

func NewPoll(creator, question string, answerOptions []string) *Poll {
	p := Poll{CreatedAt: model.GetMillis(), Creator: creator, DataSchemaVersion: CurrentDataSchemaVersion, Question: question}
	for _, o := range answerOptions {
		p.AnswerOptions = append(p.AnswerOptions, &AnswerOption{Answer: o})
	}
	return &p
}

func (p *Poll) ToPostActions(siteURL, teamID, pollID, authorName string) []*model.SlackAttachment {
	numberOfVotes := 0
	actions := []*model.PostAction{}

	for i, o := range p.AnswerOptions {
		numberOfVotes += len(o.Voter)
		actions = append(actions, &model.PostAction{
			Name: o.Answer,
			Type: model.POST_ACTION_TYPE_BUTTON,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/%v", siteURL, PluginId, CurrentApiVersion, pollID, i),
				Context: model.StringInterface{
					"team_id": teamID,
				},
			},
		})
	}

	actions = append(actions, &model.PostAction{
		Name: "Delete Poll",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginId, CurrentApiVersion, pollID),
			Context: model.StringInterface{
				"team_id": teamID,
			},
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "End Poll",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginId, CurrentApiVersion, pollID),
			Context: model.StringInterface{
				"team_id": teamID,
			},
		},
	})

	return []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       fmt.Sprintf("Total votes: %d", numberOfVotes),
		Actions:    actions,
	}}
}

func (p *Poll) ToCommandResponse(siteURL, teamID, pollID, authorName string) *model.CommandResponse {
	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "", siteURL, p.ToPostActions(siteURL, teamID, pollID, authorName))
}

func (p *Poll) ToEndPollPost(authorName string, convert func(string) (string, *model.AppError)) (*model.Post, *model.AppError) {
	post := &model.Post{}
	fields := []*model.SlackAttachmentField{}

	for _, o := range p.AnswerOptions {
		var voter string
		for i := 0; i < len(o.Voter); i++ {
			displayName, err := convert(o.Voter[i])
			if err != nil {
				return nil, err
			}
			if i+1 == len(o.Voter) && len(o.Voter) > 1 {
				voter += " and "
			} else if i != 0 {
				voter += ", "
			}
			voter += displayName
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
	model.ParseSlackAttachment(post, attachments)

	return post, nil
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

func (p *Poll) Copy() *Poll {
	p2 := new(Poll)
	*p2 = *p
	p2.AnswerOptions = make([]*AnswerOption, len(p.AnswerOptions))
	for i, o := range p.AnswerOptions {
		p2.AnswerOptions[i] = new(AnswerOption)
		p2.AnswerOptions[i].Answer = o.Answer
		p2.AnswerOptions[i].Voter = o.Voter
	}
	return p2
}
