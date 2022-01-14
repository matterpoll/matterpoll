package poll

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/i18n"
	"github.com/mattermost/mattermost-server/v5/model"
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

	pollEndPostText = &i18n.Message{
		ID:    "poll.endPost.text",
		Other: "This poll has ended. The results are:",
	}
	pollEndPostSeperator = &i18n.Message{
		ID:    "poll.endPost.seperator",
		Other: "and",
	}
)

// ToPostActions returns the poll as a message
func (p *Poll) ToPostActions(localizer *i18n.Localizer, pluginID, authorName string) []*model.SlackAttachment {
	numberOfVotes := 0
	actions := []*model.PostAction{}

	for i, o := range p.AnswerOptions {
		numberOfVotes += len(o.Voter)
		actions = append(actions, &model.PostAction{
			Id:    fmt.Sprintf("vote%v", i),
			Name:  p.getAnswerOptionName(o),
			Type:  model.POST_ACTION_TYPE_BUTTON,
			Style: "default",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/vote/%v", pluginID, p.ID, i),
			},
		})
	}

	actions = append(actions,
		&model.PostAction{
			Id: "resetVote",
			Name: localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "poll.button.resetVotes",
					One:   "Reset your vote",
					Few:   "Reset your votes",
					Many:  "Reset your votes",
					Other: "Reset your votes",
				},
				PluralCount: p.Settings.MaxVotes,
			}),
			Type:  model.POST_ACTION_TYPE_BUTTON,
			Style: "primary",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/votes/reset", pluginID, p.ID),
			},
		},
	)
	actions = append(actions,
		&model.PostAction{
			Id: "addOption",
			Name: localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{
				ID:    "poll.button.addOption",
				Other: "Add Option",
			}}),
			Type:  model.POST_ACTION_TYPE_BUTTON,
			Style: "primary",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/option/add/request", pluginID, p.ID),
			},
		}, &model.PostAction{
			Id: "endPoll",
			Name: localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{
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
			Name: localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{
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

	return []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       p.makeAdditionalText(localizer, numberOfVotes),
		Actions:    actions,
	}}
}

// makeAdditionalText make descriptions about poll
// This method returns markdown text, because it is used for SlackAttachment.Text field.
func (p *Poll) makeAdditionalText(localizer *i18n.Localizer, numberOfVotes int) string {
	var settingsText []string
	if p.Settings.Anonymous {
		settingsText = append(settingsText, "anonymous")
	}
	if p.Settings.Progress {
		settingsText = append(settingsText, "progress")
	}
	if p.Settings.PublicAddOption {
		settingsText = append(settingsText, "public-add-option")
	}
	if p.Settings.MaxVotes > 1 {
		settingsText = append(settingsText, fmt.Sprintf("votes=%d", p.Settings.MaxVotes))
	}

	lines := []string{"---"}
	if len(settingsText) > 0 {
		lines = append(lines, localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: pollMessageSettings,
			TemplateData:   map[string]interface{}{"Settings": strings.Join(settingsText, ", ")},
		}))
	}

	lines = append(lines, localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: pollMessageTotalVotes,
		TemplateData:   map[string]interface{}{"TotalVotes": numberOfVotes},
	}))
	return strings.Join(lines, "\n")
}

// ToEndPollPost returns the poll end message
func (p *Poll) ToEndPollPost(localizer *i18n.Localizer, authorName string, convert IDToNameConverter) (*model.Post, *model.AppError) {
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
					voter += " " + localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: pollEndPostSeperator}) + " "
				} else if i != 0 {
					voter += ", "
				}
				voter += displayName
			}
		}

		fields = append(fields, &model.SlackAttachmentField{
			Short: true,
			Title: localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "poll.endPost.answer.heading",
					One:   "{{.Answer}} ({{.Count}} vote)",
					Few:   "{{.Answer}} ({{.Count}} votes)",
					Many:  "{{.Answer}} ({{.Count}} votes)",
					Other: "{{.Answer}} ({{.Count}} votes)",
				},
				TemplateData: map[string]interface{}{
					"Answer": o.Answer,
					"Count":  len(o.Voter),
				},
				PluralCount: len(o.Voter),
			}),
			Value: voter,
		})
	}

	attachments := []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: pollEndPostText}),
		Fields:     fields,
	}}
	model.ParseSlackAttachment(post, attachments)

	return post, nil
}
