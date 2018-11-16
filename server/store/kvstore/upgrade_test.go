package kvstore

import (
	"testing"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
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
	t.Run("Old install", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.UpdateDatabase("1.0.0")
		assert.Nil(t, err)
	})
	t.Run("Fresh install", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte(""), nil)
		api.On("KVSet", versionKey, []byte("1.0.0")).Return(nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.UpdateDatabase("1.0.0")
		assert.Nil(t, err)
	})
	t.Run("Fresh install on patch release", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte(""), nil)
		api.On("KVSet", versionKey, []byte("1.0.0")).Return(nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.UpdateDatabase("1.0.1")
		assert.Nil(t, err)
	})
	t.Run("Fresh install, KVSet fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte(""), nil)
		api.On("KVSet", versionKey, []byte("1.0.0")).Return(&model.AppError{})
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.UpdateDatabase("1.0.0")
		assert.NotNil(t, err)
	})
	t.Run("System.GetVersion fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte{}, &model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.UpdateDatabase("1.0.0")
		assert.NotNil(t, err)
	})
}
