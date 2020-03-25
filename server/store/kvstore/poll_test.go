package kvstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestPollStoreGet(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", pollPrefix+testutils.GetPollID()).Return(testutils.GetPoll().EncodeToByte(), nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		rpoll, err := store.Poll().Get(testutils.GetPollID())
		require.NoError(t, err)
		assert.Equal(t, testutils.GetPoll(), rpoll)
	})
	t.Run("KVGet() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", pollPrefix+testutils.GetPollID()).Return([]byte{}, &model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		rpoll, err := store.Poll().Get(testutils.GetPollID())
		assert.Error(t, err)
		assert.Nil(t, rpoll)
	})
	t.Run("Decode fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", pollPrefix+testutils.GetPollID()).Return([]byte{}, nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		rpoll, err := store.Poll().Get(testutils.GetPollID())
		assert.Error(t, err)
		assert.Nil(t, rpoll)
	})
}

func TestPollStoreInsert(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		opt := model.PluginKVSetOptions{
			Atomic:   true,
			OldValue: nil,
		}
		api := &plugintest.API{}
		api.On("KVSetWithOptions", pollPrefix+testutils.GetPollID(), testutils.GetPoll().EncodeToByte(), opt).Return(true, nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Insert(testutils.GetPoll())
		require.NoError(t, err)
	})
	t.Run("KVSetWithOptions() fails", func(t *testing.T) {
		opt := model.PluginKVSetOptions{
			Atomic:   true,
			OldValue: nil,
		}
		api := &plugintest.API{}
		api.On("KVSetWithOptions", pollPrefix+testutils.GetPollID(), testutils.GetPoll().EncodeToByte(), opt).Return(false, &model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Insert(testutils.GetPoll())
		require.Error(t, err)
	})
	t.Run("Poll already exists", func(t *testing.T) {
		opt := model.PluginKVSetOptions{
			Atomic:   true,
			OldValue: nil,
		}
		api := &plugintest.API{}
		api.On("KVSetWithOptions", pollPrefix+testutils.GetPollID(), testutils.GetPoll().EncodeToByte(), opt).Return(false, nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Insert(testutils.GetPoll())
		require.Error(t, err)
	})
}

func TestPollStoreUpdate(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		oldPoll := testutils.GetPoll()
		newPoll := oldPoll.Copy()
		err := newPoll.UpdateVote(model.NewId(), 0)
		require.NoError(t, err)
		opt := model.PluginKVSetOptions{
			Atomic:   true,
			OldValue: oldPoll.EncodeToByte(),
		}

		api := &plugintest.API{}
		api.On("KVSetWithOptions", pollPrefix+newPoll.ID, newPoll.EncodeToByte(), opt).Return(true, nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err = store.Poll().Update(oldPoll, newPoll)
		require.NoError(t, err)
	})
	t.Run("KVSetWithOptions() fails", func(t *testing.T) {
		oldPoll := testutils.GetPoll()
		newPoll := oldPoll.Copy()
		err := newPoll.UpdateVote(model.NewId(), 0)
		require.NoError(t, err)
		opt := model.PluginKVSetOptions{
			Atomic:   true,
			OldValue: oldPoll.EncodeToByte(),
		}

		api := &plugintest.API{}
		api.On("KVSetWithOptions", pollPrefix+newPoll.ID, newPoll.EncodeToByte(), opt).Return(false, &model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err = store.Poll().Update(oldPoll, newPoll)
		require.Error(t, err)
	})
	t.Run("db compare fails fails", func(t *testing.T) {
		oldPoll := testutils.GetPoll()
		newPoll := oldPoll.Copy()
		err := newPoll.UpdateVote(model.NewId(), 0)
		require.NoError(t, err)
		opt := model.PluginKVSetOptions{
			Atomic:   true,
			OldValue: oldPoll.EncodeToByte(),
		}

		api := &plugintest.API{}
		api.On("KVSetWithOptions", pollPrefix+newPoll.ID, newPoll.EncodeToByte(), opt).Return(false, nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err = store.Poll().Update(oldPoll, newPoll)
		require.Error(t, err)
	})
}

func TestPollStoreDelete(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVDelete", pollPrefix+testutils.GetPollID()).Return(nil)
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Delete(testutils.GetPoll())
		require.NoError(t, err)
	})
	t.Run("KVDelete() fails", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVDelete", pollPrefix+testutils.GetPollID()).Return(&model.AppError{})
		defer api.AssertExpectations(t)
		store := setupTestStore(api)

		err := store.Poll().Delete(testutils.GetPoll())
		require.Error(t, err)
	})
}
