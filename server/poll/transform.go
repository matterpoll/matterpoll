package poll

import (
	"fmt"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/matterpoll/matterpoll/server/utils"
)

const (
	// MatterpollAdminButtonType is action_type of buttons that are used for managing a poll.
	MatterpollAdminButtonType = "custom_matterpoll_admin_button"
)

// IDToNameConverter converts a given userID to a human readable name.
type IDToNameConverter func(userID string) (string, *model.AppError)

var (
	pollMessageSettings = &i18n.Message{
		ID:    "poll.message.pollSettings",
		Other: "**Poll Settings**: {{.Settings}}",
	}
	pollMessageTotalVotes = &i18n.Message{
		ID:    "poll.message.totalVotes",
		Other: "**Total votes**: {{.TotalVotes}}",
	}
	pollMessageTotalVotesMultiSetting = &i18n.Message{
		ID:    "poll.message.totalVotesMulti",
		One:   "**Total votes**: {{.TotalVotes}} ({{ .TotalVoters }} voter)",
		Few:   "**Total votes**: {{.TotalVotes}} ({{ .TotalVoters }} voters)",
		Many:  "**Total votes**: {{.TotalVotes}} ({{ .TotalVoters }} voters)",
		Other: "**Total votes**: {{.TotalVotes}} ({{ .TotalVoters }} voters)",
	}

	pollEndPostText = &i18n.Message{
		ID:    "poll.endPost.text",
		Other: "This poll has ended. The results are:",
	}
	pollEndPostSeperator = &i18n.Message{
		ID:    "poll.endPost.seperator",
		Other: "and",
	}
	pollEndPostAnswerHeading = &i18n.Message{
		ID:    "poll.endPost.answer.heading",
		One:   "{{.Answer}} ({{.Count}} vote)",
		Few:   "{{.Answer}} ({{.Count}} votes)",
		Many:  "{{.Answer}} ({{.Count}} votes)",
		Other: "{{.Answer}} ({{.Count}} votes)",
	}
	rhsCardPollVoterSeperator = &i18n.Message{
		ID:    "rhs.card.poll.voter.seperator",
		Other: "and",
	}
	rhsCardPollCreatedBy = &i18n.Message{
		ID:    "rhs.card.poll.createdBy",
		Other: "Created by",
	}
	rhsCardPollAnswerHeading = &i18n.Message{
		ID:    "rhs.card.poll.answer.heading",
		One:   "{{.Answer}} ({{.Count}} vote)",
		Few:   "{{.Answer}} ({{.Count}} votes)",
		Many:  "{{.Answer}} ({{.Count}} votes)",
		Other: "{{.Answer}} ({{.Count}} votes)",
	}
)

// ToPostActions returns the poll as a message
func (p *Poll) ToPostActions(bundle *utils.Bundle, pluginID, authorName string) []*model.SlackAttachment {
	localizer := bundle.GetServerLocalizer()
	numberOfVotes := 0
	voters := make(map[string]struct{})
	actions := []*model.PostAction{}

	for i, o := range p.AnswerOptions {
		numberOfVotes += len(o.Voter)
		for _, v := range o.Voter {
			voters[v] = struct{}{}
		}
		answer := o.Answer
		if p.Settings.Progress {
			answer = fmt.Sprintf("%s (%d)", answer, len(o.Voter))
		}
		actions = append(actions, &model.PostAction{
			Id:    fmt.Sprintf("vote%v", i),
			Name:  answer,
			Type:  model.PostActionTypeButton,
			Style: "default",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/vote/%v", pluginID, p.ID, i),
			},
		})
	}

	actions = append(actions,
		&model.PostAction{
			Id: "resetVote",
			Name: bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "poll.button.resetVotes",
					One:   "Reset Your Vote",
					Few:   "Reset Your Votes",
					Many:  "Reset Your Votes",
					Other: "Reset Your Votes",
				},
				PluralCount: p.Settings.MaxVotes,
			}),
			Type:  model.PostActionTypeButton,
			Style: "primary",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/votes/reset", pluginID, p.ID),
			},
		},
	)
	actions = append(actions,
		&model.PostAction{
			Id: "addOption",
			Name: bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{DefaultMessage: &i18n.Message{
				ID:    "poll.button.addOption",
				Other: "Add Option",
			}}),
			Type:  model.PostActionTypeButton,
			Style: "primary",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/option/add/request", pluginID, p.ID),
			},
		}, &model.PostAction{
			Id: "endPoll",
			Name: bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{DefaultMessage: &i18n.Message{
				ID:    "poll.button.endPoll",
				Other: "End Poll",
			}}),
			Type:  MatterpollAdminButtonType,
			Style: "primary",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/end", pluginID, p.ID),
			},
		}, &model.PostAction{
			Id: "deletePoll",
			Name: bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{DefaultMessage: &i18n.Message{
				ID:    "poll.button.deletePoll",
				Other: "Delete Poll",
			}}),
			Type:  MatterpollAdminButtonType,
			Style: "danger",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/delete", pluginID, p.ID),
			},
		},
	)

	if p.Settings.AnonymousCreator {
		authorName = ""
	}

	return []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       p.makeAdditionalText(bundle, numberOfVotes, len(voters)),
		Actions:    actions,
	}}
}

// makeAdditionalText make descriptions about poll
// This method returns markdown text, because it is used for SlackAttachment.Text field.
func (p *Poll) makeAdditionalText(bundle *utils.Bundle, numberOfVotes, numberOfVoters int) string {
	localizer := bundle.GetServerLocalizer()
	settingsText := p.Settings.String()

	lines := []string{"---"}
	if len(settingsText) > 0 {
		lines = append(lines, bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{
			DefaultMessage: pollMessageSettings,
			TemplateData:   map[string]interface{}{"Settings": settingsText},
		}))
	}

	if p.IsMultiVote() {
		lines = append(lines, bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{
			DefaultMessage: pollMessageTotalVotesMultiSetting,
			TemplateData: map[string]interface{}{
				"TotalVotes":  numberOfVotes,
				"TotalVoters": numberOfVoters,
			},
			PluralCount: numberOfVoters,
		}))
	} else {
		lines = append(lines, bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{
			DefaultMessage: pollMessageTotalVotes,
			TemplateData:   map[string]interface{}{"TotalVotes": numberOfVotes},
		}))
	}
	return strings.Join(lines, "\n")
}

// ToEndPollPost returns the poll end message
func (p *Poll) ToEndPollPost(bundle *utils.Bundle, authorName string, convert IDToNameConverter) (*model.Post, *model.AppError) {
	localizer := bundle.GetServerLocalizer()
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
					voter += " " + bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{DefaultMessage: pollEndPostSeperator}) + " "
				} else if i != 0 {
					voter += ", "
				}
				voter += displayName
			}
		}

		fields = append(fields, &model.SlackAttachmentField{
			Short: true,
			Title: bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{
				DefaultMessage: pollEndPostAnswerHeading,
				TemplateData: map[string]interface{}{
					"Answer": o.Answer,
					"Count":  len(o.Voter),
				},
				PluralCount: len(o.Voter),
			}),
			Value: voter,
		})
	}

	if p.Settings.AnonymousCreator {
		authorName = ""
	}

	attachments := []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{DefaultMessage: pollEndPostText}),
		Fields:     fields,
	}}

	model.ParseSlackAttachment(post, attachments)

	return post, nil
}

// ToCard return the poll for rhs card
func (p *Poll) ToCard(bundle *utils.Bundle, convert IDToNameConverter) string {
	localizer := bundle.GetServerLocalizer()
	s := fmt.Sprintf("# %s\n", p.Question)

	if !p.Settings.AnonymousCreator {
		creatorName, _ := convert(p.Creator)
		s += fmt.Sprintf(bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{DefaultMessage: rhsCardPollCreatedBy})+" %s\n", creatorName)
	}

	const comma = ", "
	for _, o := range p.AnswerOptions {
		var voter string
		if !p.Settings.Anonymous {
			for i := 0; i < len(o.Voter); i++ {
				displayName, err := convert(o.Voter[i])
				if err != nil {
					return ""
				}
				if i+1 == len(o.Voter) && len(o.Voter) > 1 {
					voter += " " + bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{DefaultMessage: rhsCardPollVoterSeperator}) + " "
				} else if i != 0 {
					voter += comma
				}
				voter += displayName
			}
		}

		s += "### " + bundle.LocalizeWithConfig(localizer, &i18n.LocalizeConfig{
			DefaultMessage: rhsCardPollAnswerHeading,
			TemplateData: map[string]interface{}{
				"Answer": o.Answer,
				"Count":  len(o.Voter),
			},
			PluralCount: len(o.Voter),
		}) + "\n" + voter + "\n"
	}
	return s
}
