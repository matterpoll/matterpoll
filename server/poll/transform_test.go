package poll_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollToEndPollPost(t *testing.T) {
	expectedPost := &model.Post{}
	model.ParseSlackAttachment(expectedPost, []*model.SlackAttachment{{
		AuthorName: "John Doe",
		Title:      "Question",
		Text:       "This poll has ended. The results are:",
		Fields: []*model.SlackAttachmentField{{
			Title: "Answer 1 (3 votes)",
			Value: "@user1, @user2 and @user3",
			Short: true,
		}, {
			Title: "Answer 2 (1 vote)",
			Value: "@user4",
			Short: true,
		}, {
			Title: "Answer 3 (0 votes)",
			Value: "",
			Short: true,
		}},
	}})

	converter := func(userID string) (string, *model.AppError) {
		switch userID {
		case "userID1":
			return "@user1", nil
		case "userID2":
			return "@user2", nil
		case "userID3":
			return "@user3", nil
		case "userID4":
			return "@user4", nil
		default:
			return "", &model.AppError{}
		}

	}

	post, err := testutils.GetPollWithVotes().ToEndPollPost("John Doe", converter)

	require.Nil(t, err)
	assert.Equal(t, expectedPost, post)
}

func TestPollToPostActions(t *testing.T) {
	siteURL := "https://example.org"
	PluginID := "com.github.matterpoll.matterpoll"
	pollID := "1234567890abcdefghij"
	authorName := "John Doe"
	currentAPIVersion := "v1"

	for name, test := range map[string]struct {
		Poll                *poll.Poll
		ExpectedAttachments []*model.SlackAttachment
	}{
		"No argument": {
			Poll: testutils.GetPollTwoOptions(),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Name: "Yes",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", siteURL, PluginID, currentAPIVersion, pollID),
					},
				}, {
					Name: "No",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", siteURL, PluginID, currentAPIVersion, pollID),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginID, currentAPIVersion, pollID),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginID, currentAPIVersion, pollID),
					}},
				},
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.ExpectedAttachments, test.Poll.ToPostActions(siteURL, PluginID, pollID, authorName))
		})
	}

	t.Run("multipile questions, settings: progress", func(t *testing.T) {
		expectedAttachments := []*model.SlackAttachment{{
			AuthorName: "John Doe",
			Title:      "Question",
			Text:       "---\n**Poll settings**: progress\n**Total votes**: 0",
			Actions: []*model.PostAction{{
				Name: "Answer 1 (0)",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "Answer 2 (0)",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "Answer 3 (0)",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/2", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "Delete Poll",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "End Poll",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginID, currentAPIVersion, pollID),
				},
			},
			},
		}}
		p := testutils.GetPoll()
		p.Settings.Progress = true
		assert.Equal(t, expectedAttachments, p.ToPostActions(siteURL, PluginID, pollID, authorName))
	})

	t.Run("multipile questions, settings: anonymous", func(t *testing.T) {
		expectedAttachments := []*model.SlackAttachment{{
			AuthorName: "John Doe",
			Title:      "Question",
			Text:       "---\n**Poll settings**: anonymous\n**Total votes**: 0",
			Actions: []*model.PostAction{{
				Name: "Answer 1",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/0", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "Answer 2",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/1", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "Answer 3",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/vote/2", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "Delete Poll",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/delete", siteURL, PluginID, currentAPIVersion, pollID),
				},
			}, {
				Name: "End Poll",
				Type: model.POST_ACTION_TYPE_BUTTON,
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s/plugins/%s/api/%s/polls/%s/end", siteURL, PluginID, currentAPIVersion, pollID),
				},
			},
			},
		}}
		p := testutils.GetPoll()
		p.Settings.Anonymous = true
		assert.Equal(t, expectedAttachments, p.ToPostActions(siteURL, PluginID, pollID, authorName))
	})
}
