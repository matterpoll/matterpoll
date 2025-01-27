package testutils

import (
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/matterpoll/matterpoll/server/poll"
)

// GetPollID returns a static Poll ID.
func GetPollID() string {
	return "1234567890abcdefghij"
}

// GetSiteURL returns a static Site URL.
func GetSiteURL() string {
	return "https://example.org"
}

// GetBotUserID returns a static bot user ID.
func GetBotUserID() string {
	return "aegooso5na9desa0QuieV1ohfa"
}

// GetServerConfig return a static server config.
func GetServerConfig() *model.Config {
	siteURL := GetSiteURL()
	localeEn := "en"
	defaultServerLocale := localeEn
	defaultClientLocale := localeEn
	showFullName := true
	return &model.Config{
		ServiceSettings: model.ServiceSettings{
			SiteURL: &siteURL,
		},
		LocalizationSettings: model.LocalizationSettings{
			DefaultServerLocale: &defaultServerLocale,
			DefaultClientLocale: &defaultClientLocale,
		},
		PrivacySettings: model.PrivacySettings{
			ShowFullName: &showFullName,
		},
	}
}

// GetPoll returns a Poll with three Options, no votes and no Poll Settings.
func GetPoll() *poll.Poll {
	return &poll.Poll{
		ID:        GetPollID(),
		PostID:    "postID1",
		CreatedAt: 1234567890,
		Creator:   "userID1",
		Question:  "Question",
		AnswerOptions: []*poll.AnswerOption{{
			Answer: "Answer 1",
			Voter:  []string{},
		}, {
			Answer: "Answer 2",
			Voter:  []string{},
		}, {
			Answer: "Answer 3",
			Voter:  []string{},
		}},
		Settings: poll.Settings{MaxVotes: 1},
	}
}

// GetPollWithoutPostID returns a Poll with three Options, without PostID.
func GetPollWithoutPostID() *poll.Poll {
	poll := GetPoll().Copy()
	poll.PostID = ""
	return poll
}

// GetPollWithSettings returns a Poll with three Options, no votes and given Poll Settings.
func GetPollWithSettings(settings poll.Settings) *poll.Poll {
	poll := GetPoll()
	poll.Settings = settings
	return poll
}

// GetPollTwoOptionsWithSettings returns a Poll with two Options, "Yes" and "No", no votes and given Poll Settings.
func GetPollTwoOptionsWithSettings(settings poll.Settings) *poll.Poll {
	poll := GetPollTwoOptions()
	poll.Settings = settings
	return poll
}

// GetPollWithVotes returns a Poll with three Options, some votes and no Poll Settings.
func GetPollWithVotes() *poll.Poll {
	return &poll.Poll{
		ID:        GetPollID(),
		PostID:    "postID1",
		CreatedAt: 1234567890,
		Creator:   "userID1",
		Question:  "Question",
		AnswerOptions: []*poll.AnswerOption{{
			Answer: "Answer 1",
			Voter:  []string{"userID1", "userID2", "userID3"},
		}, {
			Answer: "Answer 2",
			Voter:  []string{"userID4"},
		}, {
			Answer: "Answer 3",
			Voter:  []string{},
		}},
		Settings: poll.Settings{MaxVotes: 1},
	}
}

// GetPollWithVoteUnknownUser returns a Poll with three Options, one unknown voter and no Poll Settings.
func GetPollWithVoteUnknownUser() *poll.Poll {
	return &poll.Poll{
		ID:        GetPollID(),
		PostID:    "postID1",
		CreatedAt: 1234567890,
		Creator:   "userID1",
		Question:  "Question",
		AnswerOptions: []*poll.AnswerOption{{
			Answer: "Answer 1",
			Voter:  []string{"unknowUser"},
		}, {
			Answer: "Answer 2",
			Voter:  []string{},
		}, {
			Answer: "Answer 3",
			Voter:  []string{},
		}},
		Settings: poll.Settings{MaxVotes: 1},
	}
}

// GetPollWithVotesAndSettings returns a Poll with three Options, some votes and given Poll Settings.
func GetPollWithVotesAndSettings(settings poll.Settings) *poll.Poll {
	poll := GetPollWithVotes()
	poll.Settings = settings
	return poll
}

// GetPollTwoOptions returns a Poll with two Options, "Yes" and "No", no votes and no Poll Settings.
func GetPollTwoOptions() *poll.Poll {
	return &poll.Poll{
		ID:        GetPollID(),
		PostID:    "postID1",
		CreatedAt: 1234567890,
		Creator:   "userID1",
		Question:  "Question",
		AnswerOptions: []*poll.AnswerOption{{
			Answer: "Yes",
			Voter:  []string{},
		}, {
			Answer: "No",
			Voter:  []string{},
		}},
		Settings: poll.Settings{MaxVotes: 1},
	}
}
