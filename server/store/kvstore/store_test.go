package kvstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

const latestVersion = "1.3.0"

func setupTestStore(api plugin.API) *Store {
	store := Store{
		api: api,
		pollStore: PollStore{
			api: api,
		},
		systemStore: SystemStore{
			api: api,
		},
		upgrades: nil,
	}
	return &store
}

func TestNewStore(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte(latestVersion), nil)
		defer api.AssertExpectations(t)

		store, err := NewStore(api, latestVersion)
		assert.Nil(t, err)
		assert.NotNil(t, store)
	})
	t.Run("UpdateDatabase() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte{}, &model.AppError{})
		defer api.AssertExpectations(t)

		store, err := NewStore(api, latestVersion)
		assert.NotNil(t, err)
		assert.Nil(t, store)
	})
}
