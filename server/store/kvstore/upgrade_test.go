package kvstore

import (
	"testing"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStoreShouldPerformUpgrade(t *testing.T) {
	t.Run("Should upgrade", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		b := store.shouldPerformUpgrade(semver.MustParse("1.0.0"), semver.MustParse("1.1.0"))
		assert.True(t, b)
	})
	t.Run("shouldn't upgrade", func(t *testing.T) {
		api := &plugintest.API{}
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		b := store.shouldPerformUpgrade(semver.MustParse("1.0.0"), semver.MustParse("1.0.0"))
		assert.False(t, b)
	})
}

func TestStoreUpdateDatabase(t *testing.T) {
	t.Run("Fresh install", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte(""), nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.UpdateDatabase()
		assert.Nil(t, err)
	})
}
