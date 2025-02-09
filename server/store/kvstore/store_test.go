package kvstore

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
)

const latestVersion = "1.8.0"

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
