package poll_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
func TestPollWithProgress(t *testing.T) {
	PluginID := "com.github.matterpoll.matterpoll"
	authorName := "John Doe"
	//currentAPIVersion := "v1"

	for name, test := range map[string]struct {
		Poll *poll.Poll
	}{
		"Test1": {
			Poll: testutils.GetPollWithSettings(poll.Settings{Progress: true}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := test.Poll.UpdateVote(testutils.GetBotUserID(), 1)
			require.Nil(t, err)

			err = test.Poll.UpdateVote("bar", 1)
			require.Nil(t, err)

			err = test.Poll.UpdateVote("foo", 0)
			require.Nil(t, err)

			post := test.Poll.ToPostActions(testutils.GetLocalizer(), PluginID, authorName)
			require.NotNil(t, post)

			postText := post[0].Text
			require.GreaterOrEqual(t, len(post), 1)
			//check if the correct percentages are visible
			require.Contains(t, postText, fmt.Sprintf("%3d %%", 33))
			require.Contains(t, postText, fmt.Sprintf("%3d %%", 66))
			require.Contains(t, postText, fmt.Sprintf("%3d %%", 0))

			//check if the progressbars are correctly generated
			lines := strings.Split(postText, "\n")
			require.GreaterOrEqual(t, len(lines), 4)

			filled := strings.Count(lines[1], "█")

			filled += strings.Count(lines[2], "█")

			filled += strings.Count(lines[3], "█")

			//This value should be close to the total length of a progress bar (32 chars), it might be a littel less due to rounding errors
			require.GreaterOrEqual(t, filled, 31)
		})
	}
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
					Name: "Yes",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "No",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Add Option",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					}},
				},
			}},
		},
		//XXX: Hardcoding this  might be suboptimal in the future, if the format change in any way.
		"Multipile questions, settings: progress": {
			Poll: testutils.GetPollWithSettings(poll.Settings{Progress: true}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n`░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░`\tAnswer 1\t`  0 %`\n`░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░`\tAnswer 2\t`  0 %`\n`░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░`\tAnswer 3\t`  0 %`\n**Poll Settings**: progress\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Name: "Answer 1 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Answer 2 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Answer 3 (0)",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Add Option",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
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
					Name: "Answer 1",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Answer 2",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Answer 3",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Add Option",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "Delete Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Name: "End Poll",
					Type: model.POST_ACTION_TYPE_BUTTON,
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
