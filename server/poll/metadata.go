package poll

// Metadata stores personalized metadata of a poll.
type Metadata struct {
	PollID          string   `json:"poll_id"`
	UserID          string   `json:"user_id"`
	AdminPermission bool     `json:"admin_permission"` // AdminPermission will be true if the user with "UserID" has admin permission for the poll with "PollID", otherwise false.
	VotedAnswers    []string `json:"voted_answers"`    // VotedAnswers is list of answer that the user with "UserID" have voted for the poll with "PollID"
}

// ToMap returns a Metadata as a map
func (m *Metadata) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"poll_id":          m.PollID,
		"user_id":          m.UserID,
		"admin_permission": m.AdminPermission,
		"voted_answers":    m.VotedAnswers,
	}
}
