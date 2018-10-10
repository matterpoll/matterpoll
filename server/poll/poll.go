package poll

import (
	"encoding/json"
	"fmt"

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

func NewPoll(currentDataSchemaVersion, creator, question string, answerOptions, settings []string) (*Poll, error) {
	p := Poll{CreatedAt: model.GetMillis(), DataSchemaVersion: currentDataSchemaVersion, Creator: creator, Question: question}
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
