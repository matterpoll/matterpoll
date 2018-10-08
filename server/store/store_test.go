package store

import (
	"testing"

	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	t.Run("all fine", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", versionKey).Return([]byte("1.0.0"), nil)
		store, err := NewStore(api)
		require.Nil(t, err)
		require.NotNil(t, store)

		version, err := store.System().GetVersion()
		require.Nil(t, err)
		assert.Equal(t, "1.0.0", version)
	})
}
