package kvstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemStoreGetVersion(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		version, err := store.System().GetVersion()
		require.Nil(t, err)
		assert.Equal(t, "1.0.0", version)
	})
	t.Run("KVGet() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte{}, &model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		version, err := store.System().GetVersion()
		assert.NotNil(t, err)
		assert.Equal(t, "", version)
	})
}

func TestSystemStoreSetVersion(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVSet", versionKey, []byte("1.0.0")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.System().SaveVersion("1.0.0")
		assert.Nil(t, err)
	})
	t.Run("KVSet() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVSet", versionKey, []byte("1.0.0")).Return(&model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.System().SaveVersion("1.0.0")
		assert.NotNil(t, err)
	})
}
