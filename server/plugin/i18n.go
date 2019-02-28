package plugin

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/matterpoll/matterpoll/server/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"golang.org/x/text/language"
)

func initBundle() (*i18n.Bundle, error) {
	bundle := &i18n.Bundle{DefaultLanguage: language.English}
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	i18nDir := utils.GetPluginRootPath() + "/i18n"
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
		_, err = bundle.LoadMessageFile(i18nDir + "/" + file.Name())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load message file %s", file.Name())
		}
	}

	return bundle, nil
}
