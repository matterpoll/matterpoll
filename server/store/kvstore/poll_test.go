package kvstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollStoreGet(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", pollPrefix+testutils.GetPollID()).Return(testutils.GetPoll().EncodeToByte(), nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		rpoll, err := store.Poll().Get(testutils.GetPollID())
		require.Nil(t, err)
		assert.Equal(t, testutils.GetPoll(), rpoll)
	})
	t.Run("KVGet() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", pollPrefix+testutils.GetPollID()).Return([]byte{}, &model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		rpoll, err := store.Poll().Get(testutils.GetPollID())
		assert.NotNil(t, err)
		assert.Nil(t, rpoll)
	})
	t.Run("Decode fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", pollPrefix+testutils.GetPollID()).Return([]byte{}, nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		rpoll, err := store.Poll().Get(testutils.GetPollID())
		assert.NotNil(t, err)
		assert.Nil(t, rpoll)
	})
}

func TestPollStoreSave(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVSet", pollPrefix+testutils.GetPollID(), testutils.GetPoll().EncodeToByte()).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Save(testutils.GetPoll())
		require.Nil(t, err)
	})
	t.Run("KVSet() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVSet", pollPrefix+testutils.GetPollID(), testutils.GetPoll().EncodeToByte()).Return(&model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Save(testutils.GetPoll())
		require.NotNil(t, err)
	})
}

func TestPollStoreDelete(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVDelete", pollPrefix+testutils.GetPollID()).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Delete(testutils.GetPoll())
		require.Nil(t, err)
	})
	t.Run("KVDelete() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVDelete", pollPrefix+testutils.GetPollID()).Return(&model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Delete(testutils.GetPoll())
		require.NotNil(t, err)
	})
}
