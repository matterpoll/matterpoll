package poll

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	// MatterpollAdminButtonType is action_type of buttons that are used for managing a poll.
	MatterpollAdminButtonType = "custom_matterpoll_admin_button"
)

// IDToNameConverter converts a given userID to a human readable name.
type IDToNameConverter func(userID string) (string, *model.AppError)

var (
	pollButtonAddOption = &i18n.Message{
		ID:    "poll.button.addOption",
		Other: "Add Option",
	}
	pollButtonDeletePoll = &i18n.Message{
		ID:    "poll.button.deletePoll",
		Other: "Delete Poll",
	}
	pollButtonEndPoll = &i18n.Message{
		ID:    "poll.button.endPoll",
		Other: "End Poll",
	}

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
	pollEndPostAnswerHeading = &i18n.Message{
		ID:    "poll.endPost.answer.heading",
		One:   "{{.Answer}} ({{.Count}} vote)",
		Other: "{{.Answer}} ({{.Count}} votes)",
	}
)

// TOOD
func (p *Poll) ToProps(localizer *i18n.Localizer, pluginID string, convert IDToNameConverter) map[string]interface{} {
	props := make(map[string]interface{})

	creatorName, _ := convert(p.Creator)
	props["poll_id"] = p.ID
	props["attachments"] = p.ToPostActions(localizer, pluginID, creatorName)
	props["card"] = p.ToCard(localizer, convert)

	return props
}

// ToPostActions returns the poll as a message
func (p *Poll) ToPostActions(localizer *i18n.Localizer, pluginID, authorName string) []*model.SlackAttachment {
	numberOfVotes := 0
	actions := []*model.PostAction{}

	for i, o := range p.AnswerOptions {
		numberOfVotes += len(o.Voter)
		actions = append(actions, &model.PostAction{
			Id:   fmt.Sprintf("vote%v", i),
			Name: p.getAnswerOptionName(o),
			Type: model.POST_ACTION_TYPE_BUTTON,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/vote/%v", pluginID, p.ID, i),
			},
		})
	}

	actions = append(actions,
		&model.PostAction{
			Id:   "addOption",
			Name: localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: pollButtonAddOption}),
			Type: model.POST_ACTION_TYPE_BUTTON,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/option/add/request", pluginID, p.ID),
			},
		}, &model.PostAction{
			Id:   "deletePoll",
			Name: localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: pollButtonDeletePoll}),
			Type: MatterpollAdminButtonType,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/delete", pluginID, p.ID),
			},
		}, &model.PostAction{
			Id:   "endPoll",
			Name: localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: pollButtonEndPoll}),
			Type: MatterpollAdminButtonType,
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/polls/%s/end", pluginID, p.ID),
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

	attachments := []*model.SlackAttachment{{
		AuthorName: authorName,
		Title:      p.Question,
		Text:       localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: pollEndPostText}),
		Fields:     fields,
	}}
	model.ParseSlackAttachment(post, attachments)

	return post, nil
}

// ToPostActions returns the poll as a message
func (p *Poll) ToCard(localizer *i18n.Localizer, convert IDToNameConverter) string {
	s := fmt.Sprintf("# %s\n", p.Question)

	creatorName, _ := convert(p.Creator)
	s += fmt.Sprintf("Created by %s\n", creatorName)

	for _, o := range p.AnswerOptions {
		var voter string
		if !p.Settings.Anonymous {
			for i := 0; i < len(o.Voter); i++ {
				displayName, err := convert(o.Voter[i])
				if err != nil {
					return ""
				}
				if i+1 == len(o.Voter) && len(o.Voter) > 1 {
					voter += " " + localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: pollEndPostSeperator}) + " "
				} else if i != 0 {
					voter += ", "
				}
				voter += displayName
			}
		}

		s += "### " + localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: pollEndPostAnswerHeading,
			TemplateData: map[string]interface{}{
				"Answer": o.Answer,
				"Count":  len(o.Voter),
			},
			PluralCount: len(o.Voter),
		}) + "\n" + voter + "\n"
	}
	return s
}
