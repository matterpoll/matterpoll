package poll_test

import (
	"fmt"
	"testing"

	"bou.ke/monkey"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestNewPoll(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		assert := assert.New(t)
		patch1 := monkey.Patch(model.GetMillis, func() int64 { return 1234567890 })
		patch2 := monkey.Patch(model.NewId, testutils.GetPollID)
		defer patch1.Unpatch()
		defer patch2.Unpatch()

		creator := model.NewRandomString(10)
		question := model.NewRandomString(10)
		answerOptions := []string{model.NewRandomString(10), model.NewRandomString(10), model.NewRandomString(10)}
		p, err := poll.NewPoll(creator, question, answerOptions, []string{"anonymous", "progress", "public-add-option", "votes=3"})

		require.Nil(t, err)
		require.NotNil(t, p)
		assert.Equal(testutils.GetPollID(), p.ID)
		assert.Equal(int64(1234567890), p.CreatedAt)
		assert.Equal(creator, p.Creator)
		assert.Equal(question, p.Question)
		assert.Equal(&poll.AnswerOption{Answer: answerOptions[0], Voter: []string{}}, p.AnswerOptions[0])
		assert.Equal(&poll.AnswerOption{Answer: answerOptions[1], Voter: []string{}}, p.AnswerOptions[1])
		assert.Equal(&poll.AnswerOption{Answer: answerOptions[2], Voter: []string{}}, p.AnswerOptions[2])
		assert.Equal(poll.Settings{Anonymous: true, Progress: true, PublicAddOption: true, MaxVotes: 3}, p.Settings)
	})
	t.Run("fine, without votes setting", func(t *testing.T) {
		assert := assert.New(t)
		patch1 := monkey.Patch(model.GetMillis, func() int64 { return 1234567890 })
		patch2 := monkey.Patch(model.NewId, testutils.GetPollID)
		defer patch1.Unpatch()
		defer patch2.Unpatch()

		creator := model.NewRandomString(10)
		question := model.NewRandomString(10)
		answerOptions := []string{model.NewRandomString(10), model.NewRandomString(10), model.NewRandomString(10)}
		p, err := poll.NewPoll(creator, question, answerOptions, []string{"anonymous", "progress", "public-add-option"})

		require.Nil(t, err)
		require.NotNil(t, p)
		assert.Equal(testutils.GetPollID(), p.ID)
		assert.Equal(int64(1234567890), p.CreatedAt)
		assert.Equal(creator, p.Creator)
		assert.Equal(question, p.Question)
		assert.Equal(&poll.AnswerOption{Answer: answerOptions[0], Voter: []string{}}, p.AnswerOptions[0])
		assert.Equal(&poll.AnswerOption{Answer: answerOptions[1], Voter: []string{}}, p.AnswerOptions[1])
		assert.Equal(&poll.AnswerOption{Answer: answerOptions[2], Voter: []string{}}, p.AnswerOptions[2])
		assert.Equal(poll.Settings{Anonymous: true, Progress: true, PublicAddOption: true, MaxVotes: 1}, p.Settings)
	})
	t.Run("error, invalid votes setting", func(t *testing.T) {
		assert := assert.New(t)
		patch1 := monkey.Patch(model.GetMillis, func() int64 { return 1234567890 })
		patch2 := monkey.Patch(model.NewId, testutils.GetPollID)
		defer patch1.Unpatch()
		defer patch2.Unpatch()

		creator := model.NewRandomString(10)
		question := model.NewRandomString(10)
		answerOptions := []string{model.NewRandomString(10), model.NewRandomString(10), model.NewRandomString(10)}
		p, err := poll.NewPoll(creator, question, answerOptions, []string{"anonymous", "progress", "public-add-option", "votes=4"})

		assert.Nil(p)
		assert.NotNil(err)
	})
	t.Run("error, unknown setting", func(t *testing.T) {
		assert := assert.New(t)

		creator := model.NewRandomString(10)
		question := model.NewRandomString(10)
		answerOptions := []string{model.NewRandomString(10), model.NewRandomString(10), model.NewRandomString(10)}
		p, err := poll.NewPoll(creator, question, answerOptions, []string{"unkownOption"})

		assert.Nil(p)
		assert.NotNil(err)
	})

	t.Run("error, duplicate option", func(t *testing.T) {
		assert := assert.New(t)

		creator := model.NewRandomString(10)
		question := model.NewRandomString(10)
		option := model.NewRandomString(10)
		answerOptions := []string{option, model.NewRandomString(10), option}
		p, err := poll.NewPoll(creator, question, answerOptions, nil)

		assert.Nil(p)
		assert.NotNil(err)
	})
}

func TestParseVotesSetting(t *testing.T) {
	p := testutils.GetPollTwoOptions()
	for name, test := range map[string]struct {
		Poll  *poll.Poll
		S     string
		Votes int
		Error bool
	}{
		"fine setting": {
			Poll:  p,
			S:     "votes=2",
			Votes: 2,
			Error: false,
		},
		"error, votes=0": {
			Poll:  p,
			S:     "votes=0",
			Error: true,
		},
		"error, exceed the number of options": {
			Poll:  p,
			S:     "votes=3",
			Error: true,
		},
		"error, invalid argument": {
			Poll:  p,
			S:     "invalid",
			Error: true,
		},
		"error, invalid argument 2": {
			Poll:  p,
			S:     "votes=2abc",
			Error: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			err := test.Poll.ParseVotesSetting(test.S)
			if test.Error {
				assert.NotNil((err))
			} else {
				assert.Nil((err))
				assert.Equal(test.Votes, test.Poll.Settings.MaxVotes)
			}
		})
	}
}

func TestAddAnswerOption(t *testing.T) {
	assert := assert.New(t)

	t.Run("all fine", func(t *testing.T) {
		p := testutils.GetPollWithVotes()

		err := p.AddAnswerOption("new option")
		assert.Nil(err)
		assert.Equal("new option", p.AnswerOptions[len(p.AnswerOptions)-1].Answer)
	})
	t.Run("dublicant options", func(t *testing.T) {
		p := testutils.GetPollWithVotes()

		err := p.AddAnswerOption(p.AnswerOptions[0].Answer)
		assert.NotNil(err)
	})
	t.Run("dublicant options with spaces", func(t *testing.T) {
		p := testutils.GetPollWithVotes()

		err := p.AddAnswerOption(fmt.Sprintf(" %s ", p.AnswerOptions[0].Answer))
		assert.NotNil(err)
	})
	t.Run("empty options", func(t *testing.T) {
		p := testutils.GetPollWithVotes()

		err := p.AddAnswerOption("")
		assert.NotNil(err)
	})
	t.Run("empty optinos with spaces", func(t *testing.T) {
		p := testutils.GetPollWithVotes()

		err := p.AddAnswerOption("  ")
		assert.NotNil(err)
	})
}

func TestEncodeDecode(t *testing.T) {
	p1 := testutils.GetPollWithVotes()
	p2 := poll.DecodePollFromByte(p1.EncodeToByte())
	assert.Equal(t, p1, p2)
}

func TestDecode(t *testing.T) {
	p := poll.DecodePollFromByte([]byte{})
	assert.Nil(t, p)
}

func TestUpdateVote(t *testing.T) {
	for name, test := range map[string]struct {
		Poll          poll.Poll
		UserID        string
		Index         int
		ExpectedPoll  poll.Poll
		Error         bool
		ReturnMessage bool
	}{
		"Negative Index": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  -1,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error:         true,
			ReturnMessage: false,
		},
		"To high Index": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  2,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error:         true,
			ReturnMessage: false,
		},
		"Invalid userID": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "",
			Index:  1,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error:         true,
			ReturnMessage: false,
		},
		"Idempotent": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  0,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error:         false,
			ReturnMessage: false,
		},
		"Valid Vote": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  1,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{}},
					{Answer: "Answer 2",
						Voter: []string{"a"}},
				},
			},
			Error:         false,
			ReturnMessage: false,
		},
		"Multi votes setting, first vote": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1"},
					{Answer: "Answer 2"},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			UserID: "a",
			Index:  0,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2"},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			Error:         false,
			ReturnMessage: false,
		},
		"Multi votes setting, second vote": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2"},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			UserID: "a",
			Index:  1,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			Error:         false,
			ReturnMessage: false,
		},
		"Multi votes setting, duplicated vote error": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2"},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			UserID: "a",
			Index:  0,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2"},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			Error:         false,
			ReturnMessage: true,
		},
		"Multi votes setting, exceed votes error": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			UserID: "a",
			Index:  2,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			Error:         false,
			ReturnMessage: true,
		},
		"Multi votes setting, invalid user id error": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			UserID: "",
			Index:  2,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 2},
			},
			Error:         true,
			ReturnMessage: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			msg, err := test.Poll.UpdateVote(test.UserID, test.Index)

			if test.Error {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}

			if test.ReturnMessage {
				assert.NotNil(msg)
			} else {
				assert.Nil(msg)
			}
			assert.Equal(test.ExpectedPoll, test.Poll)
		})
	}
}

func TestResetVotes(t *testing.T) {
	for name, test := range map[string]struct {
		Poll         poll.Poll
		UserID       string
		ExpectedPoll poll.Poll
		Error        bool
	}{
		"Reset success, with votes": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3", Voter: []string{"a"}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			UserID: "a",
			ExpectedPoll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{}},
					{Answer: "Answer 2", Voter: []string{}},
					{Answer: "Answer 3", Voter: []string{}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			Error: false,
		},
		"Reset success, with no votes": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{}},
					{Answer: "Answer 2", Voter: []string{}},
					{Answer: "Answer 3", Voter: []string{}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			UserID: "a",
			ExpectedPoll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{}},
					{Answer: "Answer 2", Voter: []string{}},
					{Answer: "Answer 3", Voter: []string{}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			Error: false,
		},
		"Reset success, with votes from multi user": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a", "b"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3", Voter: []string{"1", "a", "z"}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			UserID: "a",
			ExpectedPoll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"b"}},
					{Answer: "Answer 2", Voter: []string{}},
					{Answer: "Answer 3", Voter: []string{"1", "z"}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			Error: false,
		},
		"invalid user id": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			UserID: "",
			ExpectedPoll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
				},
				Settings: poll.Settings{MaxVotes: 3},
			},
			Error: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			err := test.Poll.ResetVotes(test.UserID)
			if test.Error {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
			assert.Equal(test.ExpectedPoll, test.Poll)
		})
	}
}

func TestGetMetadata(t *testing.T) {
	for name, test := range map[string]struct {
		Poll             poll.Poll
		UserID           string
		Permission       bool
		ShouldError      bool
		ExpectedResponse *poll.Metadata
	}{
		"Voted an Answer": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"b"}},
					{Answer: "Answer 3", Voter: []string{"b"}},
				},
			},
			UserID:      "a",
			Permission:  true,
			ShouldError: false,
			ExpectedResponse: &poll.Metadata{
				PollID:          testutils.GetPollID(),
				UserID:          "a",
				AdminPermission: true,
				VotedAnswers:    []string{"Answer 1"},
			},
		},
		"Voted two Answers": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"b"}},
					{Answer: "Answer 3", Voter: []string{"b"}},
				},
			},
			UserID:      "b",
			Permission:  true,
			ShouldError: false,
			ExpectedResponse: &poll.Metadata{
				PollID:          testutils.GetPollID(),
				UserID:          "b",
				AdminPermission: true,
				VotedAnswers:    []string{"Answer 2", "Answer 3"},
			},
		},
		"Voted no Answers": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"b"}},
					{Answer: "Answer 3", Voter: []string{"b"}},
				},
			},
			UserID:      "c",
			Permission:  true,
			ShouldError: false,
			ExpectedResponse: &poll.Metadata{
				PollID:          testutils.GetPollID(),
				UserID:          "c",
				AdminPermission: true,
				VotedAnswers:    []string{},
			}},
		"Invalid userID": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"b"}},
					{Answer: "Answer 3", Voter: []string{"b"}},
				},
			},
			UserID:      "",
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			metadata, err := test.Poll.GetMetadata(test.UserID, test.Permission)
			if test.ShouldError {
				assert.NotNil(err)
				assert.Nil(metadata)
			} else {
				assert.Nil(err)
				assert.Equal(test.ExpectedResponse, metadata)
			}
		})
	}
}

func TestHasVoted(t *testing.T) {
	p1 := &poll.Poll{Question: "Question",
		AnswerOptions: []*poll.AnswerOption{
			{Answer: "Answer 1",
				Voter: []string{"a"}},
			{Answer: "Answer 2"},
		},
	}
	assert.True(t, p1.HasVoted("a"))
	assert.False(t, p1.HasVoted("b"))
}

func TestPollCopy(t *testing.T) {
	assert := assert.New(t)

	t.Run("no change", func(t *testing.T) {
		p := testutils.GetPoll()
		p2 := p.Copy()

		assert.Equal(p, p2)
	})
	t.Run("change Question", func(t *testing.T) {
		p := testutils.GetPoll()
		p2 := p.Copy()

		p.Question = "Different question"
		assert.NotEqual(p.Question, p2.Question)
		assert.NotEqual(p, p2)
		assert.Equal(testutils.GetPoll(), p2)
	})
	t.Run("change AnswerOptions", func(t *testing.T) {
		p := testutils.GetPoll()
		p2 := p.Copy()

		p.AnswerOptions[0].Answer = "abc"
		assert.NotEqual(p.AnswerOptions[0].Answer, p2.AnswerOptions[0].Answer)
		assert.NotEqual(p, p2)
		assert.Equal(testutils.GetPoll(), p2)
	})
	t.Run("change Voter", func(t *testing.T) {
		p := testutils.GetPollWithVotes()
		p2 := p.Copy()

		msg, err := p.UpdateVote("userID1", 0)
		require.Nil(t, msg)
		require.NoError(t, err)
		assert.NotEqual(p, p2)
		assert.Equal(testutils.GetPollWithVotes(), p2)
	})
	t.Run("change Settings", func(t *testing.T) {
		p := testutils.GetPoll()
		p2 := p.Copy()

		p.Settings.Progress = true
		assert.NotEqual(p.Settings.Progress, p2.Settings.Progress)
		assert.NotEqual(p, p2)
		assert.Equal(testutils.GetPoll(), p2)
	})
}
