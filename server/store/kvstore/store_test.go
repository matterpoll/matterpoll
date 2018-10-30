package kvstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func setupTestStore(api *plugintest.API) *Store {
	store := Store{
		api: api,
		pollStore: PollStore{
			api: api,
		},
		systemStore: SystemStore{
			api: api,
		},
	}
	return &store
}

func TestNewStore(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		defer api.AssertExpectations(t)

		store, err := NewStore(api, "1.0.0")
		assert.Nil(t, err)
		assert.NotNil(t, store)
	})
	t.Run("UpdateDatabase() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte{}, &model.AppError{})
		defer api.AssertExpectations(t)

		store, err := NewStore(api, "1.0.0")
		assert.NotNil(t, err)
		assert.Nil(t, store)
	})
}
