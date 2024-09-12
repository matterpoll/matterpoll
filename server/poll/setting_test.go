package poll_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/matterpoll/matterpoll/server/poll"
)

func TestNewSettingsFromStrings(t *testing.T) {
	testTime := time.Now().Add(time.Minute * time.Duration(5)).Round(time.Second)
	testTimeExpected := time.Now().Add(time.Minute * time.Duration(5)).UTC().Round(time.Second)
	testTimeTruncated := time.Now().Add(time.Minute * time.Duration(5)).UTC().Round(time.Second).Truncate(time.Minute)

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
		"valid end date setting": {
			Strs:        []string{fmt.Sprintf("end=%s", testTime.Format(poll.EndSettingTimezoneLayout))},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				End:              &testTimeTruncated,
				MaxVotes:         1,
			},
		},
		"valid end duration setting": {
			Strs:        []string{fmt.Sprintf("end=%s", "5m")},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				End:              &testTimeExpected,
				MaxVotes:         1,
			},
		},
		"invalid end setting": {
			Strs:        []string{fmt.Sprintf("end=%s", testTime.Add(-time.Hour*time.Duration(1)).Format(poll.EndSettingTimezoneLayout))},
			ShouldError: true,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				End:              nil,
				MaxVotes:         1,
			},
		},
		"invalid end setting layout": {
			Strs:        []string{fmt.Sprintf("end=%s", testTime.Format(time.TimeOnly))},
			ShouldError: true,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				End:              nil,
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
	testTime := time.Now().Add(time.Minute * time.Duration(5)).Round(time.Second)
	testTimeExpected := time.Now().Add(time.Minute * time.Duration(5)).UTC().Round(time.Second)
	testTimeTruncated := time.Now().Add(time.Minute * time.Duration(5)).UTC().Round(time.Second).Truncate(time.Minute)

	for name, test := range map[string]struct {
		Submission       map[string]interface{}
		ShouldError      bool
		ExpectedSettings poll.Settings
	}{
		"no settings": {
			Submission:  map[string]interface{}{},
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
			Submission: map[string]interface{}{
				"setting-anonymous":         true,
				"setting-anonymous-creator": true,
				"setting-progress":          true,
				"setting-public-add-option": true,
				"setting-multi":             float64(4),
			},
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
			Submission: map[string]interface{}{
				"setting-anonymous":         false,
				"setting-progress":          false,
				"setting-public-add-option": false,
			},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				MaxVotes:         1,
			},
		},
		"with end date settings": {
			Submission: map[string]interface{}{
				"setting-anonymous":         false,
				"setting-progress":          false,
				"setting-public-add-option": false,
				"setting-end":               testTime.Format(poll.EndSettingTimezoneLayout),
			},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				MaxVotes:         1,
				End:              &testTimeTruncated,
			},
		},
		"with end duration settings": {
			Submission: map[string]interface{}{
				"setting-anonymous":         false,
				"setting-progress":          false,
				"setting-public-add-option": false,
				"setting-end":               "5m",
			},
			ShouldError: false,
			ExpectedSettings: poll.Settings{
				Anonymous:        false,
				AnonymousCreator: false,
				Progress:         false,
				PublicAddOption:  false,
				MaxVotes:         1,
				End:              &testTimeExpected,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			settings, errMsg := poll.NewSettingsFromSubmission(test.Submission)
			if test.ShouldError {
				assert.NotNil(errMsg)
			} else {
				assert.Nil(errMsg)
			}
			assert.Equal(test.ExpectedSettings, settings)
		})
	}
}

func TestSettingsString(t *testing.T) {
	t.Run("anonymous", func(t *testing.T) {
		s := poll.Settings{Anonymous: true}
		str := s.String()

		assert.Equal(t, str, "anonymous")
	})
	t.Run("anonymous-creator", func(t *testing.T) {
		s := poll.Settings{AnonymousCreator: true}
		str := s.String()

		assert.Equal(t, str, "anonymous-creator")
	})
	t.Run("progress", func(t *testing.T) {
		s := poll.Settings{Progress: true}
		str := s.String()

		assert.Equal(t, str, "progress")
	})
	t.Run("public-add-option", func(t *testing.T) {
		s := poll.Settings{PublicAddOption: true}
		str := s.String()

		assert.Equal(t, str, "public-add-option")
	})
	t.Run("default votes", func(t *testing.T) {
		s := poll.Settings{MaxVotes: 1}
		str := s.String()

		assert.Equal(t, str, "")
	})
	t.Run("votes", func(t *testing.T) {
		s := poll.Settings{MaxVotes: 2}
		str := s.String()

		assert.Equal(t, str, "votes=2")
	})
	t.Run("all", func(t *testing.T) {
		s := poll.Settings{Anonymous: true, AnonymousCreator: true, Progress: true, PublicAddOption: true, MaxVotes: 2}
		str := s.String()

		assert.Equal(t, str, "anonymous, anonymous-creator, progress, public-add-option, votes=2")
	})
}
