package plugin

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"golang.org/x/text/language"

	"github.com/matterpoll/matterpoll/server/poll"
)

// initBundle loads all localization files in i18n into a bundle and return this
func (p *MatterpollPlugin) initBundle() (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bundle path")
	}

	i18nDir := filepath.Join(bundlePath, "assets", "i18n")
	files, err := ioutil.ReadDir(i18nDir)
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
		_, err = bundle.LoadMessageFile(filepath.Join(i18nDir, file.Name()))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load message file %s", file.Name())
		}
	}

	return bundle, nil
}

// getUserLocalizer returns a localizer that localizes in the users locale
func (p *MatterpollPlugin) getUserLocalizer(userID string) *i18n.Localizer {
	user, err := p.API.GetUser(userID)
	if err != nil {
		p.API.LogWarn("Failed get user's locale", "error", err.Error())
		return p.getServerLocalizer()
	}

	return i18n.NewLocalizer(p.bundle, user.Locale)
}

// getServerLocalizer returns a localizer that localizes in the server default client locale
func (p *MatterpollPlugin) getServerLocalizer() *i18n.Localizer {
	return i18n.NewLocalizer(p.bundle, *p.ServerConfig.LocalizationSettings.DefaultClientLocale)
}

// LocalizeDefaultMessage localizer the provided message
func (p *MatterpollPlugin) LocalizeDefaultMessage(l *i18n.Localizer, m *i18n.Message) string {
	s, err := l.LocalizeMessage(m)
	if err != nil {
		p.API.LogWarn("Failed to localize message", "message ID", m.ID, "error", err.Error())
		return ""
	}
	return s
}

// LocalizeWithConfig localizer the provided localize config
func (p *MatterpollPlugin) LocalizeWithConfig(l *i18n.Localizer, lc *i18n.LocalizeConfig) string {
	s, err := l.Localize(lc)
	if err != nil {
		p.API.LogWarn("Failed to localize with config", "error", err.Error())
		return ""
	}
	return s
}

// LocalizeErrorMessage localizer the provided error message
func (p *MatterpollPlugin) LocalizeErrorMessage(l *i18n.Localizer, m *poll.ErrorMessage) string {
	return p.LocalizeWithConfig(l, &i18n.LocalizeConfig{
		DefaultMessage: m.Message,
		TemplateData:   m.Data,
	})
}
