package testutils

import (
	"github.com/matterpoll/matterpoll/server/poll"
)

func GetPollID() string {
	return "1234567890abcdefghij"
}

func GetSiteURL() string {
	return "https://example.org"
}

func GetPluginDirectory() string {
	return "plugins"
}

func GetPoll() *poll.Poll {
	return &poll.Poll{
		ID:        GetPollID(),
		CreatedAt: 1234567890,
		Creator:   "userID1",
		Question:  "Question",
		AnswerOptions: []*poll.AnswerOption{
			{Answer: "Answer 1"},
			{Answer: "Answer 2"},
			{Answer: "Answer 3"},
		},
	}
}

func GetPollWithSettings(settings poll.PollSettings) *poll.Poll {
	poll := GetPoll()
	poll.Settings = settings
	return poll
}

func GetPollWithVotes() *poll.Poll {
	return &poll.Poll{
		ID:        GetPollID(),
		CreatedAt: 1234567890,
		Creator:   "userID1",
		Question:  "Question",
		AnswerOptions: []*poll.AnswerOption{
			{Answer: "Answer 1",
				Voter: []string{"userID1", "userID2", "userID3"}},
			{Answer: "Answer 2",
				Voter: []string{"userID4"}},
			{Answer: "Answer 3"},
		},
	}
}

func GetPollWithVotesAndSettings(settings poll.PollSettings) *poll.Poll {
	poll := GetPollWithVotes()
	poll.Settings = settings
	return poll
}

func GetPollTwoOptions() *poll.Poll {
	return &poll.Poll{
		ID:        GetPollID(),
		CreatedAt: 1234567890,
		Creator:   "userID1",
		Question:  "Question",
		AnswerOptions: []*poll.AnswerOption{
			{Answer: "Yes"},
			{Answer: "No"},
		},
	}
}
