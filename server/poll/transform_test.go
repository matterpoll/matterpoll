package poll_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
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
		"Anonymous creator poll": {
			Poll: testutils.GetPollWithVotesAndSettings(poll.Settings{AnonymousCreator: true}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "",
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
	} {
		t.Run(name, func(t *testing.T) {
			expectedPost := &model.Post{}
			model.ParseSlackAttachment(expectedPost, test.ExpectedAttachments)

			post, err := test.Poll.ToEndPollPost(testutils.GetBundle(), "John Doe", converter)

			require.Nil(t, err)
			assert.Equal(t, expectedPost, post)
		})
	}

	t.Run("converter fails", func(t *testing.T) {
		converter := func(userID string) (string, *model.AppError) {
			return "", &model.AppError{}
		}
		poll := testutils.GetPollWithVotes()

		post, err := poll.ToEndPollPost(testutils.GetBundle(), "John Doe", converter)

		assert.NotNil(t, err)
		require.Nil(t, post)
	})
}

func TestPollToPostActions(t *testing.T) {
	PluginID := "com.github.matterpoll.matterpoll"
	authorName := "John Doe"
	currentAPIVersion := "v1"

	// single voter, multi votes
	pollWithMulti := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 3})
	pollWithMulti.AnswerOptions = []*poll.AnswerOption{
		{Answer: "Answer 1", Voter: []string{"userID1"}},
		{Answer: "Answer 2", Voter: []string{"userID1"}},
		{Answer: "Answer 3", Voter: []string{"userID1"}},
	}
	// multi voters, multi votes
	pollWithMulti2 := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 3})
	pollWithMulti2.AnswerOptions = []*poll.AnswerOption{
		{Answer: "Answer 1", Voter: []string{"userID1", "userID2", "userID3"}},
		{Answer: "Answer 2", Voter: []string{"userID1", "userID2"}},
		{Answer: "Answer 3", Voter: []string{"userID1"}},
	}

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
					Id:    "vote0",
					Name:  "Yes",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote1",
					Name:  "No",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "resetVote",
					Name:  "Reset Your Vote",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/votes/reset", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "addOption",
					Name:  "Add Option",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "endPoll",
					Name:  "End Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "deletePoll",
					Name:  "Delete Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
		"Multiple questions, settings: progress": {
			Poll: testutils.GetPollWithSettings(poll.Settings{Progress: true, MaxVotes: 1}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: progress\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Id:    "vote0",
					Name:  "Answer 1 (0)",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote1",
					Name:  "Answer 2 (0)",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote2",
					Name:  "Answer 3 (0)",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "resetVote",
					Name:  "Reset Your Vote",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/votes/reset", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "addOption",
					Name:  "Add Option",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "endPoll",
					Name:  "End Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "deletePoll",
					Name:  "Delete Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
		"Multiple questions, settings: anonymous, public-add-option": {
			Poll: testutils.GetPollWithSettings(poll.Settings{Anonymous: true, PublicAddOption: true, MaxVotes: 1}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: anonymous, public-add-option\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Id:    "vote0",
					Name:  "Answer 1",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote1",
					Name:  "Answer 2",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote2",
					Name:  "Answer 3",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "resetVote",
					Name:  "Reset Your Vote",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/votes/reset", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "addOption",
					Name:  "Add Option",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "endPoll",
					Name:  "End Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "deletePoll",
					Name:  "Delete Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
		"Multipile questions, settings: anonymous, anonymous-creator": {
			Poll: testutils.GetPollWithSettings(poll.Settings{Anonymous: true, AnonymousCreator: true, MaxVotes: 1}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: anonymous, anonymous-creator\n**Total votes**: 0",
				Actions: []*model.PostAction{{
					Id:    "vote0",
					Name:  "Answer 1",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote1",
					Name:  "Answer 2",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote2",
					Name:  "Answer 3",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "resetVote",
					Name:  "Reset Your Vote",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/votes/reset", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "addOption",
					Name:  "Add Option",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "endPoll",
					Name:  "End Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "deletePoll",
					Name:  "Delete Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
		"Multipile questions, settings: votes=3": {
			Poll: pollWithMulti,
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: votes=3\n**Total votes**: 3 (1 voter)",
				Actions: []*model.PostAction{{
					Id:    "vote0",
					Name:  "Answer 1",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote1",
					Name:  "Answer 2",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote2",
					Name:  "Answer 3",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "resetVote",
					Name:  "Reset Your Votes",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/votes/reset", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "addOption",
					Name:  "Add Option",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "endPoll",
					Name:  "End Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "deletePoll",
					Name:  "Delete Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
		"Multiple questions, settings: votes=3, multiple voters": {
			Poll: pollWithMulti2,
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: votes=3\n**Total votes**: 6 (3 voters)",
				Actions: []*model.PostAction{{
					Id:    "vote0",
					Name:  "Answer 1",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote1",
					Name:  "Answer 2",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote2",
					Name:  "Answer 3",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "resetVote",
					Name:  "Reset Your Votes",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/votes/reset", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "addOption",
					Name:  "Add Option",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "endPoll",
					Name:  "End Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "deletePoll",
					Name:  "Delete Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
		"Multiple questions, settings: votes=0": {
			Poll: testutils.GetPollWithSettings(poll.Settings{MaxVotes: 0}),
			ExpectedAttachments: []*model.SlackAttachment{{
				AuthorName: "John Doe",
				Title:      "Question",
				Text:       "---\n**Poll Settings**: votes=all\n**Total votes**: 0 (0 voters)",
				Actions: []*model.PostAction{{
					Id:    "vote0",
					Name:  "Answer 1",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/0", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote1",
					Name:  "Answer 2",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/1", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "vote2",
					Name:  "Answer 3",
					Type:  model.PostActionTypeButton,
					Style: "default",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/vote/2", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "resetVote",
					Name:  "Reset Your Votes",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/votes/reset", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "addOption",
					Name:  "Add Option",
					Type:  model.PostActionTypeButton,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/option/add/request", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "endPoll",
					Name:  "End Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/end", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				}, {
					Id:    "deletePoll",
					Name:  "Delete Poll",
					Type:  poll.MatterpollAdminButtonType,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/%s/polls/%s/delete", PluginID, currentAPIVersion, testutils.GetPollID()),
					},
				},
				},
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.ExpectedAttachments, test.Poll.ToPostActions(testutils.GetBundle(), PluginID, authorName))
		})
	}
}

func TestToCard(t *testing.T) {
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
		Poll             *poll.Poll
		ExpectedMarkdown string
	}{
		"Normal poll": {
			Poll: testutils.GetPollWithVotes(),
			ExpectedMarkdown: "# Question\n" +
				"Created by @user1\n" +
				"### Answer 1 (3 votes)" +
				"\n@user1, @user2 and @user3\n" +
				"### Answer 2 (1 vote)" +
				"\n@user4\n" +
				"### Answer 3 (0 votes)" +
				"\n\n",
		},
		"Anonymous poll": {
			Poll: testutils.GetPollWithVotesAndSettings(poll.Settings{Anonymous: true}),
			ExpectedMarkdown: "# Question\n" +
				"Created by @user1\n" +
				"### Answer 1 (3 votes)" +
				"\n\n" +
				"### Answer 2 (1 vote)" +
				"\n\n" +
				"### Answer 3 (0 votes)" +
				"\n\n",
		},
		"Anonymous creator poll": {
			Poll: testutils.GetPollWithVotesAndSettings(poll.Settings{AnonymousCreator: true}),
			ExpectedMarkdown: "# Question\n" +
				"### Answer 1 (3 votes)" +
				"\n@user1, @user2 and @user3\n" +
				"### Answer 2 (1 vote)" +
				"\n@user4\n" +
				"### Answer 3 (0 votes)" +
				"\n\n",
		},
		"Normal poll, with error in voter name convert": {
			Poll:             testutils.GetPollWithVoteUnknownUser(),
			ExpectedMarkdown: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.ExpectedMarkdown, test.Poll.ToCard(testutils.GetBundle(), converter))
		})
	}
}
