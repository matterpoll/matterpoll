package poll_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/matterpoll/matterpoll/server/poll"
)

func TestToMap(t *testing.T) {
	m := poll.Metadata{
		PollID:                 "pollID",
		UserID:                 "userID",
		CanManagePoll:          true,
		SettingProgress:        true,
		SettingPublicAddOption: true,
	}

	expectedMap := map[string]interface{}{
		"voted_answers":             []string(nil),
		"poll_id":                   "pollID",
		"user_id":                   "userID",
		"can_manage_poll":           true,
		"setting_progress":          true,
		"setting_public_add_option": true,
		"setting_poll_duration":     0,
	}
	assert.Equal(t, expectedMap, m.ToMap())
}
