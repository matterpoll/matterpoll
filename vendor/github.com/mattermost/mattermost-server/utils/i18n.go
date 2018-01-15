// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	l4g "github.com/alecthomas/log4go"
	"github.com/mattermost/mattermost-server/model"
	"github.com/nicksnyder/go-i18n/i18n"
)

var T i18n.TranslateFunc
var TDefault i18n.TranslateFunc
var locales map[string]string = make(map[string]string)
var settings model.LocalizationSettings

// this functions loads translations from filesystem
// and assign english while loading server config
func TranslationsPreInit() error {
	if err := InitTranslationsWithDir("i18n"); err != nil {
		return err
	}

	T = TfuncWithFallback("en")
	TDefault = TfuncWithFallback("en")

	return nil
}

func InitTranslations(localizationSettings model.LocalizationSettings) error {
	settings = localizationSettings

	var err error
	T, err = GetTranslationsBySystemLocale()
	return err
}

func InitTranslationsWithDir(dir string) error {
	i18nDirectory, found := FindDir(dir)
	if !found {
		return fmt.Errorf("Unable to find i18n directory")
	}

	files, _ := ioutil.ReadDir(i18nDirectory)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			filename := f.Name()
			locales[strings.Split(filename, ".")[0]] = i18nDirectory + filename

			if err := i18n.LoadTranslationFile(i18nDirectory + filename); err != nil {
				return err
			}
		}
	}

	return nil
}

func GetTranslationsBySystemLocale() (i18n.TranslateFunc, error) {
	locale := *settings.DefaultServerLocale
	if _, ok := locales[locale]; !ok {
		l4g.Error("Failed to load system translations for '%v' attempting to fall back to '%v'", locale, model.DEFAULT_LOCALE)
		locale = model.DEFAULT_LOCALE
	}

	if locales[locale] == "" {
		return nil, fmt.Errorf("Failed to load system translations for '%v'", model.DEFAULT_LOCALE)
	}

	translations := TfuncWithFallback(locale)
	if translations == nil {
		return nil, fmt.Errorf("Failed to load system translations")
	}

	l4g.Info(translations("utils.i18n.loaded"), locale, locales[locale])
	return translations, nil
}

func GetUserTranslations(locale string) i18n.TranslateFunc {
	if _, ok := locales[locale]; !ok {
		locale = model.DEFAULT_LOCALE
	}

	translations := TfuncWithFallback(locale)
	return translations
}

func SetTranslations(locale string) i18n.TranslateFunc {
	translations := TfuncWithFallback(locale)
	return translations
}

func GetTranslationsAndLocale(w http.ResponseWriter, r *http.Request) (i18n.TranslateFunc, string) {
	// This is for checking against locales like pt_BR or zn_CN
	headerLocaleFull := strings.Split(r.Header.Get("Accept-Language"), ",")[0]
	// This is for checking agains locales like en, es
	headerLocale := strings.Split(strings.Split(r.Header.Get("Accept-Language"), ",")[0], "-")[0]
	defaultLocale := *settings.DefaultClientLocale
	if locales[headerLocaleFull] != "" {
		translations := TfuncWithFallback(headerLocaleFull)
		return translations, headerLocaleFull
	} else if locales[headerLocale] != "" {
		translations := TfuncWithFallback(headerLocale)
		return translations, headerLocale
	} else if locales[defaultLocale] != "" {
		translations := TfuncWithFallback(defaultLocale)
		return translations, headerLocale
	}

	translations := TfuncWithFallback(model.DEFAULT_LOCALE)
	return translations, model.DEFAULT_LOCALE
}

func GetSupportedLocales() map[string]string {
	return locales
}

func TfuncWithFallback(pref string) i18n.TranslateFunc {
	t, _ := i18n.Tfunc(pref)
	return func(translationID string, args ...interface{}) string {
		if translated := t(translationID, args...); translated != translationID {
			return translated
		}

		t, _ := i18n.Tfunc(model.DEFAULT_LOCALE)
		return t(translationID, args...)
	}
}
