package poll

// Metadata stores personalized metadata of a poll.
type Metadata struct {
	VotedAnswers           []string `json:"voted_answers"` // VotedAnswers is list of answer that the user with "UserID" have voted for the poll with "PollID"
	PollID                 string   `json:"poll_id"`
	UserID                 string   `json:"user_id"`
	CanManagePoll          bool     `json:"can_manage_poll"` // CanManagePoll will be true if the user with "UserID" can manage the poll with "PollID", otherwise false.
	SettingPublicAddOption bool     `json:"setting_public_add_option"`
}

// ToMap returns a Metadata as a map
func (m *Metadata) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"voted_answers":             m.VotedAnswers,
		"poll_id":                   m.PollID,
		"user_id":                   m.UserID,
		"can_manage_poll":           m.CanManagePoll,
		"setting_public_add_option": m.SettingPublicAddOption,
	}
}
