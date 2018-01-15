// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"testing"

	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
	"github.com/stretchr/testify/require"
)

// There are no tests that actually run the Message Export job, because it can take a long time to complete depending
// on the size of the database that the config is pointing to. As such, these tests just ensure that the CLI command
// fails fast if invalid flags are supplied

func TestMessageExportNotEnabled(t *testing.T) {
	configPath := writeTempConfig(t, false)
	defer os.RemoveAll(filepath.Dir(configPath))

	// should fail fast because the feature isn't enabled
	require.Error(t, runCommand(t, "--config", configPath, "export"))
}

func TestMessageExportInvalidFormat(t *testing.T) {
	configPath := writeTempConfig(t, true)
	defer os.RemoveAll(filepath.Dir(configPath))

	// should fail fast because format isn't supported
	require.Error(t, runCommand(t, "--config", configPath, "--format", "not_actiance", "export"))
}

func TestMessageExportNegativeExportFrom(t *testing.T) {
	configPath := writeTempConfig(t, true)
	defer os.RemoveAll(filepath.Dir(configPath))

	// should fail fast because export from must be a valid timestamp
	require.Error(t, runCommand(t, "--config", configPath, "--format", "actiance", "--exportFrom", "-1", "export"))
}

func TestMessageExportNegativeTimeoutSeconds(t *testing.T) {
	configPath := writeTempConfig(t, true)
	defer os.RemoveAll(filepath.Dir(configPath))

	// should fail fast because timeout seconds must be a positive int
	require.Error(t, runCommand(t, "--config", configPath, "--format", "actiance", "--exportFrom", "0", "--timeoutSeconds", "-1", "export"))
}

func writeTempConfig(t *testing.T, isMessageExportEnabled bool) string {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	utils.TranslationsPreInit()
	config := utils.LoadGlobalConfig("config.json")
	config.MessageExportSettings.EnableExport = model.NewBool(isMessageExportEnabled)
	configPath := filepath.Join(dir, "foo.json")
	require.NoError(t, ioutil.WriteFile(configPath, []byte(config.ToJson()), 0600))

	return configPath
}
