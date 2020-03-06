package poll_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestPollToEndPollPost(t *testing.T) {
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

	for name, test := range map[string]struct {
		Poll                *poll.Poll
		ExpectedAttachments []*model.SlackAttachment
	}{
		"Normal poll": {
			Poll: testutils.GetPollWithVotes(),
			ExpectedAttachments: []*model.SlackAttachment{{
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
			}},
		},
		"Anonymous poll": {
			Poll: testutils.GetPollWithVotesAndSettings(poll.Settings{Anonymous: true}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "This poll has ended. The results are:",
				Fields: []*model.SlackAttachmentField{{
					Title: "Answer 1 (3 votes)",
					Value: "",
					Short: true,
				}, {
					Title: "Answer 2 (1 vote)",
					Value: "",
					Short: true,
				}, {
					Title: "Answer 3 (0 votes)",
					Value: "",
					Short: true,
				}},
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			expectedPost := &model.Post{}
			model.ParseSlackAttachment(expectedPost, test.ExpectedAttachments)

			post, err := test.Poll.ToEndPollPost(testutils.GetLocalizer(), "John Doe", converter)

			require.Nil(t, err)
			assert.Equal(t, expectedPost, post)
		})
	}

	t.Run("converter fails", func(t *testing.T) {
		converter := func(userID string) (string, *model.AppError) {
			return "", &model.AppError{}
		}
		poll := testutils.GetPollWithVotes()

		post, err := poll.ToEndPollPost(testutils.GetLocalizer(), "John Doe", converter)

		assert.NotNil(t, err)
		require.Nil(t, post)
	})
}

func TestPollToPostActions(t *testing.T) {
	PluginID := "com.github.matterpoll.matterpoll"
	authorName := "John Doe"
	currentAPIVersion := "v1"

	for name, test := range map[string]struct {
		Poll                *poll.Poll
		ExpectedAttachments []*model.SlackAttachment
	}{
		"Two options": {
			Poll: testutils.GetPollTwoOptions(),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Id:   "vote0",
					Name: "Yes",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "vote1",
					Name: "No",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "addOption",
					Name: "Add Option",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "deletePoll",
					Name: "Delete Poll",
					Type: poll.MatterpollAdminButtonType,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "endPoll",
					Name: "End Poll",
					Type: poll.MatterpollAdminButtonType,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					}},
				},
			}},
		},
		"Multipile questions, settings: progress": {
			Poll: testutils.GetPollWithSettings(poll.Settings{Progress: true}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: progress\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Id:   "vote0",
					Name: "Answer 1 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "vote1",
					Name: "Answer 2 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "vote2",
					Name: "Answer 3 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "addOption",
					Name: "Add Option",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "deletePoll",
					Name: "Delete Poll",
					Type: poll.MatterpollAdminButtonType,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "endPoll",
					Name: "End Poll",
					Type: poll.MatterpollAdminButtonType,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
		"Multipile questions, settings: anonymous, public-add-option": {
			Poll: testutils.GetPollWithSettings(poll.Settings{Anonymous: true, PublicAddOption: true}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: anonymous, public-add-option\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Id:   "vote0",
					Name: "Answer 1",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "vote1",
					Name: "Answer 2",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "vote2",
					Name: "Answer 3",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "addOption",
					Name: "Add Option",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "deletePoll",
					Name: "Delete Poll",
					Type: poll.MatterpollAdminButtonType,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:   "endPoll",
					Name: "End Poll",
					Type: poll.MatterpollAdminButtonType,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.ExpectedAttachments, test.Poll.ToPostActions(testutils.GetLocalizer(), PluginID, authorName))
		})
	}
}
