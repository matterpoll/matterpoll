package poll_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mpatch "github.com/undefinedlabs/go-mpatch"

	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestNewPoll(t *testing.T) {
	assert := assert.New(t)
	var createdAt int64 = 1234567890
	patch1, _ := mpatch.PatchMethod(model.GetMillis, func() int64 { return createdAt })
	patch2, _ := mpatch.PatchMethod(model.NewId, testutils.GetPollID)
	defer func() { require.NoError(t, patch1.Unpatch()) }()
	defer func() { require.NoError(t, patch2.Unpatch()) }()

	creator := model.NewRandomString(10)
	question := model.NewRandomString(10)
	option1 := "Yes"
	option2 := "No"
	option3 := "Other"
	option4 := "" // invalid: empty option

	for name, test := range map[string]struct {
		Options     []string
		Settings    poll.Settings
		ShouldError bool
	}{
		"fine, default settings": {
			Options:     []string{option1, option2},
			Settings:    poll.Settings{Anonymous: false, Progress: false, PublicAddOption: false, MaxVotes: 1},
			ShouldError: false,
		},
		"fine, all settings": {
			Options:     []string{option1, option2, option3},
			Settings:    poll.Settings{Anonymous: true, Progress: true, PublicAddOption: true, MaxVotes: 3},
			ShouldError: false,
		},
		"fine, votes=0": {
			Options:     []string{option1, option2},
			Settings:    poll.Settings{Anonymous: true, Progress: true, PublicAddOption: true, MaxVotes: 0},
			ShouldError: false,
		},
		"invalid, duplicated options": {
			Options:     []string{option1, option2, option1}, // options1 is duplicated
			Settings:    poll.Settings{Anonymous: true, Progress: true, PublicAddOption: true, MaxVotes: 1},
			ShouldError: true,
		},
		"invalid, empty option": {
			Options:     []string{option1, option2, option4}, // options1 is duplicated
			Settings:    poll.Settings{Anonymous: true, Progress: true, PublicAddOption: true, MaxVotes: 1},
			ShouldError: true,
		},
		"invalid, votes setting exceeds": {
			Options:     []string{option1, option2, option3}, // options1 is duplicated
			Settings:    poll.Settings{Anonymous: true, Progress: true, PublicAddOption: true, MaxVotes: 4},
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, err := poll.NewPoll(creator, question, test.Options, test.Settings)

			if test.ShouldError {
				assert.Nil(p)
				assert.NotNil(err)
			} else {
				assert.Nil(err)
				assert.NotNil(p)
				assert.Equal(testutils.GetPollID(), p.ID)
				assert.Equal(createdAt, p.CreatedAt)
				assert.Equal(creator, p.Creator)
				assert.Equal(question, p.Question)

				assert.Equal(len(test.Options), len(p.AnswerOptions))
				for i, o := range p.AnswerOptions {
					assert.Equal(&poll.AnswerOption{Answer: test.Options[i], Voter: []string{}}, o)
				}

				assert.Equal(test.Settings, p.Settings)
			}
		})
	}
}

func TestNewSettingsFromStrings(t *testing.T) {
	for name, test := range map[string]struct {
		Strs             []string
		ShouldError      bool
		ExpectedSettings poll.Settings
	}{
		"no settings": {
			Strs:        []string{},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				MaxVotes:         1,
			},
		},
		"full settings": {
			Strs:        []string{"anonymous", "anonymous-creator", "progress", "public-add-option", "votes=4"},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        true,
				AnonymousCreator: true,
				Progress:         true,
				PublicAddOption:  true,
				MaxVotes:         4,
			},
		},
		"without votes settings": {
			Strs:        []string{"anonymous", "progress", "public-add-option"},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        true,
				AnonymousCreator: false,
				Progress:         true,
				PublicAddOption:  true,
				MaxVotes:         1,
			},
		},
		"invalid votes setting": {
			Strs:        []string{"votes=9223372036854775808"}, // Exceed math.MaxInt64
			ShouldError: true,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				MaxVotes:         1,
			},
		},
		"invalid setting": {
			Strs:        []string{"anonymous", "progress", "public-add-option", "invalid"},
			ShouldError: true,
			ExpectedSettings: poll.Settings{
				Anonymous:        true,
				AnonymousCreator: false,
				Progress:         true,
				PublicAddOption:  true,
				MaxVotes:         1,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			settings, errMsg := poll.NewSettingsFromStrings(test.Strs)
			if test.ShouldError {
				assert.NotNil(errMsg)
			} else {
				assert.Nil(errMsg)
			}
			assert.Equal(test.ExpectedSettings, settings)
		})
	}
}

func TestNewSettingsFromSubmission(t *testing.T) {
	for name, test := range map[string]struct {
		Submission       map[string]interface{}
		ExpectedSettings poll.Settings
	}{
		"no settings": {
			Submission: map[string]interface{}{},
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				MaxVotes:         1,
			},
		},
		"full settings": {
			Submission: map[string]interface{}{
				"setting-anonymous":         true,
				"setting-anonymous-creator": true,
				"setting-progress":          true,
				"setting-public-add-option": true,
				"setting-multi":             float64(4),
			},
			ExpectedSettings: poll.Settings{
				Anonymous:        true,
				AnonymousCreator: true,
				Progress:         true,
				PublicAddOption:  true,
				MaxVotes:         4,
			},
		},
		"without votes settings": {
			Submission: map[string]interface{}{
				"setting-anonymous":         false,
				"setting-progress":          false,
				"setting-public-add-option": false,
			},
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				MaxVotes:         1,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			settings := poll.NewSettingsFromSubmission(test.Submission)
			assert.Equal(test.ExpectedSettings, settings)
		})
	}
}

func TestIsMultiVote(t *testing.T) {
	assert := assert.New(t)
	for name, test := range map[string]struct {
		Poll     poll.Poll
		Expected bool
	}{
		"single vote": {
			Poll: poll.Poll{
				ID:            testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{{Answer: "Answer 1"}, {Answer: "Answer 2"}},
				Settings:      poll.Settings{MaxVotes: 1},
			},
			Expected: false,
		},
		"multi vote": {
			Poll: poll.Poll{
				ID:            testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{{Answer: "Answer 1"}, {Answer: "Answer 2"}},
				Settings:      poll.Settings{MaxVotes: 2},
			},
			Expected: true,
		},
		"multi vote 0": {
			Poll: poll.Poll{
				ID:            testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{{Answer: "Answer 1"}, {Answer: "Answer 2"}},
				Settings:      poll.Settings{MaxVotes: 0},
			},
			Expected: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(test.Expected, test.Poll.IsMultiVote())
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
				Settings: poll.Settings{MaxVotes: 1},
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
		"Multi votes setting, with progress option, duplicated vote error": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2"},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{Progress: true, MaxVotes: 2},
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
				Settings: poll.Settings{Progress: true, MaxVotes: 2},
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
		"Multi votes setting (--votes=0), third vote": {
			Poll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3"},
				},
				Settings: poll.Settings{MaxVotes: 0},
			},
			UserID: "a",
			Index:  2,
			ExpectedPoll: poll.Poll{
				Question: "Question",
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"a"}},
					{Answer: "Answer 3", Voter: []string{"a"}},
				},
				Settings: poll.Settings{MaxVotes: 0},
			},
			Error:         false,
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
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			test.Poll.ResetVotes(test.UserID)
			assert.Equal(test.ExpectedPoll, test.Poll)
		})
	}
}

func TestGetMetadata(t *testing.T) {
	for name, test := range map[string]struct {
		Poll             poll.Poll
		UserID           string
		Permission       bool
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
			UserID:     "a",
			Permission: true,
			ExpectedResponse: &poll.Metadata{
				PollID:        testutils.GetPollID(),
				UserID:        "a",
				CanManagePoll: true,
				VotedAnswers:  []string{"Answer 1"},
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
			UserID:     "b",
			Permission: true,
			ExpectedResponse: &poll.Metadata{
				PollID:        testutils.GetPollID(),
				UserID:        "b",
				CanManagePoll: true,
				VotedAnswers:  []string{"Answer 2", "Answer 3"},
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
			UserID:     "c",
			Permission: true,
			ExpectedResponse: &poll.Metadata{
				PollID:        testutils.GetPollID(),
				UserID:        "c",
				CanManagePoll: true,
				VotedAnswers:  []string{},
			}},
		"With all settings": {
			Poll: poll.Poll{
				ID: testutils.GetPollID(),
				AnswerOptions: []*poll.AnswerOption{
					{Answer: "Answer 1", Voter: []string{"a"}},
					{Answer: "Answer 2", Voter: []string{"b"}},
					{Answer: "Answer 3", Voter: []string{"b"}},
				},
				Settings: poll.Settings{
					Anonymous:       true,
					Progress:        true,
					PublicAddOption: true,
					MaxVotes:        3,
				},
			},
			UserID:     "c",
			Permission: true,
			ExpectedResponse: &poll.Metadata{
				PollID:                 testutils.GetPollID(),
				UserID:                 "c",
				CanManagePoll:          true,
				VotedAnswers:           []string{},
				SettingProgress:        true,
				SettingPublicAddOption: true,
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
			UserID: "",
			ExpectedResponse: &poll.Metadata{
				PollID:        testutils.GetPollID(),
				UserID:        "",
				CanManagePoll: false,
				VotedAnswers:  []string{},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			metadata := test.Poll.GetMetadata(test.UserID, test.Permission)
			assert.Equal(test.ExpectedResponse, metadata)
		})
	}
}

func TestHasVoted(t *testing.T) {
	p1 := &poll.Poll{Question: "Question",
		AnswerOptions: []*poll.AnswerOption{
			{Answer: "Answer 1", Voter: []string{"a"}},
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

func TestSettingsString(t *testing.T) {
	assert := assert.New(t)
	for name, test := range map[string]struct {
		Settings poll.Settings
		Expected string
	}{
		"anonymous": {
			Settings: poll.Settings{Anonymous: true, MaxVotes: 1},
			Expected: "anonymous",
		},
		"anonymous-creator": {
			Settings: poll.Settings{AnonymousCreator: true, MaxVotes: 1},
			Expected: "anonymous-creator",
		},
		"progress": {
			Settings: poll.Settings{Progress: true, MaxVotes: 1},
			Expected: "progress",
		},
		"public-add-option": {
			Settings: poll.Settings{PublicAddOption: true, MaxVotes: 1},
			Expected: "public-add-option",
		},
		"default votes": {
			Settings: poll.Settings{MaxVotes: 1},
			Expected: "",
		},
		"votes": {
			Settings: poll.Settings{MaxVotes: 2},
			Expected: "votes=2",
		},
		"all votes": {
			Settings: poll.Settings{MaxVotes: 0},
			Expected: "votes=all",
		},
		"all": {
			Settings: poll.Settings{Anonymous: true, AnonymousCreator: true, Progress: true, PublicAddOption: true, MaxVotes: 2},
			Expected: "anonymous, anonymous-creator, progress, public-add-option, votes=2",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(test.Expected, test.Settings.String())
		})
	}
}
