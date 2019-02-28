package plugin

import (
	"encoding/json"

	"github.com/matterpoll/matterpoll/server/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"golang.org/x/text/language"
)

func initBundle() (*i18n.Bundle, error) {
	bundle := &i18n.Bundle{DefaultLanguage: language.English}
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	i18nPath := utils.GetPluginRootPath() + "/i18n"
	_, err := bundle.LoadMessageFile(i18nPath + "/active.de.json")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load message file %s", "de.json")
	}

	return bundle, nil
}
