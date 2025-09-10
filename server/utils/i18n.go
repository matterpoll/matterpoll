package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"golang.org/x/text/language"
)

// Bundle stores a set ot messages and translates messages.
type Bundle struct {
	*i18n.Bundle
	api plugin.API
}

// ErrorMessage contains error messsage for a user that can be localized.
// It should not be wrapped and instead always returned.
type ErrorMessage struct {
	Message *i18n.Message
	Data    map[string]interface{}
}

// InitBundle loads all localization files in i18n into a bundle and return this
func InitBundle(api plugin.API, path string) (*Bundle, error) {
	b := &Bundle{
		Bundle: i18n.NewBundle(language.English),
		api:    api,
	}
	b.RegisterUnmarshalFunc("json", json.Unmarshal)

	bundlePath, err := b.api.GetBundlePath()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bundle path")
	}

	i18nDir := filepath.Join(bundlePath, path)
	files, err := os.ReadDir(i18nDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open i18n directory")
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "active.") {
			continue
		}

		if file.Name() == "active.en.json" {
			continue
		}
		_, err = b.LoadMessageFile(filepath.Join(i18nDir, file.Name()))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load message file %s", file.Name())
		}
	}

	return b, nil
}

// GetUserLocalizer returns a localizer that localizes in the users locale
func (b *Bundle) GetUserLocalizer(userID string) *i18n.Localizer {
	user, err := b.api.GetUser(userID)
	if err != nil {
		b.api.LogWarn("Failed get user's locale", "error", err.Error())
		return b.GetServerLocalizer()
	}

	return i18n.NewLocalizer(b.Bundle, user.Locale)
}

// GetServerLocalizer returns a localizer that localizes in the server default client locale
func (b *Bundle) GetServerLocalizer() *i18n.Localizer {
	return i18n.NewLocalizer(
		b.Bundle,
		*b.api.GetConfig().LocalizationSettings.DefaultClientLocale,
	)
}

// LocalizeDefaultMessage localizer the provided message
func (b *Bundle) LocalizeDefaultMessage(l *i18n.Localizer, m *i18n.Message) string {
	s, err := l.LocalizeMessage(m)
	if err != nil {
		b.api.LogWarn("Failed to localize message", "message ID", m.ID, "error", err.Error())
	}

	return s
}

// LocalizeWithConfig localizer the provided localize config
func (b *Bundle) LocalizeWithConfig(l *i18n.Localizer, lc *i18n.LocalizeConfig) string {
	s, err := l.Localize(lc)
	if err != nil {
		b.api.LogWarn("Failed to localize with config", "error", err.Error())
	}

	return s
}

// LocalizeErrorMessage localizer the provided error message
func (b *Bundle) LocalizeErrorMessage(l *i18n.Localizer, m *ErrorMessage) string {
	return b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
		DefaultMessage: m.Message,
		TemplateData:   m.Data,
	})
}
