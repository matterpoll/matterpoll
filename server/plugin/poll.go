package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

type Poll struct {
	CreatedAt         int64
	Creator           string
	DataSchemaVersion string
	Question          string
	AnswerOptions     []*AnswerOption
	Settings          PollSettings
}

type AnswerOption struct {
	Answer string
	Voter  []string
}

type PollSettings struct {
	Anonymous bool
	Progress  bool
}

func NewPoll(creator, question string, answerOptions, settings []string) (*Poll, error) {
	p := Poll{CreatedAt: model.GetMillis(), Creator: creator, DataSchemaVersion: CurrentDataSchemaVersion, Question: question}
	for _, o := range answerOptions {
		p.AnswerOptions = append(p.AnswerOptions, &AnswerOption{Answer: o})
	}
	for _, s := range settings {
		switch s {
		case "anonymous":
			p.Settings.Anonymous = true
		case "progress":
			p.Settings.Progress = true
		default:
			return nil, fmt.Errorf("Unrecognised poll setting %s", s)
		}
	}
	return &p, nil
}

func (p *Poll) ToPostActions(siteURL, pollID, authorName string) []*model.SlackAttachment {
	numberOfVotes := 0
	actions := []*model.PostAction{}

	for i, o := range p.AnswerOptions {
		numberOfVotes += len(o.Voter)
		answer := o.Answer
		if p.Settings.Progress {
			answer = fmt.Sprintf("%s (%d)", answer, len(o.Voter))
		}
		actions = append(actions, &model.PostAction{
			Name: answer,
			Type: model.POST_ACTION_TYPE_BUTTON,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/%v", siteURL, PluginId, CurrentAPIVersion, pollID, i),
			},
		})
	}

	actions = append(actions, &model.PostAction{
		Name: "Delete Poll",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginId, CurrentAPIVersion, pollID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "End Poll",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginId, CurrentAPIVersion, pollID),
		},
	})

	return []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       p.makeAdditionalText(numberOfVotes),
		Actions:    actions,
	}}
}

// makeAdditionalText make descriptions about poll
// This method returns markdown text, because it is used for SlackAttachment.Text field.
func (p *Poll) makeAdditionalText(numberOfVotes int) string {
	var settingsText []string
	if p.Settings.Anonymous {
		settingsText = append(settingsText, "anonymous")
	}
	if p.Settings.Progress {
		settingsText = append(settingsText, "progress")
	}

	lines := []string{"---"}
	if len(settingsText) > 0 {
		lines = append(lines, fmt.Sprintf("**Poll settings**: %s", strings.Join(settingsText, ", ")))
	}
	lines = append(lines, fmt.Sprintf("**Total votes**: %d", numberOfVotes))
	return strings.Join(lines, "\n")
}

func (p *Poll) ToCommandResponse(siteURL, pollID, authorName string) *model.CommandResponse {
	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "", siteURL, p.ToPostActions(siteURL, pollID, authorName))
}

func (p *Poll) ToEndPollPost(authorName string, convert func(string) (string, *model.AppError)) (*model.Post, *model.AppError) {
	post := &model.Post{}
	fields := []*model.SlackAttachmentField{}

	for _, o := range p.AnswerOptions {
		var voter string
		if !p.Settings.Anonymous {
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
		return fmt.Errorf("invalid index")
	}
	if userID == "" {
		return fmt.Errorf("invalid userID")
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
