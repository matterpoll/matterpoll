package main

import (
	"testing"

	"github.com/bouk/monkey"
	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPoll(t *testing.T) {
	assert := assert.New(t)
	patch := monkey.Patch(model.GetMillis, func() int64 { return 1234567890 })
	defer patch.Unpatch()

	creator := model.NewRandomString(10)
	question := model.NewRandomString(10)
	answerOptions := []string{model.NewRandomString(10), model.NewRandomString(10), model.NewRandomString(10)}
	p := NewPoll(creator, question, answerOptions)

	require.NotNil(t, p)
	assert.Equal(int64(1234567890), p.CreatedAt)
	assert.Equal(creator, p.Creator)
	assert.Equal(CurrentDataSchemaVersion, p.DataSchemaVersion)
	assert.Equal(question, p.Question)
	assert.Equal(&AnswerOption{Answer: answerOptions[0], Voter: nil}, p.AnswerOptions[0])
	assert.Equal(&AnswerOption{Answer: answerOptions[1], Voter: nil}, p.AnswerOptions[1])
	assert.Equal(&AnswerOption{Answer: answerOptions[2], Voter: nil}, p.AnswerOptions[2])
}

func TestEncodeDecode(t *testing.T) {
	p1 := &Poll{
		Question: "Question",
		AnswerOptions: []*AnswerOption{
			{Answer: "Answer 1"},
			{Answer: "Answer 2"},
		},
	}
	p2 := Decode(p1.Encode())
	assert.Equal(t, p1, p2)
}

func TestUpdateVote(t *testing.T) {
	for name, test := range map[string]struct {
		Poll         Poll
		UserID       string
		Index        int
		ExpectedPoll Poll
		Error        bool
	}{
		"Negative Index": {
			Poll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  -1,
			ExpectedPoll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error: true,
		},
		"To high Index": {
			Poll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  2,
			ExpectedPoll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error: true,
		},
		"Invalid userID": {
			Poll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "",
			Index:  1,
			ExpectedPoll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error: true,
		},
		"Idempotent": {
			Poll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  0,
			ExpectedPoll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			Error: false,
		},
		"Valid Vote": {
			Poll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  1,
			ExpectedPoll: Poll{
				Question: "Question",
				AnswerOptions: []*AnswerOption{
					{Answer: "Answer 1",
						Voter: []string{}},
					{Answer: "Answer 2",
						Voter: []string{"a"}},
				},
			},
			Error: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			err := test.Poll.UpdateVote(test.UserID, test.Index)

			if test.Error {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
			assert.Equal(test.ExpectedPoll, test.Poll)
		})
	}
}

func TestHasVoted(t *testing.T) {
	p1 := &Poll{Question: "Question",
		AnswerOptions: []*AnswerOption{
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
		p := &samplePoll
		p2 := p.Copy()

		assert.Equal(p, p2)
	})
	t.Run("change Question", func(t *testing.T) {
		p := &samplePoll
		p2 := p.Copy()

		p.Question = "Different question"
		assert.NotEqual(p.Question, p2.Question)
		assert.NotEqual(p, p2)
	})
	t.Run("change AnswerOptions", func(t *testing.T) {
		p := &samplePoll
		p2 := p.Copy()

		p.AnswerOptions[0].Answer = "abc"
		assert.NotEqual(p.AnswerOptions[0].Answer, p2.AnswerOptions[0].Answer)
		assert.NotEqual(p, p2)
	})

}
