package plugin

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"

	"github.com/matterpoll/matterpoll/server/store/mockstore"
)

func TestLocalizeDefaultMessage(t *testing.T) {
	t.Run("fine", func(t *testing.T) {
		api := &plugintest.API{}

		p := setupTestPlugin(t, api, &mockstore.Store{})
		l := p.getServerLocalizer()
		m := &i18n.Message{
			Other: "test message",
		}

		assert.Equal(t, m.Other, p.LocalizeDefaultMessage(l, m))
	})
	t.Run("empty message", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogWarn", GetMockArgumentsWithType("string", 5)...).Return()
		defer api.AssertExpectations(t)

		p := setupTestPlugin(t, api, &mockstore.Store{})
		l := p.getServerLocalizer()
		m := &i18n.Message{}

		assert.Equal(t, "", p.LocalizeDefaultMessage(l, m))
	})
}

func TestLocalizeWithConfig(t *testing.T) {
	t.Run("fine", func(t *testing.T) {
		api := &plugintest.API{}

		p := setupTestPlugin(t, api, &mockstore.Store{})
		l := p.getServerLocalizer()
		lc := &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				Other: "test messsage",
			},
		}

		assert.Equal(t, lc.DefaultMessage.Other, p.LocalizeWithConfig(l, lc))
	})
	t.Run("empty config", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogWarn", GetMockArgumentsWithType("string", 3)...).Return()
		defer api.AssertExpectations(t)

		p := setupTestPlugin(t, api, &mockstore.Store{})
		l := p.getServerLocalizer()
		lc := &i18n.LocalizeConfig{}

		assert.Equal(t, "", p.LocalizeWithConfig(l, lc))
	})
	t.Run("empty message", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("LogWarn", GetMockArgumentsWithType("string", 3)...).Return()
		defer api.AssertExpectations(t)

		p := setupTestPlugin(t, api, &mockstore.Store{})
		l := p.getServerLocalizer()
		lc := &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{},
		}

		assert.Equal(t, "", p.LocalizeWithConfig(l, lc))
	})
}
