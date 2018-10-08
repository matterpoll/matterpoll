package poll

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

// ToPostActions returns the poll as a message
func (p *Poll) ToPostActions(siteURL, pluginID, authorName string) []*model.SlackAttachment {
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
				URL: fmt.Sprintf("%s/plugins/%s/api/v1/polls/%s/vote/%v", siteURL, pluginID, p.ID, i),
			},
		})
	}

	actions = append(actions, &model.PostAction{
		Name: "Delete Poll",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/v1/polls/%s/delete", siteURL, pluginID, p.ID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "End Poll",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/v1/polls/%s/end", siteURL, pluginID, p.ID),
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

// ToEndPollPost returns the poll end message
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
