package poll_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/matterpoll/matterpoll/server/poll"
)

func TestToMap(t *testing.T) {
	m := poll.Metadata{
		PollID:          "pollID",
		UserID:          "userID",
		AdminPermission: true,
	}

	expectedMap := map[string]interface{}{
		"poll_id":          "pollID",
		"user_id":          "userID",
		"admin_permission": true,
		"voted_answers":    []string(nil),
	}
	assert.Equal(t, expectedMap, m.ToMap())
}
