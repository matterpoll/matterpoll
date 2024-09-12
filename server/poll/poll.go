package poll

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/matterpoll/matterpoll/server/utils"
)

const (
	SettingKeyAnonymous        = "anonymous"
	SettingKeyAnonymousCreator = "anonymous-creator"
	SettingKeyProgress         = "progress"
	SettingKeyPublicAddOption  = "public-add-option"
	EndSettingStandardLayout   = "2006-01-02T15:04"
	EndSettingSecondsLayout    = "2006-01-02T15:04:04"
	EndSettingTimezoneLayout   = "2006-01-02T15:04Z07:00"
)

// Poll stores all needed information for a poll
type Poll struct {
	ID            string
	PostID        string `json:"post_id,omitempty"`
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

// NewPoll creates a new poll with the given parameter.
func NewPoll(creator, question string, answerOptions []string, settings Settings) (*Poll, *utils.ErrorMessage) {
	p := Poll{
		ID:        model.NewId(),
		CreatedAt: model.GetMillis(),
		Creator:   creator,
		Question:  question,
		Settings:  settings,
	}
	for _, answerOption := range answerOptions {
		if errMsg := p.AddAnswerOption(answerOption); errMsg != nil {
			return nil, errMsg
		}
	}

	if errMsg := p.validate(); errMsg != nil {
		return nil, errMsg
	}

	return &p, nil
}

// getUnexpectedErrorMessage get formatted error message for unexpected error
func getUnexpectedErrorMessage(idText, s string) *utils.ErrorMessage {
	return &utils.ErrorMessage{
		Message: &i18n.Message{
			ID:    idText,
			Other: "Unexpected error happens when parsing {{.Setting}}",
		},
		Data: map[string]interface{}{
			"Setting": s,
		},
	}
}

// validate checks if poll is valid
func (p *Poll) validate() *utils.ErrorMessage {
	if p.Settings.MaxVotes <= 0 || p.Settings.MaxVotes > len(p.AnswerOptions) {
		return &utils.ErrorMessage{
			Message: &i18n.Message{
				ID:    "poll.newPoll.votesettings.invalidSetting",
				Other: `The number of votes must be a positive number and less than or equal to the number of options. You specified "{{.MaxVotes}}", but the number of options is "{{.Options}}".`,
			},
			Data: map[string]interface{}{
				"MaxVotes": p.Settings.MaxVotes,
				"Options":  len(p.AnswerOptions),
			},
		}
	}
	return nil
}

// IsMultiVote return true if poll is set to multi vote
func (p *Poll) IsMultiVote() bool {
	return p.Settings.MaxVotes > 1
}

// AddAnswerOption adds a new AnswerOption to a poll
func (p *Poll) AddAnswerOption(newAnswerOption string) *utils.ErrorMessage {
	newAnswerOption = strings.TrimSpace(newAnswerOption)
	if newAnswerOption == "" {
		return &utils.ErrorMessage{
			Message: &i18n.Message{
				ID:    "poll.addAnswerOption.empty",
				Other: "Empty option not allowed",
			},
		}
	}
	for _, answerOption := range p.AnswerOptions {
		if answerOption.Answer == newAnswerOption {
			return &utils.ErrorMessage{
				Message: &i18n.Message{
					ID:    "poll.addAnswerOption.duplicate",
					Other: "Duplicate option: {{.Option}}",
				},
				Data: map[string]interface{}{
					"Option": newAnswerOption,
				},
			}
		}
	}
	ao := &AnswerOption{
		Answer: newAnswerOption,
		Voter:  []string{},
	}
	p.AnswerOptions = append(p.AnswerOptions, ao)
	return nil
}

// UpdateVote performs a vote for a given user
func (p *Poll) UpdateVote(userID string, index int) (*i18n.Message, error) {
	if len(p.AnswerOptions) <= index || index < 0 {
		return nil, fmt.Errorf("invalid index")
	}
	if userID == "" {
		return nil, fmt.Errorf("invalid userID")
	}

	if p.IsMultiVote() {
		// Multi Answer Mode
		votedAnswers := p.GetVotedAnswers(userID)
		for _, answer := range votedAnswers {
			if answer == p.AnswerOptions[index].Answer {
				return &i18n.Message{
					ID:    "poll.updateVote.alreadyVoted",
					Other: "You've already voted for this option.",
				}, nil
			}
		}
		if p.Settings.MaxVotes <= len(votedAnswers) {
			return &i18n.Message{
				ID:    "poll.updateVote.maxVotes",
				Other: "You could't vote for this option, because you don't have any votes left. Use the reset button to reset your votes.",
			}, nil
		}
	} else {
		// Single Answer Mode
		for _, o := range p.AnswerOptions {
			for i := 0; i < len(o.Voter); i++ {
				if userID == o.Voter[i] {
					o.Voter = append(o.Voter[:i], o.Voter[i+1:]...)
				}
			}
		}
	}

	p.AnswerOptions[index].Voter = append(p.AnswerOptions[index].Voter, userID)
	return nil, nil
}

// ResetVotes remove votes by a given user
func (p *Poll) ResetVotes(userID string) {
	for _, o := range p.AnswerOptions {
		for i := 0; i < len(o.Voter); i++ {
			if userID == o.Voter[i] {
				o.Voter = append(o.Voter[:i], o.Voter[i+1:]...)
			}
		}
	}
}

// GetVotedAnswers collect voted answers by a user and returns it as string array.
func (p *Poll) GetVotedAnswers(userID string) []string {
	votedAnswer := []string{}
	for _, o := range p.AnswerOptions {
		for _, v := range o.Voter {
			if userID == v {
				votedAnswer = append(votedAnswer, o.Answer)
			}
		}
	}

	return votedAnswer
}

// GetMetadata returns personalized metadata of a poll.
func (p *Poll) GetMetadata(userID string, permission bool) *Metadata {
	return &Metadata{
		PollID:                 p.ID,
		UserID:                 userID,
		CanManagePoll:          permission,
		VotedAnswers:           p.GetVotedAnswers(userID),
		SettingProgress:        p.Settings.Progress,
		SettingPublicAddOption: p.Settings.PublicAddOption,
	}
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

// Copy deep copies a poll
func (p *Poll) Copy() *Poll {
	p2 := new(Poll)
	*p2 = *p
	p2.AnswerOptions = make([]*AnswerOption, len(p.AnswerOptions))
	for i, o := range p.AnswerOptions {
		p2.AnswerOptions[i] = new(AnswerOption)
		p2.AnswerOptions[i].Answer = o.Answer
		// Only copy Voter if they are nil to ensure the new poll is an exact copy.
		// Please note that polls fetched from the DB might have a nil value,
		// hence we have to still think about this case in the future.
		if o.Voter != nil {
			p2.AnswerOptions[i].Voter = make([]string, len(o.Voter))
			copy(p2.AnswerOptions[i].Voter, o.Voter)
		}
	}
	return p2
}
