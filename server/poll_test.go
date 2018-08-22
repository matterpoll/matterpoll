package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	p1 := &Poll{
		Question: "Question",
		Options: []*Option{
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
				Options: []*Option{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  -1,
			ExpectedPoll: Poll{
				Question: "Question",
				Options: []*Option{
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
				Options: []*Option{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  2,
			ExpectedPoll: Poll{
				Question: "Question",
				Options: []*Option{
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
				Options: []*Option{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "",
			Index:  1,
			ExpectedPoll: Poll{
				Question: "Question",
				Options: []*Option{
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
				Options: []*Option{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  0,
			ExpectedPoll: Poll{
				Question: "Question",
				Options: []*Option{
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
				Options: []*Option{
					{Answer: "Answer 1",
						Voter: []string{"a"}},
					{Answer: "Answer 2"},
				},
			},
			UserID: "a",
			Index:  1,
			ExpectedPoll: Poll{
				Question: "Question",
				Options: []*Option{
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

			err := test.Poll.UpdateVote(test.UserID, test.Index)
			if test.Error {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.ExpectedPoll, test.Poll)
		})
	}
}

func TestHasVoted(t *testing.T) {
	p1 := &Poll{Question: "Question",
		Options: []*Option{
			{Answer: "Answer 1",
				Voter: []string{"a"}},
			{Answer: "Answer 2"},
		},
	}
	assert.True(t, p1.HasVoted("a"))
	assert.False(t, p1.HasVoted("b"))
}
