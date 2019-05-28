package testutils

import (
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// GetLocalizer return an localizer with an empty bundle
func GetLocalizer() *i18n.Localizer {
	return i18n.NewLocalizer(i18n.NewBundle(language.English))
}
