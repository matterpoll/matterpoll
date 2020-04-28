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
		AdminPermission:        true,
		SettingPublicAddOption: true,
	}

	expectedMap := map[string]interface{}{
		"voted_answers":             []string(nil),
		"poll_id":                   "pollID",
		"user_id":                   "userID",
		"admin_permission":          true,
		"setting_public_add_option": true,
	}
	assert.Equal(t, expectedMap, m.ToMap())
}
