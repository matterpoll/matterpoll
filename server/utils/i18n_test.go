package utils_test

import (
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"

	"github.com/matterpoll/matterpoll/server/utils"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestLocalizeDefaultMessage(t *testing.T) {
	t.Run("fine", func(t *testing.T) {
		b := testutils.GetBundle()
		l := b.GetServerLocalizer()
		m := &i18n.Message{
			Other: "test message",
		}

		assert.Equal(t, m.Other, b.LocalizeDefaultMessage(l, m))
	})
}

func TestLocalizeWithConfig(t *testing.T) {
	t.Run("fine", func(t *testing.T) {
		b := testutils.GetBundle()
		l := b.GetServerLocalizer()
		lc := &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				Other: "test messsage",
			},
		}

		assert.Equal(t, lc.DefaultMessage.Other, b.LocalizeWithConfig(l, lc))
	})
	t.Run("empty config", func(t *testing.T) {
		b := testutils.GetBundle()
		l := b.GetServerLocalizer()
		lc := &i18n.LocalizeConfig{}

		assert.Equal(t, "", b.LocalizeWithConfig(l, lc))
	})
	t.Run("ids missmatch", func(t *testing.T) {
		b := testutils.GetBundle()
		l := b.GetServerLocalizer()
		lc := &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: "some ID",
			},
			MessageID: "some other ID",
		}

		assert.Equal(t, "", b.LocalizeWithConfig(l, lc))
	})
}

func TestLocalizeErrorMessage(t *testing.T) {
	t.Run("fine, with no params", func(t *testing.T) {
		b := testutils.GetBundle()
		l := b.GetServerLocalizer()
		m := &utils.ErrorMessage{
			Message: &i18n.Message{
				Other: "test message",
			},
			Data: map[string]interface{}{},
		}

		assert.Equal(t, m.Message.Other, b.LocalizeErrorMessage(l, m))
	})
	t.Run("fine, with params", func(t *testing.T) {
		b := testutils.GetBundle()
		l := b.GetServerLocalizer()
		m := &utils.ErrorMessage{
			Message: &i18n.Message{
				Other: "test message {{.Param1}}, {{.Param2}}",
			},
			Data: map[string]interface{}{
				"Param1": "p1",
				"Param2": "p2",
			},
		}

		assert.Equal(t, "test message p1, p2", b.LocalizeErrorMessage(l, m))
	})
	t.Run("empty message", func(t *testing.T) {
		b := testutils.GetBundle()
		l := b.GetServerLocalizer()
		m := &utils.ErrorMessage{}

		assert.Equal(t, "", b.LocalizeErrorMessage(l, m))
	})
}
