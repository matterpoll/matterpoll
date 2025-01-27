package kvstore

import (
	"errors"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"

	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
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
	t.Run("Fresh install, SaveVersion fails", func(t *testing.T) {
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

	t.Run("Old install", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.UpdateDatabase("1.0.0")
		assert.Nil(t, err)
	})
	t.Run("Old install with empty upgrade", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		api.On("KVSet", versionKey, []byte("1.1.0")).Return(nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)
		store.upgrades = []*upgrade{
			{toVersion: "1.1.0", upgradeFunc: nil},
		}

		err := store.UpdateDatabase("1.0.0")
		assert.Nil(t, err)
	})
	t.Run("Old install with one upgrade", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		api.On("KVSet", versionKey, []byte("1.1.0")).Return(nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)
		store.upgrades = []*upgrade{
			{toVersion: "1.1.0", upgradeFunc: func(*Store) error { return nil }},
		}

		err := store.UpdateDatabase("1.0.0")
		assert.Nil(t, err)
	})
	t.Run("Old install with one upgrade that fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)
		store.upgrades = []*upgrade{
			{toVersion: "1.1.0", upgradeFunc: func(*Store) error { return errors.New("") }},
		}

		err := store.UpdateDatabase("1.0.0")
		assert.NotNil(t, err)
	})
	t.Run("Old install with empty upgrade, System.SaveVersion fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		api.On("KVSet", versionKey, []byte("1.1.0")).Return(&model.AppError{})
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)
		store.upgrades = []*upgrade{
			{toVersion: "1.1.0", upgradeFunc: nil},
		}

		err := store.UpdateDatabase("1.0.0")
		assert.NotNil(t, err)
	})
}

func TestUpgradeTo14(t *testing.T) {
	t.Run("KVList succeeds", func(t *testing.T) {
		oldPoll := poll.Poll{
			ID:       model.NewId(),
			Settings: poll.Settings{MaxVotes: 0},
		}

		migratedPoll := oldPoll
		migratedPoll.Settings.MaxVotes = 1

		newPoll := poll.Poll{
			ID:       model.NewId(),
			Settings: poll.Settings{MaxVotes: 2},
		}

		failGetPoll := poll.Poll{
			ID:       model.NewId(),
			Settings: poll.Settings{MaxVotes: 0},
		}

		failSavePoll := poll.Poll{
			ID:       model.NewId(),
			Settings: poll.Settings{MaxVotes: 0},
		}

		migratedFailSavePoll := failSavePoll
		migratedFailSavePoll.Settings.MaxVotes = 1

		keys := []string{
			"foo",
			pollPrefix + oldPoll.ID,
			"bar",
			pollPrefix + newPoll.ID,
			pollPrefix + failGetPoll.ID,
			pollPrefix + failSavePoll.ID,
		}

		api := &plugintest.API{}
		api.On("KVList", 0, perPage).Return(keys, nil)

		api.On("KVGet", pollPrefix+oldPoll.ID).Return(oldPoll.EncodeToByte(), nil)
		api.On("KVGet", pollPrefix+newPoll.ID).Return(newPoll.EncodeToByte(), nil)
		api.On("KVGet", pollPrefix+failGetPoll.ID).Return(nil, &model.AppError{})
		api.On("KVGet", pollPrefix+failSavePoll.ID).Return(failSavePoll.EncodeToByte(), nil)

		api.On("KVSet", pollPrefix+migratedPoll.ID, migratedPoll.EncodeToByte()).Return(nil)
		api.On("KVSet", pollPrefix+failSavePoll.ID, migratedFailSavePoll.EncodeToByte()).Return(&model.AppError{})

		api.On("LogError", testutils.GetMockArgumentsWithType("string", 5)...).Return(nil)

		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := upgradeTo14(store)

		require.NoError(t, err)
	})

	t.Run("KVList fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVList", 0, perPage).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := upgradeTo14(store)

		require.Error(t, err)
	})
}
