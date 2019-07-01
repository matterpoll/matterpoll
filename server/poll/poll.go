package poll

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

// Poll stores all needed information for a poll
type Poll struct {
	ID            string
	CreatedAt     int64
	Creator       string
	Question      string
	AnswerOptions []*AnswerOption
	Settings      Settings
}

// AnswerOption stores a possible answer and a list of user who voted for this
type AnswerOption struct {
	Answer string
	Voter  []string
}

// Settings stores possible settings for a poll
type Settings struct {
	Anonymous       bool
	Progress        bool
	PublicAddOption bool
}

// NewPoll creates a new poll with the given paramatern
func NewPoll(creator, question string, answerOptions, settings []string) (*Poll, error) {
	p := Poll{
		ID:        model.NewId(),
		CreatedAt: model.GetMillis(),
		Creator:   creator,
		Question:  question,
	}
	for _, answerOption := range answerOptions {
		if err := p.AddAnswerOption(answerOption); err != nil {
			return nil, err
		}
	}
	for _, s := range settings {
		switch s {
		case "anonymous":
			p.Settings.Anonymous = true
		case "progress":
			p.Settings.Progress = true
		case "public-add-option":
			p.Settings.PublicAddOption = true
		default:
			return nil, fmt.Errorf("Unrecognised poll setting %s", s)
		}
	}
	return &p, nil
}

// AddAnswerOption adds a new AnswerOption to a poll
func (p *Poll) AddAnswerOption(newAnswerOption string) error {
	newAnswerOption = strings.TrimSpace(newAnswerOption)
	if newAnswerOption == "" {
		return errors.New("empty option not allowed")
	}
	for _, answerOption := range p.AnswerOptions {
		if answerOption.Answer == newAnswerOption {
			return fmt.Errorf("duplicate options: %s", newAnswerOption)
		}
	}
	p.AnswerOptions = append(p.AnswerOptions, &AnswerOption{Answer: newAnswerOption})
	return nil
}

// UpdateVote performs a vote for a given user
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

// HasVoted return true if a given user has voted in this poll
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

// EncodeToByte returns a poll as a byte array
func (p *Poll) EncodeToByte() []byte {
	b, _ := json.Marshal(p)
	return b
}

// DecodePollFromByte tries to create a poll from a byte array
func DecodePollFromByte(b []byte) *Poll {
	p := Poll{}
	err := json.Unmarshal(b, &p)
	if err != nil {
		return nil
	}
	return &p
}

// Copy deep copys a poll
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
