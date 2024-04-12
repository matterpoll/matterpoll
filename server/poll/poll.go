package poll

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/matterpoll/matterpoll/server/utils"
)

var votesSettingPattern = regexp.MustCompile(`^votes=(\d+)$`)

const (
	SettingKeyAnonymous        = "anonymous"
	SettingKeyAnonymousCreator = "anonymous-creator"
	SettingKeyProgress         = "progress"
	SettingKeyPublicAddOption  = "public-add-option"
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

// Settings stores possible settings for a poll
type Settings struct {
	Anonymous        bool
	AnonymousCreator bool
	Progress         bool
	PublicAddOption  bool
	MaxVotes         int `json:"max_votes"`
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

// NewSettingsFromStrings creates a new settings with the given parameter.
func NewSettingsFromStrings(strs []string) (Settings, *utils.ErrorMessage) {
	settings := Settings{MaxVotes: 1}
	for _, str := range strs {
		switch {
		case str == SettingKeyAnonymous:
			settings.Anonymous = true
		case str == SettingKeyAnonymousCreator:
			settings.AnonymousCreator = true
		case str == SettingKeyProgress:
			settings.Progress = true
		case str == SettingKeyPublicAddOption:
			settings.PublicAddOption = true
		case votesSettingPattern.MatchString(str):
			i, errMsg := parseVotesSettings(str)
			if errMsg != nil {
				return settings, errMsg
			}
			settings.MaxVotes = i
		default:
			return settings, &utils.ErrorMessage{
				Message: &i18n.Message{
					ID:    "poll.newPoll.unrecognizedSetting",
					Other: "Unrecognized poll setting: {{.Setting}}",
				},
				Data: map[string]interface{}{
					"Setting": str,
				},
			}
		}
	}
	return settings, nil
}

// NewSettingsFromSubmission creates a new settings with the given parameter.
func NewSettingsFromSubmission(submission map[string]interface{}) Settings {
	settings := Settings{MaxVotes: 1}
	for k, v := range submission {
		if k == "setting-multi" {
			f, ok := v.(float64)
			if ok {
				settings.MaxVotes = int(f)
			}
		} else if strings.HasPrefix(k, "setting-") {
			b, ok := v.(bool)
			if b && ok {
				s := strings.TrimPrefix(k, "setting-")
				switch s {
				case SettingKeyAnonymous:
					settings.Anonymous = true
				case SettingKeyAnonymousCreator:
					settings.AnonymousCreator = true
				case SettingKeyProgress:
					settings.Progress = true
				case SettingKeyPublicAddOption:
					settings.PublicAddOption = true
				}
			}
		}
	}
	return settings
}

// parseVotesSettings parses setting for votes ("--votes=X")
func parseVotesSettings(s string) (int, *utils.ErrorMessage) {
	e := votesSettingPattern.FindStringSubmatch(s)
	if len(e) != 2 {
		return 0, &utils.ErrorMessage{
			Message: &i18n.Message{
				ID:    "poll.newPoll.votesettings.unexpectedError",
				Other: "Unexpected error happens when parsing {{.Setting}}",
			},
			Data: map[string]interface{}{
				"Setting": s,
			},
		}
	}
	i, err := strconv.Atoi(e[1])
	if err != nil {
		return 0, &utils.ErrorMessage{
			Message: &i18n.Message{
				ID:    "poll.newPoll.votesettings.unexpectedError",
				Other: "Unexpected error happens when parsing {{.Setting}}",
			},
			Data: map[string]interface{}{
				"Setting": s,
			},
		}
	}
	return i, nil
}

// validate checks if poll is valid
func (p *Poll) validate() *utils.ErrorMessage {
	if p.Settings.MaxVotes < 0 || p.Settings.MaxVotes > len(p.AnswerOptions) {
		return &utils.ErrorMessage{
			Message: &i18n.Message{
				ID:    "poll.newPoll.votesettings.invalidSetting",
				Other: `The number of votes must be 0 or a positive number, and must be less than or equal to the number of options. You specified "{{.MaxVotes}}", but the number of options is "{{.Options}}".`,
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
	return p.Settings.MaxVotes == 0 || p.Settings.MaxVotes > 1
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
		if p.Settings.MaxVotes != 0 && p.Settings.MaxVotes <= len(votedAnswers) {
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

func (s Settings) String() string {
	var settingsText []string
	if s.Anonymous {
		settingsText = append(settingsText, "anonymous")
	}
	if s.AnonymousCreator {
		settingsText = append(settingsText, "anonymous-creator")
	}
	if s.Progress {
		settingsText = append(settingsText, "progress")
	}
	if s.PublicAddOption {
		settingsText = append(settingsText, "public-add-option")
	}
	if s.MaxVotes != 1 {
		switch s.MaxVotes {
		case 0:
			settingsText = append(settingsText, "votes=all")
		default:
			settingsText = append(settingsText, fmt.Sprintf("votes=%d", s.MaxVotes))
		}
	}

	return strings.Join(settingsText, ", ")
}
